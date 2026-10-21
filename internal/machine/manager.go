package machine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/compiler"
	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/image"
	"github.com/Inoriol/lxspawn/internal/kernel"
	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/provision"
	"github.com/Inoriol/lxspawn/internal/storage"
)

type Manager struct {
	state   *deploy.StateManager
	systemd *deploy.SystemdManager
	storage storage.StorageBackend
}

func NewManager() (*Manager, error) {
	state, err := deploy.NewStateManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	systemd, err := deploy.NewSystemdManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd: %w", err)
	}

	// Detect and initialize storage backend
	storageBackend, err := storage.DetectBackend(deploy.MachinesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage backend: %w", err)
	}

	logger.Info(fmt.Sprintf("Using storage backend: %s", storageBackend.Type()))

	return &Manager{
		state:   state,
		systemd: systemd,
		storage: storageBackend,
	}, nil
}

func (m *Manager) Close() {
	if m.systemd != nil {
		m.systemd.Close()
	}
}

func (m *Manager) Create(cfg *config.Config, sourcePath string) ([]string, error) {
	logger.Action(fmt.Sprintf("Creating machine '%s'", cfg.Name))

	if _, exists := m.state.GetMachine(cfg.Name); exists {
		return nil, fmt.Errorf("machine '%s' already exists", cfg.Name)
	}

	if kernel.IsBtrfs(deploy.MachinesDir) {
		ok, err := kernel.CheckKernelVersion(5, 12)
		if err != nil {
			logger.Warn(fmt.Sprintf("could not check kernel version: %v", err))
		} else if !ok {
			return nil, fmt.Errorf("btrfs requires kernel 5.12+ for idmapped mounts support (required for correct UID handling in containers)")
		}
	}

	compiled, err := compiler.Compile(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to compile config: %w", err)
	}

	rootfsPath := deploy.MachineRootfsPath(cfg.Name)
	if !deploy.FileExists(rootfsPath) || !isPopulatedRootfs(rootfsPath) {
		// Use storage backend to create machine storage
		sizeGB, err := config.ParseSizeToGB(cfg.Storage.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to parse storage size: %w", err)
		}

		if err := m.storage.CreateMachine(cfg.Name, sizeGB); err != nil {
			return nil, fmt.Errorf("failed to create machine storage: %w", err)
		}

		fetcher, err := image.NewFetcher(cfg.Name, cfg.Distro, rootfsPath)
		if err != nil {
			m.storage.DeleteMachine(cfg.Name)
			return nil, fmt.Errorf("failed to create image fetcher: %w", err)
		}

		if err := fetcher.Fetch(); err != nil {
			m.storage.DeleteMachine(cfg.Name)
			return nil, fmt.Errorf("failed to fetch image: %w", err)
		}

		if err := image.CleanupRootfs(rootfsPath); err != nil {
			logger.Warn(fmt.Sprintf("rootfs cleanup warning: %v", err))
		}

		if err := image.InjectNetworkConfig(rootfsPath, cfg.Network.Mode, cfg.Network.Interface); err != nil {
			logger.Warn(fmt.Sprintf("network config injection warning: %v", err))
		}
	}

	var filesWritten []string

	if err := deploy.WriteFileAtomic(compiled.NspawnPath, []byte(compiled.Nspawn), 0644); err != nil {
		m.storage.DeleteMachine(cfg.Name)
		return nil, fmt.Errorf("failed to write .nspawn file: %w", err)
	}
	filesWritten = append(filesWritten, compiled.NspawnPath)
	logger.Debug(fmt.Sprintf("Wrote %s", compiled.NspawnPath))

	if compiled.Slice != "" {
		if err := deploy.WriteFileAtomic(compiled.SlicePath, []byte(compiled.Slice), 0644); err != nil {
			deploy.RemoveAll(compiled.NspawnPath)
			m.storage.DeleteMachine(cfg.Name)
			return nil, fmt.Errorf("failed to write slice file: %w", err)
		}
		filesWritten = append(filesWritten, compiled.SlicePath)
		logger.Debug(fmt.Sprintf("Wrote %s", compiled.SlicePath))
	}

	if err := deploy.WriteFileAtomic(compiled.DropInPath, []byte(compiled.DropIn), 0644); err != nil {
		deploy.RemoveAll(compiled.NspawnPath)
		if compiled.Slice != "" {
			deploy.RemoveAll(compiled.SlicePath)
		}
		m.storage.DeleteMachine(cfg.Name)
		return nil, fmt.Errorf("failed to write drop-in file: %w", err)
	}
	filesWritten = append(filesWritten, compiled.DropInPath)
	logger.Debug(fmt.Sprintf("Wrote %s", compiled.DropInPath))

	if len(compiled.HookScripts) > 0 {
		hooksDir := deploy.HooksDir(cfg.Name)
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			cleanupCreatedFiles(filesWritten, compiled)
			m.storage.DeleteMachine(cfg.Name)
			return nil, fmt.Errorf("failed to create hooks directory: %w", err)
		}

		for name, content := range compiled.HookScripts {
			scriptPath := hooksDir + "/" + name
			if err := deploy.WriteFileAtomic(scriptPath, []byte(content), 0755); err != nil {
				cleanupCreatedFiles(filesWritten, compiled)
				m.storage.DeleteMachine(cfg.Name)
				return nil, fmt.Errorf("failed to write hook script %s: %w", name, err)
			}
			if err := deploy.SetSELinuxContext(scriptPath, "bin_t"); err != nil {
				logger.Warn(fmt.Sprintf("failed to set SELinux context on %s: %v", scriptPath, err))
			}
			filesWritten = append(filesWritten, scriptPath)
			logger.Debug(fmt.Sprintf("Wrote hook %s", scriptPath))
		}

		if err := deploy.WriteFileAtomic(compiled.HooksPath, []byte(compiled.HooksDropIn), 0644); err != nil {
			cleanupCreatedFiles(filesWritten, compiled)
			m.storage.DeleteMachine(cfg.Name)
			return nil, fmt.Errorf("failed to write hooks drop-in: %w", err)
		}
		filesWritten = append(filesWritten, compiled.HooksPath)
		logger.Debug(fmt.Sprintf("Wrote %s", compiled.HooksPath))
	}

	lxspawnDir := deploy.LxspawnMachineDir(cfg.Name)
	if err := os.MkdirAll(lxspawnDir, 0755); err != nil {
		cleanupCreatedFiles(filesWritten, compiled)
		m.storage.DeleteMachine(cfg.Name)
		return nil, fmt.Errorf("failed to create machine data directory: %w", err)
	}

	machineInfo := deploy.MachineInfo{
		Name:       cfg.Name,
		Distro:     cfg.Distro,
		SourcePath: sourcePath,
		Files:      filesWritten,
		Network:    string(cfg.Network.Mode),
		CPUs:       cfg.Resources.CPUs,
		Memory:     cfg.Resources.Memory,
		Swap:       cfg.Resources.Swap,
		Tasks:      cfg.Resources.Tasks,
		CreatedAt:  deploy.CreatedAt(),
	}

	if err := m.state.RegisterMachine(machineInfo); err != nil {
		cleanupCreatedFiles(filesWritten, compiled)
		deploy.RemoveAll(lxspawnDir)
		m.storage.DeleteMachine(cfg.Name)
		return nil, fmt.Errorf("failed to register machine in state: %w", err)
	}

	// Run pre-boot provisioning if configured
	if len(cfg.Provision) > 0 {
		engine, err := provision.NewEngine(cfg.Name)
		if err != nil {
			return filesWritten, fmt.Errorf("failed to create provision engine: %w", err)
		}
		if err := engine.Provision(cfg, sourcePath); err != nil {
			return filesWritten, fmt.Errorf("pre-boot provisioning failed: %w", err)
		}
	}

	if err := fixRootfsOwnership(rootfsPath); err != nil {
		logger.Warn(fmt.Sprintf("rootfs ownership fix warning: %v", err))
	}

	logger.Success(fmt.Sprintf("Machine '%s' created successfully", cfg.Name))
	return filesWritten, nil
}

func (m *Manager) Launch(cfg *config.Config, sourcePath string) error {
	if _, err := m.Create(cfg, sourcePath); err != nil {
		return err
	}
	if err := m.Start(cfg.Name); err != nil {
		return err
	}

	// Run post-boot provisioning if configured
	hasPostBoot := false
	for _, p := range cfg.Provision {
		if p.Mode == config.ProvisionSystem || p.Mode == config.ProvisionUser {
			hasPostBoot = true
			break
		}
	}
	if len(cfg.Probes) > 0 {
		hasPostBoot = true
	}

	if hasPostBoot {
		logger.Info("Waiting for machine to be ready...")
		if err := m.WaitForMachine(cfg.Name, 30*time.Second); err != nil {
			return fmt.Errorf("machine not ready: %w", err)
		}

		engine, err := provision.NewEngine(cfg.Name)
		if err != nil {
			return fmt.Errorf("failed to create provision engine: %w", err)
		}
		if err := engine.ProvisionPostBoot(cfg, sourcePath); err != nil {
			return fmt.Errorf("post-boot provisioning failed: %w", err)
		}
	}

	return nil
}

func (m *Manager) Start(name string) error {
	logger.Action(fmt.Sprintf("Starting machine '%s'", name))

	if _, exists := m.state.GetMachine(name); !exists {
		return fmt.Errorf("machine '%s' not found", name)
	}

	if err := m.systemd.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)
	if err := m.systemd.StartUnit(unitName); err != nil {
		return fmt.Errorf("failed to start machine: %w", err)
	}

	logger.Success(fmt.Sprintf("Machine '%s' started", name))
	return nil
}

func (m *Manager) Stop(name string) error {
	logger.Action(fmt.Sprintf("Stopping machine '%s'", name))

	unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)
	if err := m.systemd.StopUnit(unitName); err != nil {
		logger.Warn(fmt.Sprintf("stop unit warning: %v", err))
		
		if err := m.systemd.ResetFailedUnit(unitName); err != nil {
			logger.Warn(fmt.Sprintf("reset failed warning: %v", err))
		}
		
		cleanupCmd := exec.Command("machinectl", "terminate", name)
		if output, err := cleanupCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to stop machine: %w (%s)", err, string(output))
		}
	}

	logger.Success(fmt.Sprintf("Machine '%s' stopped", name))
	return nil
}

func (m *Manager) Delete(name string) error {
	logger.Action(fmt.Sprintf("Deleting machine '%s'", name))

	if _, exists := m.state.GetMachine(name); !exists {
		return fmt.Errorf("machine '%s' not found", name)
	}

	unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)
	
	if err := m.systemd.StopUnit(unitName); err != nil {
		logger.Warn(fmt.Sprintf("failed to stop machine (may already be stopped): %v", err))
	}
	
	if err := m.systemd.ResetFailedUnit(unitName); err != nil {
		logger.Warn(fmt.Sprintf("failed to reset failed state (may not be in failed state): %v", err))
	}
	
	cleanupCmd := exec.Command("machinectl", "terminate", name)
	if output, err := cleanupCmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("machinectl terminate warning: %v (%s)", err, string(output)))
	}

	var paths []string
	paths = append(paths, deploy.NspawnFilePath(name))
	paths = append(paths, deploy.SliceFilePath(name))
	paths = append(paths, deploy.DropInDir(name))
	paths = append(paths, deploy.LxspawnMachineDir(name))

	if err := deploy.RemoveAll(paths...); err != nil {
		return fmt.Errorf("failed to remove files: %w", err)
	}

	// Use storage backend to delete machine storage
	if err := m.storage.DeleteMachine(name); err != nil {
		return fmt.Errorf("failed to delete machine storage: %w", err)
	}

	if err := m.systemd.ReloadDaemon(); err != nil {
		logger.Warn(fmt.Sprintf("failed to reload systemd after delete: %v", err))
	}

	if err := m.state.UnregisterMachine(name); err != nil {
		return fmt.Errorf("failed to unregister machine: %w", err)
	}

	logger.Success(fmt.Sprintf("Machine '%s' deleted", name))
	return nil
}

func removeRootfs(rootfsPath string) error {
	if isBtrfsSubvolume(rootfsPath) {
		return removeBtrfsSubvolumeTree(rootfsPath)
	}

	cmd := deploy.SudoHelper("rm-rf", rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rm -rf failed: %w (%s)", err, string(output))
	}
	return nil
}

func isBtrfsSubvolume(path string) bool {
	out, err := exec.Command("sudo", "btrfs", "subvolume", "show", path).CombinedOutput()
	if err != nil {
		return false
	}
	return !strings.Contains(string(out), "Not a Btrfs subvolume") && !strings.Contains(string(out), "ERROR")
}

func removeBtrfsSubvolumeTree(rootfsPath string) error {
	out, err := exec.Command("sudo", "btrfs", "subvolume", "list", "-o", rootfsPath).Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		var subvols []string
		for _, line := range lines {
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				subvolPath := filepath.Join("/", fields[len(fields)-1])
				subvols = append(subvols, subvolPath)
			}
		}

		sort.Sort(sort.Reverse(sort.StringSlice(subvols)))
		for _, subvol := range subvols {
			logger.Info(fmt.Sprintf("Deleting child subvolume: %s", subvol))
			cmd := deploy.SudoHelper("btrfs", "subvolume", "delete", subvol)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to delete child subvolume %s: %w (%s)", subvol, err, string(output))
			}
		}
	}

	logger.Info(fmt.Sprintf("Deleting root subvolume: %s", rootfsPath))
	cmd := deploy.SudoHelper("btrfs", "subvolume", "delete", rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete root subvolume: %w (%s)", err, string(output))
	}

	return nil
}

func (m *Manager) Shell(name string) error {
	if _, exists := m.state.GetMachine(name); !exists {
		return fmt.Errorf("machine '%s' not found", name)
	}

	cmd := exec.Command("machinectl", "shell", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) Exec(name string, command []string) error {
	if _, exists := m.state.GetMachine(name); !exists {
		return fmt.Errorf("machine '%s' not found", name)
	}

	shellCmd := shellQuoteArgs(command)
	args := []string{"shell", name, "/bin/bash", "-c", shellCmd}
	cmd := exec.Command("machinectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	needsQuoting := false
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '/' || c == '.' || c == '=' || c == '@' || c == '+' || c == ',') {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func (m *Manager) Logs(name string) error {
	if _, exists := m.state.GetMachine(name); !exists {
		return fmt.Errorf("machine '%s' not found", name)
	}

	cmd := exec.Command("journalctl", "-M", name, "-f")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type LogEntry struct {
	Timestamp string
	Priority  int
	Message   string
}

func (m *Manager) GetLogs(name string, lines int) ([]LogEntry, error) {
	if _, exists := m.state.GetMachine(name); !exists {
		return nil, fmt.Errorf("machine '%s' not found", name)
	}

	unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)
	cmd := deploy.SudoHelper("journalctl", "-u", unitName, "-n", fmt.Sprintf("%d", lines), "-o", "short-iso")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var entries []LogEntry
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}

		entry := LogEntry{
			Timestamp: parts[0],
			Priority:  6,
			Message:   parts[2],
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

type MachineStatus struct {
	Name      string
	Distro    string
	State     string
	Leader    int
	Addresses []string
	Network   string
	CreatedAt string
	SourcePath string
	Files     []string
	Ephemeral bool
	ReadOnly  bool
}

func (m *Manager) GetStatus(name string) (*MachineStatus, error) {
	info, exists := m.state.GetMachine(name)
	if !exists {
		return nil, fmt.Errorf("machine '%s' not found", name)
	}

	status := &MachineStatus{
		Name:       info.Name,
		Distro:     info.Distro,
		State:      "stopped",
		Network:    info.Network,
		CreatedAt:  info.CreatedAt,
		SourcePath: info.SourcePath,
		Files:      info.Files,
	}

	if info.SourcePath != "" {
		if cfg, err := config.Parse(info.SourcePath); err == nil {
			status.Ephemeral = cfg.Advanced.Ephemeral
			status.ReadOnly = cfg.Advanced.ReadOnly
		}
	}

	state, addrs, err := m.systemd.GetMachineState(name)
	if err == nil {
		status.State = state
		status.Addresses = addrs
	}

	return status, nil
}

func (m *Manager) ListMachines() ([]deploy.MachineInfo, error) {
	return m.state.ListMachines(), nil
}

type MachineWithState struct {
	deploy.MachineInfo
	State     string
	Addresses []string
}

func (m *Manager) ListMachinesWithState() ([]MachineWithState, error) {
	machines := m.state.ListMachines()

	running, err := m.systemd.ListRunningMachines()
	if err != nil {
		running = nil
	}

	runningMap := make(map[string]deploy.RunningMachineInfo)
	for _, r := range running {
		runningMap[r.Name] = r
	}

	result := make([]MachineWithState, 0, len(machines))
	for _, machine := range machines {
		mws := MachineWithState{
			MachineInfo: machine,
			State:       "stopped",
		}
		if r, ok := runningMap[machine.Name]; ok {
			mws.State = "running"
			mws.Addresses = r.Addresses
		}
		result = append(result, mws)
	}

	return result, nil
}

func (m *Manager) WaitForMachine(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := m.GetStatus(name)
		if err == nil && status.State == "running" {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for machine '%s' to start", name)
}

func (m *Manager) GetMachineInfo(name string) (deploy.MachineInfo, bool) {
	return m.state.GetMachine(name)
}

func (m *Manager) Provision(cfg *config.Config, sourcePath string) error {
	status, err := m.GetStatus(cfg.Name)
	if err != nil {
		return fmt.Errorf("failed to get machine status: %w", err)
	}

	wasRunning := status.State == "running"

	if wasRunning {
		logger.Info("Stopping machine for pre-boot provisioning...")
		if err := m.Stop(cfg.Name); err != nil {
			return fmt.Errorf("failed to stop machine: %w", err)
		}
	}

	// Run pre-boot provisioning (boot, data, yq modes)
	hasPreBoot := false
	for _, p := range cfg.Provision {
		if p.Mode == config.ProvisionBoot || p.Mode == config.ProvisionData || p.Mode == config.ProvisionYQ {
			hasPreBoot = true
			break
		}
	}

	if hasPreBoot {
		engine, err := provision.NewEngine(cfg.Name)
		if err != nil {
			return fmt.Errorf("failed to create provision engine: %w", err)
		}
		if err := engine.Provision(cfg, sourcePath); err != nil {
			return fmt.Errorf("pre-boot provisioning failed: %w", err)
		}
	}

	if wasRunning {
		logger.Info("Starting machine for post-boot provisioning...")
		if err := m.Start(cfg.Name); err != nil {
			return fmt.Errorf("failed to start machine: %w", err)
		}

		hasPostBoot := false
		for _, p := range cfg.Provision {
			if p.Mode == config.ProvisionSystem || p.Mode == config.ProvisionUser {
				hasPostBoot = true
				break
			}
		}
		if len(cfg.Probes) > 0 {
			hasPostBoot = true
		}

		if hasPostBoot {
			logger.Info("Waiting for machine to be ready...")
			if err := m.WaitForMachine(cfg.Name, 30*time.Second); err != nil {
				return fmt.Errorf("machine not ready: %w", err)
			}

			engine, err := provision.NewEngine(cfg.Name)
			if err != nil {
				return fmt.Errorf("failed to create provision engine: %w", err)
			}
			if err := engine.ProvisionPostBoot(cfg, sourcePath); err != nil {
				return fmt.Errorf("post-boot provisioning failed: %w", err)
			}
		}
	}

	logger.Success("Provisioning complete")
	return nil
}

func isPopulatedRootfs(path string) bool {
	osRelease := filepath.Join(path, "etc", "os-release")
	_, err := os.Stat(osRelease)
	return err == nil
}

func fixRootfsOwnership(rootfsPath string) error {
	cmd := deploy.SudoHelper("chown", "-R", "0:0", rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %w (%s)", err, string(output))
	}
	return nil
}

type MachineStats struct {
	Name              string
	State             string
	CPUUsagePercent   float64
	MemoryUsedBytes   int64
	MemoryTotalBytes  int64
	MemoryUsagePercent float64
	DiskUsedBytes     int64
	DiskTotalBytes    int64
}

func (m *Manager) GetStats(name string) (*MachineStats, error) {
	info, exists := m.state.GetMachine(name)
	if !exists {
		return nil, fmt.Errorf("machine '%s' not found", name)
	}

	state, _, err := m.systemd.GetMachineState(name)
	if err != nil {
		state = "stopped"
	}

	stats := &MachineStats{
		Name:  name,
		State: state,
	}

	if state != "running" {
		return stats, nil
	}

	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/machine.slice/machine-%s.slice", name)

	if cpuUsage, err := readCPUUsage(cgroupPath); err == nil {
		stats.CPUUsagePercent = cpuUsage
	}

	if memUsed, memTotal, err := readMemoryUsage(cgroupPath); err == nil {
		stats.MemoryUsedBytes = memUsed
		stats.MemoryTotalBytes = memTotal
		if memTotal > 0 {
			stats.MemoryUsagePercent = float64(memUsed) / float64(memTotal) * 100
		}
	}

	if diskUsed, diskTotal, err := readDiskUsage(deploy.MachineRootfsPath(name)); err == nil {
		stats.DiskUsedBytes = diskUsed
		stats.DiskTotalBytes = diskTotal
	}

	_ = info

	return stats, nil
}

func readCPUUsage(cgroupPath string) (float64, error) {
	cpuStatPath := filepath.Join(cgroupPath, "cpu.stat")
	
	data1, err := os.ReadFile(cpuStatPath)
	if err != nil {
		return 0, err
	}

	var usageUsec1 int64
	for _, line := range strings.Split(string(data1), "\n") {
		if strings.HasPrefix(line, "usage_usec") {
			fmt.Sscanf(line, "usage_usec %d", &usageUsec1)
			break
		}
	}

	if usageUsec1 == 0 {
		return 0, nil
	}

	time.Sleep(100 * time.Millisecond)

	data2, err := os.ReadFile(cpuStatPath)
	if err != nil {
		return 0, err
	}

	var usageUsec2 int64
	for _, line := range strings.Split(string(data2), "\n") {
		if strings.HasPrefix(line, "usage_usec") {
			fmt.Sscanf(line, "usage_usec %d", &usageUsec2)
			break
		}
	}

	deltaUsec := usageUsec2 - usageUsec1
	if deltaUsec < 0 {
		return 0, nil
	}

	intervalUsec := int64(100000)
	cpuPercent := float64(deltaUsec) / float64(intervalUsec) * 100.0

	return cpuPercent, nil
}

func readMemoryUsage(cgroupPath string) (int64, int64, error) {
	memoryCurrentPath := filepath.Join(cgroupPath, "memory.current")
	memoryMaxPath := filepath.Join(cgroupPath, "memory.max")

	memCurrentData, err := os.ReadFile(memoryCurrentPath)
	if err != nil {
		return 0, 0, err
	}

	var memCurrent int64
	fmt.Sscanf(strings.TrimSpace(string(memCurrentData)), "%d", &memCurrent)

	var memMax int64
	if memMaxData, err := os.ReadFile(memoryMaxPath); err == nil {
		memMaxStr := strings.TrimSpace(string(memMaxData))
		if memMaxStr != "max" {
			fmt.Sscanf(memMaxStr, "%d", &memMax)
		}
	}

	return memCurrent, memMax, nil
}

func readDiskUsage(path string) (int64, int64, error) {
	cmd := exec.Command("df", "-B1", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("df command failed for path %s: %w", path, err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("unexpected df output for path %s: %q", path, string(output))
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return 0, 0, fmt.Errorf("unexpected df output format for path %s: %q", path, string(output))
	}

	var total, used int64
	fmt.Sscanf(fields[1], "%d", &total)
	fmt.Sscanf(fields[2], "%d", &used)

	return used, total, nil
}

func cleanupCreatedFiles(filesWritten []string, compiled *compiler.CompiledFiles) {
	for _, f := range filesWritten {
		deploy.RemoveAll(f)
	}
	if compiled.HooksDropIn != "" {
		deploy.RemoveAll(compiled.HooksPath)
	}
	deploy.RemoveAll(compiled.DropInPath)
	if compiled.Slice != "" {
		deploy.RemoveAll(compiled.SlicePath)
	}
	deploy.RemoveAll(compiled.NspawnPath)
}

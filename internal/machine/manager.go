package machine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Inoriol/lxspawn/internal/compiler"
	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	lxerrors "github.com/Inoriol/lxspawn/internal/errors"
	"github.com/Inoriol/lxspawn/internal/image"
	"github.com/Inoriol/lxspawn/internal/kernel"
	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/provision"
	"github.com/Inoriol/lxspawn/internal/storage"
)

type Manager struct {
	state   deploy.StateStore
	systemd deploy.SystemdClient
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

	storageBackend, err := storage.DetectBackend(deploy.MachinesDirPath())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage backend: %w", err)
	}

	logger.Info(fmt.Sprintf("Using storage backend: %s", storageBackend.Type()))

	return NewManagerWithDeps(state, systemd, storageBackend), nil
}

func NewManagerWithDeps(state deploy.StateStore, systemd deploy.SystemdClient, storage storage.StorageBackend) *Manager {
	return &Manager{
		state:   state,
		systemd: systemd,
		storage: storage,
	}
}

func (m *Manager) Close() {
	if m.systemd != nil {
		m.systemd.Close()
	}
}

func (m *Manager) Create(cfg *config.Config, sourcePath string) ([]string, error) {
	logger.Action(fmt.Sprintf("Creating machine '%s'", cfg.Name))

	if _, exists := m.state.GetMachine(cfg.Name); exists {
		return nil, lxerrors.NewMachineExistsError(cfg.Name)
	}

	rootfsPath := deploy.MachineRootfsPath(cfg.Name)
	if deploy.FileExists(rootfsPath) && isPopulatedRootfs(rootfsPath) {
		return nil, lxerrors.NewStorageError(fmt.Sprintf("rootfs at %s already exists but is not managed by lxspawn. Remove it manually or use a different machine name", rootfsPath), nil)
	}

	if kernel.IsBtrfs(deploy.MachinesDirPath()) {
		ok, err := kernel.CheckKernelVersion(5, 12)
		if err != nil {
			logger.Warn(fmt.Sprintf("could not check kernel version: %v", err))
		} else if !ok {
			return nil, lxerrors.NewValidationError("kernel", "btrfs requires kernel 5.12+ for idmapped mounts support (required for correct UID handling in containers)")
		}
	}

	compiled, err := compiler.Compile(cfg)
	if err != nil {
		return nil, lxerrors.NewConfigInvalidError(fmt.Sprintf("failed to compile config: %v", err))
	}

	{
		sizeGB, err := config.ParseSizeToGB(cfg.Storage.Size)
		if err != nil {
			return nil, lxerrors.NewValidationError("storage.size", fmt.Sprintf("failed to parse storage size: %v", err))
		}

		if err := m.storage.CreateMachine(cfg.Name, sizeGB); err != nil {
			return nil, lxerrors.NewStorageError("failed to create machine storage", err)
		}

		fetcher, err := image.NewFetcher(cfg.Name, cfg.Distro, rootfsPath)
		if err != nil {
			m.storage.DeleteMachine(cfg.Name)
			return nil, lxerrors.NewNetworkError("failed to create image fetcher", err)
		}

		if err := fetcher.Fetch(); err != nil {
			m.storage.DeleteMachine(cfg.Name)
			return nil, lxerrors.NewNetworkError("failed to fetch image", err)
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

	canonicalConfigPath := deploy.CanonicalConfigPath(cfg.Name)
	if err := copyFile(sourcePath, canonicalConfigPath); err != nil {
		logger.Warn(fmt.Sprintf("failed to copy config to canonical path: %v", err))
	}

	machineInfo := deploy.MachineInfo{
		Name:       cfg.Name,
		Distro:     cfg.Distro,
		SourcePath: sourcePath,
		ConfigPath: canonicalConfigPath,
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

	if len(cfg.Provision) > 0 {
		engine, err := provision.NewEngine(cfg.Name)
		if err != nil {
			cleanupCreatedFiles(filesWritten, compiled)
			deploy.RemoveAll(lxspawnDir)
			m.storage.DeleteMachine(cfg.Name)
			m.state.UnregisterMachine(cfg.Name)
			return nil, fmt.Errorf("failed to create provision engine: %w", err)
		}
		if err := engine.Provision(cfg, sourcePath); err != nil {
			cleanupCreatedFiles(filesWritten, compiled)
			deploy.RemoveAll(lxspawnDir)
			m.storage.DeleteMachine(cfg.Name)
			m.state.UnregisterMachine(cfg.Name)
			return nil, lxerrors.NewProvisionError("pre-boot provisioning failed", err)
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
		logger.Debug(fmt.Sprintf("stop unit warning: %v", err))

		if err := m.systemd.ResetFailedUnit(unitName); err != nil {
			logger.Debug(fmt.Sprintf("reset failed warning: %v", err))
		}

		cleanupCmd := exec.Command("machinectl", "terminate", name)
		if output, err := cleanupCmd.CombinedOutput(); err != nil {
			logger.Debug(fmt.Sprintf("machinectl terminate warning: %v (%s)", err, string(output)))

			systemctlCmd := deploy.Sudo("systemctl", "stop", unitName)
			if output, err := systemctlCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to stop machine: %w (%s)", err, string(output))
			}
		}
	}

	logger.Success(fmt.Sprintf("Machine '%s' stopped", name))
	return nil
}

func (m *Manager) ForceStop(name string) error {
	logger.Action(fmt.Sprintf("Force stopping machine '%s'", name))

	poweroffCmd := exec.Command("machinectl", "poweroff", name)
	if output, err := poweroffCmd.CombinedOutput(); err != nil {
		logger.Debug(fmt.Sprintf("machinectl poweroff warning: %v (%s)", err, string(output)))
	}

	time.Sleep(2 * time.Second)

	state, _, err := m.systemd.GetMachineState(name)
	if err == nil && state != "stopped" {
		terminateCmd := exec.Command("machinectl", "terminate", name)
		if output, err := terminateCmd.CombinedOutput(); err != nil {
			logger.Debug(fmt.Sprintf("machinectl terminate warning: %v (%s)", err, string(output)))
		}
	}

	time.Sleep(1 * time.Second)

	state, _, err = m.systemd.GetMachineState(name)
	if err == nil && state != "stopped" {
		unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)
		killCmd := deploy.Sudo("systemctl", "kill", "-s", "SIGKILL", unitName)
		if output, err := killCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to force stop machine: %w (%s)", err, string(output))
		}
	}

	logger.Success(fmt.Sprintf("Machine '%s' force stopped", name))
	return nil
}

func (m *Manager) Restart(name string) error {
	logger.Action(fmt.Sprintf("Restarting machine '%s'", name))

	if _, exists := m.state.GetMachine(name); !exists {
		return fmt.Errorf("machine '%s' not found", name)
	}

	if err := m.Stop(name); err != nil {
		return fmt.Errorf("failed to stop machine for restart: %w", err)
	}

	time.Sleep(1 * time.Second)

	if err := m.Start(name); err != nil {
		return fmt.Errorf("failed to start machine after stop: %w", err)
	}

	logger.Success(fmt.Sprintf("Machine '%s' restarted", name))
	return nil
}

func (m *Manager) Delete(name string) error {
	logger.Action(fmt.Sprintf("Deleting machine '%s'", name))

	if _, exists := m.state.GetMachine(name); !exists {
		return fmt.Errorf("machine '%s' not found", name)
	}

	unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)

	if err := m.systemd.StopUnit(unitName); err != nil {
		logger.Debug(fmt.Sprintf("stop unit warning (may already be stopped): %v", err))
	}

	if err := m.systemd.ResetFailedUnit(unitName); err != nil {
		logger.Debug(fmt.Sprintf("reset failed warning (may not be in failed state): %v", err))
	}

	cleanupCmd := exec.Command("machinectl", "terminate", name)
	if output, err := cleanupCmd.CombinedOutput(); err != nil {
		logger.Debug(fmt.Sprintf("machinectl terminate warning (may already be terminated): %v (%s)", err, string(output)))
	}

	state, _, err := m.systemd.GetMachineState(name)
	if err == nil && state != "stopped" {
		return fmt.Errorf("machine '%s' is still running (state: %s). Please stop it manually before deleting", name, state)
	}

	var paths []string
	paths = append(paths, deploy.NspawnFilePath(name))
	paths = append(paths, deploy.SliceFilePath(name))
	paths = append(paths, deploy.DropInDir(name))
	paths = append(paths, deploy.LxspawnMachineDir(name))

	if err := deploy.RemoveAll(paths...); err != nil {
		return fmt.Errorf("failed to remove files: %w", err)
	}

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

type MachineStatus struct {
	Name       string
	Distro     string
	State      string
	Leader     int
	Addresses  []string
	Network    string
	CreatedAt  string
	SourcePath string
	Files      []string
	Ephemeral  bool
	ReadOnly   bool
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

	configPath := info.ConfigPath
	if configPath == "" || !deploy.FileExists(configPath) {
		configPath = info.SourcePath
	}

	if configPath != "" {
		if cfg, err := config.Parse(configPath); err == nil {
			status.Ephemeral = cfg.Advanced.Ephemeral
			status.ReadOnly = cfg.Advanced.ReadOnly
		}
	}

	state, addrs, err := m.systemd.GetMachineState(name)
	if err == nil {
		status.State = state
		status.Addresses = addrs
	}

	if status.State == "running" {
		if leader, err := m.systemd.GetMachineLeader(name); err == nil {
			status.Leader = leader
		}
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
	cmd := deploy.Sudo("chown", "-R", "0:0", rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %w (%s)", err, string(output))
	}
	return nil
}

func (m *Manager) Apply(cfg *config.Config, sourcePath string) error {
	logger.Action(fmt.Sprintf("Applying config changes to machine '%s'", cfg.Name))

	if _, exists := m.state.GetMachine(cfg.Name); !exists {
		return fmt.Errorf("machine '%s' not found", cfg.Name)
	}

	compiled, err := compiler.Compile(cfg)
	if err != nil {
		return fmt.Errorf("failed to compile config: %w", err)
	}

	var filesWritten []string

	if err := deploy.WriteFileAtomic(compiled.NspawnPath, []byte(compiled.Nspawn), 0644); err != nil {
		return fmt.Errorf("failed to write .nspawn file: %w", err)
	}
	filesWritten = append(filesWritten, compiled.NspawnPath)
	logger.Debug(fmt.Sprintf("Wrote %s", compiled.NspawnPath))

	if compiled.Slice != "" {
		if err := deploy.WriteFileAtomic(compiled.SlicePath, []byte(compiled.Slice), 0644); err != nil {
			return fmt.Errorf("failed to write slice file: %w", err)
		}
		filesWritten = append(filesWritten, compiled.SlicePath)
		logger.Debug(fmt.Sprintf("Wrote %s", compiled.SlicePath))
	}

	if err := deploy.WriteFileAtomic(compiled.DropInPath, []byte(compiled.DropIn), 0644); err != nil {
		return fmt.Errorf("failed to write drop-in file: %w", err)
	}
	filesWritten = append(filesWritten, compiled.DropInPath)
	logger.Debug(fmt.Sprintf("Wrote %s", compiled.DropInPath))

	if len(compiled.HookScripts) > 0 {
		hooksDir := deploy.HooksDir(cfg.Name)
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			return fmt.Errorf("failed to create hooks directory: %w", err)
		}

		for name, content := range compiled.HookScripts {
			scriptPath := hooksDir + "/" + name
			if err := deploy.WriteFileAtomic(scriptPath, []byte(content), 0755); err != nil {
				return fmt.Errorf("failed to write hook script %s: %w", name, err)
			}
			if err := deploy.SetSELinuxContext(scriptPath, "bin_t"); err != nil {
				logger.Warn(fmt.Sprintf("failed to set SELinux context on %s: %v", scriptPath, err))
			}
			filesWritten = append(filesWritten, scriptPath)
			logger.Debug(fmt.Sprintf("Wrote hook %s", scriptPath))
		}

		if err := deploy.WriteFileAtomic(compiled.HooksPath, []byte(compiled.HooksDropIn), 0644); err != nil {
			return fmt.Errorf("failed to write hooks drop-in: %w", err)
		}
		filesWritten = append(filesWritten, compiled.HooksPath)
		logger.Debug(fmt.Sprintf("Wrote %s", compiled.HooksPath))
	}

	machineInfo := deploy.MachineInfo{
		Name:       cfg.Name,
		Distro:     cfg.Distro,
		SourcePath: sourcePath,
		ConfigPath: deploy.CanonicalConfigPath(cfg.Name),
		Files:      filesWritten,
		Network:    string(cfg.Network.Mode),
		CPUs:       cfg.Resources.CPUs,
		Memory:     cfg.Resources.Memory,
		Swap:       cfg.Resources.Swap,
		Tasks:      cfg.Resources.Tasks,
	}

	if err := m.state.RegisterMachine(machineInfo); err != nil {
		return fmt.Errorf("failed to update machine state: %w", err)
	}

	canonicalConfigPath := deploy.CanonicalConfigPath(cfg.Name)
	if err := copyFile(sourcePath, canonicalConfigPath); err != nil {
		logger.Warn(fmt.Sprintf("failed to update canonical config: %v", err))
	}

	if err := m.systemd.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	logger.Success(fmt.Sprintf("Configuration applied to machine '%s'", cfg.Name))
	return nil
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

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return deploy.WriteFileAtomic(dst, data, 0644)
}

package machine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	lxerrors "github.com/Inoriol/lxspawn/internal/errors"
	"github.com/Inoriol/lxspawn/internal/deploy"
)

type MachineStats struct {
	Name               string
	State              string
	CPUUsagePercent    float64
	CPUUsageUsec       int64
	MemoryUsedBytes    int64
	MemoryTotalBytes   int64
	MemoryUsagePercent float64
	DiskUsedBytes      int64
	DiskTotalBytes     int64
}

func (m *Manager) GetStats(name string) (*MachineStats, error) {
	if _, exists := m.state.GetMachine(name); !exists {
		return nil, lxerrors.NewMachineNotFoundError(name)
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

	cgroupPath, err := m.getMachineCgroupPath(name)
	if err != nil {
		return stats, nil
	}

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

	return stats, nil
}

func (m *Manager) GetAllStats() ([]MachineStats, error) {
	machines := m.state.ListMachines()
	statsList := make([]MachineStats, 0, len(machines))

	for _, machine := range machines {
		state, _, err := m.systemd.GetMachineState(machine.Name)
		if err != nil {
			state = "stopped"
		}

		stats := MachineStats{
			Name:  machine.Name,
			State: state,
		}

		if state == "running" {
			cgroupPath, err := m.getMachineCgroupPath(machine.Name)
			if err == nil {
				if cpuUsageUsec, err := readCPUUsageUsec(cgroupPath); err == nil {
					stats.CPUUsageUsec = cpuUsageUsec
				}

				if memUsed, memTotal, err := readMemoryUsage(cgroupPath); err == nil {
					stats.MemoryUsedBytes = memUsed
					stats.MemoryTotalBytes = memTotal
					if memTotal > 0 {
						stats.MemoryUsagePercent = float64(memUsed) / float64(memTotal) * 100
					}
				}

				if diskUsed, diskTotal, err := readDiskUsage(deploy.MachineRootfsPath(machine.Name)); err == nil {
					stats.DiskUsedBytes = diskUsed
					stats.DiskTotalBytes = diskTotal
				}
			}
		}

		statsList = append(statsList, stats)
	}

	return statsList, nil
}

func (m *Manager) getMachineCgroupPath(name string) (string, error) {
	leader, err := m.systemd.GetMachineLeader(name)
	if err != nil {
		return "", fmt.Errorf("failed to get machine leader: %w", err)
	}

	cgroupFile := fmt.Sprintf("/proc/%d/cgroup", leader)
	data, err := os.ReadFile(cgroupFile)
	if err != nil {
		return "", fmt.Errorf("failed to read cgroup file: %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "machine.slice") {
			parts := strings.Split(line, "::")
			if len(parts) == 2 {
				cgroupPath := parts[1]
				cgroupPath = strings.TrimSuffix(cgroupPath, "/systemd-nspawn@"+name+".service/payload/init.scope")
				cgroupPath = strings.TrimSuffix(cgroupPath, "/systemd-nspawn@"+name+".service")
				return "/sys/fs/cgroup" + cgroupPath, nil
			}
		}
	}

	return "", fmt.Errorf("machine cgroup not found")
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

	cpuMaxPath := filepath.Join(cgroupPath, "cpu.max")
	if data, err := os.ReadFile(cpuMaxPath); err == nil {
		var quota, period int64
		if n, _ := fmt.Sscanf(string(data), "%d %d", &quota, &period); n == 2 && quota > 0 && period > 0 {
			numCPUs := float64(quota) / float64(period)
			if numCPUs > 0 {
				cpuPercent = cpuPercent / numCPUs
			}
		} else {
			numCPUs := float64(runtime.NumCPU())
			if numCPUs > 0 {
				cpuPercent = cpuPercent / numCPUs
			}
		}
	} else {
		numCPUs := float64(runtime.NumCPU())
		if numCPUs > 0 {
			cpuPercent = cpuPercent / numCPUs
		}
	}

	return cpuPercent, nil
}

func readCPUUsageUsec(cgroupPath string) (int64, error) {
	cpuStatPath := filepath.Join(cgroupPath, "cpu.stat")

	data, err := os.ReadFile(cpuStatPath)
	if err != nil {
		return 0, err
	}

	var usageUsec int64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "usage_usec") {
			fmt.Sscanf(line, "usage_usec %d", &usageUsec)
			break
		}
	}

	return usageUsec, nil
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
	if isBtrfsPath(path) {
		used, err := btrfsSubvolumeUsage(path)
		if err == nil {
			total, totalErr := btrfsFilesystemTotal(path)
			if totalErr == nil {
				return used, total, nil
			}
		}
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, fmt.Errorf("failed to stat filesystem: %w", err)
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bfree) * int64(stat.Bsize)
	used := total - free

	return used, total, nil
}

func isBtrfsPath(path string) bool {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return false
	}
	return stat.Type == 0x9123683e
}

func btrfsSubvolumeUsage(path string) (int64, error) {
	cmd := exec.Command("btrfs", "filesystem", "du", "-s", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected output from btrfs filesystem du")
	}

	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected output format")
	}

	return parseBtrfsSize(fields[1])
}

func btrfsFilesystemTotal(path string) (int64, error) {
	cmd := exec.Command("btrfs", "filesystem", "usage", "-b", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Device size:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				var total int64
				fmt.Sscanf(parts[2], "%d", &total)
				return total, nil
			}
		}
	}

	return 0, fmt.Errorf("could not determine total size")
}

func parseBtrfsSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	multipliers := map[string]int64{
		"KiB": 1024,
		"MiB": 1024 * 1024,
		"GiB": 1024 * 1024 * 1024,
		"TiB": 1024 * 1024 * 1024 * 1024,
		"K":   1000,
		"M":   1000 * 1000,
		"G":   1000 * 1000 * 1000,
		"T":   1000 * 1000 * 1000 * 1000,
		"B":   1,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			numStr = strings.TrimSpace(numStr)
			var num float64
			if _, err := fmt.Sscanf(numStr, "%f", &num); err == nil {
				return int64(num * float64(mult)), nil
			}
		}
	}

	var bytes int64
	if _, err := fmt.Sscanf(s, "%d", &bytes); err == nil {
		return bytes, nil
	}

	return 0, fmt.Errorf("cannot parse size: %s", s)
}

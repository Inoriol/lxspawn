package snapshot

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/logger"
)

type LVMBackend struct {
	machineName string
}

func NewLVMBackend(machineName string) (*LVMBackend, error) {
	return &LVMBackend{machineName: machineName}, nil
}

func (l *LVMBackend) Type() BackendType {
	return BackendLVM
}

func findLVMBinary(name string) string {
	path, err := exec.LookPath(name)
	if err == nil {
		return path
	}
	return "/usr/sbin/" + name
}

func (l *LVMBackend) getMachineLVInfo(rootfsPath string) (vgName, lvName string, err error) {
	out, err := exec.Command("df", "--output=source", rootfsPath).Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get device for %s: %w", rootfsPath, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "", "", fmt.Errorf("unexpected df output")
	}

	device := strings.TrimSpace(lines[len(lines)-1])

	lvsPath := findLVMBinary("lvs")
	out, err = exec.Command("sudo", lvsPath, "--noheadings", "-o", "vg_name,lv_name", device).Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get LVM info for %s: %w", device, err)
	}

	fields := strings.Fields(string(out))
	if len(fields) < 2 {
		return "", "", fmt.Errorf("device %s is not an LVM volume", device)
	}

	return strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]), nil
}

func (l *LVMBackend) CreateSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Creating LVM snapshot '%s' for machine '%s'", snapName, machine))

	vgName, lvName, err := l.getMachineLVInfo(rootfsPath)
	if err != nil {
		return err
	}

	snapLVName := fmt.Sprintf("%s-snap-%s", lvName, snapName)

	lvsPath := findLVMBinary("lvs")
	existing, err := exec.Command("sudo", lvsPath, "--noheadings", "-o", "lv_name", fmt.Sprintf("%s/%s", vgName, snapLVName)).CombinedOutput()
	if err == nil && strings.TrimSpace(string(existing)) != "" {
		return fmt.Errorf("snapshot '%s' already exists", snapName)
	}

	lvcreatePath := findLVMBinary("lvcreate")
	cmd := exec.Command("sudo", lvcreatePath, "--snapshot", "--name", snapLVName, "--size", "512M", fmt.Sprintf("%s/%s", vgName, lvName))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create LVM snapshot: %w (%s)", err, string(output))
	}

	logger.Success(fmt.Sprintf("LVM snapshot '%s' created for %s/%s", snapName, vgName, lvName))
	return nil
}

func (l *LVMBackend) ListSnapshots(machine, rootfsPath string) ([]SnapshotInfo, error) {
	vgName, lvName, err := l.getMachineLVInfo(rootfsPath)
	if err != nil {
		return nil, err
	}

	lvsPath := findLVMBinary("lvs")
	out, err := exec.Command("sudo", lvsPath, "--noheadings", "-o", "lv_name,lv_time,lv_size", "--select", fmt.Sprintf("vg_name=%s", vgName)).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list LVM snapshots: %w", err)
	}

	var snapshots []SnapshotInfo
	prefix := fmt.Sprintf("%s-snap-", lvName)

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		lv := strings.TrimSpace(fields[0])
		if !strings.HasPrefix(lv, prefix) {
			continue
		}

		snapName := strings.TrimPrefix(lv, prefix)
		creationTime := time.Now()
		if t, err := time.Parse("2006-01-02 15:04:05 -0700", fields[1]); err == nil {
			creationTime = t
		}

		info := SnapshotInfo{
			Name:      snapName,
			Machine:   machine,
			Backend:   string(BackendLVM),
			LV:        fmt.Sprintf("%s/%s", vgName, lv),
			CreatedAt: FormatCreatedAt(creationTime),
		}

		snapshots = append(snapshots, info)
	}

	return snapshots, nil
}

func (l *LVMBackend) RollbackSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Rolling back machine '%s' to LVM snapshot '%s'", machine, snapName))

	vgName, lvName, err := l.getMachineLVInfo(rootfsPath)
	if err != nil {
		return err
	}

	snapLVName := fmt.Sprintf("%s-snap-%s", lvName, snapName)

	lvconvertPath := findLVMBinary("lvconvert")
	cmd := exec.Command("sudo", lvconvertPath, "--merge", fmt.Sprintf("%s/%s", vgName, snapLVName))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge LVM snapshot: %w (%s)", err, string(output))
	}

	logger.Success(fmt.Sprintf("Machine '%s' will be rolled back to LVM snapshot '%s' on next activation", machine, snapName))
	return nil
}

func (l *LVMBackend) isSnapshotMerging(vgName, snapLVName string) bool {
	lvsPath := findLVMBinary("lvs")
	cmd := exec.Command("sudo", lvsPath, "--noheadings", "-o", "lv_attr", fmt.Sprintf("%s/%s", vgName, snapLVName))
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	attrs := strings.TrimSpace(string(out))
	if len(attrs) < 1 {
		return false
	}

	return attrs[0] == 'v'
}

func (l *LVMBackend) DeleteSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Deleting LVM snapshot '%s' for machine '%s'", snapName, machine))

	vgName, lvName, err := l.getMachineLVInfo(rootfsPath)
	if err != nil {
		return err
	}

	snapLVName := fmt.Sprintf("%s-snap-%s", lvName, snapName)

	if l.isSnapshotMerging(vgName, snapLVName) {
		return fmt.Errorf("snapshot '%s' is currently merging and cannot be deleted. "+
			"Wait for merge to complete (machine restart required), then retry", snapName)
	}

	lvremovePath := findLVMBinary("lvremove")
	cmd := exec.Command("sudo", lvremovePath, "-f", fmt.Sprintf("%s/%s", vgName, snapLVName))
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "merging") || strings.Contains(outputStr, "merge") {
			return fmt.Errorf("snapshot '%s' is currently merging and cannot be deleted. "+
				"Wait for merge to complete (machine restart required), then retry", snapName)
		}
		return fmt.Errorf("failed to delete LVM snapshot: %w (%s)", err, outputStr)
	}

	logger.Success(fmt.Sprintf("LVM snapshot '%s' deleted", snapName))
	return nil
}

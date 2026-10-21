package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

// LVMThinBackend implements StorageBackend using LVM thin provisioning
type LVMThinBackend struct {
	basePath string // Base path for mounting thin volumes (e.g., /var/lib/machines)
	vgName   string // Volume group name
	poolName string // Thin pool name
}

// NewLVMThinBackend creates a new LVM thin-based storage backend
func NewLVMThinBackend(basePath, vgName, poolName string) (*LVMThinBackend, error) {
	// Verify thin pool exists
	if err := verifyThinPool(vgName, poolName); err != nil {
		return nil, err
	}

	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path %s: %w", basePath, err)
	}

	return &LVMThinBackend{
		basePath: basePath,
		vgName:   vgName,
		poolName: poolName,
	}, nil
}

// Type returns the backend type
func (l *LVMThinBackend) Type() BackendType {
	return BackendLVMThin
}

// CreateMachine creates a thin volume and mounts it
func (l *LVMThinBackend) CreateMachine(name string, sizeGB int) error {
	machinePath := l.GetMachinePath(name)
	lvName := fmt.Sprintf("lxspawn-%s", name)

	// Check if already exists
	if _, err := os.Stat(machinePath); err == nil {
		return fmt.Errorf("machine storage already exists at %s", machinePath)
	}

	// Create thin volume

	cmd := deploy.SudoHelper("lvcreate",
		"--type", "thin",
		"--name", lvName,
		"--virtualsize", fmt.Sprintf("%dG", sizeGB),
		"--thinpool", l.poolName,
		l.vgName)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create thin volume %s: %w (%s)", lvName, err, string(output))
	}

	// Format with ext4
	lvPath := fmt.Sprintf("/dev/%s/%s", l.vgName, lvName)

	cmd = deploy.SudoHelper("mkfs-ext4", "-q", lvPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Clean up LV
		deploy.SudoHelper("lvremove", "-f", lvPath).Run()
		return fmt.Errorf("failed to format %s: %w (%s)", lvPath, err, string(output))
	}

	// Create mount point
	if err := os.MkdirAll(machinePath, 0755); err != nil {
		deploy.SudoHelper("lvremove", "-f", lvPath).Run()
		return fmt.Errorf("failed to create mount point %s: %w", machinePath, err)
	}

	// Mount
	cmd = deploy.SudoHelper("mount", lvPath, machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(machinePath)
		deploy.SudoHelper("lvremove", "-f", lvPath).Run()
		return fmt.Errorf("failed to mount %s at %s: %w (%s)", lvPath, machinePath, err, string(output))
	}

	// Set ownership (use sudo since freshly mounted ext4 is root-owned)
	cmd = deploy.SudoHelper("chown", "root:lxspawn", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("failed to set ownership on %s: %v (%s)", machinePath, err, string(output)))
	}

	// Set permissions (use sudo since freshly mounted ext4 is root-owned)
	cmd = deploy.SudoHelper("chmod", "0775", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("failed to set permissions on %s: %v (%s)", machinePath, err, string(output)))
	}

	logger.Info(fmt.Sprintf("Created thin volume %s and mounted at %s", lvPath, machinePath))
	return nil
}

// DeleteMachine unmounts and removes the thin volume
func (l *LVMThinBackend) DeleteMachine(name string) error {
	machinePath := l.GetMachinePath(name)
	lvName := fmt.Sprintf("lxspawn-%s", name)
	lvPath := fmt.Sprintf("/dev/%s/%s", l.vgName, lvName)

	// Unmount
	cmd := deploy.SudoHelper("umount", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), "not mounted") {
			logger.Warn(fmt.Sprintf("failed to unmount %s: %v (%s)", machinePath, err, string(output)))
		}
	}

	// Remove mount point
	if err := os.Remove(machinePath); err != nil && !os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("failed to remove mount point %s: %v", machinePath, err))
	}

	// Remove LV

	cmd = deploy.SudoHelper("lvremove", "-f", lvPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove thin volume %s: %w (%s)", lvPath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Removed thin volume %s", lvPath))
	return nil
}

// MachineExists checks if the thin volume exists and is mounted
func (l *LVMThinBackend) MachineExists(name string) (bool, error) {
	machinePath := l.GetMachinePath(name)
	lvName := fmt.Sprintf("lxspawn-%s", name)

	// Check if LV exists

	cmd := deploy.SudoHelper("lvs", fmt.Sprintf("%s/%s", l.vgName, lvName))
	if err := cmd.Run(); err != nil {
		return false, nil
	}

	// Check if mounted
	cmd = exec.Command("mountpoint", "-q", machinePath)
	if err := cmd.Run(); err != nil {
		return false, nil
	}

	return true, nil
}

// GetMachinePath returns the mount point for a machine
func (l *LVMThinBackend) GetMachinePath(name string) string {
	return filepath.Join(l.basePath, name)
}

// CreateSnapshot creates an LVM thin snapshot
func (l *LVMThinBackend) CreateSnapshot(machine, snapName string) error {
	lvName := fmt.Sprintf("lxspawn-%s", machine)
	snapLVName := fmt.Sprintf("%s-snap-%s", lvName, snapName)

	// Check if machine exists
	exists, err := l.MachineExists(machine)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("machine %s does not exist", machine)
	}

	// Check if snapshot already exists

	cmd := deploy.SudoHelper("lvs", fmt.Sprintf("%s/%s", l.vgName, snapLVName))
	if err := cmd.Run(); err == nil {
		return fmt.Errorf("snapshot %s already exists", snapName)
	}

	// Create thin snapshot

	cmd = deploy.SudoHelper("lvcreate",
		"--snapshot",
		"--name", snapLVName,
		fmt.Sprintf("%s/%s", l.vgName, lvName))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create thin snapshot: %w (%s)", err, string(output))
	}

	logger.Info(fmt.Sprintf("Created thin snapshot %s for machine %s", snapName, machine))
	return nil
}

// DeleteSnapshot deletes a thin snapshot
func (l *LVMThinBackend) DeleteSnapshot(machine, snapName string) error {
	lvName := fmt.Sprintf("lxspawn-%s", machine)
	snapLVName := fmt.Sprintf("%s-snap-%s", lvName, snapName)

	// Check if snapshot is merging
	if l.isSnapshotMerging(snapLVName) {
		return fmt.Errorf("snapshot %s is currently merging and cannot be deleted. Wait for merge to complete (machine restart required), then retry", snapName)
	}

	// Remove snapshot LV

	cmd := deploy.SudoHelper("lvremove", "-f", fmt.Sprintf("%s/%s", l.vgName, snapLVName))
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "merging") || strings.Contains(outputStr, "merge") {
			return fmt.Errorf("snapshot %s is currently merging and cannot be deleted. Wait for merge to complete (machine restart required), then retry", snapName)
		}
		return fmt.Errorf("failed to delete thin snapshot %s: %w (%s)", snapLVName, err, outputStr)
	}

	logger.Info(fmt.Sprintf("Deleted thin snapshot %s for machine %s", snapName, machine))
	return nil
}

// ListSnapshots lists all thin snapshots for a machine
func (l *LVMThinBackend) ListSnapshots(machine string) ([]SnapshotInfo, error) {
	lvName := fmt.Sprintf("lxspawn-%s", machine)
	prefix := fmt.Sprintf("%s-snap-", lvName)


	cmd := deploy.SudoHelper("lvs",
		"--noheadings",
		"-o", "lv_name,lv_time",
		"--select", fmt.Sprintf("vg_name=%s", l.vgName))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list LVM volumes: %w", err)
	}

	var snapshots []SnapshotInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		lv := strings.TrimSpace(fields[0])
		if !strings.HasPrefix(lv, prefix) {
			continue
		}

		snapName := strings.TrimPrefix(lv, prefix)
		createdAt := time.Now()
		if t, err := time.Parse("2006-01-02 15:04:05 -0700", fields[1]); err == nil {
			createdAt = t
		}

		snapshots = append(snapshots, SnapshotInfo{
			Name:      snapName,
			Machine:   machine,
			Backend:   string(BackendLVMThin),
			CreatedAt: createdAt,
			Location:  fmt.Sprintf("%s/%s", l.vgName, lv),
		})
	}

	return snapshots, nil
}

// RollbackSnapshot rolls back a machine to a thin snapshot
func (l *LVMThinBackend) RollbackSnapshot(machine, snapName string) error {
	lvName := fmt.Sprintf("lxspawn-%s", machine)
	snapLVName := fmt.Sprintf("%s-snap-%s", lvName, snapName)

	// Merge snapshot

	cmd := deploy.SudoHelper("lvconvert",
		"--merge",
		fmt.Sprintf("%s/%s", l.vgName, snapLVName))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge thin snapshot: %w (%s)", err, string(output))
	}

	logger.Info(fmt.Sprintf("Machine %s will be rolled back to thin snapshot %s on next activation", machine, snapName))
	return nil
}

// GetUsage returns LVM thin pool usage
func (l *LVMThinBackend) GetUsage() (int64, int64, error) {

	cmd := deploy.SudoHelper("lvs",
		"--noheadings",
		"--nosuffix",
		"--units", "b",
		"-o", "lv_size,data_percent",
		fmt.Sprintf("%s/%s", l.vgName, l.poolName))

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get thin pool usage: %w", err)
	}

	fields := strings.Fields(string(output))
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected lvs output")
	}

	var totalBytes, dataPercent float64
	fmt.Sscanf(fields[0], "%f", &totalBytes)
	fmt.Sscanf(fields[1], "%f", &dataPercent)

	usedBytes := int64(totalBytes * dataPercent / 100)
	return usedBytes, int64(totalBytes), nil
}

// isSnapshotMerging checks if a snapshot is in merging state
func (l *LVMThinBackend) isSnapshotMerging(snapLVName string) bool {

	cmd := deploy.SudoHelper("lvs", "--noheadings", "-o", "lv_attr", fmt.Sprintf("%s/%s", l.vgName, snapLVName))
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

// verifyThinPool checks if a thin pool exists
func verifyThinPool(vgName, poolName string) error {

	cmd := deploy.SudoHelper("lvs",
		"--noheadings",
		"-o", "lv_attr",
		fmt.Sprintf("%s/%s", vgName, poolName))

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("thin pool %s/%s not found: %w", vgName, poolName, err)
	}

	attrs := strings.TrimSpace(string(output))
	if len(attrs) < 1 {
		return fmt.Errorf("thin pool %s/%s not found", vgName, poolName)
	}

	// Check if it's a thin pool (attribute 't' or 'T')
	if attrs[0] != 't' && attrs[0] != 'T' {
		return fmt.Errorf("%s/%s is not a thin pool", vgName, poolName)
	}

	return nil
}

// DetectThinPool attempts to find a thin pool in the VG
func DetectThinPool(vgName string) (string, error) {

	cmd := deploy.SudoHelper("lvs",
		"--noheadings",
		"-o", "lv_name,lv_attr",
		"--select", fmt.Sprintf("vg_name=%s", vgName))

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list LVs in VG %s: %w", vgName, err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		lvName := strings.TrimSpace(fields[0])
		attrs := strings.TrimSpace(fields[1])

		// Check if it's a thin pool
		if len(attrs) > 0 && (attrs[0] == 't' || attrs[0] == 'T') {
			return lvName, nil
		}
	}

	return "", fmt.Errorf("no thin pool found in VG %s", vgName)
}

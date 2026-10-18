package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/logger"
)

// BtrfsBackend implements StorageBackend using btrfs subvolumes
type BtrfsBackend struct {
	basePath string // Base path for machine storage (e.g., /var/lib/machines)
}

// NewBtrfsBackend creates a new btrfs-based storage backend
func NewBtrfsBackend(basePath string) (*BtrfsBackend, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path %s: %w", basePath, err)
	}

	return &BtrfsBackend{basePath: basePath}, nil
}

// Type returns the backend type
func (b *BtrfsBackend) Type() BackendType {
	return BackendBtrfs
}

// CreateMachine creates a btrfs subvolume for the machine
func (b *BtrfsBackend) CreateMachine(name string, sizeGB int) error {
	machinePath := b.GetMachinePath(name)

	// Check if already exists
	if _, err := os.Stat(machinePath); err == nil {
		return fmt.Errorf("machine storage already exists at %s", machinePath)
	}

	// Create parent directory if needed
	parentDir := filepath.Dir(machinePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	// Create btrfs subvolume
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	cmd := exec.Command("sudo", btrfsPath, "subvolume", "create", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create btrfs subvolume %s: %w (%s)", machinePath, err, string(output))
	}

	// Set ownership to root:lxspawn
	if err := setOwnership(machinePath, "root", "lxspawn"); err != nil {
		logger.Warn(fmt.Sprintf("failed to set ownership on %s: %v", machinePath, err))
	}

	// Set group-writable permissions
	cmd = exec.Command("sudo", "chmod", "0775", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("failed to set permissions on %s: %v (%s)", machinePath, err, string(output)))
	}

	logger.Info(fmt.Sprintf("Created btrfs subvolume at %s", machinePath))
	return nil
}

// DeleteMachine removes the btrfs subvolume
func (b *BtrfsBackend) DeleteMachine(name string) error {
	machinePath := b.GetMachinePath(name)

	// Check if exists
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		return nil // Already gone
	}

	// First, delete any child subvolumes (snapshots)
	if err := b.deleteChildSubvolumes(machinePath); err != nil {
		logger.Warn(fmt.Sprintf("failed to delete child subvolumes: %v", err))
	}

	// Delete the subvolume
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	cmd := exec.Command("sudo", btrfsPath, "subvolume", "delete", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete btrfs subvolume %s: %w (%s)", machinePath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Removed btrfs subvolume at %s", machinePath))
	return nil
}

// MachineExists checks if the btrfs subvolume exists
func (b *BtrfsBackend) MachineExists(name string) (bool, error) {
	machinePath := b.GetMachinePath(name)

	// Check if path exists
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		return false, nil
	}

	// Check if it's a btrfs subvolume
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	cmd := exec.Command("sudo", btrfsPath, "subvolume", "show", machinePath)
	if err := cmd.Run(); err != nil {
		return false, nil // Not a subvolume
	}

	return true, nil
}

// GetMachinePath returns the filesystem path for a machine
func (b *BtrfsBackend) GetMachinePath(name string) string {
	return filepath.Join(b.basePath, name)
}

// CreateSnapshot creates a read-only btrfs snapshot
func (b *BtrfsBackend) CreateSnapshot(machine, snapName string) error {
	machinePath := b.GetMachinePath(machine)
	snapshotPath := b.getSnapshotPath(machine, snapName)

	// Check if machine exists
	exists, err := b.MachineExists(machine)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("machine %s does not exist", machine)
	}

	// Check if snapshot already exists
	if _, err := os.Stat(snapshotPath); err == nil {
		return fmt.Errorf("snapshot %s already exists", snapName)
	}

	// Create snapshots directory
	snapshotsDir := filepath.Dir(snapshotPath)
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshots directory: %w", err)
	}

	// Create read-only snapshot
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	cmd := exec.Command("sudo", btrfsPath, "subvolume", "snapshot", "-r", machinePath, snapshotPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create btrfs snapshot: %w (%s)", err, string(output))
	}

	logger.Info(fmt.Sprintf("Created btrfs snapshot %s for machine %s", snapName, machine))
	return nil
}

// DeleteSnapshot deletes a btrfs snapshot
func (b *BtrfsBackend) DeleteSnapshot(machine, snapName string) error {
	snapshotPath := b.getSnapshotPath(machine, snapName)

	// Check if exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s does not exist", snapName)
	}

	// Delete the snapshot
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	cmd := exec.Command("sudo", btrfsPath, "subvolume", "delete", snapshotPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete btrfs snapshot %s: %w (%s)", snapshotPath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Deleted snapshot %s for machine %s", snapName, machine))
	return nil
}

// ListSnapshots lists all snapshots for a machine
func (b *BtrfsBackend) ListSnapshots(machine string) ([]SnapshotInfo, error) {
	machinePath := b.GetMachinePath(machine)

	// List child subvolumes
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	cmd := exec.Command("sudo", btrfsPath, "subvolume", "list", "-o", machinePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list btrfs subvolumes: %w", err)
	}

	var snapshots []SnapshotInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line: ID <id> gen <gen> top level <top> path <path>
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		subvolPath := fields[len(fields)-1]
		snapName := filepath.Base(subvolPath)

		// Only include snapshots (in .snapshots directory)
		if !strings.Contains(subvolPath, ".snapshots") {
			continue
		}

		snapshots = append(snapshots, SnapshotInfo{
			Name:      snapName,
			Machine:   machine,
			Backend:   string(BackendBtrfs),
			CreatedAt: time.Now(), // btrfs doesn't store creation time in subvolume list
			Location:  subvolPath,
		})
	}

	return snapshots, nil
}

// RollbackSnapshot rolls back a machine to a snapshot
func (b *BtrfsBackend) RollbackSnapshot(machine, snapName string) error {
	machinePath := b.GetMachinePath(machine)
	snapshotPath := b.getSnapshotPath(machine, snapName)

	// Check if snapshot exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s does not exist", snapName)
	}

	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	// Create writable snapshot
	rollbackPath := machinePath + ".rollback-" + fmt.Sprintf("%d", time.Now().Unix())
	cmd := exec.Command("sudo", btrfsPath, "subvolume", "snapshot", snapshotPath, rollbackPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create writable snapshot: %w (%s)", err, string(output))
	}

	// Delete current machine subvolume (and all children)
	if err := b.deleteChildSubvolumes(machinePath); err != nil {
		logger.Warn(fmt.Sprintf("failed to delete child subvolumes: %v", err))
	}

	cmd = exec.Command("sudo", btrfsPath, "subvolume", "delete", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try to clean up rollback snapshot
		exec.Command("sudo", btrfsPath, "subvolume", "delete", rollbackPath).Run()
		return fmt.Errorf("failed to delete current subvolume: %w (%s)", err, string(output))
	}

	// Rename rollback snapshot to machine path
	cmd = exec.Command("sudo", "mv", rollbackPath, machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename rollback snapshot: %w (%s)", err, string(output))
	}

	logger.Info(fmt.Sprintf("Rolled back machine %s to snapshot %s", machine, snapName))
	return nil
}

// GetUsage returns btrfs filesystem usage
func (b *BtrfsBackend) GetUsage() (int64, int64, error) {
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	cmd := exec.Command("sudo", btrfsPath, "filesystem", "usage", "-b", b.basePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get btrfs usage: %w", err)
	}

	// Parse output for "Device size" and "Used"
	var total, used int64
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Device size") {
			fmt.Sscanf(line, "Device size: %d", &total)
		}
		if strings.Contains(line, "Used") {
			fmt.Sscanf(line, "Used: %d", &used)
		}
	}

	return used, total, nil
}

// getSnapshotPath returns the path for a snapshot
func (b *BtrfsBackend) getSnapshotPath(machine, snapName string) string {
	return filepath.Join(b.getSnapshotsDir(machine), snapName)
}

// getSnapshotsDir returns the directory containing snapshots for a machine
func (b *BtrfsBackend) getSnapshotsDir(machine string) string {
	return filepath.Join(b.basePath, ".snapshots", machine)
}

// deleteChildSubvolumes recursively deletes child subvolumes
func (b *BtrfsBackend) deleteChildSubvolumes(parentPath string) error {
	btrfsPath, err := exec.LookPath("btrfs")
	if err != nil {
		btrfsPath = "/usr/sbin/btrfs"
	}

	// List child subvolumes
	cmd := exec.Command("sudo", btrfsPath, "subvolume", "list", "-o", parentPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list child subvolumes: %w", err)
	}

	// Delete children (deepest first)
	lines := strings.Split(string(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		childPath := fields[len(fields)-1]
		// Make path absolute if relative
		if !filepath.IsAbs(childPath) {
			// Get the root of the btrfs filesystem
			cmd := exec.Command("sudo", btrfsPath, "subvolume", "show", parentPath)
			showOutput, err := cmd.Output()
			if err != nil {
				continue
			}
			// Parse "Path:" line to get absolute path
			for _, showLine := range strings.Split(string(showOutput), "\n") {
				if strings.Contains(showLine, "Path:") {
					parts := strings.Fields(showLine)
					if len(parts) >= 2 {
						childPath = filepath.Join(b.basePath, childPath)
					}
					break
				}
			}
		}

		cmd := exec.Command("sudo", btrfsPath, "subvolume", "delete", childPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			logger.Warn(fmt.Sprintf("failed to delete child subvolume %s: %v (%s)", childPath, err, string(output)))
		}
	}

	return nil
}

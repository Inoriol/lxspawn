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

// DirectoryBackend implements StorageBackend using simple directories
type DirectoryBackend struct {
	basePath string // Base path for machine storage (e.g., /var/lib/machines)
}

// NewDirectoryBackend creates a new directory-based storage backend
func NewDirectoryBackend(basePath string) (*DirectoryBackend, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path %s: %w", basePath, err)
	}

	return &DirectoryBackend{basePath: basePath}, nil
}

// Type returns the backend type
func (d *DirectoryBackend) Type() BackendType {
	return BackendDirectory
}

// CreateMachine creates a directory for the machine
func (d *DirectoryBackend) CreateMachine(name string, sizeGB int) error {
	machinePath := d.GetMachinePath(name)

	// Check if already exists
	if _, err := os.Stat(machinePath); err == nil {
		return fmt.Errorf("machine storage already exists at %s", machinePath)
	}

	// Create directory with proper permissions
	if err := os.MkdirAll(machinePath, 0755); err != nil {
		return fmt.Errorf("failed to create machine directory %s: %w", machinePath, err)
	}

	// Set ownership to root:lxspawn if group exists
	if err := setOwnership(machinePath, "root", "lxspawn"); err != nil {
		logger.Warn(fmt.Sprintf("failed to set ownership on %s: %v", machinePath, err))
	}

	// Set group-writable permissions (use sudo since directory is now owned by root)
	cmd := deploy.SudoHelper("chmod", "0775", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("failed to set permissions on %s: %v (%s)", machinePath, err, string(output)))
	}

	logger.Info(fmt.Sprintf("Created directory storage at %s", machinePath))
	return nil
}

// DeleteMachine removes the machine directory
func (d *DirectoryBackend) DeleteMachine(name string) error {
	machinePath := d.GetMachinePath(name)

	// Check if exists
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		return nil // Already gone
	}

	// Use sudo rm -rf to handle files owned by root (from container namespace)
	cmd := deploy.SudoHelper("rm-rf", "-rf", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove machine directory %s: %w (%s)", machinePath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Removed directory storage at %s", machinePath))
	return nil
}

// MachineExists checks if the machine directory exists
func (d *DirectoryBackend) MachineExists(name string) (bool, error) {
	machinePath := d.GetMachinePath(name)
	_, err := os.Stat(machinePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetMachinePath returns the filesystem path for a machine
func (d *DirectoryBackend) GetMachinePath(name string) string {
	return filepath.Join(d.basePath, name)
}

// CreateSnapshot creates a snapshot by copying the directory
func (d *DirectoryBackend) CreateSnapshot(machine, snapName string) error {
	machinePath := d.GetMachinePath(machine)
	snapshotPath := d.getSnapshotPath(machine, snapName)

	// Check if machine exists
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		return fmt.Errorf("machine %s does not exist", machine)
	}

	// Check if snapshot already exists
	if _, err := os.Stat(snapshotPath); err == nil {
		return fmt.Errorf("snapshot %s already exists", snapName)
	}

	// Create snapshot directory
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Use rsync for efficient copying if available, otherwise use cp
	if _, err := exec.LookPath("rsync"); err == nil {
		cmd := exec.Command("rsync", "-a", "--delete", machinePath+"/", snapshotPath+"/")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("rsync failed: %w (%s)", err, string(output))
		}
	} else {
		// Fallback to cp
		cmd := exec.Command("cp", "-a", machinePath, snapshotPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("cp failed: %w (%s)", err, string(output))
		}
	}

	logger.Info(fmt.Sprintf("Created directory snapshot %s for machine %s", snapName, machine))
	return nil
}

// DeleteSnapshot deletes a snapshot directory
func (d *DirectoryBackend) DeleteSnapshot(machine, snapName string) error {
	snapshotPath := d.getSnapshotPath(machine, snapName)

	// Check if exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s does not exist", snapName)
	}

	// Use sudo rm -rf to handle files owned by root
	cmd := deploy.SudoHelper("rm-rf", "-rf", snapshotPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove snapshot %s: %w (%s)", snapName, err, string(output))
	}

	logger.Info(fmt.Sprintf("Deleted directory snapshot %s for machine %s", snapName, machine))
	return nil
}

// ListSnapshots lists all snapshots for a machine
func (d *DirectoryBackend) ListSnapshots(machine string) ([]SnapshotInfo, error) {
	snapshotsDir := d.getSnapshotsDir(machine)
	var snapshots []SnapshotInfo

	// Check if snapshots directory exists
	if _, err := os.Stat(snapshotsDir); os.IsNotExist(err) {
		return snapshots, nil // No snapshots
	}

	// Read directory entries
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		snapshots = append(snapshots, SnapshotInfo{
			Name:      entry.Name(),
			Machine:   machine,
			Backend:   string(BackendDirectory),
			CreatedAt: info.ModTime(),
			Location:  filepath.Join(snapshotsDir, entry.Name()),
		})
	}

	return snapshots, nil
}

// RollbackSnapshot rolls back a machine to a snapshot
func (d *DirectoryBackend) RollbackSnapshot(machine, snapName string) error {
	machinePath := d.GetMachinePath(machine)
	snapshotPath := d.getSnapshotPath(machine, snapName)

	// Check if snapshot exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s does not exist", snapName)
	}

	// Create backup of current state
	backupPath := machinePath + ".rollback-backup-" + fmt.Sprintf("%d", time.Now().Unix())
	if err := os.Rename(machinePath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current state: %w", err)
	}

	// Copy snapshot to machine path
	cmd := exec.Command("cp", "-a", snapshotPath, machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Restore backup
		os.Rename(backupPath, machinePath)
		return fmt.Errorf("failed to restore snapshot: %w (%s)", err, string(output))
	}

	// Remove backup using sudo rm -rf
	cmd = deploy.SudoHelper("rm-rf", backupPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("failed to remove backup %s: %v (%s)", backupPath, err, string(output)))
	}

	logger.Info(fmt.Sprintf("Rolled back machine %s to snapshot %s", machine, snapName))
	return nil
}

// GetUsage returns storage usage for the base path
func (d *DirectoryBackend) GetUsage() (int64, int64, error) {
	// Use df to get filesystem usage
	cmd := exec.Command("df", "-B1", "--output=used,size", d.basePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get filesystem usage: %w", err)
	}

	// Parse df output (skip header line)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("unexpected df output")
	}

	// Parse the data line
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected df output format")
	}

	var used, total int64
	fmt.Sscanf(fields[0], "%d", &used)
	fmt.Sscanf(fields[1], "%d", &total)

	return used, total, nil
}

// getSnapshotPath returns the path for a snapshot
func (d *DirectoryBackend) getSnapshotPath(machine, snapName string) string {
	return filepath.Join(d.getSnapshotsDir(machine), snapName)
}

// getSnapshotsDir returns the directory containing snapshots for a machine
func (d *DirectoryBackend) getSnapshotsDir(machine string) string {
	return filepath.Join(d.basePath, ".snapshots", machine)
}

// setOwnership sets the ownership of a path
func setOwnership(path, user, group string) error {
	cmd := deploy.SudoHelper("chown", fmt.Sprintf("%s:%s", user, group), path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %w (%s)", err, string(output))
	}
	return nil
}

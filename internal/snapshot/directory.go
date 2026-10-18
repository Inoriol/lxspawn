package snapshot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type DirectoryBackend struct {
	machineName string
}

func NewDirectoryBackend(machineName string) (*DirectoryBackend, error) {
	return &DirectoryBackend{
		machineName: machineName,
	}, nil
}

func (b *DirectoryBackend) Type() BackendType {
	return BackendDirectory
}

func (b *DirectoryBackend) CreateSnapshot(machine, snapName, rootfsPath string) error {
	snapDir := filepath.Join(rootfsPath, ".snapshots", snapName)
	
	// Create snapshot directory
	if err := os.MkdirAll(filepath.Dir(snapDir), 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Use rsync to copy the rootfs (preserves permissions, handles large files)
	cmd := exec.Command("sudo", "rsync", "-a", "--delete", rootfsPath+"/", snapDir+"/", "--exclude=.snapshots")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync failed: %w (%s)", err, string(output))
	}

	return nil
}

func (b *DirectoryBackend) ListSnapshots(machine, rootfsPath string) ([]SnapshotInfo, error) {
	snapDir := filepath.Join(rootfsPath, ".snapshots")
	
	// Check if snapshot directory exists
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		return []SnapshotInfo{}, nil
	}

	entries, err := os.ReadDir(snapDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var snapshots []SnapshotInfo
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
			CreatedAt: info.ModTime().Format(time.RFC3339),
			Path:      filepath.Join(snapDir, entry.Name()),
		})
	}

	return snapshots, nil
}

func (b *DirectoryBackend) RollbackSnapshot(machine, snapName, rootfsPath string) error {
	snapDir := filepath.Join(rootfsPath, ".snapshots", snapName)
	
	// Check if snapshot exists
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' does not exist", snapName)
	}

	// Use rsync to restore from snapshot
	// Use --checksum to ensure files are compared by content, not just size/mtime
	cmd := exec.Command("sudo", "rsync", "-a", "--delete", "--checksum", snapDir+"/", rootfsPath+"/", "--exclude=.snapshots")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync failed: %w (%s)", err, string(output))
	}

	return nil
}

func (b *DirectoryBackend) DeleteSnapshot(machine, snapName, rootfsPath string) error {
	snapDir := filepath.Join(rootfsPath, ".snapshots", snapName)
	
	// Check if snapshot exists
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' does not exist", snapName)
	}

	// Remove snapshot directory
	cmd := exec.Command("sudo", "rm", "-rf", snapDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w (%s)", err, string(output))
	}

	return nil
}

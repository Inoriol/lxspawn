package snapshot

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

type BtrfsBackend struct {
	machineName string
}

func NewBtrfsBackend(machineName string) (*BtrfsBackend, error) {
	return &BtrfsBackend{machineName: machineName}, nil
}

func (b *BtrfsBackend) Type() BackendType {
	return BackendBtrfs
}

func (b *BtrfsBackend) snapshotDir(rootfsPath string) string {
	parentDir := filepath.Dir(rootfsPath)
	return filepath.Join(parentDir, b.machineName+".snapshots")
}

func (b *BtrfsBackend) CreateSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Creating btrfs snapshot '%s' for machine '%s'", snapName, machine))

	snapDir := b.snapshotDir(rootfsPath)
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		cmd := deploy.Sudo("mkdir", "-p", snapDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create snapshot directory: %w (%s)", err, string(output))
		}
	}

	snapPath := filepath.Join(snapDir, snapName)
	if _, err := os.Stat(snapPath); err == nil {
		return fmt.Errorf("snapshot '%s' already exists", snapName)
	}

	cmd := deploy.Sudo("btrfs", "subvolume", "snapshot", "-r", rootfsPath, snapPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create btrfs snapshot: %w (%s)", err, string(output))
	}

	logger.Success(fmt.Sprintf("Snapshot '%s' created at %s", snapName, snapPath))
	return nil
}

func (b *BtrfsBackend) ListSnapshots(machine, rootfsPath string) ([]SnapshotInfo, error) {
	snapDir := b.snapshotDir(rootfsPath)
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

		entryPath := filepath.Join(snapDir, entry.Name())
		if !isBtrfsSubvolume(entryPath) {
			continue
		}

		info := SnapshotInfo{
			Name:    entry.Name(),
			Machine: machine,
			Backend: string(BackendBtrfs),
			Path:    entryPath,
		}

		if createdAt := getBtrfsSubvolumeCreatedAt(entryPath); createdAt != nil {
			info.CreatedAt = FormatCreatedAt(*createdAt)
		} else if stat, err := os.Stat(entryPath); err == nil {
			info.CreatedAt = FormatCreatedAt(stat.ModTime())
		} else {
			info.CreatedAt = FormatCreatedAt(time.Now())
		}

		snapshots = append(snapshots, info)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Name < snapshots[j].Name
	})

	return snapshots, nil
}

func isBtrfsSubvolume(path string) bool {
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return false
	}
	return stat.Ino == 256 && stat.Mode&syscall.S_IFDIR != 0
}

func getBtrfsSubvolumeCreatedAt(path string) *time.Time {
	cmd := exec.Command("sudo", "btrfs", "subvolume", "show", path)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Creation time:") {
			timeStr := strings.TrimPrefix(line, "Creation time:")
			timeStr = strings.TrimSpace(timeStr)
			if t, err := time.Parse("2006-01-02 15:04:05 -0700", timeStr); err == nil {
				return &t
			}
		}
	}
	return nil
}

func (b *BtrfsBackend) listChildSubvolumes(rootfsPath string) ([]string, error) {
	entries, err := os.ReadDir(rootfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var subvols []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entryPath := filepath.Join(rootfsPath, entry.Name())
		if isBtrfsSubvolume(entryPath) {
			children, err := b.listChildSubvolumes(entryPath)
			if err != nil {
				return nil, err
			}
			subvols = append(subvols, children...)
			subvols = append(subvols, entryPath)
		}
	}

	return subvols, nil
}

func (b *BtrfsBackend) deleteSubvolumeTree(rootfsPath string) error {
	children, err := b.listChildSubvolumes(rootfsPath)
	if err != nil {
		return err
	}

	for _, child := range children {
		logger.Info(fmt.Sprintf("Deleting child subvolume: %s", child))
		cmd := deploy.Sudo("btrfs", "subvolume", "delete", child)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to delete child subvolume %s: %w (%s)", child, err, string(output))
		}
	}

	logger.Info(fmt.Sprintf("Deleting root subvolume: %s", rootfsPath))
	cmd := deploy.Sudo("btrfs", "subvolume", "delete", rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete root subvolume: %w (%s)", err, string(output))
	}

	return nil
}

func (b *BtrfsBackend) RollbackSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Rolling back machine '%s' to snapshot '%s'", machine, snapName))

	snapPath := filepath.Join(b.snapshotDir(rootfsPath), snapName)
	if _, err := os.Stat(snapPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' does not exist", snapName)
	}

	parentDir := filepath.Dir(rootfsPath)
	tempPath := filepath.Join(parentDir, b.machineName+".rollback-temp")
	oldPath := filepath.Join(parentDir, b.machineName+".rollback-old")

	if _, err := os.Stat(tempPath); err == nil {
		logger.Info("Cleaning up leftover rollback temp")
		if err := b.deleteSubvolumeTree(tempPath); err != nil {
			return fmt.Errorf("failed to clean up leftover temp: %w", err)
		}
	}

	if _, err := os.Stat(oldPath); err == nil {
		logger.Info("Cleaning up leftover rollback old")
		if err := b.deleteSubvolumeTree(oldPath); err != nil {
			return fmt.Errorf("failed to clean up leftover old: %w", err)
		}
	}

	cmd := deploy.Sudo("btrfs", "subvolume", "snapshot", snapPath, tempPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create writable snapshot for rollback: %w (%s)", err, string(output))
	}

	cmd = deploy.Sudo("mv", rootfsPath, oldPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		deploy.Sudo("btrfs", "subvolume", "delete", tempPath)
		return fmt.Errorf("failed to move old rootfs aside: %w (%s)", err, string(output))
	}

	cmd = deploy.Sudo("mv", tempPath, rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		deploy.Sudo("mv", oldPath, rootfsPath)
		return fmt.Errorf("failed to rename restored snapshot: %w (%s)", err, string(output))
	}

	if err := b.deleteSubvolumeTree(oldPath); err != nil {
		logger.Warn(fmt.Sprintf("failed to delete old rootfs (can be cleaned up manually): %v", err))
	}

	cmd = deploy.Sudo("chown", "-R", "0:0", rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("chown warning: %v (%s)", err, string(output)))
	}

	logger.Success(fmt.Sprintf("Machine '%s' rolled back to snapshot '%s'", machine, snapName))
	return nil
}

func (b *BtrfsBackend) DeleteSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Deleting snapshot '%s' for machine '%s'", snapName, machine))

	snapPath := filepath.Join(b.snapshotDir(rootfsPath), snapName)
	if _, err := os.Stat(snapPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot '%s' does not exist", snapName)
	}

	cmd := deploy.Sudo("btrfs", "subvolume", "delete", snapPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete btrfs snapshot: %w (%s)", err, string(output))
	}

	logger.Success(fmt.Sprintf("Snapshot '%s' deleted", snapName))
	return nil
}

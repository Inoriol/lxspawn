package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	return filepath.Join(rootfsPath, ".snapshots")
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

	out, err := deploy.Sudo("btrfs", "subvolume", "list", "-o", rootfsPath).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list btrfs subvolumes: %w", err)
	}

	var snapshots []SnapshotInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		subvolPath := fields[len(fields)-1]

		fullPath := filepath.Join("/", subvolPath)
		if !strings.HasPrefix(fullPath, snapDir) {
			continue
		}

		snapName := filepath.Base(subvolPath)
		if snapName == ".snapshots" {
			continue
		}

		info := SnapshotInfo{
			Name:    snapName,
			Machine: machine,
			Backend: string(BackendBtrfs),
			Path:    fullPath,
		}

		if stat, err := os.Stat(fullPath); err == nil {
			info.CreatedAt = FormatCreatedAt(stat.ModTime())
		} else {
			info.CreatedAt = FormatCreatedAt(time.Now())
		}

		snapshots = append(snapshots, info)
	}

	return snapshots, nil
}

func (b *BtrfsBackend) listChildSubvolumes(rootfsPath string) ([]string, error) {
	out, err := deploy.Sudo("btrfs", "subvolume", "list", "-o", rootfsPath).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list child subvolumes: %w", err)
	}

	var subvols []string
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		subvolPath := filepath.Join("/", fields[len(fields)-1])
		subvols = append(subvols, subvolPath)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(subvols)))
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

	if _, err := os.Stat(tempPath); err == nil {
		logger.Info("Cleaning up leftover rollback temp")
		if err := b.deleteSubvolumeTree(tempPath); err != nil {
			return fmt.Errorf("failed to clean up leftover temp: %w", err)
		}
	}

	cmd := deploy.Sudo("btrfs", "subvolume", "snapshot", snapPath, tempPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create writable snapshot for rollback: %w (%s)", err, string(output))
	}

	if err := b.deleteSubvolumeTree(rootfsPath); err != nil {
		deploy.Sudo("btrfs", "subvolume", "delete", tempPath)
		return fmt.Errorf("failed to delete old rootfs: %w", err)
	}

	cmd = deploy.Sudo("mv", tempPath, rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename restored snapshot: %w (%s)", err, string(output))
	}

	cmd = deploy.Sudo("chown", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), rootfsPath)
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

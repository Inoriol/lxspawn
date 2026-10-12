package snapshot

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/logger"
)

type ZFSBackend struct {
	machineName string
}

func NewZFSBackend(machineName string) (*ZFSBackend, error) {
	return &ZFSBackend{machineName: machineName}, nil
}

func (z *ZFSBackend) Type() BackendType {
	return BackendZFS
}

func (z *ZFSBackend) getDataset(rootfsPath string) (string, error) {
	out, err := exec.Command("zfs", "list", "-H", "-o", "name", rootfsPath).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get ZFS dataset for %s: %w", rootfsPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (z *ZFSBackend) CreateSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Creating ZFS snapshot '%s' for machine '%s'", snapName, machine))

	dataset, err := z.getDataset(rootfsPath)
	if err != nil {
		return err
	}

	snapFullName := fmt.Sprintf("%s@%s", dataset, snapName)

	existing, err := exec.Command("zfs", "list", "-H", "-t", "snapshot", snapFullName).CombinedOutput()
	if err == nil && strings.TrimSpace(string(existing)) != "" {
		return fmt.Errorf("snapshot '%s' already exists", snapName)
	}

	cmd := exec.Command("sudo", "zfs", "snapshot", snapFullName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create ZFS snapshot: %w (%s)", err, string(output))
	}

	logger.Success(fmt.Sprintf("Snapshot '%s' created for dataset '%s'", snapName, dataset))
	return nil
}

func (z *ZFSBackend) ListSnapshots(machine, rootfsPath string) ([]SnapshotInfo, error) {
	dataset, err := z.getDataset(rootfsPath)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command("zfs", "list", "-H", "-t", "snapshot", "-o", "name,creation,used", "-r", dataset).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []SnapshotInfo{}, nil
		}
		return nil, fmt.Errorf("failed to list ZFS snapshots: %w", err)
	}

	var snapshots []SnapshotInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}

		fullSnapName := fields[0]
		parts := strings.SplitN(fullSnapName, "@", 2)
		if len(parts) != 2 {
			continue
		}

		snapName := parts[1]
		creationTime := time.Now()
		if len(fields) >= 2 {
			if t, err := time.Parse("Mon Jan 2 15:04 2006", fields[1]); err == nil {
				creationTime = t
			}
		}

		info := SnapshotInfo{
			Name:      snapName,
			Machine:   machine,
			Backend:   string(BackendZFS),
			Dataset:   dataset,
			CreatedAt: FormatCreatedAt(creationTime),
		}

		snapshots = append(snapshots, info)
	}

	return snapshots, nil
}

func (z *ZFSBackend) RollbackSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Rolling back machine '%s' to ZFS snapshot '%s'", machine, snapName))

	dataset, err := z.getDataset(rootfsPath)
	if err != nil {
		return err
	}

	snapFullName := fmt.Sprintf("%s@%s", dataset, snapName)

	existing, err := exec.Command("zfs", "list", "-H", "-t", "snapshot", snapFullName).CombinedOutput()
	if err != nil || strings.TrimSpace(string(existing)) == "" {
		return fmt.Errorf("snapshot '%s' does not exist", snapName)
	}

	cmd := exec.Command("sudo", "zfs", "rollback", "-r", snapFullName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rollback ZFS snapshot: %w (%s)", err, string(output))
	}

	logger.Success(fmt.Sprintf("Machine '%s' rolled back to ZFS snapshot '%s'", machine, snapName))
	return nil
}

func (z *ZFSBackend) DeleteSnapshot(machine, snapName, rootfsPath string) error {
	logger.Action(fmt.Sprintf("Deleting ZFS snapshot '%s' for machine '%s'", snapName, machine))

	dataset, err := z.getDataset(rootfsPath)
	if err != nil {
		return err
	}

	snapFullName := fmt.Sprintf("%s@%s", dataset, snapName)

	cmd := exec.Command("sudo", "zfs", "destroy", snapFullName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete ZFS snapshot: %w (%s)", err, string(output))
	}

	logger.Success(fmt.Sprintf("ZFS snapshot '%s' deleted", snapName))
	return nil
}

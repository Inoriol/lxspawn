package snapshot

import (
	"fmt"
	"os/exec"
	"strings"
)

func DetectFilesystem(path string) (BackendType, error) {
	// Check filesystem type
	out, err := exec.Command("stat", "-f", "-c", "%T", path).Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect filesystem type for %s: %w", path, err)
	}

	fsType := strings.TrimSpace(string(out))

	switch fsType {
	case "btrfs":
		return BackendBtrfs, nil
	case "zfs":
		return "", fmt.Errorf("ZFS snapshots are not yet supported (untested) for %s - please use btrfs", path)
	default:
		return "", fmt.Errorf("snapshots require btrfs filesystem - %s is on %s", path, fsType)
	}
}

func GetBackend(backendType BackendType, machineName string) (SnapshotBackend, error) {
	switch backendType {
	case BackendBtrfs:
		return NewBtrfsBackend(machineName)
	case BackendZFS:
		return nil, fmt.Errorf("ZFS snapshots are not yet supported (untested) - please use btrfs")
	case BackendLVM:
		return nil, fmt.Errorf("LVM snapshots are not yet supported (untested) - please use btrfs")
	case BackendDirectory:
		return nil, fmt.Errorf("directory mode does not support snapshots - please use btrfs")
	default:
		return nil, fmt.Errorf("unsupported backend type: %s (supported: btrfs)", backendType)
	}
}

func DetectAndGetBackend(path, machineName string) (SnapshotBackend, error) {
	backendType, err := DetectFilesystem(path)
	if err != nil {
		return nil, err
	}
	return GetBackend(backendType, machineName)
}

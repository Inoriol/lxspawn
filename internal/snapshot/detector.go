package snapshot

import (
	"fmt"
	"syscall"
)

func DetectFilesystem(path string) (BackendType, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", fmt.Errorf("failed to detect filesystem type for %s: %w", path, err)
	}

	fsType := filesystemTypeToString(stat.Type)

	switch fsType {
	case "btrfs":
		return BackendBtrfs, nil
	case "zfs":
		return "", fmt.Errorf("ZFS snapshots are not yet supported (untested) for %s - please use btrfs", path)
	default:
		return "", fmt.Errorf("snapshots require btrfs filesystem - %s is on %s", path, fsType)
	}
}

func filesystemTypeToString(fsType int64) string {
	switch fsType {
	case 0x9123683E:
		return "btrfs"
	case 0x2FC12FC1:
		return "zfs"
	case 0xEF53:
		return "ext4"
	case 0x01021994:
		return "tmpfs"
	case 0x58465342:
		return "xfs"
	default:
		return fmt.Sprintf("unknown(0x%x)", fsType)
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

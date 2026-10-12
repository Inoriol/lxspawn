package snapshot

import (
	"fmt"
	"os/exec"
	"strings"
)

func DetectFilesystem(path string) (BackendType, error) {
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
		lvm, err := detectLVM(path)
		if err == nil && lvm {
			return "", fmt.Errorf("LVM snapshots are not yet supported (untested) for %s - please use btrfs", path)
		}
		return "", fmt.Errorf("unsupported filesystem type %q for %s (supported: btrfs)", fsType, path)
	}
}

func detectLVM(path string) (bool, error) {
	out, err := exec.Command("df", "--output=source", path).Output()
	if err != nil {
		return false, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return false, fmt.Errorf("unexpected df output")
	}

	device := lines[len(lines)-1]

	out, err = exec.Command("pvs", "--noheadings", "-o", "vg_name", device).Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(out)) != "", nil
}

func GetBackend(backendType BackendType, machineName string) (SnapshotBackend, error) {
	switch backendType {
	case BackendBtrfs:
		return NewBtrfsBackend(machineName)
	case BackendZFS:
		return nil, fmt.Errorf("ZFS snapshots are not yet supported (untested) - please use btrfs")
	case BackendLVM:
		return nil, fmt.Errorf("LVM snapshots are not yet supported (untested) - please use btrfs")
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

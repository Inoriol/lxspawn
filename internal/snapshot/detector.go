package snapshot

import (
	"fmt"
	"os/exec"
	"strings"
)

func DetectFilesystem(path string) (BackendType, error) {
	// Check if path is a mount point (indicates dedicated LV per machine)
	if isMountPoint(path) {
		lvm, err := detectLVM(path)
		if err == nil && lvm {
			return BackendLVM, nil
		}
	}

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
		// Not a dedicated mount point, use directory mode (file-based snapshots)
		return BackendDirectory, nil
	}
}

func isMountPoint(path string) bool {
	// Check if path is a mount point by comparing device IDs
	out, err := exec.Command("findmnt", "-n", "-o", "SOURCE", path).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
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

	device := strings.TrimSpace(lines[len(lines)-1])

	lvsPath, err := exec.LookPath("lvs")
	if err != nil {
		lvsPath = "/usr/sbin/lvs"
	}

	out, err = exec.Command("sudo", lvsPath, "--noheadings", "-o", "vg_name", device).CombinedOutput()
	if err != nil {
		return false, nil
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
		return NewLVMBackend(machineName)
	case BackendDirectory:
		return NewDirectoryBackend(machineName)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s (supported: btrfs, lvm, directory)", backendType)
	}
}

func DetectAndGetBackend(path, machineName string) (SnapshotBackend, error) {
	backendType, err := DetectFilesystem(path)
	if err != nil {
		return nil, err
	}
	return GetBackend(backendType, machineName)
}

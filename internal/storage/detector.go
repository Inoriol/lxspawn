package storage

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

// DetectBackend automatically selects the appropriate storage backend
// based on the filesystem type and configuration
func DetectBackend(basePath string) (StorageBackend, error) {
	// Get filesystem type
	fsType, err := getFilesystemType(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect filesystem type: %w", err)
	}

	logger.Info(fmt.Sprintf("Detected filesystem type: %s", fsType))

	// Select backend based on filesystem type
	switch fsType {
	case "btrfs":
		return NewBtrfsBackend(basePath)

	case "ext4", "ext3", "ext2", "xfs":
		// Check if it's LVM
		if isLVM(basePath) {
			// Try to find or use thin pool
			vgName, err := getVGName(basePath)
			if err != nil {
				logger.Warn(fmt.Sprintf("Failed to get VG name: %v", err))
				return NewDirectoryBackend(basePath)
			}

			// Try to detect thin pool
			poolName, err := DetectThinPool(vgName)
			if err != nil {
				// No thin pool found, suggest creating one or use directory mode
				logger.Warn(fmt.Sprintf("No LVM thin pool found in VG %s", vgName))
				logger.Warn("For efficient space usage, consider creating a thin pool:")
				logger.Warn(fmt.Sprintf("  sudo lvcreate -L 50G -n lxspawn-pool %s", vgName))
				logger.Warn(fmt.Sprintf("  sudo lvconvert --type thin-pool %s/lxspawn-pool", vgName))
				logger.Warn("Falling back to directory mode (no block-level snapshots)")

				return NewDirectoryBackend(basePath)
			}

			logger.Info(fmt.Sprintf("Using LVM thin pool: %s/%s", vgName, poolName))
			return NewLVMThinBackend(basePath, vgName, poolName)
		}

		// Not LVM, use directory mode
		return NewDirectoryBackend(basePath)

	default:
		logger.Warn(fmt.Sprintf("Unknown filesystem type %s, using directory mode", fsType))
		return NewDirectoryBackend(basePath)
	}
}

// getFilesystemType returns the filesystem type for a path
func getFilesystemType(path string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", fmt.Errorf("failed to stat filesystem: %w", err)
	}

	return filesystemTypeToString(stat.Type), nil
}

func filesystemTypeToString(fsType int64) string {
	switch fsType {
	case 0x9123683E:
		return "btrfs"
	case 0x2FC12FC1:
		return "zfs"
	case 0xEF53:
		return "ext4"
	case 0xEF51:
		return "ext2"
	case 0xEF52:
		return "ext3"
	case 0x01021994:
		return "tmpfs"
	case 0x58465342:
		return "xfs"
	default:
		return fmt.Sprintf("unknown(0x%x)", fsType)
	}
}

// isLVM checks if a path is on an LVM volume
func isLVM(path string) bool {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return false
	}

	// Get the device name from Statfs
	device := fmt.Sprintf("%v", stat.Fsid)
	if device == "" || device == "0" {
		return false
	}

	// Check if device is an LVM volume
	cmd := deploy.Sudo("lvs", "--noheadings", "-o", "vg_name", device)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// getVGName returns the VG name for an LVM path
func getVGName(path string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", fmt.Errorf("failed to stat filesystem: %w", err)
	}

	device := fmt.Sprintf("%v", stat.Fsid)
	if device == "" || device == "0" {
		return "", fmt.Errorf("could not determine device for %s", path)
	}

	cmd := deploy.Sudo("lvs", "--noheadings", "-o", "vg_name", device)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get VG name: %w", err)
	}

	vgName := strings.TrimSpace(string(output))
	if vgName == "" {
		return "", fmt.Errorf("device %s is not an LVM volume", device)
	}

	return vgName, nil
}

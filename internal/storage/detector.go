package storage

import (
	"fmt"
	"os/exec"
	"strings"

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
	cmd := exec.Command("df", "--output=fstype", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("df failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected df output")
	}

	return strings.TrimSpace(lines[len(lines)-1]), nil
}

// isLVM checks if a path is on an LVM volume
func isLVM(path string) bool {
	cmd := exec.Command("df", "--output=source", path)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return false
	}

	device := strings.TrimSpace(lines[len(lines)-1])

	// Check if device is an LVM volume

	cmd = deploy.SudoHelper("lvs", "--noheadings", "-o", "vg_name", device)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// getVGName returns the VG name for an LVM path
func getVGName(path string) (string, error) {
	cmd := exec.Command("df", "--output=source", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("df failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected df output")
	}

	device := strings.TrimSpace(lines[len(lines)-1])


	cmd = deploy.SudoHelper("lvs", "--noheadings", "-o", "vg_name", device)
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get VG name: %w", err)
	}

	vgName := strings.TrimSpace(string(output))
	if vgName == "" {
		return "", fmt.Errorf("device %s is not an LVM volume", device)
	}

	return vgName, nil
}

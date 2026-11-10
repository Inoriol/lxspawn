package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

// BtrfsBackend implements StorageBackend using btrfs subvolumes
type BtrfsBackend struct {
	basePath string // Base path for machine storage (e.g., /var/lib/machines)
}

// NewBtrfsBackend creates a new btrfs-based storage backend
func NewBtrfsBackend(basePath string) (*BtrfsBackend, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path %s: %w", basePath, err)
	}

	return &BtrfsBackend{basePath: basePath}, nil
}

// Type returns the backend type
func (b *BtrfsBackend) Type() BackendType {
	return BackendBtrfs
}

// CreateMachine creates a btrfs subvolume for the machine
func (b *BtrfsBackend) CreateMachine(name string, sizeGB int) error {
	machinePath := b.GetMachinePath(name)

	// Check if already exists
	if _, err := os.Stat(machinePath); err == nil {
		return fmt.Errorf("machine storage already exists at %s", machinePath)
	}

	// Create parent directory if needed
	parentDir := filepath.Dir(machinePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	// Create btrfs subvolume
	cmd := deploy.Sudo("btrfs", "subvolume", "create", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create btrfs subvolume %s: %w (%s)", machinePath, err, string(output))
	}

	// Set ownership to root:lxspawn
	if err := setOwnership(machinePath, "root", "lxspawn"); err != nil {
		logger.Warn(fmt.Sprintf("failed to set ownership on %s: %v", machinePath, err))
	}

	// Set group-writable permissions
	cmd = deploy.Sudo("chmod", "0775", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("failed to set permissions on %s: %v (%s)", machinePath, err, string(output)))
	}

	logger.Info(fmt.Sprintf("Created btrfs subvolume at %s", machinePath))
	return nil
}

// DeleteMachine removes the btrfs subvolume
func (b *BtrfsBackend) DeleteMachine(name string) error {
	machinePath := b.GetMachinePath(name)

	// Check if exists
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		return nil // Already gone
	}

	// First, delete any child subvolumes (snapshots)
	if err := b.deleteChildSubvolumes(machinePath); err != nil {
		logger.Warn(fmt.Sprintf("failed to delete child subvolumes: %v", err))
	}

	// Delete the subvolume
	cmd := deploy.Sudo("btrfs", "subvolume", "delete", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete btrfs subvolume %s: %w (%s)", machinePath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Removed btrfs subvolume at %s", machinePath))
	return nil
}

// MachineExists checks if the btrfs subvolume exists
func (b *BtrfsBackend) MachineExists(name string) (bool, error) {
	machinePath := b.GetMachinePath(name)

	// Check if path exists
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		return false, nil
	}

	// Check if it's a btrfs subvolume
	cmd := deploy.Sudo("btrfs", "subvolume", "show", machinePath)
	if err := cmd.Run(); err != nil {
		return false, nil // Not a subvolume
	}

	return true, nil
}

// GetMachinePath returns the filesystem path for a machine
func (b *BtrfsBackend) GetMachinePath(name string) string {
	return filepath.Join(b.basePath, name)
}

// GetUsage returns btrfs filesystem usage
func (b *BtrfsBackend) GetUsage() (int64, int64, error) {
	cmd := deploy.Sudo("btrfs", "filesystem", "usage", "-b", b.basePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get btrfs usage: %w", err)
	}

	var total, used int64
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Device size:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &total)
			}
		}
		if strings.HasPrefix(line, "Used:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &used)
			}
		}
	}

	return used, total, nil
}

// deleteChildSubvolumes recursively deletes child subvolumes
func (b *BtrfsBackend) deleteChildSubvolumes(parentPath string) error {
	children, err := findChildSubvolumes(parentPath)
	if err != nil {
		return err
	}

	// Delete children (deepest first - they're already sorted by depth)
	for i := len(children) - 1; i >= 0; i-- {
		childPath := children[i]
		deleteCmd := deploy.Sudo("btrfs", "subvolume", "delete", childPath)
		if deleteOutput, deleteErr := deleteCmd.CombinedOutput(); deleteErr != nil {
			logger.Warn(fmt.Sprintf("failed to delete child subvolume %s: %v (%s)", childPath, deleteErr, string(deleteOutput)))
		}
	}

	return nil
}

func findChildSubvolumes(parentPath string) ([]string, error) {
	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", parentPath, err)
	}

	var subvols []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entryPath := filepath.Join(parentPath, entry.Name())
		if isBtrfsSubvolume(entryPath) {
			children, err := findChildSubvolumes(entryPath)
			if err != nil {
				return nil, err
			}
			subvols = append(subvols, children...)
			subvols = append(subvols, entryPath)
		}
	}

	return subvols, nil
}

func isBtrfsSubvolume(path string) bool {
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return false
	}
	return stat.Ino == 256 && stat.Mode&syscall.S_IFDIR != 0
}

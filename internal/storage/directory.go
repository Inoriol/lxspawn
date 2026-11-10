package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

// DirectoryBackend implements StorageBackend using simple directories
type DirectoryBackend struct {
	basePath string // Base path for machine storage (e.g., /var/lib/machines)
}

// NewDirectoryBackend creates a new directory-based storage backend
func NewDirectoryBackend(basePath string) (*DirectoryBackend, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path %s: %w", basePath, err)
	}

	return &DirectoryBackend{basePath: basePath}, nil
}

// Type returns the backend type
func (d *DirectoryBackend) Type() BackendType {
	return BackendDirectory
}

// CreateMachine creates a directory for the machine
func (d *DirectoryBackend) CreateMachine(name string, sizeGB int) error {
	machinePath := d.GetMachinePath(name)

	// Check if already exists
	if _, err := os.Stat(machinePath); err == nil {
		return fmt.Errorf("machine storage already exists at %s", machinePath)
	}

	// Create directory with proper permissions
	if err := os.MkdirAll(machinePath, 0755); err != nil {
		return fmt.Errorf("failed to create machine directory %s: %w", machinePath, err)
	}

	// Set ownership to root:lxspawn if group exists
	if err := setOwnership(machinePath, "root", "lxspawn"); err != nil {
		logger.Warn(fmt.Sprintf("failed to set ownership on %s: %v", machinePath, err))
	}

	// Set group-writable permissions (use sudo since directory is now owned by root)
	cmd := deploy.Sudo("chmod", "0775", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn(fmt.Sprintf("failed to set permissions on %s: %v (%s)", machinePath, err, string(output)))
	}

	logger.Info(fmt.Sprintf("Created directory storage at %s", machinePath))
	return nil
}

// DeleteMachine removes the machine directory
func (d *DirectoryBackend) DeleteMachine(name string) error {
	machinePath := d.GetMachinePath(name)

	// Check if exists
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		return nil // Already gone
	}

	// Use sudo rm -rf to handle files owned by root (from container namespace)
	cmd := deploy.Sudo("rm", "-rf", machinePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove machine directory %s: %w (%s)", machinePath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Removed directory storage at %s", machinePath))
	return nil
}

// MachineExists checks if the machine directory exists
func (d *DirectoryBackend) MachineExists(name string) (bool, error) {
	machinePath := d.GetMachinePath(name)
	_, err := os.Stat(machinePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetMachinePath returns the filesystem path for a machine
func (d *DirectoryBackend) GetMachinePath(name string) string {
	return filepath.Join(d.basePath, name)
}

// GetUsage returns storage usage for the base path
func (d *DirectoryBackend) GetUsage() (int64, int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(d.basePath, &stat); err != nil {
		return 0, 0, fmt.Errorf("failed to stat filesystem: %w", err)
	}

	// Calculate total and used space in bytes
	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bfree) * int64(stat.Bsize)
	used := total - free

	return used, total, nil
}

// setOwnership sets the ownership of a path
func setOwnership(path, user, group string) error {
	cmd := deploy.Sudo("chown", fmt.Sprintf("%s:%s", user, group), path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %w (%s)", err, string(output))
	}
	return nil
}

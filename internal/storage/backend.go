package storage

import (
	"time"
)

// BackendType represents the type of storage backend
type BackendType string

const (
	BackendBtrfs     BackendType = "btrfs"
	BackendLVMThin   BackendType = "lvm-thin"
	BackendDirectory BackendType = "directory"
)

// SnapshotInfo contains information about a snapshot
type SnapshotInfo struct {
	Name      string    `json:"name"`
	Machine   string    `json:"machine"`
	Backend   string    `json:"backend"`
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size,omitempty"`      // Size in bytes (for directory mode)
	Location  string    `json:"location,omitempty"` // Backend-specific location info
}

// StorageBackend defines the interface for storage operations
type StorageBackend interface {
	// Type returns the backend type
	Type() BackendType

	// CreateMachine creates storage for a new machine
	// name: machine name
	// sizeGB: requested size in GB (may be ignored by some backends)
	CreateMachine(name string, sizeGB int) error

	// DeleteMachine removes all storage for a machine
	DeleteMachine(name string) error

	// MachineExists checks if a machine's storage exists
	MachineExists(name string) (bool, error)

	// GetMachinePath returns the filesystem path for a machine's rootfs
	GetMachinePath(name string) string

	// CreateSnapshot creates a snapshot of a machine
	CreateSnapshot(machine, snapName string) error

	// DeleteSnapshot deletes a snapshot
	DeleteSnapshot(machine, snapName string) error

	// ListSnapshots lists all snapshots for a machine
	ListSnapshots(machine string) ([]SnapshotInfo, error)

	// RollbackSnapshot rolls back a machine to a snapshot
	RollbackSnapshot(machine, snapName string) error

	// GetUsage returns storage usage information
	// Returns: used bytes, total bytes, error
	GetUsage() (int64, int64, error)
}

package storage

// BackendType represents the type of storage backend
type BackendType string

const (
	BackendBtrfs     BackendType = "btrfs"
	BackendLVMThin   BackendType = "lvm-thin"
	BackendDirectory BackendType = "directory"
)

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

	// GetUsage returns storage usage information
	// Returns: used bytes, total bytes, error
	GetUsage() (int64, int64, error)
}

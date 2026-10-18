package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryBackend_CreateMachine(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewDirectoryBackend(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	machineName := "test-machine"

	// Test creation
	if err := backend.CreateMachine(machineName, 10); err != nil {
		t.Fatalf("CreateMachine() error = %v", err)
	}

	// Verify directory exists
	machinePath := backend.GetMachinePath(machineName)
	if _, err := os.Stat(machinePath); os.IsNotExist(err) {
		t.Errorf("Machine directory not created at %s", machinePath)
	}

	// Test duplicate creation
	if err := backend.CreateMachine(machineName, 10); err == nil {
		t.Error("CreateMachine() should fail for duplicate machine")
	}

	// Test exists
	exists, err := backend.MachineExists(machineName)
	if err != nil {
		t.Fatalf("MachineExists() error = %v", err)
	}
	if !exists {
		t.Error("MachineExists() returned false for existing machine")
	}

	// Test deletion
	if err := backend.DeleteMachine(machineName); err != nil {
		t.Fatalf("DeleteMachine() error = %v", err)
	}

	// Verify directory removed
	if _, err := os.Stat(machinePath); !os.IsNotExist(err) {
		t.Error("Machine directory not removed after deletion")
	}

	// Test exists after deletion
	exists, err = backend.MachineExists(machineName)
	if err != nil {
		t.Fatalf("MachineExists() error = %v", err)
	}
	if exists {
		t.Error("MachineExists() returned true for deleted machine")
	}
}

func TestDirectoryBackend_Snapshots(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewDirectoryBackend(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	machineName := "test-machine"
	snapName := "snap1"

	// Create machine
	if err := backend.CreateMachine(machineName, 10); err != nil {
		t.Fatalf("CreateMachine() error = %v", err)
	}

	// Create a test file in the machine
	machinePath := backend.GetMachinePath(machineName)
	testFile := filepath.Join(machinePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test snapshot creation
	if err := backend.CreateSnapshot(machineName, snapName); err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}

	// Verify snapshot exists
	snapshots, err := backend.ListSnapshots(machineName)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("ListSnapshots() returned %d snapshots, want 1", len(snapshots))
	}
	if snapshots[0].Name != snapName {
		t.Errorf("Snapshot name = %s, want %s", snapshots[0].Name, snapName)
	}

	// Test duplicate snapshot
	if err := backend.CreateSnapshot(machineName, snapName); err == nil {
		t.Error("CreateSnapshot() should fail for duplicate snapshot")
	}

	// Modify the machine
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Test rollback
	if err := backend.RollbackSnapshot(machineName, snapName); err != nil {
		t.Fatalf("RollbackSnapshot() error = %v", err)
	}

	// Verify rollback restored original content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Rollback did not restore original content, got %s", string(content))
	}

	// Test snapshot deletion
	if err := backend.DeleteSnapshot(machineName, snapName); err != nil {
		t.Fatalf("DeleteSnapshot() error = %v", err)
	}

	// Verify snapshot removed
	snapshots, err = backend.ListSnapshots(machineName)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("ListSnapshots() returned %d snapshots after deletion, want 0", len(snapshots))
	}
}

func TestDirectoryBackend_Type(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewDirectoryBackend(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	if backend.Type() != BackendDirectory {
		t.Errorf("Type() = %s, want %s", backend.Type(), BackendDirectory)
	}
}

func TestDirectoryBackend_GetUsage(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewDirectoryBackend(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	used, total, err := backend.GetUsage()
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}

	if used < 0 {
		t.Errorf("GetUsage() used = %d, want >= 0", used)
	}
	if total <= 0 {
		t.Errorf("GetUsage() total = %d, want > 0", total)
	}
}

func TestBtrfsBackend_Type(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewBtrfsBackend(tmpDir)
	if err != nil {
		t.Fatalf("NewBtrfsBackend() error = %v", err)
	}

	if backend.Type() != BackendBtrfs {
		t.Errorf("Type() = %s, want %s", backend.Type(), BackendBtrfs)
	}
}

func TestLVMThinBackend_Type(t *testing.T) {
	// This test requires actual LVM setup, so we'll just test the type method
	// with a mock backend
	backend := &LVMThinBackend{
		basePath: "/tmp",
		vgName:   "test-vg",
		poolName: "test-pool",
	}

	if backend.Type() != BackendLVMThin {
		t.Errorf("Type() = %s, want %s", backend.Type(), BackendLVMThin)
	}
}

func TestDetectBackend(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with a regular directory (should use directory backend)
	backend, err := DetectBackend(tmpDir)
	if err != nil {
		t.Fatalf("DetectBackend() error = %v", err)
	}

	// On most test systems, this will be a directory backend
	// unless running on btrfs or LVM
	if backend == nil {
		t.Error("DetectBackend() returned nil")
	}
}

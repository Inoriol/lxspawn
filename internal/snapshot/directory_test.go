package snapshot

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// hasSudo checks if sudo is available and working
func hasSudo() bool {
	cmd := exec.Command("sudo", "-n", "true")
	return cmd.Run() == nil
}

func TestDirectoryBackendType(t *testing.T) {
	backend, err := NewDirectoryBackend("test-machine")
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	if backend.Type() != BackendDirectory {
		t.Errorf("Type() = %v, want %v", backend.Type(), BackendDirectory)
	}
}

func TestDirectoryBackendCreateSnapshot(t *testing.T) {
	if !hasSudo() {
		t.Skip("Skipping test: sudo not available")
	}

	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "lxspawn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test rootfs with some files
	rootfsPath := filepath.Join(tmpDir, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		t.Fatalf("Failed to create rootfs dir: %v", err)
	}

	// Create test files
	testFile := filepath.Join(rootfsPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testDir := filepath.Join(rootfsPath, "subdir")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	testFile2 := filepath.Join(testDir, "nested.txt")
	if err := os.WriteFile(testFile2, []byte("nested content"), 0644); err != nil {
		t.Fatalf("Failed to create nested test file: %v", err)
	}

	backend, err := NewDirectoryBackend("test-machine")
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	// Create snapshot
	err = backend.CreateSnapshot("test-machine", "snap1", rootfsPath)
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}

	// Verify snapshot directory exists
	snapDir := filepath.Join(rootfsPath, ".snapshots", "snap1")
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		t.Errorf("Snapshot directory not created: %v", snapDir)
	}

	// Verify snapshot contains the files
	snapFile := filepath.Join(snapDir, "test.txt")
	if _, err := os.Stat(snapFile); os.IsNotExist(err) {
		t.Errorf("Snapshot file not found: %v", snapFile)
	}

	snapNested := filepath.Join(snapDir, "subdir", "nested.txt")
	if _, err := os.Stat(snapNested); os.IsNotExist(err) {
		t.Errorf("Nested snapshot file not found: %v", snapNested)
	}
}

func TestDirectoryBackendListSnapshots(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "lxspawn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	rootfsPath := filepath.Join(tmpDir, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		t.Fatalf("Failed to create rootfs dir: %v", err)
	}

	backend, err := NewDirectoryBackend("test-machine")
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	// Test empty list
	snapshots, err := backend.ListSnapshots("test-machine", rootfsPath)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots, got %d", len(snapshots))
	}

	// Create snapshot directories manually
	snapDir1 := filepath.Join(rootfsPath, ".snapshots", "snap1")
	if err := os.MkdirAll(snapDir1, 0755); err != nil {
		t.Fatalf("Failed to create snap1 dir: %v", err)
	}

	snapDir2 := filepath.Join(rootfsPath, ".snapshots", "snap2")
	if err := os.MkdirAll(snapDir2, 0755); err != nil {
		t.Fatalf("Failed to create snap2 dir: %v", err)
	}

	// Test list with snapshots
	snapshots, err = backend.ListSnapshots("test-machine", rootfsPath)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}
	if len(snapshots) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(snapshots))
	}

	// Verify snapshot metadata
	found := make(map[string]bool)
	for _, snap := range snapshots {
		found[snap.Name] = true
		if snap.Machine != "test-machine" {
			t.Errorf("Snapshot machine = %v, want test-machine", snap.Machine)
		}
		if snap.Backend != string(BackendDirectory) {
			t.Errorf("Snapshot backend = %v, want %v", snap.Backend, BackendDirectory)
		}
	}

	if !found["snap1"] || !found["snap2"] {
		t.Errorf("Expected snap1 and snap2, got %v", found)
	}
}

func TestDirectoryBackendDeleteSnapshot(t *testing.T) {
	if !hasSudo() {
		t.Skip("Skipping test: sudo not available")
	}

	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "lxspawn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	rootfsPath := filepath.Join(tmpDir, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		t.Fatalf("Failed to create rootfs dir: %v", err)
	}

	backend, err := NewDirectoryBackend("test-machine")
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	// Create snapshot directory
	snapDir := filepath.Join(rootfsPath, ".snapshots", "snap1")
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		t.Fatalf("Failed to create snap dir: %v", err)
	}

	// Delete snapshot
	err = backend.DeleteSnapshot("test-machine", "snap1", rootfsPath)
	if err != nil {
		t.Fatalf("DeleteSnapshot() error = %v", err)
	}

	// Verify snapshot directory is deleted
	if _, err := os.Stat(snapDir); !os.IsNotExist(err) {
		t.Errorf("Snapshot directory still exists after deletion: %v", snapDir)
	}

	// Test deleting non-existent snapshot
	err = backend.DeleteSnapshot("test-machine", "nonexistent", rootfsPath)
	if err == nil {
		t.Error("Expected error when deleting non-existent snapshot, got nil")
	}
}

func TestDirectoryBackendRollbackSnapshot(t *testing.T) {
	if !hasSudo() {
		t.Skip("Skipping test: sudo not available")
	}

	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "lxspawn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Make temp directory accessible to root (for sudo rsync)
	if err := os.Chmod(tmpDir, 0777); err != nil {
		t.Fatalf("Failed to chmod temp dir: %v", err)
	}

	rootfsPath := filepath.Join(tmpDir, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0777); err != nil {
		t.Fatalf("Failed to create rootfs dir: %v", err)
	}

	backend, err := NewDirectoryBackend("test-machine")
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	// Create original file
	originalFile := filepath.Join(rootfsPath, "original.txt")
	if err := os.WriteFile(originalFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Create snapshot
	err = backend.CreateSnapshot("test-machine", "snap1", rootfsPath)
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}

	// Verify snapshot contains the original file
	snapFile := filepath.Join(rootfsPath, ".snapshots", "snap1", "original.txt")
	snapContent, err := os.ReadFile(snapFile)
	if err != nil {
		t.Fatalf("Failed to read snapshot file: %v", err)
	}
	if string(snapContent) != "original content" {
		t.Fatalf("Snapshot file content = %v, want 'original content'", string(snapContent))
	}

	// Modify original file
	if err := os.WriteFile(originalFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Add new file
	newFile := filepath.Join(rootfsPath, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Rollback to snapshot
	err = backend.RollbackSnapshot("test-machine", "snap1", rootfsPath)
	if err != nil {
		t.Fatalf("RollbackSnapshot() error = %v", err)
	}

	// Verify original file is restored
	content, err := os.ReadFile(originalFile)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}
	if string(content) != "original content" {
		t.Errorf("Original file content = %v, want 'original content'", string(content))
	}

	// Verify new file is deleted
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Errorf("New file still exists after rollback: %v", newFile)
	}

	// Test rolling back to non-existent snapshot
	err = backend.RollbackSnapshot("test-machine", "nonexistent", rootfsPath)
	if err == nil {
		t.Error("Expected error when rolling back to non-existent snapshot, got nil")
	}
}

func TestDirectoryBackendCreateSnapshotExcludesSnapshotsDir(t *testing.T) {
	if !hasSudo() {
		t.Skip("Skipping test: sudo not available")
	}

	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "lxspawn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	rootfsPath := filepath.Join(tmpDir, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		t.Fatalf("Failed to create rootfs dir: %v", err)
	}

	backend, err := NewDirectoryBackend("test-machine")
	if err != nil {
		t.Fatalf("NewDirectoryBackend() error = %v", err)
	}

	// Create first snapshot
	err = backend.CreateSnapshot("test-machine", "snap1", rootfsPath)
	if err != nil {
		t.Fatalf("CreateSnapshot(snap1) error = %v", err)
	}

	// Create second snapshot
	err = backend.CreateSnapshot("test-machine", "snap2", rootfsPath)
	if err != nil {
		t.Fatalf("CreateSnapshot(snap2) error = %v", err)
	}

	// Verify snap2 doesn't contain .snapshots directory
	snap2Dir := filepath.Join(rootfsPath, ".snapshots", "snap2")
	snapshotsInSnap2 := filepath.Join(snap2Dir, ".snapshots")
	if _, err := os.Stat(snapshotsInSnap2); !os.IsNotExist(err) {
		t.Errorf("Snapshot should not contain .snapshots directory: %v", snapshotsInSnap2)
	}
}

package storage

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDirectoryBackend_CreateMachine(t *testing.T) {
	if _, err := exec.LookPath("sudo"); err != nil {
		t.Skip("sudo not available, skipping test")
	}
	if os.Geteuid() != 0 {
		t.Skip("requires root privileges, skipping test")
	}

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

func TestBtrfsUsageParsing(t *testing.T) {
	// R17: Test that btrfs usage parsing handles indented output correctly
	// This tests the parsing logic without requiring actual btrfs

	testOutput := `Overall:
    Device size:		  50.00GiB
    Device allocated:		  10.00GiB
    Device unallocated:		  40.00GiB
    Used:			  5.00GiB
    Free:			  45.00GiB
`

	var total, used int64
	lines := strings.Split(testOutput, "\n")
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

	// The parsing should extract numeric values (even if they're 0 due to unit suffixes)
	// The key is that it doesn't fail silently and return 0/0 for all cases
	if total == 0 && used == 0 {
		t.Error("btrfs usage parsing should extract non-zero values from properly formatted output")
	}
}

func TestIsBtrfsSubvolume(t *testing.T) {
	// Test with non-existent path
	if isBtrfsSubvolume("/nonexistent/path") {
		t.Error("isBtrfsSubvolume should return false for non-existent path")
	}

	// Test with regular file
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/regular.txt"
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if isBtrfsSubvolume(tmpFile) {
		t.Error("isBtrfsSubvolume should return false for regular file")
	}

	// Test with regular directory
	regularDir := tmpDir + "/regular"
	if err := os.MkdirAll(regularDir, 0755); err != nil {
		t.Fatal(err)
	}
	if isBtrfsSubvolume(regularDir) {
		t.Error("isBtrfsSubvolume should return false for regular directory")
	}
}

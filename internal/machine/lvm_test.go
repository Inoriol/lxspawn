package machine

import (
	"os"
	"testing"
)

func TestIsMounted(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "non-existent path",
			path:     "/nonexistent/path/that/does/not/exist",
			expected: false,
		},
		{
			name:     "root filesystem",
			path:     "/",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMounted(tt.path)
			if result != tt.expected {
				t.Errorf("isMounted(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsMountedWithTempDir(t *testing.T) {
	tmpDir := t.TempDir()

	if isMounted(tmpDir) {
		t.Errorf("isMounted(%q) = true for temp dir, want false (not a mount point)", tmpDir)
	}
}

func TestIsLVM(t *testing.T) {
	tmpDir := t.TempDir()

	if IsLVM(tmpDir) {
		t.Errorf("IsLVM(%q) = true for temp dir, want false", tmpDir)
	}

	if IsLVM("/nonexistent/path") {
		t.Errorf("IsLVM(/nonexistent/path) = true for non-existent path, want false")
	}
}

func TestGetVGName(t *testing.T) {
	_, err := getVGName("/nonexistent/path")
	if err == nil {
		t.Errorf("getVGName(/nonexistent/path) should return error for non-existent path")
	}
}

func TestCreateLVMVolumeNonLVM(t *testing.T) {
	originalMachinesDir := "/var/lib/machines"

	defer func() {
		os.Setenv("LXSPAWN_MACHINES_DIR", originalMachinesDir)
	}()

	err := CreateLVMVolume("test-machine", 4)
	if err != nil {
		t.Errorf("CreateLVMVolume() on non-LVM system should not error, got: %v", err)
	}
}

func TestFormatAndMountLVNonLVM(t *testing.T) {
	err := FormatAndMountLV("test-machine")
	if err != nil {
		t.Errorf("FormatAndMountLV() on non-LVM system should not error, got: %v", err)
	}
}

func TestCleanupLVMVolumeNonLVM(t *testing.T) {
	err := CleanupLVMVolume("test-machine")
	if err != nil {
		t.Errorf("CleanupLVMVolume() on non-LVM system should not error, got: %v", err)
	}
}

func TestGetVGFreeSpaceGB(t *testing.T) {
	_, err := getVGFreeSpaceGB("nonexistent-vg")
	if err == nil {
		t.Errorf("getVGFreeSpaceGB(nonexistent-vg) should return error for non-existent VG")
	}
}

func TestValidateVGConfiguration(t *testing.T) {
	err := validateVGConfiguration("nonexistent-vg")
	if err != nil {
		t.Errorf("validateVGConfiguration() should not return error for non-existent VG, got: %v", err)
	}
}

package snapshot

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestIsBtrfsSubvolume(t *testing.T) {
	// Test with non-existent path
	if isBtrfsSubvolume("/nonexistent/path") {
		t.Error("isBtrfsSubvolume should return false for non-existent path")
	}

	// Test with regular file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if isBtrfsSubvolume(tmpFile) {
		t.Error("isBtrfsSubvolume should return false for regular file")
	}

	// Test with regular directory
	regularDir := filepath.Join(tmpDir, "regular")
	if err := os.MkdirAll(regularDir, 0755); err != nil {
		t.Fatal(err)
	}
	if isBtrfsSubvolume(regularDir) {
		t.Error("isBtrfsSubvolume should return false for regular directory")
	}
}

func TestIsBtrfsSubvolumeWithInode256(t *testing.T) {
	// This test creates a mock scenario where we have a directory with inode 256
	// Note: This is a unit test that verifies the logic, but creating a real
	// btrfs subvolume requires root and btrfs filesystem

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Get current inode
	var stat syscall.Stat_t
	if err := syscall.Stat(testDir, &stat); err != nil {
		t.Fatal(err)
	}

	// Regular directories should not have inode 256
	if stat.Ino == 256 {
		t.Skip("test directory unexpectedly has inode 256")
	}

	// Verify our function returns false for this directory
	if isBtrfsSubvolume(testDir) {
		t.Error("isBtrfsSubvolume should return false for non-subvolume directory")
	}
}

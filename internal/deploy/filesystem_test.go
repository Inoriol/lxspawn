package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNspawnFilePath(t *testing.T) {
	got := NspawnFilePath("testbox")
	want := "/etc/systemd/nspawn/testbox.nspawn"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSliceFilePath(t *testing.T) {
	got := SliceFilePath("testbox")
	want := "/etc/systemd/system/machine-testbox.slice"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDropInDir(t *testing.T) {
	got := DropInDir("testbox")
	want := "/etc/systemd/system/systemd-nspawn@testbox.service.d"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDropInFilePath(t *testing.T) {
	got := DropInFilePath("testbox")
	want := "/etc/systemd/system/systemd-nspawn@testbox.service.d/lxspawn.conf"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHooksDropInFilePath(t *testing.T) {
	got := HooksDropInFilePath("testbox")
	want := "/etc/systemd/system/systemd-nspawn@testbox.service.d/lxspawn-hooks.conf"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMachineRootfsPath(t *testing.T) {
	got := MachineRootfsPath("testbox")
	want := "/var/lib/machines/testbox"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLxspawnMachineDir(t *testing.T) {
	got := LxspawnMachineDir("testbox")
	want := "/var/lib/lxspawn/testbox"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHooksDir(t *testing.T) {
	got := HooksDir("testbox")
	want := "/var/lib/lxspawn/testbox/.hooks"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := WriteFileAtomic(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}

	if string(content) != "hello" {
		t.Errorf("content = %q, want %q", string(content), "hello")
	}
}

func TestWriteFileAtomicCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "deep", "test.txt")

	if err := WriteFileAtomic(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestWriteFileAtomicPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "executable.sh")

	if err := WriteFileAtomic(path, []byte("#!/bin/bash"), 0755); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions = %o, want 0755", info.Mode().Perm())
	}
}

func TestWriteFileAtomicOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := WriteFileAtomic(path, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(path, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "second" {
		t.Errorf("content = %q, want %q", string(content), "second")
	}
}

func TestWriteFileAtomicNoTempLeftover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := WriteFileAtomic(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if e.Name() != "test.txt" {
			t.Errorf("unexpected file left: %q", e.Name())
		}
	}
}

func TestWriteFileAtomicEmptyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	if err := WriteFileAtomic(path, []byte{}, 0644); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(content) != 0 {
		t.Errorf("content should be empty, got %d bytes", len(content))
	}
}

func TestRemoveAll(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "file1.txt")
	path2 := filepath.Join(dir, "file2.txt")

	os.WriteFile(path1, []byte("a"), 0644)
	os.WriteFile(path2, []byte("b"), 0644)

	if err := RemoveAll(path1, path2); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	if _, err := os.Stat(path1); !os.IsNotExist(err) {
		t.Error("file1 should be removed")
	}
	if _, err := os.Stat(path2); !os.IsNotExist(err) {
		t.Error("file2 should be removed")
	}
}

func TestRemoveAllNonexistent(t *testing.T) {
	if err := RemoveAll("/nonexistent/path"); err != nil {
		t.Errorf("RemoveAll should not fail for nonexistent paths: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if FileExists(path) {
		t.Error("file should not exist yet")
	}

	os.WriteFile(path, []byte("hello"), 0644)

	if !FileExists(path) {
		t.Error("file should exist after write")
	}
}

func TestFileExistsDirectory(t *testing.T) {
	dir := t.TempDir()

	if !FileExists(dir) {
		t.Error("directory should exist")
	}
}

func TestCreatedAt(t *testing.T) {
	ts := CreatedAt()

	if len(ts) == 0 {
		t.Error("CreatedAt should return non-empty string")
	}
}

func TestPathHelpersDifferentNames(t *testing.T) {
	names := []string{"webserver", "db-server", "dev.box"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			nspawn := NspawnFilePath(name)
			slice := SliceFilePath(name)
			dropin := DropInFilePath(name)
			rootfs := MachineRootfsPath(name)

			if !contains(nspawn, name) {
				t.Errorf("nspawn path %q doesn't contain name %q", nspawn, name)
			}
			if !contains(slice, name) {
				t.Errorf("slice path %q doesn't contain name %q", slice, name)
			}
			if !contains(dropin, name) {
				t.Errorf("dropin path %q doesn't contain name %q", dropin, name)
			}
			if !contains(rootfs, name) {
				t.Errorf("rootfs path %q doesn't contain name %q", rootfs, name)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSudo(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		wantArgs []string
	}{
		{
			name:     "simple command",
			args:     []string{"rm", "-rf", "/var/lib/machines/test"},
			wantPath: "sudo",
			wantArgs: []string{"sudo", "rm", "-rf", "/var/lib/machines/test"},
		},
		{
			name:     "btrfs command",
			args:     []string{"btrfs", "subvolume", "create", "/var/lib/machines/test"},
			wantPath: "sudo",
			wantArgs: []string{"sudo", "btrfs", "subvolume", "create", "/var/lib/machines/test"},
		},
		{
			name:     "nsenter command",
			args:     []string{"nsenter", "-t", "1234", "-m", "-u", "-i", "-n", "-p", "--", "/bin/bash", "-c", "echo test"},
			wantPath: "sudo",
			wantArgs: []string{"sudo", "nsenter", "-t", "1234", "-m", "-u", "-i", "-n", "-p", "--", "/bin/bash", "-c", "echo test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Sudo(tt.args...)
			if cmd.Path != tt.wantPath && cmd.Args[0] != tt.wantPath {
				t.Errorf("Sudo() path = %v, want %v", cmd.Path, tt.wantPath)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("Sudo() args length = %v, want %v", len(cmd.Args), len(tt.wantArgs))
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Sudo() arg[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

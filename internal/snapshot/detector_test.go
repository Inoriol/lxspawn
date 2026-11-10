package snapshot

import (
	"testing"
)

func TestGetBackend(t *testing.T) {
	tests := []struct {
		name        string
		backendType BackendType
		machineName string
		wantErr     bool
		wantType    BackendType
	}{
		{"btrfs backend", BackendBtrfs, "test-machine", false, BackendBtrfs},
		{"zfs backend disabled", BackendZFS, "test-machine", true, ""},
		{"lvm backend disabled", BackendLVM, "test-machine", true, ""},
		{"directory backend disabled", BackendDirectory, "test-machine", true, ""},
		{"unsupported backend", BackendType("unsupported"), "test-machine", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := GetBackend(tt.backendType, tt.machineName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && backend.Type() != tt.wantType {
				t.Errorf("GetBackend().Type() = %v, want %v", backend.Type(), tt.wantType)
			}
		})
	}
}

func TestBtrfsBackendType(t *testing.T) {
	b, err := NewBtrfsBackend("test")
	if err != nil {
		t.Fatalf("NewBtrfsBackend() error = %v", err)
	}
	if b.Type() != BackendBtrfs {
		t.Errorf("BtrfsBackend.Type() = %v, want %v", b.Type(), BackendBtrfs)
	}
}

func TestZFSBackendType(t *testing.T) {
	b, err := NewZFSBackend("test")
	if err != nil {
		t.Fatalf("NewZFSBackend() error = %v", err)
	}
	if b.Type() != BackendZFS {
		t.Errorf("ZFSBackend.Type() = %v, want %v", b.Type(), BackendZFS)
	}
}

func TestLVMBackendType(t *testing.T) {
	b, err := NewLVMBackend("test")
	if err != nil {
		t.Fatalf("NewLVMBackend() error = %v", err)
	}
	if b.Type() != BackendLVM {
		t.Errorf("LVMBackend.Type() = %v, want %v", b.Type(), BackendLVM)
	}
}

func TestBtrfsSnapshotDir(t *testing.T) {
	b, _ := NewBtrfsBackend("test")
	rootfsPath := "/var/lib/machines/test"
	expected := "/var/lib/machines/test.snapshots"
	result := b.snapshotDir(rootfsPath)
	if result != expected {
		t.Errorf("snapshotDir() = %v, want %v", result, expected)
	}
}

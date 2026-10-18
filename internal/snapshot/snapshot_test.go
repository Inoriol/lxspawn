package snapshot

import (
	"testing"
	"time"
)

func TestValidateSnapshotName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "snap1", false},
		{"valid with dashes", "my-snapshot", false},
		{"valid with underscores", "my_snapshot", false},
		{"valid with dots", "snap.v1", false},
		{"empty", "", true},
		{"with slash", "snap/1", true},
		{"with backslash", `snap\1`, true},
		{"with space", "snap 1", true},
		{"with tab", "snap\t1", true},
		{"with newline", "snap\n1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSnapshotName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSnapshotName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestFormatCreatedAt(t *testing.T) {
	ti := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := FormatCreatedAt(ti)
	expected := "2024-01-15T10:30:00Z"
	if result != expected {
		t.Errorf("FormatCreatedAt() = %v, want %v", result, expected)
	}
}

func TestNewSnapshotState(t *testing.T) {
	state := NewSnapshotState()
	if state == nil {
		t.Fatal("NewSnapshotState() returned nil")
	}
	if state.Snapshots == nil {
		t.Error("Snapshots map should be initialized")
	}
	if len(state.Snapshots) != 0 {
		t.Error("Snapshots map should be empty")
	}
}

func TestBackendType(t *testing.T) {
	tests := []struct {
		backend BackendType
		want    string
	}{
		{BackendBtrfs, "btrfs"},
		{BackendZFS, "zfs"},
		{BackendLVM, "lvm"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.backend) != tt.want {
				t.Errorf("BackendType = %v, want %v", tt.backend, tt.want)
			}
		})
	}
}

func TestSnapshotInfo(t *testing.T) {
	info := SnapshotInfo{
		Name:      "test-snap",
		Machine:   "test-machine",
		Backend:   "btrfs",
		CreatedAt: "2024-01-15T10:30:00Z",
		Path:      "/var/lib/machines/test-machine/.snapshots/test-snap",
	}

	if info.Name != "test-snap" {
		t.Errorf("Name = %v, want test-snap", info.Name)
	}
	if info.Machine != "test-machine" {
		t.Errorf("Machine = %v, want test-machine", info.Machine)
	}
	if info.Backend != "btrfs" {
		t.Errorf("Backend = %v, want btrfs", info.Backend)
	}
}

func TestLVMBackendIsSnapshotMerging(t *testing.T) {
	backend, err := NewLVMBackend("test-machine")
	if err != nil {
		t.Fatalf("NewLVMBackend() error = %v", err)
	}

	result := backend.isSnapshotMerging("nonexistent-vg", "nonexistent-snap")
	if result {
		t.Errorf("isSnapshotMerging() = true for non-existent snapshot, want false")
	}
}

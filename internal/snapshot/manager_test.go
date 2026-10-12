package snapshot

import (
	"encoding/json"
	"testing"
)

func TestSnapshotStateSerialization(t *testing.T) {
	state := NewSnapshotState()
	state.Snapshots["snap1"] = SnapshotInfo{
		Name:      "snap1",
		Machine:   "test-machine",
		Backend:   "btrfs",
		CreatedAt: "2024-01-15T10:30:00Z",
		Path:      "/var/lib/machines/test-machine/.snapshots/snap1",
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	var loaded SnapshotState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}

	if len(loaded.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(loaded.Snapshots))
	}

	snap, ok := loaded.Snapshots["snap1"]
	if !ok {
		t.Fatal("snap1 not found in loaded state")
	}

	if snap.Name != "snap1" {
		t.Errorf("Name = %v, want snap1", snap.Name)
	}
	if snap.Machine != "test-machine" {
		t.Errorf("Machine = %v, want test-machine", snap.Machine)
	}
	if snap.Backend != "btrfs" {
		t.Errorf("Backend = %v, want btrfs", snap.Backend)
	}
	if snap.Path != "/var/lib/machines/test-machine/.snapshots/snap1" {
		t.Errorf("Path = %v, want /var/lib/machines/test-machine/.snapshots/snap1", snap.Path)
	}
}

func TestSnapshotStateMultipleSnapshots(t *testing.T) {
	state := NewSnapshotState()
	state.Snapshots["snap1"] = SnapshotInfo{Name: "snap1", Machine: "m1", Backend: "btrfs", CreatedAt: "2024-01-15T10:30:00Z"}
	state.Snapshots["snap2"] = SnapshotInfo{Name: "snap2", Machine: "m1", Backend: "btrfs", CreatedAt: "2024-01-15T11:00:00Z"}
	state.Snapshots["snap3"] = SnapshotInfo{Name: "snap3", Machine: "m1", Backend: "btrfs", CreatedAt: "2024-01-15T12:00:00Z"}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	var loaded SnapshotState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}

	if len(loaded.Snapshots) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(loaded.Snapshots))
	}

	for _, name := range []string{"snap1", "snap2", "snap3"} {
		if _, ok := loaded.Snapshots[name]; !ok {
			t.Errorf("snapshot %s not found", name)
		}
	}
}

func TestSnapshotStateEmpty(t *testing.T) {
	state := NewSnapshotState()

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal empty state: %v", err)
	}

	var loaded SnapshotState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal empty state: %v", err)
	}

	if len(loaded.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(loaded.Snapshots))
	}
}

func TestSnapshotInfoJSONFields(t *testing.T) {
	info := SnapshotInfo{
		Name:      "test-snap",
		Machine:   "test-machine",
		Backend:   "zfs",
		CreatedAt: "2024-01-15T10:30:00Z",
		Dataset:   "rpool/machines/test-machine",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal SnapshotInfo: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedFields := []string{"name", "machine", "backend", "created_at", "dataset"}
	for _, field := range expectedFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("expected field %s not found in JSON", field)
		}
	}

	if raw["name"] != "test-snap" {
		t.Errorf("name = %v, want test-snap", raw["name"])
	}
	if raw["backend"] != "zfs" {
		t.Errorf("backend = %v, want zfs", raw["backend"])
	}
}

func TestSnapshotInfoOmitsEmptyFields(t *testing.T) {
	info := SnapshotInfo{
		Name:      "test-snap",
		Machine:   "test-machine",
		Backend:   "btrfs",
		CreatedAt: "2024-01-15T10:30:00Z",
		Path:      "/some/path",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal SnapshotInfo: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["dataset"]; ok {
		t.Error("dataset should be omitted when empty")
	}
	if _, ok := raw["lv"]; ok {
		t.Error("lv should be omitted when empty")
	}
}

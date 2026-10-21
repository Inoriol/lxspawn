package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newStateManager(t *testing.T) *StateManager {
	t.Helper()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "machines.json")
	return &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}
}

func TestRegisterMachine(t *testing.T) {
	sm := newStateManager(t)

	info := MachineInfo{
		Name:       "testbox",
		Distro:     "debian:bookworm",
		SourcePath: "/home/user/testbox.yaml",
		Files:      []string{"/etc/systemd/nspawn/testbox.nspawn"},
		Network:    "host",
		CPUs:       2,
		Memory:     "2G",
		CreatedAt:  "2024-01-15T10:30:00Z",
	}

	if err := sm.RegisterMachine(info); err != nil {
		t.Fatalf("RegisterMachine failed: %v", err)
	}

	got, ok := sm.GetMachine("testbox")
	if !ok {
		t.Fatal("machine not found after register")
	}
	if got.Name != "testbox" {
		t.Errorf("name = %q, want %q", got.Name, "testbox")
	}
	if got.Distro != "debian:bookworm" {
		t.Errorf("distro = %q, want %q", got.Distro, "debian:bookworm")
	}
	if got.CPUs != 2 {
		t.Errorf("cpus = %v, want 2", got.CPUs)
	}
}

func TestUnregisterMachine(t *testing.T) {
	sm := newStateManager(t)

	sm.RegisterMachine(MachineInfo{Name: "testbox", Distro: "debian:bookworm"})
	sm.RegisterMachine(MachineInfo{Name: "otherbox", Distro: "ubuntu:noble"})

	if err := sm.UnregisterMachine("testbox"); err != nil {
		t.Fatalf("UnregisterMachine failed: %v", err)
	}

	if _, ok := sm.GetMachine("testbox"); ok {
		t.Error("testbox should be gone after unregister")
	}
	if _, ok := sm.GetMachine("otherbox"); !ok {
		t.Error("otherbox should still exist")
	}
}

func TestUnregisterNonexistent(t *testing.T) {
	sm := newStateManager(t)

	if err := sm.UnregisterMachine("nonexistent"); err != nil {
		t.Fatalf("UnregisterMachine should not fail for missing machine: %v", err)
	}
}

func TestGetMachineNotFound(t *testing.T) {
	sm := newStateManager(t)

	_, ok := sm.GetMachine("nonexistent")
	if ok {
		t.Error("GetMachine should return false for missing machine")
	}
}

func TestListMachinesSorted(t *testing.T) {
	sm := newStateManager(t)

	sm.RegisterMachine(MachineInfo{Name: "zebra"})
	sm.RegisterMachine(MachineInfo{Name: "alpha"})
	sm.RegisterMachine(MachineInfo{Name: "mango"})

	list := sm.ListMachines()
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}
	if list[0].Name != "alpha" {
		t.Errorf("list[0] = %q, want %q", list[0].Name, "alpha")
	}
	if list[1].Name != "mango" {
		t.Errorf("list[1] = %q, want %q", list[1].Name, "mango")
	}
	if list[2].Name != "zebra" {
		t.Errorf("list[2] = %q, want %q", list[2].Name, "zebra")
	}
}

func TestListMachinesEmpty(t *testing.T) {
	sm := newStateManager(t)

	list := sm.ListMachines()
	if len(list) != 0 {
		t.Errorf("len = %d, want 0", len(list))
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "machines.json")

	sm1 := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}

	sm1.RegisterMachine(MachineInfo{
		Name:    "testbox",
		Distro:  "debian:bookworm",
		Network: "host",
		CPUs:    2,
		Memory:  "2G",
	})

	sm2 := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}
	if err := sm2.load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	got, ok := sm2.Machines["testbox"]
	if !ok {
		t.Fatal("machine not found after reload")
	}
	if got.Name != "testbox" {
		t.Errorf("name = %q, want %q", got.Name, "testbox")
	}
	if got.Distro != "debian:bookworm" {
		t.Errorf("distro = %q, want %q", got.Distro, "debian:bookworm")
	}
	if got.CPUs != 2 {
		t.Errorf("cpus = %v, want 2", got.CPUs)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "subdir", "deep", "machines.json")

	sm := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}

	sm.RegisterMachine(MachineInfo{Name: "testbox"})

	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("state file not created: %v", err)
	}
}

func TestSaveIsAtomic(t *testing.T) {
	sm := newStateManager(t)

	sm.RegisterMachine(MachineInfo{Name: "testbox"})

	data, err := os.ReadFile(sm.StateFilePath)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}

	var machines map[string]MachineInfo
	if err := json.Unmarshal(data, &machines); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}

	if _, ok := machines["testbox"]; !ok {
		t.Error("machine not found in state file")
	}
}

func TestRegisterOverwrite(t *testing.T) {
	sm := newStateManager(t)

	sm.RegisterMachine(MachineInfo{Name: "testbox", Distro: "debian:bookworm"})
	sm.RegisterMachine(MachineInfo{Name: "testbox", Distro: "ubuntu:noble"})

	got, ok := sm.GetMachine("testbox")
	if !ok {
		t.Fatal("machine not found")
	}
	if got.Distro != "ubuntu:noble" {
		t.Errorf("distro = %q, want %q (should be overwritten)", got.Distro, "ubuntu:noble")
	}
}

func TestSetMachine(t *testing.T) {
	sm := newStateManager(t)

	sm.SetMachine("testbox", MachineInfo{Name: "testbox", Distro: "debian:bookworm"})

	got, ok := sm.GetMachine("testbox")
	if !ok {
		t.Fatal("machine not found after SetMachine")
	}
	if got.Distro != "debian:bookworm" {
		t.Errorf("distro = %q", got.Distro)
	}
}

func TestGetStateFilePath(t *testing.T) {
	sm := newStateManager(t)

	if got := sm.GetStateFilePath(); got != sm.StateFilePath {
		t.Errorf("GetStateFilePath() = %q, want %q", got, sm.StateFilePath)
	}
}

func TestSaveFilePermissions(t *testing.T) {
	sm := newStateManager(t)

	sm.RegisterMachine(MachineInfo{Name: "testbox"})

	info, err := os.Stat(sm.StateFilePath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestMachineInfoJSONFields(t *testing.T) {
	info := MachineInfo{
		Name:       "testbox",
		Distro:     "debian:bookworm",
		SourcePath: "/path/to/yaml",
		Files:      []string{"/etc/systemd/nspawn/testbox.nspawn"},
		Network:    "host",
		CPUs:       2,
		Memory:     "2G",
		Swap:       "512M",
		Tasks:      "512",
		CreatedAt:  "2024-01-15T10:30:00Z",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	expectedKeys := []string{"name", "distro", "source_path", "files", "network", "cpus", "memory", "swap", "tasks", "created_at"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		}
	}
}

func TestMachineInfoOmitEmpty(t *testing.T) {
	info := MachineInfo{
		Name:   "testbox",
		Distro: "debian:bookworm",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := raw["cpus"]; ok {
		t.Error("cpus should be omitted when zero")
	}
	if _, ok := raw["memory"]; ok {
		t.Error("memory should be omitted when empty")
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "nonexistent.json")

	sm := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}

	err := sm.load()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist error, got: %v", err)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "machines.json")

	if err := os.WriteFile(statePath, []byte("{invalid json"), 0600); err != nil {
		t.Fatal(err)
	}

	sm := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}

	err := sm.load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMultipleMachinesPersistence(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "machines.json")

	sm1 := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}

	sm1.RegisterMachine(MachineInfo{Name: "box1", Distro: "debian:bookworm", CPUs: 1})
	sm1.RegisterMachine(MachineInfo{Name: "box2", Distro: "ubuntu:noble", CPUs: 2})
	sm1.RegisterMachine(MachineInfo{Name: "box3", Distro: "fedora:44", CPUs: 4})

	sm2 := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}
	if err := sm2.load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(sm2.Machines) != 3 {
		t.Errorf("machines count = %d, want 3", len(sm2.Machines))
	}

	for _, name := range []string{"box1", "box2", "box3"} {
		if _, ok := sm2.Machines[name]; !ok {
			t.Errorf("machine %q not found after reload", name)
		}
	}
}

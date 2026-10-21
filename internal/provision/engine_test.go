package provision

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetState(t *testing.T) {
	tmpDir := t.TempDir()
	machineName := "test-machine"
	
	lxspawnDir := filepath.Join(tmpDir, "lxspawn", machineName)
	if err := os.MkdirAll(lxspawnDir, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	engine := &Engine{
		machineName: machineName,
		rootfsPath:  filepath.Join(tmpDir, "machines", machineName),
		statePath:   filepath.Join(lxspawnDir, "provision-state.json"),
	}

	state, err := engine.GetState()
	if err == nil {
		t.Error("expected error for non-existent state file")
	}

	if state != nil {
		t.Error("expected nil state for non-existent file")
	}

	testState := &ProvisionState{
		LastRun:        "2024-01-15T10:30:00Z",
		PhasesComplete: []string{"pre-boot", "post-boot"},
		ProbesPassed:   []string{"nginx"},
		ScriptHashes: map[string]string{
			"system:0": "sha256:abc123",
		},
	}

	if err := engine.saveState(testState); err != nil {
		t.Fatalf("saveState failed: %v", err)
	}

	state, err = engine.GetState()
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}

	if state == nil {
		t.Fatal("expected non-nil state")
	}

	if state.LastRun != testState.LastRun {
		t.Errorf("expected LastRun %s, got %s", testState.LastRun, state.LastRun)
	}

	if len(state.PhasesComplete) != len(testState.PhasesComplete) {
		t.Errorf("expected %d phases, got %d", len(testState.PhasesComplete), len(state.PhasesComplete))
	}

	if len(state.ProbesPassed) != len(testState.ProbesPassed) {
		t.Errorf("expected %d probes, got %d", len(testState.ProbesPassed), len(state.ProbesPassed))
	}

	if len(state.ScriptHashes) != len(testState.ScriptHashes) {
		t.Errorf("expected %d script hashes, got %d", len(testState.ScriptHashes), len(state.ScriptHashes))
	}
}

func TestProvisionStateSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	machineName := "test-machine"
	
	lxspawnDir := filepath.Join(tmpDir, "lxspawn", machineName)
	if err := os.MkdirAll(lxspawnDir, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	engine := &Engine{
		machineName: machineName,
		rootfsPath:  filepath.Join(tmpDir, "machines", machineName),
		statePath:   filepath.Join(lxspawnDir, "provision-state.json"),
	}

	state := &ProvisionState{
		LastRun:        "2024-01-15T10:30:00Z",
		PhasesComplete: []string{"pre-boot", "post-boot", "probes"},
		ProbesPassed:   []string{"nginx", "postgresql"},
		ScriptHashes: map[string]string{
			"system:0": "sha256:abc123",
			"user:0":   "sha256:def456",
		},
	}

	if err := engine.saveState(state); err != nil {
		t.Fatalf("saveState failed: %v", err)
	}

	loaded, err := engine.GetState()
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}

	if loaded.LastRun != state.LastRun {
		t.Errorf("LastRun mismatch: expected %s, got %s", state.LastRun, loaded.LastRun)
	}

	if len(loaded.PhasesComplete) != 3 {
		t.Errorf("expected 3 phases, got %d", len(loaded.PhasesComplete))
	}

	if len(loaded.ProbesPassed) != 2 {
		t.Errorf("expected 2 probes, got %d", len(loaded.ProbesPassed))
	}

	if len(loaded.ScriptHashes) != 2 {
		t.Errorf("expected 2 script hashes, got %d", len(loaded.ScriptHashes))
	}

	if loaded.ScriptHashes["system:0"] != "sha256:abc123" {
		t.Errorf("script hash mismatch: expected sha256:abc123, got %s", loaded.ScriptHashes["system:0"])
	}
}

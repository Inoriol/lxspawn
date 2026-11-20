package machine

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/testutil"
)

func setupTestManager(t *testing.T) (*Manager, *deploy.StateManager, *testutil.FakeSystemdClient, *testutil.FakeStorageBackend) {
	t.Helper()

	tmpDir := t.TempDir()

	paths := deploy.PathConfig{
		NspawnDir:      filepath.Join(tmpDir, "nspawn"),
		SystemdDir:     filepath.Join(tmpDir, "systemd"),
		MachinesDir:    filepath.Join(tmpDir, "machines"),
		LxspawnDataDir: filepath.Join(tmpDir, "lxspawn"),
	}
	deploy.SetPaths(paths)
	t.Cleanup(deploy.ResetPaths)

	state := &deploy.StateManager{
		StateFilePath: filepath.Join(tmpDir, "machines.json"),
		Machines:      make(map[string]deploy.MachineInfo),
	}

	systemd := testutil.NewFakeSystemdClient()
	storage := testutil.NewFakeStorageBackend(paths.MachinesDir)

	mgr := NewManagerWithDeps(state, systemd, storage)

	return mgr, state, systemd, storage
}

func TestStart_MachineNotFound(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)

	err := mgr.Start("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent machine")
	}
}

func TestStart_Success(t *testing.T) {
	mgr, state, systemd, _ := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{
		Name:   "testbox",
		Distro: "debian:bookworm",
	}

	err := mgr.Start("testbox")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	reloadCalls := systemd.GetCalls("ReloadDaemon")
	if len(reloadCalls) != 1 {
		t.Errorf("expected 1 ReloadDaemon call, got %d", len(reloadCalls))
	}

	startCalls := systemd.GetCalls("StartUnit")
	if len(startCalls) != 1 {
		t.Errorf("expected 1 StartUnit call, got %d", len(startCalls))
	}
	if len(startCalls) > 0 {
		unitName := startCalls[0].Args[0].(string)
		expected := "systemd-nspawn@testbox.service"
		if unitName != expected {
			t.Errorf("expected StartUnit(%q), got StartUnit(%q)", expected, unitName)
		}
	}
}

func TestStop_Success(t *testing.T) {
	mgr, _, systemd, _ := setupTestManager(t)

	err := mgr.Stop("testbox")
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	stopCalls := systemd.GetCalls("StopUnit")
	if len(stopCalls) != 1 {
		t.Errorf("expected 1 StopUnit call, got %d", len(stopCalls))
	}
}

func TestRestart_MachineNotFound(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)

	err := mgr.Restart("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent machine")
	}
}

func TestRestart_Success(t *testing.T) {
	mgr, state, systemd, _ := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{
		Name:   "testbox",
		Distro: "debian:bookworm",
	}

	err := mgr.Restart("testbox")
	if err != nil {
		t.Fatalf("Restart failed: %v", err)
	}

	stopCalls := systemd.GetCalls("StopUnit")
	if len(stopCalls) < 1 {
		t.Error("expected at least 1 StopUnit call")
	}

	startCalls := systemd.GetCalls("StartUnit")
	if len(startCalls) < 1 {
		t.Error("expected at least 1 StartUnit call")
	}
}

func TestDelete_MachineNotFound(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)

	err := mgr.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent machine")
	}
}

func TestDelete_Success(t *testing.T) {
	mgr, state, systemd, storage := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{
		Name:   "testbox",
		Distro: "debian:bookworm",
	}
	storage.Machines["testbox"] = true

	systemd.MachineState["testbox"] = "stopped"

	err := mgr.Delete("testbox")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, exists := state.Machines["testbox"]; exists {
		t.Error("expected machine to be unregistered from state")
	}

	deleteCalls := storage.GetCalls("DeleteMachine")
	if len(deleteCalls) != 1 {
		t.Errorf("expected 1 DeleteMachine call, got %d", len(deleteCalls))
	}

	reloadCalls := systemd.GetCalls("ReloadDaemon")
	if len(reloadCalls) != 1 {
		t.Errorf("expected 1 ReloadDaemon call, got %d", len(reloadCalls))
	}
}

func TestGetStatus_MachineNotFound(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)

	_, err := mgr.GetStatus("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent machine")
	}
}

func TestGetStatus_Success(t *testing.T) {
	mgr, state, systemd, _ := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{
		Name:    "testbox",
		Distro:  "debian:bookworm",
		Network: "host",
	}

	systemd.MachineState["testbox"] = "running"

	status, err := mgr.GetStatus("testbox")
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.Name != "testbox" {
		t.Errorf("expected name 'testbox', got %q", status.Name)
	}
	if status.State != "running" {
		t.Errorf("expected state 'running', got %q", status.State)
	}
	if status.Distro != "debian:bookworm" {
		t.Errorf("expected distro 'debian:bookworm', got %q", status.Distro)
	}
}

func TestListMachines_Empty(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)

	machines, err := mgr.ListMachines()
	if err != nil {
		t.Fatalf("ListMachines failed: %v", err)
	}

	if len(machines) != 0 {
		t.Errorf("expected 0 machines, got %d", len(machines))
	}
}

func TestListMachines_WithMachines(t *testing.T) {
	mgr, state, _, _ := setupTestManager(t)

	state.Machines["box1"] = deploy.MachineInfo{Name: "box1", Distro: "debian:bookworm"}
	state.Machines["box2"] = deploy.MachineInfo{Name: "box2", Distro: "ubuntu:noble"}

	machines, err := mgr.ListMachines()
	if err != nil {
		t.Fatalf("ListMachines failed: %v", err)
	}

	if len(machines) != 2 {
		t.Errorf("expected 2 machines, got %d", len(machines))
	}
}

func TestListMachinesWithState_Stopped(t *testing.T) {
	mgr, state, _, _ := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{Name: "testbox", Distro: "debian:bookworm"}

	machines, err := mgr.ListMachinesWithState()
	if err != nil {
		t.Fatalf("ListMachinesWithState failed: %v", err)
	}

	if len(machines) != 1 {
		t.Fatalf("expected 1 machine, got %d", len(machines))
	}

	if machines[0].State != "stopped" {
		t.Errorf("expected state 'stopped', got %q", machines[0].State)
	}
}

func TestListMachinesWithState_Running(t *testing.T) {
	mgr, state, systemd, _ := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{Name: "testbox", Distro: "debian:bookworm"}

	systemd.ListRunningMachinesFn = func() ([]deploy.RunningMachineInfo, error) {
		return []deploy.RunningMachineInfo{
			{Name: "testbox", Addresses: []string{"192.168.1.100"}},
		}, nil
	}

	machines, err := mgr.ListMachinesWithState()
	if err != nil {
		t.Fatalf("ListMachinesWithState failed: %v", err)
	}

	if len(machines) != 1 {
		t.Fatalf("expected 1 machine, got %d", len(machines))
	}

	if machines[0].State != "running" {
		t.Errorf("expected state 'running', got %q", machines[0].State)
	}

	if len(machines[0].Addresses) != 1 || machines[0].Addresses[0] != "192.168.1.100" {
		t.Errorf("expected address '192.168.1.100', got %v", machines[0].Addresses)
	}
}

func TestGetMachineInfo_Found(t *testing.T) {
	mgr, state, _, _ := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{
		Name:   "testbox",
		Distro: "debian:bookworm",
	}

	info, found := mgr.GetMachineInfo("testbox")
	if !found {
		t.Error("expected machine to be found")
	}
	if info.Name != "testbox" {
		t.Errorf("expected name 'testbox', got %q", info.Name)
	}
}

func TestGetMachineInfo_NotFound(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)

	_, found := mgr.GetMachineInfo("nonexistent")
	if found {
		t.Error("expected machine to not be found")
	}
}

func TestNewManagerWithDeps(t *testing.T) {
	state := &deploy.StateManager{
		StateFilePath: "/tmp/test.json",
		Machines:      make(map[string]deploy.MachineInfo),
	}
	systemd := testutil.NewFakeSystemdClient()
	storage := testutil.NewFakeStorageBackend("/tmp/machines")

	mgr := NewManagerWithDeps(state, systemd, storage)

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.state != state {
		t.Error("expected state to be set")
	}
	if mgr.systemd != systemd {
		t.Error("expected systemd to be set")
	}
	if mgr.storage != storage {
		t.Error("expected storage to be set")
	}
}

func TestClose_NilSystemd(t *testing.T) {
	mgr := &Manager{}
	mgr.Close()
}

func TestClose_WithSystemd(t *testing.T) {
	systemd := testutil.NewFakeSystemdClient()
	mgr := NewManagerWithDeps(nil, systemd, nil)

	mgr.Close()

	closeCalls := systemd.GetCalls("Close")
	if len(closeCalls) != 1 {
		t.Errorf("expected 1 Close call, got %d", len(closeCalls))
	}
}

func TestStart_SystemdError(t *testing.T) {
	mgr, state, systemd, _ := setupTestManager(t)

	state.Machines["testbox"] = deploy.MachineInfo{Name: "testbox"}

	systemd.StartUnitFn = func(name string) error {
		return errors.New("systemd start failed")
	}

	err := mgr.Start("testbox")
	if err == nil {
		t.Error("expected error when StartUnit fails")
	}
}

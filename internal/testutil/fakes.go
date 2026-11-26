package testutil

import (
	"sync"

	systemdbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/storage"
)

type FakeSystemdClient struct {
	mu sync.Mutex

	CloseFn              func() error
	ReloadDaemonFn       func() error
	StartUnitFn          func(string) error
	StopUnitFn           func(string) error
	RestartUnitFn        func(string) error
	ResetFailedUnitFn    func(string) error
	ListUnitsByNamesFn   func([]string) ([]systemdbus.UnitStatus, error)
	ListAllUnitsFn       func() ([]systemdbus.UnitStatus, error)
	GetUnitPropertiesFn  func(string) (map[string]interface{}, error)
	ListRunningMachinesFn func() ([]deploy.RunningMachineInfo, error)
	GetMachineStateFn    func(string) (string, []string, error)
	GetMachineLeaderFn   func(string) (int, error)

	Calls        []FakeCall
	MachineState map[string]string
}

type FakeCall struct {
	Method string
	Args   []interface{}
}

func NewFakeSystemdClient() *FakeSystemdClient {
	return &FakeSystemdClient{
		MachineState: make(map[string]string),
	}
}

func (f *FakeSystemdClient) recordCall(method string, args ...interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, FakeCall{Method: method, Args: args})
}

func (f *FakeSystemdClient) Close() error {
	f.recordCall("Close")
	if f.CloseFn != nil {
		return f.CloseFn()
	}
	return nil
}

func (f *FakeSystemdClient) ReloadDaemon() error {
	f.recordCall("ReloadDaemon")
	if f.ReloadDaemonFn != nil {
		return f.ReloadDaemonFn()
	}
	return nil
}

func (f *FakeSystemdClient) StartUnit(unitName string) error {
	f.recordCall("StartUnit", unitName)
	if f.StartUnitFn != nil {
		return f.StartUnitFn(unitName)
	}
	return nil
}

func (f *FakeSystemdClient) StopUnit(unitName string) error {
	f.recordCall("StopUnit", unitName)
	if f.StopUnitFn != nil {
		return f.StopUnitFn(unitName)
	}
	return nil
}

func (f *FakeSystemdClient) RestartUnit(unitName string) error {
	f.recordCall("RestartUnit", unitName)
	if f.RestartUnitFn != nil {
		return f.RestartUnitFn(unitName)
	}
	return nil
}

func (f *FakeSystemdClient) ResetFailedUnit(unitName string) error {
	f.recordCall("ResetFailedUnit", unitName)
	if f.ResetFailedUnitFn != nil {
		return f.ResetFailedUnitFn(unitName)
	}
	return nil
}

func (f *FakeSystemdClient) ListUnitsByNames(unitNames []string) ([]systemdbus.UnitStatus, error) {
	f.recordCall("ListUnitsByNames", unitNames)
	if f.ListUnitsByNamesFn != nil {
		return f.ListUnitsByNamesFn(unitNames)
	}
	return nil, nil
}

func (f *FakeSystemdClient) ListAllUnits() ([]systemdbus.UnitStatus, error) {
	f.recordCall("ListAllUnits")
	if f.ListAllUnitsFn != nil {
		return f.ListAllUnitsFn()
	}
	return nil, nil
}

func (f *FakeSystemdClient) GetUnitProperties(unitName string) (map[string]interface{}, error) {
	f.recordCall("GetUnitProperties", unitName)
	if f.GetUnitPropertiesFn != nil {
		return f.GetUnitPropertiesFn(unitName)
	}
	return nil, nil
}

func (f *FakeSystemdClient) ListRunningMachines() ([]deploy.RunningMachineInfo, error) {
	f.recordCall("ListRunningMachines")
	if f.ListRunningMachinesFn != nil {
		return f.ListRunningMachinesFn()
	}
	return nil, nil
}

func (f *FakeSystemdClient) GetMachineState(name string) (string, []string, error) {
	f.recordCall("GetMachineState", name)
	if f.GetMachineStateFn != nil {
		return f.GetMachineStateFn(name)
	}
	state, ok := f.MachineState[name]
	if !ok {
		state = "stopped"
	}
	return state, nil, nil
}

func (f *FakeSystemdClient) GetMachineLeader(name string) (int, error) {
	f.recordCall("GetMachineLeader", name)
	if f.GetMachineLeaderFn != nil {
		return f.GetMachineLeaderFn(name)
	}
	return 12345, nil
}

func (f *FakeSystemdClient) GetCalls(method string) []FakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var calls []FakeCall
	for _, c := range f.Calls {
		if c.Method == method {
			calls = append(calls, c)
		}
	}
	return calls
}

func (f *FakeSystemdClient) ResetCalls() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = nil
}

var _ deploy.SystemdClient = (*FakeSystemdClient)(nil)

type FakeStorageBackend struct {
	mu sync.Mutex

	TypeFn          func() storage.BackendType
	CreateMachineFn func(string, int) error
	DeleteMachineFn func(string) error
	MachineExistsFn func(string) (bool, error)
	GetMachinePathFn func(string) string
	GetUsageFn      func() (int64, int64, error)

	Calls      []FakeCall
	BasePath   string
	Machines   map[string]bool
	BackendTyp storage.BackendType
}

func NewFakeStorageBackend(basePath string) *FakeStorageBackend {
	return &FakeStorageBackend{
		BasePath:   basePath,
		Machines:   make(map[string]bool),
		BackendTyp: storage.BackendDirectory,
	}
}

func (f *FakeStorageBackend) recordCall(method string, args ...interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, FakeCall{Method: method, Args: args})
}

func (f *FakeStorageBackend) Type() storage.BackendType {
	if f.TypeFn != nil {
		return f.TypeFn()
	}
	return f.BackendTyp
}

func (f *FakeStorageBackend) CreateMachine(name string, sizeGB int) error {
	f.recordCall("CreateMachine", name, sizeGB)
	if f.CreateMachineFn != nil {
		return f.CreateMachineFn(name, sizeGB)
	}
	f.mu.Lock()
	f.Machines[name] = true
	f.mu.Unlock()
	return nil
}

func (f *FakeStorageBackend) DeleteMachine(name string) error {
	f.recordCall("DeleteMachine", name)
	if f.DeleteMachineFn != nil {
		return f.DeleteMachineFn(name)
	}
	f.mu.Lock()
	delete(f.Machines, name)
	f.mu.Unlock()
	return nil
}

func (f *FakeStorageBackend) MachineExists(name string) (bool, error) {
	f.recordCall("MachineExists", name)
	if f.MachineExistsFn != nil {
		return f.MachineExistsFn(name)
	}
	f.mu.Lock()
	exists := f.Machines[name]
	f.mu.Unlock()
	return exists, nil
}

func (f *FakeStorageBackend) GetMachinePath(name string) string {
	f.recordCall("GetMachinePath", name)
	if f.GetMachinePathFn != nil {
		return f.GetMachinePathFn(name)
	}
	return f.BasePath + "/" + name
}

func (f *FakeStorageBackend) GetUsage() (int64, int64, error) {
	f.recordCall("GetUsage")
	if f.GetUsageFn != nil {
		return f.GetUsageFn()
	}
	return 0, 1073741824, nil
}

func (f *FakeStorageBackend) GetCalls(method string) []FakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var calls []FakeCall
	for _, c := range f.Calls {
		if c.Method == method {
			calls = append(calls, c)
		}
	}
	return calls
}

var _ storage.StorageBackend = (*FakeStorageBackend)(nil)

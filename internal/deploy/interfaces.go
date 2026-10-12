package deploy

import (
	systemdbus "github.com/coreos/go-systemd/v22/dbus"
)

type SystemdClient interface {
	Close() error
	ReloadDaemon() error
	StartUnit(unitName string) error
	StopUnit(unitName string) error
	RestartUnit(unitName string) error
	ListUnitsByNames(unitNames []string) ([]systemdbus.UnitStatus, error)
	ListAllUnits() ([]systemdbus.UnitStatus, error)
	GetUnitProperties(unitName string) (map[string]interface{}, error)
	ListRunningMachines() ([]RunningMachineInfo, error)
	GetMachineState(name string) (string, []string, error)
}

type StateStore interface {
	GetMachine(name string) (MachineInfo, bool)
	GetStateFilePath() string
	ListMachines() []MachineInfo
	RegisterMachine(machine MachineInfo) error
	UnregisterMachine(name string) error
	Save() error
}

var _ SystemdClient = (*SystemdManager)(nil)
var _ StateStore = (*StateManager)(nil)

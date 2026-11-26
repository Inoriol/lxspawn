package deploy

import (
	"path/filepath"
	"sync"
)

const (
	DefaultNspawnDir      = "/etc/systemd/nspawn"
	DefaultSystemdDir     = "/etc/systemd/system"
	DefaultMachinesDir    = "/var/lib/machines"
	DefaultLxspawnDataDir = "/var/lib/lxspawn"
)

type PathConfig struct {
	NspawnDir      string
	SystemdDir     string
	MachinesDir    string
	LxspawnDataDir string
}

func DefaultPathConfig() PathConfig {
	return PathConfig{
		NspawnDir:      DefaultNspawnDir,
		SystemdDir:     DefaultSystemdDir,
		MachinesDir:    DefaultMachinesDir,
		LxspawnDataDir: DefaultLxspawnDataDir,
	}
}

var (
	globalPaths   PathConfig
	globalPathsMu sync.RWMutex
	pathsSet      bool
)

func init() {
	globalPaths = DefaultPathConfig()
}

func SetPaths(cfg PathConfig) {
	globalPathsMu.Lock()
	defer globalPathsMu.Unlock()
	globalPaths = cfg
	pathsSet = true
}

func GetPaths() PathConfig {
	globalPathsMu.RLock()
	defer globalPathsMu.RUnlock()
	return globalPaths
}

func ResetPaths() {
	globalPathsMu.Lock()
	defer globalPathsMu.Unlock()
	globalPaths = DefaultPathConfig()
	pathsSet = false
}

func NspawnFilePath(name string) string {
	paths := GetPaths()
	return filepath.Join(paths.NspawnDir, name+".nspawn")
}

func SliceFilePath(name string) string {
	paths := GetPaths()
	return filepath.Join(paths.SystemdDir, "machine-"+name+".slice")
}

func DropInDir(name string) string {
	paths := GetPaths()
	return filepath.Join(paths.SystemdDir, "systemd-nspawn@"+name+".service.d")
}

func DropInFilePath(name string) string {
	return filepath.Join(DropInDir(name), "lxspawn.conf")
}

func HooksDropInFilePath(name string) string {
	return filepath.Join(DropInDir(name), "lxspawn-hooks.conf")
}

func MachineRootfsPath(name string) string {
	paths := GetPaths()
	return filepath.Join(paths.MachinesDir, name)
}

func LxspawnMachineDir(name string) string {
	paths := GetPaths()
	return filepath.Join(paths.LxspawnDataDir, name)
}

func HooksDir(name string) string {
	return filepath.Join(LxspawnMachineDir(name), ".hooks")
}

func CanonicalConfigPath(name string) string {
	return filepath.Join(LxspawnMachineDir(name), "config.yaml")
}

func MachinesDirPath() string {
	paths := GetPaths()
	return paths.MachinesDir
}

func LxspawnDataDirPath() string {
	paths := GetPaths()
	return paths.LxspawnDataDir
}

func SnapshotsDir(name string) string {
	paths := GetPaths()
	return filepath.Join(paths.MachinesDir, name+".snapshots")
}

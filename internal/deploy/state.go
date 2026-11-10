package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
)

type MachineInfo struct {
	Name       string   `json:"name"`
	Distro     string   `json:"distro"`
	SourcePath string   `json:"source_path"`
	Files      []string `json:"files"`
	Network    string   `json:"network"`
	CPUs       float64  `json:"cpus,omitempty"`
	Memory     string   `json:"memory,omitempty"`
	Swap       string   `json:"swap,omitempty"`
	Tasks      string   `json:"tasks,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

type StateManager struct {
	mu            sync.Mutex
	StateFilePath string
	Machines      map[string]MachineInfo
	lockFile      *os.File
}

func NewStateManager() (*StateManager, error) {
	statePath := filepath.Join(LxspawnDataDir, "machines.json")

	sm := &StateManager{
		StateFilePath: statePath,
		Machines:      make(map[string]MachineInfo),
	}

	if err := sm.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return sm, nil
}

func (sm *StateManager) lock() error {
	dir := filepath.Dir(sm.StateFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	lockPath := sm.StateFilePath + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return err
	}

	sm.lockFile = f

	if err := sm.load(); err != nil && !os.IsNotExist(err) {
		sm.unlock()
		return err
	}

	return nil
}

func (sm *StateManager) unlock() error {
	if sm.lockFile == nil {
		return nil
	}

	fd := int(sm.lockFile.Fd())
	err := syscall.Flock(fd, syscall.LOCK_UN)
	sm.lockFile.Close()
	sm.lockFile = nil
	return err
}

func (sm *StateManager) load() error {
	data, err := os.ReadFile(sm.StateFilePath)
	if err != nil {
		return err
	}
	var freshMap map[string]MachineInfo
	if err := json.Unmarshal(data, &freshMap); err != nil {
		return err
	}
	sm.Machines = freshMap
	return nil
}

func (sm *StateManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.save()
}

func (sm *StateManager) save() error {
	dir := filepath.Dir(sm.StateFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm.Machines, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".machines-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, sm.StateFilePath); err != nil {
		os.Remove(tmpName)
		return err
	}

	return nil
}

func (sm *StateManager) RegisterMachine(machine MachineInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.lock(); err != nil {
		return err
	}
	defer sm.unlock()

	sm.Machines[machine.Name] = machine
	return sm.save()
}

func (sm *StateManager) UnregisterMachine(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.lock(); err != nil {
		return err
	}
	defer sm.unlock()

	delete(sm.Machines, name)
	return sm.save()
}

func (sm *StateManager) GetMachine(name string) (MachineInfo, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	m, ok := sm.Machines[name]
	return m, ok
}

func (sm *StateManager) GetStateFilePath() string {
	return sm.StateFilePath
}

func (sm *StateManager) ListMachines() []MachineInfo {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	machines := make([]MachineInfo, 0, len(sm.Machines))
	for _, m := range sm.Machines {
		machines = append(machines, m)
	}
	sort.Slice(machines, func(i, j int) bool {
		return machines[i].Name < machines[j].Name
	})
	return machines
}

func (sm *StateManager) SetMachine(name string, info MachineInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.lock(); err != nil {
		return err
	}
	defer sm.unlock()

	sm.Machines[name] = info
	return sm.save()
}

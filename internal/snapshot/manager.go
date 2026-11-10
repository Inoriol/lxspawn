package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

type Manager struct {
	mu            sync.Mutex
	stateFilePath string
	state         *SnapshotState
	machineName   string
	rootfsPath    string
	backend       SnapshotBackend
}

func NewManager(machineName string) (*Manager, error) {
	if err := config.ValidateMachineName(machineName); err != nil {
		return nil, err
	}

	rootfsPath := deploy.MachineRootfsPath(machineName)
	if !deploy.FileExists(rootfsPath) {
		return nil, fmt.Errorf("machine rootfs not found at %s", rootfsPath)
	}

	backend, err := DetectAndGetBackend(rootfsPath, machineName)
	if err != nil {
		return nil, fmt.Errorf("failed to detect snapshot backend: %w", err)
	}

	statePath := filepath.Join(deploy.LxspawnMachineDir(machineName), "snapshots.json")

	mgr := &Manager{
		stateFilePath: statePath,
		state:         NewSnapshotState(),
		machineName:   machineName,
		rootfsPath:    rootfsPath,
		backend:       backend,
	}

	if err := mgr.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return mgr, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.stateFilePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, m.state)
}

func (m *Manager) save() error {
	dir := filepath.Dir(m.stateFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return deploy.WriteFileAtomic(m.stateFilePath, data, 0600)
}

func (m *Manager) CreateSnapshot(snapName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := ValidateSnapshotName(snapName); err != nil {
		return err
	}

	if _, exists := m.state.Snapshots[snapName]; exists {
		return fmt.Errorf("snapshot '%s' already exists in state", snapName)
	}

	if err := m.backend.CreateSnapshot(m.machineName, snapName, m.rootfsPath); err != nil {
		return err
	}

	snapshots, err := m.backend.ListSnapshots(m.machineName, m.rootfsPath)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to list snapshots after create: %v", err))
	} else {
		for _, s := range snapshots {
			if s.Name == snapName {
				m.state.Snapshots[snapName] = s
				break
			}
		}
	}

	if err := m.save(); err != nil {
		logger.Warn(fmt.Sprintf("failed to save snapshot state: %v", err))
	}

	return nil
}

func (m *Manager) ListSnapshots() ([]SnapshotInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.backend.ListSnapshots(m.machineName, m.rootfsPath)
}

func (m *Manager) RollbackSnapshot(snapName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := ValidateSnapshotName(snapName); err != nil {
		return err
	}

	if err := m.backend.RollbackSnapshot(m.machineName, snapName, m.rootfsPath); err != nil {
		return err
	}

	m.state = NewSnapshotState()
	if err := m.save(); err != nil {
		logger.Warn(fmt.Sprintf("failed to reset snapshot state after rollback: %v", err))
	}

	return nil
}

func (m *Manager) DeleteSnapshot(snapName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := ValidateSnapshotName(snapName); err != nil {
		return err
	}

	if err := m.backend.DeleteSnapshot(m.machineName, snapName, m.rootfsPath); err != nil {
		return err
	}

	delete(m.state.Snapshots, snapName)
	if err := m.save(); err != nil {
		logger.Warn(fmt.Sprintf("failed to save snapshot state after delete: %v", err))
	}

	return nil
}

func (m *Manager) BackendType() BackendType {
	return m.backend.Type()
}

func (m *Manager) Sync() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshots, err := m.backend.ListSnapshots(m.machineName, m.rootfsPath)
	if err != nil {
		return fmt.Errorf("failed to list snapshots for sync: %w", err)
	}

	newState := NewSnapshotState()
	for _, s := range snapshots {
		newState.Snapshots[s.Name] = s
	}

	m.state = newState
	if err := m.save(); err != nil {
		return fmt.Errorf("failed to save synced state: %w", err)
	}

	return nil
}

func (m *Manager) Close() {
	// Currently no resources to release, but provided for API consistency
	// with machine.Manager and to allow future resource management
}

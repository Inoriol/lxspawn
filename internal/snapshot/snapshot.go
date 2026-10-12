package snapshot

import (
	"fmt"
	"time"
)

type BackendType string

const (
	BackendBtrfs BackendType = "btrfs"
	BackendZFS   BackendType = "zfs"
	BackendLVM   BackendType = "lvm"
)

type SnapshotInfo struct {
	Name      string    `json:"name"`
	Machine   string    `json:"machine"`
	Backend   string    `json:"backend"`
	CreatedAt string    `json:"created_at"`
	Path      string    `json:"path,omitempty"`
	Dataset   string    `json:"dataset,omitempty"`
	LV        string    `json:"lv,omitempty"`
}

type SnapshotBackend interface {
	CreateSnapshot(machine, snapName, rootfsPath string) error
	ListSnapshots(machine, rootfsPath string) ([]SnapshotInfo, error)
	RollbackSnapshot(machine, snapName, rootfsPath string) error
	DeleteSnapshot(machine, snapName, rootfsPath string) error
	Type() BackendType
}

type SnapshotState struct {
	Snapshots map[string]SnapshotInfo `json:"snapshots"`
}

func NewSnapshotState() *SnapshotState {
	return &SnapshotState{
		Snapshots: make(map[string]SnapshotInfo),
	}
}

func ValidateSnapshotName(name string) error {
	if name == "" {
		return fmt.Errorf("snapshot name cannot be empty")
	}
	for _, c := range name {
		if c == '/' || c == '\\' || c == ' ' || c == '\t' || c == '\n' {
			return fmt.Errorf("snapshot name cannot contain spaces, slashes, or special characters: %q", name)
		}
	}
	return nil
}

func FormatCreatedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

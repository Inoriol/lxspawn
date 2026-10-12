package snapshot

import (
	"fmt"
	"regexp"
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

var validSnapshotNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}[a-zA-Z0-9]$`)

func ValidateSnapshotName(name string) error {
	if name == "" {
		return fmt.Errorf("snapshot name cannot be empty")
	}
	if len(name) == 1 {
		if !regexp.MustCompile(`^[a-zA-Z0-9]$`).MatchString(name) {
			return fmt.Errorf("snapshot name must be alphanumeric")
		}
		return nil
	}
	if !validSnapshotNameRegex.MatchString(name) {
		return fmt.Errorf("snapshot name must start and end with an alphanumeric character, contain only alphanumeric characters, hyphens, underscores, and dots, and be between 2 and 64 characters")
	}
	return nil
}

func FormatCreatedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

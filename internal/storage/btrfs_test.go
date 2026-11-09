package storage

import (
	"strings"
	"testing"
)

func TestBtrfsBackend_parseBtrfsSubvolumeList(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		wantChildren []string
	}{
		{
			name: "single child subvolume",
			output: `ID 256 gen 10 top level 5 path var/lib/machines/test/.snapshots/snap1`,
			wantChildren: []string{"/var/lib/machines/test/.snapshots/snap1"},
		},
		{
			name: "multiple child subvolumes",
			output: `ID 256 gen 10 top level 5 path var/lib/machines/test/.snapshots/snap1
ID 257 gen 11 top level 5 path var/lib/machines/test/.snapshots/snap2`,
			wantChildren: []string{
				"/var/lib/machines/test/.snapshots/snap1",
				"/var/lib/machines/test/.snapshots/snap2",
			},
		},
		{
			name:         "empty output",
			output:       "",
			wantChildren: []string{},
		},
		{
			name:         "whitespace only",
			output:       "   \n  \n  ",
			wantChildren: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			children := parseBtrfsSubvolumeList(tt.output)
			if len(children) != len(tt.wantChildren) {
				t.Errorf("parseBtrfsSubvolumeList() returned %d children, want %d", len(children), len(tt.wantChildren))
				return
			}
			for i, child := range children {
				if child != tt.wantChildren[i] {
					t.Errorf("parseBtrfsSubvolumeList() child[%d] = %v, want %v", i, child, tt.wantChildren[i])
				}
			}
		})
	}
}

func TestBtrfsBackend_makeAbsolutePath(t *testing.T) {
	backend := &BtrfsBackend{basePath: "/var/lib/machines"}

	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{
			name:     "relative path",
			path:     "var/lib/machines/test/.snapshots/snap1",
			wantPath: "/var/lib/machines/test/.snapshots/snap1",
		},
		{
			name:     "already absolute path",
			path:     "/var/lib/machines/test/.snapshots/snap1",
			wantPath: "/var/lib/machines/test/.snapshots/snap1",
		},
		{
			name:     "path with leading slash",
			path:     "/var/lib/machines/test",
			wantPath: "/var/lib/machines/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath := backend.makeAbsolutePath(tt.path)
			if gotPath != tt.wantPath {
				t.Errorf("makeAbsolutePath() = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

// parseBtrfsSubvolumeList is a helper function to parse btrfs subvolume list output
func parseBtrfsSubvolumeList(output string) []string {
	var children []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		childPath := fields[len(fields)-1]
		if !strings.HasPrefix(childPath, "/") {
			childPath = "/" + childPath
		}
		children = append(children, childPath)
	}
	return children
}

// makeAbsolutePath converts a relative path to absolute
func (b *BtrfsBackend) makeAbsolutePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

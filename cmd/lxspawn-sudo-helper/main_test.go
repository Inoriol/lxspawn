package main

import (
	"testing"
)

func TestIsPathAllowed(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Allowed paths
		{"machines dir", "/var/lib/machines/test", true},
		{"machines dir exact", "/var/lib/machines/", true},
		{"lxspawn dir", "/var/lib/lxspawn/test", true},
		{"nspawn dir", "/etc/systemd/nspawn/test.nspawn", true},
		{"systemd dir", "/etc/systemd/system/machine-test.slice", true},
		{"machines nested", "/var/lib/machines/test/rootfs/etc/passwd", true},

		// Disallowed paths
		{"root", "/root/.ssh/authorized_keys", false},
		{"etc shadow", "/etc/shadow", false},
		{"home dir", "/home/user/.bashrc", false},
		{"tmp dir", "/tmp/malicious", false},
		{"usr bin", "/usr/bin/ls", false},
		{"var log", "/var/log/syslog", false},
		{"empty", "", false},
		{"relative path", "relative/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPathAllowed(tt.path)
			if result != tt.expected {
				t.Errorf("isPathAllowed(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"no traversal", "/var/lib/machines/test", false},
		{"double dot", "/var/lib/machines/../etc/passwd", true},
		{"double dot middle", "/var/lib/..test", true},
		{"double dot end", "/var/lib/machines/test/..", true},
		{"single dot ok", "/var/lib/machines/./test", false},
		{"no dots", "/var/lib/machines/test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsPathTraversal(tt.path)
			if result != tt.expected {
				t.Errorf("containsPathTraversal(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestValidateMachineName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"simple", "test", false},
		{"with hyphen", "test-machine", false},
		{"with underscore", "test_machine", false},
		{"with numbers", "test123", false},
		{"starts with number", "1test", false},
		{"max length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false}, // 64 chars
		{"min length", "ab", false},

		// Invalid names
		{"empty", "", true},
		{"single char", "a", true}, // Regex requires at least 2 chars
		{"starts with hyphen", "-test", true},
		{"ends with hyphen", "test-", true},
		{"starts with underscore", "_test", true},
		{"ends with underscore", "test_", true},
		{"contains dot", "test.machine", true},
		{"contains space", "test machine", true},
		{"contains slash", "test/machine", true},
		{"contains semicolon", "test;machine", true},
		{"contains pipe", "test|machine", true},
		{"contains dollar", "test$machine", true},
		{"contains backtick", "test`machine`", true},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true}, // 65 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMachineName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMachineName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSnapshotName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"simple", "snap1", false},
		{"with hyphen", "snap-1", false},
		{"with underscore", "snap_1", false},
		{"with dot", "snap.v1", false},
		{"with numbers", "snap123", false},
		{"multiple dots", "snap.v1.2", false},

		// Invalid names
		{"empty", "", true},
		{"starts with hyphen", "-snap", true},
		{"ends with hyphen", "snap-", true},
		{"starts with dot", ".snap", true},
		{"ends with dot", "snap.", true},
		{"contains space", "snap 1", true},
		{"contains slash", "snap/1", true},
		{"contains semicolon", "snap;1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSnapshotName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSnapshotName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLVName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"simple", "lxspawn-test", false},
		{"with underscore", "lxspawn-test_machine", false},
		{"with numbers", "lxspawn-test123", false},

		// Invalid names
		{"empty", "", true},
		{"no prefix", "test", true},
		{"wrong prefix", "docker-test", true},
		{"starts with hyphen after prefix", "lxspawn--test", true},
		{"ends with hyphen", "lxspawn-test-", true},
		{"contains dot", "lxspawn-test.v1", true},
		{"contains space", "lxspawn-test machine", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLVName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLVName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestExtractMachineNameFromPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantName    string
		wantErr     bool
	}{
		{"machines path", "/var/lib/machines/test/rootfs", "test", false},
		{"lxspawn path", "/var/lib/lxspawn/test/provision-state.json", "test", false},
		{"nspawn path", "/etc/systemd/nspawn/test.nspawn", "test.nspawn", false}, // Includes extension
		{"no machine name", "/var/lib/machines/", "", true},
		{"not in allowed dirs", "/home/user/test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractMachineNameFromPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractMachineNameFromPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if got != tt.wantName {
				t.Errorf("extractMachineNameFromPath(%q) = %q, want %q", tt.path, got, tt.wantName)
			}
		})
	}
}

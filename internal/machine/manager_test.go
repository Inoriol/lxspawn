package machine

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestIsPopulatedRootfs(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "empty directory",
			setup: func(path string) error {
				return os.MkdirAll(path, 0755)
			},
			expected: false,
		},
		{
			name: "directory with etc but no os-release",
			setup: func(path string) error {
				etcDir := filepath.Join(path, "etc")
				return os.MkdirAll(etcDir, 0755)
			},
			expected: false,
		},
		{
			name: "directory with os-release",
			setup: func(path string) error {
				etcDir := filepath.Join(path, "etc")
				if err := os.MkdirAll(etcDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(etcDir, "os-release"), []byte("ID=debian"), 0644)
			},
			expected: true,
		},
		{
			name: "directory with other files but no os-release",
			setup: func(path string) error {
				etcDir := filepath.Join(path, "etc")
				if err := os.MkdirAll(etcDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(etcDir, "hostname"), []byte("test"), 0644)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testPath := filepath.Join(tmpDir, "rootfs")

			if err := tt.setup(testPath); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			result := isPopulatedRootfs(testPath)
			if result != tt.expected {
				t.Errorf("isPopulatedRootfs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
		{
			name:     "simple alphanumeric",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "with spaces",
			input:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "with single quote",
			input:    "it's",
			expected: "'it'\"'\"'s'",
		},
		{
			name:     "with double quote",
			input:    `say "hello"`,
			expected: `'say "hello"'`,
		},
		{
			name:     "with special chars",
			input:    "test$var",
			expected: "'test$var'",
		},
		{
			name:     "path with slashes",
			input:    "/usr/bin/test",
			expected: "/usr/bin/test",
		},
		{
			name:     "with equals",
			input:    "KEY=value",
			expected: "KEY=value",
		},
		{
			name:     "with semicolon",
			input:    "cmd1; cmd2",
			expected: "'cmd1; cmd2'",
		},
		{
			name:     "with pipe",
			input:    "cmd1 | cmd2",
			expected: "'cmd1 | cmd2'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuote(tt.input)
			if result != tt.expected {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShellQuoteArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "single arg",
			args:     []string{"echo"},
			expected: "echo",
		},
		{
			name:     "multiple args",
			args:     []string{"echo", "hello", "world"},
			expected: "echo hello world",
		},
		{
			name:     "args with spaces",
			args:     []string{"echo", "hello world"},
			expected: "echo 'hello world'",
		},
		{
			name:     "args with special chars",
			args:     []string{"sh", "-c", "echo $HOME"},
			expected: "sh -c 'echo $HOME'",
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuoteArgs(tt.args)
			if result != tt.expected {
				t.Errorf("shellQuoteArgs(%v) = %q, want %q", tt.args, result, tt.expected)
			}
		})
	}
}

func TestFixRootfsOwnership(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "rootfs")
	if err := os.MkdirAll(filepath.Join(testPath, "etc"), 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(testPath, "etc", "test"), []byte("test"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := fixRootfsOwnership(testPath); err != nil {
		t.Fatalf("fixRootfsOwnership failed: %v", err)
	}

	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	if info.Sys().(*syscall.Stat_t).Uid != 0 {
		t.Errorf("expected uid 0, got %d", info.Sys().(*syscall.Stat_t).Uid)
	}
}

package machine

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Inoriol/lxspawn/internal/compiler"
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

func TestReadCPUUsage(t *testing.T) {
	tmpDir := t.TempDir()
	cgroupPath := filepath.Join(tmpDir, "cgroup")
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	cpuStatPath := filepath.Join(cgroupPath, "cpu.stat")
	cpuStatContent := `usage_usec 1234567
user_usec 1000000
system_usec 234567`
	if err := os.WriteFile(cpuStatPath, []byte(cpuStatContent), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	usage, err := readCPUUsage(cgroupPath)
	if err != nil {
		t.Fatalf("readCPUUsage failed: %v", err)
	}

	if usage < 0 {
		t.Errorf("expected non-negative CPU usage, got %.2f", usage)
	}
}

func TestReadCPUUsageMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cgroupPath := filepath.Join(tmpDir, "cgroup")
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err := readCPUUsage(cgroupPath)
	if err == nil {
		t.Error("expected error for missing cpu.stat file")
	}
}

func TestReadMemoryUsage(t *testing.T) {
	tmpDir := t.TempDir()
	cgroupPath := filepath.Join(tmpDir, "cgroup")
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	memCurrentPath := filepath.Join(cgroupPath, "memory.current")
	if err := os.WriteFile(memCurrentPath, []byte("536870912\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	memMaxPath := filepath.Join(cgroupPath, "memory.max")
	if err := os.WriteFile(memMaxPath, []byte("2147483648\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	current, max, err := readMemoryUsage(cgroupPath)
	if err != nil {
		t.Fatalf("readMemoryUsage failed: %v", err)
	}

	if current != 536870912 {
		t.Errorf("expected current 536870912, got %d", current)
	}

	if max != 2147483648 {
		t.Errorf("expected max 2147483648, got %d", max)
	}
}

func TestReadMemoryUsageUnlimited(t *testing.T) {
	tmpDir := t.TempDir()
	cgroupPath := filepath.Join(tmpDir, "cgroup")
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	memCurrentPath := filepath.Join(cgroupPath, "memory.current")
	if err := os.WriteFile(memCurrentPath, []byte("536870912\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	memMaxPath := filepath.Join(cgroupPath, "memory.max")
	if err := os.WriteFile(memMaxPath, []byte("max\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	current, max, err := readMemoryUsage(cgroupPath)
	if err != nil {
		t.Fatalf("readMemoryUsage failed: %v", err)
	}

	if current != 536870912 {
		t.Errorf("expected current 536870912, got %d", current)
	}

	if max != 0 {
		t.Errorf("expected max 0 for unlimited, got %d", max)
	}
}

func TestReadDiskUsage(t *testing.T) {
	tmpDir := t.TempDir()

	used, total, err := readDiskUsage(tmpDir)
	if err != nil {
		t.Fatalf("readDiskUsage failed: %v", err)
	}

	if total <= 0 {
		t.Errorf("expected total > 0, got %d", total)
	}

	if used < 0 {
		t.Errorf("expected used >= 0, got %d", used)
	}

	if used > total {
		t.Errorf("expected used <= total, got used=%d total=%d", used, total)
	}
}

func TestReadDiskUsageInvalidPath(t *testing.T) {
	_, _, err := readDiskUsage("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestCleanupCreatedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	dir1 := filepath.Join(tmpDir, "subdir")

	if err := os.WriteFile(file1, []byte("test1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("test2"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatal(err)
	}

	filesWritten := []string{file1, file2}

	compiled := &compiler.CompiledFiles{
		NspawnPath: filepath.Join(tmpDir, "nspawn"),
		SlicePath:  filepath.Join(tmpDir, "slice"),
		DropInPath: filepath.Join(tmpDir, "dropin"),
		HooksPath:  filepath.Join(tmpDir, "hooks"),
		Slice:      "has-slice",
		HooksDropIn: "has-hooks",
	}

	if err := os.WriteFile(compiled.NspawnPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(compiled.SlicePath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(compiled.DropInPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(compiled.HooksPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cleanupCreatedFiles(filesWritten, compiled)

	for _, f := range filesWritten {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("expected file %s to be removed", f)
		}
	}

	for _, f := range []string{compiled.NspawnPath, compiled.SlicePath, compiled.DropInPath, compiled.HooksPath} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("expected file %s to be removed", f)
		}
	}
}

func TestIsPopulatedRootfsSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "rootfs")

	etcDir := filepath.Join(testPath, "etc")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	osReleaseTarget := filepath.Join(tmpDir, "os-release-target")
	if err := os.WriteFile(osReleaseTarget, []byte("ID=debian"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(osReleaseTarget, filepath.Join(etcDir, "os-release")); err != nil {
		t.Fatal(err)
	}

	result := isPopulatedRootfs(testPath)
	if !result {
		t.Error("expected isPopulatedRootfs to return true for symlinked os-release")
	}
}

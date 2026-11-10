package provision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Inoriol/lxspawn/internal/config"
)

func boolPtr(b bool) *bool { return &b }

func TestDataProvisionerWritesFile(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:    config.ProvisionData,
		Path:    "/etc/myapp/config.yaml",
		Content: "hostname: {{.Name}}\nport: {{.Param.port}}",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	targetPath := filepath.Join(rootfs, "etc", "myapp", "config.yaml")
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}

	expected := "hostname: myserver\nport: 8080"
	if string(content) != expected {
		t.Errorf("content = %q, want %q", string(content), expected)
	}
}

func TestDataProvisionerCreatesParentDirs(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:    config.ProvisionData,
		Path:    "/a/b/c/d/file.txt",
		Content: "deep",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	targetPath := filepath.Join(rootfs, "a", "b", "c", "d", "file.txt")
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestDataProvisionerOverwriteTrue(t *testing.T) {
	rootfs := t.TempDir()
	targetDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(targetDir, "config.txt")
	if err := os.WriteFile(targetPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:      config.ProvisionData,
		Path:      "/etc/config.txt",
		Content:   "new content",
		Overwrite: boolPtr(true),
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "new content" {
		t.Errorf("content = %q, want %q", string(content), "new content")
	}
}

func TestDataProvisionerOverwriteFalse(t *testing.T) {
	rootfs := t.TempDir()
	targetDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(targetDir, "config.txt")
	if err := os.WriteFile(targetPath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:      config.ProvisionData,
		Path:      "/etc/config.txt",
		Content:   "new content",
		Overwrite: boolPtr(false),
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "original" {
		t.Errorf("content = %q, want %q (should not be overwritten)", string(content), "original")
	}
}

func TestDataProvisionerDefaultOverwrite(t *testing.T) {
	rootfs := t.TempDir()
	targetDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(targetDir, "config.txt")
	if err := os.WriteFile(targetPath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:    config.ProvisionData,
		Path:    "/etc/config.txt",
		Content: "new content",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "new content" {
		t.Errorf("content = %q, want %q (default overwrite=true)", string(content), "new content")
	}
}

func TestDataProvisionerCustomPermissions(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:        config.ProvisionData,
		Path:        "/etc/script.sh",
		Content:     "#!/bin/bash\necho hello",
		Permissions: "0755",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	targetPath := filepath.Join(rootfs, "etc", "script.sh")
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions = %o, want 0755", info.Mode().Perm())
	}
}

func TestDataProvisionerDefaultPermissions(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:    config.ProvisionData,
		Path:    "/etc/config.txt",
		Content: "hello",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	targetPath := filepath.Join(rootfs, "etc", "config.txt")
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0644 {
		t.Errorf("permissions = %o, want 0644", info.Mode().Perm())
	}
}

func TestDataProvisionerInvalidPermissions(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:        config.ProvisionData,
		Path:        "/etc/config.txt",
		Content:     "hello",
		Permissions: "invalid",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err == nil {
		t.Fatal("expected error for invalid permissions")
	}
}

func TestDataProvisionerWithOwner(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("skipping test that requires root privileges")
	}
	rootfs := t.TempDir()
	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:    config.ProvisionData,
		Path:    "/etc/config.txt",
		Content: "hello",
		Owner:   "root:root",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	targetPath := filepath.Join(rootfs, "etc", "config.txt")
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestDataProvisionerWithFileRef(t *testing.T) {
	rootfs := t.TempDir()
	baseDir := t.TempDir()

	fileContent := "content from file"
	filePath := filepath.Join(baseDir, "config.txt")
	if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode: config.ProvisionData,
		Path: "/etc/config.txt",
		File: &config.FileRef{URL: "config.txt"},
	}

	if err := provisioner.Provision(prov, ctx, fh, baseDir); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	targetPath := filepath.Join(rootfs, "etc", "config.txt")
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != fileContent {
		t.Errorf("content = %q, want %q", string(content), fileContent)
	}
}

func TestDataProvisionerTemplateRendering(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewDataProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:    config.ProvisionData,
		Path:    "/etc/app.conf",
		Content: "name={{.Name}}\ndistro={{.Distro}}\nuser={{.User}}\nuid={{.UID}}\nhome={{.Home}}",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	targetPath := filepath.Join(rootfs, "etc", "app.conf")
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	expected := "name=myserver\ndistro=debian:bookworm\nuser=testuser\nuid=1000\nhome=/home/testuser"
	if string(content) != expected {
		t.Errorf("content = %q, want %q", string(content), expected)
	}
}

func TestValidatePath(t *testing.T) {
	rootfs := "/var/lib/machines/test"

	tests := []struct {
		name         string
		relativePath string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid absolute path",
			relativePath: "/etc/config.yaml",
			wantErr:      false,
		},
		{
			name:         "valid nested path",
			relativePath: "/a/b/c/d/file.txt",
			wantErr:      false,
		},
		{
			name:         "path traversal with ..",
			relativePath: "/etc/../../../etc/passwd",
			wantErr:      true,
			errContains:  "escapes rootfs",
		},
		{
			name:         "relative path not absolute",
			relativePath: "etc/config.yaml",
			wantErr:      true,
			errContains:  "must be absolute",
		},
		{
			name:         "path with .. in middle",
			relativePath: "/etc/../config.yaml",
			wantErr:      false,
		},
		{
			name:         "root path",
			relativePath: "/",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(rootfs, tt.relativePath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePath() expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("validatePath() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validatePath() unexpected error: %v", err)
				}
				expected := filepath.Join(rootfs, tt.relativePath)
				if result != expected {
					t.Errorf("validatePath() = %v, want %v", result, expected)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSecureJoin(t *testing.T) {
	rootfs := t.TempDir()

	t.Run("simple path without symlinks", func(t *testing.T) {
		result, err := secureJoin(rootfs, "/etc/config.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(rootfs, "etc", "config.yaml")
		if result != expected {
			t.Errorf("secureJoin() = %q, want %q", result, expected)
		}
	})

	t.Run("path with symlink pointing inside rootfs", func(t *testing.T) {
		etcDir := filepath.Join(rootfs, "etc")
		if err := os.MkdirAll(etcDir, 0755); err != nil {
			t.Fatal(err)
		}
		linkTarget := filepath.Join(rootfs, "real-config")
		if err := os.Symlink(linkTarget, filepath.Join(etcDir, "config-link")); err != nil {
			t.Fatal(err)
		}

		result, err := secureJoin(rootfs, "/etc/config-link/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(rootfs, "real-config", "file.txt")
		if result != expected {
			t.Errorf("secureJoin() = %q, want %q", result, expected)
		}
	})

	t.Run("path with absolute symlink escaping rootfs", func(t *testing.T) {
		etcDir := filepath.Join(rootfs, "etc2")
		if err := os.MkdirAll(etcDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("/etc/shadow", filepath.Join(etcDir, "evil-link")); err != nil {
			t.Fatal(err)
		}

		_, err := secureJoin(rootfs, "/etc2/evil-link")
		if err == nil {
			t.Error("expected error for symlink escaping rootfs")
		}
	})

	t.Run("path with relative symlink escaping rootfs", func(t *testing.T) {
		etcDir := filepath.Join(rootfs, "etc3")
		if err := os.MkdirAll(etcDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../../../../etc/shadow", filepath.Join(etcDir, "evil-link")); err != nil {
			t.Fatal(err)
		}

		_, err := secureJoin(rootfs, "/etc3/evil-link")
		if err == nil {
			t.Error("expected error for symlink escaping rootfs")
		}
	})

	t.Run("path with chained symlinks", func(t *testing.T) {
		dir1 := filepath.Join(rootfs, "dir1")
		dir2 := filepath.Join(rootfs, "dir2")
		if err := os.MkdirAll(dir1, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir2, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(dir2, filepath.Join(dir1, "link-to-dir2")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(dir1, "link-to-dir2"), filepath.Join(rootfs, "link-to-link")); err != nil {
			t.Fatal(err)
		}

		result, err := secureJoin(rootfs, "/link-to-link/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(dir2, "file.txt")
		if result != expected {
			t.Errorf("secureJoin() = %q, want %q", result, expected)
		}
	})

	t.Run("non-existent path is allowed", func(t *testing.T) {
		result, err := secureJoin(rootfs, "/nonexistent/path/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(rootfs, "nonexistent", "path", "file.txt")
		if result != expected {
			t.Errorf("secureJoin() = %q, want %q", result, expected)
		}
	})
}

func TestValidatePathSymlinkEscape(t *testing.T) {
	rootfs := t.TempDir()

	etcDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/shadow", filepath.Join(etcDir, "evil-link")); err != nil {
		t.Fatal(err)
	}

	_, err := validatePath(rootfs, "/etc/evil-link")
	if err == nil {
		t.Error("expected error for path through symlink escaping rootfs")
	}
	if err != nil && !strings.Contains(err.Error(), "points outside root") {
		t.Errorf("expected 'points outside root' error, got: %v", err)
	}
}

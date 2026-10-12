package provision

import (
	"os"
	"path/filepath"
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

package provision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Inoriol/lxspawn/internal/config"
)

func TestBootProvisionerWritesScript(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:   config.ProvisionBoot,
		Script: "#!/bin/bash\necho hello",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	bootDir := filepath.Join(rootfs, "etc", "lxspawn", "boot")
	entries, err := os.ReadDir(bootDir)
	if err != nil {
		t.Fatalf("failed to read boot dir: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 script, got %d", len(entries))
	}

	if entries[0].Name() != "00-script.sh" {
		t.Errorf("script name = %q, want %q", entries[0].Name(), "00-script.sh")
	}

	content, err := os.ReadFile(filepath.Join(bootDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "#!/bin/bash\necho hello" {
		t.Errorf("script content = %q", string(content))
	}
}

func TestBootProvisionerScriptNumbering(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}
	baseDir := t.TempDir()

	for i := 0; i < 3; i++ {
		prov := config.Provision{
			Mode:   config.ProvisionBoot,
			Script: "#!/bin/bash\necho script",
		}
		if err := provisioner.Provision(prov, ctx, fh, baseDir); err != nil {
			t.Fatalf("Provision[%d] failed: %v", i, err)
		}
	}

	bootDir := filepath.Join(rootfs, "etc", "lxspawn", "boot")
	entries, err := os.ReadDir(bootDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 scripts, got %d", len(entries))
	}

	expected := []string{"00-script.sh", "01-script.sh", "02-script.sh"}
	for i, e := range expected {
		if entries[i].Name() != e {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i].Name(), e)
		}
	}
}

func TestBootProvisionerInstallsService(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:   config.ProvisionBoot,
		Script: "#!/bin/bash\necho hello",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	servicePath := filepath.Join(rootfs, "etc", "systemd", "system", "lxspawn-boot.service")
	if _, err := os.Stat(servicePath); err != nil {
		t.Errorf("boot service not installed: %v", err)
	}

	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "[Unit]") {
		t.Error("service missing [Unit] section")
	}
	if !strings.Contains(string(content), "WantedBy=multi-user.target") {
		t.Error("service missing WantedBy")
	}
}

func TestBootProvisionerEnablesService(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:   config.ProvisionBoot,
		Script: "#!/bin/bash\necho hello",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	enableLink := filepath.Join(rootfs, "etc", "systemd", "system", "multi-user.target.wants", "lxspawn-boot.service")
	target, err := os.Readlink(enableLink)
	if err != nil {
		t.Fatalf("enable symlink not found: %v", err)
	}

	expectedTarget := filepath.Join(rootfs, "etc", "systemd", "system", "lxspawn-boot.service")
	if target != expectedTarget {
		t.Errorf("symlink target = %q, want %q", target, expectedTarget)
	}
}

func TestBootProvisionerServiceNotDuplicated(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}
	baseDir := t.TempDir()

	for i := 0; i < 3; i++ {
		prov := config.Provision{
			Mode:   config.ProvisionBoot,
			Script: "#!/bin/bash\necho hello",
		}
		if err := provisioner.Provision(prov, ctx, fh, baseDir); err != nil {
			t.Fatalf("Provision[%d] failed: %v", i, err)
		}
	}

	servicePath := filepath.Join(rootfs, "etc", "systemd", "system", "lxspawn-boot.service")
	if _, err := os.Stat(servicePath); err != nil {
		t.Errorf("boot service not found: %v", err)
	}
}

func TestBootProvisionerTemplateRendering(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:   config.ProvisionBoot,
		Script: "#!/bin/bash\necho {{.Name}} port={{.Param.port}}",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	bootDir := filepath.Join(rootfs, "etc", "lxspawn", "boot")
	content, err := os.ReadFile(filepath.Join(bootDir, "00-script.sh"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "echo myserver") {
		t.Error("template not rendered: missing name")
	}
	if !strings.Contains(string(content), "port=8080") {
		t.Error("template not rendered: missing param")
	}
}

func TestBootProvisionerScriptIsExecutable(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:   config.ProvisionBoot,
		Script: "#!/bin/bash\necho hello",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	scriptPath := filepath.Join(rootfs, "etc", "lxspawn", "boot", "00-script.sh")
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode()&0111 == 0 {
		t.Errorf("script is not executable: %o", info.Mode())
	}
}

func TestBootProvisionerWithFileRef(t *testing.T) {
	rootfs := t.TempDir()
	baseDir := t.TempDir()

	scriptContent := "#!/bin/bash\necho from file"
	scriptPath := filepath.Join(baseDir, "setup.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewBootProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode: config.ProvisionBoot,
		File: &config.FileRef{URL: "setup.sh"},
	}

	if err := provisioner.Provision(prov, ctx, fh, baseDir); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	bootDir := filepath.Join(rootfs, "etc", "lxspawn", "boot")
	content, err := os.ReadFile(filepath.Join(bootDir, "00-script.sh"))
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != scriptContent {
		t.Errorf("content = %q, want %q", string(content), scriptContent)
	}
}

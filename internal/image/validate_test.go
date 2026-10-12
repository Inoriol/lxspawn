package image

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateInitSbinInit(t *testing.T) {
	rootfs := t.TempDir()
	initPath := filepath.Join(rootfs, "sbin", "init")
	if err := os.MkdirAll(filepath.Dir(initPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initPath, []byte("init"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ValidateInit(rootfs); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateInitLibSystemd(t *testing.T) {
	rootfs := t.TempDir()
	initPath := filepath.Join(rootfs, "lib", "systemd", "systemd")
	if err := os.MkdirAll(filepath.Dir(initPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initPath, []byte("systemd"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ValidateInit(rootfs); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateInitUsrLibSystemd(t *testing.T) {
	rootfs := t.TempDir()
	initPath := filepath.Join(rootfs, "usr", "lib", "systemd", "systemd")
	if err := os.MkdirAll(filepath.Dir(initPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initPath, []byte("systemd"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ValidateInit(rootfs); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateInitNotFound(t *testing.T) {
	rootfs := t.TempDir()

	if err := ValidateInit(rootfs); err == nil {
		t.Fatal("expected error when no init system found")
	}
}

func TestValidateInitDirectoryNotFile(t *testing.T) {
	rootfs := t.TempDir()
	initDir := filepath.Join(rootfs, "sbin", "init")
	if err := os.MkdirAll(initDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := ValidateInit(rootfs); err == nil {
		t.Fatal("expected error when init is a directory")
	}
}

func TestValidateInitEmptyRootfs(t *testing.T) {
	rootfs := t.TempDir()

	err := ValidateInit(rootfs)
	if err == nil {
		t.Fatal("expected error for empty rootfs")
	}
}

func TestValidateInitMultiplePaths(t *testing.T) {
	rootfs := t.TempDir()

	sbinInit := filepath.Join(rootfs, "sbin", "init")
	if err := os.MkdirAll(filepath.Dir(sbinInit), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sbinInit, []byte("init"), 0755); err != nil {
		t.Fatal(err)
	}

	libSystemd := filepath.Join(rootfs, "lib", "systemd", "systemd")
	if err := os.MkdirAll(filepath.Dir(libSystemd), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libSystemd, []byte("systemd"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ValidateInit(rootfs); err != nil {
		t.Errorf("should pass when multiple init paths exist: %v", err)
	}
}

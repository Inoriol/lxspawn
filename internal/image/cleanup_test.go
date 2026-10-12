package image

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupRootfsRemovesMachineID(t *testing.T) {
	rootfs := t.TempDir()
	etcDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		t.Fatal(err)
	}

	machineID := filepath.Join(etcDir, "machine-id")
	if err := os.WriteFile(machineID, []byte("abc123"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CleanupRootfs(rootfs); err != nil {
		t.Fatalf("CleanupRootfs failed: %v", err)
	}

	if _, err := os.Stat(machineID); !os.IsNotExist(err) {
		t.Error("machine-id should be removed")
	}
}

func TestCleanupRootfsRemovesResolvConf(t *testing.T) {
	rootfs := t.TempDir()
	etcDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		t.Fatal(err)
	}

	resolvConf := filepath.Join(etcDir, "resolv.conf")
	if err := os.WriteFile(resolvConf, []byte("nameserver 127.0.0.53"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CleanupRootfs(rootfs); err != nil {
		t.Fatalf("CleanupRootfs failed: %v", err)
	}

	// resolv.conf should be replaced with working DNS
	content, err := os.ReadFile(resolvConf)
	if err != nil {
		t.Fatalf("resolv.conf not created: %v", err)
	}

	if !strings.Contains(string(content), "nameserver 8.8.8.8") {
		t.Error("resolv.conf should contain public DNS servers")
	}
}

func TestCleanupRootfsWritesSecuretty(t *testing.T) {
	rootfs := t.TempDir()

	if err := CleanupRootfs(rootfs); err != nil {
		t.Fatalf("CleanupRootfs failed: %v", err)
	}

	securetty := filepath.Join(rootfs, "etc", "securetty")
	content, err := os.ReadFile(securetty)
	if err != nil {
		t.Fatalf("securetty not created: %v", err)
	}

	for i := 0; i <= 9; i++ {
		expected := strings.TrimSpace(string(content))
		pts := "pts/" + string(rune('0'+i))
		if !strings.Contains(expected, pts) {
			t.Errorf("securetty missing %s", pts)
		}
	}
}

func TestCleanupRootfsMissingMachineID(t *testing.T) {
	rootfs := t.TempDir()
	etcDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CleanupRootfs(rootfs); err != nil {
		t.Fatalf("CleanupRootfs should not fail when machine-id is missing: %v", err)
	}
}

func TestCleanupRootfsMissingResolvConf(t *testing.T) {
	rootfs := t.TempDir()
	etcDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CleanupRootfs(rootfs); err != nil {
		t.Fatalf("CleanupRootfs should not fail when resolv.conf is missing: %v", err)
	}
}

func TestCleanupRootfsCreatesEtcDir(t *testing.T) {
	rootfs := t.TempDir()

	if err := CleanupRootfs(rootfs); err != nil {
		t.Fatalf("CleanupRootfs failed: %v", err)
	}

	etcDir := filepath.Join(rootfs, "etc")
	if _, err := os.Stat(etcDir); err != nil {
		t.Errorf("etc directory not created: %v", err)
	}
}

func TestCleanupRootfsSecurettyContent(t *testing.T) {
	rootfs := t.TempDir()

	if err := CleanupRootfs(rootfs); err != nil {
		t.Fatalf("CleanupRootfs failed: %v", err)
	}

	securetty := filepath.Join(rootfs, "etc", "securetty")
	content, err := os.ReadFile(securetty)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}

	for i, line := range lines {
		expected := "pts/" + string(rune('0'+i))
		if line != expected {
			t.Errorf("line[%d] = %q, want %q", i, line, expected)
		}
	}
}

package kernel

import (
	"testing"
)

func TestCheckKernelVersion(t *testing.T) {
	major, minor, err := GetVersion()
	if err != nil {
		t.Fatalf("GetVersion() failed: %v", err)
	}

	if major < 1 {
		t.Errorf("expected major >= 1, got %d", major)
	}

	ok, err := CheckKernelVersion(major, minor)
	if err != nil {
		t.Fatalf("CheckKernelVersion(%d, %d) failed: %v", major, minor, err)
	}
	if !ok {
		t.Errorf("current kernel %d.%d should satisfy CheckKernelVersion(%d, %d)", major, minor, major, minor)
	}

	ok, err = CheckKernelVersion(1, 0)
	if err != nil {
		t.Fatalf("CheckKernelVersion(1, 0) failed: %v", err)
	}
	if !ok {
		t.Error("any modern kernel should satisfy CheckKernelVersion(1, 0)")
	}

	ok, err = CheckKernelVersion(999, 0)
	if err != nil {
		t.Fatalf("CheckKernelVersion(999, 0) failed: %v", err)
	}
	if ok {
		t.Error("no kernel should satisfy CheckKernelVersion(999, 0)")
	}
}

func TestIsBtrfs(t *testing.T) {
	result := IsBtrfs("/nonexistent-path-xyz")
	if result {
		t.Error("IsBtrfs should return false for nonexistent path")
	}
}

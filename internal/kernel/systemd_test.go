package kernel

import (
	"testing"
)

func TestSystemdVersion(t *testing.T) {
	version, err := SystemdVersion()
	if err != nil {
		t.Skipf("systemctl not available or failed: %v", err)
	}
	if version < 245 {
		t.Errorf("expected systemd version >= 245, got %d", version)
	}
}

func TestSupportsPrivateUsersIdentity(t *testing.T) {
	result := SupportsPrivateUsersIdentity()
	version, err := SystemdVersion()
	if err != nil {
		t.Skipf("systemctl not available: %v", err)
	}
	expected := version >= 249
	if result != expected {
		t.Errorf("SupportsPrivateUsersIdentity() = %v, expected %v (systemd version %d)", result, expected, version)
	}
}

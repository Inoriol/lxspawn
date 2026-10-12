package network

import (
	"testing"
)

func TestIsVirtualInterface(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		expected bool
	}{
		{"docker interface", "docker0", true},
		{"podman interface", "podman0", true},
		{"veth interface", "veth123", true},
		{"bridge interface", "br-abc123", true},
		{"virtual bridge", "virbr0", true},
		{"loopback", "lo", true},
		{"tunnel", "tun0", true},
		{"wireguard", "wg0", true},
		{"ethernet", "enp2s0", false},
		{"wifi", "wlp3s0", false},
		{"custom ethernet", "eth0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVirtualInterface(tt.iface)
			if result != tt.expected {
				t.Errorf("isVirtualInterface(%s) = %v, want %v", tt.iface, result, tt.expected)
			}
		})
	}
}

func TestGetInterfaceType(t *testing.T) {
	// This test requires actual system interfaces, so we'll just test the function exists
	// and doesn't panic on invalid input
	result := getInterfaceType("nonexistent0")
	if result != "Unknown" {
		t.Errorf("getInterfaceType(nonexistent0) = %s, want Unknown", result)
	}
}

func TestHasCarrier(t *testing.T) {
	// This test requires actual system interfaces, so we'll just test the function exists
	// and doesn't panic on invalid input
	result := hasCarrier("nonexistent0")
	if result != false {
		t.Errorf("hasCarrier(nonexistent0) = %v, want false", result)
	}
}

func TestGetInterfaceIP(t *testing.T) {
	// This test requires actual system interfaces, so we'll just test the function exists
	// and doesn't panic on invalid input
	result := getInterfaceIP("nonexistent0")
	if result != "" {
		t.Errorf("getInterfaceIP(nonexistent0) = %s, want empty string", result)
	}
}

// Integration tests - these require actual system interfaces
// Run with: go test -tags=integration

func TestDetectPrimaryInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	iface, err := DetectPrimaryInterface()
	if err != nil {
		t.Logf("DetectPrimaryInterface() returned error (may be expected in CI): %v", err)
		return
	}

	if iface == "" {
		t.Error("DetectPrimaryInterface() returned empty string")
	}

	t.Logf("Detected primary interface: %s", iface)
}

func TestListPhysicalInterfaces(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ifaces, err := ListPhysicalInterfaces()
	if err != nil {
		t.Logf("ListPhysicalInterfaces() returned error (may be expected in CI): %v", err)
		return
	}

	if len(ifaces) == 0 {
		t.Error("ListPhysicalInterfaces() returned empty list")
	}

	for _, iface := range ifaces {
		t.Logf("Interface: %s, Type: %s, IP: %s, Carrier: %v, Primary: %v",
			iface.Name, iface.Type, iface.IP, iface.Carrier, iface.Primary)
	}
}

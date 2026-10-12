package image

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Inoriol/lxspawn/internal/config"
)

func TestInjectNetworkConfigHostMode(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkHost, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkDir := filepath.Join(rootfs, "etc", "systemd", "network")
	if _, err := os.Stat(networkDir); err == nil {
		t.Error("host mode should not create network directory")
	}
}

func TestInjectNetworkConfigPrivateMode(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkPrivate, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkFile := filepath.Join(rootfs, "etc", "systemd", "network", "80-container-host0.network")
	content, err := os.ReadFile(networkFile)
	if err != nil {
		t.Fatalf("network config not created: %v", err)
	}

	if !strings.Contains(string(content), "Name=host0") {
		t.Error("missing Name=host0")
	}
	if !strings.Contains(string(content), "DHCP=yes") {
		t.Error("missing DHCP=yes")
	}
	if !strings.Contains(string(content), "LinkLocalAddressing=ipv4") {
		t.Error("missing LinkLocalAddressing=ipv4")
	}
}

func TestInjectNetworkConfigBridgeMode(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkBridge, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkFile := filepath.Join(rootfs, "etc", "systemd", "network", "80-container-host0.network")
	content, err := os.ReadFile(networkFile)
	if err != nil {
		t.Fatalf("network config not created: %v", err)
	}

	if !strings.Contains(string(content), "Name=host0") {
		t.Error("missing Name=host0")
	}
}

func TestInjectNetworkConfigZoneMode(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkZone, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkFile := filepath.Join(rootfs, "etc", "systemd", "network", "80-container-host0.network")
	if _, err := os.ReadFile(networkFile); err != nil {
		t.Fatalf("network config not created: %v", err)
	}
}

func TestInjectNetworkConfigMacvlan(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkMacvlan, "eth0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkFile := filepath.Join(rootfs, "etc", "systemd", "network", "80-container-mv-eth0.network")
	content, err := os.ReadFile(networkFile)
	if err != nil {
		t.Fatalf("network config not created: %v", err)
	}

	if !strings.Contains(string(content), "Name=mv-eth0") {
		t.Errorf("missing Name=mv-eth0 in:\n%s", string(content))
	}
	if !strings.Contains(string(content), "DHCP=yes") {
		t.Error("missing DHCP=yes")
	}
}

func TestInjectNetworkConfigIpvlan(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkIpvlan, "eth0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkFile := filepath.Join(rootfs, "etc", "systemd", "network", "80-container-iv-eth0.network")
	content, err := os.ReadFile(networkFile)
	if err != nil {
		t.Fatalf("network config not created: %v", err)
	}

	if !strings.Contains(string(content), "Name=iv-eth0") {
		t.Errorf("missing Name=iv-eth0 in:\n%s", string(content))
	}
	if !strings.Contains(string(content), "DHCP=yes") {
		t.Error("missing DHCP=yes")
	}
}

func TestInjectNetworkConfigWritesPreset(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkPrivate, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	presetFile := filepath.Join(rootfs, "etc", "systemd", "system-preset", "80-lxspawn.preset")
	content, err := os.ReadFile(presetFile)
	if err != nil {
		t.Fatalf("preset not created: %v", err)
	}

	if !strings.Contains(string(content), "enable systemd-networkd.service") {
		t.Error("preset should enable systemd-networkd")
	}
}

func TestInjectNetworkConfigCreatesDirectories(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkPrivate, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkDir := filepath.Join(rootfs, "etc", "systemd", "network")
	if _, err := os.Stat(networkDir); err != nil {
		t.Errorf("network directory not created: %v", err)
	}

	presetDir := filepath.Join(rootfs, "etc", "systemd", "system-preset")
	if _, err := os.Stat(presetDir); err != nil {
		t.Errorf("preset directory not created: %v", err)
	}
}

func TestInjectNetworkConfigMacvlanDifferentInterfaces(t *testing.T) {
	tests := []struct {
		iface      string
		expectName string
	}{
		{"eth0", "mv-eth0"},
		{"enp2s0", "mv-enp2s0"},
		{"wlan0", "mv-wlan0"},
	}

	for _, tc := range tests {
		t.Run(tc.iface, func(t *testing.T) {
			rootfs := t.TempDir()

			if err := InjectNetworkConfig(rootfs, config.NetworkMacvlan, tc.iface); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			networkFile := filepath.Join(rootfs, "etc", "systemd", "network", "80-container-"+tc.expectName+".network")
			content, err := os.ReadFile(networkFile)
			if err != nil {
				t.Fatalf("network config not created: %v", err)
			}

			if !strings.Contains(string(content), "Name="+tc.expectName) {
				t.Errorf("missing Name=%s", tc.expectName)
			}
		})
	}
}

func TestInjectNetworkConfigDHCPSection(t *testing.T) {
	rootfs := t.TempDir()

	if err := InjectNetworkConfig(rootfs, config.NetworkPrivate, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	networkFile := filepath.Join(rootfs, "etc", "systemd", "network", "80-container-host0.network")
	content, err := os.ReadFile(networkFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "[DHCP]") {
		t.Error("missing [DHCP] section")
	}
	if !strings.Contains(string(content), "UseHostname=no") {
		t.Error("missing UseHostname=no")
	}
}

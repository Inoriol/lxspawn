package image

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/logger"
)

func InjectNetworkConfig(rootfs string, mode config.NetworkMode, interfaceName string) error {
	if mode == config.NetworkHost {
		return nil
	}

	logger.Info("Injecting systemd-networkd configuration for private networking")

	networkDir := filepath.Join(rootfs, "etc", "systemd", "network")
	if err := os.MkdirAll(networkDir, 0755); err != nil {
		return fmt.Errorf("failed to create network directory: %w", err)
	}

	var networkContent string
	var networkFile string

	switch mode {
	case config.NetworkMacvlan:
		// For macvlan, the interface in the container is named mv-<host-interface>
		containerInterface := fmt.Sprintf("mv-%s", interfaceName)
		networkFile = filepath.Join(networkDir, fmt.Sprintf("80-container-%s.network", containerInterface))
		networkContent = fmt.Sprintf(`[Match]
Name=%s

[Network]
DHCP=yes
LinkLocalAddressing=ipv4

[DHCP]
UseHostname=no
`, containerInterface)
	case config.NetworkIpvlan:
		// For ipvlan, the interface in the container is named iv-<host-interface>
		containerInterface := fmt.Sprintf("iv-%s", interfaceName)
		networkFile = filepath.Join(networkDir, fmt.Sprintf("80-container-%s.network", containerInterface))
		networkContent = fmt.Sprintf(`[Match]
Name=%s

[Network]
DHCP=yes
LinkLocalAddressing=ipv4

[DHCP]
UseHostname=no
`, containerInterface)
	default:
		// For private/bridge/zone modes, the interface is host0
		networkFile = filepath.Join(networkDir, "80-container-host0.network")
		networkContent = `[Match]
Name=host0

[Network]
DHCP=yes
LinkLocalAddressing=ipv4

[DHCP]
UseHostname=no
`
	}

	if err := os.WriteFile(networkFile, []byte(networkContent), 0644); err != nil {
		return fmt.Errorf("failed to write network config: %w", err)
	}

	presetDir := filepath.Join(rootfs, "etc", "systemd", "system-preset")
	if err := os.MkdirAll(presetDir, 0755); err != nil {
		return fmt.Errorf("failed to create preset directory: %w", err)
	}

	presetFile := filepath.Join(presetDir, "80-lxspawn.preset")
	presetContent := "enable systemd-networkd.service\n"
	if err := os.WriteFile(presetFile, []byte(presetContent), 0644); err != nil {
		return fmt.Errorf("failed to write preset: %w", err)
	}

	logger.Info("Network config injected successfully")
	return nil
}

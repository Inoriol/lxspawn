package compiler

import (
	"fmt"
	"hash/fnv"
	"os"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/network"
)

type CompiledFiles struct {
	Nspawn       string
	NspawnPath   string
	Slice        string
	SlicePath    string
	DropIn       string
	DropInPath   string
	HooksDropIn  string
	HooksPath    string
	HookScripts  map[string]string
}

func Compile(cfg *config.Config) (*CompiledFiles, error) {
	cf := &CompiledFiles{
		HookScripts: make(map[string]string),
	}

	if err := validateVolumes(cfg); err != nil {
		return nil, fmt.Errorf("volume validation failed: %w", err)
	}

	if err := resolveNetworkMode(cfg); err != nil {
		return nil, fmt.Errorf("failed to resolve network mode: %w", err)
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to compile .nspawn: %w", err)
	}
	cf.Nspawn = nspawn
	cf.NspawnPath = fmt.Sprintf("/etc/systemd/nspawn/%s.nspawn", cfg.Name)

	if hasResources(cfg) {
		slice := compileSlice(cfg)
		cf.Slice = slice
		cf.SlicePath = fmt.Sprintf("/etc/systemd/system/machine-%s.slice", cfg.Name)
	}

	dropin := compileDropIn(cfg)
	cf.DropIn = dropin
	cf.DropInPath = fmt.Sprintf("/etc/systemd/system/systemd-nspawn@%s.service.d/lxspawn.conf", cfg.Name)

	if hasHooks(cfg) || hasHostAccess(cfg) {
		hooksDropIn, scripts, err := compileHooks(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to compile hooks: %w", err)
		}
		cf.HooksDropIn = hooksDropIn
		cf.HooksPath = fmt.Sprintf("/etc/systemd/system/systemd-nspawn@%s.service.d/lxspawn-hooks.conf", cfg.Name)
		cf.HookScripts = scripts
	}

	return cf, nil
}

func validateVolumes(cfg *config.Config) error {
	for i, v := range cfg.Volumes {
		if v.Type == "tmpfs" {
			continue
		}
		
		if _, err := os.Stat(v.Host); os.IsNotExist(err) {
			logger.Warn(fmt.Sprintf("volumes[%d]: host path does not exist: %s", i, v.Host))
		}
	}
	return nil
}

// resolveNetworkMode resolves adaptive mode to bridge or macvlan
func resolveNetworkMode(cfg *config.Config) error {
	if cfg.Network.Mode != config.NetworkAdaptive {
		return nil
	}

	// Check if bridge exists
	if cfg.Network.Bridge != "" {
		exists, err := network.BridgeExists(cfg.Network.Bridge)
		if err == nil && exists {
			logger.Info(fmt.Sprintf("Adaptive mode: using bridge %s", cfg.Network.Bridge))
			cfg.Network.Mode = config.NetworkBridge
			return nil
		}
		logger.Warn(fmt.Sprintf("Bridge %s not found, falling back to macvlan", cfg.Network.Bridge))
	}

	// Resolve interface
	iface := cfg.Network.Interface
	if iface == "auto" || iface == "" {
		netCfg, err := network.LoadNetworkConfig()
		if err != nil {
			return fmt.Errorf("no interface specified and auto-detection not configured. Run: sudo lxspawn setup network")
		}
		iface = netCfg.AutoInterface
		logger.Info(fmt.Sprintf("Adaptive mode: using auto-detected interface %s", iface))
	}

	// Enable host access
	cfg.Network.HostAccess.Enabled = true
	if cfg.Network.HostAccess.Interface == "" {
		cfg.Network.HostAccess.Interface = generateHostAccessInterfaceName(cfg.Name)
	}
	if cfg.Network.HostAccess.IP == "" {
		ip, err := generateHostAccessIP(iface, cfg.Name)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to generate host access IP: %v", err))
		} else {
			cfg.Network.HostAccess.IP = ip
		}
	}

	if cfg.Network.HostAccess.IP != "" {
		logger.Info(fmt.Sprintf("Adaptive mode: using macvlan on %s with host access (IP: %s)", iface, cfg.Network.HostAccess.IP))
	} else {
		logger.Info(fmt.Sprintf("Adaptive mode: using macvlan on %s with host access", iface))
	}
	cfg.Network.Mode = config.NetworkMacvlan
	cfg.Network.Interface = iface

	return nil
}

func hasResources(cfg *config.Config) bool {
	r := cfg.Resources
	return r.CPUs > 0 || r.Memory != "" || r.MemoryHigh != "" || r.Swap != "" || r.Tasks != "" || (r.IOWeight > 0 && r.IOWeight != 100)
}

func hasHooks(cfg *config.Config) bool {
	return cfg.Hooks.PreStart != "" || cfg.Hooks.PostStart != "" || cfg.Hooks.PostStop != ""
}

func hasHostAccess(cfg *config.Config) bool {
	return cfg.Network.HostAccess.Enabled && cfg.Network.Mode == config.NetworkMacvlan
}

func generateHostAccessIP(hostInterface, machineName string) (string, error) {
	ipWithCIDR, err := network.GetInterfaceIPWithCIDR(hostInterface)
	if err != nil {
		return "", fmt.Errorf("failed to get IP for interface %s: %w", hostInterface, err)
	}

	ip, cidr, err := parseIPAndCIDR(ipWithCIDR)
	if err != nil {
		return "", err
	}

	h := fnv.New32a()
	h.Write([]byte(machineName))
	hash := h.Sum32()

	lastOctet := (hash % 254) + 1
	return fmt.Sprintf("%d.%d.%d.%d/%d", ip[0], ip[1], ip[2], lastOctet, cidr), nil
}

func parseIPAndCIDR(ipWithCIDR string) ([4]byte, int, error) {
	var ip [4]byte
	var cidr int

	n, err := fmt.Sscanf(ipWithCIDR, "%d.%d.%d.%d/%d", &ip[0], &ip[1], &ip[2], &ip[3], &cidr)
	if err != nil || n != 5 {
		return ip, 0, fmt.Errorf("failed to parse IP/CIDR: %s", ipWithCIDR)
	}

	return ip, cidr, nil
}

func generateHostAccessInterfaceName(machineName string) string {
	// Linux interface names are limited to 15 characters (IFNAMSIZ - 1)
	// Format: mv-<machinename>, truncated if necessary
	const maxLen = 15
	const prefix = "mv-"
	
	if len(machineName) <= maxLen-len(prefix) {
		return prefix + machineName
	}
	// Truncate machine name to fit, keeping it unique by using the hash
	h := fnv.New32a()
	h.Write([]byte(machineName))
	hash := h.Sum32()
	hashSuffix := fmt.Sprintf("%x", hash)[:4] // 4 char hash suffix
	
	availLen := maxLen - len(prefix) - len(hashSuffix) - 1 // -1 for separator
	if availLen < 0 {
		availLen = 0
	}
	truncated := machineName[:availLen]
	return fmt.Sprintf("%s%s-%s", prefix, truncated, hashSuffix)
}



package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var validMachineNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`)

func ValidateMachineName(name string) error {
	if name == "" {
		return fmt.Errorf("machine name cannot be empty")
	}
	if len(name) == 1 {
		if !regexp.MustCompile(`^[a-zA-Z0-9]$`).MatchString(name) {
			return fmt.Errorf("machine name must be alphanumeric")
		}
		return nil
	}
	if !validMachineNameRegex.MatchString(name) {
		return fmt.Errorf("machine name must start and end with an alphanumeric character, contain only alphanumeric characters, hyphens, and underscores, and be between 2 and 64 characters")
	}
	if isReservedName(name) {
		return fmt.Errorf("machine name %q conflicts with a systemd reserved name", name)
	}
	return nil
}

func Parse(filename string) (*Config, error) {
	return parse(filename, false)
}

func ParseStrict(filename string) (*Config, error) {
	return parse(filename, true)
}

func parse(filename string, strict bool) (*Config, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if strict {
		decoder := yaml.NewDecoder(strings.NewReader(string(data)))
		decoder.KnownFields(true)
		if err := decoder.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	}

	if cfg.Network.Mode == "" {
		cfg.Network.Mode = NetworkHost
	}

	if cfg.Advanced.Hostname == "" {
		cfg.Advanced.Hostname = cfg.Name
	}

	if cfg.Security.NoNewPrivileges == nil {
		t := true
		cfg.Security.NoNewPrivileges = &t
	}

	if cfg.Security.Seccomp == nil {
		t := true
		cfg.Security.Seccomp = &t
	}

	if cfg.Resources.IOWeight == 0 {
		cfg.Resources.IOWeight = 100
	}

	if cfg.Storage.Size == "" {
		cfg.Storage.Size = "10G"
	}

	return &cfg, nil
}

func Validate(cfg *Config) error {
	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(cfg.Name) == 1 {
		// Single character names are allowed if alphanumeric
		if !regexp.MustCompile(`^[a-zA-Z0-9]$`).MatchString(cfg.Name) {
			return fmt.Errorf("name must start and end with an alphanumeric character")
		}
	} else if !validMachineNameRegex.MatchString(cfg.Name) {
		return fmt.Errorf("name must start and end with an alphanumeric character, contain only alphanumeric characters, hyphens, and underscores, and be between 2 and 64 characters")
	}
	if isReservedName(cfg.Name) {
		return fmt.Errorf("name %q conflicts with a systemd reserved name", cfg.Name)
	}

	if cfg.Distro == "" {
		return fmt.Errorf("distro is required")
	}

	if err := validateAdvanced(cfg); err != nil {
		return err
	}

	if err := validateSecurity(cfg); err != nil {
		return err
	}

	if err := validateNetwork(cfg); err != nil {
		return err
	}

	if err := validateVolumes(cfg); err != nil {
		return err
	}

	if err := validateProvision(cfg); err != nil {
		return err
	}

	if err := validateProbes(cfg); err != nil {
		return err
	}

	if err := validateGPU(cfg); err != nil {
		return err
	}

	if err := validateStorage(cfg); err != nil {
		return err
	}

	if err := validateResources(cfg); err != nil {
		return err
	}

	return nil
}

func validateAdvanced(cfg *Config) error {
	if cfg.Advanced.Hostname != "" {
		if strings.ContainsAny(cfg.Advanced.Hostname, "\n\r") {
			return fmt.Errorf("advanced.hostname cannot contain newlines")
		}
		if !validMachineNameRegex.MatchString(cfg.Advanced.Hostname) && len(cfg.Advanced.Hostname) > 1 {
			return fmt.Errorf("advanced.hostname contains invalid characters")
		}
	}

	for k, v := range cfg.Advanced.Environment {
		if strings.ContainsAny(k, "\n\r=") {
			return fmt.Errorf("environment variable name %q cannot contain newlines or '='", k)
		}
		if strings.ContainsAny(v, "\n\r") {
			return fmt.Errorf("environment variable %q value cannot contain newlines", k)
		}
	}

	return nil
}

func validateSecurity(cfg *Config) error {
	for _, cap := range cfg.Security.Capabilities {
		if !strings.HasPrefix(cap, "CAP_") {
			return fmt.Errorf("capability %q must start with CAP_", cap)
		}
		if strings.ContainsAny(cap, " \n\r") {
			return fmt.Errorf("capability %q cannot contain spaces or newlines", cap)
		}
	}

	for _, cap := range cfg.Security.DropCapabilities {
		if !strings.HasPrefix(cap, "CAP_") {
			return fmt.Errorf("drop_capability %q must start with CAP_", cap)
		}
		if strings.ContainsAny(cap, " \n\r") {
			return fmt.Errorf("drop_capability %q cannot contain spaces or newlines", cap)
		}
	}

	for _, path := range cfg.Security.Inaccessible {
		if strings.ContainsAny(path, "\n\r") {
			return fmt.Errorf("inaccessible path %q cannot contain newlines", path)
		}
		if !filepath.IsAbs(path) {
			return fmt.Errorf("inaccessible path %q must be absolute", path)
		}
	}

	return nil
}

func validateResources(cfg *Config) error {
	if cfg.Resources.Memory != "" {
		if !isValidSizeString(cfg.Resources.Memory) {
			return fmt.Errorf("resources.memory %q is not a valid size", cfg.Resources.Memory)
		}
	}
	if cfg.Resources.MemoryHigh != "" {
		if !isValidSizeString(cfg.Resources.MemoryHigh) {
			return fmt.Errorf("resources.memory_high %q is not a valid size", cfg.Resources.MemoryHigh)
		}
	}
	if cfg.Resources.Swap != "" {
		if !isValidSizeString(cfg.Resources.Swap) {
			return fmt.Errorf("resources.swap %q is not a valid size", cfg.Resources.Swap)
		}
	}
	if cfg.Resources.Tasks != "" && cfg.Resources.Tasks != "infinity" {
		if _, err := strconv.ParseUint(cfg.Resources.Tasks, 10, 64); err != nil {
			return fmt.Errorf("resources.tasks %q must be a number or 'infinity'", cfg.Resources.Tasks)
		}
	}
	return nil
}

var validSizeRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?[KMGTP]?$`)

func isValidSizeString(s string) bool {
	return validSizeRegex.MatchString(s)
}

var reservedNames = map[string]bool{
	"system":      true,
	"user":        true,
	"systemd":     true,
	"dbus":        true,
	"init":        true,
	"shutdown":    true,
	"halt":        true,
	"poweroff":    true,
	"reboot":      true,
	"run":         true,
	"login":       true,
	"host":        true,
	"container":   true,
	"machinectl":  true,
	"journalctl":  true,
	"localhost":   true,
}

func isReservedName(name string) bool {
	lower := strings.ToLower(name)
	return reservedNames[lower]
}

var validIfaceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateNetwork(cfg *Config) error {
	switch cfg.Network.Mode {
	case NetworkHost, "":
	case NetworkPrivate, NetworkBridge, NetworkZone, NetworkMacvlan, NetworkIpvlan, NetworkAdaptive:
	default:
		return fmt.Errorf("invalid network mode %q: must be host, private, bridge, zone, macvlan, ipvlan, or adaptive", cfg.Network.Mode)
	}

	if cfg.Network.Mode == NetworkBridge && cfg.Network.Bridge == "" {
		return fmt.Errorf("network.bridge is required when mode is bridge")
	}

	if cfg.Network.Mode == NetworkZone && cfg.Network.Zone == "" {
		return fmt.Errorf("network.zone is required when mode is zone")
	}

	if cfg.Network.Mode == NetworkMacvlan && cfg.Network.Interface == "" {
		return fmt.Errorf("network.interface is required when mode is macvlan")
	}

	if cfg.Network.Mode == NetworkIpvlan && cfg.Network.Interface == "" {
		return fmt.Errorf("network.interface is required when mode is ipvlan")
	}

	if cfg.Network.Bridge != "" && !validIfaceNameRegex.MatchString(cfg.Network.Bridge) {
		return fmt.Errorf("network.bridge %q contains invalid characters", cfg.Network.Bridge)
	}

	if cfg.Network.Zone != "" && !validIfaceNameRegex.MatchString(cfg.Network.Zone) {
		return fmt.Errorf("network.zone %q contains invalid characters", cfg.Network.Zone)
	}

	if cfg.Network.Interface != "" && cfg.Network.Interface != "auto" && !validIfaceNameRegex.MatchString(cfg.Network.Interface) {
		return fmt.Errorf("network.interface %q contains invalid characters", cfg.Network.Interface)
	}

	if cfg.Network.HostAccess.Interface != "" && !validIfaceNameRegex.MatchString(cfg.Network.HostAccess.Interface) {
		return fmt.Errorf("network.host_access.interface %q contains invalid characters", cfg.Network.HostAccess.Interface)
	}

	// Adaptive mode validation
	if cfg.Network.Mode == NetworkAdaptive {
		// Bridge and interface are optional for adaptive mode
		// If interface is empty, it will use "auto"
		if cfg.Network.Interface == "" {
			cfg.Network.Interface = "auto"
		}
		// Host access defaults
		if cfg.Network.HostAccess.Interface == "" {
			cfg.Network.HostAccess.Interface = "mv-lxspawn"
		}
	}

	if len(cfg.Network.Ports) > 0 && cfg.Network.Mode == NetworkHost {
		return fmt.Errorf("port forwarding requires a private network mode (private, bridge, or zone)")
	}

	for i, port := range cfg.Network.Ports {
		if err := validatePort(port); err != nil {
			return fmt.Errorf("network.ports[%d]: %w", i, err)
		}
	}

	return nil
}

var portRegex = regexp.MustCompile(`^(tcp|udp):(\d+):(\d+)$`)

func validatePort(port string) error {
	matches := portRegex.FindStringSubmatch(port)
	if matches == nil {
		return fmt.Errorf("invalid port format %q: expected <proto>:<host>:<container> (e.g., tcp:8080:80)", port)
	}
	
	hostPort, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("invalid host port: %w", err)
	}
	if hostPort < 1 || hostPort > 65535 {
		return fmt.Errorf("host port %d is out of valid range (1-65535)", hostPort)
	}
	
	containerPort, err := strconv.Atoi(matches[3])
	if err != nil {
		return fmt.Errorf("invalid container port: %w", err)
	}
	if containerPort < 1 || containerPort > 65535 {
		return fmt.Errorf("container port %d is out of valid range (1-65535)", containerPort)
	}
	
	return nil
}

func validateVolumes(cfg *Config) error {
	// Sensitive paths that should not be exposed to containers
	sensitivePaths := []string{
		"/etc/shadow",
		"/etc/gshadow",
		"/etc/master.passwd",
		"/proc",
		"/sys",
		"/dev",
		"/run",
		"/var/run",
	}

	for i, v := range cfg.Volumes {
		if v.Type == "tmpfs" {
			if v.Mount == "" {
				return fmt.Errorf("volumes[%d]: tmpfs volume requires mount point", i)
			}
			if strings.ContainsAny(v.Mount, "\n\r") {
				return fmt.Errorf("volumes[%d]: mount point cannot contain newlines", i)
			}
			continue
		}
		if v.Host == "" {
			return fmt.Errorf("volumes[%d]: host path is required", i)
		}
		if v.Mount == "" {
			return fmt.Errorf("volumes[%d]: mount point is required", i)
		}
		if strings.ContainsAny(v.Host, "\n\r") {
			return fmt.Errorf("volumes[%d]: host path cannot contain newlines", i)
		}
		if strings.ContainsAny(v.Mount, "\n\r") {
			return fmt.Errorf("volumes[%d]: mount point cannot contain newlines", i)
		}
		if !filepath.IsAbs(v.Host) {
			return fmt.Errorf("volumes[%d]: host path must be absolute: %s", i, v.Host)
		}
		if !filepath.IsAbs(v.Mount) {
			return fmt.Errorf("volumes[%d]: mount point must be absolute: %s", i, v.Mount)
		}

		// Check for sensitive paths
		cleanHost := filepath.Clean(v.Host)
		for _, sensitive := range sensitivePaths {
			if cleanHost == sensitive || strings.HasPrefix(cleanHost, sensitive+"/") {
				return fmt.Errorf("volumes[%d]: host path %q exposes sensitive system directory", i, v.Host)
			}
		}
	}
	return nil
}

func validateProbes(cfg *Config) error {
	for i, p := range cfg.Probes {
		if p.Mode != "readiness" {
			return fmt.Errorf("probes[%d]: invalid mode %q (only \"readiness\" is supported)", i, p.Mode)
		}
		if p.Description == "" {
			return fmt.Errorf("probes[%d]: description is required", i)
		}
		if p.Script == "" {
			return fmt.Errorf("probes[%d]: script is required", i)
		}
	}
	return nil
}

func validateProvision(cfg *Config) error {
	for i, p := range cfg.Provision {
		switch p.Mode {
		case ProvisionBoot, ProvisionSystem, ProvisionUser:
			if p.Script == "" && p.File == nil {
				return fmt.Errorf("provision[%d]: %s mode requires script or file", i, p.Mode)
			}
		case ProvisionData:
			if p.Path == "" {
				return fmt.Errorf("provision[%d]: data mode requires path", i)
			}
			if p.Content == "" && p.File == nil {
				return fmt.Errorf("provision[%d]: data mode requires content or file", i)
			}
		case ProvisionYQ:
			if p.Path == "" {
				return fmt.Errorf("provision[%d]: yq mode requires path", i)
			}
			if p.Expression == "" {
				return fmt.Errorf("provision[%d]: yq mode requires expression", i)
			}
		default:
			return fmt.Errorf("provision[%d]: invalid mode %q", i, p.Mode)
		}
	}
	return nil
}

func validateGPU(cfg *Config) error {
	if !cfg.GPU.Enabled {
		return nil
	}

	validTypes := map[string]bool{
		"":       true,
		"auto":   true,
		"intel":  true,
		"amd":    true,
		"nvidia": true,
		"mali":   true,
		"adreno": true,
	}

	if !validTypes[cfg.GPU.Type] {
		return fmt.Errorf("gpu.type must be one of: auto, intel, amd, nvidia, mali, adreno")
	}

	for i, device := range cfg.GPU.Devices {
		if device == "" {
			return fmt.Errorf("gpu.devices[%d]: device path cannot be empty", i)
		}
		if !strings.HasPrefix(device, "/dev/") {
			return fmt.Errorf("gpu.devices[%d]: device path must start with /dev/: %s", i, device)
		}
		cleanDevice := filepath.Clean(device)
		if cleanDevice != device {
			return fmt.Errorf("gpu.devices[%d]: device path must be clean (no '..' or extra slashes): %s", i, device)
		}
		if strings.Contains(device, "..") {
			return fmt.Errorf("gpu.devices[%d]: device path cannot contain '..': %s", i, device)
		}
	}

	return nil
}

func validateStorage(cfg *Config) error {
	if cfg.Storage.Size == "" {
		return nil
	}

	_, err := ParseSizeToGB(cfg.Storage.Size)
	if err != nil {
		return fmt.Errorf("storage.size %q: %w", cfg.Storage.Size, err)
	}

	return nil
}

func ParseSizeToGB(size string) (int, error) {
	size = strings.TrimSpace(size)
	if size == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}

	size = strings.ToUpper(size)

	var num float64
	var gb int

	if strings.HasSuffix(size, "G") {
		numStr := strings.TrimSuffix(size, "G")
		var err error
		num, err = strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size number %q in %q: %w", numStr, size, err)
		}
		if num <= 0 {
			return 0, fmt.Errorf("size must be positive: %s", size)
		}
		gb = int(num)
		if gb == 0 && num > 0 {
			gb = 1
		}
	} else if strings.HasSuffix(size, "M") {
		numStr := strings.TrimSuffix(size, "M")
		var err error
		num, err = strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size number %q in %q: %w", numStr, size, err)
		}
		if num <= 0 {
			return 0, fmt.Errorf("size must be positive: %s", size)
		}
		gb = int(num / 1024)
		if gb == 0 && num > 0 {
			gb = 1
		}
	} else if strings.HasSuffix(size, "T") {
		numStr := strings.TrimSuffix(size, "T")
		var err error
		num, err = strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size number %q in %q: %w", numStr, size, err)
		}
		if num <= 0 {
			return 0, fmt.Errorf("size must be positive: %s", size)
		}
		gb = int(num * 1024)
	} else {
		return 0, fmt.Errorf("size %q must end with G, M, or T", size)
	}

	return gb, nil
}

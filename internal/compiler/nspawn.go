package compiler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/gpu"
	"github.com/Inoriol/lxspawn/internal/kernel"
)

func compileNspawn(cfg *config.Config) (string, error) {
	sections := make(map[string][]string)

	execLines, err := buildExecSection(cfg)
	if err != nil {
		return "", err
	}
	if len(execLines) > 0 {
		sections["Exec"] = execLines
	}

	filesLines := buildFilesSection(cfg)
	if len(filesLines) > 0 {
		sections["Files"] = filesLines
	}

	networkLines := buildNetworkSection(cfg)
	if len(networkLines) > 0 {
		sections["Network"] = networkLines
	}

	return writeINIOrdered(sections, []string{"Exec", "Files", "Network"}), nil
}

func buildExecSection(cfg *config.Config) ([]string, error) {
	var lines []string

	lines = append(lines, "Boot=yes")

	if cfg.Advanced.Hostname != "" {
		lines = append(lines, fmt.Sprintf("Hostname=%s", cfg.Advanced.Hostname))
	}

	if cfg.Advanced.Ephemeral {
		lines = append(lines, "Ephemeral=yes")
	}

	envKeys := make([]string, 0, len(cfg.Advanced.Environment))
	for k := range cfg.Advanced.Environment {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	for _, k := range envKeys {
		lines = append(lines, fmt.Sprintf("Environment=%s=%s", k, cfg.Advanced.Environment[k]))
	}

	if cfg.Security.NoNewPrivileges != nil && *cfg.Security.NoNewPrivileges {
		lines = append(lines, "NoNewPrivileges=yes")
	}

	allCaps := make([]string, 0, len(cfg.Security.Capabilities)+4)
	allCaps = append(allCaps, cfg.Security.Capabilities...)

	if cfg.GPU.Enabled {
		gpuCaps, err := getGPUCapabilities(cfg)
		if err == nil {
			allCaps = append(allCaps, gpuCaps...)
		}
	}

	if len(allCaps) > 0 {
		lines = append(lines, fmt.Sprintf("Capability=%s", strings.Join(allCaps, " ")))
	}

	if len(cfg.Security.DropCapabilities) > 0 {
		lines = append(lines, fmt.Sprintf("DropCapability=%s", strings.Join(cfg.Security.DropCapabilities, " ")))
	}

	if cfg.Security.PrivateUsers != "" {
		lines = append(lines, fmt.Sprintf("PrivateUsers=%s", cfg.Security.PrivateUsers))
	} else if cfg.Security.Seccomp != nil && !*cfg.Security.Seccomp {
		lines = append(lines, "PrivateUsers=no")
	} else if cfg.Network.Mode == config.NetworkMacvlan || cfg.Network.Mode == config.NetworkIpvlan {
		lines = append(lines, "PrivateUsers=no")
	} else if kernel.SupportsPrivateUsersIdentity() {
		lines = append(lines, "PrivateUsers=identity")
	}

	if cfg.Security.Seccomp != nil && !*cfg.Security.Seccomp {
		lines = append(lines, "SystemCallFilter=")
	} else if len(cfg.Security.SystemCallFilter) > 0 {
		lines = append(lines, fmt.Sprintf("SystemCallFilter=%s", strings.Join(cfg.Security.SystemCallFilter, " ")))
	}

	if cfg.Journal.Link != "" {
		lines = append(lines, fmt.Sprintf("LinkJournal=%s", cfg.Journal.Link))
	} else {
		lines = append(lines, "LinkJournal=try-guest")
	}

	return lines, nil
}

func buildFilesSection(cfg *config.Config) []string {
	var lines []string

	if cfg.Advanced.ReadOnly {
		lines = append(lines, "ReadOnly=yes")
	}

	if cfg.GPU.Enabled {
		gpuDevices, err := getGPUDevices(cfg)
		if err == nil {
			for _, device := range gpuDevices {
				lines = append(lines, fmt.Sprintf("Bind=%s", device))
			}
		}
	}

	for _, v := range cfg.Volumes {
		if v.Type == "tmpfs" {
			entry := v.Mount
			if v.Options != "" {
				entry = fmt.Sprintf("%s:%s", v.Mount, v.Options)
			}
			lines = append(lines, fmt.Sprintf("TemporaryFileSystem=%s", entry))
			continue
		}

		entry := fmt.Sprintf("%s:%s", v.Host, v.Mount)
		if v.ReadOnly {
			lines = append(lines, fmt.Sprintf("BindReadOnly=%s", entry))
		} else {
			lines = append(lines, fmt.Sprintf("Bind=%s", entry))
		}
	}

	for _, path := range cfg.Security.Inaccessible {
		lines = append(lines, fmt.Sprintf("Inaccessible=%s", path))
	}

	return lines
}

func getGPUDevices(cfg *config.Config) ([]string, error) {
	if len(cfg.GPU.Devices) > 0 {
		return cfg.GPU.Devices, nil
	}

	detector := gpu.NewDetector()
	gpus, err := detector.DetectGPUs()
	if err != nil {
		return nil, err
	}

	selectedGPU, err := gpu.SelectGPU(cfg.GPU.Type, gpus)
	if err != nil {
		return nil, err
	}

	return gpu.GetDeviceNodes(selectedGPU), nil
}

func getGPUCapabilities(cfg *config.Config) ([]string, error) {
	if len(cfg.GPU.Devices) > 0 {
		return nil, nil
	}

	detector := gpu.NewDetector()
	gpus, err := detector.DetectGPUs()
	if err != nil {
		return nil, err
	}

	selectedGPU, err := gpu.SelectGPU(cfg.GPU.Type, gpus)
	if err != nil {
		return nil, err
	}

	return gpu.GetRequiredCapabilities(selectedGPU), nil
}

func buildNetworkSection(cfg *config.Config) []string {
	var lines []string

	switch cfg.Network.Mode {
	case config.NetworkHost:
		lines = append(lines, "Private=no")
	case config.NetworkPrivate:
		lines = append(lines, "Private=yes")
		lines = append(lines, "VirtualEthernet=yes")
	case config.NetworkBridge:
		lines = append(lines, "Private=yes")
		lines = append(lines, "VirtualEthernet=yes")
		lines = append(lines, fmt.Sprintf("Bridge=%s", cfg.Network.Bridge))
	case config.NetworkZone:
		lines = append(lines, "Private=yes")
		lines = append(lines, "VirtualEthernet=yes")
		lines = append(lines, fmt.Sprintf("Zone=%s", cfg.Network.Zone))
	case config.NetworkMacvlan:
		lines = append(lines, fmt.Sprintf("MACVLAN=%s", cfg.Network.Interface))
	case config.NetworkIpvlan:
		lines = append(lines, fmt.Sprintf("IPVLAN=%s", cfg.Network.Interface))
	}

	for _, port := range cfg.Network.Ports {
		lines = append(lines, fmt.Sprintf("Port=%s", port))
	}

	return lines
}

func writeINIOrdered(sections map[string][]string, order []string) string {
	var b strings.Builder
	for _, section := range order {
		lines, ok := sections[section]
		if !ok || len(lines) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("[%s]\n", section))
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

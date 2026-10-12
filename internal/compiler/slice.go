package compiler

import (
	"fmt"
	"strings"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/gpu"
)

func compileSlice(cfg *config.Config) string {
	var b strings.Builder

	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=%s Container Slice\n\n", cfg.Name))

	b.WriteString("[Slice]\n")

	r := cfg.Resources
	if r.Memory != "" {
		b.WriteString(fmt.Sprintf("MemoryMax=%s\n", r.Memory))
	}
	if r.MemoryHigh != "" {
		b.WriteString(fmt.Sprintf("MemoryHigh=%s\n", r.MemoryHigh))
	}
	if r.CPUs > 0 {
		pct := int(r.CPUs * 100)
		b.WriteString(fmt.Sprintf("CPUQuota=%d%%\n", pct))
	}
	if r.Tasks != "" {
		b.WriteString(fmt.Sprintf("TasksMax=%s\n", r.Tasks))
	}
	if r.IOWeight > 0 && r.IOWeight != 100 {
		b.WriteString(fmt.Sprintf("IOWeight=%d\n", r.IOWeight))
	}
	if r.Swap != "" {
		b.WriteString(fmt.Sprintf("MemorySwapMax=%s\n", r.Swap))
	}

	return b.String()
}

func compileDropIn(cfg *config.Config) string {
	var b strings.Builder

	b.WriteString("[Service]\n")

	b.WriteString("Environment=SYSTEMD_NSPAWN_LOCK=0\n")

	if hasResources(cfg) {
		b.WriteString(fmt.Sprintf("Slice=machine-%s.slice\n", cfg.Name))
	}

	// Enable cgroup delegation for containers that need it (e.g., Docker-in-container)
	if cfg.Security.Seccomp != nil && !*cfg.Security.Seccomp {
		b.WriteString("Delegate=yes\n")
	}

	if cfg.GPU.Enabled {
		deviceAllowRules, err := getGPUDeviceAllowRules(cfg)
		if err == nil {
			for _, rule := range deviceAllowRules {
				b.WriteString(fmt.Sprintf("DeviceAllow=%s\n", rule))
			}
		}

		caps := getGPUCapabilities(cfg)
		if len(caps) > 0 {
			existingCaps := cfg.Security.Capabilities
			allCaps := append(existingCaps, caps...)
			b.WriteString(fmt.Sprintf("Capability=%s\n", strings.Join(allCaps, " ")))
		}
	}

	return b.String()
}

func getGPUDeviceAllowRules(cfg *config.Config) ([]string, error) {
	if len(cfg.GPU.Devices) > 0 {
		var rules []string
		for _, device := range cfg.GPU.Devices {
			rules = append(rules, fmt.Sprintf("%s rw", device))
		}
		return rules, nil
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

	return gpu.GetDeviceAllowRules(selectedGPU), nil
}

func getGPUCapabilities(cfg *config.Config) []string {
	if len(cfg.GPU.Devices) > 0 {
		return nil
	}

	detector := gpu.NewDetector()
	gpus, err := detector.DetectGPUs()
	if err != nil {
		return nil
	}

	selectedGPU, err := gpu.SelectGPU(cfg.GPU.Type, gpus)
	if err != nil {
		return nil
	}

	return gpu.GetRequiredCapabilities(selectedGPU)
}

func compileHooks(cfg *config.Config) (string, map[string]string) {
	var b strings.Builder
	scripts := make(map[string]string)

	hooksDir := fmt.Sprintf("/var/lib/lxspawn/%s/.hooks", cfg.Name)

	b.WriteString("[Service]\n")

	// Generate host-side macvlan setup if host_access is enabled
	if cfg.Network.HostAccess.Enabled && cfg.Network.Mode == config.NetworkMacvlan {
		hostAccessScript := generateHostAccessScript(cfg)
		if cfg.Hooks.PostStart != "" {
			// Prepend to existing post_start hook
			cfg.Hooks.PostStart = hostAccessScript + "\n" + cfg.Hooks.PostStart
		} else {
			cfg.Hooks.PostStart = hostAccessScript
		}

		// Generate cleanup script for post_stop
		cleanupScript := generateHostAccessCleanupScript(cfg)
		if cfg.Hooks.PostStop != "" {
			// Prepend to existing post_stop hook
			cfg.Hooks.PostStop = cleanupScript + "\n" + cfg.Hooks.PostStop
		} else {
			cfg.Hooks.PostStop = cleanupScript
		}
	}

	if cfg.Hooks.PreStart != "" {
		scriptPath := hooksDir + "/pre_start.sh"
		b.WriteString(fmt.Sprintf("ExecStartPre=%s\n", scriptPath))
		scripts["pre_start.sh"] = ensureShebang(cfg.Hooks.PreStart)
	}

	if cfg.Hooks.PostStart != "" {
		scriptPath := hooksDir + "/post_start.sh"
		b.WriteString(fmt.Sprintf("ExecStartPost=%s\n", scriptPath))
		scripts["post_start.sh"] = ensureShebang(cfg.Hooks.PostStart)
	}

	if cfg.Hooks.PostStop != "" {
		scriptPath := hooksDir + "/post_stop.sh"
		b.WriteString(fmt.Sprintf("ExecStopPost=%s\n", scriptPath))
		scripts["post_stop.sh"] = ensureShebang(cfg.Hooks.PostStop)
	}

	return b.String(), scripts
}

func generateHostAccessScript(cfg *config.Config) string {
	iface := cfg.Network.Interface
	hostIface := cfg.Network.HostAccess.Interface
	ip := cfg.Network.HostAccess.IP

	script := fmt.Sprintf(`# Create host-side macvlan for container visibility
ip link add %s link %s type macvlan mode bridge`, hostIface, iface)

	if ip != "" && ip != "dhcp" {
		script += fmt.Sprintf(`
ip addr add %s dev %s`, ip, hostIface)
	}

	script += fmt.Sprintf(`
ip link set %s up`, hostIface)

	return script
}

func generateHostAccessCleanupScript(cfg *config.Config) string {
	hostIface := cfg.Network.HostAccess.Interface

	return fmt.Sprintf(`# Remove host-side macvlan
ip link del %s 2>/dev/null || true`, hostIface)
}

func ensureShebang(script string) string {
	trimmed := strings.TrimSpace(script)
	if strings.HasPrefix(trimmed, "#!") {
		if !strings.HasSuffix(script, "\n") {
			return script + "\n"
		}
		return script
	}
	return "#!/bin/bash\nset -euo pipefail\n" + script + "\n"
}

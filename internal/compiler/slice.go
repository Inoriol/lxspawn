package compiler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/gpu"
)

var validIfaceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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

	// Use local copies to avoid mutating the input config
	preStart := cfg.Hooks.PreStart
	postStart := cfg.Hooks.PostStart
	postStop := cfg.Hooks.PostStop

	// Generate host-side macvlan setup if host_access is enabled
	if cfg.Network.HostAccess.Enabled && cfg.Network.Mode == config.NetworkMacvlan {
		hostAccessScript := generateHostAccessScript(cfg)
		if postStart != "" {
			postStart = hostAccessScript + "\n" + postStart
		} else {
			postStart = hostAccessScript
		}

		// Generate cleanup script for post_stop
		cleanupScript := generateHostAccessCleanupScript(cfg)
		if postStop != "" {
			postStop = cleanupScript + "\n" + postStop
		} else {
			postStop = cleanupScript
		}
	}

	if preStart != "" {
		scriptPath := hooksDir + "/pre_start.sh"
		b.WriteString(fmt.Sprintf("ExecStartPre=%s\n", scriptPath))
		scripts["pre_start.sh"] = ensureShebang(preStart)
	}

	if postStart != "" {
		scriptPath := hooksDir + "/post_start.sh"
		b.WriteString(fmt.Sprintf("ExecStartPost=%s\n", scriptPath))
		scripts["post_start.sh"] = ensureShebang(postStart)
	}

	if postStop != "" {
		scriptPath := hooksDir + "/post_stop.sh"
		b.WriteString(fmt.Sprintf("ExecStopPost=%s\n", scriptPath))
		scripts["post_stop.sh"] = ensureShebang(postStop)
	}

	return b.String(), scripts
}

func generateHostAccessScript(cfg *config.Config) string {
	iface := cfg.Network.Interface
	hostIface := cfg.Network.HostAccess.Interface
	ip := cfg.Network.HostAccess.IP

	if !validIfaceNameRegex.MatchString(iface) {
		return fmt.Sprintf("echo 'Invalid interface name: %s'", iface)
	}
	if !validIfaceNameRegex.MatchString(hostIface) {
		return fmt.Sprintf("echo 'Invalid host interface name: %s'", hostIface)
	}

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

	if !validIfaceNameRegex.MatchString(hostIface) {
		return fmt.Sprintf("echo 'Invalid host interface name: %s'", hostIface)
	}

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

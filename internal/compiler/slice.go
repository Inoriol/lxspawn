package compiler

import (
	"fmt"
	"math"
	"net"
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
		pct := int(math.Round(r.CPUs * 100))
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

func compileHooks(cfg *config.Config) (string, map[string]string, error) {
	var b strings.Builder
	scripts := make(map[string]string)

	hooksDir := fmt.Sprintf("/var/lib/lxspawn/%s/.hooks", cfg.Name)

	b.WriteString("[Service]\n")

	preStart := cfg.Hooks.PreStart
	postStart := cfg.Hooks.PostStart
	postStop := cfg.Hooks.PostStop

	if cfg.Network.HostAccess.Enabled && cfg.Network.Mode == config.NetworkMacvlan {
		hostAccessScript, err := generateHostAccessScript(cfg)
		if err != nil {
			return "", nil, fmt.Errorf("invalid host_access configuration: %w", err)
		}
		if postStart != "" {
			postStart = hostAccessScript + "\n" + postStart
		} else {
			postStart = hostAccessScript
		}

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

	return b.String(), scripts, nil
}

func generateHostAccessScript(cfg *config.Config) (string, error) {
	iface := cfg.Network.Interface
	hostIface := cfg.Network.HostAccess.Interface
	ip := cfg.Network.HostAccess.IP

	if !validIfaceNameRegex.MatchString(iface) {
		return "", fmt.Errorf("invalid interface name: %s", iface)
	}
	if !validIfaceNameRegex.MatchString(hostIface) {
		return "", fmt.Errorf("invalid host interface name: %s", hostIface)
	}

	if ip != "" && ip != "dhcp" {
		if _, _, err := net.ParseCIDR(ip); err != nil {
			if net.ParseIP(ip) == nil {
				return "", fmt.Errorf("invalid IP address: %s", ip)
			}
		}
	}

	script := fmt.Sprintf(`# Create host-side macvlan for container visibility
ip link add %s link %s type macvlan mode bridge`, shellQuote(hostIface), shellQuote(iface))

	if ip != "" && ip != "dhcp" {
		script += fmt.Sprintf(`
ip addr add %s dev %s`, shellQuote(ip), shellQuote(hostIface))
	}

	script += fmt.Sprintf(`
ip link set %s up`, shellQuote(hostIface))

	return script, nil
}

func generateHostAccessCleanupScript(cfg *config.Config) string {
	hostIface := cfg.Network.HostAccess.Interface

	if !validIfaceNameRegex.MatchString(hostIface) {
		return "# Invalid host interface name, skipping cleanup"
	}

	return fmt.Sprintf(`# Remove host-side macvlan
ip link del %s 2>/dev/null || true`, shellQuote(hostIface))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
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

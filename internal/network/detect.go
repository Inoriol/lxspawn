package network

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InterfaceInfo contains information about a network interface
type InterfaceInfo struct {
	Name    string
	Type    string // "Ethernet", "WiFi", "Unknown"
	IP      string
	Carrier bool
	Primary bool
}

// DetectPrimaryInterface returns the primary network interface
// Logic: Parse `ip route show default`, filter virtual interfaces,
// return first physical interface with carrier.
// Resolves bridge/bond slaves to find the actual physical interface.
func DetectPrimaryInterface() (string, error) {
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "default") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "dev" && i+1 < len(fields) {
					iface := fields[i+1]
					resolved := resolveToPhysical(iface)
					if resolved != "" && hasCarrier(resolved) {
						return resolved, nil
					}
					if isPhysicalInterface(iface) && hasCarrier(iface) {
						return iface, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no primary interface found")
}

// resolveToPhysical resolves bridge/bond/vlan slaves to the underlying physical interface
func resolveToPhysical(iface string) string {
	if isPhysicalInterface(iface) {
		return iface
	}

	slavePath := fmt.Sprintf("/sys/class/net/%s/lower_%s", iface, iface)
	if _, err := os.Stat(slavePath); err == nil {
		link, err := os.Readlink(slavePath)
		if err == nil {
			slave := filepath.Base(link)
			return resolveToPhysical(slave)
		}
	}

	bondSlavesPath := fmt.Sprintf("/sys/class/net/%s/bonding/slaves", iface)
	if data, err := os.ReadFile(bondSlavesPath); err == nil {
		slaves := strings.Fields(strings.TrimSpace(string(data)))
		if len(slaves) > 0 {
			return resolveToPhysical(slaves[0])
		}
	}

	brPortPath := fmt.Sprintf("/sys/class/net/%s/brport", iface)
	if _, err := os.Stat(brPortPath); err == nil {
		return iface
	}

	masterPath := fmt.Sprintf("/sys/class/net/%s/master", iface)
	if link, err := os.Readlink(masterPath); err == nil {
		master := filepath.Base(link)
		return resolveToPhysical(master)
	}

	return ""
}

// ListPhysicalInterfaces returns all physical interfaces
func ListPhysicalInterfaces() ([]InterfaceInfo, error) {
	var interfaces []InterfaceInfo

	cmd := exec.Command("ip", "-o", "link", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	primary, _ := DetectPrimaryInterface()

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := strings.TrimSuffix(fields[1], ":")
		if idx := strings.Index(name, "@"); idx != -1 {
			name = name[:idx]
		}

		if isVirtualInterface(name) {
			continue
		}

		resolved := resolveToPhysical(name)
		if resolved == "" && !isPhysicalInterface(name) {
			continue
		}

		effectiveName := name
		if resolved != "" {
			effectiveName = resolved
		}

		iface := InterfaceInfo{
			Name:    effectiveName,
			Type:    getInterfaceType(effectiveName),
			IP:      getInterfaceIP(name),
			Carrier: hasCarrier(effectiveName),
			Primary: effectiveName == primary,
		}

		found := false
		for i, existing := range interfaces {
			if existing.Name == iface.Name {
				found = true
				if iface.Primary {
					interfaces[i].Primary = true
				}
				break
			}
		}
		if !found {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces, nil
}

// isVirtualInterface checks if an interface is virtual
func isVirtualInterface(name string) bool {
	virtualPrefixes := []string{
		"docker", "podman", "veth", "br-", "virbr", "vnet",
		"lo", "sit", "tun", "tap", "wg", "vxlan",
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// isPhysicalInterface checks if an interface is physical
func isPhysicalInterface(name string) bool {
	devicePath := fmt.Sprintf("/sys/class/net/%s/device", name)
	if _, err := os.Stat(devicePath); err == nil {
		return true
	}
	return false
}

// hasCarrier checks if an interface has carrier (link detected)
func hasCarrier(name string) bool {
	carrierPath := fmt.Sprintf("/sys/class/net/%s/carrier", name)
	data, err := os.ReadFile(carrierPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

// getInterfaceType returns the type of interface (Ethernet, WiFi, etc.)
func getInterfaceType(name string) string {
	// Check if wireless
	typePath := fmt.Sprintf("/sys/class/net/%s/type", name)
	data, err := os.ReadFile(typePath)
	if err != nil {
		return "Unknown"
	}

	// Type 1 is Ethernet, type 801 is IEEE 802.11 (WiFi)
	typeStr := strings.TrimSpace(string(data))
	if typeStr == "1" {
		// Check if it's actually wireless
		wirelessPath := fmt.Sprintf("/sys/class/net/%s/wireless", name)
		if _, err := os.Stat(wirelessPath); err == nil {
			return "WiFi"
		}
		return "Ethernet"
	}

	return "Unknown"
}

// getInterfaceIP returns the IP address of an interface
func getInterfaceIP(name string) string {
	cmd := exec.Command("ip", "-4", "addr", "show", "dev", name)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output: "inet 192.168.0.200/24 ..."
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.Contains(field, "/") {
					// Return IP without CIDR
					parts := strings.Split(field, "/")
					return parts[0]
				}
			}
		}
	}

	return ""
}

// GetInterfaceIPWithCIDR returns the IP address with CIDR notation of an interface
func GetInterfaceIPWithCIDR(name string) (string, error) {
	cmd := exec.Command("ip", "-4", "addr", "show", "dev", name)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get IP for interface %s: %w", name, err)
	}

	// Parse output: "inet 192.168.0.200/24 ..."
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.Contains(field, "/") {
					return field, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no IPv4 address found for interface %s", name)
}

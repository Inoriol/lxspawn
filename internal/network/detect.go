package network

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
// return first physical interface with carrier
func DetectPrimaryInterface() (string, error) {
	// Get default route interface
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}

	// Parse output: "default via 192.168.0.1 dev enp2s0 ..."
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "default") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "dev" && i+1 < len(fields) {
					iface := fields[i+1]
					// Check if it's a physical interface with carrier
					if isPhysicalInterface(iface) && hasCarrier(iface) {
						return iface, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no primary interface found")
}

// ListPhysicalInterfaces returns all physical interfaces
func ListPhysicalInterfaces() ([]InterfaceInfo, error) {
	var interfaces []InterfaceInfo

	// Get all interfaces
	cmd := exec.Command("ip", "-o", "link", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	// Get primary interface
	primary, _ := DetectPrimaryInterface()

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Parse: "2: enp2s0: <BROADCAST,MULTICAST,UP,LOWER_UP> ..."
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Extract interface name (remove trailing colon and @... suffix)
		name := strings.TrimSuffix(fields[1], ":")
		if idx := strings.Index(name, "@"); idx != -1 {
			name = name[:idx]
		}

		// Skip virtual interfaces
		if isVirtualInterface(name) {
			continue
		}

		// Skip if not physical
		if !isPhysicalInterface(name) {
			continue
		}

		iface := InterfaceInfo{
			Name:    name,
			Type:    getInterfaceType(name),
			IP:      getInterfaceIP(name),
			Carrier: hasCarrier(name),
			Primary: name == primary,
		}

		interfaces = append(interfaces, iface)
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

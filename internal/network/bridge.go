package network

import (
	"fmt"
	"os"
)

// BridgeExists checks if a bridge interface exists
func BridgeExists(name string) (bool, error) {
	// Check if /sys/class/net/<name>/bridge exists
	bridgePath := fmt.Sprintf("/sys/class/net/%s/bridge", name)
	if _, err := os.Stat(bridgePath); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("failed to check bridge existence: %w", err)
	}
}

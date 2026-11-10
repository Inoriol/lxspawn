package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Inoriol/lxspawn/internal/deploy"
)

// NetworkConfig stores detected network configuration
type NetworkConfig struct {
	AutoInterface string    `json:"auto_interface"`
	DetectedAt    time.Time `json:"detected_at"`
}

// configPathOverride allows tests to override the config path
var configPathOverride string

// SetConfigPathOverride sets a custom config path (for testing)
func SetConfigPathOverride(path string) {
	configPathOverride = path
}

// GetConfigPath returns the path to the network config file
func GetConfigPath() string {
	if configPathOverride != "" {
		return configPathOverride
	}
	return filepath.Join(deploy.LxspawnDataDir, "config.json")
}

// LoadNetworkConfig loads the network configuration from disk
func LoadNetworkConfig() (*NetworkConfig, error) {
	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("network config not found. Run: sudo lxspawn setup network")
		}
		return nil, fmt.Errorf("failed to read network config: %w", err)
	}

	var config NetworkConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse network config: %w", err)
	}

	return &config, nil
}

// SaveNetworkConfig saves the network configuration to disk
func SaveNetworkConfig(cfg *NetworkConfig) error {
	configPath := GetConfigPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal network config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write network config: %w", err)
	}

	return nil
}

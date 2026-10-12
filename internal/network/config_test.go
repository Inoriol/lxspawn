package network

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNetworkConfig(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	SetConfigPathOverride(configPath)
	defer SetConfigPathOverride("")

	// Test SaveNetworkConfig
	cfg := &NetworkConfig{
		AutoInterface: "enp2s0",
		DetectedAt:    time.Now(),
	}

	err := SaveNetworkConfig(cfg)
	if err != nil {
		t.Fatalf("SaveNetworkConfig() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Test LoadNetworkConfig
	loaded, err := LoadNetworkConfig()
	if err != nil {
		t.Fatalf("LoadNetworkConfig() error = %v", err)
	}

	if loaded.AutoInterface != cfg.AutoInterface {
		t.Errorf("LoadNetworkConfig().AutoInterface = %s, want %s", loaded.AutoInterface, cfg.AutoInterface)
	}
}

func TestLoadNetworkConfigNotFound(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	SetConfigPathOverride(configPath)
	defer SetConfigPathOverride("")

	// Test loading non-existent config
	_, err := LoadNetworkConfig()
	if err == nil {
		t.Error("LoadNetworkConfig() should return error when config doesn't exist")
	}
}

package network

import (
	"testing"
)

func TestBridgeExists(t *testing.T) {
	// Test with non-existent bridge
	exists, err := BridgeExists("nonexistent-bridge-12345")
	if err != nil {
		t.Errorf("BridgeExists(nonexistent-bridge-12345) returned error: %v", err)
	}
	if exists {
		t.Error("BridgeExists(nonexistent-bridge-12345) = true, want false")
	}

	// Note: Testing with a real bridge requires creating one, which needs root
	// Integration tests should be run manually with:
	// sudo ip link add name testbr0 type bridge
	// go test -tags=integration
	// sudo ip link del testbr0
}

// Integration test - requires root to create bridge
func TestBridgeExistsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test is informational only
	// In a real test environment, you would:
	// 1. Create a test bridge
	// 2. Call BridgeExists
	// 3. Verify it returns true
	// 4. Delete the test bridge

	t.Log("Integration test for BridgeExists requires manual setup")
}

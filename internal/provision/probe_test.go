package provision

import (
	"testing"
	"time"
)

func TestDefaultProbeTimeout(t *testing.T) {
	if defaultProbeTimeout != 10*time.Second {
		t.Errorf("expected default probe timeout to be 10s, got %v", defaultProbeTimeout)
	}
}

func TestProbeRunnerCreation(t *testing.T) {
	runner := NewProbeRunner("test-machine")
	if runner == nil {
		t.Fatal("expected non-nil probe runner")
	}
	if runner.machineName != "test-machine" {
		t.Errorf("expected machine name 'test-machine', got %q", runner.machineName)
	}
}

func TestProbeTimeoutConstant(t *testing.T) {
	// Verify the timeout constant is reasonable (not too short, not too long)
	if defaultProbeTimeout < 5*time.Second {
		t.Errorf("probe timeout too short: %v (minimum 5s)", defaultProbeTimeout)
	}
	if defaultProbeTimeout > 30*time.Second {
		t.Errorf("probe timeout too long: %v (maximum 30s)", defaultProbeTimeout)
	}
}

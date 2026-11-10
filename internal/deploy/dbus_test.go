package deploy

import (
	"fmt"
	"testing"
)

func TestMachineName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "myserver",
			expected: "myserver",
		},
		{
			name:     "name with hyphen",
			input:    "test-machine",
			expected: "test_2dmachine",
		},
		{
			name:     "name with leading digit",
			input:    "3dprint",
			expected: "_33dprint",
		},
		{
			name:     "name with multiple leading digits",
			input:    "123test",
			expected: "_3123test",
		},
		{
			name:     "name with underscore",
			input:    "my_server",
			expected: "my_5fserver",
		},
		{
			name:     "name with dot",
			input:    "my.server",
			expected: "my_2eserver",
		},
		{
			name:     "complex name",
			input:    "test-machine-123",
			expected: "test_2dmachine_2d123",
		},
		{
			name:     "all special chars",
			input:    "a-b.c_d",
			expected: "a_2db_2ec_5fd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := machineName(tt.input)
			if result != tt.expected {
				t.Errorf("machineName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsMachineNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "UnknownMachine error",
			err:      fmt.Errorf("org.freedesktop.machine1.UnknownMachine"),
			expected: true,
		},
		{
			name:     "unknown machine lowercase",
			err:      fmt.Errorf("unknown machine 'test'"),
			expected: true,
		},
		{
			name:     "UnknownObject error",
			err:      fmt.Errorf("org.freedesktop.DBus.Error.UnknownObject"),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("connection refused"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMachineNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("isMachineNotFound(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

package provision

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Inoriol/lxspawn/internal/config"
)

func TestBuildEnvExportNoParams(t *testing.T) {
	result := buildEnvExport(nil)
	if result != "" {
		t.Errorf("expected empty string for nil params, got %q", result)
	}

	result = buildEnvExport(map[string]string{})
	if result != "" {
		t.Errorf("expected empty string for empty params, got %q", result)
	}
}

func TestBuildEnvExportSingleParam(t *testing.T) {
	params := map[string]string{"port": "8080"}
	result := buildEnvExport(params)
	expected := "export LXSPAWN_PARAM_port='8080'"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildEnvExportMultipleParamsSorted(t *testing.T) {
	params := map[string]string{
		"zebra": "z",
		"alpha": "a",
		"mid":   "m",
	}
	result := buildEnvExport(params)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "LXSPAWN_PARAM_alpha") {
		t.Errorf("expected first line to contain alpha, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "LXSPAWN_PARAM_mid") {
		t.Errorf("expected second line to contain mid, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "LXSPAWN_PARAM_zebra") {
		t.Errorf("expected third line to contain zebra, got %q", lines[2])
	}
}

func TestBuildEnvExportEscapesSingleQuotes(t *testing.T) {
	params := map[string]string{"key": "it's a value"}
	result := buildEnvExport(params)
	expected := "export LXSPAWN_PARAM_key='it'\"'\"'s a value'"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildEnvExportSpecialChars(t *testing.T) {
	params := map[string]string{"path": "/usr/bin:/usr/local/bin"}
	result := buildEnvExport(params)
	if !strings.Contains(result, "LXSPAWN_PARAM_path=") {
		t.Errorf("expected param path in output, got %q", result)
	}
	if !strings.Contains(result, "/usr/bin:/usr/local/bin") {
		t.Errorf("expected value preserved, got %q", result)
	}
}

func TestNewExecProvisioner(t *testing.T) {
	p := NewExecProvisioner("testmachine")
	if p == nil {
		t.Fatal("expected non-nil provisioner")
	}
	if p.machineName != "testmachine" {
		t.Errorf("expected machineName 'testmachine', got %q", p.machineName)
	}
	if p.timeout != defaultExecTimeout {
		t.Errorf("expected timeout %v, got %v", defaultExecTimeout, p.timeout)
	}
}

func TestBuildEnvExportParamValues(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		contains []string
	}{
		{
			name:     "empty string value",
			params:   map[string]string{"key": ""},
			contains: []string{"LXSPAWN_PARAM_key='"},
		},
		{
			name:     "numeric value",
			params:   map[string]string{"port": "80"},
			contains: []string{"LXSPAWN_PARAM_port='80'"},
		},
		{
			name:     "value with spaces",
			params:   map[string]string{"msg": "hello world"},
			contains: []string{"LXSPAWN_PARAM_msg='hello world'"},
		},
		{
			name:     "value with dollar sign",
			params:   map[string]string{"var": "$HOME"},
			contains: []string{"LXSPAWN_PARAM_var='$HOME'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEnvExport(tt.params)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q, got %q", expected, result)
				}
			}
		})
	}
}

func TestExecProvisionerModeValidation(t *testing.T) {
	tests := []struct {
		mode config.ProvisionMode
		valid bool
	}{
		{config.ProvisionSystem, true},
		{config.ProvisionUser, true},
		{config.ProvisionBoot, false},
		{config.ProvisionData, false},
		{config.ProvisionYQ, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			isExec := tt.mode == config.ProvisionSystem || tt.mode == config.ProvisionUser
			if isExec != tt.valid {
				t.Errorf("mode %s: expected valid=%v, got %v", tt.mode, tt.valid, isExec)
			}
		})
	}
}

func TestGetExecTimeoutDefault(t *testing.T) {
	os.Unsetenv("LXSPAWN_EXEC_TIMEOUT")
	timeout := getExecTimeout()
	if timeout != defaultExecTimeout {
		t.Errorf("expected default timeout %v, got %v", defaultExecTimeout, timeout)
	}
}

func TestGetExecTimeoutFromEnv(t *testing.T) {
	os.Setenv("LXSPAWN_EXEC_TIMEOUT", "60")
	defer os.Unsetenv("LXSPAWN_EXEC_TIMEOUT")

	timeout := getExecTimeout()
	expected := 60 * time.Minute
	if timeout != expected {
		t.Errorf("expected timeout %v, got %v", expected, timeout)
	}
}

func TestGetExecTimeoutInvalidEnv(t *testing.T) {
	os.Setenv("LXSPAWN_EXEC_TIMEOUT", "invalid")
	defer os.Unsetenv("LXSPAWN_EXEC_TIMEOUT")

	timeout := getExecTimeout()
	if timeout != defaultExecTimeout {
		t.Errorf("expected default timeout %v for invalid env, got %v", defaultExecTimeout, timeout)
	}
}

func TestGetExecTimeoutNegativeEnv(t *testing.T) {
	os.Setenv("LXSPAWN_EXEC_TIMEOUT", "-5")
	defer os.Unsetenv("LXSPAWN_EXEC_TIMEOUT")

	timeout := getExecTimeout()
	if timeout != defaultExecTimeout {
		t.Errorf("expected default timeout %v for negative env, got %v", defaultExecTimeout, timeout)
	}
}

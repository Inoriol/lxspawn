package provision

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

const defaultExecTimeout = 30 * time.Minute

func getExecTimeout() time.Duration {
	if v := os.Getenv("LXSPAWN_EXEC_TIMEOUT"); v != "" {
		if minutes, err := strconv.Atoi(v); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return defaultExecTimeout
}

type ExecProvisioner struct {
	machineName string
	timeout     time.Duration
}

func NewExecProvisioner(machineName string) *ExecProvisioner {
	return &ExecProvisioner{machineName: machineName, timeout: getExecTimeout()}
}

func (e *ExecProvisioner) Provision(prov config.Provision, ctx *TemplateContext, fileHandler *FileHandler, baseDir string) error {
	script := prov.Script

	envExport := buildEnvExport(ctx.Param)

	logger.Info(fmt.Sprintf("Executing %s provision script", prov.Mode))

	maxAttempts := 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var retryCmd *exec.Cmd
		var err error
		switch prov.Mode {
		case config.ProvisionSystem:
			fullScript := fmt.Sprintf("%s\n%s", envExport, script)
			retryCmd, err = e.buildSystemdRunCommand("root", fullScript, e.timeout)
			if err != nil {
				return fmt.Errorf("failed to build systemd-run command: %w", err)
			}
		case config.ProvisionUser:
			user := prov.User
			if user == "" {
				logger.Warn("user mode has no 'user' field set, defaulting to 'root'. Set the 'user' field to specify a container user.")
				user = "root"
			}
			escapedScript := strings.ReplaceAll(script, "'", "'\"'\"'")
			fullScript := fmt.Sprintf("%s\nsu -p %s -c '%s'", envExport, user, escapedScript)
			retryCmd, err = e.buildSystemdRunCommand("root", fullScript, e.timeout)
			if err != nil {
				return fmt.Errorf("failed to build systemd-run command: %w", err)
			}
		default:
			return fmt.Errorf("unsupported provision mode for exec: %s", prov.Mode)
		}
		retryCmd.Stdout = os.Stdout
		retryCmd.Stderr = os.Stderr

		err = retryCmd.Run()
		if err == nil {
			return nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode == 137 || exitCode == 143 {
				return fmt.Errorf("exec timed out after %v", e.timeout)
			}
			return fmt.Errorf("script exited with code %d", exitCode)
		}

		if attempt < maxAttempts-1 {
			logger.Warn(fmt.Sprintf("Attempt %d failed (connection error), retrying in 2 seconds...", attempt+1))
			time.Sleep(2 * time.Second)
			continue
		}
		return fmt.Errorf("exec failed after %d attempts: %w", maxAttempts, err)
	}

	return nil
}

func (e *ExecProvisioner) buildSystemdRunCommand(user string, script string, timeout time.Duration) (*exec.Cmd, error) {
	envPrefix := "export DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket\nexport XDG_RUNTIME_DIR=/run/user/0\n"
	fullScript := envPrefix + script

	timeoutSec := int(timeout.Seconds())
	args := []string{
		"systemd-run", "-M", e.machineName, "--pipe", "--wait",
		fmt.Sprintf("--property=RuntimeMaxSec=%d", timeoutSec),
		"--", "/bin/bash", "-s",
	}
	cmd := deploy.Sudo(args...)
	cmd.Stdin = strings.NewReader(fullScript)
	return cmd, nil
}

func buildEnvExport(params map[string]string) string {
	lines := []string{"export DEBIAN_FRONTEND=noninteractive"}
	for k, v := range params {
		if !isValidEnvVarName(k) {
			continue
		}
		escaped := strings.ReplaceAll(v, "'", "'\"'\"'")
		lines = append(lines, fmt.Sprintf("export LXSPAWN_PARAM_%s='%s'", k, escaped))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// isValidEnvVarName checks if a string is a valid shell environment variable name
func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		if i == 0 {
			// First character must be letter or underscore
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
				return false
			}
		} else {
			// Subsequent characters can be letters, digits, or underscores
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				return false
			}
		}
	}
	return true
}

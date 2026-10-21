package provision

import (
	"context"
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
	if prov.File != nil {
		resolvedPath, err := fileHandler.ResolveFileRef(prov.File.URL, prov.File.Digest, baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve file: %w", err)
		}
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		script = string(data)
	}

	rendered, err := RenderTemplate(script, ctx)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	envExport := buildEnvExport(ctx.Param)

	logger.Info(fmt.Sprintf("Executing %s provision script", prov.Mode))

	maxAttempts := 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var retryCmd *exec.Cmd
		switch prov.Mode {
		case config.ProvisionSystem:
			fullScript := fmt.Sprintf("%s\n%s", envExport, rendered)
			retryCmd, err = e.buildNsenterCommand("root", fullScript)
			if err != nil {
				return fmt.Errorf("failed to build nsenter command: %w", err)
			}
		case config.ProvisionUser:
			user := ctx.User
			escapedScript := strings.ReplaceAll(rendered, "'", "'\"'\"'")
			fullScript := fmt.Sprintf("%s\nsu - %s -c '%s'", envExport, user, escapedScript)
			retryCmd, err = e.buildNsenterCommand("root", fullScript)
			if err != nil {
				return fmt.Errorf("failed to build nsenter command: %w", err)
			}
		default:
			return fmt.Errorf("unsupported provision mode for exec: %s", prov.Mode)
		}
		retryCmd.Stdout = os.Stdout
		retryCmd.Stderr = os.Stderr

		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		err := retryCmd.Start()
		if err != nil {
			cancel()
			if attempt < maxAttempts-1 {
				logger.Warn(fmt.Sprintf("Attempt %d failed to start, retrying in 2 seconds...", attempt+1))
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("exec failed after %d attempts: %w", maxAttempts, err)
		}

		done := make(chan error, 1)
		go func() { done <- retryCmd.Wait() }()

		select {
		case <-ctx.Done():
			cancel()
			if retryCmd.Process != nil {
				retryCmd.Process.Kill()
			}
			return fmt.Errorf("exec timed out after %v", e.timeout)
		case err := <-done:
			cancel()
			if err != nil {
				if attempt < maxAttempts-1 {
					logger.Warn(fmt.Sprintf("Attempt %d failed, retrying in 2 seconds...", attempt+1))
					time.Sleep(2 * time.Second)
					continue
				}
				return fmt.Errorf("exec failed after %d attempts: %w", maxAttempts, err)
			}
		}
		break
	}

	return nil
}

func (e *ExecProvisioner) buildNsenterCommand(user string, script string) (*exec.Cmd, error) {
	leaderPID, err := e.getLeaderPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get leader PID: %w", err)
	}

	// Set environment variables for D-Bus and systemd
	envScript := fmt.Sprintf("export DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket\nexport XDG_RUNTIME_DIR=/run/user/0\n%s", script)
	
	args := []string{"nsenter", "-t", strconv.Itoa(leaderPID), "-m", "-u", "-i", "-n", "-p", "--", "/bin/bash", "-c", envScript}
	return deploy.SudoHelper(args...), nil
}

func (e *ExecProvisioner) getLeaderPID() (int, error) {
	systemd, err := deploy.NewSystemdManager()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer systemd.Close()

	state, _, err := systemd.GetMachineState(e.machineName)
	if err != nil {
		return 0, fmt.Errorf("failed to get machine state: %w", err)
	}

	if state != "running" {
		return 0, fmt.Errorf("machine is not running")
	}

	leader, err := systemd.GetMachineLeader(e.machineName)
	if err != nil {
		return 0, fmt.Errorf("failed to get leader PID: %w", err)
	}

	return leader, nil
}

func buildEnvExport(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	var lines []string
	for k, v := range params {
		escaped := strings.ReplaceAll(v, "'", "'\"'\"'")
		lines = append(lines, fmt.Sprintf("export LXSPAWN_PARAM_%s='%s'", k, escaped))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

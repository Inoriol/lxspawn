package machine

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	lxerrors "github.com/Inoriol/lxspawn/internal/errors"
)

func (m *Manager) Shell(name string) error {
	if _, exists := m.state.GetMachine(name); !exists {
		return lxerrors.NewMachineNotFoundError(name)
	}

	cmd := exec.Command("machinectl", "shell", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) Exec(name string, command []string) error {
	if _, exists := m.state.GetMachine(name); !exists {
		return lxerrors.NewMachineNotFoundError(name)
	}

	shellCmd := shellQuoteArgs(command)
	envScript := fmt.Sprintf("export DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket\nexport XDG_RUNTIME_DIR=/run/user/0\n%s", shellCmd)

	args := []string{"-n", "systemd-run", "-M", name, "--pipe", "--wait", "--", "/bin/bash", "-c", envScript}
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	needsQuoting := false
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '/' || c == '.' || c == '=' || c == '@' || c == '+' || c == ',') {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func (m *Manager) Logs(name string) error {
	if _, exists := m.state.GetMachine(name); !exists {
		return lxerrors.NewMachineNotFoundError(name)
	}

	cmd := exec.Command("journalctl", "-M", name, "-f")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type LogOptions struct {
	Lines    int
	Since    string
	Priority string
	Follow   bool
}

func (m *Manager) LogsWithOptions(name string, opts LogOptions) error {
	if _, exists := m.state.GetMachine(name); !exists {
		return lxerrors.NewMachineNotFoundError(name)
	}

	args := []string{"-M", name}

	if opts.Follow {
		args = append(args, "-f")
	} else {
		args = append(args, "--no-pager")
		if opts.Lines > 0 {
			args = append(args, "-n", fmt.Sprintf("%d", opts.Lines))
		}
	}

	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	if opts.Priority != "" {
		args = append(args, "-p", opts.Priority)
	}

	args = append(args, "-o", "short-iso")

	cmd := exec.Command("journalctl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type LogEntry struct {
	Timestamp string
	Priority  int
	Message   string
}

func (m *Manager) GetLogs(name string, lines int) ([]LogEntry, error) {
	return m.GetLogsWithOptions(name, LogOptions{Lines: lines})
}

func (m *Manager) GetLogsWithOptions(name string, opts LogOptions) ([]LogEntry, error) {
	if _, exists := m.state.GetMachine(name); !exists {
		return nil, lxerrors.NewMachineNotFoundError(name)
	}

	args := []string{"-M", name, "--no-pager", "-o", "short-iso"}

	if opts.Lines > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", opts.Lines))
	}

	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	if opts.Priority != "" {
		args = append(args, "-p", opts.Priority)
	}

	cmd := exec.Command("journalctl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var entries []LogEntry
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}

		entry := LogEntry{
			Timestamp: parts[0],
			Priority:  6,
			Message:   parts[2],
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

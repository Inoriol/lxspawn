package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var autostartCmd = &cobra.Command{
	Use:       "autostart <name> <on|off>",
	Short:     "Enable or disable autostart for a machine",
	Args:      cobra.ExactArgs(2),
	ValidArgs: []string{"on", "off"},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		action := args[1]

		if action != "on" && action != "off" {
			return fmt.Errorf("action must be 'on' or 'off', got '%s'", action)
		}

		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if _, exists := mgr.GetMachineInfo(name); !exists {
			return fmt.Errorf("machine '%s' not found", name)
		}

		unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)

		var systemctlArgs []string
		if action == "on" {
			systemctlArgs = []string{"systemctl", "enable", unitName}
		} else {
			systemctlArgs = []string{"systemctl", "disable", unitName}
		}

		enableCmd := deploy.Sudo(systemctlArgs...)
		if output, err := enableCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to %s autostart: %w (%s)", action, err, string(output))
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.AutostartData{
				Machine:   name,
				Autostart: action == "on",
				Success:   true,
			})
		}

		if action == "on" {
			fmt.Printf("Autostart enabled for '%s'\n", name)
		} else {
			fmt.Printf("Autostart disabled for '%s'\n", name)
		}

		return nil
	},
}

func isAutostartEnabled(name string) (bool, error) {
	unitName := fmt.Sprintf("systemd-nspawn@%s.service", name)
	cmd := exec.Command("systemctl", "is-enabled", unitName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}
	return string(output) == "enabled\n", nil
}

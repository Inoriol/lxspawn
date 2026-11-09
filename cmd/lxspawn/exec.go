package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
)

var execCmd = &cobra.Command{
	Use:   "exec <name> -- <command> [args...]",
	Short: "Run a command inside a machine",
	Args:  cobra.MinimumNArgs(2),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find the -- separator
		sepIdx := -1
		for i, arg := range args {
			if arg == "--" {
				sepIdx = i
				break
			}
		}

		var machineName string
		var command []string

		if sepIdx == -1 {
			// No -- separator, assume first arg is machine name
			machineName = args[0]
			command = args[1:]
		} else {
			machineName = args[0]
			command = args[sepIdx+1:]
		}

		if len(command) == 0 {
			return fmt.Errorf("command is required")
		}

		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		return mgr.Exec(machineName, command)
	},
}

package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start an existing machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if err := mgr.Start(args[0]); err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "start",
				Machine: args[0],
				Success: true,
			})
		}

		return nil
	},
}

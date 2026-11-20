package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var stopForce bool

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a running machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if stopForce {
			if err := mgr.ForceStop(args[0]); err != nil {
				return err
			}
		} else {
			if err := mgr.Stop(args[0]); err != nil {
				return err
			}
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "stop",
				Machine: args[0],
				Success: true,
			})
		}

		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "Force stop (poweroff, then terminate, then SIGKILL)")
}

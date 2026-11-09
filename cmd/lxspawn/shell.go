package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
)

var shellCmd = &cobra.Command{
	Use:   "shell <name>",
	Short: "Open an interactive shell in a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		return mgr.Shell(args[0])
	},
}

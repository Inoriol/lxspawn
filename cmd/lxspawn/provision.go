package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var provisionCmd = &cobra.Command{
	Use:   "provision <name>",
	Short: "Re-run provisioning on an existing machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		name := args[0]
		info, exists := mgr.GetMachineInfo(name)
		if !exists {
			return fmt.Errorf("machine '%s' not found", name)
		}

		cfg, err := machine.LoadMachineConfig(info)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := mgr.Provision(cfg, info.ConfigPath); err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "provision",
				Machine: name,
				Success: true,
			})
		}

		return nil
	},
}

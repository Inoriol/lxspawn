package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var launchCmd = &cobra.Command{
	Use:   "launch <yaml>",
	Short: "Create and start a machine from a YAML config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse(args[0])
		if err != nil {
			return err
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		sourcePath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("failed to resolve source path: %w", err)
		}

		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if err := mgr.Launch(cfg, sourcePath); err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "launch",
				Machine: cfg.Name,
				Success: true,
			})
		}

		return nil
	},
}

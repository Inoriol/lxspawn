package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var createCmd = &cobra.Command{
	Use:   "create <yaml>",
	Short: "Create a machine from a YAML config (don't start)",
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

		filesWritten, err := mgr.Create(cfg, sourcePath)
		if err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.CreateData{
				Machine:      cfg.Name,
				SourcePath:   sourcePath,
				FilesWritten: filesWritten,
				Success:      true,
			})
		}

		return nil
	},
}

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open machine config in editor and re-apply changes",
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

		if info.SourcePath == "" {
			return fmt.Errorf("machine '%s' has no source path recorded", name)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		editorCmd := exec.Command(editor, info.SourcePath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		cfg, err := config.Parse(info.SourcePath)
		if err != nil {
			return fmt.Errorf("failed to parse updated config: %w", err)
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("invalid config after edit: %w", err)
		}

		if cfg.Name != name {
			return fmt.Errorf("machine name changed from '%s' to '%s'; use 'lxspawn delete' and 'lxspawn launch' instead", name, cfg.Name)
		}

		status, err := mgr.GetStatus(name)
		if err != nil {
			return fmt.Errorf("failed to get machine status: %w", err)
		}
		wasRunning := status.State == "running"

		if wasRunning {
			logger.Info("Stopping machine to apply changes...")
			if err := mgr.Stop(name); err != nil {
				return fmt.Errorf("failed to stop machine: %w", err)
			}
		}

		if err := mgr.Apply(cfg, info.SourcePath); err != nil {
			return fmt.Errorf("failed to apply config changes: %w", err)
		}

		if wasRunning {
			logger.Info("Restarting machine...")
			if err := mgr.Start(name); err != nil {
				return fmt.Errorf("failed to restart machine: %w", err)
			}
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "edit",
				Machine: name,
				Success: true,
			})
		}

		return nil
	},
}

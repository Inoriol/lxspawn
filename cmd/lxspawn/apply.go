package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var applyDryRun bool

var applyCmd = &cobra.Command{
	Use:   "apply <name> [file]",
	Short: "Apply configuration from file or stdin",
	Long: `Apply configuration changes to an existing machine without recreating it.
The rootfs is preserved. If the machine is running, a restart is required.

Examples:
  lxspawn apply mymachine config.yaml
  cat config.yaml | lxspawn apply mymachine -
  lxspawn apply mymachine - < config.yaml`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		name := args[0]

		if _, exists := mgr.GetMachineInfo(name); !exists {
			return fmt.Errorf("machine '%s' not found", name)
		}

		var configData []byte
		var sourcePath string

		if len(args) == 2 && args[1] != "-" {
			sourcePath = args[1]
			configData, err = os.ReadFile(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
		} else {
			configData, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			sourcePath = "<stdin>"
		}

		cfg, err := config.ParseString(string(configData))
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		if cfg.Name != name {
			return fmt.Errorf("config name '%s' does not match machine name '%s'", cfg.Name, name)
		}

		if applyDryRun {
			if output.IsJSONMode() {
				return output.PrintJSON(&output.ApplyData{
					Machine:        name,
					Success:        true,
					DryRun:         true,
					RestartRequired: false,
				})
			}
			fmt.Println("Configuration is valid (dry run)")
			return nil
		}

		status, err := mgr.GetStatus(name)
		if err != nil {
			return fmt.Errorf("failed to get machine status: %w", err)
		}
		isRunning := status.State == "running"

		if err := mgr.Apply(cfg, sourcePath); err != nil {
			return fmt.Errorf("failed to apply config: %w", err)
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.ApplyData{
				Machine:        name,
				Success:        true,
				DryRun:         false,
				RestartRequired: isRunning,
			})
		}

		fmt.Printf("Configuration applied to '%s'\n", name)
		if isRunning {
			fmt.Println("Restart required to apply changes. Run: lxspawn restart " + name)
		}

		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Validate config without applying")
}

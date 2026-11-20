package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/progress"
)

var (
	launchTemplate string
	launchName     string
	launchParams   []string
	launchProgress bool
)

var launchCmd = &cobra.Command{
	Use:   "launch [yaml]",
	Short: "Create and start a machine from a YAML config",
	Long: `Create and start a machine from a YAML config file, stdin, or a template.

Examples:
  # From file
  lxspawn launch myserver.yaml

  # From stdin
  cat myserver.yaml | lxspawn launch -

  # From template
  lxspawn launch --template debian-base --name myserver

  # With parameter overrides
  lxspawn launch --template pihole --name dns --param ip=192.168.1.100

  # With progress events
  lxspawn launch --template debian-base --name myserver --progress`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := ""
		if len(args) > 0 {
			source = args[0]
		}

		if source == "" && launchTemplate == "" {
			return cmd.Help()
		}

		cfg, sourcePath, err := loadConfig(source, launchTemplate, launchName, launchParams)
		if err != nil {
			return err
		}

		if source != "-" && source != "" {
			absPath, err := filepath.Abs(source)
			if err == nil {
				sourcePath = absPath
			}
		}

		var reporter *progress.Reporter
		if launchProgress {
			reporter = progress.NewReporter(os.Stdout, true)
			reporter.Emit(progress.EventProvisioning, "Creating machine", "create", 0)
		}

		mgr, err := machine.NewManager()
		if err != nil {
			if reporter != nil {
				reporter.EmitError(err)
			}
			return err
		}
		defer mgr.Close()

		if reporter != nil {
			reporter.Emit(progress.EventStarting, "Starting machine", "start", 0)
		}

		if err := mgr.Launch(cfg, sourcePath); err != nil {
			if reporter != nil {
				reporter.EmitError(err)
			}
			return err
		}

		if reporter != nil {
			reporter.EmitDone("Machine launched successfully")
		}

		if output.IsJSONMode() && !launchProgress {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "launch",
				Machine: cfg.Name,
				Success: true,
			})
		}

		return nil
	},
}

func init() {
	launchCmd.Flags().StringVar(&launchTemplate, "template", "", "Use embedded template")
	launchCmd.Flags().StringVar(&launchName, "name", "", "Override machine name")
	launchCmd.Flags().StringArrayVar(&launchParams, "param", nil, "Override template params (key=value, repeatable)")
	launchCmd.Flags().BoolVar(&launchProgress, "progress", false, "Emit progress events as newline-delimited JSON")
}

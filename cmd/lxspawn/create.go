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
	createTemplate string
	createName     string
	createParams   []string
	createProgress bool
)

var createCmd = &cobra.Command{
	Use:   "create [yaml]",
	Short: "Create a machine from a YAML config (don't start)",
	Long: `Create a machine from a YAML config file, stdin, or a template.

Examples:
  # From file
  lxspawn create myserver.yaml

  # From stdin
  cat myserver.yaml | lxspawn create -

  # From template
  lxspawn create --template debian-base --name myserver

  # With parameter overrides
  lxspawn create --template pihole --name dns --param ip=192.168.1.100

  # With progress events
  lxspawn create --template debian-base --name myserver --progress`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := ""
		if len(args) > 0 {
			source = args[0]
		}

		if source == "" && createTemplate == "" {
			return cmd.Help()
		}

		cfg, sourcePath, err := loadConfig(source, createTemplate, createName, createParams)
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
		if createProgress {
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

		filesWritten, err := mgr.Create(cfg, sourcePath)
		if err != nil {
			if reporter != nil {
				reporter.EmitError(err)
			}
			return err
		}

		if reporter != nil {
			reporter.EmitDone("Machine created successfully")
		}

		if output.IsJSONMode() && !createProgress {
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

func init() {
	createCmd.Flags().StringVar(&createTemplate, "template", "", "Use embedded template")
	createCmd.Flags().StringVar(&createName, "name", "", "Override machine name")
	createCmd.Flags().StringArrayVar(&createParams, "param", nil, "Override template params (key=value, repeatable)")
	createCmd.Flags().BoolVar(&createProgress, "progress", false, "Emit progress events as newline-delimited JSON")
}

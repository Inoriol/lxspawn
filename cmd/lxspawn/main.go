package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/output"
)

var version = "dev"

var quiet bool
var verbose bool
var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "lxspawn",
	Short: "LXC-style system containers powered by systemd-nspawn",
	Long: `lxspawn is a daemonless, lightweight "Light VM" orchestrator for
immutable desktop OSs and server homelabs. It fills the gap between
Podman (app containers) and KVM/QEMU (full VMs).`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.SetQuiet(quiet)
		logger.SetVerbose(verbose)
		output.SetJSONMode(jsonOutput)
	},
}

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
	rootCmd.Version = version
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all non-error output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
}

func main() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(provisionCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(listSnapshotsCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(deleteSnapshotCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(provisionStatusCmd)
	rootCmd.AddCommand(listDistrosCmd)
	rootCmd.AddCommand(editResourcesCmd)
	rootCmd.AddCommand(editNetworkCmd)
	rootCmd.AddCommand(addVolumeCmd)
	rootCmd.AddCommand(editVolumeCmd)
	rootCmd.AddCommand(removeVolumeCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(listTemplatesCmd)

	if err := rootCmd.Execute(); err != nil {
		if output.IsJSONMode() {
			output.PrintError("COMMAND_FAILED", err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

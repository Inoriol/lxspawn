package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var logsLines int

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Follow logs from a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if output.IsJSONMode() {
			entries, err := mgr.GetLogs(args[0], logsLines)
			if err != nil {
				return err
			}

			jsonEntries := make([]output.LogEntryJSON, len(entries))
			for i, e := range entries {
				jsonEntries[i] = output.LogEntryJSON{
					Timestamp: e.Timestamp,
					Priority:  e.Priority,
					Message:   e.Message,
				}
			}

			return output.PrintJSON(&output.LogsData{
				Machine: args[0],
				Entries: jsonEntries,
			})
		}

		return mgr.Logs(args[0])
	},
}

func init() {
	logsCmd.Flags().IntVar(&logsLines, "lines", 100, "Number of log lines to show (JSON mode only)")
}

package main

import (
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var (
	logsLines    int
	logsSince    string
	logsPriority string
	logsFollow   bool
)

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

		opts := machine.LogOptions{
			Lines:    logsLines,
			Since:    logsSince,
			Priority: logsPriority,
			Follow:   logsFollow,
		}

		if output.IsJSONMode() {
			entries, err := mgr.GetLogsWithOptions(args[0], opts)
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

		return mgr.LogsWithOptions(args[0], opts)
	},
}

func init() {
	logsCmd.Flags().IntVar(&logsLines, "lines", 100, "Number of log lines to show")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since date (e.g., '1h ago', '2024-01-01', 'yesterday')")
	logsCmd.Flags().StringVar(&logsPriority, "priority", "", "Filter by priority (emerg, alert, crit, err, warning, notice, info, debug)")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
}

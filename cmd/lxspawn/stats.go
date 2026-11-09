package main

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var statsCmd = &cobra.Command{
	Use:   "stats <name>",
	Short: "Show runtime statistics of a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		stats, err := mgr.GetStats(args[0])
		if err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.StatsData{
				Name:              stats.Name,
				State:             stats.State,
				CPUUsagePercent:   stats.CPUUsagePercent,
				MemoryUsedBytes:   stats.MemoryUsedBytes,
				MemoryTotalBytes:  stats.MemoryTotalBytes,
				MemoryUsagePercent: stats.MemoryUsagePercent,
				DiskUsedBytes:     stats.DiskUsedBytes,
				DiskTotalBytes:    stats.DiskTotalBytes,
			})
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendRow(table.Row{"Name", stats.Name})
		t.AppendRow(table.Row{"State", stats.State})

		if stats.State == "running" {
			t.AppendRow(table.Row{"CPU Usage", fmt.Sprintf("%.2f%%", stats.CPUUsagePercent)})
			t.AppendRow(table.Row{"Memory", fmt.Sprintf("%s / %s (%.1f%%)",
				formatBytes(stats.MemoryUsedBytes),
				formatBytes(stats.MemoryTotalBytes),
				stats.MemoryUsagePercent)})
			if stats.DiskTotalBytes > 0 {
				t.AppendRow(table.Row{"Disk", fmt.Sprintf("%s / %s",
					formatBytes(stats.DiskUsedBytes),
					formatBytes(stats.DiskTotalBytes))})
			}
		}

		t.Render()
		return nil
	},
}

func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

package main

import (
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var statsAll bool

var statsCmd = &cobra.Command{
	Use:   "stats [name]",
	Short: "Show runtime statistics of a machine",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if statsAll {
			return showAllStats(mgr, cmd)
		}

		if len(args) == 0 {
			return fmt.Errorf("machine name required (or use --all)")
		}

		stats, err := mgr.GetStats(args[0])
		if err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.StatsData{
				Name:               stats.Name,
				State:              stats.State,
				CPUUsagePercent:    stats.CPUUsagePercent,
				CPUUsageUsec:       stats.CPUUsageUsec,
				MemoryUsedBytes:    stats.MemoryUsedBytes,
				MemoryTotalBytes:   stats.MemoryTotalBytes,
				MemoryUsagePercent: stats.MemoryUsagePercent,
				DiskUsedBytes:      stats.DiskUsedBytes,
				DiskTotalBytes:     stats.DiskTotalBytes,
				Timestamp:          time.Now().UTC().Format(time.RFC3339),
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

func showAllStats(mgr *machine.Manager, cmd *cobra.Command) error {
	statsList, err := mgr.GetAllStats()
	if err != nil {
		return err
	}

	if output.IsJSONMode() {
		jsonStats := make([]output.StatsData, len(statsList))
		timestamp := time.Now().UTC().Format(time.RFC3339)
		for i, s := range statsList {
			jsonStats[i] = output.StatsData{
				Name:               s.Name,
				State:              s.State,
				CPUUsagePercent:    s.CPUUsagePercent,
				CPUUsageUsec:       s.CPUUsageUsec,
				MemoryUsedBytes:    s.MemoryUsedBytes,
				MemoryTotalBytes:   s.MemoryTotalBytes,
				MemoryUsagePercent: s.MemoryUsagePercent,
				DiskUsedBytes:      s.DiskUsedBytes,
				DiskTotalBytes:     s.DiskTotalBytes,
				Timestamp:          timestamp,
			}
		}
		return output.PrintJSON(&output.AllStatsData{
			Timestamp: timestamp,
			Machines:  jsonStats,
		})
	}

	t := table.NewWriter()
	t.SetOutputMirror(cmd.OutOrStdout())
	t.AppendHeader(table.Row{"Name", "State", "CPU %", "Memory", "Disk"})

	for _, stats := range statsList {
		cpuStr := "-"
		memStr := "-"
		diskStr := "-"

		if stats.State == "running" {
			cpuStr = fmt.Sprintf("%.1f%%", stats.CPUUsagePercent)
			memStr = fmt.Sprintf("%s/%s",
				formatBytes(stats.MemoryUsedBytes),
				formatBytes(stats.MemoryTotalBytes))
			if stats.DiskTotalBytes > 0 {
				diskStr = fmt.Sprintf("%s/%s",
					formatBytes(stats.DiskUsedBytes),
					formatBytes(stats.DiskTotalBytes))
			}
		}

		t.AppendRow(table.Row{stats.Name, stats.State, cpuStr, memStr, diskStr})
	}

	t.Render()
	return nil
}

func init() {
	statsCmd.Flags().BoolVar(&statsAll, "all", false, "Show stats for all machines")
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

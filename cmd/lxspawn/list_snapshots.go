package main

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/snapshot"
)

var listSnapshotsCmd = &cobra.Command{
	Use:   "list-snapshots <machine>",
	Short: "List snapshots for a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineName := args[0]

		mgr, err := snapshot.NewManager(machineName)
		if err != nil {
			return err
		}
		defer mgr.Close()

		snapshots, err := mgr.ListSnapshots()
		if err != nil {
			return err
		}

		if output.IsJSONMode() {
			var jsonSnapshots []output.SnapshotJSON
			for _, s := range snapshots {
				jsonSnapshots = append(jsonSnapshots, output.SnapshotJSON{
					Name:      s.Name,
					Machine:   s.Machine,
					Backend:   s.Backend,
					CreatedAt: s.CreatedAt,
					Path:      s.Path,
					Dataset:   s.Dataset,
					LV:        s.LV,
				})
			}
			return output.PrintJSON(&output.SnapshotListData{
				Machine:   machineName,
				Backend:   string(mgr.BackendType()),
				Snapshots: jsonSnapshots,
			})
		}

		if len(snapshots) == 0 {
			fmt.Printf("No snapshots found for machine '%s'.\n", machineName)
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendHeader(table.Row{"NAME", "BACKEND", "CREATED", "LOCATION"})

		for _, s := range snapshots {
			location := s.Path
			if s.Dataset != "" {
				location = s.Dataset
			}
			if s.LV != "" {
				location = s.LV
			}
			t.AppendRow(table.Row{s.Name, s.Backend, s.CreatedAt, location})
		}

		t.Render()
		return nil
	},
}

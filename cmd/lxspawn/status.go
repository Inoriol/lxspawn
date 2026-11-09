package main

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show detailed status of a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		status, err := mgr.GetStatus(args[0])
		if err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.MachineStatusData{
				Name:       status.Name,
				Distro:     status.Distro,
				State:      status.State,
				Leader:     status.Leader,
				Addresses:  status.Addresses,
				Network:    status.Network,
				CreatedAt:  status.CreatedAt,
				SourcePath: status.SourcePath,
				Files:      status.Files,
				Ephemeral:  status.Ephemeral,
				ReadOnly:   status.ReadOnly,
			})
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendRow(table.Row{"Name", status.Name})
		t.AppendRow(table.Row{"Distro", status.Distro})
		t.AppendRow(table.Row{"State", status.State})
		if status.Leader > 0 {
			t.AppendRow(table.Row{"Leader PID", status.Leader})
		}
		if len(status.Addresses) > 0 {
			for i, addr := range status.Addresses {
				if i == 0 {
					t.AppendRow(table.Row{"Addresses", addr})
				} else {
					t.AppendRow(table.Row{"", addr})
				}
			}
		}
		if status.Network != "" {
			t.AppendRow(table.Row{"Network", status.Network})
		}
		if status.CreatedAt != "" {
			t.AppendRow(table.Row{"Created", status.CreatedAt})
		}
		if status.SourcePath != "" {
			t.AppendRow(table.Row{"Source", status.SourcePath})
		}

		t.Render()
		return nil
	},
}

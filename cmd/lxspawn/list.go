package main

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all machines",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		machines, err := mgr.ListMachinesWithState()
		if err != nil {
			return err
		}

		if output.IsJSONMode() {
			var jsonMachines []output.MachineJSON
			for _, m := range machines {
				jsonMachines = append(jsonMachines, output.MachineJSON{
					Name:       m.Name,
					Distro:     m.Distro,
					State:      m.State,
					Network:    m.Network,
					Addresses:  m.Addresses,
					CreatedAt:  m.CreatedAt,
					SourcePath: m.SourcePath,
					Resources: &output.ResourceJSON{
						CPUs:   m.CPUs,
						Memory: m.Memory,
						Swap:   m.Swap,
						Tasks:  m.Tasks,
					},
				})
			}
			return output.FormatMachineList(jsonMachines)
		}

		if len(machines) == 0 {
			fmt.Println("No machines found.")
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendHeader(table.Row{"NAME", "STATE", "DISTRO", "NETWORK", "CREATED"})

		for _, m := range machines {
			t.AppendRow(table.Row{m.Name, m.State, m.Distro, m.Network, m.CreatedAt})
		}

		t.Render()
		return nil
	},
}

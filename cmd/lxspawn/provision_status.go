package main

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/provision"
)

var provisionStatusCmd = &cobra.Command{
	Use:   "provision-status <name>",
	Short: "Show provisioning status of a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if _, exists := mgr.GetMachineInfo(args[0]); !exists {
			return fmt.Errorf("machine '%s' not found", args[0])
		}

		engine, err := provision.NewEngine(args[0])
		if err != nil {
			return fmt.Errorf("failed to create provision engine: %w", err)
		}

		state, err := engine.GetState()
		if err != nil {
			if output.IsJSONMode() {
				return output.PrintJSON(&output.ProvisionStatusData{
					Name: args[0],
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "No provisioning state found for machine '%s'\n", args[0])
			return nil
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.ProvisionStatusData{
				Name:           args[0],
				LastRun:        state.LastRun,
				PhasesComplete: state.PhasesComplete,
				ProbesPassed:   state.ProbesPassed,
				ScriptHashes:   state.ScriptHashes,
			})
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendRow(table.Row{"Machine", args[0]})

		if state.LastRun != "" {
			t.AppendRow(table.Row{"Last Run", state.LastRun})
		}

		if len(state.PhasesComplete) > 0 {
			for i, phase := range state.PhasesComplete {
				if i == 0 {
					t.AppendRow(table.Row{"Phases Completed", phase})
				} else {
					t.AppendRow(table.Row{"", phase})
				}
			}
		}

		if len(state.ProbesPassed) > 0 {
			for i, probe := range state.ProbesPassed {
				if i == 0 {
					t.AppendRow(table.Row{"Probes Passed", probe})
				} else {
					t.AppendRow(table.Row{"", probe})
				}
			}
		}

		if len(state.ScriptHashes) > 0 {
			first := true
			for key, hash := range state.ScriptHashes {
				if first {
					t.AppendRow(table.Row{"Script Hashes", fmt.Sprintf("%s: %s", key, hash)})
					first = false
				} else {
					t.AppendRow(table.Row{"", fmt.Sprintf("%s: %s", key, hash)})
				}
			}
		}

		t.Render()
		return nil
	},
}

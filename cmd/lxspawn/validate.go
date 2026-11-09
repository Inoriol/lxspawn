package main

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/output"
)

var validateCmd = &cobra.Command{
	Use:   "validate <yaml>",
	Short: "Validate a YAML configuration file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Parse(args[0])
		if err != nil {
			if output.IsJSONMode() {
				return output.PrintJSON(map[string]interface{}{
					"valid":  false,
					"errors": []string{err.Error()},
				})
			}
			return fmt.Errorf("validation failed: %w", err)
		}

		if output.IsJSONMode() {
			return output.PrintJSON(map[string]interface{}{
				"valid":    true,
				"errors":   []string{},
				"warnings": []string{},
				"name":     cfg.Name,
				"distro":   cfg.Distro,
			})
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendRow(table.Row{"Valid", "yes"})
		t.AppendRow(table.Row{"Name", cfg.Name})
		t.AppendRow(table.Row{"Distro", cfg.Distro})
		t.Render()

		return nil
	},
}

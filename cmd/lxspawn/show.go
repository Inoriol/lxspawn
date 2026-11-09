package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Display generated configuration files for a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		name := args[0]
		if _, exists := mgr.GetMachineInfo(name); !exists {
			return fmt.Errorf("machine '%s' not found", name)
		}

		nspawnPath := deploy.NspawnFilePath(name)
		slicePath := deploy.SliceFilePath(name)
		dropInPath := deploy.DropInFilePath(name)
		hooksDropInPath := deploy.HooksDropInFilePath(name)

		files := map[string]string{
			"nspawn":     nspawnPath,
			"slice":      slicePath,
			"drop-in":    dropInPath,
			"hooks":      hooksDropInPath,
		}

		if output.IsJSONMode() {
			result := make(map[string]string)
			for label, path := range files {
				content, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				result[label] = string(content)
			}
			return output.PrintJSON(result)
		}

		for label, path := range files {
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			fmt.Printf("=== %s (%s) ===\n", label, path)
			fmt.Println(string(content))
			fmt.Println()
		}

		return nil
	},
}

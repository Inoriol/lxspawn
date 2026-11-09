package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var deleteYes bool

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a machine and all its data",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !deleteYes && isStdinTerminal() {
			fmt.Printf("Are you sure you want to delete machine '%s'? [y/N]: ", args[0])
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("aborted")
			}
		}

		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		if err := mgr.Delete(args[0]); err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "delete",
				Machine: args[0],
				Success: true,
			})
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation prompt")
}

func isStdinTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

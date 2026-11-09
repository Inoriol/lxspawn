package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/snapshot"
)

var deleteSnapshotYes bool

var deleteSnapshotCmd = &cobra.Command{
	Use:   "delete-snapshot <machine> <snapshot-name>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineName := args[0]
		snapName := args[1]

		if !deleteSnapshotYes && isStdinTerminal() {
			fmt.Printf("Are you sure you want to delete snapshot '%s' from machine '%s'? [y/N]: ", snapName, machineName)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("aborted")
			}
		}

		mgr, err := snapshot.NewManager(machineName)
		if err != nil {
			return err
		}
		defer mgr.Close()

		if err := mgr.DeleteSnapshot(snapName); err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.SnapshotData{
				Action:   "delete",
				Machine:  machineName,
				Snapshot: snapName,
				Backend:  string(mgr.BackendType()),
				Success:  true,
			})
		}

		fmt.Printf("Snapshot '%s' deleted from machine '%s'\n", snapName, machineName)
		return nil
	},
}

func init() {
	deleteSnapshotCmd.Flags().BoolVarP(&deleteSnapshotYes, "yes", "y", false, "Skip confirmation prompt")
}

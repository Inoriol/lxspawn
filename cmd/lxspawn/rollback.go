package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/snapshot"
)

var rollbackYes bool

var rollbackCmd = &cobra.Command{
	Use:   "rollback <machine> <snapshot-name>",
	Short: "Rollback a machine to a snapshot",
	Long:  "Rollback a machine to a previous snapshot. WARNING: This is destructive and will discard all changes since the snapshot was taken. The machine must be stopped.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineName := args[0]
		snapName := args[1]

		machineMgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer machineMgr.Close()

		if _, exists := machineMgr.GetMachineInfo(machineName); !exists {
			return fmt.Errorf("machine '%s' not found", machineName)
		}

		status, err := machineMgr.GetStatus(machineName)
		if err != nil {
			return fmt.Errorf("failed to get machine status: %w", err)
		}
		if status.State == "running" {
			return fmt.Errorf("machine '%s' is running; stop it before rolling back", machineName)
		}

		if !rollbackYes && isStdinTerminal() {
			fmt.Printf("WARNING: This will discard all changes to machine '%s' since snapshot '%s' was taken.\n", machineName, snapName)
			fmt.Printf("Are you sure you want to rollback? [y/N]: ")
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

		if err := mgr.RollbackSnapshot(snapName); err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.SnapshotData{
				Action:   "rollback",
				Machine:  machineName,
				Snapshot: snapName,
				Backend:  string(mgr.BackendType()),
				Success:  true,
			})
		}

		fmt.Printf("Machine '%s' rolled back to snapshot '%s' (backend: %s)\n", machineName, snapName, mgr.BackendType())
		return nil
	},
}

func init() {
	rollbackCmd.Flags().BoolVarP(&rollbackYes, "yes", "y", false, "Skip confirmation prompt")
}

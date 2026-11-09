package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/snapshot"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot <machine> <snapshot-name>",
	Short: "Create a snapshot of a machine",
	Long:  "Create a snapshot of a machine's rootfs. The machine must be stopped. Requires btrfs, zfs, or LVM.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineName := args[0]
		snapName := args[1]

		mgr, err := snapshot.NewManager(machineName)
		if err != nil {
			return err
		}
		defer mgr.Close()

		if err := mgr.CreateSnapshot(snapName); err != nil {
			return err
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.SnapshotData{
				Action:   "create",
				Machine:  machineName,
				Snapshot: snapName,
				Backend:  string(mgr.BackendType()),
				Success:  true,
			})
		}

		fmt.Printf("Snapshot '%s' created for machine '%s' (backend: %s)\n", snapName, machineName, mgr.BackendType())
		return nil
	},
}

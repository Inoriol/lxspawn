package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open machine config in editor and re-apply changes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		name := args[0]
		info, exists := mgr.GetMachineInfo(name)
		if !exists {
			return fmt.Errorf("machine '%s' not found", name)
		}

		if info.SourcePath == "" {
			return fmt.Errorf("machine '%s' has no source path recorded", name)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		editorCmd := exec.Command(editor, info.SourcePath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		cfg, err := config.Parse(info.SourcePath)
		if err != nil {
			return fmt.Errorf("failed to parse updated config: %w", err)
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("invalid config after edit: %w", err)
		}

		if cfg.Name != name {
			return fmt.Errorf("machine name changed from '%s' to '%s'; use 'lxspawn delete' and 'lxspawn launch' instead", name, cfg.Name)
		}

		// Create backup by renaming rootfs for rollback protection
		rootfsPath := deploy.MachineRootfsPath(name)
		backupPath := rootfsPath + ".backup-" + fmt.Sprintf("%d", time.Now().Unix())
		hasBackup := false
		if deploy.FileExists(rootfsPath) {
			if err := os.Rename(rootfsPath, backupPath); err != nil {
				logger.Warn(fmt.Sprintf("failed to create rootfs backup: %v", err))
			} else {
				hasBackup = true
				logger.Info("Created rootfs backup for rollback protection")
			}
		}

		if err := mgr.Delete(name); err != nil {
			// Restore backup if delete failed
			if hasBackup {
				if restoreErr := os.Rename(backupPath, rootfsPath); restoreErr != nil {
					logger.Warn(fmt.Sprintf("failed to restore rootfs backup: %v", restoreErr))
				}
			}
			return fmt.Errorf("failed to delete old machine: %w", err)
		}

		if err := mgr.Launch(cfg, info.SourcePath); err != nil {
			// Launch failed, try to restore from backup
			if hasBackup {
				logger.Warn("Launch failed, attempting to restore from backup")
				// Delete the failed new rootfs if it exists
				newRootfsPath := deploy.MachineRootfsPath(name)
				if deploy.FileExists(newRootfsPath) {
					os.RemoveAll(newRootfsPath)
				}
				// Restore backup
				if restoreErr := os.Rename(backupPath, newRootfsPath); restoreErr != nil {
					return fmt.Errorf("failed to launch updated machine: %w (also failed to restore backup: %v)", err, restoreErr)
				}
		// Try to launch again with the restored rootfs
			if restoreErr := mgr.Launch(cfg, info.SourcePath); restoreErr != nil {
				return fmt.Errorf("failed to launch updated machine: %w (backup restored but launch still failed: %v)", err, restoreErr)
			}
			logger.Info("Successfully restored from backup")
			return nil
		}
		return fmt.Errorf("failed to launch updated machine: %w", err)
		}

		// Clean up backup if launch succeeded
		if hasBackup {
			if err := os.RemoveAll(backupPath); err != nil {
				logger.Warn(fmt.Sprintf("failed to remove backup: %v", err))
			}
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "edit",
				Machine: name,
				Success: true,
			})
		}

		return nil
	},
}

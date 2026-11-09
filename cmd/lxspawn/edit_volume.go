package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var editVolumeCmd = &cobra.Command{
	Use:   "edit-volume <name>",
	Short: "Edit a volume mount of a machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := machine.NewManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		info, exists := mgr.GetMachineInfo(args[0])
		if !exists {
			return fmt.Errorf("machine '%s' not found", args[0])
		}

		if info.SourcePath == "" {
			return fmt.Errorf("machine '%s' has no source YAML path", args[0])
		}

		data, err := os.ReadFile(info.SourcePath)
		if err != nil {
			return fmt.Errorf("failed to read YAML: %w", err)
		}

		var node yaml.Node
		if err := yaml.Unmarshal(data, &node); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		if node.Kind == 0 {
			return fmt.Errorf("empty YAML file")
		}

		if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
			return fmt.Errorf("invalid YAML structure")
		}

		root := node.Content[0]
		if root.Kind != yaml.MappingNode {
			return fmt.Errorf("invalid YAML structure")
		}

		mountPath, _ := cmd.Flags().GetString("mount")
		if mountPath == "" {
			return fmt.Errorf("--mount is required to identify the volume")
		}

		var volumesNode *yaml.Node
		for i := 0; i < len(root.Content)-1; i += 2 {
			if root.Content[i].Value == "volumes" {
				volumesNode = root.Content[i+1]
				break
			}
		}

		if volumesNode == nil || volumesNode.Kind != yaml.SequenceNode {
			return fmt.Errorf("no volumes found")
		}

		found := false
		for _, volNode := range volumesNode.Content {
			if volNode.Kind != yaml.MappingNode {
				continue
			}

			var currentMount string
			for j := 0; j < len(volNode.Content)-1; j += 2 {
				if volNode.Content[j].Value == "mount" {
					currentMount = volNode.Content[j+1].Value
					break
				}
			}

			if currentMount == mountPath {
				found = true

				if cmd.Flags().Changed("host") {
					hostPath, _ := cmd.Flags().GetString("host")
					setVolumeField(volNode, "host", hostPath)
				}
				if cmd.Flags().Changed("readonly") {
					readonly, _ := cmd.Flags().GetBool("readonly")
					setVolumeField(volNode, "readonly", fmt.Sprintf("%t", readonly))
				}
				break
			}
		}

		if !found {
			return fmt.Errorf("volume with mount path '%s' not found", mountPath)
		}

		out, err := yaml.Marshal(&node)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}

		if err := os.WriteFile(info.SourcePath, out, 0644); err != nil {
			return fmt.Errorf("failed to write YAML: %w", err)
		}

		logger.Success(fmt.Sprintf("Updated volume %s for machine '%s'", mountPath, args[0]))
		logger.Info("Restart the machine for changes to take effect")

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "edit-volume",
				Machine: args[0],
				Success: true,
			})
		}

		return nil
	},
}

func setVolumeField(volNode *yaml.Node, key, value string) {
	for j := 0; j < len(volNode.Content)-1; j += 2 {
		if volNode.Content[j].Value == key {
			volNode.Content[j+1].Value = value
			return
		}
	}

	volNode.Content = append(volNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

func init() {
	editVolumeCmd.Flags().String("mount", "", "Container mount path (identifier)")
	editVolumeCmd.Flags().String("host", "", "New host path")
	editVolumeCmd.Flags().Bool("readonly", false, "Read-only mount")
}

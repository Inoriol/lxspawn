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

var addVolumeCmd = &cobra.Command{
	Use:   "add-volume <name>",
	Short: "Add a volume mount to a machine",
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

		hostPath, _ := cmd.Flags().GetString("host")
		mountPath, _ := cmd.Flags().GetString("mount")
		readonly, _ := cmd.Flags().GetBool("readonly")

		if hostPath == "" || mountPath == "" {
			return fmt.Errorf("both --host and --mount are required")
		}

		var volumesNode *yaml.Node
		for i := 0; i < len(root.Content)-1; i += 2 {
			if root.Content[i].Value == "volumes" {
				volumesNode = root.Content[i+1]
				break
			}
		}

		if volumesNode == nil {
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "volumes",
			}
			volumesNode = &yaml.Node{
				Kind: yaml.SequenceNode,
			}
			root.Content = append(root.Content, keyNode, volumesNode)
		}

		volumeNode := &yaml.Node{
			Kind: yaml.MappingNode,
		}

		volumeNode.Content = append(volumeNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "host"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: hostPath},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "mount"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: mountPath},
		)

		if readonly {
			volumeNode.Content = append(volumeNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "readonly"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: "true"},
			)
		}

		volumesNode.Content = append(volumesNode.Content, volumeNode)

		out, err := yaml.Marshal(&node)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}

		if err := os.WriteFile(info.SourcePath, out, 0644); err != nil {
			return fmt.Errorf("failed to write YAML: %w", err)
		}

		logger.Success(fmt.Sprintf("Added volume %s:%s to machine '%s'", hostPath, mountPath, args[0]))
		logger.Info("Restart the machine for changes to take effect")

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "add-volume",
				Machine: args[0],
				Success: true,
			})
		}

		return nil
	},
}

func init() {
	addVolumeCmd.Flags().String("host", "", "Host path")
	addVolumeCmd.Flags().String("mount", "", "Container mount path")
	addVolumeCmd.Flags().Bool("readonly", false, "Read-only mount")
}

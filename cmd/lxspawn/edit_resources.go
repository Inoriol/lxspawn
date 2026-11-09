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

var editResourcesCmd = &cobra.Command{
	Use:   "edit-resources <name>",
	Short: "Edit resource limits of a machine",
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

		cpus, _ := cmd.Flags().GetFloat64("cpus")
		memory, _ := cmd.Flags().GetString("memory")
		swap, _ := cmd.Flags().GetString("swap")
		tasks, _ := cmd.Flags().GetString("tasks")

		modified := false

		if cmd.Flags().Changed("cpus") {
			setYAMLField(root, "resources", "cpus", cpus)
			modified = true
		}
		if cmd.Flags().Changed("memory") {
			setYAMLField(root, "resources", "memory", memory)
			modified = true
		}
		if cmd.Flags().Changed("swap") {
			setYAMLField(root, "resources", "swap", swap)
			modified = true
		}
		if cmd.Flags().Changed("tasks") {
			setYAMLField(root, "resources", "tasks", tasks)
			modified = true
		}

		if !modified {
			return fmt.Errorf("no changes specified")
		}

		out, err := yaml.Marshal(&node)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}

		if err := os.WriteFile(info.SourcePath, out, 0644); err != nil {
			return fmt.Errorf("failed to write YAML: %w", err)
		}

		logger.Success(fmt.Sprintf("Updated resources for machine '%s'", args[0]))
		logger.Info("Restart the machine for changes to take effect")

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "edit-resources",
				Machine: args[0],
				Success: true,
			})
		}

		return nil
	},
}

func init() {
	editResourcesCmd.Flags().Float64("cpus", 0, "CPU cores")
	editResourcesCmd.Flags().String("memory", "", "Memory limit (e.g., 2G)")
	editResourcesCmd.Flags().String("swap", "", "Swap limit (e.g., 512M)")
	editResourcesCmd.Flags().String("tasks", "", "Max tasks (e.g., 512 or infinity)")
}

func setYAMLField(root *yaml.Node, section, key string, value interface{}) {
	var resourcesNode *yaml.Node

	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == section {
			resourcesNode = root.Content[i+1]
			break
		}
	}

	if resourcesNode == nil {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: section,
		}
		resourcesNode = &yaml.Node{
			Kind: yaml.MappingNode,
		}
		root.Content = append(root.Content, keyNode, resourcesNode)
	}

	found := false
	for i := 0; i < len(resourcesNode.Content)-1; i += 2 {
		if resourcesNode.Content[i].Value == key {
			switch v := value.(type) {
			case string:
				resourcesNode.Content[i+1].Value = v
			case float64:
				resourcesNode.Content[i+1].Value = fmt.Sprintf("%g", v)
			}
			found = true
			break
		}
	}

	if !found {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		}
		var valueNode *yaml.Node
		switch v := value.(type) {
		case string:
			valueNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: v,
			}
		case float64:
			valueNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%g", v),
			}
		}
		resourcesNode.Content = append(resourcesNode.Content, keyNode, valueNode)
	}
}

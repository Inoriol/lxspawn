package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var editNetworkCmd = &cobra.Command{
	Use:   "edit-network <name>",
	Short: "Edit network configuration of a machine",
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

		mode, _ := cmd.Flags().GetString("mode")
		bridge, _ := cmd.Flags().GetString("bridge")
		iface, _ := cmd.Flags().GetString("interface")

		modified := false

		if cmd.Flags().Changed("mode") {
			setYAMLField(root, "network", "mode", mode)
			modified = true
		}
		if cmd.Flags().Changed("bridge") {
			setYAMLField(root, "network", "bridge", bridge)
			modified = true
		}
		if cmd.Flags().Changed("interface") {
			setYAMLField(root, "network", "interface", iface)
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

		cfg, err := config.Parse(info.SourcePath)
		if err != nil {
			return fmt.Errorf("failed to parse updated config: %w", err)
		}

		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		if err := mgr.Apply(cfg, info.SourcePath); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		logger.Success(fmt.Sprintf("Updated network config for machine '%s'", args[0]))

		status, err := mgr.GetStatus(args[0])
		if err == nil && status.State == "running" {
			logger.Info("Restart the machine for changes to take effect")
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.LifecycleData{
				Action:  "edit-network",
				Machine: args[0],
				Success: true,
			})
		}

		return nil
	},
}

func init() {
	editNetworkCmd.Flags().String("mode", "", "Network mode (host, private, bridge, macvlan, ipvlan, adaptive)")
	editNetworkCmd.Flags().String("bridge", "", "Bridge name (for bridge mode)")
	editNetworkCmd.Flags().String("interface", "", "Interface name (for macvlan/ipvlan mode)")
}

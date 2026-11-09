package main

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/machine"
	"github.com/Inoriol/lxspawn/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config <name>",
	Short: "Show detailed configuration of a machine",
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

		cfg, err := config.Parse(info.SourcePath)
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		if output.IsJSONMode() {
			configData := &output.ConfigData{
				Name:   cfg.Name,
				Distro: cfg.Distro,
			}

			if cfg.Resources.CPUs > 0 || cfg.Resources.Memory != "" {
				configData.Resources = &output.ResourceJSON{
					CPUs:   cfg.Resources.CPUs,
					Memory: cfg.Resources.Memory,
					Swap:   cfg.Resources.Swap,
					Tasks:  cfg.Resources.Tasks,
				}
			}

			if cfg.Storage.Size != "" {
				configData.Storage = &output.StorageJSON{
					Size: cfg.Storage.Size,
				}
			}

			if cfg.Network.Mode != "" {
				configData.Network = &output.NetworkConfigJSON{
					Mode:      string(cfg.Network.Mode),
					Bridge:    cfg.Network.Bridge,
					Zone:      cfg.Network.Zone,
					Interface: cfg.Network.Interface,
					Ports:     cfg.Network.Ports,
				}
			}

			if len(cfg.Volumes) > 0 {
				configData.Volumes = make([]output.VolumeJSON, len(cfg.Volumes))
				for i, v := range cfg.Volumes {
					configData.Volumes[i] = output.VolumeJSON{
						Host:     v.Host,
						Mount:    v.Mount,
						ReadOnly: v.ReadOnly,
						Type:     v.Type,
						Options:  v.Options,
					}
				}
			}

			noNewPriv := true
			if cfg.Security.NoNewPrivileges != nil {
				noNewPriv = *cfg.Security.NoNewPrivileges
			}
			seccomp := true
			if cfg.Security.Seccomp != nil {
				seccomp = *cfg.Security.Seccomp
			}

			if noNewPriv != true || !seccomp || len(cfg.Security.Capabilities) > 0 || len(cfg.Security.DropCapabilities) > 0 || len(cfg.Security.Inaccessible) > 0 {
				configData.Security = &output.SecurityJSON{
					NoNewPrivileges:  noNewPriv,
					Seccomp:          seccomp,
					Capabilities:     cfg.Security.Capabilities,
					DropCapabilities: cfg.Security.DropCapabilities,
					Inaccessible:     cfg.Security.Inaccessible,
				}
			}

			if cfg.GPU.Enabled {
				configData.GPU = &output.GPUJSON{
					Enabled: cfg.GPU.Enabled,
					Type:    cfg.GPU.Type,
					Devices: cfg.GPU.Devices,
				}
			}

			if cfg.Advanced.Ephemeral || cfg.Advanced.ReadOnly || cfg.Advanced.Hostname != "" || len(cfg.Advanced.Environment) > 0 {
				configData.Advanced = &output.AdvancedJSON{
					Ephemeral:   cfg.Advanced.Ephemeral,
					ReadOnly:    cfg.Advanced.ReadOnly,
					Hostname:    cfg.Advanced.Hostname,
					Environment: cfg.Advanced.Environment,
				}
			}

			return output.PrintJSON(configData)
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendRow(table.Row{"Name", cfg.Name})
		t.AppendRow(table.Row{"Distro", cfg.Distro})

		if cfg.Resources.CPUs > 0 {
			t.AppendRow(table.Row{"CPUs", cfg.Resources.CPUs})
		}
		if cfg.Resources.Memory != "" {
			t.AppendRow(table.Row{"Memory", cfg.Resources.Memory})
		}
		if cfg.Resources.Swap != "" {
			t.AppendRow(table.Row{"Swap", cfg.Resources.Swap})
		}
		if cfg.Resources.Tasks != "" {
			t.AppendRow(table.Row{"Tasks", cfg.Resources.Tasks})
		}

		if cfg.Network.Mode != "" {
			t.AppendRow(table.Row{"Network Mode", cfg.Network.Mode})
		}
		if cfg.Network.Bridge != "" {
			t.AppendRow(table.Row{"Bridge", cfg.Network.Bridge})
		}
		if cfg.Network.Interface != "" {
			t.AppendRow(table.Row{"Interface", cfg.Network.Interface})
		}

		if len(cfg.Volumes) > 0 {
			for i, v := range cfg.Volumes {
				if i == 0 {
					t.AppendRow(table.Row{"Volumes", fmt.Sprintf("%s:%s", v.Host, v.Mount)})
				} else {
					t.AppendRow(table.Row{"", fmt.Sprintf("%s:%s", v.Host, v.Mount)})
				}
			}
		}

		if cfg.Advanced.Hostname != "" {
			t.AppendRow(table.Row{"Hostname", cfg.Advanced.Hostname})
		}
		if cfg.Advanced.Ephemeral {
			t.AppendRow(table.Row{"Ephemeral", "yes"})
		}
		if cfg.Advanced.ReadOnly {
			t.AppendRow(table.Row{"Read Only", "yes"})
		}

		t.Render()
		return nil
	},
}

package main

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/output"
)

var listDistrosCmd = &cobra.Command{
	Use:   "list-distros",
	Short: "List available distributions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		distros := []output.DistroInfo{
			{Name: "debian:bookworm", Description: "Debian 12 (stable)"},
			{Name: "debian:trixie", Description: "Debian 13 (testing)"},
			{Name: "ubuntu:noble", Description: "Ubuntu 24.04 LTS"},
			{Name: "ubuntu:jammy", Description: "Ubuntu 22.04 LTS"},
			{Name: "fedora:44", Description: "Fedora 44"},
			{Name: "fedora:43", Description: "Fedora 43"},
			{Name: "centos:9-Stream", Description: "CentOS 9 Stream"},
			{Name: "rockylinux:9", Description: "Rocky Linux 9"},
			{Name: "almalinux:9", Description: "AlmaLinux 9"},
			{Name: "alpine:3.20", Description: "Alpine Linux 3.20"},
			{Name: "alpine:3.19", Description: "Alpine Linux 3.19"},
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.DistroListData{
				Distros: distros,
			})
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendHeader(table.Row{"Distro", "Description"})

		for _, d := range distros {
			t.AppendRow(table.Row{d.Name, d.Description})
		}

		t.Render()
		return nil
	},
}

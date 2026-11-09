package main

import (
	"os"
	"path/filepath"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/output"
)

var listTemplatesCmd = &cobra.Command{
	Use:   "list-templates",
	Short: "List available machine templates",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		templateDirs := []string{
			"/usr/share/lxspawn/templates",
			"/usr/local/share/lxspawn/templates",
			filepath.Join(os.Getenv("HOME"), ".local/share/lxspawn/templates"),
			"examples/templates",
		}

		var templates []map[string]string

		for _, dir := range templateDirs {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				continue
			}

			files, err := os.ReadDir(dir)
			if err != nil {
				continue
			}

			for _, file := range files {
				if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
					continue
				}

				name := file.Name()
				name = name[:len(name)-5]

				description := getTemplateDescription(name)

				templates = append(templates, map[string]string{
					"name":        name,
					"description": description,
					"path":        filepath.Join(dir, file.Name()),
				})
			}
		}

		if output.IsJSONMode() {
			return output.PrintJSON(map[string]interface{}{
				"templates": templates,
			})
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.AppendHeader(table.Row{"Template", "Description", "Path"})

		for _, tmpl := range templates {
			t.AppendRow(table.Row{tmpl["name"], tmpl["description"], tmpl["path"]})
		}

		t.Render()
		return nil
	},
}

func getTemplateDescription(name string) string {
	descriptions := map[string]string{
		"debian-base":  "Debian base system",
		"ubuntu-base":  "Ubuntu base system",
		"fedora-base":  "Fedora base system",
		"docker-host":  "Docker-in-container with full support",
		"pihole":       "Pi-hole DNS server",
		"adguard":      "AdGuard Home DNS server",
		"jellyfin":     "Jellyfin media server with GPU support",
		"plex":         "Plex media server with GPU support",
		"samba":        "Samba file server",
		"nextcloud":    "Nextcloud file sync server",
	}

	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return "Custom template"
}

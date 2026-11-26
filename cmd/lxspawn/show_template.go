package main

import (
	"fmt"

	"github.com/spf13/cobra"

	lxerrors "github.com/Inoriol/lxspawn/internal/errors"
	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/templates"
)

var showTemplateCmd = &cobra.Command{
	Use:   "show-template <name>",
	Short: "Display the content of an embedded template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		content, ok := templates.GetEmbeddedTemplateContent(name)
		if !ok {
			return lxerrors.NewValidationError("template", fmt.Sprintf("template '%s' not found", name))
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.TemplateData{
				Name:    name,
				Content: string(content),
			})
		}

		fmt.Fprint(cmd.OutOrStdout(), string(content))
		return nil
	},
}

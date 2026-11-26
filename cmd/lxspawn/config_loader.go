package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Inoriol/lxspawn/internal/config"
	lxerrors "github.com/Inoriol/lxspawn/internal/errors"
	"github.com/Inoriol/lxspawn/templates"
)

func loadConfig(source string, templateName string, machineName string, params []string) (*config.Config, string, error) {
	var cfg *config.Config
	var sourcePath string
	var err error

	if templateName != "" {
		content, ok := templates.GetEmbeddedTemplateContent(templateName)
		if !ok {
			return nil, "", lxerrors.NewValidationError("template", fmt.Sprintf("template '%s' not found", templateName))
		}

		cfg, err = config.ParseString(string(content))
		if err != nil {
			return nil, "", lxerrors.NewConfigInvalidError(fmt.Sprintf("failed to parse template: %v", err))
		}

		if machineName != "" {
			cfg.Name = machineName
			cfg.Advanced.Hostname = machineName
		} else if cfg.Name == "" {
			return nil, "", lxerrors.NewValidationError("name", "machine name required (use --name)")
		}

		sourcePath = fmt.Sprintf("template:%s", templateName)
	} else if source == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "", lxerrors.NewInternalError("failed to read from stdin", err)
		}

		cfg, err = config.ParseString(string(data))
		if err != nil {
			return nil, "", lxerrors.NewConfigInvalidError(fmt.Sprintf("failed to parse stdin: %v", err))
		}

		if machineName != "" {
			cfg.Name = machineName
			cfg.Advanced.Hostname = machineName
		}

		sourcePath = "stdin"
	} else {
		cfg, err = config.Parse(source)
		if err != nil {
			return nil, "", lxerrors.NewConfigInvalidError(fmt.Sprintf("failed to parse config: %v", err))
		}

		if machineName != "" {
			cfg.Name = machineName
			cfg.Advanced.Hostname = machineName
		}

		sourcePath = source
	}

	if len(params) > 0 {
		if cfg.Param == nil {
			cfg.Param = make(map[string]string)
		}
		for _, p := range params {
			parts := strings.SplitN(p, "=", 2)
			if len(parts) != 2 {
				return nil, "", lxerrors.NewValidationError("param", fmt.Sprintf("invalid param format '%s' (expected key=value)", p))
			}
			key := parts[0]
			if cfg.Param != nil {
				if _, exists := cfg.Param[key]; !exists {
					return nil, "", lxerrors.NewValidationError("param", fmt.Sprintf("unknown param key '%s' (not declared in param: section)", key))
				}
			}
			cfg.Param[key] = parts[1]
		}
	}

	if err := config.Validate(cfg); err != nil {
		return nil, "", lxerrors.NewValidationError("config", fmt.Sprintf("invalid config: %v", err))
	}

	return cfg, sourcePath, nil
}

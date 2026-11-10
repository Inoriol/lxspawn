package provision

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/logger"
)

const bootServiceContent = `[Unit]
Description=lxspawn boot provisioning
ConditionPathExists=/etc/lxspawn/boot/
Before=multi-user.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/bash -c 'for f in /etc/lxspawn/boot/*.sh; do "$f"; done'

[Install]
WantedBy=multi-user.target
`

type BootProvisioner struct {
	rootfsPath string
}

func NewBootProvisioner(rootfsPath string) *BootProvisioner {
	return &BootProvisioner{rootfsPath: rootfsPath}
}

func (b *BootProvisioner) Provision(prov config.Provision, ctx *TemplateContext, fileHandler *FileHandler, baseDir string) error {
	script := prov.Script
	if prov.File != nil {
		resolvedPath, err := fileHandler.ResolveFileRef(prov.File.URL, prov.File.Digest, baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve file: %w", err)
		}
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		script = string(data)
	}

	rendered, err := RenderTemplate(script, ctx)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	bootDir := filepath.Join(b.rootfsPath, "etc", "lxspawn", "boot")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return fmt.Errorf("failed to create boot dir: %w", err)
	}

	// Find next script number
	entries, err := os.ReadDir(bootDir)
	if err != nil {
		return fmt.Errorf("failed to read boot dir: %w", err)
	}

	scriptNum := len(entries)
	scriptName := fmt.Sprintf("%02d-script.sh", scriptNum)
	scriptPath := filepath.Join(bootDir, scriptName)

	if err := os.WriteFile(scriptPath, []byte(rendered), 0755); err != nil {
		return fmt.Errorf("failed to write boot script: %w", err)
	}

	logger.Info(fmt.Sprintf("Wrote boot script: %s", scriptPath))

	// Install boot service if not already present
	serviceDir := filepath.Join(b.rootfsPath, "etc", "systemd", "system")
	servicePath := filepath.Join(serviceDir, "lxspawn-boot.service")
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("failed to create service dir: %w", err)
		}
		if err := os.WriteFile(servicePath, []byte(bootServiceContent), 0644); err != nil {
			return fmt.Errorf("failed to write boot service: %w", err)
		}

		// Enable the service
		enableDir := filepath.Join(b.rootfsPath, "etc", "systemd", "system", "multi-user.target.wants")
		if err := os.MkdirAll(enableDir, 0755); err != nil {
			return fmt.Errorf("failed to create enable dir: %w", err)
		}

		enableLink := filepath.Join(enableDir, "lxspawn-boot.service")
		if err := os.Symlink("../lxspawn-boot.service", enableLink); err != nil {
			return fmt.Errorf("failed to enable boot service: %w", err)
		}

		logger.Info("Installed lxspawn-boot.service")
	}

	return nil
}

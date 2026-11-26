package provision

import (
	"fmt"
	"os"

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

	if err := rootfsMkdirAll(b.rootfsPath, "/etc/lxspawn/boot"); err != nil {
		return fmt.Errorf("failed to create boot dir: %w", err)
	}

	exists, err := rootfsFileExists(b.rootfsPath, "/etc/lxspawn/boot")
	if err != nil {
		return fmt.Errorf("failed to check boot dir: %w", err)
	}
	if !exists {
		return fmt.Errorf("boot directory does not exist after creation")
	}

	entries, err := listRootfsDir(b.rootfsPath, "/etc/lxspawn/boot")
	if err != nil {
		return fmt.Errorf("failed to read boot dir: %w", err)
	}

	scriptNum := len(entries)
	scriptName := fmt.Sprintf("%02d-script.sh", scriptNum)
	scriptRelPath := fmt.Sprintf("/etc/lxspawn/boot/%s", scriptName)

	if err := rootfsWriteFile(b.rootfsPath, scriptRelPath, []byte(rendered), 0700); err != nil {
		return fmt.Errorf("failed to write boot script: %w", err)
	}

	logger.Info(fmt.Sprintf("Wrote boot script: %s", scriptRelPath))

	serviceRelPath := "/etc/systemd/system/lxspawn-boot.service"
	exists, err = rootfsFileExists(b.rootfsPath, serviceRelPath)
	if err != nil {
		return fmt.Errorf("failed to check service file: %w", err)
	}

	if !exists {
		if err := rootfsWriteFile(b.rootfsPath, serviceRelPath, []byte(bootServiceContent), 0644); err != nil {
			return fmt.Errorf("failed to write boot service: %w", err)
		}

		enableLinkRelPath := "/etc/systemd/system/multi-user.target.wants/lxspawn-boot.service"
		if err := rootfsSymlink(b.rootfsPath, "../lxspawn-boot.service", enableLinkRelPath); err != nil {
			return fmt.Errorf("failed to enable boot service: %w", err)
		}

		logger.Info("Installed lxspawn-boot.service")
	}

	return nil
}

func listRootfsDir(rootfsPath, relPath string) ([]string, error) {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

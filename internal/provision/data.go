package provision

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/logger"
)

type DataProvisioner struct {
	rootfsPath string
}

func NewDataProvisioner(rootfsPath string) *DataProvisioner {
	return &DataProvisioner{rootfsPath: rootfsPath}
}

func (d *DataProvisioner) Provision(prov config.Provision, ctx *TemplateContext, fileHandler *FileHandler, baseDir string) error {
	content := prov.Content
	if prov.File != nil {
		resolvedPath, err := fileHandler.ResolveFileRef(prov.File.URL, prov.File.Digest, baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve file: %w", err)
		}
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		content = string(data)
	}

	rendered, err := RenderTemplate(content, ctx)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	relPath := prov.Path
	if !filepath.IsAbs(relPath) {
		return fmt.Errorf("path must be absolute: %s", relPath)
	}

	overwrite := true
	if prov.Overwrite != nil {
		overwrite = *prov.Overwrite
	}

	exists, err := rootfsFileExists(d.rootfsPath, relPath)
	if err != nil {
		return fmt.Errorf("failed to check if file exists: %w", err)
	}

	if exists && !overwrite {
		logger.Info(fmt.Sprintf("Skipping %s (already exists, overwrite=false)", prov.Path))
		return nil
	}

	perm := os.FileMode(0644)
	if prov.Permissions != "" {
		parsed, err := strconv.ParseUint(prov.Permissions, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid permissions %q: %w", prov.Permissions, err)
		}
		perm = os.FileMode(parsed)
	}

	if err := rootfsWriteFile(d.rootfsPath, relPath, []byte(rendered), perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", prov.Path, err)
	}

	logger.Debug(fmt.Sprintf("Wrote %s", prov.Path))

	if prov.Owner != "" {
		if err := setOwnership(d.rootfsPath, relPath, prov.Owner); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", prov.Path, err)
		}
	}

	return nil
}

func setOwnership(rootfsPath, relPath string, owner string) error {
	parts := strings.SplitN(owner, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid owner format: %s (expected user:group)", owner)
	}

	uidStr, gidStr := parts[0], parts[1]

	uid, uidErr := strconv.Atoi(uidStr)
	gid, gidErr := strconv.Atoi(gidStr)

	if uidErr != nil {
		uid = lookupUIDInRootfs(uidStr, rootfsPath)
		if uid < 0 {
			logger.Info(fmt.Sprintf("Note: user %s not found in container rootfs; ownership will be applied at runtime", uidStr))
			return nil
		}
	}

	if gidErr != nil {
		gid = lookupGIDInRootfs(gidStr, rootfsPath)
		if gid < 0 {
			logger.Info(fmt.Sprintf("Note: group %s not found in container rootfs; ownership will be applied at runtime", gidStr))
			return nil
		}
	}

	if err := rootfsChown(rootfsPath, relPath, uid, gid); err != nil {
		return fmt.Errorf("failed to chown %s to %d:%d: %w", relPath, uid, gid, err)
	}

	logger.Info(fmt.Sprintf("Set ownership of %s to %d:%d", relPath, uid, gid))
	return nil
}

func lookupUIDInRootfs(name string, rootfsPath string) int {
	data, err := rootfsReadFile(rootfsPath, "/etc/passwd")
	if err != nil {
		return lookupUID(name)
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[0] == name {
			if uid, err := strconv.Atoi(fields[2]); err == nil {
				return uid
			}
		}
	}

	return lookupUID(name)
}

func lookupGIDInRootfs(name string, rootfsPath string) int {
	data, err := rootfsReadFile(rootfsPath, "/etc/group")
	if err != nil {
		return lookupGID(name)
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[0] == name {
			if gid, err := strconv.Atoi(fields[2]); err == nil {
				return gid
			}
		}
	}

	return lookupGID(name)
}

func lookupUID(name string) int {
	switch name {
	case "root":
		return 0
	case "nobody":
		return 65534
	default:
		return -1
	}
}

func lookupGID(name string) int {
	switch name {
	case "root":
		return 0
	case "nobody":
		return 65534
	default:
		return -1
	}
}

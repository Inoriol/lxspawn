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

	targetPath, err := validatePath(d.rootfsPath, prov.Path)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", prov.Path, err)
	}

	// Check if file exists and overwrite is false
	overwrite := true
	if prov.Overwrite != nil {
		overwrite = *prov.Overwrite
	}

	if _, err := os.Stat(targetPath); err == nil && !overwrite {
		logger.Info(fmt.Sprintf("Skipping %s (already exists, overwrite=false)", prov.Path))
		return nil
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent dirs: %w", err)
	}

	// Determine permissions
	perm := os.FileMode(0644)
	if prov.Permissions != "" {
		parsed, err := strconv.ParseUint(prov.Permissions, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid permissions %q: %w", prov.Permissions, err)
		}
		perm = os.FileMode(parsed)
	}

	if err := os.WriteFile(targetPath, []byte(rendered), perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info(fmt.Sprintf("Wrote %s", prov.Path))

	// Set ownership if specified
	if prov.Owner != "" {
		if err := setOwnership(targetPath, prov.Owner, d.rootfsPath); err != nil {
			logger.Warn(fmt.Sprintf("failed to set ownership on %s: %v", prov.Path, err))
		}
	}

	return nil
}

func setOwnership(path string, owner string, rootfsPath string) error {
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

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("failed to chown %s to %d:%d: %w", path, uid, gid, err)
	}

	logger.Info(fmt.Sprintf("Set ownership of %s to %d:%d", path, uid, gid))
	return nil
}

func lookupUIDInRootfs(name string, rootfsPath string) int {
	passwdPath := filepath.Join(rootfsPath, "etc", "passwd")
	data, err := os.ReadFile(passwdPath)
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
	groupPath := filepath.Join(rootfsPath, "etc", "group")
	data, err := os.ReadFile(groupPath)
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

func validatePath(rootfsPath, relativePath string) (string, error) {
	if !filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("path must be absolute")
	}

	targetPath := filepath.Join(rootfsPath, relativePath)
	cleanTarget := filepath.Clean(targetPath)
	cleanRootfs := filepath.Clean(rootfsPath)

	if !strings.HasPrefix(cleanTarget, cleanRootfs+string(os.PathSeparator)) && cleanTarget != cleanRootfs {
		return "", fmt.Errorf("path escapes rootfs directory")
	}

	return targetPath, nil
}

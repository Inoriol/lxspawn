package provision

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Inoriol/lxspawn/internal/deploy"
)

func isRoot() bool {
	return os.Geteuid() == 0
}

func canWriteToRootfs(rootfsPath string) bool {
	testFile := filepath.Join(rootfsPath, ".lxspawn-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}

func sudoOrDirectForRootfs(rootfsPath string, args ...string) *exec.Cmd {
	if isRoot() || canWriteToRootfs(rootfsPath) {
		return exec.Command(args[0], args[1:]...)
	}
	return deploy.Sudo(args...)
}

func secureJoin(root, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute: %s", path)
	}

	cleanRoot := filepath.Clean(root)

	relPath := strings.TrimPrefix(filepath.Clean(path), "/")
	if relPath == "" || relPath == "." {
		return cleanRoot, nil
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	current := cleanRoot

	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		next := filepath.Join(current, part)

		fi, err := os.Lstat(next)
		if err != nil {
			if os.IsNotExist(err) {
				remaining := parts[i+1:]
				if len(remaining) > 0 {
					return filepath.Join(next, filepath.Join(remaining...)), nil
				}
				return next, nil
			}
			return "", err
		}

		for fi.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(next)
			if err != nil {
				return "", err
			}

			var resolved string
			if filepath.IsAbs(linkTarget) {
				if strings.HasPrefix(linkTarget, cleanRoot+string(os.PathSeparator)) || linkTarget == cleanRoot {
					resolved = linkTarget
				} else {
					return "", fmt.Errorf("symlink %q points outside root: %s", next, linkTarget)
				}
			} else {
				resolved = filepath.Join(filepath.Dir(next), linkTarget)
			}
			resolved = filepath.Clean(resolved)

			if !strings.HasPrefix(resolved, cleanRoot+string(os.PathSeparator)) && resolved != cleanRoot {
				return "", fmt.Errorf("symlink %q points outside root: %s", next, linkTarget)
			}

			next = resolved
			fi, err = os.Lstat(next)
			if err != nil {
				if os.IsNotExist(err) {
					remaining := parts[i+1:]
					if len(remaining) > 0 {
						return filepath.Join(next, filepath.Join(remaining...)), nil
					}
					return next, nil
				}
				return "", err
			}
		}

		current = next
	}

	return current, nil
}

func openFileNoFollow(path string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flag|syscall.O_NOFOLLOW, perm)
}

func rootfsMkdirAll(rootfsPath, relPath string) error {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}

	if deploy.FileExists(fullPath) {
		return nil
	}

	cmd := sudoOrDirectForRootfs(rootfsPath, "mkdir", "-p", fullPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create directory %s: %w (%s)", fullPath, err, string(output))
	}
	return nil
}

func rootfsWriteFile(rootfsPath, relPath string, content []byte, perm os.FileMode) error {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}

	parentDir := filepath.Dir(fullPath)
	if err := rootfsMkdirAll(rootfsPath, strings.TrimPrefix(parentDir, rootfsPath)); err != nil {
		return err
	}

	cmd := sudoOrDirectForRootfs(rootfsPath, "tee", fullPath)
	cmd.Stdin = strings.NewReader(string(content))
	cmd.Stdout = nil
	cmd.Stderr = nil
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write file %s: %w (%s)", fullPath, err, string(output))
	}

	permStr := fmt.Sprintf("%04o", perm)
	chmodCmd := sudoOrDirectForRootfs(rootfsPath, "chmod", permStr, fullPath)
	if output, err := chmodCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chmod %s: %w (%s)", fullPath, err, string(output))
	}

	return nil
}

func rootfsSymlink(rootfsPath, target, linkRelPath string) error {
	linkPath, err := secureJoin(rootfsPath, linkRelPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", linkRelPath, err)
	}

	if deploy.FileExists(linkPath) {
		return nil
	}

	parentDir := filepath.Dir(linkPath)
	if err := rootfsMkdirAll(rootfsPath, strings.TrimPrefix(parentDir, rootfsPath)); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	cmd := sudoOrDirectForRootfs(rootfsPath, "ln", "-s", target, linkPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w (%s)", linkPath, target, err, string(output))
	}
	return nil
}

func rootfsRemove(rootfsPath, relPath string) error {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}

	if !deploy.FileExists(fullPath) {
		return nil
	}

	cmd := sudoOrDirectForRootfs(rootfsPath, "rm", "-f", fullPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove %s: %w (%s)", fullPath, err, string(output))
	}
	return nil
}

func rootfsReadFile(rootfsPath, relPath string) ([]byte, error) {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", fullPath, err)
	}
	return data, nil
}

func rootfsFileExists(rootfsPath, relPath string) (bool, error) {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}
	return deploy.FileExists(fullPath), nil
}

func rootfsChmod(rootfsPath, relPath string, perm os.FileMode) error {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}

	permStr := fmt.Sprintf("%04o", perm)
	cmd := sudoOrDirectForRootfs(rootfsPath, "chmod", permStr, fullPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chmod %s: %w (%s)", fullPath, err, string(output))
	}
	return nil
}

func rootfsChown(rootfsPath, relPath string, uid, gid int) error {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}

	owner := fmt.Sprintf("%d:%d", uid, gid)
	cmd := sudoOrDirectForRootfs(rootfsPath, "chown", owner, fullPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chown %s: %w (%s)", fullPath, err, string(output))
	}
	return nil
}

func rootfsOpenFileNoFollow(rootfsPath, relPath string, flag int, perm os.FileMode) (*os.File, error) {
	fullPath, err := secureJoin(rootfsPath, relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", relPath, err)
	}
	return openFileNoFollow(fullPath, flag, perm)
}

func rootfsExecSudo(rootfsPath string, args ...string) *exec.Cmd {
	return sudoOrDirectForRootfs(rootfsPath,args...)
}

func validatePath(rootfsPath, relativePath string) (string, error) {
	if !filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("path must be absolute: %s", relativePath)
	}

	targetPath := filepath.Join(rootfsPath, relativePath)
	cleanTarget := filepath.Clean(targetPath)
	cleanRootfs := filepath.Clean(rootfsPath)

	if !strings.HasPrefix(cleanTarget, cleanRootfs+string(os.PathSeparator)) && cleanTarget != cleanRootfs {
		return "", fmt.Errorf("path %q escapes rootfs directory %s", relativePath, rootfsPath)
	}

	resolvedPath, err := secureJoin(cleanRootfs, relativePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %q: %w", relativePath, err)
	}

	if !strings.HasPrefix(resolvedPath, cleanRootfs+string(os.PathSeparator)) && resolvedPath != cleanRootfs {
		return "", fmt.Errorf("path %q resolves outside rootfs (via symlink)", relativePath)
	}

	return resolvedPath, nil
}

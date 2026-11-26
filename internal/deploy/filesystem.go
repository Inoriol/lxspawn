package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	NspawnDir      = DefaultNspawnDir
	SystemdDir     = DefaultSystemdDir
	MachinesDir    = DefaultMachinesDir
	LxspawnDataDir = DefaultLxspawnDataDir
)

func Sudo(args ...string) *exec.Cmd {
	sudoArgs := append([]string{"-n"}, args...)
	return exec.Command("sudo", sudoArgs...)
}

func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".lxspawn-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func RemoveAll(paths ...string) error {
	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("failed to remove %s: %w", p, err)
		}
	}
	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func CreatedAt() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func SELinuxEnforcing() bool {
	out, err := exec.Command("getenforce").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "Enforcing"
}

func SetSELinuxContext(path, context string) error {
	if !SELinuxEnforcing() {
		return nil
	}

	dir := filepath.Dir(path)
	cmd := exec.Command("semanage", "fcontext", "-a", "-t", context, dir+"/.*")
	if err := cmd.Run(); err != nil {
		return exec.Command("chcon", "-t", context, path).Run()
	}

	return exec.Command("restorecon", "-R", dir).Run()
}

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
	NspawnDir      = "/etc/systemd/nspawn"
	SystemdDir     = "/etc/systemd/system"
	MachinesDir    = "/var/lib/machines"
	LxspawnDataDir = "/var/lib/lxspawn"
)

func Sudo(args ...string) *exec.Cmd {
	return exec.Command("sudo", args...)
}

func NspawnFilePath(name string) string {
	return filepath.Join(NspawnDir, name+".nspawn")
}

func SliceFilePath(name string) string {
	return filepath.Join(SystemdDir, "machine-"+name+".slice")
}

func DropInDir(name string) string {
	return filepath.Join(SystemdDir, "systemd-nspawn@"+name+".service.d")
}

func DropInFilePath(name string) string {
	return filepath.Join(DropInDir(name), "lxspawn.conf")
}

func HooksDropInFilePath(name string) string {
	return filepath.Join(DropInDir(name), "lxspawn-hooks.conf")
}

func MachineRootfsPath(name string) string {
	return filepath.Join(MachinesDir, name)
}

func LxspawnMachineDir(name string) string {
	return filepath.Join(LxspawnDataDir, name)
}

func HooksDir(name string) string {
	return filepath.Join(LxspawnMachineDir(name), ".hooks")
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
	return exec.Command("chcon", "-t", context, path).Run()
}

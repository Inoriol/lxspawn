package machine

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

func IsLVM(path string) bool {
	out, err := exec.Command("df", "--output=source", path).Output()
	if err != nil {
		return false
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return false
	}

	device := strings.TrimSpace(lines[len(lines)-1])


	out, err = deploy.SudoHelper("lvs", "--noheadings", "-o", "vg_name", device).CombinedOutput()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) != ""
}

func getVGName(path string) (string, error) {
	out, err := exec.Command("df", "--output=source", path).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get device for %s: %w", path, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected df output")
	}

	device := strings.TrimSpace(lines[len(lines)-1])


	out, err = deploy.SudoHelper("lvs", "--noheadings", "-o", "vg_name", device).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get VG name for %s: %w", device, err)
	}

	vgName := strings.TrimSpace(string(out))
	if vgName == "" {
		return "", fmt.Errorf("device %s is not an LVM volume", device)
	}

	return vgName, nil
}

func getVGFreeSpaceGB(vgName string) (int, error) {

	cmd := deploy.SudoHelper("lvs", "--noheadings", "--nosuffix", "--units", "g", "-o", "vg_free", vgName)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get VG free space: %w", err)
	}

	freeStr := strings.TrimSpace(string(out))
	var freeGB float64
	if _, err := fmt.Sscanf(freeStr, "%f", &freeGB); err != nil {
		return 0, fmt.Errorf("failed to parse VG free space %q: %w", freeStr, err)
	}

	return int(freeGB), nil
}

func validateVGConfiguration(vgName string) error {

	cmd := deploy.SudoHelper("lvs", "--noheadings", "-o", "pv_name", vgName)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	if strings.Contains(string(out), "/dev/loop") {
		logger.Warn("VG contains loop devices - this may cause boot failures if VG contains root filesystem")
		logger.Warn("Consider using dedicated partitions or virtual disks instead of loop devices")
	}

	return nil
}

func CreateLVMVolume(machineName string, sizeGB int) error {
	if !IsLVM(deploy.MachinesDir) {
		return nil
	}

	vgName, err := getVGName(deploy.MachinesDir)
	if err != nil {
		return err
	}

	if err := validateVGConfiguration(vgName); err != nil {
		logger.Warn(fmt.Sprintf("VG validation warning: %v", err))
	}

	freeSpaceGB, err := getVGFreeSpaceGB(vgName)
	if err != nil {
		logger.Warn(fmt.Sprintf("Could not check VG free space: %v", err))
	} else if freeSpaceGB < sizeGB {
		return fmt.Errorf("insufficient VG free space: need %dGB, have %dGB. "+
			"Add space with: sudo lvextend or sudo vgextend, or reduce storage.size in config",
			sizeGB, freeSpaceGB)
	}

	lvName := fmt.Sprintf("lxspawn-%s", machineName)


	cmd := deploy.SudoHelper("lvcreate", "--name", lvName, "--size", fmt.Sprintf("%dG", sizeGB), "--yes", vgName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create LV %s: %w (%s)", lvName, err, string(output))
	}

	logger.Info(fmt.Sprintf("Created LVM volume %s/%s", vgName, lvName))
	return nil
}

func FormatAndMountLV(machineName string) error {
	if !IsLVM(deploy.MachinesDir) {
		return nil
	}

	vgName, err := getVGName(deploy.MachinesDir)
	if err != nil {
		return err
	}

	lvName := fmt.Sprintf("lxspawn-%s", machineName)
	lvPath := fmt.Sprintf("/dev/%s/%s", vgName, lvName)
	mountPath := deploy.MachineRootfsPath(machineName)

	cmd := deploy.SudoHelper("mkfs-ext4", "-q", lvPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to format %s: %w (%s)", lvPath, err, string(output))
	}

	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", mountPath, err)
	}

	cmd = deploy.SudoHelper("mount", lvPath, mountPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount %s at %s: %w (%s)", lvPath, mountPath, err, string(output))
	}

	cmd = deploy.SudoHelper("chown", "root:lxspawn", mountPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chown %s: %w (%s)", mountPath, err, string(output))
	}

	cmd = deploy.SudoHelper("chmod", "2775", mountPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chmod %s: %w (%s)", mountPath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Mounted %s at %s", lvPath, mountPath))
	return nil
}

func isMounted(path string) bool {
	out, err := exec.Command("mountpoint", "-q", path).CombinedOutput()
	return err == nil && len(out) == 0
}

func CleanupLVMVolume(machineName string) error {
	if !IsLVM(deploy.MachinesDir) {
		return nil
	}

	vgName, err := getVGName(deploy.MachinesDir)
	if err != nil {
		return err
	}

	lvName := fmt.Sprintf("lxspawn-%s", machineName)
	lvPath := fmt.Sprintf("/dev/%s/%s", vgName, lvName)
	mountPath := deploy.MachineRootfsPath(machineName)

	cmd := deploy.SudoHelper("umount", mountPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), "not mounted") {
			logger.Warn(fmt.Sprintf("failed to unmount %s: %v (%s)", mountPath, err, string(output)))
		}
	}


	cmd = deploy.SudoHelper("lvremove", "-f", lvPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove LV %s: %w (%s)", lvPath, err, string(output))
	}

	logger.Info(fmt.Sprintf("Removed LVM volume %s", lvPath))
	return nil
}

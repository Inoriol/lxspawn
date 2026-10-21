package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

var allowedBasePaths = []string{
	"/var/lib/machines/",
	"/var/lib/lxspawn/",
	"/etc/systemd/nspawn/",
	"/etc/systemd/system/",
}

var validMachineNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`)
var validSnapshotNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}[a-zA-Z0-9]$`)
var validLVNameRegex = regexp.MustCompile(`^lxspawn-[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`)

func main() {
	if len(os.Args) < 2 {
		fatal("usage: lxspawn-sudo-helper <subcommand> [args...]")
	}

	subcmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch subcmd {
	case "rm-rf":
		err = cmdRmRf(args)
	case "nsenter":
		err = cmdNsenter(args)
	case "chown":
		err = cmdChown(args)
	case "chmod":
		err = cmdChmod(args)
	case "mv":
		err = cmdMv(args)
	case "btrfs":
		err = cmdBtrfs(args)
	case "zfs":
		err = cmdZfs(args)
	case "mount":
		err = cmdMount(args)
	case "umount":
		err = cmdUmount(args)
	case "rsync":
		err = cmdRsync(args)
	case "lvcreate":
		err = cmdLvcreate(args)
	case "lvconvert":
		err = cmdLvconvert(args)
	case "lvremove":
		err = cmdLvremove(args)
	case "lvs":
		err = cmdLvs(args)
	case "mkfs-ext4":
		err = cmdMkfsExt4(args)
	case "journalctl":
		err = cmdJournalctl(args)
	case "findmnt":
		err = cmdFindmnt(args)
	case "mountpoint":
		err = cmdMountpoint(args)
	default:
		fatal("unknown subcommand: %s", subcmd)
	}

	if err != nil {
		fatal("%v", err)
	}
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "lxspawn-sudo-helper: "+format+"\n", args...)
	os.Exit(1)
}

func isPathAllowed(path string) bool {
	clean := filepath.Clean(path) + "/"
	for _, base := range allowedBasePaths {
		if strings.HasPrefix(clean, base) || clean == strings.TrimSuffix(base, "/") {
			return true
		}
	}
	return false
}

func containsPathTraversal(path string) bool {
	return strings.Contains(path, "..")
}

func validateMachineName(name string) error {
	if !validMachineNameRegex.MatchString(name) {
		return fmt.Errorf("invalid machine name: %s", name)
	}
	return nil
}

func validateSnapshotName(name string) error {
	if !validSnapshotNameRegex.MatchString(name) {
		return fmt.Errorf("invalid snapshot name: %s", name)
	}
	return nil
}

func validateLVName(name string) error {
	if !validLVNameRegex.MatchString(name) {
		return fmt.Errorf("invalid LV name: %s (must start with lxspawn-)", name)
	}
	return nil
}

func extractMachineNameFromPath(path string) (string, error) {
	clean := filepath.Clean(path)
	for _, base := range allowedBasePaths {
		baseClean := strings.TrimSuffix(base, "/")
		if strings.HasPrefix(clean, baseClean+"/") {
			rest := strings.TrimPrefix(clean, baseClean+"/")
			parts := strings.SplitN(rest, "/", 2)
			if len(parts) > 0 {
				return parts[0], nil
			}
		}
	}
	return "", fmt.Errorf("could not extract machine name from path: %s", path)
}

func cmdRmRf(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("rm-rf requires exactly 1 argument (path)")
	}
	path := args[0]
	if containsPathTraversal(path) {
		return fmt.Errorf("path contains '..': %s", path)
	}
	if !isPathAllowed(path) {
		return fmt.Errorf("path not in allowed directories: %s", path)
	}
	return execCommand("rm", "-rf", path)
}

func cmdNsenter(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("nsenter requires at least 2 arguments (-t <pid>)")
	}

	var targetPID string
	for i, arg := range args {
		if arg == "-t" && i+1 < len(args) {
			targetPID = args[i+1]
			break
		}
	}

	if targetPID == "" {
		return fmt.Errorf("nsenter requires -t <pid>")
	}

	if !isLxspawnMachinePID(targetPID) {
		return fmt.Errorf("PID %s does not belong to an lxspawn machine", targetPID)
	}

	return execCommand("nsenter", args...)
}

func isLxspawnMachinePID(pid string) bool {
	procExe, err := os.Readlink(fmt.Sprintf("/proc/%s/exe", pid))
	if err != nil {
		return false
	}
	if !strings.Contains(procExe, "systemd") {
		return false
	}

	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%s/cmdline", pid))
	if err != nil {
		return false
	}
	cmdStr := strings.ReplaceAll(string(cmdline), "\x00", " ")
	if !strings.Contains(cmdStr, "systemd-nspawn") {
		return false
	}

	cgroupPath := fmt.Sprintf("/proc/%s/cgroup", pid)
	cgroupData, err := os.ReadFile(cgroupPath)
	if err != nil {
		return false
	}
	cgroupStr := string(cgroupData)
	if !strings.Contains(cgroupStr, "machine-") || !strings.Contains(cgroupStr, ".slice") {
		return false
	}

	return true
}

func cmdChown(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("chown requires at least 2 arguments (owner path)")
	}

	var paths []string
	hasRecursive := false
	for _, arg := range args {
		if arg == "-R" || arg == "--recursive" {
			hasRecursive = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		paths = append(paths, arg)
	}

	if len(paths) < 2 {
		return fmt.Errorf("chown requires owner and path")
	}

	targetPath := paths[len(paths)-1]
	if containsPathTraversal(targetPath) {
		return fmt.Errorf("path contains '..': %s", targetPath)
	}
	if !isPathAllowed(targetPath) {
		return fmt.Errorf("path not in allowed directories: %s", targetPath)
	}

	_ = hasRecursive
	return execCommand("chown", args...)
}

func cmdChmod(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("chmod requires at least 2 arguments (mode path)")
	}

	var targetPath string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			targetPath = arg
		}
	}

	if targetPath == "" {
		return fmt.Errorf("chmod requires a path argument")
	}

	if containsPathTraversal(targetPath) {
		return fmt.Errorf("path contains '..': %s", targetPath)
	}
	if !isPathAllowed(targetPath) {
		return fmt.Errorf("path not in allowed directories: %s", targetPath)
	}

	return execCommand("chmod", args...)
}

func cmdMv(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("mv requires exactly 2 arguments (src dst)")
	}

	src, dst := args[0], args[1]

	if containsPathTraversal(src) {
		return fmt.Errorf("source path contains '..': %s", src)
	}
	if containsPathTraversal(dst) {
		return fmt.Errorf("destination path contains '..': %s", dst)
	}
	if !isPathAllowed(src) {
		return fmt.Errorf("source path not in allowed directories: %s", src)
	}
	if !isPathAllowed(dst) {
		return fmt.Errorf("destination path not in allowed directories: %s", dst)
	}

	return execCommand("mv", src, dst)
}

func cmdBtrfs(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("btrfs requires at least 2 arguments")
	}

	if args[0] != "subvolume" && args[0] != "filesystem" {
		return fmt.Errorf("btrfs subcommand must be 'subvolume' or 'filesystem', got: %s", args[0])
	}

	if args[0] == "filesystem" {
		if args[1] != "usage" {
			return fmt.Errorf("btrfs filesystem subcommand must be 'usage', got: %s", args[1])
		}
		for _, arg := range args[2:] {
			if !strings.HasPrefix(arg, "-") && !isPathAllowed(arg) {
				return fmt.Errorf("btrfs path not in allowed directories: %s", arg)
			}
		}
		return execCommand("btrfs", args...)
	}

	validSubcmds := map[string]bool{
		"create":  true,
		"delete":  true,
		"show":    true,
		"list":    true,
		"snapshot": true,
	}

	if !validSubcmds[args[1]] {
		return fmt.Errorf("btrfs subvolume subcommand not allowed: %s", args[1])
	}

	for _, arg := range args[2:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if containsPathTraversal(arg) {
			return fmt.Errorf("path contains '..': %s", arg)
		}
		if !isPathAllowed(arg) {
			return fmt.Errorf("path not in allowed directories: %s", arg)
		}
	}

	return execCommand("btrfs", args...)
}

func cmdZfs(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zfs requires at least 2 arguments")
	}

	validSubcmds := map[string]bool{
		"snapshot":  true,
		"rollback":  true,
		"destroy":   true,
	}

	if !validSubcmds[args[0]] {
		return fmt.Errorf("zfs subcommand not allowed: %s", args[0])
	}

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if !strings.Contains(arg, "/lxspawn-") && !strings.HasPrefix(arg, "lxspawn-") {
			return fmt.Errorf("zfs dataset must contain 'lxspawn-': %s", arg)
		}
	}

	return execCommand("zfs", args...)
}

func cmdMount(args []string) error {
	var targetPath string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "/dev/") {
			targetPath = arg
		}
	}

	if targetPath == "" {
		return fmt.Errorf("mount requires a mount point")
	}

	if containsPathTraversal(targetPath) {
		return fmt.Errorf("mount point contains '..': %s", targetPath)
	}
	if !isPathAllowed(targetPath) {
		return fmt.Errorf("mount point not in allowed directories: %s", targetPath)
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "/dev/") && !strings.Contains(arg, "lxspawn-") {
			return fmt.Errorf("device must be an lxspawn device: %s", arg)
		}
	}

	return execCommand("mount", args...)
}

func cmdUmount(args []string) error {
	var targetPath string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			targetPath = arg
		}
	}

	if targetPath == "" {
		return fmt.Errorf("umount requires a mount point")
	}

	if containsPathTraversal(targetPath) {
		return fmt.Errorf("mount point contains '..': %s", targetPath)
	}
	if !isPathAllowed(targetPath) {
		return fmt.Errorf("mount point not in allowed directories: %s", targetPath)
	}

	return execCommand("umount", args...)
}

func cmdRsync(args []string) error {
	var paths []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		paths = append(paths, arg)
	}

	if len(paths) < 2 {
		return fmt.Errorf("rsync requires at least source and destination")
	}

	for _, p := range paths {
		cleanPath := strings.TrimSuffix(p, "/")
		if containsPathTraversal(cleanPath) {
			return fmt.Errorf("path contains '..': %s", p)
		}
		if !isPathAllowed(cleanPath) {
			return fmt.Errorf("path not in allowed directories: %s", p)
		}
	}

	return execCommand("rsync", args...)
}

func cmdLvcreate(args []string) error {
	var lvName string
	for i, arg := range args {
		if (arg == "--name" || arg == "-n") && i+1 < len(args) {
			lvName = args[i+1]
			break
		}
	}

	if lvName == "" {
		return fmt.Errorf("lvcreate requires --name")
	}

	if err := validateLVName(lvName); err != nil {
		return err
	}

	return execCommand("lvcreate", args...)
}

func cmdLvconvert(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.Contains(arg, "/") {
			parts := strings.SplitN(arg, "/", 2)
			if len(parts) == 2 {
				lvName := parts[1]
				if err := validateLVName(lvName); err != nil {
					return err
				}
			}
		}
	}

	return execCommand("lvconvert", args...)
}

func cmdLvremove(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.Contains(arg, "/") {
			parts := strings.SplitN(arg, "/", 2)
			if len(parts) == 2 {
				lvName := parts[1]
				if err := validateLVName(lvName); err != nil {
					return err
				}
			}
		}
	}

	return execCommand("lvremove", args...)
}

func cmdLvs(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.Contains(arg, "/") {
			parts := strings.SplitN(arg, "/", 2)
			if len(parts) == 2 {
				lvName := parts[1]
				if err := validateLVName(lvName); err != nil {
					return err
				}
			}
		}
	}

	return execCommand("lvs", args...)
}

func cmdMkfsExt4(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mkfs.ext4 requires a device argument")
	}

	device := args[len(args)-1]
	if !strings.HasPrefix(device, "/dev/") {
		return fmt.Errorf("device must be a block device: %s", device)
	}
	if !strings.Contains(device, "lxspawn-") {
		return fmt.Errorf("device must be an lxspawn device: %s", device)
	}

	return execCommand("mkfs.ext4", args...)
}

func cmdJournalctl(args []string) error {
	return execCommand("journalctl", args...)
}

func cmdFindmnt(args []string) error {
	return execCommand("findmnt", args...)
}

func cmdMountpoint(args []string) error {
	return execCommand("mountpoint", args...)
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

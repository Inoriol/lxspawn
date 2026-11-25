package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
	"github.com/Inoriol/lxspawn/internal/network"
	"github.com/Inoriol/lxspawn/internal/output"
)

const polkitRule = `polkit.addRule(function(action, subject) {
  if (subject.isInGroup("lxspawn")) {
    if (action.id == "org.freedesktop.systemd1.manage-units") {
      var unit = action.lookup("unit");
      if (unit && (unit.match(/^machine-[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]\.slice$/) ||
                   unit.match(/^systemd-nspawn@[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]\.service$/)))
        return polkit.Result.YES;
    }
    if (action.id == "org.freedesktop.systemd1.reload-daemon")
      return polkit.Result.YES;
    if (action.id == "org.freedesktop.machine1.manage-machines") {
      var name = action.lookup("name");
      if (name && name.match(/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$/))
        return polkit.Result.YES;
    }
    if (action.id == "org.freedesktop.machine1.shell") {
      var name = action.lookup("name");
      if (name && name.match(/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$/))
        return polkit.Result.YES;
    }
    if (action.id == "org.freedesktop.machine1.open-pty") {
      var name = action.lookup("name");
      if (name && name.match(/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$/))
        return polkit.Result.YES;
    }
    if (action.id == "org.freedesktop.machine1.login") {
      var name = action.lookup("name");
      if (name && name.match(/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$/))
        return polkit.Result.YES;
    }
  }
});
`

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "First-run system configuration (requires sudo)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if deploy.IsRootless() {
			return fmt.Errorf("setup requires root privileges. Run with: sudo lxspawn setup")
		}

		var dirsCreated []string
		var groupCreated bool
		var polkitInstalled bool
		var sudoersInstalled bool
		var failures []string

		if err := ensureGroup(); err != nil {
			failures = append(failures, fmt.Sprintf("group creation: %v", err))
			logger.Warn(fmt.Sprintf("group creation: %v", err))
		} else {
			groupCreated = true
		}

		dirs := []struct {
			path string
			desc string
		}{
			{deploy.LxspawnDataDir, "machine data"},
			{filepath.Join(deploy.LxspawnDataDir, "cache"), "download cache"},
			{deploy.MachinesDir, "machine rootfs"},
			{deploy.NspawnDir, "container configs"},
			{deploy.SystemdDir, "systemd units"},
			{filepath.Join(deploy.SystemdDir, "systemd-nspawn@.service.d"), "drop-in overrides"},
		}

		for _, d := range dirs {
			if err := ensureDir(d.path); err != nil {
				return fmt.Errorf("failed to create %s (%s): %w", d.path, d.desc, err)
			}
			dirsCreated = append(dirsCreated, d.path)
			logger.Info(fmt.Sprintf("Created %s (%s)", d.path, d.desc))
		}

		if err := installPolkitRule(); err != nil {
			failures = append(failures, fmt.Sprintf("polkit rule: %v", err))
			logger.Warn(fmt.Sprintf("polkit rule: %v", err))
		} else {
			polkitInstalled = true
			logger.Info("Installed polkit rule")
		}

		if err := installSudoersRules(); err != nil {
			failures = append(failures, fmt.Sprintf("sudoers rules: %v", err))
			logger.Warn(fmt.Sprintf("sudoers rules: %v", err))
		} else {
			sudoersInstalled = true
			logger.Info("Installed sudoers rules")
		}

		if err := reloadPolkit(); err != nil {
			failures = append(failures, fmt.Sprintf("polkit reload: %v", err))
			logger.Warn(fmt.Sprintf("polkit reload: %v", err))
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.SetupData{
				GroupCreated:     groupCreated,
				DirsCreated:      dirsCreated,
				PolkitInstalled:  polkitInstalled,
				SudoersInstalled: sudoersInstalled,
				Success:          len(failures) == 0,
			})
		}

		if len(failures) > 0 {
			fmt.Println()
			fmt.Println("Setup completed with errors:")
			for _, f := range failures {
				fmt.Printf("  - %s\n", f)
			}
			fmt.Println()
			fmt.Println("Without sudoers and polkit rules, lxspawn commands will require a password.")
			fmt.Println("Please fix the above issues and re-run 'sudo lxspawn setup'.")
			return fmt.Errorf("setup completed with %d error(s)", len(failures))
		}

		logger.Success("Setup complete.")
		fmt.Println()
		fmt.Println("Add users to the lxspawn group:")
		fmt.Println("  sudo usermod -aG lxspawn <username>")
		fmt.Println("Users must log out and back in for group changes to take effect.")

		return nil
	},
}

func ensureGroup() error {
	if _, err := exec.LookPath("groupadd"); err != nil {
		return fmt.Errorf("groupadd not found")
	}

	cmd := exec.Command("groupadd", "-f", "lxspawn")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("groupadd failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0775); err != nil {
		return err
	}

	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	cmd := exec.Command("chown", currentUser.Uid+":lxspawn", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	cmd = exec.Command("chmod", "775", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chmod failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

func installPolkitRule() error {
	rulePath := "/etc/polkit-1/rules.d/50-lxspawn.rules"
	if err := os.MkdirAll(filepath.Dir(rulePath), 0755); err != nil {
		return err
	}
	return deploy.WriteFileAtomic(rulePath, []byte(polkitRule), 0644)
}

func findBinary(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

func installSudoersRules() error {
	sudoersPath := "/etc/sudoers.d/lxspawn"
	if err := os.MkdirAll(filepath.Dir(sudoersPath), 0755); err != nil {
		return err
	}

	// Check sudo version - regex arguments require sudo 1.9.10+
	sudoVersion, err := getSudoVersion()
	if err != nil {
		return fmt.Errorf("failed to check sudo version: %w", err)
	}
	if !sudoVersionSupportsRegex(sudoVersion) {
		return fmt.Errorf("sudo version %s does not support regex arguments (requires 1.9.10+). Please upgrade sudo.", sudoVersion)
	}

	var rules []string
	namePat := `[a-zA-Z0-9][a-zA-Z0-9_-]*`
	machinesDir := `/var/lib/machines/`

	rules = append(rules, "# Core operations (always needed)")
	if bin := findBinary("rm"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-rf %s%s$", bin, machinesDir, namePat))
	}
	if bin := findBinary("chown"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^root\\:lxspawn %s%s$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-R 0\\:0 %s%s$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^[0-9]+\\:[0-9]+ %s%s$", bin, machinesDir, namePat))
	}
	if bin := findBinary("chmod"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^0[0-7]{3} %s%s$", bin, machinesDir, namePat))
	}
	if bin := findBinary("mv"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^%s%s %s%s$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^%s%s %s%s\\.rollback-old$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^%s%s\\.rollback-temp %s%s$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^%s%s\\.rollback-old %s%s$", bin, machinesDir, namePat, machinesDir, namePat))
	}
	if bin := findBinary("mkdir"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-p %s%s$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-p %s%s\\.snapshots$", bin, machinesDir, namePat))
	}
	if bin := findBinary("systemd-run"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-M %s --pipe --wait -- /bin/bash -s$", bin, namePat))
	}
	if bin := findBinary("journalctl"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-M %s --no-pager -o short-iso$", bin, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-M %s -f$", bin, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^-u systemd-nspawn@%s\\.service -n [0-9]+ -o short-iso$", bin, namePat))
	}
	if bin := findBinary("systemctl"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^enable systemd-nspawn@%s\\.service$", bin, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^disable systemd-nspawn@%s\\.service$", bin, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^stop systemd-nspawn@%s\\.service$", bin, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^kill -s SIGKILL systemd-nspawn@%s\\.service$", bin, namePat))
	}

	if bin := findBinary("btrfs"); bin != "" {
		rules = append(rules, "")
		rules = append(rules, "# btrfs operations")
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume create %s%s$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume delete %s%s$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume delete %s%s\\.snapshots/[a-zA-Z0-9][a-zA-Z0-9_.-]*$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume delete %s%s\\.rollback-temp$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume delete %s%s\\.rollback-old$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume show %s%s$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume show %s%s\\.snapshots/[a-zA-Z0-9][a-zA-Z0-9_.-]*$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume list -o %s%s\\.snapshots$", bin, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume snapshot -r %s%s %s%s$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume snapshot -r %s%s %s%s\\.snapshots/[a-zA-Z0-9][a-zA-Z0-9_.-]*$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume snapshot %s%s %s%s$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume snapshot %s%s\\.snapshots/[a-zA-Z0-9][a-zA-Z0-9_.-]* %s%s\\.rollback-temp$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^subvolume snapshot %s%s %s%s\\.rollback-temp$", bin, machinesDir, namePat, machinesDir, namePat))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s ^filesystem usage -b %s$", bin, machinesDir))
	}

	content := strings.Join(rules, "\n") + "\n"
	
	// Validate sudoers syntax before installing
	tmpFile, err := os.CreateTemp("", "lxspawn-sudoers-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp sudoers file: %w", err)
	}
	tmpFile.Close()
	
	// Validate with visudo
	cmd := exec.Command("visudo", "-cf", tmpPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sudoers syntax validation failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	
	return deploy.WriteFileAtomic(sudoersPath, []byte(content), 0440)
}

func getSudoVersion() (string, error) {
	cmd := exec.Command("sudo", "-V")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Parse first line: "Sudo version 1.9.17p2"
	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no output from sudo -V")
	}
	parts := strings.Fields(lines[0])
	if len(parts) < 3 {
		return "", fmt.Errorf("unexpected sudo version format: %s", lines[0])
	}
	return parts[2], nil
}

func sudoVersionSupportsRegex(version string) bool {
	// Sudo 1.9.10+ supports regex arguments
	// Parse version string like "1.9.17p2"
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	major := 0
	minor := 0
	patch := 0
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	// Extract numeric part from patch (e.g., "17p2" -> 17)
	patchStr := strings.TrimRight(parts[2], "pP0123456789")
	if patchStr == "" {
		patchStr = parts[2]
	}
	fmt.Sscanf(patchStr, "%d", &patch)
	
	if major > 1 {
		return true
	}
	if major == 1 && minor > 9 {
		return true
	}
	if major == 1 && minor == 9 && patch >= 10 {
		return true
	}
	return false
}

func reloadPolkit() error {
	cmd := exec.Command("systemctl", "reload", "polkit")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("polkit reload failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

var setupNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Configure network interface for containers",
	Long: `Detect and select the primary network interface for container networking.
This is required for adaptive network mode and macvlan/ipvlan configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if deploy.IsRootless() {
			return fmt.Errorf("setup network requires root privileges. Run with: sudo lxspawn setup network")
		}

		ifaces, err := network.ListPhysicalInterfaces()
		if err != nil {
			return fmt.Errorf("failed to list interfaces: %w", err)
		}

		if len(ifaces) == 0 {
			return fmt.Errorf("no physical network interfaces found")
		}

		fmt.Println("Detected interfaces:")
		for i, iface := range ifaces {
			marker := ""
			if iface.Primary {
				marker = " [primary]"
			}
			status := "down"
			if iface.Carrier {
				status = iface.IP
			}
			fmt.Printf("  %d. %s (%s, %s)%s\n", i+1, iface.Name, iface.Type, status, marker)
		}

		defaultChoice := 1
		for i, iface := range ifaces {
			if iface.Primary {
				defaultChoice = i + 1
				break
			}
		}

		fmt.Printf("\nSelect interface for containers [%d]: ", defaultChoice)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice := defaultChoice
		if input != "" {
			fmt.Sscanf(input, "%d", &choice)
		}

		if choice < 1 || choice > len(ifaces) {
			return fmt.Errorf("invalid selection: %d", choice)
		}

		selectedIface := ifaces[choice-1]

		cfg := &network.NetworkConfig{
			AutoInterface: selectedIface.Name,
			DetectedAt:    time.Now(),
		}

		if err := network.SaveNetworkConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		if output.IsJSONMode() {
			return output.PrintJSON(map[string]interface{}{
				"interface": selectedIface.Name,
				"type":      selectedIface.Type,
				"ip":        selectedIface.IP,
				"success":   true,
			})
		}

		logger.Success(fmt.Sprintf("Saved interface %s as auto-detected interface", selectedIface.Name))
		fmt.Println("\nYou can now use 'adaptive' network mode in your YAML configs:")
		fmt.Println("  network:")
		fmt.Println("    mode: adaptive")
		fmt.Printf("    interface: auto  # Will use %s\n", selectedIface.Name)

		return nil
	},
}

func init() {
	setupCmd.AddCommand(setupNetworkCmd)
}

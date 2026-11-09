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

		if err := ensureGroup(); err != nil {
			logger.Warn(fmt.Sprintf("group creation: %v", err))
		} else {
			groupCreated = true
		}

		dirs := []struct {
			path string
			desc string
		}{
			{deploy.LxspawnDataDir, "machine data"},
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
			logger.Warn(fmt.Sprintf("polkit rule: %v", err))
		} else {
			polkitInstalled = true
			logger.Info("Installed polkit rule")
		}

		if err := installSudoersRules(); err != nil {
			logger.Warn(fmt.Sprintf("sudoers rules: %v", err))
		} else {
			sudoersInstalled = true
			logger.Info("Installed sudoers rules")
		}

		if err := reloadPolkit(); err != nil {
			logger.Warn(fmt.Sprintf("polkit reload: %v", err))
		}

		if output.IsJSONMode() {
			return output.PrintJSON(&output.SetupData{
				GroupCreated:     groupCreated,
				DirsCreated:      dirsCreated,
				PolkitInstalled:  polkitInstalled,
				SudoersInstalled: sudoersInstalled,
				Success:          true,
			})
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
	return cmd.Run()
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

	var rules []string

	rules = append(rules, "# Core operations (always needed)")
	if bin := findBinary("rm"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s -rf /var/lib/machines/*", bin))
	}
	if bin := findBinary("chown"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s root\\:lxspawn /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s -R 0\\:0 /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s [0-9]*\\:[0-9]* /var/lib/machines/*", bin))
	}
	if bin := findBinary("chmod"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s [0-7]* /var/lib/machines/*", bin))
	}
	if bin := findBinary("mv"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s /var/lib/machines/* /var/lib/machines/*", bin))
	}
	if bin := findBinary("mkdir"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s -p /var/lib/machines/*", bin))
	}
	if bin := findBinary("nsenter"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s -t [0-9]* -m -u -i -n -p -- /bin/bash -c *", bin))
	}
	if bin := findBinary("journalctl"); bin != "" {
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s -u systemd-nspawn@*.service -n [0-9]* -o short-iso", bin))
	}

	if bin := findBinary("btrfs"); bin != "" {
		rules = append(rules, "")
		rules = append(rules, "# btrfs operations")
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume create /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume delete /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume delete /var/lib/machines/*/.snapshots/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume show /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume list -o /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume snapshot -r /var/lib/machines/* /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume snapshot -r /var/lib/machines/* /var/lib/machines/*/.snapshots/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume snapshot /var/lib/machines/* /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s subvolume snapshot /var/lib/machines/*/.snapshots/* /var/lib/machines/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s filesystem usage -b /var/lib/machines", bin))
	}

	if bin := findBinary("lvs"); bin != "" {
		rules = append(rules, "")
		rules = append(rules, "# LVM operations")
		if bin := findBinary("lvs"); bin != "" {
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s --noheadings *", bin))
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s */lxspawn-*", bin))
		}
		if bin := findBinary("lvcreate"); bin != "" {
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s --name lxspawn-* *", bin))
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s --snapshot --name lxspawn-* *", bin))
		}
		if bin := findBinary("lvconvert"); bin != "" {
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s --merge */lxspawn-*", bin))
		}
		if bin := findBinary("lvremove"); bin != "" {
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s -f */lxspawn-*", bin))
		}
		if bin := findBinary("mkfs.ext4"); bin != "" {
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s -q /dev/*/lxspawn-*", bin))
		}
		if bin := findBinary("mount"); bin != "" {
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s /dev/*/lxspawn-* /var/lib/machines/*", bin))
		}
		if bin := findBinary("umount"); bin != "" {
			rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s /var/lib/machines/*", bin))
		}
	}

	if bin := findBinary("zfs"); bin != "" {
		rules = append(rules, "")
		rules = append(rules, "# ZFS operations")
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s snapshot lxspawn/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s rollback -r lxspawn/*", bin))
		rules = append(rules, fmt.Sprintf("%%lxspawn ALL=(root) NOPASSWD: %s destroy lxspawn/*", bin))
	}

	content := strings.Join(rules, "\n") + "\n"
	return deploy.WriteFileAtomic(sudoersPath, []byte(content), 0440)
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

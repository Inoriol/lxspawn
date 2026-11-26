package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/kernel"
	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/storage"
)

type DoctorData struct {
	Checks []DoctorCheck `json:"checks"`
	Passed int           `json:"passed"`
	Failed int           `json:"failed"`
	OK     bool          `json:"ok"`
}

type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system configuration and dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		var checks []DoctorCheck
		passed := 0
		failed := 0

		checks = append(checks, checkLxspawnGroup())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		checks = append(checks, checkNspawn())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		checks = append(checks, checkMachinectl())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		checks = append(checks, checkPolkit())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		checks = append(checks, checkSudoers())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		checks = append(checks, checkStorageBackend())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		checks = append(checks, checkNftables())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		checks = append(checks, checkMachinesDir())
		if checks[len(checks)-1].Status == "pass" {
			passed++
		} else {
			failed++
		}

		data := &DoctorData{
			Checks: checks,
			Passed: passed,
			Failed: failed,
			OK:     failed == 0,
		}

		if output.IsJSONMode() {
			return output.PrintJSON(data)
		}

		fmt.Println("lxspawn doctor")
		fmt.Println("==============")
		for _, check := range checks {
			icon := "✓"
			if check.Status != "pass" {
				icon = "✗"
			}
			fmt.Printf("%s %s", icon, check.Name)
			if check.Message != "" {
				fmt.Printf(": %s", check.Message)
			}
			fmt.Println()
		}
		fmt.Println()
		if failed == 0 {
			fmt.Println("All checks passed!")
		} else {
			fmt.Printf("%d check(s) failed. Run 'sudo lxspawn setup' to fix issues.\n", failed)
		}

		return nil
	},
}

func checkLxspawnGroup() DoctorCheck {
	currentUser, err := user.Current()
	if err != nil {
		return DoctorCheck{Name: "lxspawn group", Status: "fail", Message: "could not get current user"}
	}

	groups, err := currentUser.GroupIds()
	if err != nil {
		return DoctorCheck{Name: "lxspawn group", Status: "fail", Message: "could not get groups"}
	}

	for _, gid := range groups {
		group, err := user.LookupGroupId(gid)
		if err != nil {
			continue
		}
		if group.Name == "lxspawn" {
			return DoctorCheck{Name: "lxspawn group", Status: "pass", Message: "user is member"}
		}
	}

	return DoctorCheck{Name: "lxspawn group", Status: "fail", Message: "user is not member. Run: sudo usermod -aG lxspawn $USER"}
}

func checkNspawn() DoctorCheck {
	if _, err := exec.LookPath("systemd-nspawn"); err != nil {
		return DoctorCheck{Name: "systemd-nspawn", Status: "fail", Message: "not found. Install systemd-container package"}
	}
	return DoctorCheck{Name: "systemd-nspawn", Status: "pass"}
}

func checkMachinectl() DoctorCheck {
	if _, err := exec.LookPath("machinectl"); err != nil {
		return DoctorCheck{Name: "machinectl", Status: "fail", Message: "not found. Install systemd-container package"}
	}
	return DoctorCheck{Name: "machinectl", Status: "pass"}
}

func checkPolkit() DoctorCheck {
	polkitFile := "/etc/polkit-1/rules.d/50-lxspawn.rules"
	if _, err := os.Stat(polkitFile); os.IsNotExist(err) {
		return DoctorCheck{Name: "polkit rules", Status: "fail", Message: "not installed. Run: sudo lxspawn setup"}
	}
	return DoctorCheck{Name: "polkit rules", Status: "pass"}
}

func checkSudoers() DoctorCheck {
	sudoersFile := "/etc/sudoers.d/lxspawn"
	if _, err := os.Stat(sudoersFile); os.IsNotExist(err) {
		return DoctorCheck{Name: "sudoers rules", Status: "fail", Message: "not installed. Run: sudo lxspawn setup"}
	}
	return DoctorCheck{Name: "sudoers rules", Status: "pass"}
}

func checkStorageBackend() DoctorCheck {
	backend, err := storage.DetectBackend(deploy.MachinesDirPath())
	if err != nil {
		return DoctorCheck{Name: "storage backend", Status: "fail", Message: fmt.Sprintf("detection failed: %v", err)}
	}

	backendType := string(backend.Type())
	if backendType == "directory" {
		return DoctorCheck{Name: "storage backend", Status: "warn", Message: "directory mode (no snapshots). Consider btrfs or LVM thin pool"}
	}

	return DoctorCheck{Name: "storage backend", Status: "pass", Message: backendType}
}

func checkNftables() DoctorCheck {
	if _, err := exec.LookPath("nft"); err != nil {
		return DoctorCheck{Name: "nftables", Status: "warn", Message: "not found. Required for private/zone network modes"}
	}
	return DoctorCheck{Name: "nftables", Status: "pass"}
}

func checkMachinesDir() DoctorCheck {
	if _, err := os.Stat(deploy.MachinesDirPath()); os.IsNotExist(err) {
		return DoctorCheck{Name: "machines directory", Status: "fail", Message: "not found. Run: sudo lxspawn setup"}
	}

	if kernel.IsBtrfs(deploy.MachinesDirPath()) {
		ok, _ := kernel.CheckKernelVersion(5, 12)
		if !ok {
			return DoctorCheck{Name: "machines directory", Status: "warn", Message: "btrfs requires kernel 5.12+ for idmapped mounts"}
		}
	}

	return DoctorCheck{Name: "machines directory", Status: "pass"}
}

func getSystemdVersionForDoctor() string {
	cmd := exec.Command("systemctl", "--version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

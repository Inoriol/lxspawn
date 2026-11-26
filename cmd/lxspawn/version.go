package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/kernel"
	"github.com/Inoriol/lxspawn/internal/output"
	"github.com/Inoriol/lxspawn/internal/storage"
)

type VersionData struct {
	Version         string `json:"version"`
	GoVersion       string `json:"go_version"`
	Platform        string `json:"platform"`
	SystemdVersion  string `json:"systemd_version,omitempty"`
	KernelVersion   string `json:"kernel_version,omitempty"`
	StorageBackend  string `json:"storage_backend,omitempty"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := &VersionData{
			Version:   version,
			GoVersion: runtime.Version(),
			Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		}

		if systemdVer, err := getSystemdVersion(); err == nil {
			data.SystemdVersion = systemdVer
		}

		if kernelVer, err := getKernelVersion(); err == nil {
			data.KernelVersion = kernelVer
		}

		if backend, err := storage.DetectBackend(deploy.MachinesDirPath()); err == nil {
			data.StorageBackend = string(backend.Type())
		}

		if output.IsJSONMode() {
			return output.PrintJSON(data)
		}

		fmt.Printf("lxspawn version %s\n", data.Version)
		fmt.Printf("  Go: %s\n", data.GoVersion)
		fmt.Printf("  Platform: %s\n", data.Platform)
		if data.SystemdVersion != "" {
			fmt.Printf("  systemd: %s\n", data.SystemdVersion)
		}
		if data.KernelVersion != "" {
			fmt.Printf("  Kernel: %s\n", data.KernelVersion)
		}
		if data.StorageBackend != "" {
			fmt.Printf("  Storage: %s\n", data.StorageBackend)
		}

		return nil
	},
}

func getSystemdVersion() (string, error) {
	cmd := exec.Command("systemctl", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("could not parse systemd version")
}

func getKernelVersion() (string, error) {
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func isBtrfs() bool {
	return kernel.IsBtrfs(deploy.MachinesDirPath())
}

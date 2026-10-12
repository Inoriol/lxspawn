package image

import (
	"fmt"
	"os"
	"path/filepath"
)

func ValidateInit(rootfs string) error {
	initPaths := []string{
		filepath.Join(rootfs, "sbin", "init"),
		filepath.Join(rootfs, "lib", "systemd", "systemd"),
		filepath.Join(rootfs, "usr", "lib", "systemd", "systemd"),
	}

	for _, p := range initPaths {
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() {
			return nil
		}
	}

	return fmt.Errorf("no init system found in rootfs (checked /sbin/init, /lib/systemd/systemd, /usr/lib/systemd/systemd). The image must contain systemd")
}

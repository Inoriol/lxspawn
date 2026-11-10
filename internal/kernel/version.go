package kernel

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

func GetVersion() (major, minor int, err error) {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return 0, 0, fmt.Errorf("uname -r failed: %w", err)
	}

	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(strings.TrimSpace(string(out)))
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("could not parse kernel version from %q", string(out))
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %w", err)
	}

	minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %w", err)
	}

	return major, minor, nil
}

func CheckKernelVersion(minMajor, minMinor int) (bool, error) {
	major, minor, err := GetVersion()
	if err != nil {
		return false, err
	}

	if major > minMajor {
		return true, nil
	}
	if major == minMajor && minor >= minMinor {
		return true, nil
	}
	return false, nil
}

func IsBtrfs(path string) bool {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return false
	}
	// Btrfs magic number: 0x9123683E
	return stat.Type == 0x9123683E
}

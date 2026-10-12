package kernel

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	systemdVersion     int
	systemdVersionOnce sync.Once
)

func SystemdVersion() (int, error) {
	var err error
	systemdVersionOnce.Do(func() {
		var out []byte
		out, err = exec.Command("systemctl", "--version").Output()
		if err != nil {
			return
		}

		re := regexp.MustCompile(`^systemd\s+(\d+)`)
		matches := re.FindStringSubmatch(strings.TrimSpace(string(out)))
		if len(matches) < 2 {
			err = fmt.Errorf("could not parse systemd version from %q", string(out))
			return
		}

		systemdVersion, err = strconv.Atoi(matches[1])
	})
	return systemdVersion, err
}

func SupportsPrivateUsersIdentity() bool {
	v, err := SystemdVersion()
	if err != nil {
		return false
	}
	return v >= 249
}

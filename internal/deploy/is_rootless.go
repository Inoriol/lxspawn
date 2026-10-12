package deploy

import "os"

func IsRootless() bool {
	return os.Getuid() != 0
}

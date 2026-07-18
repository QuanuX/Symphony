//go:build !darwin && !linux

package config

import (
	"fmt"
	"os"
)

func fileOwnerUID(os.FileInfo) (uint32, error) {
	return 0, fmt.Errorf("SSIAG ownership verification is unsupported on this operating system")
}

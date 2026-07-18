//go:build !darwin && !linux

package client

import (
	"fmt"
	"os"
)

func socketOwnerUID(os.FileInfo) (uint32, error) {
	return 0, fmt.Errorf("SSIAG socket ownership verification is unsupported on this operating system")
}

//go:build !darwin && !linux

package packageinstall

import (
	"fmt"
	"os"
)

func openInstallerSource(string) (*os.File, error) {
	return nil, fmt.Errorf("native Accordare installation is unsupported on this platform")
}

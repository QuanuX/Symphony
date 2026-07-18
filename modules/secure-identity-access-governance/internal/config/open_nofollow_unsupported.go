//go:build !darwin && !linux

package config

import (
	"fmt"
	"os"
)

func openNoFollow(string) (*os.File, error) {
	return nil, fmt.Errorf("secure SSIAG configuration loading is unsupported on this operating system")
}

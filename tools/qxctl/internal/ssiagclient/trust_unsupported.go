//go:build !darwin && !linux

package ssiagclient

import (
	"fmt"
	"os"
)

func openNoFollow(string) (*os.File, error) {
	return nil, fmt.Errorf("secure SSIAG configuration loading is unsupported on this operating system")
}

func fileOwnerUID(os.FileInfo) (uint32, error) {
	return 0, fmt.Errorf("SSIAG configuration ownership verification is unsupported on this operating system")
}

func socketOwnerUID(os.FileInfo) (uint32, error) {
	return 0, fmt.Errorf("SSIAG socket ownership verification is unsupported on this operating system")
}

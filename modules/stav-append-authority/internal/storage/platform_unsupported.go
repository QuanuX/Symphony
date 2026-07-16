//go:build !darwin && !linux

package storage

import (
	"fmt"
	"os"
)

func openRegularNoFollow(string, os.FileMode) (*os.File, bool, error) {
	return nil, false, fmt.Errorf("stav storage: unsupported operating system")
}

func createExclusiveRegularNoFollow(string, os.FileMode) (*os.File, error) {
	return nil, fmt.Errorf("stav storage: unsupported operating system")
}

func lockFile(*os.File) error   { return fmt.Errorf("stav storage: unsupported operating system") }
func unlockFile(*os.File) error { return fmt.Errorf("stav storage: unsupported operating system") }

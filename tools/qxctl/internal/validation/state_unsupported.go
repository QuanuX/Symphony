//go:build !darwin && !linux

package validation

import (
	"fmt"
	"os"
)

func canonicalStateRoot(string) (string, error) {
	return "", fmt.Errorf("validation state is supported on Linux and macOS only")
}

func (s *Store) withStateLock(string, bool, func(*os.File) error) error {
	return fmt.Errorf("validation state is supported on Linux and macOS only")
}

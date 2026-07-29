//go:build !darwin && !linux

package knowledgebinding

import (
	"fmt"
	"os"
)

func (s *Store) withStateLock(bool, func(*os.File) error) error {
	return fmt.Errorf("knowledge engine binding state is unsupported on this operating system")
}

func readRegistryFile(*os.File) ([]byte, bool, error) {
	return nil, false, fmt.Errorf("knowledge engine binding state is unsupported on this operating system")
}

func writeRegistry(*os.File, []byte) error {
	return fmt.Errorf("knowledge engine binding state is unsupported on this operating system")
}

//go:build !darwin && !linux

package knowledgelifecycle

import "fmt"

type runtimeDirectory struct{}

func (s *RuntimeStore) withRuntimeLock(bool, bool, func(runtimeDirectory) error) error {
	return fmt.Errorf("native lifecycle runtime state is unsupported on this operating system; use WSL or a remote supported TOPS node")
}

func readRuntimeFile(runtimeDirectory) ([]byte, bool, error) {
	return nil, false, fmt.Errorf("native lifecycle runtime state is unsupported")
}

func writeRuntimeFile(runtimeDirectory, []byte) error {
	return fmt.Errorf("native lifecycle runtime state is unsupported")
}

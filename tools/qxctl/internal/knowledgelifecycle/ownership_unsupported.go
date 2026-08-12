//go:build !darwin && !linux

package knowledgelifecycle

import "fmt"

func (s *OwnershipStore) snapshotOwnership() (OwnershipSnapshot, error) {
	return OwnershipSnapshot{}, fmt.Errorf("shared-root ownership is supported only on Linux and macOS")
}

func (s *OwnershipStore) withOwnershipLock(bool, func(int, bool, OwnershipRegistry) error) error {
	return fmt.Errorf("shared-root ownership is supported only on Linux and macOS")
}

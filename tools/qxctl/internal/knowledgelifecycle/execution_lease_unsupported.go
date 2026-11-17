//go:build !darwin && !linux

package knowledgelifecycle

import "fmt"

func (s *Store) WithExecutionLease(operation func() error) error {
	return fmt.Errorf("lifecycle execution leases are unsupported on this platform")
}

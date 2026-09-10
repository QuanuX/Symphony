//go:build !darwin && !linux

package scvstate

import "fmt"

func WithProjectionDocument(root, topsID, domain, graphID string, operation func(func() ([]byte, error), func([]byte, func() error) error) error) error {
	return fmt.Errorf("protected graph selection is unsupported on this platform")
}

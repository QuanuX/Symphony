//go:build !darwin && !linux

package shvpublicationstate

import "fmt"

func (s Store) withDirectory(operation func(func() ([]byte, error), func([]byte, func() error) error) error) error {
	return fmt.Errorf("protected SHV head storage requires a supported no-follow filesystem adapter")
}

//go:build !darwin && !linux

package snvstate

import "fmt"

func (s Store) withDirectory(operation func(func() ([]byte, error), func([]byte, func() error) error) error) error {
	return fmt.Errorf("protected SNV journal storage requires a supported no-follow filesystem adapter")
}

func (s Store) readDirectory(operation func([]byte) error) error {
	return fmt.Errorf("protected SNV journal observation requires a supported no-follow filesystem adapter")
}

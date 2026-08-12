//go:build !darwin && !linux

package knowledgelifecycle

import (
	"fmt"
	"os"
)

func (s *Store) withProfileLock(bool, func(*os.File) error) error {
	return fmt.Errorf("lifecycle profile state is unsupported on this operating system")
}

func readProfileFile(*os.File, string) ([]byte, bool, error) {
	return nil, false, fmt.Errorf("lifecycle profile state is unsupported on this operating system")
}

func readStateFile(*os.File, string, int64, string) ([]byte, bool, error) {
	return nil, false, fmt.Errorf("lifecycle state is unsupported on this operating system")
}

type listedProfileFile struct {
	name string
	data []byte
}

func listProfileFiles(*os.File) ([]listedProfileFile, error) {
	return nil, fmt.Errorf("lifecycle profile state is unsupported on this operating system")
}

func writeProfileFile(*os.File, string, []byte) error {
	return fmt.Errorf("lifecycle profile state is unsupported on this operating system")
}

func writeStateFile(*os.File, string, []byte, string) error {
	return fmt.Errorf("lifecycle state is unsupported on this operating system")
}

func removeProfileFile(*os.File, string) error {
	return fmt.Errorf("lifecycle profile state is unsupported on this operating system")
}

func removeStateFile(*os.File, string, string) error {
	return fmt.Errorf("lifecycle state is unsupported on this operating system")
}

//go:build !darwin && !linux

package provider

import (
	"fmt"
	"os"
)

func openProviderFile(string) (*os.File, error) {
	return nil, fmt.Errorf("provider trust is unsupported on this platform")
}
func fileOwner(os.FileInfo) (uint32, uint32, error) {
	return 0, 0, fmt.Errorf("provider ownership is unsupported on this platform")
}

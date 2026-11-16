//go:build !darwin && !linux

package packageretirement

import (
	"fmt"
	"io"
	"os"
)

func (*Lease) PublishFile(string, io.Reader, string, uint64, os.FileMode) (bool, error) {
	return false, fmt.Errorf("package retirement is unsupported on this platform")
}

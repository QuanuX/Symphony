//go:build !darwin && !linux

package knowledgeengine

import (
	"fmt"
	"os"
)

func openRelativeNoFollow(string, []string) (*os.File, error) {
	return nil, fmt.Errorf("secure local SKVI installation access is unsupported on this operating system")
}

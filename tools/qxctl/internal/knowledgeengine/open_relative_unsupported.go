//go:build !darwin && !linux

package knowledgeengine

import (
	"fmt"
	"os"
)

func validateTrustedInstallPrefix(string) error {
	return fmt.Errorf("trusted knowledge-engine installation access is unsupported on this operating system")
}

func validateTrustedInstalledFile(*os.File) error {
	return fmt.Errorf("trusted knowledge-engine installation access is unsupported on this operating system")
}

func openTrustedRelativeNoFollow(string, []string) (*os.File, error) {
	return nil, fmt.Errorf("trusted knowledge-engine installation access is unsupported on this operating system")
}

func openRelativeNoFollow(string, []string) (*os.File, error) {
	return nil, fmt.Errorf("secure local SKVI installation access is unsupported on this operating system")
}

//go:build !darwin && !linux

package sovremote

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func processCancellation(cmd *exec.Cmd) { cmd.WaitDelay = 250 * time.Millisecond }
func cleanupProcess(cmd *exec.Cmd)      {}
func ReadConfiguration(path, expected string) (Configuration, string, error) {
	return Configuration{}, "", fmt.Errorf("SOV target configuration requires supported POSIX host")
}
func Configure(raw []byte) (ConfigurationResult, error) {
	return ConfigurationResult{}, fmt.Errorf("SOV target configuration requires supported POSIX host")
}
func RemoveConfiguration(path, expected string) error {
	return fmt.Errorf("SOV target configuration requires supported POSIX host")
}

func openExecutable(path string) (*os.File, error) { return os.Open(path) }

//go:build darwin || linux

package knowledgeengine

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// An engine invocation owns its process group. Cancellation terminates the
// leader and its descendants, including descendants retaining output pipes.
func configureEngineProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return terminateEngineProcessGroup(command) }
	// Bound pipe draining even when the leader exits before its descendants or
	// a descendant moves out of the group. Cleanup still kills the owned group.
	command.WaitDelay = 250 * time.Millisecond
}

func terminateEngineProcessGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return os.ErrProcessDone
	}
	err := unix.Kill(-command.Process.Pid, unix.SIGKILL)
	if errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}

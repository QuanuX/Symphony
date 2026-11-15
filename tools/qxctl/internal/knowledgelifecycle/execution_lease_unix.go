//go:build darwin || linux

package knowledgelifecycle

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// WithExecutionLease serializes qxctl apply/recovery workers across observation,
// prepare, external execution and finalize. The short native journal lock and
// profile snapshot lock alone do not cover that external-execution interval.
// This is a controller lease, not proof that package consumers have stopped.
func (s *Store) WithExecutionLease(operation func() error) error {
	directory, err := openProfileDirectory(s.stateRoot, s.topsID)
	if err != nil {
		return err
	}
	defer directory.Close()
	const name = ".execution.lock"
	fd, err := unix.Openat(int(directory.Fd()), name, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("open lifecycle execution lease: %w", err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	if err := validateOwnedRegular(fd); err != nil {
		return fmt.Errorf("validate lifecycle execution lease: %w", err)
	}
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil || status.Nlink != 1 {
		return fmt.Errorf("lifecycle execution lease must have exactly one link")
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("lifecycle execution is busy: %w", err)
	}
	defer unix.Flock(fd, unix.LOCK_UN)
	// Retain a stable inode: never unlink this file on completion or recovery.
	return operation()
}

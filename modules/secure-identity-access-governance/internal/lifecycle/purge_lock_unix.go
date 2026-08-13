//go:build darwin || linux

package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

type purgeSocketLease struct {
	file *os.File
}

// acquirePurgeSocketLease shares the server's persistent adjacent lock. A
// purge can therefore neither unlink a live endpoint nor race a supervised
// restart through stale-socket inspection.
func acquirePurgeSocketLease(socketPath string) (*purgeSocketLease, error) {
	if !filepath.IsAbs(socketPath) {
		return nil, fmt.Errorf("refusing non-absolute SSIAG socket path")
	}
	lockPath := socketPath + ".lock"
	fd, err := unix.Open(lockPath, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open SSIAG purge lifecycle lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), lockPath)
	fail := func(value error) (*purgeSocketLease, error) {
		_ = file.Close()
		return nil, value
	}
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return fail(fmt.Errorf("inspect SSIAG purge lifecycle lock: %w", err))
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return fail(fmt.Errorf("SSIAG purge lifecycle lock is not a regular file"))
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fail(fmt.Errorf("refusing to purge SSIAG while a process owns the socket lifecycle"))
		}
		return fail(fmt.Errorf("lock SSIAG purge lifecycle: %w", err))
	}
	return &purgeSocketLease{file: file}, nil
}

func (lease *purgeSocketLease) Close() error {
	if lease == nil || lease.file == nil {
		return nil
	}
	fd := int(lease.file.Fd())
	unlockErr := unix.Flock(fd, unix.LOCK_UN)
	closeErr := lease.file.Close()
	lease.file = nil
	return errors.Join(unlockErr, closeErr)
}

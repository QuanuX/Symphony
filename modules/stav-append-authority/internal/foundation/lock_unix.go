//go:build darwin || linux

package foundation

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type lifecycleLease struct{ file *os.File }

func acquireLifecycleLease(path string) (*lifecycleLease, error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open foundational lifecycle lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	fail := func(err error) (*lifecycleLease, error) { _ = file.Close(); return nil, err }
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return fail(err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Uid != uint32(os.Geteuid()) {
		return fail(fmt.Errorf("foundational lifecycle lock is unsafe"))
	}
	if err := unix.Fchmod(fd, 0o600); err != nil {
		return fail(err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fail(fmt.Errorf("another foundational lifecycle operation is active"))
		}
		return fail(err)
	}
	return &lifecycleLease{file: file}, nil
}

func (lease *lifecycleLease) Close() error {
	if lease == nil || lease.file == nil {
		return nil
	}
	fd := int(lease.file.Fd())
	err := errors.Join(unix.Flock(fd, unix.LOCK_UN), lease.file.Close())
	lease.file = nil
	return err
}

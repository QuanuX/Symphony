//go:build darwin || linux

package server

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type socketLease struct {
	file *os.File
}

func acquireSocketLease(socketPath string) (*socketLease, error) {
	lockPath := socketPath + ".lock"
	fd, err := unix.Open(lockPath, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open STAV socket lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), lockPath)
	closeOnError := func(err error) (*socketLease, error) {
		_ = file.Close()
		return nil, err
	}

	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return closeOnError(fmt.Errorf("inspect STAV socket lock: %w", err))
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return closeOnError(fmt.Errorf("STAV socket lock is not a regular file"))
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return closeOnError(fmt.Errorf("STAV socket lock owner uid=%d does not match authority uid=%d", stat.Uid, os.Geteuid()))
	}
	if err := unix.Fchmod(fd, 0o600); err != nil {
		return closeOnError(fmt.Errorf("restrict STAV socket lock: %w", err))
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return closeOnError(fmt.Errorf("another STAV process owns the socket lifecycle"))
		}
		return closeOnError(fmt.Errorf("lock STAV socket lifecycle: %w", err))
	}
	return &socketLease{file: file}, nil
}

func (lease *socketLease) Close() error {
	if lease == nil || lease.file == nil {
		return nil
	}
	fd := int(lease.file.Fd())
	unlockErr := unix.Flock(fd, unix.LOCK_UN)
	closeErr := lease.file.Close()
	lease.file = nil
	return errors.Join(unlockErr, closeErr)
}

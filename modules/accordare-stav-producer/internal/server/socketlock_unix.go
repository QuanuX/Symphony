//go:build darwin || linux

package server

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type socketLease struct{ file *os.File }

func acquireSocketLease(socketPath string) (*socketLease, error) {
	path := socketPath + ".lock"
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open Accordare socket lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	fail := func(err error) (*socketLease, error) { _ = file.Close(); return nil, err }
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return fail(fmt.Errorf("inspect Accordare socket lock: %w", err))
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Uid != uint32(os.Geteuid()) {
		return fail(fmt.Errorf("Accordare socket lock is unsafe"))
	}
	if err := unix.Fchmod(fd, 0o600); err != nil {
		return fail(fmt.Errorf("restrict Accordare socket lock: %w", err))
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fail(fmt.Errorf("another Accordare producer owns the socket lifecycle"))
		}
		return fail(fmt.Errorf("lock Accordare socket lifecycle: %w", err))
	}
	return &socketLease{file: file}, nil
}

func (lease *socketLease) Close() error {
	if lease == nil || lease.file == nil {
		return nil
	}
	err := errors.Join(unix.Flock(int(lease.file.Fd()), unix.LOCK_UN), lease.file.Close())
	lease.file = nil
	return err
}

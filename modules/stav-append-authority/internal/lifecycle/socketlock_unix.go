//go:build darwin || linux

package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type socketLifecycleLease struct{ file *os.File }

func acquireSocketLifecycleLease(socketPath string) (*socketLifecycleLease, error) {
	path := socketPath + ".lock"
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open STAV socket lifecycle lock for purge: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	fail := func(err error) (*socketLifecycleLease, error) { _ = file.Close(); return nil, err }
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return fail(fmt.Errorf("inspect STAV socket lifecycle lock for purge: %w", err))
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Uid != uint32(os.Geteuid()) {
		return fail(fmt.Errorf("STAV socket lifecycle lock is unsafe for purge"))
	}
	if err := unix.Fchmod(fd, 0o600); err != nil {
		return fail(err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fail(fmt.Errorf("refusing purge while a STAV process owns the socket lifecycle"))
		}
		return fail(fmt.Errorf("lock STAV socket lifecycle for purge: %w", err))
	}
	return &socketLifecycleLease{file: file}, nil
}

func (lease *socketLifecycleLease) Close() error {
	if lease == nil || lease.file == nil {
		return nil
	}
	fd := int(lease.file.Fd())
	unlockErr := unix.Flock(fd, unix.LOCK_UN)
	closeErr := lease.file.Close()
	lease.file = nil
	return errors.Join(unlockErr, closeErr)
}

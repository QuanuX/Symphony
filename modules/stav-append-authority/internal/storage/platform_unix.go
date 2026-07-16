//go:build darwin || linux

package storage

import (
	"errors"
	"os"
	"syscall"
)

func openRegularNoFollow(path string, mode os.FileMode) (*os.File, bool, error) {
	_, statErr := os.Lstat(path)
	created := errors.Is(statErr, os.ErrNotExist)
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return nil, false, err
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		_ = file.Close()
		if err != nil {
			return nil, false, err
		}
		return nil, false, syscall.EINVAL
	}
	return file, created, nil
}

func createExclusiveRegularNoFollow(path string, mode os.FileMode) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func lockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

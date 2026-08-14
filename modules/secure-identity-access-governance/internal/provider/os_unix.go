//go:build darwin || linux

package provider

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func openProviderFile(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("convert provider file descriptor")
	}
	return file, nil
}

func fileOwner(info os.FileInfo) (uint32, uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("provider file ownership unavailable")
	}
	return stat.Uid, stat.Gid, nil
}

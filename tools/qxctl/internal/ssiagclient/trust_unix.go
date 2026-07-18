//go:build darwin || linux

package ssiagclient

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func openNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("convert SSIAG configuration descriptor")
	}
	return file, nil
}

func fileOwnerUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("SSIAG configuration has unsupported ownership metadata")
	}
	return stat.Uid, nil
}

func socketOwnerUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("SSIAG socket has unsupported ownership metadata")
	}
	return stat.Uid, nil
}

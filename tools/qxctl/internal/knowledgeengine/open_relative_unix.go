//go:build darwin || linux

package knowledgeengine

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func validateTrustedInstallPrefix(path string) error {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("installation prefix is not a no-follow directory")
	}
	status, ok := info.Sys().(*syscall.Stat_t)
	if !ok || (status.Uid != 0 && status.Uid != uint32(os.Geteuid())) {
		return fmt.Errorf("installation prefix owner is not trusted")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("installation prefix is writable by group or other")
	}
	return nil
}

func validateTrustedInstalledFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	status, ok := info.Sys().(*syscall.Stat_t)
	if !ok || (status.Uid != 0 && status.Uid != uint32(os.Geteuid())) {
		return fmt.Errorf("installed file owner is not trusted")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("installed file is writable by group or other")
	}
	return nil
}

func openRelativeNoFollow(root string, components []string) (*os.File, error) {
	return openRelativeNoFollowMode(root, components, false)
}

func openTrustedRelativeNoFollow(root string, components []string) (*os.File, error) {
	return openRelativeNoFollowMode(root, components, true)
}

func validateTrustedDescriptor(descriptor int) error {
	var status unix.Stat_t
	if err := unix.Fstat(descriptor, &status); err != nil {
		return err
	}
	if status.Uid != 0 && status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("installed path component owner is not trusted")
	}
	if status.Mode&0o022 != 0 {
		return fmt.Errorf("installed path component is writable by group or other")
	}
	return nil
}

func openRelativeNoFollowMode(root string, components []string, trusted bool) (*os.File, error) {
	if len(components) == 0 {
		return nil, fmt.Errorf("relative path has no components")
	}
	current, err := unix.Open(root, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	if trusted {
		if err := validateTrustedDescriptor(current); err != nil {
			_ = unix.Close(current)
			return nil, err
		}
	}
	for _, component := range components[:len(components)-1] {
		next, openErr := unix.Openat(
			current, component,
			unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY,
			0,
		)
		_ = unix.Close(current)
		if openErr != nil {
			return nil, openErr
		}
		if trusted {
			if err := validateTrustedDescriptor(next); err != nil {
				_ = unix.Close(next)
				return nil, err
			}
		}
		current = next
	}
	fileDescriptor, err := unix.Openat(
		current, components[len(components)-1],
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK,
		0,
	)
	_ = unix.Close(current)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fileDescriptor), components[len(components)-1])
	if file == nil {
		_ = unix.Close(fileDescriptor)
		return nil, fmt.Errorf("could not bind opened descriptor")
	}
	if trusted {
		if err := validateTrustedInstalledFile(file); err != nil {
			_ = file.Close()
			return nil, err
		}
	}
	return file, nil
}

//go:build darwin || linux

package knowledgeengine

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func openRelativeNoFollow(root string, components []string) (*os.File, error) {
	if len(components) == 0 {
		return nil, fmt.Errorf("relative path has no components")
	}
	current, err := unix.Open(root, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
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
		current = next
	}
	fileDescriptor, err := unix.Openat(
		current, components[len(components)-1],
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
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
	return file, nil
}

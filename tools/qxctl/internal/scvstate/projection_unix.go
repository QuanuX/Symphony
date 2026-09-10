//go:build darwin || linux

package scvstate

import (
	"fmt"
	"golang.org/x/sys/unix"
	"path/filepath"
	"strings"
)

// WithProjectionDocument reuses the protected file mechanics, not the source
// record schema. Graph selection has its own namespace and owner state.
func WithProjectionDocument(root, topsID, domain, graphID string, operation func(func() ([]byte, error), func([]byte, func() error) error) error) error {
	if _, err := New(root, topsID, domain, graphID); err != nil {
		return err
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return fmt.Errorf("graph state root must be a clean absolute descendant path")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	for _, part := range strings.Split(strings.TrimPrefix(root, "/"), "/") {
		next, e := openChild(fd, part, false)
		_ = unix.Close(fd)
		if e != nil {
			return e
		}
		fd = next
	}
	if err := privateDirectory(fd, false); err != nil {
		_ = unix.Close(fd)
		return err
	}
	key, err := knowledgeengineDigestIdentity(domain, graphID)
	if err != nil {
		_ = unix.Close(fd)
		return err
	}
	for _, part := range []string{"symphony", "qxctl", "scv", "graphs-v1", topsID, key} {
		next, e := openChild(fd, part, true)
		_ = unix.Close(fd)
		if e != nil {
			return e
		}
		fd = next
	}
	defer unix.Close(fd)
	lock, err := unix.Openat(fd, "projection.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0o600)
	if err != nil {
		return err
	}
	defer unix.Close(lock)
	if err := privateFile(lock); err != nil {
		return err
	}
	if err := unix.Flock(lock, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("graph store is busy: %w", err)
	}
	defer unix.Flock(lock, unix.LOCK_UN)
	return operation(func() ([]byte, error) { return readState(fd) }, func(data []byte, guard func() error) error { return writeStateGuarded(fd, data, guard) })
}

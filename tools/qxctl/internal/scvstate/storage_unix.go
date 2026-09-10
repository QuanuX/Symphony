//go:build darwin || linux

package scvstate

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// These no-follow/openat/flock/fsync mechanics follow the existing
// knowledgebinding adapter, with a separate namespace and hard-link rejection.
func (s Store) withDirectory(operation func(func() ([]byte, error), func([]byte, func() error) error) error) error {
	root := filepath.Clean(s.Root)
	if !filepath.IsAbs(root) || root == "/" || root != s.Root {
		return fmt.Errorf("source state root must be a clean absolute descendant path")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	for _, part := range strings.Split(strings.TrimPrefix(root, "/"), "/") {
		next, err := openChild(fd, part, false)
		_ = unix.Close(fd)
		if err != nil {
			return err
		}
		fd = next
	}
	if err := privateDirectory(fd, false); err != nil {
		_ = unix.Close(fd)
		return err
	}
	identity, _ := knowledgeengineDigestIdentity(s.Domain, s.SourceID)
	for _, part := range []string{"symphony", "qxctl", "scv", "sources-v1", s.TOPSID, identity} {
		next, err := openChild(fd, part, true)
		_ = unix.Close(fd)
		if err != nil {
			return err
		}
		fd = next
	}
	defer unix.Close(fd)
	lockfd, err := unix.Openat(fd, "source.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0o600)
	if err != nil {
		return fmt.Errorf("source lock: %w", err)
	}
	defer unix.Close(lockfd)
	if err := privateFile(lockfd); err != nil {
		return err
	}
	if err := unix.Flock(lockfd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("source store is busy or cannot be locked: %w", err)
	}
	defer unix.Flock(lockfd, unix.LOCK_UN)
	return operation(func() ([]byte, error) { return readState(fd) }, func(data []byte, guard func() error) error { return writeStateGuarded(fd, data, guard) })
}

func knowledgeengineDigestIdentity(domain, sourceID string) (string, error) {
	// Hashing is solely an unambiguous filesystem key, not source identity.
	digest, err := seal(map[string]any{"domain": domain, "source_id": sourceID})
	return strings.TrimPrefix(digest, "sha256:"), err
}

func openChild(parent int, name string, private bool) (int, error) {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return -1, fmt.Errorf("unsafe state path component")
	}
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if errors.Is(err, unix.ENOENT) {
		if err := unix.Mkdirat(parent, name, 0o700); err != nil && !errors.Is(err, unix.EEXIST) {
			return -1, err
		}
		if err := unix.Fsync(parent); err != nil {
			return -1, err
		}
		fd, err = unix.Openat(parent, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	}
	if err != nil {
		return -1, fmt.Errorf("open source state directory %q: %w", name, err)
	}
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	// Ancestors must be owner/root controlled; the conventional root-owned
	// sticky temporary directory is permitted, but selected state itself is private.
	if status.Uid != uint32(os.Geteuid()) && status.Uid != 0 || status.Mode&0o022 != 0 && !(status.Uid == 0 && status.Mode&unix.S_ISVTX != 0) {
		_ = unix.Close(fd)
		return -1, fmt.Errorf("untrusted source state ancestor")
	}
	if private {
		if err := privateDirectory(fd, true); err != nil {
			_ = unix.Close(fd)
			return -1, err
		}
	}
	return fd, nil
}
func privateDirectory(fd int, strict bool) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uint32(os.Geteuid()) || st.Mode&0o022 != 0 || strict && st.Mode&0o077 != 0 {
		return fmt.Errorf("source directory must be owned and private")
	}
	return nil
}
func privateFile(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o077 != 0 || st.Nlink != 1 {
		return fmt.Errorf("source state file must be private, owned, regular and singly linked")
	}
	return nil
}
func readState(directory int) ([]byte, error) {
	fd, err := unix.Openat(directory, "state.json", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "state.json")
	defer file.Close()
	if err := privateFile(fd); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxStoreBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxStoreBytes {
		return nil, fmt.Errorf("source store exceeds bound")
	}
	return data, nil
}
func writeState(directory int, data []byte) error {
	return writeStateGuarded(directory, data, nil)
}
func writeStateGuarded(directory int, data []byte, beforePublication func() error) error {
	// Validate an existing destination before replacing it: no symlink or hard
	// link should be silently repaired and thereby hide an unsafe state path.
	if _, err := readState(directory); err != nil {
		return err
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	name := ".state-" + hex.EncodeToString(random[:]) + ".tmp"
	fd, err := unix.Openat(directory, name, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), name)
	defer func() { _ = file.Close(); _ = unix.Unlinkat(directory, name, 0) }()
	if err := privateFile(fd); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if beforePublication != nil {
		if err := beforePublication(); err != nil {
			return fmt.Errorf("source publication authority no longer valid; retained intent is recoverable: %w", err)
		}
	}
	if err := unix.Renameat(directory, name, directory, "state.json"); err != nil {
		return err
	}
	if err := unix.Fsync(directory); err != nil {
		return fmt.Errorf("source commit durability uncertain; inspect/recover exact operation: %w", err)
	}
	return nil
}

//go:build darwin || linux

// Filesystem adapter follows the corpus store: no-follow descriptors, private
// singly-linked files, exclusive immutable publication and directory fsync.
package scvworkflow

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"strings"
)

func child(parent int, name string, private, create bool) (int, error) {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return -1, fmt.Errorf("unsafe workflow directory")
	}
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if create && errors.Is(err, unix.ENOENT) {
		if err = unix.Mkdirat(parent, name, 0o700); err != nil && !errors.Is(err, unix.EEXIST) {
			return -1, err
		}
		if err = unix.Fsync(parent); err != nil {
			return -1, err
		}
		fd, err = unix.Openat(parent, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	}
	if err != nil {
		return -1, err
	}
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	if st.Uid != uint32(os.Geteuid()) && st.Uid != 0 || st.Mode&0o022 != 0 && !(st.Uid == 0 && st.Mode&unix.S_ISVTX != 0) || private && (st.Uid != uint32(os.Geteuid()) || st.Mode&0o077 != 0) {
		_ = unix.Close(fd)
		return -1, fmt.Errorf("untrusted or nonprivate workflow directory")
	}
	// A prior mkdir may have become visible before its parent's fsync failed.
	// Reused owned ancestors receive the same durability barrier before descendants.
	if create {
		var parentStatus unix.Stat_t
		if err = unix.Fstat(parent, &parentStatus); err != nil {
			_ = unix.Close(fd)
			return -1, err
		}
		if parentStatus.Uid == uint32(os.Geteuid()) {
			if err = unix.Fsync(parent); err != nil {
				_ = unix.Close(fd)
				return -1, err
			}
		}
	}
	return fd, nil
}
func privateFile(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o077 != 0 || st.Nlink != 1 {
		return fmt.Errorf("workflow file must be owned, private, regular and singly linked")
	}
	return nil
}
func readFile(directory int, name string, limit int) ([]byte, error) {
	fd, err := unix.Openat(directory, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	if err = privateFile(fd); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if len(raw) > limit {
		return nil, fmt.Errorf("workflow file exceeds byte bound")
	}
	return raw, err
}
func publish(directory int, name string, raw []byte, noReplace bool) error {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	temporary := ".publish-" + hex.EncodeToString(random[:]) + ".tmp"
	fd, err := unix.Openat(directory, temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), temporary)
	defer func() { _ = file.Close(); _ = unix.Unlinkat(directory, temporary, 0) }()
	if err = privateFile(fd); err != nil {
		return err
	}
	if _, err = file.Write(raw); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if noReplace {
		err = renameExclusive(directory, temporary, name)
	} else {
		err = unix.Renameat(directory, temporary, directory, name)
	}
	if err != nil {
		return err
	}
	if err = unix.Fsync(directory); err != nil {
		return fmt.Errorf("workflow publication durability uncertain; recover exact operation: %w", err)
	}
	return nil
}

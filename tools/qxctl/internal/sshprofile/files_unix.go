//go:build darwin || linux

package sshprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var errTrust = errors.New("transport_profile.untrusted_store")

type directory struct {
	file  *os.File
	owner uint32
}

func metadata(fd int, owner uint32, kind uint32) error {
	var s unix.Stat_t
	if unix.Fstat(fd, &s) != nil || s.Uid != owner || uint32(s.Mode)&unix.S_IFMT != kind || s.Mode&0022 != 0 || (kind == unix.S_IFREG && s.Nlink != 1) {
		return errTrust
	}
	return nil
}
func openProtected(path string) (*directory, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, 0) {
		return nil, errTrust
	}
	fd, e := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if e != nil {
		return nil, errTrust
	}
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if metadata(fd, 0, unix.S_IFDIR) != nil {
			unix.Close(fd)
			return nil, errTrust
		}
		if part == "" {
			continue
		}
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		unix.Close(fd)
		if e != nil {
			return nil, errTrust
		}
		fd = next
	}
	if metadata(fd, 0, unix.S_IFDIR) != nil {
		unix.Close(fd)
		return nil, errTrust
	}
	return &directory{file: os.NewFile(uintptr(fd), path), owner: 0}, nil
}
func (d *directory) close() {
	if d != nil && d.file != nil {
		_ = d.file.Close()
	}
}
func leaf(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, "/\\\x00")
}
func (d *directory) child(name string) (*directory, error) {
	if !leaf(name) {
		return nil, errTrust
	}
	fd, e := unix.Openat(int(d.file.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if e != nil {
		return nil, e
	}
	if metadata(fd, d.owner, unix.S_IFDIR) != nil {
		unix.Close(fd)
		return nil, errTrust
	}
	return &directory{file: os.NewFile(uintptr(fd), name), owner: d.owner}, nil
}
func (d *directory) read(name string, max int64) ([]byte, error) {
	if !leaf(name) {
		return nil, errTrust
	}
	fd, e := unix.Openat(int(d.file.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	if metadata(fd, d.owner, unix.S_IFREG) != nil {
		return nil, errTrust
	}
	b, e := io.ReadAll(io.LimitReader(f, max+1))
	if e != nil || int64(len(b)) > max {
		return nil, errTrust
	}
	return b, nil
}
func (d *directory) lock(exclusive bool) (func(), error) {
	// The lock is provisioned once by host setup and is never renamed/deleted.
	flags := unix.O_RDONLY
	if exclusive {
		flags = unix.O_RDWR
	}
	fd, e := unix.Openat(int(d.file.Fd()), "LOCK", flags|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if e != nil {
		return nil, errTrust
	}
	if metadata(fd, d.owner, unix.S_IFREG) != nil {
		unix.Close(fd)
		return nil, errTrust
	}
	op := unix.LOCK_SH
	if exclusive {
		op = unix.LOCK_EX
	}
	if unix.Flock(fd, op|unix.LOCK_NB) != nil {
		unix.Close(fd)
		return nil, errors.New("transport_profile.busy")
	}
	return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = unix.Close(fd) }, nil
}
func (d *directory) names() ([]string, error) {
	// ReadDir on a fresh descriptor avoids reusing a directory enumeration offset.
	fd, e := unix.Openat(int(d.file.Fd()), ".", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), ".")
	defer f.Close()
	names, e := f.Readdirnames(-1)
	sort.Strings(names)
	return names, e
}
func (d *directory) digestFile(relative string) (string, string, bool, error) {
	parts := strings.Split(relative, "/")
	current := d
	for _, p := range parts[:len(parts)-1] {
		n, e := current.child(p)
		if current != d {
			current.close()
		}
		if e != nil {
			return "", "", false, e
		}
		current = n
	}
	if current != d {
		defer current.close()
	}
	name := parts[len(parts)-1]
	if !leaf(name) {
		return "", "", false, errTrust
	}
	fd, e := unix.Openat(int(current.file.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if e != nil {
		return "", "", false, e
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	if metadata(fd, d.owner, unix.S_IFREG) != nil {
		return "", "", false, errTrust
	}
	var before, after unix.Stat_t
	if unix.Fstat(fd, &before) != nil {
		return "", "", false, errTrust
	}
	h := sha256.New()
	n, e := io.Copy(h, f)
	if e != nil || unix.Fstat(fd, &after) != nil || before.Size != n || before.Size != after.Size || before.Ino != after.Ino || before.Mode != after.Mode {
		return "", "", false, errTrust
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), fmt.Sprint(n), before.Mode&0111 != 0, nil
}

//go:build darwin || linux

package snvstate

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

// Filesystem mechanics adapted from internal/shvpublicationstate/storage_unix.go.
// SNV native semantics and the private SNV namespace are separate; SHV files are unchanged.
// These no-follow/openat/flock/fsync mechanics follow the existing
// knowledgebinding adapter, with a separate namespace and hard-link rejection.
func (s Store) withDirectory(operation func(func() ([]byte, error), func([]byte, func() error) error) error) error {
	root := filepath.Clean(s.Root)
	if !filepath.IsAbs(root) || root == "/" || root != s.Root {
		return fmt.Errorf("head state root must be a clean absolute descendant path")
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
	parts := []string{"symphony", "qxctl", "snv", "evidence-v1"}
	if s.Kind == "view" {
		identity, err := knowledgeengineDigestIdentity(s.TOPSID, s.ViewID)
		if err != nil {
			_ = unix.Close(fd)
			return err
		}
		parts = []string{"symphony", "qxctl", "snv", "views-v1", s.TOPSID, identity}
	}
	for _, part := range parts {
		next, err := openChild(fd, part, true)
		_ = unix.Close(fd)
		if err != nil {
			return err
		}
		fd = next
	}
	defer unix.Close(fd)
	lockfd, err := unix.Openat(fd, "head.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0o600)
	if err != nil {
		return fmt.Errorf("head lock: %w", err)
	}
	defer unix.Close(lockfd)
	if err := privateFile(lockfd); err != nil {
		return err
	}
	if err := unix.Flock(lockfd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return Refusal("snv.state_busy", err)
		}
		return fmt.Errorf("head store is busy or cannot be locked: %w", err)
	}
	defer unix.Flock(lockfd, unix.LOCK_UN)
	return operation(func() ([]byte, error) { return readState(fd) }, func(data []byte, guard func() error) error { return writeStateGuarded(fd, data, guard) })
}

func knowledgeengineDigestIdentity(topsID, viewID string) (string, error) {
	// Hashing is solely an unambiguous filesystem key, not head identity.
	digest, err := seal(map[string]any{"tops_id": topsID, "owner_engine_id": "symphony-snv", "view_id": viewID})
	return strings.TrimPrefix(digest, "sha256:"), err
}

func openChild(parent int, name string, private bool) (int, error) {
	return openChildMode(parent, name, private, true)
}
func openChildMode(parent int, name string, private, create bool) (int, error) {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return -1, Refusal("snv.unsafe_state", nil)
	}
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if create && errors.Is(err, unix.ENOENT) {
		if err := unix.Mkdirat(parent, name, 0o700); err != nil && !errors.Is(err, unix.EEXIST) {
			return -1, err
		}
		if err := unix.Fsync(parent); err != nil {
			return -1, err
		}
		fd, err = unix.Openat(parent, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	}
	if err != nil {
		return -1, fmt.Errorf("open head state directory %q: %w", name, err)
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
		return -1, Refusal("snv.unsafe_state", nil)
	}
	if private {
		if err := privateDirectory(fd, true); err != nil {
			_ = unix.Close(fd)
			return -1, err
		}
	}
	return fd, nil
}

func (s Store) readDirectory(operation func([]byte) error) error {
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	for _, part := range strings.Split(strings.TrimPrefix(s.Root, "/"), "/") {
		next, err := openChildMode(fd, part, false, false)
		_ = unix.Close(fd)
		if errors.Is(err, unix.ENOENT) {
			return operation(nil)
		}
		if err != nil {
			return err
		}
		fd = next
	}
	if err := privateDirectory(fd, false); err != nil {
		_ = unix.Close(fd)
		return err
	}
	parts := []string{"symphony", "qxctl", "snv", "evidence-v1"}
	if s.Kind == "view" {
		identity, err := knowledgeengineDigestIdentity(s.TOPSID, s.ViewID)
		if err != nil {
			_ = unix.Close(fd)
			return err
		}
		parts = []string{"symphony", "qxctl", "snv", "views-v1", s.TOPSID, identity}
	}
	for _, part := range parts {
		next, err := openChildMode(fd, part, true, false)
		_ = unix.Close(fd)
		if errors.Is(err, unix.ENOENT) {
			return operation(nil)
		}
		if err != nil {
			return err
		}
		fd = next
	}
	defer unix.Close(fd)
	lockfd, err := unix.Openat(fd, "head.lock", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		// An absent journal may precede its first writer. Existing state without
		// the stable lock file cannot be safely observed.
		data, readErr := readState(fd)
		if readErr != nil {
			return readErr
		}
		if data != nil {
			return Refusal("snv.recovery_required", nil)
		}
		return operation(nil)
	}
	if err != nil {
		return err
	}
	defer unix.Close(lockfd)
	if err := privateFile(lockfd); err != nil {
		return err
	}
	if err := unix.Flock(lockfd, unix.LOCK_SH|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return Refusal("snv.state_busy", err)
		}
		return fmt.Errorf("SNV observation is busy or cannot be locked: %w", err)
	}
	defer unix.Flock(lockfd, unix.LOCK_UN)
	data, err := readState(fd)
	if err != nil {
		return err
	}
	return operation(data)
}
func privateDirectory(fd int, strict bool) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uint32(os.Geteuid()) || st.Mode&0o022 != 0 || strict && st.Mode&0o077 != 0 {
		return Refusal("snv.unsafe_state", nil)
	}
	return nil
}
func privateFile(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o077 != 0 || st.Nlink != 1 {
		return Refusal("snv.unsafe_state", nil)
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
		return nil, Refusal("snv.capacity_exceeded", nil)
	}
	return data, nil
}
func writeState(directory int, data []byte) error {
	return writeStateGuarded(directory, data, nil)
}
func writeStateGuarded(directory int, data []byte, beforePublication func() error) (result error) {
	defer func() {
		var known *BoundaryError
		if !errors.As(result, &known) && errors.Is(result, unix.ENOSPC) {
			result = Refusal("snv.capacity_exceeded", result)
		}
	}()
	// Validate an existing destination before replacing it: no symlink or hard
	// link should be silently repaired and thereby hide an unsafe state path.
	if _, err := readState(directory); err != nil {
		return err
	}
	if err := cleanupOwnedTemps(directory); err != nil {
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
		barrier("snv.before_guard")
		if err := beforePublication(); err != nil {
			return Refusal("snv.recovery_required", err)
		}
	}
	if beforePublication != nil {
		barrier("snv.before_rename")
	}
	if err := unix.Renameat(directory, name, directory, "state.json"); err != nil {
		return err
	}
	if beforePublication != nil {
		barrier("snv.after_rename")
	}
	if err := unix.Fsync(directory); err != nil {
		return Refusal("snv.recovery_required", err)
	}
	return nil
}

// Only a mutation/recovery path removes the writer's private interrupted temp
// artifacts. Selected heads, immutable evidence and unrelated files are retained.
func cleanupOwnedTemps(directory int) error {
	copyFD, err := unix.Dup(directory)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(copyFD), "SNV journal directory")
	defer f.Close()
	names, err := f.Readdirnames(514)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(names) > 513 {
		return Refusal("snv.capacity_exceeded", nil)
	}
	changed := false
	for _, name := range names {
		if !strings.HasPrefix(name, ".state-") || !strings.HasSuffix(name, ".tmp") {
			continue
		}
		body := strings.TrimSuffix(strings.TrimPrefix(name, ".state-"), ".tmp")
		if len(body) != 32 {
			continue
		}
		if _, err := hex.DecodeString(body); err != nil || body != strings.ToLower(body) {
			continue
		}
		fd, err := unix.Openat(directory, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		err = privateFile(fd)
		_ = unix.Close(fd)
		if err != nil {
			return err
		}
		if err := unix.Unlinkat(directory, name, 0); err != nil {
			return err
		}
		changed = true
	}
	if changed {
		return unix.Fsync(directory)
	}
	return nil
}

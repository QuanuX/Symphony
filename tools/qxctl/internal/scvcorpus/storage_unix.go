//go:build darwin || linux

package scvcorpus

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

// Filesystem mechanics follow the existing scvstate no-follow/fsync adapter;
// this namespace has immutable blobs and bookkeeping, never an authority head.
type Session struct {
	directories  map[string]int
	jobDirectory int
}

func (s Store) With(operation string, work func(*Session, *Job) error) error {
	return s.with(operation, true, work)
}
func (s Store) WithRead(work func(*Session, *Job) error) error {
	return s.with("", false, work)
}
func (s Store) with(operation string, create bool, work func(*Session, *Job) error) error {
	if !filepath.IsAbs(s.Root) || filepath.Clean(s.Root) != s.Root || s.Root == "/" {
		return fmt.Errorf("corpus root must be a clean absolute descendant path")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	for _, part := range strings.Split(strings.TrimPrefix(s.Root, "/"), "/") {
		next, e := child(fd, part, false, create)
		_ = unix.Close(fd)
		if e != nil {
			return e
		}
		fd = next
	}
	for _, part := range []string{"symphony", "qxctl", "scv", "corpora-v1", s.TOPSID, s.Domain} {
		next, e := child(fd, part, true, create)
		_ = unix.Close(fd)
		if e != nil {
			return e
		}
		fd = next
	}
	defer unix.Close(fd)
	session := &Session{directories: map[string]int{}, jobDirectory: -1}
	defer func() {
		for _, directory := range session.directories {
			_ = unix.Close(directory)
		}
	}()
	for _, kind := range []string{"captures", "indexes", "snapshots"} {
		directory, e := child(fd, kind, true, create)
		if e != nil {
			return e
		}
		session.directories[kind] = directory
	}
	if operation == "" {
		return work(session, nil)
	}
	if !ValidID(operation) {
		return fmt.Errorf("invalid corpus operation ID")
	}
	jobs, err := child(fd, "jobs", true, create)
	if err != nil {
		return err
	}
	defer unix.Close(jobs)
	directory, err := child(jobs, keyForOperation(operation), true, create)
	if err != nil {
		return err
	}
	defer unix.Close(directory)
	session.jobDirectory = directory
	lock, err := unix.Openat(directory, "job.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0o600)
	if err != nil {
		return err
	}
	defer unix.Close(lock)
	if err = privateFile(lock); err != nil {
		return err
	}
	if err = unix.Flock(lock, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("corpus job busy: %w", err)
	}
	defer unix.Flock(lock, unix.LOCK_UN)
	raw, err := readFile(directory, "job.json", maxJobBytes)
	if err != nil {
		return err
	}
	if raw == nil {
		return work(session, nil)
	}
	job, err := validateJob(raw, operation)
	if err != nil {
		return err
	}
	return work(session, &job)
}
func (s *Session) Get(kind, digest string) ([]byte, error) {
	directory, ok := s.directories[kind]
	if !ok || !ValidDigest(digest) {
		return nil, fmt.Errorf("invalid corpus object key %s", digest)
	}
	raw, err := readFile(directory, strings.TrimPrefix(digest, "sha256:")+".json", MaxArtifactBytes)
	if err != nil {
		return nil, fmt.Errorf("read %s %s: %w", kind, digest, err)
	}
	if raw == nil {
		return nil, fmt.Errorf("missing %s %s", kind, digest)
	}
	if err = validateArtifact(kind, digest, raw); err != nil {
		return nil, err
	}
	return raw, nil
}
func (s *Session) Put(kind string, raw []byte) (string, error) {
	digest, err := objectDigest(raw)
	if err != nil {
		return "", err
	}
	if err = validateArtifact(kind, digest, raw); err != nil {
		return "", err
	}
	directory, ok := s.directories[kind]
	if !ok {
		return "", fmt.Errorf("unknown artifact kind")
	}
	name := strings.TrimPrefix(digest, "sha256:") + ".json"
	if existing, e := readFile(directory, name, MaxArtifactBytes); e != nil {
		return "", e
	} else if existing != nil {
		if e = validateArtifact(kind, digest, existing); e != nil {
			return "", e
		}
		if !SameRaw(existing, raw) {
			return "", fmt.Errorf("existing %s bytes disagree", digest)
		}
		return digest, unix.Fsync(directory)
	}
	err = publish(directory, name, raw, true)
	if errors.Is(err, unix.EEXIST) {
		existing, e := s.Get(kind, digest)
		if e != nil {
			return "", e
		}
		if !SameRaw(existing, raw) {
			return "", fmt.Errorf("concurrent %s bytes disagree", digest)
		}
		return digest, unix.Fsync(directory)
	}
	return digest, err
}
func SameRaw(a, b []byte) bool {
	x, xe := Decode(a)
	y, ye := Decode(b)
	return xe == nil && ye == nil && Same(x, y)
}
func (s *Session) Save(job Job) (Job, error) {
	if s.jobDirectory < 0 {
		return job, fmt.Errorf("corpus job not locked")
	}
	sealed, raw, err := sealJob(job)
	if err != nil {
		return job, err
	}
	if _, err = validateJob(raw, job.OperationID); err != nil {
		return job, err
	}
	previousRaw, err := readFile(s.jobDirectory, "job.json", maxJobBytes)
	if err != nil {
		return job, err
	}
	if previousRaw != nil {
		previous, err := validateJob(previousRaw, job.OperationID)
		if err != nil {
			return job, err
		}
		if previous.IntentDigest != job.IntentDigest || (previous.SnapshotTime != "" && previous.SnapshotTime != job.SnapshotTime) || (previous.SnapshotDigest != "" && previous.SnapshotDigest != job.SnapshotDigest) {
			return job, fmt.Errorf("corpus bookkeeping cannot replace pinned intent or completed result")
		}
		for id, checkpoint := range previous.Completed {
			if job.Completed[id] != checkpoint {
				return job, fmt.Errorf("corpus bookkeeping cannot replace completed member %s", id)
			}
		}
	}
	if err = publish(s.jobDirectory, "job.json", raw, false); err != nil {
		return job, err
	}
	return sealed, nil
}
func child(parent int, name string, private, create bool) (int, error) {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return -1, fmt.Errorf("unsafe corpus directory")
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
		return -1, fmt.Errorf("untrusted or nonprivate corpus directory")
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
		return fmt.Errorf("corpus file must be owned, private, regular and singly linked")
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
		return nil, fmt.Errorf("corpus file exceeds byte bound")
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
		return fmt.Errorf("corpus publication durability uncertain; recover exact operation: %w", err)
	}
	return nil
}

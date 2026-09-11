//go:build darwin || linux

package scvworkflow

import (
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Session struct {
	namespaceDirectory int
	directories        map[string]int
	runDirectory       int
}

func (s Store) With(operation string, create bool, work func(*Session, *Run) error) error {
	if !filepath.IsAbs(s.Root) || filepath.Clean(s.Root) != s.Root || s.Root == "/" {
		return fmt.Errorf("workflow root must be a clean absolute descendant path")
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
	for _, part := range []string{"symphony", "qxctl", "scv", "workflows-v1", s.TOPSID} {
		next, e := child(fd, part, true, create)
		_ = unix.Close(fd)
		if e != nil {
			return e
		}
		fd = next
	}
	defer unix.Close(fd)
	session := &Session{directories: map[string]int{}, runDirectory: -1, namespaceDirectory: fd}
	defer func() {
		for _, d := range session.directories {
			_ = unix.Close(d)
		}
	}()
	for _, kind := range []string{"records", "payloads"} {
		d, e := child(fd, kind, true, create)
		if e != nil {
			return e
		}
		session.directories[kind] = d
	}
	if operation == "" {
		return work(session, nil)
	}
	if !scvcorpus.ValidID(operation) {
		return fmt.Errorf("invalid workflow operation ID")
	}
	runs, err := child(fd, "runs", true, create)
	if err != nil {
		return err
	}
	defer unix.Close(runs)
	directory, err := child(runs, keyForOperation(operation), true, create)
	if err != nil {
		return err
	}
	defer unix.Close(directory)
	session.runDirectory = directory
	flags := unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	if create {
		flags |= unix.O_CREAT
	}
	lock, err := unix.Openat(directory, "run.lock", flags, 0o600)
	if err != nil {
		return err
	}
	defer unix.Close(lock)
	if err = privateFile(lock); err != nil {
		return err
	}
	mode := unix.LOCK_SH
	if create {
		mode = unix.LOCK_EX
	}
	if err = unix.Flock(lock, mode|unix.LOCK_NB); err != nil {
		return fmt.Errorf("workflow run busy: %w", err)
	}
	defer unix.Flock(lock, unix.LOCK_UN)
	raw, err := readFile(directory, "run.json", MaxBytes)
	if err != nil {
		return err
	}
	if raw == nil {
		return work(session, nil)
	}
	run, err := ReadRun(raw, operation)
	if err != nil {
		return err
	}
	return work(session, &run)
}
func validateObject(kind, digest string, raw []byte) error {
	actual, err := Digest(raw)
	if err != nil {
		return err
	}
	if actual != digest {
		return fmt.Errorf("workflow object identity mismatch")
	}
	if kind == "records" {
		_, err = ReadRecord(raw)
		return err
	}
	if kind == "payloads" {
		m, e := Decode(raw)
		if e != nil {
			return e
		}
		if len(m) != 3 || m["protocol"] != "symphony.qxctl.scv-workflow-payload.v1" {
			return fmt.Errorf("invalid workflow payload wrapper")
		}
		v, ok := m["input"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing workflow payload input")
		}
		b, e := Canonical(v)
		if e != nil {
			return e
		}
		return validateInputSize(b)
	}
	return fmt.Errorf("unknown workflow object kind")
}
func validateInputSize(raw []byte) error {
	if len(raw) > MaxInputBytes {
		return fmt.Errorf("workflow stage input exceeds 1 MiB")
	}
	_, err := scvcorpus.Decode(raw)
	return err
}
func (s *Session) Get(kind, digest string) ([]byte, error) {
	d, ok := s.directories[kind]
	if !ok || !scvcorpus.ValidDigest(digest) {
		return nil, fmt.Errorf("invalid workflow object reference")
	}
	raw, err := readFile(d, strings.TrimPrefix(digest, "sha256:")+".json", MaxBytes)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("missing retained workflow %s %s", kind, digest)
	}
	if err = validateObject(kind, digest, raw); err != nil {
		return nil, err
	}
	return raw, nil
}
func (s *Session) Put(kind string, raw []byte) (string, error) {
	digest, err := Digest(raw)
	if err != nil {
		return "", err
	}
	if err = validateObject(kind, digest, raw); err != nil {
		return "", err
	}
	directory, ok := s.directories[kind]
	if !ok {
		return "", fmt.Errorf("unknown workflow object kind")
	}
	name := strings.TrimPrefix(digest, "sha256:") + ".json"
	if existing, e := readFile(directory, name, MaxBytes); e != nil {
		return "", e
	} else if existing != nil {
		if e = validateObject(kind, digest, existing); e != nil {
			return "", e
		}
		if !Same(existing, raw) {
			return "", fmt.Errorf("retained workflow object differs")
		}
		return digest, unix.Fsync(directory)
	}
	err = publish(directory, name, raw, true)
	if errors.Is(err, unix.EEXIST) {
		existing, e := s.Get(kind, digest)
		if e != nil {
			return "", e
		}
		if !Same(existing, raw) {
			return "", fmt.Errorf("concurrent workflow object differs")
		}
		return digest, unix.Fsync(directory)
	}
	return digest, err
}
func Same(a, b []byte) bool {
	x, xe := Decode(a)
	y, ye := Decode(b)
	return xe == nil && ye == nil && scvcorpus.Same(x, y)
}
func (s *Session) Save(run Run) (Run, error) {
	if s.runDirectory < 0 {
		return run, fmt.Errorf("workflow run not locked")
	}
	raw, err := Seal(run)
	if err != nil {
		return run, err
	}
	sealed, err := ReadRun(raw, run.OperationID)
	if err != nil {
		return run, err
	}
	previous, err := readFile(s.runDirectory, "run.json", MaxBytes)
	if err != nil {
		return run, err
	}
	if previous != nil {
		old, e := ReadRun(previous, run.OperationID)
		if e != nil {
			return run, e
		}
		if old.IntentDigest != run.IntentDigest || (old.Complete && !run.Complete) {
			return run, fmt.Errorf("cannot replace pinned workflow intent/completion")
		}
		for name, c := range old.Stages {
			n, ok := run.Stages[name]
			if !ok || n.InputRef != c.InputRef || (c.ResultRef != "" && c.ResultRef != n.ResultRef) {
				return run, fmt.Errorf("cannot replace pinned workflow stage %s", name)
			}
		}
	}
	if err = publish(s.runDirectory, "run.json", raw, false); err != nil {
		return run, err
	}
	return sealed, nil
}

// List scans bounded directory metadata, validates each selected envelope and
// returns digest-ordered pages. It never executes owners or loads native bodies
// into its result. A corpus query's member bound is unrelated to this page size.
func (s *Session) List(after string, limit int) ([]Record, *string, error) {
	if (after != "" && !scvcorpus.ValidDigest(after)) || limit < 1 || limit > MaxList {
		return nil, nil, fmt.Errorf("invalid artifact list cursor/limit")
	}
	fd, err := unix.Openat(s.directories["records"], ".", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}
	f := os.NewFile(uintptr(fd), "records")
	defer f.Close()
	names := []string{}
	count := 0
	for {
		entries, e := f.Readdirnames(256)
		for _, name := range entries {
			count++
			if count > 65536 {
				return nil, nil, fmt.Errorf("artifact directory exceeds scan bound of 65536 entries")
			}
			if strings.HasPrefix(name, ".publish-") {
				continue
			}
			digest := "sha256:" + strings.TrimSuffix(name, ".json")
			if !strings.HasSuffix(name, ".json") || !scvcorpus.ValidDigest(digest) {
				return nil, nil, fmt.Errorf("invalid artifact directory entry")
			}
			if digest > after {
				names = append(names, digest)
			}
		}
		if e != nil {
			if !errors.Is(e, io.EOF) {
				return nil, nil, e
			}
			break
		}
	}
	sort.Strings(names)
	var next *string
	if len(names) > limit {
		cursor := names[limit-1]
		next = &cursor
		names = names[:limit]
	}
	result := []Record{}
	for _, digest := range names {
		raw, e := s.Get("records", digest)
		if e != nil {
			return nil, nil, e
		}
		record, e := ReadRecord(raw)
		if e != nil {
			return nil, nil, e
		}
		record.Input = nil
		record.Artifact = nil
		result = append(result, record)
	}
	return result, next, nil
}

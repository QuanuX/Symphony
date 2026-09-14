//go:build darwin || linux

package shvtransfer

import (
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const Directory = "shv-transfer-journal"

type Journal struct {
	root, dir int
	rootPath  string
	Intent    Intent
	Events    []Event
}

func private(fd int, directory bool) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	kind, mode := uint32(unix.S_IFREG), uint32(0o600)
	if directory {
		kind = unix.S_IFDIR
		mode = 0o700
	}
	if uint32(st.Mode)&unix.S_IFMT != kind || uint32(st.Mode)&0o7777 != mode || st.Uid != uint32(os.Geteuid()) || (!directory && st.Nlink != 1) {
		return fmt.Errorf("transfer storage ownership/type/mode differs")
	}
	return nil
}
func ReadPrivate(dir int, name string) ([]byte, error) {
	fd, err := unix.Openat(dir, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	if err = private(fd, false); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil || len(raw) > MaxBytes {
		return nil, fmt.Errorf("transfer file unreadable or oversized")
	}
	return raw, nil
}
func (j *Journal) publish(name string, raw []byte) error {
	if err := j.checkIdentity(); err != nil {
		return err
	}
	// Fixed scratch file is never selected as progress. Under the exclusive lock
	// an interrupted scratch write may be safely replaced; final names are unique.
	old, err := ReadPrivate(j.dir, name)
	if err != nil {
		return err
	}
	if old != nil {
		if string(old) != string(raw) {
			return fmt.Errorf("immutable transfer record conflict")
		}
		return unix.Fsync(j.dir)
	}
	if _, err = ReadPrivate(j.dir, "pending.json"); err != nil {
		return err
	}
	if err = unix.Unlinkat(j.dir, "pending.json", 0); err != nil && !errors.Is(err, unix.ENOENT) {
		return err
	}
	fd, err := unix.Openat(j.dir, "pending.json", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), "pending.json")
	_, err = f.Write(raw)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = renameExclusive(j.dir, "pending.json", name); err != nil {
		return err
	}
	return unix.Fsync(j.dir)
}
func With(root string, create bool, wanted *Intent, work func(*Journal) error) error {
	if wanted != nil {
		raw, err := knowledgeengine.SCVCanonical(wanted)
		if err != nil {
			return err
		}
		if _, err = ReadIntent(raw); err != nil {
			return err
		}
		if wanted.TargetRoot != root {
			return fmt.Errorf("wanted target differs")
		}
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return fmt.Errorf("transfer root must be clean absolute")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return fmt.Errorf("transfer root must exist without symlinks")
	}
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	if err = private(fd, true); err != nil {
		return err
	}
	if create {
		names, e := boundedNames(fd, 6)
		if e != nil {
			return e
		}
		hasJournal := false
		for _, n := range names {
			if n == Directory {
				hasJournal = true
			}
		}
		if !hasJournal && len(names) != 0 {
			return fmt.Errorf("new transfer requires empty destination")
		}
		if err = unix.Mkdirat(fd, Directory, 0o700); err != nil && !errors.Is(err, unix.EEXIST) {
			return err
		}
		if err = unix.Fsync(fd); err != nil {
			return err
		}
	}
	dir, err := unix.Openat(fd, Directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(dir)
	if err = private(dir, true); err != nil {
		return err
	}
	flags := unix.O_RDWR | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
	if create {
		flags |= unix.O_CREAT
	}
	lock, err := unix.Openat(dir, "journal.lock", flags, 0o600)
	if err != nil {
		return err
	}
	defer unix.Close(lock)
	if err = private(lock, false); err != nil {
		return err
	}
	mode := unix.LOCK_SH
	if wanted != nil {
		mode = unix.LOCK_EX
	}
	if err = unix.Flock(lock, mode|unix.LOCK_NB); err != nil {
		return fmt.Errorf("transfer busy: %w", err)
	}
	defer unix.Flock(lock, unix.LOCK_UN)
	j := &Journal{root: fd, dir: dir, rootPath: root}
	raw, err := ReadPrivate(dir, "intent.json")
	if err != nil {
		return err
	}
	if raw == nil {
		if !create || wanted == nil {
			return fmt.Errorf("transfer intent absent")
		}
		entries, err := boundedNames(fd, 6)
		if err != nil {
			return err
		}
		if len(entries) != 1 || entries[0] != Directory {
			return fmt.Errorf("new transfer requires empty destination")
		}
		raw, err = Seal(*wanted)
		if err != nil {
			return err
		}
		if err = j.publish("intent.json", raw); err != nil {
			return err
		}
	}
	j.Intent, err = ReadIntent(raw)
	if err != nil {
		return err
	}
	if j.Intent.TargetRoot != root {
		return fmt.Errorf("journal target differs")
	}
	if wanted != nil && wanted.Digest != j.Intent.Digest {
		return fmt.Errorf("target reserved for a different exact transfer")
	}
	// Bound the whole directory, including scratch and foreign names. All immutable
	// event filenames must form the one contiguous chain read below.
	dup, err := unix.Openat(dir, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(dup), "journal")
	names, err := f.Readdirnames(70)
	f.Close()
	if err != nil && err != io.EOF {
		return err
	}
	if len(names) >= 70 {
		return fmt.Errorf("journal directory exceeds bound")
	}
	expected := map[string]bool{"intent.json": true, "journal.lock": true, "pending.json": true}
	for n := 0; n <= 65; n++ {
		name := fmt.Sprintf("event-%02d.json", n)
		raw, err = ReadPrivate(dir, name)
		if err != nil {
			return err
		}
		if raw == nil {
			break
		}
		e, err := ReadEvent(raw)
		if err != nil {
			return err
		}
		j.Events = append(j.Events, e)
		expected[name] = true
	}
	for _, name := range names {
		if !expected[name] {
			return fmt.Errorf("unexpected or nonadjacent transfer record: %s", name)
		}
	}
	if err = ValidateChain(j.Intent, j.Events); err != nil {
		return err
	}
	if wanted != nil {
		if err = unix.Fsync(dir); err != nil {
			return err
		}
		if err = unix.Fsync(fd); err != nil {
			return err
		}
	}
	return work(j)
}
func (j *Journal) Append(phase, id, result string) error {
	prev := j.Intent.Digest
	if len(j.Events) > 0 {
		prev = j.Events[len(j.Events)-1].Digest
	}
	e := Event{Protocol: "symphony.qxctl.shv-store-transfer-event.v1", TransferDigest: j.Intent.Digest, Sequence: len(j.Events), Previous: prev, Phase: phase, OperationID: id, ResultDigest: result}
	raw, err := Seal(e)
	if err != nil {
		return err
	}
	e, err = ReadEvent(raw)
	if err != nil {
		return err
	}
	events := append(append([]Event{}, j.Events...), e)
	if err = ValidateChain(j.Intent, events); err != nil {
		return err
	}
	if err = j.publish(fmt.Sprintf("event-%02d.json", e.Sequence), raw); err != nil {
		return err
	}
	j.Events = events
	return nil
}
func (j *Journal) CheckRoot() (bool, error) {
	if err := j.checkIdentity(); err != nil {
		return false, err
	}
	var opened, current unix.Stat_t
	if err := unix.Fstat(j.root, &opened); err != nil {
		return false, err
	}
	if err := unix.Lstat(j.rootPath, &current); err != nil {
		return false, err
	}
	resolved, err := filepath.EvalSymlinks(j.rootPath)
	if err != nil || resolved != j.rootPath || opened.Dev != current.Dev || opened.Ino != current.Ino {
		return false, fmt.Errorf("destination root identity changed")
	}
	// No external directory growth can be silently treated as transfer progress.
	dup, err := unix.Openat(j.root, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return false, err
	}
	f := os.NewFile(uintptr(dup), "target")
	names, err := f.Readdirnames(6)
	f.Close()
	if err != nil && err != io.EOF {
		return false, err
	}
	if len(names) > 4 {
		return false, fmt.Errorf("unexpected destination content")
	}
	database := false
	for _, name := range names {
		if name == Directory {
			continue
		}
		if name != "connector.lock" && name != "index.duckdb" && name != "index.duckdb.wal" {
			return false, fmt.Errorf("unexpected destination content: %s", name)
		}
		if strings.HasSuffix(name, "duckdb") {
			database = true
		}
	}
	return database, nil
}

func boundedNames(dir, limit int) ([]string, error) {
	fd, err := unix.Openat(dir, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "directory")
	defer f.Close()
	names, err := f.Readdirnames(limit)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(names) >= limit {
		return nil, fmt.Errorf("directory exceeds bound")
	}
	return names, nil
}

func (j *Journal) checkIdentity() error {
	var root, path, opened, current unix.Stat_t
	if err := unix.Fstat(j.root, &root); err != nil {
		return err
	}
	if err := unix.Lstat(j.rootPath, &path); err != nil {
		return err
	}
	if root.Dev != path.Dev || root.Ino != path.Ino {
		return fmt.Errorf("target root replaced")
	}
	if err := unix.Fstat(j.dir, &opened); err != nil {
		return err
	}
	if err := unix.Fstatat(j.root, Directory, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if opened.Dev != current.Dev || opened.Ino != current.Ino {
		return fmt.Errorf("journal directory replaced")
	}
	return nil
}

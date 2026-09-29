//go:build darwin || linux

package credential

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"golang.org/x/sys/unix"
)

type diskJournal struct {
	directory, lock *os.File
	name            string
	created         bool
}

func openJournalFile(j *Journal, requestID string, create bool) (journalFile, error) {
	if stav.ValidateRequestUUID(requestID) != nil {
		return nil, errJournal
	}
	directory, err := journalDirectory(j, create)
	if err != nil {
		return nil, errJournal
	}
	fail := func() (journalFile, error) { directory.Close(); return nil, errJournal }
	created := false
	flags := unix.O_RDWR | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
	fd, err := unix.Openat(int(directory.Fd()), requestID+".lock", flags, 0)
	if errors.Is(err, unix.ENOENT) && create {
		fd, err = createJournalLock(int(directory.Fd()), requestID+".lock")
		created = err == nil
		if errors.Is(err, unix.EEXIST) {
			fd, err = unix.Openat(int(directory.Fd()), requestID+".lock", flags, 0)
		}
	}
	if err != nil {
		return fail()
	}
	lock := os.NewFile(uintptr(fd), "credential-request-lock")
	if journalRegular(fd, true) != nil || (!created && unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB) != nil) {
		lock.Close()
		return fail()
	}
	if lock.Sync() != nil || directory.Sync() != nil {
		lock.Close()
		return fail()
	}
	return &diskJournal{directory: directory, lock: lock, name: requestID + ".json", created: created}, nil
}

// Publish a newly created lock only after it is exclusively held. Otherwise
// another process could win flock between creation and the creator's claim.
func createJournalLock(directory int, name string) (int, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return -1, errJournal
	}
	temporary := ".lock-" + hex.EncodeToString(entropy[:]) + ".tmp"
	fd, err := unix.Openat(directory, temporary, unix.O_CREAT|unix.O_EXCL|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return -1, err
	}
	keep := false
	defer func() {
		unix.Unlinkat(directory, temporary, 0)
		if !keep {
			unix.Close(fd)
		}
	}()
	if journalRegular(fd, true) != nil || unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB) != nil {
		return -1, errJournal
	}
	if err := unix.Linkat(directory, temporary, directory, name, 0); err != nil {
		return -1, err
	}
	if err := unix.Unlinkat(directory, temporary, 0); err != nil {
		return -1, err
	}
	keep = true
	return fd, nil
}

func journalDirectory(j *Journal, create bool) (*os.File, error) {
	path := j.root
	// Permit only the documented macOS system aliases, never arbitrary symlinks.
	for alias, target := range map[string]string{"/var": "/private/var", "/tmp": "/private/tmp", "/etc": "/private/etc"} {
		if path == alias || strings.HasPrefix(path, alias+"/") {
			if resolved, err := filepath.EvalSymlinks(alias); err == nil && resolved == target {
				path = target + strings.TrimPrefix(path, alias)
			}
			break
		}
	}
	components := strings.Split(strings.TrimPrefix(path, "/"), "/")
	current, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, errJournal
	}
	for i, c := range components {
		if c == "" || c == "." || c == ".." {
			unix.Close(current)
			return nil, errJournal
		}
		next, err := unix.Openat(current, c, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		unix.Close(current)
		if err != nil {
			return nil, errJournal
		}
		current = next
		if journalDirectoryMode(current, i == len(components)-1) != nil {
			unix.Close(current)
			return nil, errJournal
		}
	}
	for _, c := range []string{"credential-requests", j.topsID} {
		var err error
		if create {
			err = unix.Mkdirat(current, c, 0700)
		}
		if err != nil && !errors.Is(err, unix.EEXIST) {
			unix.Close(current)
			return nil, errJournal
		}
		if unix.Fsync(current) != nil {
			unix.Close(current)
			return nil, errJournal
		}
		next, err := unix.Openat(current, c, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		unix.Close(current)
		if err != nil {
			return nil, errJournal
		}
		current = next
		if journalDirectoryMode(current, true) != nil {
			unix.Close(current)
			return nil, errJournal
		}
	}
	return os.NewFile(uintptr(current), "credential-journal"), nil
}
func journalDirectoryMode(fd int, private bool) error {
	var s unix.Stat_t
	if unix.Fstat(fd, &s) != nil || s.Mode&unix.S_IFMT != unix.S_IFDIR {
		return errJournal
	}
	if private {
		if s.Uid != uint32(os.Geteuid()) || s.Mode&07777 != 0700 {
			return errJournal
		}
		return nil
	}
	if s.Uid != 0 && s.Uid != uint32(os.Geteuid()) {
		return errJournal
	}
	if s.Mode&0022 != 0 && !(s.Uid == 0 && s.Mode&unix.S_ISVTX != 0) {
		return errJournal
	}
	return nil
}
func journalRegular(fd int, lock bool) error {
	var s unix.Stat_t
	if unix.Fstat(fd, &s) != nil || s.Mode&unix.S_IFMT != unix.S_IFREG || s.Uid != uint32(os.Geteuid()) || s.Mode&07777 != 0600 || s.Nlink != 1 {
		return errJournal
	}
	if lock && s.Size != 0 {
		return errJournal
	}
	return nil
}
func (s *diskJournal) close() { s.lock.Close(); s.directory.Close() }
func (s *diskJournal) read() ([]byte, bool, error) {
	fd, err := unix.Openat(int(s.directory.Fd()), s.name, unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		if !s.created {
			return nil, false, errJournal
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errJournal
	}
	f := os.NewFile(uintptr(fd), "credential-request")
	defer f.Close()
	if s.created {
		return nil, false, errJournal
	} // Existing state without its stable lock is inconsistent.
	if journalRegular(fd, false) != nil {
		return nil, false, errJournal
	}
	data, err := io.ReadAll(io.LimitReader(f, maxJournalRecordBytes+1))
	if err != nil || len(data) > maxJournalRecordBytes {
		return nil, false, errJournal
	}
	// A recovered observation must also establish directory-entry durability.
	if f.Sync() != nil || s.directory.Sync() != nil {
		return nil, false, errJournal
	}
	return data, true, nil
}
func (s *diskJournal) write(data []byte) error {
	if len(data) > maxJournalRecordBytes {
		return errJournal
	}
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return errJournal
	}
	temporary := "." + s.name + "." + hex.EncodeToString(entropy[:]) + ".tmp"
	fd, err := unix.Openat(int(s.directory.Fd()), temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return errJournal
	}
	f := os.NewFile(uintptr(fd), "credential-request-temporary")
	defer func() { f.Close(); unix.Unlinkat(int(s.directory.Fd()), temporary, 0) }()
	if journalRegular(fd, false) != nil {
		return errJournal
	}
	if n, err := f.Write(data); err != nil || n != len(data) {
		return errJournal
	}
	if f.Sync() != nil {
		return errJournal
	}
	if f.Close() != nil {
		return errJournal
	}
	if unix.Renameat(int(s.directory.Fd()), temporary, int(s.directory.Fd()), s.name) != nil {
		return errJournal
	}
	if s.directory.Sync() != nil {
		return errJournal
	}
	return nil
}

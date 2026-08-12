//go:build darwin || linux

package knowledgelifecycle

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func (s *Store) withProfileLock(exclusive bool, operation func(*os.File) error) error {
	directory, err := openProfileDirectory(s.stateRoot, s.topsID)
	if err != nil {
		return err
	}
	defer directory.Close()
	lock, err := acquireProfileLock(directory, exclusive)
	if err != nil {
		return err
	}
	defer lock.Close()
	return operation(directory)
}

func openProfileDirectory(root, topsID string) (*os.File, error) {
	current, err := openStateRootNoFollow(root)
	if err != nil {
		return nil, fmt.Errorf("open lifecycle state root: %w", err)
	}
	if err := validateOwnedDirectory(current, false); err != nil {
		_ = unix.Close(current)
		return nil, fmt.Errorf("validate lifecycle state root: %w", err)
	}
	for _, component := range []string{"symphony", topsID, "qxctl", "knowledge", "lifecycle", "profiles"} {
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkdirErr := unix.Mkdirat(current, component, 0o700); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return nil, fmt.Errorf("create lifecycle state component %q: %w", component, mkdirErr)
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return nil, fmt.Errorf("durably create lifecycle state component %q: %w", component, syncErr)
			}
			next, openErr = unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return nil, fmt.Errorf("open lifecycle state component %q: %w", component, openErr)
		}
		if err := validateOwnedDirectory(next, true); err != nil {
			_ = unix.Close(next)
			return nil, fmt.Errorf("validate lifecycle state component %q: %w", component, err)
		}
		current = next
	}
	return os.NewFile(uintptr(current), filepath.Join(root, "symphony", topsID, "qxctl/knowledge/lifecycle/profiles")), nil
}

func openStateRootNoFollow(root string) (int, error) {
	clean := filepath.Clean(root)
	if !filepath.IsAbs(clean) || clean == string(os.PathSeparator) {
		return -1, fmt.Errorf("state root must be an absolute descendant path")
	}
	current, err := unix.Open(string(os.PathSeparator), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return -1, err
	}
	for _, component := range strings.Split(strings.TrimPrefix(clean, string(os.PathSeparator)), string(os.PathSeparator)) {
		if component == "" || component == "." || component == ".." {
			_ = unix.Close(current)
			return -1, fmt.Errorf("state root contains an unsafe component")
		}
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkdirErr := unix.Mkdirat(current, component, 0o700); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return -1, mkdirErr
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return -1, syncErr
			}
			next, openErr = unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return -1, openErr
		}
		current = next
	}
	return current, nil
}

func validateOwnedDirectory(fd int, forcePrivate bool) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFDIR || status.Uid != uint32(os.Geteuid()) || status.Mode&0o022 != 0 {
		return fmt.Errorf("state directory is not effective-user-owned and protected")
	}
	if forcePrivate {
		return unix.Fchmod(fd, 0o700)
	}
	return nil
}

type profileLock struct{ file *os.File }

func acquireProfileLock(directory *os.File, exclusive bool) (*profileLock, error) {
	fd, err := unix.Openat(int(directory.Fd()), "profiles.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lifecycle profile lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), "profiles.lock")
	closeWith := func(cause error) (*profileLock, error) {
		_ = file.Close()
		return nil, cause
	}
	if err := validateOwnedRegular(fd); err != nil {
		return closeWith(fmt.Errorf("validate lifecycle profile lock: %w", err))
	}
	if err := unix.Fchmod(fd, 0o600); err != nil {
		return closeWith(fmt.Errorf("restrict lifecycle profile lock: %w", err))
	}
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	if err := unix.Flock(fd, mode); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return closeWith(fmt.Errorf("lifecycle profile store is busy"))
		}
		return closeWith(fmt.Errorf("lock lifecycle profile store: %w", err))
	}
	return &profileLock{file: file}, nil
}

func (lock *profileLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	fd := int(lock.file.Fd())
	err := errors.Join(unix.Flock(fd, unix.LOCK_UN), lock.file.Close())
	lock.file = nil
	return err
}

func validateOwnedRegular(fd int) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFREG || status.Uid != uint32(os.Geteuid()) || status.Mode&0o077 != 0 {
		return fmt.Errorf("state file is not effective-user-owned and owner-only")
	}
	return nil
}

func readProfileFile(directory *os.File, profileID string) ([]byte, bool, error) {
	return readStateFile(directory, profileFileName(profileID), maxProfileBytes, "lifecycle profile")
}

func readStateFile(directory *os.File, name string, maximum int64, label string) ([]byte, bool, error) {
	fd, err := unix.Openat(int(directory.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("open %s: %w", label, err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	if err := validateOwnedRegular(fd); err != nil {
		return nil, false, fmt.Errorf("validate %s: %w", label, err)
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(data)) > maximum {
		return nil, false, fmt.Errorf("%s exceeds %d bytes", label, maximum)
	}
	return data, true, nil
}

type listedProfileFile struct {
	name string
	data []byte
}

func listProfileFiles(directory *os.File) ([]listedProfileFile, error) {
	duplicate, err := unix.Dup(int(directory.Fd()))
	if err != nil {
		return nil, fmt.Errorf("duplicate lifecycle profile directory: %w", err)
	}
	file := os.NewFile(uintptr(duplicate), "profiles")
	defer file.Close()
	entries, err := file.ReadDir(maxProfiles + 2)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("enumerate lifecycle profiles: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if name == "profiles.lock" || strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasPrefix(name, "profile-") || !strings.HasSuffix(name, ".json") || len(name) != len("profile-")+64+len(".json") {
			return nil, fmt.Errorf("lifecycle profile directory contains an unknown object")
		}
		names = append(names, name)
	}
	if len(names) > maxProfiles {
		return nil, fmt.Errorf("lifecycle profile count exceeds %d", maxProfiles)
	}
	sort.Strings(names)
	result := make([]listedProfileFile, 0, len(names))
	for _, name := range names {
		fd, err := unix.Openat(int(directory.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
		if err != nil {
			return nil, fmt.Errorf("open listed lifecycle profile: %w", err)
		}
		profileFile := os.NewFile(uintptr(fd), name)
		if err := validateOwnedRegular(fd); err != nil {
			_ = profileFile.Close()
			return nil, fmt.Errorf("validate listed lifecycle profile: %w", err)
		}
		data, err := io.ReadAll(io.LimitReader(profileFile, maxProfileBytes+1))
		_ = profileFile.Close()
		if err != nil || len(data) > maxProfileBytes {
			return nil, fmt.Errorf("read listed lifecycle profile failed or exceeded its bound")
		}
		result = append(result, listedProfileFile{name: name, data: data})
	}
	return result, nil
}

func writeProfileFile(directory *os.File, profileID string, data []byte) error {
	return writeStateFile(directory, profileFileName(profileID), data, "lifecycle profile")
}

func writeStateFile(directory *os.File, name string, data []byte, label string) error {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return fmt.Errorf("generate %s temporary name: %w", label, err)
	}
	temp := "." + name + ".tmp-" + hex.EncodeToString(random)
	fd, err := unix.Openat(int(directory.Fd()), temp, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create %s temporary file: %w", label, err)
	}
	file := os.NewFile(uintptr(fd), temp)
	cleanup := func() {
		_ = file.Close()
		_ = unix.Unlinkat(int(directory.Fd()), temp, 0)
	}
	if err := validateOwnedRegular(fd); err != nil {
		cleanup()
		return err
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write %s: %w", label, err)
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("durably write %s: %w", label, err)
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(int(directory.Fd()), temp, 0)
		return fmt.Errorf("close %s temporary file: %w", label, err)
	}
	if err := unix.Renameat(int(directory.Fd()), temp, int(directory.Fd()), name); err != nil {
		_ = unix.Unlinkat(int(directory.Fd()), temp, 0)
		return fmt.Errorf("atomically replace %s: %w", label, err)
	}
	if err := unix.Fsync(int(directory.Fd())); err != nil {
		return fmt.Errorf("durably commit %s directory: %w", label, err)
	}
	return nil
}

func removeProfileFile(directory *os.File, profileID string) error {
	return removeStateFile(directory, profileFileName(profileID), "lifecycle profile")
}

func removeStateFile(directory *os.File, name, label string) error {
	if err := unix.Unlinkat(int(directory.Fd()), name, 0); err != nil {
		return fmt.Errorf("remove %s: %w", label, err)
	}
	if err := unix.Fsync(int(directory.Fd())); err != nil {
		return fmt.Errorf("durably commit %s removal: %w", label, err)
	}
	return nil
}

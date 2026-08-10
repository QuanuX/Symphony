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
	"syscall"

	"golang.org/x/sys/unix"
)

type runtimeDirectory struct {
	file *os.File
	name string
}

func (s *RuntimeStore) withRuntimeLock(exclusive, create bool, operation func(runtimeDirectory) error) error {
	directory, exists, err := openRuntimeDirectory(s.stateRoot, s.topsID, create)
	if err != nil {
		return err
	}
	if !exists {
		return operation(runtimeDirectory{name: runtimeProfileFile(s.profileID)})
	}
	defer directory.Close()
	lock, err := acquireRuntimeLock(directory, exclusive, create)
	if err != nil {
		return err
	}
	if lock != nil {
		defer lock.Close()
	}
	return operation(runtimeDirectory{file: directory, name: runtimeProfileFile(s.profileID)})
}

func openRuntimeDirectory(root, topsID string, create bool) (*os.File, bool, error) {
	current, err := openStateRootNoFollow(root)
	if err != nil {
		return nil, false, fmt.Errorf("open lifecycle runtime root: %w", err)
	}
	if err := validateOwnedDirectory(current, false); err != nil {
		_ = unix.Close(current)
		return nil, false, err
	}
	for _, component := range []string{"symphony", topsID, "qxctl", "knowledge", "lifecycle", "runtime"} {
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) && !create {
			_ = unix.Close(current)
			return nil, false, nil
		}
		if errors.Is(openErr, syscall.ENOENT) && create {
			if mkdirErr := unix.Mkdirat(current, component, 0o700); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return nil, false, mkdirErr
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return nil, false, syncErr
			}
			next, openErr = unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return nil, false, openErr
		}
		if err := validateOwnedDirectory(next, true); err != nil {
			_ = unix.Close(next)
			return nil, false, err
		}
		current = next
	}
	return os.NewFile(uintptr(current), filepath.Join(root, "symphony", topsID, "qxctl/knowledge/lifecycle/runtime")), true, nil
}

type runtimeLock struct{ file *os.File }

func acquireRuntimeLock(directory *os.File, exclusive, create bool) (*runtimeLock, error) {
	flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	if create {
		flags = unix.O_CREAT | unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW
	}
	fd, err := unix.Openat(int(directory.Fd()), "runtime.lock", flags, 0o600)
	if errors.Is(err, syscall.ENOENT) && !create {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open lifecycle runtime lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), "runtime.lock")
	if err := validateOwnedRegular(fd); err != nil {
		_ = file.Close()
		return nil, err
	}
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	if err := unix.Flock(fd, mode); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock lifecycle runtime state: %w", err)
	}
	return &runtimeLock{file: file}, nil
}

func (lock *runtimeLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	err := errors.Join(unix.Flock(int(lock.file.Fd()), unix.LOCK_UN), lock.file.Close())
	lock.file = nil
	return err
}

func readRuntimeFile(directory runtimeDirectory) ([]byte, bool, error) {
	if directory.file == nil {
		return nil, false, nil
	}
	fd, err := unix.Openat(int(directory.file.Fd()), directory.name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	file := os.NewFile(uintptr(fd), "runtime-state")
	defer file.Close()
	if err := validateOwnedRegular(fd); err != nil {
		return nil, false, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRuntimeBytes+1))
	if err != nil || len(data) > maxRuntimeBytes {
		return nil, false, fmt.Errorf("read lifecycle runtime state failed or exceeded its bound")
	}
	return data, true, nil
}

func writeRuntimeFile(directory runtimeDirectory, data []byte) error {
	if directory.file == nil {
		return fmt.Errorf("lifecycle runtime directory is absent")
	}
	name := directory.name
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	temporary := "." + name + ".tmp-" + hex.EncodeToString(random)
	fd, err := unix.Openat(int(directory.file.Fd()), temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), temporary)
	cleanup := func() {
		_ = file.Close()
		_ = unix.Unlinkat(int(directory.file.Fd()), temporary, 0)
	}
	if err := validateOwnedRegular(fd); err != nil {
		cleanup()
		return err
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(int(directory.file.Fd()), temporary, 0)
		return err
	}
	if err := unix.Renameat(int(directory.file.Fd()), temporary, int(directory.file.Fd()), name); err != nil {
		_ = unix.Unlinkat(int(directory.file.Fd()), temporary, 0)
		return err
	}
	return unix.Fsync(int(directory.file.Fd()))
}

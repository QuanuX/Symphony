//go:build darwin || linux

package knowledgebinding

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const registryFileName = "default.json"

func (s *Store) withStateLock(exclusive bool, operation func(*os.File) error) error {
	directory, err := openStateDirectory(s.stateRoot)
	if err != nil {
		return err
	}
	defer directory.Close()
	lock, err := acquireRegistryLock(directory, exclusive)
	if err != nil {
		return err
	}
	defer lock.Close()
	return operation(directory)
}

func openStateDirectory(root string) (*os.File, error) {
	rootFD, err := openStateRootNoFollow(root)
	if err != nil {
		return nil, fmt.Errorf("open knowledge binding state root: %w", err)
	}
	if err := validateOwnedDirectory(rootFD, false); err != nil {
		_ = unix.Close(rootFD)
		return nil, fmt.Errorf("validate knowledge binding state root: %w", err)
	}
	current := rootFD
	for _, component := range []string{"symphony", "qxctl", "knowledge", "engine-bindings"} {
		next, openErr := unix.Openat(
			current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkdirErr := unix.Mkdirat(current, component, 0o700); mkdirErr != nil &&
				!errors.Is(mkdirErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return nil, fmt.Errorf("create knowledge binding state component %q: %w", component, mkdirErr)
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return nil, fmt.Errorf("durably create knowledge binding state component %q: %w", component, syncErr)
			}
			next, openErr = unix.Openat(
				current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return nil, fmt.Errorf("open knowledge binding state component %q: %w", component, openErr)
		}
		if err := validateOwnedDirectory(next, true); err != nil {
			_ = unix.Close(next)
			return nil, fmt.Errorf("validate knowledge binding state component %q: %w", component, err)
		}
		current = next
	}
	return os.NewFile(uintptr(current), filepath.Join(root, "symphony/qxctl/knowledge/engine-bindings")), nil
}

func openStateRootNoFollow(root string) (int, error) {
	clean := filepath.Clean(root)
	if !filepath.IsAbs(clean) || clean == string(os.PathSeparator) {
		return -1, fmt.Errorf("state root must be an absolute descendant path")
	}
	current, err := unix.Open(
		string(os.PathSeparator),
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY,
		0,
	)
	if err != nil {
		return -1, err
	}
	relative := strings.TrimPrefix(clean, string(os.PathSeparator))
	for _, component := range strings.Split(relative, string(os.PathSeparator)) {
		if component == "" || component == "." || component == ".." {
			_ = unix.Close(current)
			return -1, fmt.Errorf("state root contains an unsafe component")
		}
		next, openErr := unix.Openat(
			current, component,
			unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY,
			0,
		)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkdirErr := unix.Mkdirat(current, component, 0o700); mkdirErr != nil &&
				!errors.Is(mkdirErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return -1, mkdirErr
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return -1, syncErr
			}
			next, openErr = unix.Openat(
				current, component,
				unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY,
				0,
			)
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
	if status.Mode&unix.S_IFMT != unix.S_IFDIR {
		return fmt.Errorf("state path is not a directory")
	}
	if status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("state directory owner uid=%d does not match effective uid=%d", status.Uid, os.Geteuid())
	}
	if status.Mode&0o022 != 0 {
		return fmt.Errorf("state directory is writable by group or other")
	}
	if forcePrivate {
		if err := unix.Fchmod(fd, 0o700); err != nil {
			return err
		}
	}
	return nil
}

type registryLock struct {
	file *os.File
}

func acquireRegistryLock(directory *os.File, exclusive bool) (*registryLock, error) {
	fd, err := unix.Openat(
		int(directory.Fd()), "registry.lock",
		unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open knowledge binding registry lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), "registry.lock")
	closeWith := func(cause error) (*registryLock, error) {
		_ = file.Close()
		return nil, cause
	}
	if err := validateOwnedRegular(fd); err != nil {
		return closeWith(fmt.Errorf("validate knowledge binding registry lock: %w", err))
	}
	if err := unix.Fchmod(fd, 0o600); err != nil {
		return closeWith(fmt.Errorf("restrict knowledge binding registry lock: %w", err))
	}
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	if err := unix.Flock(fd, mode); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return closeWith(fmt.Errorf("knowledge binding registry is busy"))
		}
		return closeWith(fmt.Errorf("lock knowledge binding registry: %w", err))
	}
	return &registryLock{file: file}, nil
}

func (lock *registryLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	fd := int(lock.file.Fd())
	unlockErr := unix.Flock(fd, unix.LOCK_UN)
	closeErr := lock.file.Close()
	lock.file = nil
	return errors.Join(unlockErr, closeErr)
}

func validateOwnedRegular(fd int) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFREG {
		return fmt.Errorf("state object is not a regular file")
	}
	if status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("state file owner uid=%d does not match effective uid=%d", status.Uid, os.Geteuid())
	}
	if status.Mode&0o077 != 0 {
		return fmt.Errorf("state file is accessible by group or other")
	}
	return nil
}

func readRegistryFile(directory *os.File) ([]byte, bool, error) {
	fd, err := unix.Openat(
		int(directory.Fd()), registryFileName,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("open knowledge binding registry: %w", err)
	}
	file := os.NewFile(uintptr(fd), registryFileName)
	defer file.Close()
	if err := validateOwnedRegular(fd); err != nil {
		return nil, false, fmt.Errorf("validate knowledge binding registry: %w", err)
	}
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil {
		return nil, false, fmt.Errorf("read knowledge binding registry: %w", err)
	}
	if len(data) > 1024*1024 {
		return nil, false, fmt.Errorf("knowledge binding registry exceeds 1048576 bytes")
	}
	return data, true, nil
}

func writeRegistry(directory *os.File, data []byte) error {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return fmt.Errorf("generate knowledge binding registry temporary name: %w", err)
	}
	tempName := "." + registryFileName + ".tmp-" + hex.EncodeToString(random)
	fd, err := unix.Openat(
		int(directory.Fd()), tempName,
		unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create knowledge binding registry temporary file: %w", err)
	}
	file := os.NewFile(uintptr(fd), tempName)
	cleanup := func() {
		_ = file.Close()
		_ = unix.Unlinkat(int(directory.Fd()), tempName, 0)
	}
	if err := validateOwnedRegular(fd); err != nil {
		cleanup()
		return fmt.Errorf("validate knowledge binding registry temporary file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write knowledge binding registry temporary file: %w", err)
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("durably write knowledge binding registry temporary file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(int(directory.Fd()), tempName, 0)
		return fmt.Errorf("close knowledge binding registry temporary file: %w", err)
	}
	if err := unix.Renameat(
		int(directory.Fd()), tempName, int(directory.Fd()), registryFileName); err != nil {
		_ = unix.Unlinkat(int(directory.Fd()), tempName, 0)
		return fmt.Errorf("atomically replace knowledge binding registry: %w", err)
	}
	if err := unix.Fsync(int(directory.Fd())); err != nil {
		return fmt.Errorf("durably commit knowledge binding registry directory: %w", err)
	}
	return nil
}

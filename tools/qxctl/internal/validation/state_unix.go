//go:build darwin || linux

package validation

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

type stateFile struct {
	name string
	data []byte
}

func canonicalStateRoot(root string) (string, error) {
	clean := filepath.Clean(root)
	if !filepath.IsAbs(clean) || clean == string(os.PathSeparator) {
		return "", fmt.Errorf("validation state root must be an absolute descendant path")
	}
	if info, err := os.Lstat(clean); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("validation state root must be a no-follow directory")
		}
		resolved, err := filepath.EvalSymlinks(clean)
		if err != nil {
			return "", err
		}
		return filepath.Clean(resolved), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect validation state root: %w", err)
	}
	return clean, nil
}

func (s *Store) withStateLock(area string, exclusive bool, operation func(*os.File) error) error {
	if area != "profiles" && area != "baselines" && area != "warnings" {
		return fmt.Errorf("unknown validation state area")
	}
	baseFD, areaFD, err := openValidationState(s.stateRoot, s.topsID, area)
	if err != nil {
		return err
	}
	base := os.NewFile(uintptr(baseFD), "validation")
	areaFile := os.NewFile(uintptr(areaFD), area)
	defer base.Close()
	defer areaFile.Close()
	lockFD, err := unix.Openat(int(base.Fd()), "validation.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("open validation state lock: %w", err)
	}
	lock := os.NewFile(uintptr(lockFD), "validation.lock")
	defer lock.Close()
	if err := validateOwnedRegular(lockFD); err != nil {
		return fmt.Errorf("validate validation state lock: %w", err)
	}
	if err := unix.Fchmod(lockFD, 0o600); err != nil {
		return fmt.Errorf("restrict validation state lock: %w", err)
	}
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	if err := unix.Flock(lockFD, mode); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fmt.Errorf("validation state store is busy")
		}
		return fmt.Errorf("lock validation state store: %w", err)
	}
	defer unix.Flock(lockFD, unix.LOCK_UN)
	return operation(areaFile)
}

func openValidationState(root, topsID, area string) (int, int, error) {
	current, err := openStateRootNoFollow(root)
	if err != nil {
		return -1, -1, fmt.Errorf("open validation state root: %w", err)
	}
	if err := validateOwnedDirectory(current, false); err != nil {
		_ = unix.Close(current)
		return -1, -1, fmt.Errorf("validate validation state root: %w", err)
	}
	for _, component := range []string{"symphony", topsID, "qxctl", "validation"} {
		next, err := openOrCreatePrivateDirectory(current, component)
		_ = unix.Close(current)
		if err != nil {
			return -1, -1, err
		}
		current = next
	}
	base := current
	areaFD, err := openOrCreatePrivateDirectory(base, area)
	if err != nil {
		_ = unix.Close(base)
		return -1, -1, err
	}
	return base, areaFD, nil
}

func openOrCreatePrivateDirectory(parent int, component string) (int, error) {
	next, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if errors.Is(err, syscall.ENOENT) {
		if mkdirErr := unix.Mkdirat(parent, component, 0o700); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
			return -1, fmt.Errorf("create validation state component %q: %w", component, mkdirErr)
		}
		if syncErr := unix.Fsync(parent); syncErr != nil {
			return -1, fmt.Errorf("durably create validation state component %q: %w", component, syncErr)
		}
		next, err = unix.Openat(parent, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	}
	if err != nil {
		return -1, fmt.Errorf("open validation state component %q: %w", component, err)
	}
	if err := validateOwnedDirectory(next, true); err != nil {
		_ = unix.Close(next)
		return -1, fmt.Errorf("validate validation state component %q: %w", component, err)
	}
	return next, nil
}

func openStateRootNoFollow(root string) (int, error) {
	current, err := unix.Open(string(os.PathSeparator), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return -1, err
	}
	for _, component := range strings.Split(strings.TrimPrefix(filepath.Clean(root), string(os.PathSeparator)), string(os.PathSeparator)) {
		if component == "" || component == "." || component == ".." {
			_ = unix.Close(current)
			return -1, fmt.Errorf("validation state root contains an unsafe component")
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

func validateOwnedDirectory(fd int, private bool) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFDIR || status.Uid != uint32(os.Geteuid()) || status.Mode&0o022 != 0 {
		return fmt.Errorf("state directory is not effective-user-owned and protected")
	}
	if private {
		return unix.Fchmod(fd, 0o700)
	}
	return nil
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

func stateFileName(kind, id string) string {
	digest, _ := digestValue(map[string]string{"kind": kind, "id": id})
	return kind + "-" + strings.TrimPrefix(digest, "sha256:") + ".json"
}

func readStateFile(directory *os.File, kind, id string) ([]byte, bool, error) {
	name := stateFileName(kind, id)
	fd, err := unix.Openat(int(directory.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("open validation %s: %w", kind, err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	if err := validateOwnedRegular(fd); err != nil {
		return nil, false, fmt.Errorf("validate validation %s: %w", kind, err)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxStateBytes+1))
	if err != nil || len(data) > maxStateBytes {
		return nil, false, fmt.Errorf("read validation %s failed or exceeded its bound", kind)
	}
	return data, true, nil
}

func listStateFiles(directory *os.File, kind string, limit int) ([]stateFile, error) {
	duplicate, err := unix.Dup(int(directory.Fd()))
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(duplicate), kind+"s")
	defer file.Close()
	entries, err := file.ReadDir(limit + 2)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasPrefix(name, kind+"-") || !strings.HasSuffix(name, ".json") || len(name) != len(kind)+1+64+5 {
			return nil, fmt.Errorf("validation %s directory contains an unknown object", kind)
		}
		names = append(names, name)
	}
	if len(names) > limit {
		return nil, fmt.Errorf("validation %s count exceeds %d", kind, limit)
	}
	sort.Strings(names)
	result := make([]stateFile, 0, len(names))
	for _, name := range names {
		fd, err := unix.Openat(int(directory.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
		if err != nil {
			return nil, err
		}
		state := os.NewFile(uintptr(fd), name)
		if err := validateOwnedRegular(fd); err != nil {
			_ = state.Close()
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(state, maxStateBytes+1))
		_ = state.Close()
		if err != nil || len(data) > maxStateBytes {
			return nil, fmt.Errorf("read validation %s failed or exceeded its bound", kind)
		}
		result = append(result, stateFile{name: name, data: data})
	}
	return result, nil
}

func writeStateFile(directory *os.File, kind, id string, data []byte) error {
	if len(data) > maxStateBytes {
		return fmt.Errorf("validation %s exceeds its bound", kind)
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	name := stateFileName(kind, id)
	temporary := "." + name + ".tmp-" + hex.EncodeToString(random)
	fd, err := unix.Openat(int(directory.Fd()), temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create validation %s temporary file: %w", kind, err)
	}
	file := os.NewFile(uintptr(fd), temporary)
	cleanup := func() {
		_ = file.Close()
		_ = unix.Unlinkat(int(directory.Fd()), temporary, 0)
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
		_ = unix.Unlinkat(int(directory.Fd()), temporary, 0)
		return err
	}
	if err := unix.Renameat(int(directory.Fd()), temporary, int(directory.Fd()), name); err != nil {
		_ = unix.Unlinkat(int(directory.Fd()), temporary, 0)
		return err
	}
	return unix.Fsync(int(directory.Fd()))
}

func removeStateFile(directory *os.File, kind, id string) error {
	if err := unix.Unlinkat(int(directory.Fd()), stateFileName(kind, id), 0); err != nil {
		return err
	}
	return unix.Fsync(int(directory.Fd()))
}

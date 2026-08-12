//go:build darwin || linux

package policyadmin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const maxStateBytes = 1 << 20

type store struct{ directory *os.File }

func withStore(stateDir string, exclusive bool, operation func(*store) error) error {
	directory, err := openPolicyDirectory(stateDir)
	if err != nil {
		return err
	}
	defer directory.Close()
	lockFD, err := unix.Openat(int(directory.Fd()), "policy.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("open SSIAG policy lock: %w", err)
	}
	lock := os.NewFile(uintptr(lockFD), "policy.lock")
	defer lock.Close()
	if err := validateRegular(lockFD); err != nil {
		return fmt.Errorf("validate SSIAG policy lock: %w", err)
	}
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	if err := unix.Flock(lockFD, mode); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fmt.Errorf("SSIAG policy state is busy")
		}
		return fmt.Errorf("lock SSIAG policy state: %w", err)
	}
	defer unix.Flock(lockFD, unix.LOCK_UN)
	return operation(&store{directory: directory})
}

func openPolicyDirectory(stateDir string) (*os.File, error) {
	clean := filepath.Clean(stateDir)
	for alias, target := range map[string]string{"/var": "/private/var", "/tmp": "/private/tmp", "/etc": "/private/etc"} {
		if clean == alias || strings.HasPrefix(clean, alias+string(os.PathSeparator)) {
			resolved, err := filepath.EvalSymlinks(alias)
			if err == nil && resolved == target {
				clean = target + strings.TrimPrefix(clean, alias)
			}
			break
		}
	}
	if !filepath.IsAbs(clean) || clean == string(os.PathSeparator) {
		return nil, fmt.Errorf("SSIAG policy state root must be an absolute descendant path")
	}
	components := strings.Split(strings.TrimPrefix(clean, string(os.PathSeparator)), string(os.PathSeparator))
	components = append(components, "policy")
	current, err := unix.Open(string(os.PathSeparator), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	for index, component := range components {
		if component == "" || component == "." || component == ".." {
			_ = unix.Close(current)
			return nil, fmt.Errorf("SSIAG policy state path has an unsafe component")
		}
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkdirErr := unix.Mkdirat(current, component, 0o700); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return nil, fmt.Errorf("create SSIAG policy state component %q: %w", component, mkdirErr)
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return nil, fmt.Errorf("sync SSIAG policy state parent: %w", syncErr)
			}
			next, openErr = unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return nil, fmt.Errorf("open SSIAG policy state component %q: %w", component, openErr)
		}
		current = next
		if index >= len(components)-2 {
			if err := validateDirectory(current, index == len(components)-1); err != nil {
				_ = unix.Close(current)
				return nil, err
			}
		}
	}
	return os.NewFile(uintptr(current), "ssiag-policy"), nil
}

func validateDirectory(fd int, private bool) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	forbidden := status.Mode & 0
	forbidden = 0o022
	if private {
		forbidden = 0o077
	}
	if status.Mode&unix.S_IFMT != unix.S_IFDIR || status.Uid != uint32(os.Geteuid()) || status.Mode&forbidden != 0 {
		return fmt.Errorf("SSIAG policy state directory is not effective-user-owned and protected: uid=%d expected=%d mode=%#o", status.Uid, os.Geteuid(), status.Mode&0o777)
	}
	if !private {
		return nil
	}
	return unix.Fchmod(fd, 0o700)
}

func validateRegular(fd int) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFREG || status.Uid != uint32(os.Geteuid()) || status.Mode&0o077 != 0 {
		return fmt.Errorf("SSIAG policy state file is not effective-user-owned and owner-only")
	}
	return unix.Fchmod(fd, 0o600)
}

func (s *store) read(name string, target any) (bool, error) {
	fd, err := unix.Openat(int(s.directory.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open SSIAG policy %s: %w", name, err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	if err := validateRegular(fd); err != nil {
		return false, err
	}
	payload, err := io.ReadAll(io.LimitReader(file, maxStateBytes+1))
	if err != nil || len(payload) > maxStateBytes {
		return false, fmt.Errorf("SSIAG policy %s exceeds its read bound", name)
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false, fmt.Errorf("decode SSIAG policy %s: %w", name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return false, fmt.Errorf("SSIAG policy %s must contain one JSON value", name)
	}
	return true, nil
}

func (s *store) write(name string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if len(payload) > maxStateBytes {
		return fmt.Errorf("SSIAG policy %s exceeds its write bound", name)
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	temporary := "." + name + ".tmp-" + hex.EncodeToString(random)
	fd, err := unix.Openat(int(s.directory.Fd()), temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary SSIAG policy state: %w", err)
	}
	file := os.NewFile(uintptr(fd), temporary)
	clean := false
	defer func() {
		_ = file.Close()
		if !clean {
			_ = unix.Unlinkat(int(s.directory.Fd()), temporary, 0)
		}
	}()
	if err := validateRegular(fd); err != nil {
		return err
	}
	if _, err := file.Write(payload); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := unix.Renameat(int(s.directory.Fd()), temporary, int(s.directory.Fd()), name); err != nil {
		return fmt.Errorf("atomically replace SSIAG policy %s: %w", name, err)
	}
	clean = true
	return s.directory.Sync()
}

func (s *store) remove(name string) error {
	err := unix.Unlinkat(int(s.directory.Fd()), name, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove SSIAG policy %s: %w", name, err)
	}
	return s.directory.Sync()
}

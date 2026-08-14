//go:build darwin || linux

package provider

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

const maximumProviderBindingStateBytes = 2 << 20

type bindingStore struct{ directory *os.File }

func withBindingStore(root, providerName string, exclusive bool, operation func(*bindingStore) error) error {
	directory, err := openBindingDirectory(root, providerName)
	if err != nil {
		return err
	}
	defer directory.Close()
	lockFD, err := unix.Openat(int(directory.Fd()), "binding.lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("open SSIAG provider binding lock: %w", err)
	}
	lock := os.NewFile(uintptr(lockFD), "binding.lock")
	defer lock.Close()
	if err := validateBindingRegular(lockFD); err != nil {
		return fmt.Errorf("validate SSIAG provider binding lock: %w", err)
	}
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	if err := unix.Flock(lockFD, mode); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return fmt.Errorf("SSIAG provider binding state is busy")
		}
		return fmt.Errorf("lock SSIAG provider binding state: %w", err)
	}
	defer unix.Flock(lockFD, unix.LOCK_UN)
	return operation(&bindingStore{directory: directory})
}

func openBindingDirectory(root, providerName string) (*os.File, error) {
	clean := filepath.Clean(root)
	for alias, target := range map[string]string{"/var": "/private/var", "/tmp": "/private/tmp", "/etc": "/private/etc"} {
		if clean == alias || strings.HasPrefix(clean, alias+string(os.PathSeparator)) {
			resolved, err := filepath.EvalSymlinks(alias)
			if err == nil && resolved == target {
				clean = target + strings.TrimPrefix(clean, alias)
			}
			break
		}
	}
	if !filepath.IsAbs(clean) || clean == string(os.PathSeparator) || !validToken(providerName) {
		return nil, fmt.Errorf("SSIAG provider binding state path is unsafe")
	}
	components := strings.Split(strings.TrimPrefix(clean, string(os.PathSeparator)), string(os.PathSeparator))
	components = append(components, providerName)
	current, err := unix.Open(string(os.PathSeparator), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	for index, component := range components {
		if component == "" || component == "." || component == ".." {
			_ = unix.Close(current)
			return nil, fmt.Errorf("SSIAG provider binding state path has an unsafe component")
		}
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkdirErr := unix.Mkdirat(current, component, 0o700); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return nil, fmt.Errorf("create SSIAG provider binding state component %q: %w", component, mkdirErr)
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return nil, fmt.Errorf("sync SSIAG provider binding state parent: %w", syncErr)
			}
			next, openErr = unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return nil, fmt.Errorf("open SSIAG provider binding state component %q: %w", component, openErr)
		}
		current = next
		if index >= len(components)-2 {
			if err := validateBindingDirectory(current); err != nil {
				_ = unix.Close(current)
				return nil, err
			}
		}
	}
	return os.NewFile(uintptr(current), "ssiag-provider-binding"), nil
}

func validateBindingDirectory(fd int) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFDIR || status.Uid != uint32(os.Geteuid()) || status.Mode&0o077 != 0 {
		return fmt.Errorf("SSIAG provider binding state directory is not effective-user-owned and owner-only")
	}
	return unix.Fchmod(fd, 0o700)
}

func validateBindingRegular(fd int) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFREG || status.Uid != uint32(os.Geteuid()) || status.Mode&0o077 != 0 {
		return fmt.Errorf("SSIAG provider binding state file is not effective-user-owned and owner-only")
	}
	return unix.Fchmod(fd, 0o600)
}

func (s *bindingStore) read(name string, target any) (bool, error) {
	fd, err := unix.Openat(int(s.directory.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open SSIAG provider binding %s: %w", name, err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	if err := validateBindingRegular(fd); err != nil {
		return false, err
	}
	payload, err := io.ReadAll(io.LimitReader(file, maximumProviderBindingStateBytes+1))
	if err != nil || len(payload) > maximumProviderBindingStateBytes {
		return false, fmt.Errorf("SSIAG provider binding %s exceeds its read bound", name)
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false, fmt.Errorf("decode SSIAG provider binding %s: %w", name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return false, fmt.Errorf("SSIAG provider binding %s must contain one JSON value", name)
	}
	return true, nil
}

func (s *bindingStore) write(name string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if len(payload) > maximumProviderBindingStateBytes {
		return fmt.Errorf("SSIAG provider binding %s exceeds its write bound", name)
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	temporary := "." + name + ".tmp-" + hex.EncodeToString(random)
	fd, err := unix.Openat(int(s.directory.Fd()), temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary SSIAG provider binding state: %w", err)
	}
	file := os.NewFile(uintptr(fd), temporary)
	clean := false
	defer func() {
		_ = file.Close()
		if !clean {
			_ = unix.Unlinkat(int(s.directory.Fd()), temporary, 0)
		}
	}()
	if err := validateBindingRegular(fd); err != nil {
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
		return fmt.Errorf("atomically replace SSIAG provider binding %s: %w", name, err)
	}
	clean = true
	return s.directory.Sync()
}

func (s *bindingStore) remove(name string) error {
	err := unix.Unlinkat(int(s.directory.Fd()), name, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove SSIAG provider binding %s: %w", name, err)
	}
	return s.directory.Sync()
}

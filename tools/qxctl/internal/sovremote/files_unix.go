//go:build darwin || linux

package sovremote

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func processCancellation(cmd *exec.Cmd) {
	// This bounds cancellation/pipe cleanup only, never calculation duration.
	cmd.WaitDelay = 250 * time.Millisecond
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
}
func privateRegular(f *os.File) error {
	st, e := f.Stat()
	if e != nil {
		return e
	}
	u, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || int(u.Uid) != os.Geteuid() || u.Nlink != 1 {
		return fmt.Errorf("SOV state must be private owner regular file")
	}
	return nil
}
func stateDirectory(path string) (*os.File, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("SOV state path must be canonical absolute")
	}
	dir := filepath.Dir(path)
	real, e := filepath.EvalSymlinks(dir)
	if e != nil || real != dir {
		return nil, fmt.Errorf("SOV state directory has symlink components")
	}
	fd, e := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), dir)
	st, e := f.Stat()
	if e != nil {
		f.Close()
		return nil, e
	}
	u, ok := st.Sys().(*syscall.Stat_t)
	if !ok || int(u.Uid) != os.Geteuid() || st.Mode().Perm()&0077 != 0 {
		f.Close()
		return nil, fmt.Errorf("SOV state directory must be private and owned by caller")
	}
	return f, nil
}
func lockedState(path string) (*os.File, *os.File, error) {
	dir, e := stateDirectory(path)
	if e != nil {
		return nil, nil, e
	}
	fd, e := unix.Openat(int(dir.Fd()), filepath.Base(path)+".lock", unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e != nil {
		dir.Close()
		return nil, nil, e
	}
	lock := os.NewFile(uintptr(fd), "SOV lock")
	if e = privateRegular(lock); e == nil {
		e = unix.Flock(fd, unix.LOCK_EX)
	}
	if e != nil {
		lock.Close()
		dir.Close()
		return nil, nil, e
	}
	return dir, lock, nil
}
func readState(dir *os.File, name string) ([]byte, error) {
	fd, e := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	if e = privateRegular(f); e != nil {
		return nil, e
	}
	b, e := io.ReadAll(io.LimitReader(f, MaxDocumentBytes+1))
	if e != nil || int64(len(b)) > MaxDocumentBytes {
		return nil, fmt.Errorf("SOV state exceeds control bound")
	}
	return b, nil
}
func ReadConfiguration(path, expected string) (Configuration, string, error) {
	var c Configuration
	dir, e := stateDirectory(path)
	if e != nil {
		return c, "", e
	}
	defer dir.Close()
	b, e := readState(dir, filepath.Base(path))
	if e != nil {
		return c, "", e
	}
	digest := Digest(b)
	if expected != "" && expected != digest {
		return c, "", fmt.Errorf("SOV configuration changed")
	}
	if e = decode(b, &c); e != nil {
		return c, "", e
	}
	if e = ValidateConfiguration(c); e != nil {
		return c, "", e
	}
	return c, digest, nil
}
func Configure(raw []byte) (ConfigurationResult, error) {
	var in ConfigureInput
	if e := decode(raw, &in); e != nil {
		return ConfigurationResult{}, e
	}
	if in.Protocol != "symphony.sov.target-configure-input.v1" || !nullableSHA(in.ExpectedSHA256) {
		return ConfigurationResult{}, fmt.Errorf("invalid configuration request")
	}
	if e := ValidateConfiguration(in.Configuration); e != nil {
		return ConfigurationResult{}, e
	}
	dir, lock, e := lockedState(in.Path)
	if e != nil {
		return ConfigurationResult{}, e
	}
	defer dir.Close()
	defer lock.Close()
	old, e := readState(dir, filepath.Base(in.Path))
	if in.ExpectedSHA256 == nil {
		if !os.IsNotExist(e) {
			return ConfigurationResult{}, fmt.Errorf("configuration already exists or cannot be inspected")
		}
	} else if e != nil || Digest(old) != *in.ExpectedSHA256 {
		return ConfigurationResult{}, fmt.Errorf("stale configuration digest")
	}
	b, e := json.Marshal(in.Configuration)
	if e != nil {
		return ConfigurationResult{}, e
	}
	temp, e := os.CreateTemp(filepath.Dir(in.Path), ".sov-config-")
	if e != nil {
		return ConfigurationResult{}, e
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, e = temp.Write(b); e == nil {
		e = temp.Sync()
	}
	closeErr := temp.Close()
	if e != nil {
		return ConfigurationResult{}, e
	}
	if closeErr != nil {
		return ConfigurationResult{}, closeErr
	}
	if e = unix.Renameat(int(dir.Fd()), filepath.Base(name), int(dir.Fd()), filepath.Base(in.Path)); e != nil {
		return ConfigurationResult{}, e
	}
	if e = dir.Sync(); e != nil {
		return ConfigurationResult{}, e
	}
	return ConfigurationResult{Protocol: "symphony.sov.target-configuration-result.v1", Path: in.Path, SHA256: Digest(b), Configuration: in.Configuration}, nil
}
func RemoveConfiguration(path, expected string) error {
	if !sha.MatchString(expected) {
		return fmt.Errorf("exact configuration digest required")
	}
	dir, lock, e := lockedState(path)
	if e != nil {
		return e
	}
	defer dir.Close()
	defer lock.Close()
	b, e := readState(dir, filepath.Base(path))
	if e != nil || Digest(b) != expected {
		return fmt.Errorf("stale configuration digest")
	}
	if e = unix.Unlinkat(int(dir.Fd()), filepath.Base(path), 0); e != nil {
		return e
	}
	return dir.Sync()
}

func cleanupProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func openExecutable(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}

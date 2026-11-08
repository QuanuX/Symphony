//go:build darwin || linux

// Package transportprofile is a presentation client for the fixed protected
// profile administrator. It never writes profiles or runs a connection.
package transportprofile

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	w "github.com/QuanuX/Symphony/tools/qxctl/internal/transportprofile/wire"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func HelperPath() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/Symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin"
	}
	return "/opt/symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin"
}
func protectedBinary(path string) (string, error) {
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
			return "", errors.New("protected helper ancestry unavailable")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			return "", errors.New("protected helper owner unavailable")
		}
		if p == "/" {
			break
		}
	}
	fd, e := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if e != nil {
		return "", errors.New("protected helper unavailable")
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 || info.Mode().Perm()&0111 == 0 {
		return "", errors.New("protected helper permissions invalid")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || stat.Nlink != 1 {
		return "", errors.New("protected helper owner invalid")
	}
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", errors.New("protected helper unreadable")
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
func invocation(operation string) (string, []string) {
	if operation == "apply" || operation == "recover" {
		return "/usr/bin/sudo", []string{"-n", "--", HelperPath(), "administer"}
	}
	return HelperPath(), []string{"query"}
}

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, errors.New("control output exceeded")
	}
	return b.Buffer.Write(p)
}
func Run(ctx context.Context, operation string, raw []byte) (w.Result, error) {
	var input w.Request
	if w.Decode(raw, &input) != nil || w.ValidateRequest(input) != nil || input.Operation != operation {
		return w.Result{}, errors.New("invalid transport profile input")
	}
	// qxctl itself never recovers an original user from SUDO_*.
	if os.Geteuid() == 0 {
		return w.Result{}, errors.New("run qxctl profile administration as your ordinary user; only apply/recover elevate the fixed helper")
	}
	identity, e := protectedBinary(HelperPath())
	if e != nil {
		return w.Result{}, e
	}
	if input.Plan != nil && input.Plan.AdministratorDigest != identity {
		return w.Result{}, errors.New("profile administrator changed; create a new plan")
	}
	executable, args := invocation(operation)
	if executable == "/usr/bin/sudo" {
		if _, e = protectedBinary(executable); e != nil {
			return w.Result{}, errors.New("protected sudo unavailable")
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = "/"
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.Stdin = bytes.NewReader(raw)
	output := &boundedBuffer{limit: w.MaxDocumentBytes + 1}
	diagnostic := &boundedBuffer{limit: 65536}
	cmd.Stdout = output
	cmd.Stderr = diagnostic
	cmd.WaitDelay = 5 * time.Second
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	err := cmd.Run()
	var result w.Result
	data := bytes.TrimSpace(output.Bytes())
	if len(data) > w.MaxDocumentBytes || w.Decode(data, &result) != nil || w.ValidateResult(result, input) != nil {
		if operation == "apply" || operation == "recover" {
			return w.Result{}, errors.New("profile administration did not return a verified outcome; inspect status before any further mutation (sudo authorization may be required)")
		}
		return w.Result{}, errors.New("protected profile query unavailable")
	}
	if result.Plan != nil && result.Plan.AdministratorDigest != identity {
		return w.Result{}, errors.New("profile administrator identity mismatch")
	}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 5 || result.Completed {
			return w.Result{}, errors.New("profile process outcome unavailable; inspect status")
		}
	} else if !result.Completed {
		return w.Result{}, errors.New("profile process result mismatch")
	}
	return result, nil
}

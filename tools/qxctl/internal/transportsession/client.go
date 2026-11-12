//go:build darwin || linux

package transportsession

import (
	"bytes"
	"context"
	"fmt"
	profile "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofile"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

type bounded struct{ bytes.Buffer }

func (b *bounded) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 65536 {
		return 0, fmt.Errorf("session response bound")
	}
	return b.Buffer.Write(p)
}
func Run(ctx context.Context, operation string, raw []byte) (Result, error) {
	r, err := Decode(raw)
	if err != nil || r.Operation != operation || os.Geteuid() == 0 {
		return Result{}, Error()
	}
	result := Failure(r, false)
	store, err := profile.Open(r.Endpoint.Scope)
	if err != nil {
		return result, nil
	}
	defer store.Close()
	var carrier string
	err = store.WithSession(r.Endpoint, ClearOnly(r), func(path, _, _ string) error { carrier = path; return nil })
	if err != nil {
		return result, nil
	}
	ms, _ := strconv.ParseInt(r.Deadline, 10, 64)
	ctx, cancel := context.WithDeadline(ctx, time.UnixMilli(ms))
	defer cancel()
	cmd := exec.CommandContext(ctx, carrier, "--session")
	cmd.Stdin = bytes.NewReader(raw)
	var output bounded
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	for _, key := range []string{"DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
		if value, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Let the carrier cancel and reap its native process group before force cleanup.
	cmd.Cancel = func() error { return syscall.Kill(cmd.Process.Pid, syscall.SIGTERM) }
	cmd.WaitDelay = 6 * time.Second
	if err = cmd.Start(); err != nil {
		return result, nil
	}
	defer func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }()
	err = cmd.Wait()
	result = Failure(r, true)
	if value, e := DecodeResult(output.Bytes(), r); e == nil && (err == nil) == value.Completed {
		result = value
	}
	return result, nil
}

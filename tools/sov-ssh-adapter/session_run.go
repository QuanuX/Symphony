//go:build darwin || linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profile"
	s "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/sessionwire"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func runSession(ctx context.Context, raw []byte, out io.Writer) int {
	r, err := s.Decode(raw)
	if err != nil || os.Geteuid() == 0 {
		return 2
	}
	result := s.Failure(r, false)
	emit := func() int {
		if json.NewEncoder(out).Encode(result) != nil {
			return 1
		}
		if result.Completed {
			return 0
		}
		return 5
	}
	store, err := profile.Open(r.Endpoint.Scope)
	if err != nil {
		return emit()
	}
	defer store.Close()
	ms, _ := strconv.ParseInt(r.Deadline, 10, 64)
	ctx, cancel := context.WithDeadline(ctx, time.UnixMilli(ms))
	defer cancel()
	var cmd *exec.Cmd
	var output profileBuffer
	output.limit = 65536
	err = store.WithSession(r.Endpoint, s.ClearOnly(r), func(carrier, _, helper string) error {
		self, e := os.Executable()
		if e != nil || self != carrier {
			return s.Error()
		}
		raw, e := json.Marshal(s.Native(r))
		if e != nil {
			return e
		}
		cmd = exec.CommandContext(ctx, helper, "--session")
		cmd.Stdin = bytes.NewReader(raw)
		cmd.Stdout = &output
		cmd.Stderr = io.Discard
		cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
		for _, key := range []string{"DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
			if value, ok := os.LookupEnv(key); ok {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.WaitDelay = 250 * time.Millisecond
		cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
		if e = ctx.Err(); e != nil {
			return e
		}
		return cmd.Start()
	})
	if err != nil {
		return emit()
	}
	defer func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }()
	err = cmd.Wait()
	result = s.Failure(r, true)
	if value, e := s.DecodeResult(output.Bytes(), r); e == nil && (err == nil) == value.Completed {
		result = value
	}
	return emit()
}

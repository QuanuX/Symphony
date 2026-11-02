//go:build darwin || linux

package knowledgeengine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestInvokeDeadlineTerminatesDescendantsHoldingOutput(t *testing.T) {
	prefix, directory := t.TempDir(), t.TempDir()
	createInstalledFixture(t, prefix, "#!/bin/sh\n/bin/sleep 30 &\nprintf '%s' \"$!\" > descendant.pid\nprintf '%s' \"$$\" > leader.pid\nwait\n")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(context.Canceled)
	finished := make(chan error, 1)
	go func() {
		_, err := Invoke(ctx, prefix, "0.1.0-dev", directory, "inspect", []byte(`{}`))
		finished <- err
	}()
	leader := waitEnginePID(t, directory, "leader.pid", finished)
	descendant := waitEnginePID(t, directory, "descendant.pid", finished)
	t.Cleanup(func() {
		_ = unix.Kill(-leader, unix.SIGKILL)
		_ = unix.Kill(descendant, unix.SIGKILL)
	})
	// Arm the real deadline only after the engine acknowledges both processes.
	// Startup admission is covered separately by the caller-deadline test.
	deadlineContext, stopDeadline := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer stopDeadline()
	stopBridge := context.AfterFunc(deadlineContext, func() { cancel(context.Cause(deadlineContext)) })
	defer stopBridge()
	started := time.Now()
	select {
	case err := <-finished:
		if err == nil || !strings.Contains(err.Error(), "hard process deadline") || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("descendant timeout did not retain the caller deadline: %v", err)
		}
	case <-time.After(time.Second):
		_ = unix.Kill(-leader, unix.SIGKILL)
		_ = unix.Kill(descendant, unix.SIGKILL)
		t.Fatal("descendant retaining stdout delayed the caller deadline")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("hard deadline returned after %s", elapsed)
	}
	assertEngineProcessGone(t, descendant)
	assertEngineProcessGone(t, leader)
}

func TestInvokeLeaderExitBoundsPipesAndCleansDescendants(t *testing.T) {
	prefix, directory := t.TempDir(), t.TempDir()
	createInstalledFixture(t, prefix, "#!/bin/sh\n/bin/sleep 30 &\nprintf '%s' \"$!\" > descendant.pid\nprintf '%s' \"$$\" > leader.pid\nexit 0\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		_, err := Invoke(ctx, prefix, "0.1.0-dev", directory, "inspect", []byte(`{}`))
		finished <- err
	}()
	leader := waitEnginePID(t, directory, "leader.pid", finished)
	descendant := waitEnginePID(t, directory, "descendant.pid", finished)
	t.Cleanup(func() {
		_ = unix.Kill(-leader, unix.SIGKILL)
		_ = unix.Kill(descendant, unix.SIGKILL)
	})
	select {
	case err := <-finished:
		if err == nil || strings.Contains(err.Error(), "hard process deadline") {
			t.Fatalf("exited leader with an open descendant pipe did not fail during bounded draining: %v", err)
		}
	case <-time.After(2 * time.Second):
		_ = unix.Kill(-leader, unix.SIGKILL)
		_ = unix.Kill(descendant, unix.SIGKILL)
		t.Fatal("leader exit did not bound descendant pipe draining")
	}
	assertEngineProcessGone(t, descendant)
}

func waitEnginePID(t *testing.T, directory, name string, finished <-chan error) int {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err == nil {
			pid, err := strconv.Atoi(string(data))
			if err == nil && pid > 1 {
				return pid
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Preserve the original startup bound and expose an early admission/process
	// failure if it caused the missing acknowledgement. Successful waits do not
	// consume the invocation result used by the lifecycle assertions below.
	select {
	case err := <-finished:
		t.Fatalf("engine did not record %s; invocation completed with: %v", name, err)
	default:
		t.Fatalf("engine did not record %s; invocation remains pending", name)
	}
	return 0
}

func assertEngineProcessGone(t *testing.T, pid int) {
	t.Helper()
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		if errors.Is(unix.Kill(pid, 0), unix.ESRCH) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("owned engine process %d survived cleanup", pid)
}

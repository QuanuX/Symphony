//go:build darwin || linux

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSBVSIGTERMCancelsCaller(t *testing.T) {
	if os.Getenv("SBV_SIGNAL_HELPER") == "1" {
		ctx, stop := sbvSignalContext(context.Background())
		defer stop()
		if err := os.WriteFile(os.Getenv("SBV_SIGNAL_READY"), []byte("ready"), 0600); err != nil {
			os.Exit(2)
		}
		<-ctx.Done()
		// Receiving SIGTERM must return through cleanup instead of default exit.
		os.Exit(0)
	}
	ready := filepath.Join(t.TempDir(), "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSBVSIGTERMCancelsCaller$")
	child.Env = append(os.Environ(), "SBV_SIGNAL_HELPER=1", "SBV_SIGNAL_READY="+ready)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer child.Process.Kill()
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if ctx.Err() != nil {
			t.Fatal("helper not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := child.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err != nil {
		t.Fatalf("SIGTERM bypassed cancellation cleanup: %v", err)
	}
}

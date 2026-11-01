//go:build darwin || linux

package server

import (
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestWaitForSocketRequiresListeningSocket(t *testing.T) {
	path := shortSocketPath(t)
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	defer syscall.Close(fd)
	if err := syscall.Bind(fd, &syscall.SockaddrUnix{Name: path}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("bound fixture has no socket inode: info=%v error=%v", info, err)
	}
	if connection, err := net.DialTimeout("unix", path, 100*time.Millisecond); err == nil {
		_ = connection.Close()
		t.Fatal("bound fixture accepted a connection before listen")
	}
	ready := make(chan struct{})
	go func() {
		defer close(ready)
		waitForSocket(t, path)
	}()
	select {
	case <-ready:
		t.Fatal("socket readiness returned before listen")
	case <-time.After(50 * time.Millisecond):
	}
	if err := syscall.Listen(fd, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
	case <-time.After(4 * time.Second):
		t.Fatal("socket readiness did not observe listen")
	}
}

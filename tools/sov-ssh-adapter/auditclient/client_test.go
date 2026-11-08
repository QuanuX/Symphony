package auditclient

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestKernelPeerAndProtectedSocketRefusal(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "audit.sock")
	// macOS pathname sockets have a small bound; use a short system temp prefix.
	dir, e := os.MkdirTemp("/tmp", "profile-peer-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	socket = filepath.Join(dir, "s")
	listener, e := net.Listen("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, e := listener.Accept()
		if e == nil {
			defer conn.Close()
			var peer Credentials
			peer, e = getPeerCredentials(conn)
			if e == nil && peer.UID != uint32(os.Geteuid()) {
				done <- unavailable
				return
			}
		}
		done <- e
	}()
	conn, e := net.Dial("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	t.Setenv("SUDO_UID", "0")
	if protectedSocket(socket) == nil {
		t.Fatal("unprotected socket ancestry admitted")
	}
}

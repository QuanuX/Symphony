//go:build darwin || linux

package peerauth

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialsFromAcceptedUnixConnection(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "peer.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("Unix sockets unavailable: %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()

	client, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var server net.Conn
	select {
	case err := <-acceptErr:
		t.Fatal(err)
	case server = <-accepted:
	}
	defer server.Close()

	credentials, err := CredentialsFromConn(server)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.PID != int32(os.Getpid()) || credentials.UID != uint32(os.Geteuid()) || credentials.GID != uint32(os.Getegid()) {
		t.Fatalf("credentials = %+v, want pid=%d uid=%d gid=%d", credentials, os.Getpid(), os.Geteuid(), os.Getegid())
	}
}

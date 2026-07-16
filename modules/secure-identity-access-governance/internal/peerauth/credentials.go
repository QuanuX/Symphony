package peerauth

import (
	"fmt"
	"net"
	"syscall"
)

type syscallConnection interface {
	SyscallConn() (syscall.RawConn, error)
}

func CredentialsFromConn(conn net.Conn) (Credentials, error) {
	if conn == nil {
		return Credentials{}, fmt.Errorf("connection is nil")
	}
	socket, ok := conn.(syscallConnection)
	if !ok {
		return Credentials{}, fmt.Errorf("connection does not expose a kernel socket descriptor")
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return Credentials{}, fmt.Errorf("access connection descriptor: %w", err)
	}
	var (
		credentials   Credentials
		credentialErr error
	)
	if err := raw.Control(func(fd uintptr) {
		credentials, credentialErr = credentialsFromFD(int(fd))
	}); err != nil {
		return Credentials{}, fmt.Errorf("inspect connection descriptor: %w", err)
	}
	if credentialErr != nil {
		return Credentials{}, credentialErr
	}
	return credentials, nil
}

package peerauth

import (
	"fmt"
	"net"
	"syscall"
)

type Credentials struct {
	PID int32
	UID uint32
	GID uint32
}

type syscallConnection interface {
	SyscallConn() (syscall.RawConn, error)
}

func CredentialsFromConn(conn net.Conn) (Credentials, error) {
	if conn == nil {
		return Credentials{}, fmt.Errorf("stav peer authentication: nil connection")
	}
	socket, ok := conn.(syscallConnection)
	if !ok {
		return Credentials{}, fmt.Errorf("stav peer authentication: connection has no kernel descriptor")
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return Credentials{}, fmt.Errorf("stav peer authentication: access descriptor: %w", err)
	}
	var credentials Credentials
	var credentialErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, credentialErr = credentialsFromFD(int(fd))
	}); err != nil {
		return Credentials{}, fmt.Errorf("stav peer authentication: inspect descriptor: %w", err)
	}
	if credentialErr != nil {
		return Credentials{}, credentialErr
	}
	return credentials, nil
}

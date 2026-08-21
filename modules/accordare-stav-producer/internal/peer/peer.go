package peer

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

func FromConn(conn net.Conn) (Credentials, error) {
	if conn == nil {
		return Credentials{}, fmt.Errorf("nil peer connection")
	}
	socket, ok := conn.(syscallConnection)
	if !ok {
		return Credentials{}, fmt.Errorf("peer connection has no kernel descriptor")
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return Credentials{}, err
	}
	var credentials Credentials
	var credentialErr error
	if err := raw.Control(func(fd uintptr) { credentials, credentialErr = fromFD(int(fd)) }); err != nil {
		return Credentials{}, err
	}
	return credentials, credentialErr
}

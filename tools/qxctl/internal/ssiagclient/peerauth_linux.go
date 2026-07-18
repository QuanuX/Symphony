//go:build linux

package ssiagclient

import (
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

type syscallConnection interface {
	SyscallConn() (syscall.RawConn, error)
}

type Credentials struct {
	PID int32
	UID uint32
	GID uint32
}

func getPeerCredentials(conn net.Conn) (Credentials, error) {
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
	err = raw.Control(func(fd uintptr) {
		credential, err := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if err != nil {
			credentialErr = fmt.Errorf("read Linux SO_PEERCRED: %w", err)
			return
		}
		if credential.Pid <= 0 {
			credentialErr = fmt.Errorf("Linux SO_PEERCRED returned invalid pid %d", credential.Pid)
			return
		}
		credentials = Credentials{
			PID: credential.Pid,
			UID: credential.Uid,
			GID: credential.Gid,
		}
	})
	if err != nil {
		return Credentials{}, fmt.Errorf("inspect connection descriptor: %w", err)
	}
	if credentialErr != nil {
		return Credentials{}, credentialErr
	}
	return credentials, nil
}

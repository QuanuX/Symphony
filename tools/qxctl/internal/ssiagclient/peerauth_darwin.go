//go:build darwin

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
		credential, err := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if err != nil {
			credentialErr = fmt.Errorf("read Darwin LOCAL_PEERCRED: %w", err)
			return
		}
		if credential.Ngroups < 1 {
			credentialErr = fmt.Errorf("Darwin LOCAL_PEERCRED returned no effective group")
			return
		}
		pid, err := unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
		if err != nil {
			credentialErr = fmt.Errorf("read Darwin LOCAL_PEERPID: %w", err)
			return
		}
		if pid <= 0 {
			credentialErr = fmt.Errorf("Darwin LOCAL_PEERPID returned invalid pid %d", pid)
			return
		}
		credentials = Credentials{
			PID: int32(pid),
			UID: credential.Uid,
			GID: credential.Groups[0],
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

//go:build linux

package peerauth

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func credentialsFromFD(fd int) (Credentials, error) {
	credential, err := unix.GetsockoptUcred(fd, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil {
		return Credentials{}, fmt.Errorf("stav peer authentication: read Linux SO_PEERCRED: %w", err)
	}
	if credential.Pid <= 0 {
		return Credentials{}, fmt.Errorf("stav peer authentication: invalid Linux peer PID")
	}
	return Credentials{PID: credential.Pid, UID: credential.Uid, GID: credential.Gid}, nil
}

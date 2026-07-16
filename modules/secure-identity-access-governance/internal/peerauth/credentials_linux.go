//go:build linux

package peerauth

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func credentialsFromFD(fd int) (Credentials, error) {
	credential, err := unix.GetsockoptUcred(fd, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil {
		return Credentials{}, fmt.Errorf("read Linux SO_PEERCRED: %w", err)
	}
	if credential.Pid <= 0 {
		return Credentials{}, fmt.Errorf("Linux SO_PEERCRED returned invalid pid %d", credential.Pid)
	}
	return Credentials{
		PID: credential.Pid,
		UID: credential.Uid,
		GID: credential.Gid,
	}, nil
}

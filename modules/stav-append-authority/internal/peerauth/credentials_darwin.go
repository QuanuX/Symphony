//go:build darwin

package peerauth

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func credentialsFromFD(fd int) (Credentials, error) {
	credential, err := unix.GetsockoptXucred(fd, unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	if err != nil {
		return Credentials{}, fmt.Errorf("stav peer authentication: read Darwin LOCAL_PEERCRED: %w", err)
	}
	if credential.Ngroups < 1 {
		return Credentials{}, fmt.Errorf("stav peer authentication: Darwin peer has no effective group")
	}
	pid, err := unix.GetsockoptInt(fd, unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	if err != nil {
		return Credentials{}, fmt.Errorf("stav peer authentication: read Darwin LOCAL_PEERPID: %w", err)
	}
	if pid <= 0 {
		return Credentials{}, fmt.Errorf("stav peer authentication: invalid Darwin peer PID")
	}
	return Credentials{PID: int32(pid), UID: credential.Uid, GID: credential.Groups[0]}, nil
}

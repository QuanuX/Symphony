//go:build linux

package peer

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func fromFD(fd int) (Credentials, error) {
	credential, err := unix.GetsockoptUcred(fd, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil || credential.Pid <= 0 {
		return Credentials{}, fmt.Errorf("read Linux peer credentials: %w", err)
	}
	return Credentials{PID: credential.Pid, UID: credential.Uid, GID: credential.Gid}, nil
}

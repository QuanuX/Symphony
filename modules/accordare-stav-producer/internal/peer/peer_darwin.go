//go:build darwin

package peer

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func fromFD(fd int) (Credentials, error) {
	credential, err := unix.GetsockoptXucred(fd, unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	if err != nil || credential.Ngroups < 1 {
		return Credentials{}, fmt.Errorf("read Darwin peer credentials: %w", err)
	}
	pid, err := unix.GetsockoptInt(fd, unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	if err != nil || pid <= 0 {
		return Credentials{}, fmt.Errorf("read Darwin peer PID: %w", err)
	}
	return Credentials{PID: int32(pid), UID: credential.Uid, GID: credential.Groups[0]}, nil
}

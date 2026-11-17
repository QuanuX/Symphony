//go:build darwin || linux

package packageinstall

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

func openInstallerSource(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0o022 != 0 || st.Nlink != 1 || st.Size <= 0 || st.Size > 1<<34 {
		f.Close()
		return nil, fmt.Errorf("installer source is unsafe")
	}
	return f, nil
}

//go:build symphony_publication_faults && (darwin || linux)

package shvpublicationstate

import (
	"golang.org/x/sys/unix"
	"os"
	"strconv"
)

// Test-only process barriers exercise real journal interruption boundaries.
func publicationBarrier(point string) {
	if os.Getenv("SYMPHONY_PUBLICATION_BARRIER") != point {
		return
	}
	fd, err := strconv.Atoi(os.Getenv("SYMPHONY_PUBLICATION_BARRIER_FD"))
	if err != nil || fd < 3 {
		panic("invalid publication barrier descriptor")
	}
	if _, err = unix.Write(fd, []byte(point+"\n")); err != nil {
		panic(err)
	}
	if err = unix.Kill(os.Getpid(), unix.SIGSTOP); err != nil {
		panic(err)
	}
}

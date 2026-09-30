//go:build symphony_snv_faults && (darwin || linux)

package snvstate

import (
	"golang.org/x/sys/unix"
	"os"
	"strconv"
)

// Test-only process barriers exercise real journal interruption boundaries.
func barrier(point string) {
	if os.Getenv("SYMPHONY_SNV_BARRIER") != point {
		return
	}
	fd, err := strconv.Atoi(os.Getenv("SYMPHONY_SNV_BARRIER_FD"))
	if err != nil || fd < 3 {
		panic("invalid SNV barrier descriptor")
	}
	if _, err = unix.Write(fd, []byte(point+"\n")); err != nil {
		panic(err)
	}
	if err = unix.Kill(os.Getpid(), unix.SIGSTOP); err != nil {
		panic(err)
	}
}

//go:build darwin || linux

package client

import (
	"fmt"
	"os"
	"syscall"
)

func socketOwnerUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("SSIAG socket has unsupported ownership metadata")
	}
	return stat.Uid, nil
}

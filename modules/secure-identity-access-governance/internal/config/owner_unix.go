//go:build darwin || linux

package config

import (
	"fmt"
	"os"
	"syscall"
)

func fileOwnerUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("trusted SSIAG configuration has unsupported ownership metadata")
	}
	return stat.Uid, nil
}

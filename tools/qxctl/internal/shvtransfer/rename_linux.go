package shvtransfer

import "golang.org/x/sys/unix"

func renameExclusive(dir int, from, to string) error {
	return unix.Renameat2(dir, from, dir, to, unix.RENAME_NOREPLACE)
}

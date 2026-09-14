package shvtransfer

import "golang.org/x/sys/unix"

func renameExclusive(dir int, from, to string) error {
	return unix.RenameatxNp(dir, from, dir, to, unix.RENAME_EXCL)
}

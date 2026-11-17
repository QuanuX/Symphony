//go:build darwin || linux

package packageretirement

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// PublishFile installs exact immutable bytes through the pinned root. The
// destination is never replaced; the owner publishes the receipt last and
// activates admission only after both payload and receipt are durable.
func (g *Lease) PublishFile(relative string, source io.Reader, digest string, size uint64, mode os.FileMode) (bool, error) {
	if g == nil || !g.exclusive || (g.value.Phase != "installing" && g.value.Phase != "open") {
		return false, fmt.Errorf("exclusive installation ownership required")
	}
	if (relative != g.value.Identity.BinaryRelative && relative != g.value.Identity.ReceiptRelative) || !validRelative(relative) || !validDigest(digest) || size == 0 || size > 1<<34 || (mode != 0o644 && mode != 0o755) {
		return false, fmt.Errorf("invalid package publication evidence")
	}
	if err := g.CheckRoot(); err != nil {
		return false, err
	}
	// Existing open packages can be verified but never silently repaired.
	if err := g.VerifyFile(relative, digest, size); err == nil {
		return false, nil
	} else if !errors.Is(err, unix.ENOENT) {
		return false, err
	}
	if g.value.Phase != "installing" {
		return false, fmt.Errorf("open package lost immutable bytes; explicit recovery is required")
	}
	if source == nil {
		return false, fmt.Errorf("package publication source is absent")
	}
	parentPath := filepath.ToSlash(filepath.Dir(relative))
	d, err := g.openDir(parentPath, true)
	if err != nil {
		return false, err
	}
	defer d.Close()
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return false, err
	}
	temporary := ".package-" + hex.EncodeToString(nonce[:])
	fd, err := unix.Openat(int(d.Fd()), temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(mode))
	if err != nil {
		return false, err
	}
	f := os.NewFile(uintptr(fd), temporary)
	defer f.Close()
	defer unix.Unlinkat(int(d.Fd()), temporary, 0)
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(source, int64(size)+1))
	if err != nil {
		return false, err
	}
	if uint64(n) != size || "sha256:"+hex.EncodeToString(h.Sum(nil)) != digest {
		return false, fmt.Errorf("package publication source differs from reviewed bytes")
	}
	if err = f.Chmod(mode); err != nil {
		return false, err
	}
	if err = f.Sync(); err != nil {
		return false, err
	}
	if err = g.CheckRoot(); err != nil {
		return false, err
	}
	current, err := g.openDir(parentPath, false)
	if err != nil {
		return false, err
	}
	same := sameFile(current, d)
	_ = current.Close()
	if !same {
		return false, fmt.Errorf("package publication directory was replaced")
	}
	// linkat is an atomic no-replace commit. Rename would overwrite a competing
	// administrator file even when a prior absence check succeeded.
	if err = unix.Linkat(int(d.Fd()), temporary, int(d.Fd()), filepath.Base(relative), 0); err != nil {
		return false, err
	}
	if err = unix.Unlinkat(int(d.Fd()), temporary, 0); err != nil {
		return true, err
	}
	if err = d.Sync(); err != nil {
		return true, err
	}
	if err = g.CheckRoot(); err != nil {
		return true, err
	}
	current, err = g.openDir(parentPath, false)
	if err != nil {
		return true, err
	}
	same = sameFile(current, d)
	_ = current.Close()
	if !same {
		return true, fmt.Errorf("published package directory was replaced")
	}
	destination, err := g.openFile(d, filepath.Base(relative), false)
	if err != nil {
		return true, err
	}
	same = sameFile(destination, f)
	_ = destination.Close()
	if !same {
		return true, fmt.Errorf("published package file was replaced")
	}
	return true, nil
}

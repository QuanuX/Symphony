//go:build linux && symphony_node_keyring_fixture

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Build the Linux POSIX ACL xattr for a file owned by root with read
// permission for the named service user. The negative test grants the
// owning group read permission as well.
func credentialACL(uid uint32, groupPerm uint16) []byte {
	const undefined = ^uint32(0)
	var acl bytes.Buffer
	must(binary.Write(&acl, binary.LittleEndian, uint32(2)))
	for _, entry := range []struct {
		Tag, Perm uint16
		ID        uint32
	}{
		{1, 4, undefined}, {2, 4, uid}, {4, groupPerm, undefined}, {16, 4, undefined}, {32, 0, undefined},
	} {
		must(binary.Write(&acl, binary.LittleEndian, entry))
	}
	return acl.Bytes()
}

func restartChecks(launcher, importer, consumer string) {
	disposableNode()
	var filesystem syscall.Statfs_t
	must(syscall.Statfs("/dev/shm", &filesystem))
	require(filesystem.Type == 0x01021994, "a private tmpfs is required for unlock input")
	a := addAccount("symphonybootqa")
	uid := fmt.Sprint(a.uid)
	must(os.Chmod(a.home, 0700))
	runtime := filepath.Join("/run", "symphony-node-keyring-"+uid)
	a.mkdir(runtime)
	defer os.RemoveAll(runtime)
	bus := filepath.Join(runtime, "bus")
	cdir := checked(os.MkdirTemp("/dev/shm", "symphony-unlock-"))
	defer os.RemoveAll(cdir)
	a.chown(cdir)
	work := checked(os.MkdirTemp("/run", "symphony-restart-"))
	defer os.RemoveAll(work)
	must(os.Chmod(work, 0755))
	state := filepath.Join(work, "receipts")
	a.mkdir(state)
	tops, ref, request := uuid(), uuid(), uuid()
	value := append([]byte{0, 255}, randomBytes(42)...)
	unlock := fmt.Appendf(nil, "%x", randomBytes(32))
	credential := filepath.Join(cdir, "vault-unlock")
	must(os.WriteFile(credential, unlock, 0400))
	a.chown(credential)
	p := process{env: append(a.environment(), "CREDENTIALS_DIRECTORY="+cdir), identity: a.credentials(), protected: [][]byte{value, unlock}}
	refuse := func(mode string) {
		r := p.run(nil, launcher, mode, uid)
		require(r.code != 0, "unexpected "+mode+" success")
		_, err := os.Lstat(bus)
		require(os.IsNotExist(err), "private bus remained after refusal")
	}
	withService := func(use func()) {
		func() {
			d := p.start(nil, launcher, "serve", uid)
			defer d.stop()
			waitFor(func() bool {
				if !isSocket(bus) {
					return false
				}
				status := p.run(nil, "gdbus", "call", "--address", "unix:path="+bus, "--timeout", "1",
					"--dest", "org.freedesktop.secrets", "--object-path", collection,
					"--method", "org.freedesktop.DBus.Properties.Get", "org.freedesktop.Secret.Collection", "Locked")
				return status.code == 0 && strings.TrimSpace(status.stdout) == "(<false>,)"
			}, d)
			use()
		}()
		_, err := os.Lstat(bus)
		require(os.IsNotExist(err), "private bus remained after shutdown")
	}
	refuse("serve")
	p.success(nil, launcher, "enroll", uid)
	refuse("enroll")
	var gen string
	withService(func() {
		in := p
		in.env = append(a.environment(), "SSH_ORIGINAL_COMMAND=symphony-keyring-import-v1")
		r := in.run(frame(request, tops, ref, value), importer, collection, tops, ref, state, bus)
		gen = generation(r, request)
	})
	vault := checked(os.ReadFile(filepath.Join(a.home, ".local/share/keyrings/login.keyring")))
	require(len(vault) > 0 && !bytes.Contains(vault, value), "unsafe persistent credential")

	must(os.WriteFile(credential, fmt.Appendf(nil, "%x", randomBytes(32)), 0400))
	refuse("serve")
	must(os.Remove(credential))
	refuse("serve")
	must(os.WriteFile(credential, unlock, 0400)) // root-owned after recreation
	const attr = "system.posix_acl_access"
	err := syscall.Setxattr(credential, attr, credentialACL(a.uid, 0), 0)
	if errors.Is(err, syscall.ENOTSUP) {
		// systemd falls back to a service-owned, read-only credential on
		// kernels where tmpfs does not support POSIX ACLs.
		a.chown(credential)
	} else {
		must(err)
	}

	// Use a public dummy marker on a filesystem with ACL support. Only
	// the real unlock input, held on private tmpfs, can unlock the vault.
	sample := checked(os.MkdirTemp("/var/tmp", "symphony-acl-"))
	defer os.RemoveAll(sample)
	must(os.Chmod(sample, 0755))
	marker := filepath.Join(sample, "vault-unlock")
	must(os.WriteFile(marker, bytes.Repeat([]byte("x"), 64), 0400))
	must(syscall.Setxattr(marker, attr, credentialACL(a.uid, 0), 0))
	check := p
	check.env = append(a.environment(), "CREDENTIALS_DIRECTORY="+sample)
	check.success(nil, launcher, "check-unlock", uid)
	must(syscall.Setxattr(marker, attr, credentialACL(a.uid, 4), 0))
	require(check.run(nil, launcher, "check-unlock", uid).code != 0, "broad credential ACL accepted")

	selection := filepath.Join(work, "binding")
	binding(selection, a, bus, tops, ref, gen)
	p.success(nil, consumer, "--expect-refusal", "unavailable", selection)
	withService(func() { p.success(value, consumer, selection) })
	fmt.Println("PASS unattended service restart, persistent generation, missing/wrong unlock refusal, and offline native use")
}

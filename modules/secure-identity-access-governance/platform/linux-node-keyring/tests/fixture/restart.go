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

func credentialACLChecks(launcher string) {
	// This unit directory uses the container filesystem, which supports
	// POSIX ACLs even on kernels where tmpfs does not. It holds only a marker.
	a := addAccount("symphonyaclqa")
	uid := fmt.Sprint(a.uid)
	dir := filepath.Join("/run/credentials", "symphony-node-keyring@"+uid+".service")
	a.mkdir(dir)
	defer os.RemoveAll(dir)
	misplaced := filepath.Join(dir, "unassigned")
	a.mkdir(misplaced)
	marker := filepath.Join(misplaced, "vault-unlock")
	must(os.WriteFile(marker, bytes.Repeat([]byte("x"), 64), 0400))
	const attr = "system.posix_acl_access"
	must(syscall.Setxattr(marker, attr, credentialACL(a.uid, 0), 0))
	p := process{env: append(a.environment(), "CREDENTIALS_DIRECTORY="+misplaced), identity: a.credentials()}
	require(p.run(nil, launcher, "check-unlock", uid).code != 0, "unassigned credential directory accepted")
	installed := filepath.Join(dir, "vault-unlock")
	must(os.Rename(marker, installed))
	p.env = append(a.environment(), "CREDENTIALS_DIRECTORY="+dir)
	p.success(nil, launcher, "check-unlock", uid)
	must(syscall.Setxattr(installed, attr, credentialACL(a.uid, 4), 0))
	require(p.run(nil, launcher, "check-unlock", uid).code != 0, "broad credential ACL accepted")
}

func restartChecks(launcher, importer, consumer string) {
	disposableNode()
	expectedUID := checked(parseFixtureID(os.Getenv("SYMPHONY_RESTART_FIXTURE_UID")))
	uid := fmt.Sprint(expectedUID)
	a := addAccount("symphonybootqa", "--uid", uid)
	require(a.uid == expectedUID, "unexpected restart account")
	must(os.Chmod(a.home, 0700))
	runtime := filepath.Join("/run", "symphony-node-keyring-"+uid)
	a.mkdir(runtime)
	defer os.RemoveAll(runtime)
	bus := filepath.Join(runtime, "bus")
	work := checked(os.MkdirTemp("/run", "symphony-restart-"))
	defer os.RemoveAll(work)
	must(os.Chmod(work, 0755))
	state := filepath.Join(work, "receipts")
	a.mkdir(state)
	tops, ref, request := uuid(), uuid(), uuid()
	value := append([]byte{0, 255}, randomBytes(42)...)
	unlock := fmt.Appendf(nil, "%x", randomBytes(32))
	enrollDir := filepath.Join("/run/credentials", "symphony-node-keyring-enroll@"+uid+".service")
	serveDir := filepath.Join("/run/credentials", "symphony-node-keyring@"+uid+".service")
	for _, dir := range []string{enrollDir, serveDir} {
		var filesystem syscall.Statfs_t
		must(syscall.Statfs(dir, &filesystem))
		require(filesystem.Type == 0x01021994, "each unit requires a private tmpfs for unlock input")
		a.chown(dir)
		must(os.Chmod(dir, 0500))
		file := filepath.Join(dir, "vault-unlock")
		must(os.WriteFile(file, unlock, 0400))
		a.chown(file)
		defer os.Remove(file)
	}
	bare := process{env: a.environment(), identity: a.credentials(), protected: [][]byte{value, unlock}}
	enroller := bare
	enroller.env = append(a.environment(), "CREDENTIALS_DIRECTORY="+enrollDir)
	p := bare
	p.env = append(a.environment(), "CREDENTIALS_DIRECTORY="+serveDir)
	refuse := func(proc process, mode string) {
		r := proc.run(nil, launcher, mode, uid)
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
	refuse(bare, "check-unlock")
	enroller.success(nil, launcher, "check-unlock", uid)
	p.success(nil, launcher, "check-unlock", uid)
	refuse(p, "enroll") // The serve unit cannot enroll a vault.
	refuse(p, "serve")  // No vault exists yet.
	enroller.success(nil, launcher, "enroll", uid)
	refuse(enroller, "enroll")
	refuse(enroller, "serve") // The enroll unit cannot start the service.
	var gen string
	withService(func() {
		in := p
		in.env = append(a.environment(), "SSH_ORIGINAL_COMMAND=symphony-keyring-import-v1")
		r := in.run(frame(request, tops, ref, value), importer, collection, tops, ref, state, bus)
		gen = generation(r, request)
	})
	vault := checked(os.ReadFile(filepath.Join(a.home, ".local/share/keyrings/login.keyring")))
	require(len(vault) > 0 && !bytes.Contains(vault, value), "unsafe persistent credential")

	credential := filepath.Join(serveDir, "vault-unlock")
	must(os.WriteFile(credential, fmt.Appendf(nil, "%x", randomBytes(32)), 0400))
	refuse(p, "serve")
	must(os.Remove(credential))
	refuse(p, "serve")
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

	credentialACLChecks(launcher)

	selection := filepath.Join(work, "binding")
	binding(selection, a, bus, tops, ref, gen)
	p.success(nil, consumer, "--expect-refusal", "unavailable", selection)
	withService(func() { p.success(value, consumer, selection) })
	fmt.Println("PASS unattended service restart, persistent generation, missing/wrong unlock refusal, and offline native use")
}

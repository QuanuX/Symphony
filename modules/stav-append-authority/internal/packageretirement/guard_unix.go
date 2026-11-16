//go:build darwin || linux

package packageretirement

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const protocol = "symphony.package-retirement-state.v1"

type state struct {
	Protocol        string   `json:"protocol"`
	Identity        Identity `json:"identity"`
	Epoch           string   `json:"epoch"`
	Device          uint64   `json:"root_device"`
	Inode           uint64   `json:"root_inode"`
	Phase           string   `json:"phase"`
	GuardDevice     uint64   `json:"guard_device"`
	GuardInode      uint64   `json:"guard_inode"`
	AdmissionDevice uint64   `json:"admission_device"`
	AdmissionInode  uint64   `json:"admission_inode"`
	UseDevice       uint64   `json:"use_device"`
	UseInode        uint64   `json:"use_inode"`
}

// Lease pins the installation root and stable lock inodes. Callers must retain
// a use lease until all owned work, including workers and open stores, ends.
type Lease struct {
	prefix                                    string
	root, directory, admission, use, mutation *os.File
	rootStat                                  unix.Stat_t
	value                                     state
	exclusive                                 bool
}

func validSegment(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == '+') {
			return false
		}
	}
	return true
}
func validRelative(s string) bool {
	if s == "" || filepath.IsAbs(s) || filepath.ToSlash(filepath.Clean(s)) != s {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." || p == ".." {
			return false
		}
	}
	return true
}
func validDigest(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(s[7:])
	return err == nil
}
func (id Identity) valid() bool {
	return validSegment(id.Owner) && validSegment(id.Version) && validDigest(id.ReceiptDigest) && validDigest(id.BinaryDigest) && id.BinarySize > 0 && validRelative(id.BinaryRelative) && validRelative(id.ReceiptRelative) && id.BinaryRelative != id.ReceiptRelative
}
func guardPath(owner, version string) string {
	return ".symphony-package-retirement/" + owner + "/" + version
}

// openRoot traverses from / using no-follow descriptors; an ancestor symlink
// cannot redirect a subsequent open or unlink into a replacement directory.
func openRoot(prefix string) (*os.File, unix.Stat_t, error) {
	var st unix.Stat_t
	if !filepath.IsAbs(prefix) || filepath.Clean(prefix) != prefix || prefix == "/" {
		return nil, st, fmt.Errorf("retirement prefix must be a canonical absolute directory")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, st, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(prefix, "/"), "/") {
		next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(fd)
		if openErr != nil {
			return nil, st, fmt.Errorf("open retirement root: %w", openErr)
		}
		fd = next
	}
	file := os.NewFile(uintptr(fd), prefix)
	if err = unix.Fstat(fd, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Mode&0o022 != 0 || st.Uid != 0 && st.Uid != uint32(os.Geteuid()) {
		_ = file.Close()
		return nil, st, fmt.Errorf("retirement root ownership or mode is unsafe")
	}
	return file, st, nil
}

func (g *Lease) openDir(relative string, create bool) (*os.File, error) {
	if relative != "." && !validRelative(relative) {
		return nil, fmt.Errorf("invalid package-relative directory")
	}
	fd, err := unix.Dup(int(g.root.Fd()))
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	for _, part := range strings.Split(relative, "/") {
		if part == "." {
			continue
		}
		if create {
			if err := unix.Mkdirat(fd, part, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
				_ = unix.Close(fd)
				return nil, err
			}
		}
		next, openErr := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil {
			_ = unix.Close(fd)
			return nil, openErr
		}
		var st unix.Stat_t
		if err := unix.Fstat(next, &st); err != nil || st.Uid != g.rootStat.Uid || st.Mode&0o022 != 0 {
			_ = unix.Close(next)
			_ = unix.Close(fd)
			return nil, fmt.Errorf("package directory ownership or mode is unsafe")
		}
		if create {
			if err := unix.Fsync(fd); err != nil {
				_ = unix.Close(next)
				_ = unix.Close(fd)
				return nil, err
			}
		}
		_ = unix.Close(fd)
		fd = next
	}
	return os.NewFile(uintptr(fd), relative), nil
}

func (g *Lease) openFile(dir *os.File, name string, create bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	if create {
		flags = unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CREAT
	}
	fd, err := unix.Openat(int(dir.Fd()), name, flags, 0o644)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), name)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 || st.Uid != g.rootStat.Uid || st.Mode&0o022 != 0 {
		_ = f.Close()
		return nil, fmt.Errorf("retirement file ownership, links, or mode is unsafe")
	}
	return f, nil
}

func open(prefix, owner, version string, create, mutation bool) (*Lease, error) {
	if !validSegment(owner) || !validSegment(version) {
		return nil, fmt.Errorf("invalid retirement owner or version")
	}
	root, st, err := openRoot(prefix)
	if err != nil {
		return nil, err
	}
	g := &Lease{prefix: prefix, root: root, rootStat: st, value: state{Identity: Identity{Owner: owner, Version: version}}}
	fail := func(err error) (*Lease, error) {
		_ = g.Close()
		return nil, fmt.Errorf("package retirement evidence unavailable: %w", err)
	}
	if mutation {
		if err := g.lockMutation(); err != nil {
			return fail(err)
		}
	}
	g.directory, err = g.openDir(guardPath(owner, version), create)
	if err != nil {
		return fail(err)
	}
	g.admission, err = g.openFile(g.directory, "admission.lock", create)
	if err != nil {
		return fail(err)
	}
	g.use, err = g.openFile(g.directory, "use.lock", create)
	if err != nil {
		return fail(err)
	}
	if create {
		if err := g.directory.Sync(); err != nil {
			return fail(err)
		}
	}
	return g, nil
}

// The generic lifecycle owner uses this same stable lock. Until native owner
// dispatch understands its claims, any ownership marker refuses mutation.
func (g *Lease) lockMutation() error {
	var err error
	g.mutation, err = g.openFile(g.root, ".symphony-lifecycle.lock", true)
	if err != nil {
		return err
	}
	if err = unix.Flock(int(g.mutation.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("package root is busy: %w", err)
	}
	if err = g.root.Sync(); err != nil {
		return err
	}
	for _, relative := range []string{".symphony-lifecycle-ownership-v1.json", "share/symphony/receipts/symphony-root-ownership/1/install-receipt.json"} {
		parent := g.root
		if filepath.Dir(relative) != "." {
			parent, err = g.openDir(filepath.ToSlash(filepath.Dir(relative)), false)
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			if err != nil {
				return fmt.Errorf("shared-root ownership evidence is unsafe: %w", err)
			}
		}
		var st unix.Stat_t
		err = unix.Fstatat(int(parent.Fd()), filepath.Base(relative), &st, unix.AT_SYMLINK_NOFOLLOW)
		if parent != g.root {
			_ = parent.Close()
		}
		if err == nil {
			return fmt.Errorf("native package mutation refuses a qxctl-administered shared root; qualified owner dispatch is required")
		}
		if !errors.Is(err, unix.ENOENT) {
			return err
		}
	}
	return nil
}

func (g *Lease) lockAdmission() error {
	if err := unix.Flock(int(g.admission.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("package admission is busy: %w", err)
	}
	return nil
}
func (g *Lease) unlockAdmission() {
	if g.admission != nil {
		_ = unix.Flock(int(g.admission.Fd()), unix.LOCK_UN)
	}
}
func (g *Lease) readState() (state, error) {
	var s state
	f, err := g.openFile(g.directory, "state.json", false)
	if err != nil {
		return s, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() <= 0 || info.Size() > 16384 {
		return s, fmt.Errorf("retirement state is unbounded")
	}
	d := json.NewDecoder(io.LimitReader(f, 16385))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return s, err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return s, fmt.Errorf("retirement state contains trailing data")
	}
	if s.Protocol != protocol || !s.Identity.valid() || len(s.Epoch) != 32 || s.Device != uint64(g.rootStat.Dev) || s.Inode != g.rootStat.Ino {
		return s, fmt.Errorf("retirement installation identity changed")
	}
	if _, err := hex.DecodeString(s.Epoch); err != nil {
		return s, fmt.Errorf("invalid retirement epoch")
	}
	if err := g.verifyInodes(s); err != nil {
		return s, err
	}
	switch s.Phase {
	case "installing", "open", "retiring", "retired":
	default:
		return s, fmt.Errorf("unknown retirement state")
	}
	return s, nil
}
func (g *Lease) writeState(s state) error {
	if os.Geteuid() != int(g.rootStat.Uid) {
		return fmt.Errorf("package retirement mutation requires the installation owner")
	}
	if err := g.CheckRoot(); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	name := ".state-" + hex.EncodeToString(nonce[:])
	fd, err := unix.Openat(int(g.directory.Fd()), name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o644)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), name)
	defer unix.Unlinkat(int(g.directory.Fd()), name, 0)
	if _, err = f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = unix.Renameat(int(g.directory.Fd()), name, int(g.directory.Fd()), "state.json"); err != nil {
		return err
	}
	if err = g.directory.Sync(); err != nil {
		return err
	}
	g.value = s
	return nil
}

func (g *Lease) bindInodes(s *state) error {
	var d, a, u unix.Stat_t
	if err := unix.Fstat(int(g.directory.Fd()), &d); err != nil {
		return err
	}
	if err := unix.Fstat(int(g.admission.Fd()), &a); err != nil {
		return err
	}
	if err := unix.Fstat(int(g.use.Fd()), &u); err != nil {
		return err
	}
	s.GuardDevice, s.GuardInode = uint64(d.Dev), d.Ino
	s.AdmissionDevice, s.AdmissionInode = uint64(a.Dev), a.Ino
	s.UseDevice, s.UseInode = uint64(u.Dev), u.Ino
	return nil
}
func (g *Lease) verifyInodes(s state) error {
	current := state{}
	if err := g.bindInodes(&current); err != nil {
		return err
	}
	if current.GuardDevice != s.GuardDevice || current.GuardInode != s.GuardInode || current.AdmissionDevice != s.AdmissionDevice || current.AdmissionInode != s.AdmissionInode || current.UseDevice != s.UseDevice || current.UseInode != s.UseInode {
		return fmt.Errorf("persistent package guard inode identity changed")
	}
	return nil
}
func (g *Lease) requireAbsent(relative string) error {
	d, err := g.openDir(filepath.ToSlash(filepath.Dir(relative)), false)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return err
	}
	defer d.Close()
	var st unix.Stat_t
	err = unix.Fstatat(int(d.Fd()), filepath.Base(relative), &st, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("existing package bytes cannot be adopted into retirement custody")
}

// PrepareInstall must only receive allowCreate when the owner has established
// that neither payload nor receipt exists. An existing package is never adopted.
func PrepareInstall(prefix string, id Identity, allowCreate bool) (*Lease, error) {
	if !id.valid() {
		return nil, fmt.Errorf("invalid package retirement identity")
	}
	g, err := open(prefix, id.Owner, id.Version, allowCreate, true)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Lease, error) { _ = g.Close(); return nil, err }
	if err = g.lockAdmission(); err != nil {
		return fail(err)
	}
	defer g.unlockAdmission()
	s, err := g.readState()
	if errors.Is(err, unix.ENOENT) && allowCreate {
		for _, relative := range []string{id.BinaryRelative, id.ReceiptRelative} {
			if absentErr := g.requireAbsent(relative); absentErr != nil {
				return fail(absentErr)
			}
		}
		var nonce [16]byte
		if _, err = rand.Read(nonce[:]); err != nil {
			return fail(err)
		}
		s = state{Protocol: protocol, Identity: id, Epoch: hex.EncodeToString(nonce[:]), Device: uint64(g.rootStat.Dev), Inode: g.rootStat.Ino, Phase: "installing"}
		if err = g.bindInodes(&s); err != nil {
			return fail(err)
		}
		g.value = s
		if err = g.writeState(s); err != nil {
			return fail(err)
		}
	} else if err != nil {
		return fail(fmt.Errorf("package retirement evidence unavailable: %w", err))
	}
	if s.Identity != id || s.Phase != "installing" && s.Phase != "open" {
		return fail(fmt.Errorf("package installation is retired, retiring, or has different identity"))
	}
	g.value = s
	if err = unix.Flock(int(g.use.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fail(fmt.Errorf("package installation is in use: %w", err))
	}
	g.exclusive = true
	return g, nil
}

func AcquireUse(prefix string, id Identity) (*Lease, error) { return acquireUse(prefix, id, false) }

// AcquireRecoveryUse preserves the existing native controller's recovery/drain
// custody while admission is closed. It never authorizes a new service start.
func AcquireRecoveryUse(prefix string, id Identity) (*Lease, error) {
	return acquireUse(prefix, id, true)
}
func acquireUse(prefix string, id Identity, recovery bool) (*Lease, error) {
	if !id.valid() {
		return nil, fmt.Errorf("invalid package retirement identity")
	}
	g, err := open(prefix, id.Owner, id.Version, false, false)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Lease, error) { _ = g.Close(); return nil, err }
	if err = g.lockAdmission(); err != nil {
		return fail(err)
	}
	defer g.unlockAdmission()
	s, err := g.readState()
	if err != nil {
		return fail(err)
	}
	if s.Identity != id || s.Phase != "open" && !(recovery && s.Phase == "retiring") {
		return fail(fmt.Errorf("package admission is closed or installation identity differs"))
	}
	g.value = s
	if err = g.CheckRoot(); err != nil {
		return fail(err)
	}
	if err = unix.Flock(int(g.use.Fd()), unix.LOCK_SH|unix.LOCK_NB); err != nil {
		return fail(fmt.Errorf("package reclamation owns admission: %w", err))
	}
	return g, nil
}

func BeginRetirement(prefix string, id Identity) (*Lease, error) {
	if !id.valid() {
		return nil, fmt.Errorf("invalid package retirement identity")
	}
	g, err := open(prefix, id.Owner, id.Version, false, true)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Lease, error) { _ = g.Close(); return nil, err }
	if err = g.lockAdmission(); err != nil {
		return fail(err)
	}
	s, err := g.readState()
	if err != nil {
		g.unlockAdmission()
		return fail(err)
	}
	if s.Identity != id || s.Phase == "installing" {
		g.unlockAdmission()
		return fail(fmt.Errorf("package retirement identity is not admitted"))
	}
	g.value = s
	if s.Phase == "open" {
		s.Phase = "retiring"
		err = g.writeState(s)
	}
	g.unlockAdmission()
	if err != nil {
		return fail(err)
	}
	// Never wait for a live consumer while holding the admission lock. A busy
	// consumer leaves the durable retiring state intact for exact owner recovery.
	if err = unix.Flock(int(g.use.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fail(fmt.Errorf("package use is still live; admission remains closed: %w", err))
	}
	g.exclusive = true
	if err = g.CheckRoot(); err != nil {
		return fail(err)
	}
	return g, nil
}

func Inspect(prefix, owner, version string) (Identity, string, error) {
	g, err := open(prefix, owner, version, false, false)
	if err != nil {
		return Identity{}, "", err
	}
	defer g.Close()
	if err = g.lockAdmission(); err != nil {
		return Identity{}, "", err
	}
	defer g.unlockAdmission()
	s, err := g.readState()
	if err != nil {
		return Identity{}, "", err
	}
	g.value = s
	if s.Identity.Owner != owner || s.Identity.Version != version {
		return Identity{}, "", fmt.Errorf("package guard owner differs")
	}
	if err = g.CheckRoot(); err != nil {
		return Identity{}, "", err
	}
	return s.Identity, s.Phase, nil
}
func (g *Lease) State() string { return g.value.Phase }
func (g *Lease) transition(from, to string) error {
	if !g.exclusive {
		return fmt.Errorf("exclusive package ownership required")
	}
	if err := g.lockAdmission(); err != nil {
		return err
	}
	defer g.unlockAdmission()
	s, err := g.readState()
	if err != nil {
		return err
	}
	if s.Epoch != g.value.Epoch || s.Identity != g.value.Identity {
		return fmt.Errorf("package admission generation changed")
	}
	if s.Phase == to {
		g.value = s
		return nil
	}
	if s.Phase != from {
		return fmt.Errorf("invalid package retirement transition")
	}
	s.Phase = to
	return g.writeState(s)
}
func (g *Lease) Activate() error { return g.transition("installing", "open") }
func (g *Lease) FinishRetirement() error {
	for _, relative := range []string{g.value.Identity.BinaryRelative, g.value.Identity.ReceiptRelative} {
		if err := g.requireAbsent(relative); err != nil {
			return err
		}
	}
	return g.transition("retiring", "retired")
}

func sameFile(a, b *os.File) bool {
	var x, y unix.Stat_t
	return unix.Fstat(int(a.Fd()), &x) == nil && unix.Fstat(int(b.Fd()), &y) == nil && x.Dev == y.Dev && x.Ino == y.Ino && x.Nlink == y.Nlink
}
func (g *Lease) CheckRoot() error {
	if g == nil || g.root == nil {
		return fmt.Errorf("package lease is closed")
	}
	f, _, err := openRoot(g.prefix)
	if err != nil {
		return err
	}
	same := sameFile(f, g.root)
	_ = f.Close()
	if !same {
		return fmt.Errorf("package root was replaced")
	}
	d, err := g.openDir(guardPath(g.value.Identity.Owner, g.value.Identity.Version), false)
	if err != nil {
		return err
	}
	defer d.Close()
	if !sameFile(d, g.directory) {
		return fmt.Errorf("package guard directory was replaced")
	}
	if g.mutation != nil {
		f, err := g.openFile(g.root, ".symphony-lifecycle.lock", false)
		if err != nil {
			return err
		}
		same := sameFile(f, g.mutation)
		_ = f.Close()
		if !same {
			return fmt.Errorf("package root lock was replaced")
		}
	}
	for name, old := range map[string]*os.File{"admission.lock": g.admission, "use.lock": g.use} {
		f, err := g.openFile(d, name, false)
		if err != nil {
			return err
		}
		same := sameFile(f, old)
		_ = f.Close()
		if !same {
			return fmt.Errorf("package lock inode was replaced")
		}
	}
	return nil
}

func (g *Lease) file(relative, digest string, size uint64, remove bool) error {
	if relative != g.value.Identity.BinaryRelative && relative != g.value.Identity.ReceiptRelative {
		return fmt.Errorf("file is outside the exact package retirement identity")
	}
	if relative == g.value.Identity.BinaryRelative && (digest != g.value.Identity.BinaryDigest || size != g.value.Identity.BinarySize) {
		return fmt.Errorf("executable evidence differs from admitted identity")
	}
	if !validRelative(relative) || !validDigest(digest) || size == 0 {
		return fmt.Errorf("invalid owned file evidence")
	}
	if remove && (!g.exclusive || g.value.Phase != "retiring") {
		return fmt.Errorf("closed retirement and exclusive reclamation required")
	}
	if err := g.CheckRoot(); err != nil {
		return err
	}
	d, err := g.openDir(filepath.ToSlash(filepath.Dir(relative)), false)
	if err != nil {
		return err
	}
	defer d.Close()
	f, err := g.openFile(d, filepath.Base(relative), false)
	if errors.Is(err, unix.ENOENT) && remove {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || uint64(info.Size()) != size {
		return fmt.Errorf("owned package file size changed")
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if "sha256:"+hex.EncodeToString(h.Sum(nil)) != digest {
		return fmt.Errorf("owned package file digest changed")
	}
	if !remove {
		return nil
	}
	if err := g.CheckRoot(); err != nil {
		return err
	}
	current, err := g.openDir(filepath.ToSlash(filepath.Dir(relative)), false)
	if err != nil {
		return err
	}
	same := sameFile(current, d)
	_ = current.Close()
	if !same {
		return fmt.Errorf("owned package directory was replaced")
	}
	currentFile, err := g.openFile(d, filepath.Base(relative), false)
	if err != nil {
		return err
	}
	same = sameFile(currentFile, f)
	_ = currentFile.Close()
	if !same {
		return fmt.Errorf("owned package file was replaced")
	}
	if err := unix.Unlinkat(int(d.Fd()), filepath.Base(relative), 0); err != nil {
		return err
	}
	return d.Sync()
}
func (g *Lease) VerifyFile(relative, digest string, size uint64) error {
	return g.file(relative, digest, size, false)
}
func (g *Lease) RemoveFile(relative, digest string, size uint64) error {
	return g.file(relative, digest, size, true)
}
func (g *Lease) Close() error {
	if g == nil {
		return nil
	}
	var result error
	for _, p := range []**os.File{&g.use, &g.admission, &g.directory, &g.mutation, &g.root} {
		if *p != nil {
			result = errors.Join(result, (*p).Close())
			*p = nil
		}
	}
	return result
}

// ReadFile returns bounded bytes from the pinned, trusted package tree. Receipt
// semantic validation remains with the native owner while this lease is held.
func (g *Lease) ReadFile(relative string, limit int64) ([]byte, error) {
	if !validRelative(relative) || limit <= 0 {
		return nil, fmt.Errorf("invalid bounded package read")
	}
	if err := g.CheckRoot(); err != nil {
		return nil, err
	}
	d, err := g.openDir(filepath.ToSlash(filepath.Dir(relative)), false)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	f, err := g.openFile(d, filepath.Base(relative), false)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("package file exceeds its read bound")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("bounded package read failed: %w", err)
	}
	if err := g.CheckRoot(); err != nil {
		return nil, err
	}
	return data, nil
}

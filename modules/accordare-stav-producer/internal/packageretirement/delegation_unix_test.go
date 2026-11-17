//go:build darwin || linux

package packageretirement

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

type dispatchFixture struct {
	prefix     string
	id         Identity
	root, lock *os.File
	proof      DispatchProof
	registry   dispatchRegistry
}

func newDispatchFixture(t *testing.T, operation string) *dispatchFixture {
	t.Helper()
	f := &dispatchFixture{}
	if operation == "uninstall" {
		f.prefix, f.id = fixture(t)
	} else {
		var err error
		f.prefix, err = filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		f.id = Identity{Owner: "fixture-owner", Version: "1", ReceiptDigest: testHash([]byte("receipt")), BinaryDigest: testHash([]byte("binary")), BinarySize: 6, BinaryRelative: "libexec/owner", ReceiptRelative: "share/receipt"}
	}
	root, st, err := openRoot(f.prefix)
	if err != nil {
		t.Fatal(err)
	}
	f.root = root
	g := &Lease{prefix: f.prefix, root: root, rootStat: st}
	f.lock, err = g.openFile(root, ".symphony-lifecycle.lock", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(int(f.lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	var lst unix.Stat_t
	if err := unix.Fstat(int(f.lock.Fd()), &lst); err != nil {
		t.Fatal(err)
	}
	f.proof = DispatchProof{Protocol: DispatchCapability, FormatVersion: 1, Operation: operation, ComponentID: f.id.Owner, Version: f.id.Version, InstallRoot: f.prefix, ReceiptDigest: f.id.ReceiptDigest, BinaryDigest: f.id.BinaryDigest, ControlDomainID: "sha256:" + strings.Repeat("a", 64), TOPSID: "123e4567-e89b-42d3-a456-426614174000", ProfileID: "default", OwnershipGeneration: 1, RootDevice: uint64(st.Dev), RootInode: st.Ino, LockDevice: uint64(lst.Dev), LockInode: lst.Ino, Nonce: strings.Repeat("b", 32)}
	disposition := "retained"
	observed := []string{}
	if operation == "uninstall" {
		disposition = "retiring"
		observed = append(observed, f.id.ReceiptDigest)
	}
	f.registry = dispatchRegistry{Protocol: "symphony.knowledge.lifecycle-root-ownership.v1", FormatVersion: 1, InstallRoot: f.prefix, EnforcementState: "enforced", Generation: 1, Claims: []dispatchClaim{{ClaimKind: "profile", ControlDomainID: f.proof.ControlDomainID, TOPSID: f.proof.TOPSID, ProfileID: f.proof.ProfileID, ComponentID: f.id.Owner, ReceiptDigest: f.id.ReceiptDigest, Disposition: disposition}}, Observed: observed, Released: []string{}, UpdatedAt: "2026-11-16T00:00:00Z"}
	f.writeRegistry(t)
	fence := filepath.Join(f.prefix, ownershipFencePath)
	if err := os.MkdirAll(filepath.Dir(fence), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fence, []byte(ownershipFenceBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.lock.Close(); f.root.Close() })
	return f
}

func (f *dispatchFixture) writeRegistry(t *testing.T) {
	t.Helper()
	for i := range f.registry.Claims {
		c := &f.registry.Claims[i]
		c.ClaimID = dispatchDigest([]byte(strings.Join([]string{c.ClaimKind, c.ControlDomainID, c.TOPSID, c.ProfileID, c.ComponentID, c.ReceiptDigest, c.Disposition}, "\n")))
	}
	sort.Slice(f.registry.Claims, func(i, j int) bool { return f.registry.Claims[i].ClaimID < f.registry.Claims[j].ClaimID })
	raw, _ := json.Marshal(f.registry)
	var obj map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	delete(obj, "ownership_registry_digest")
	canonical, _ := json.Marshal(obj)
	f.registry.Digest = dispatchDigest(canonical)
	f.proof.OwnershipRegistryDigest = f.registry.Digest
	f.proof.OwnershipGeneration = f.registry.Generation
	raw, _ = json.Marshal(f.registry)
	if err := os.WriteFile(filepath.Join(f.prefix, ownershipRegistryPath), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f *dispatchFixture) proofFile(t *testing.T) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(f.proof)
	if _, err = w.Write(raw); err != nil {
		t.Fatal(err)
	}
	w.Close()
	t.Cleanup(func() { r.Close() })
	return r
}
func (f *dispatchFixture) read(t *testing.T) (*Delegation, error) {
	t.Helper()
	r := f.proofFile(t)
	return ReadDispatch(int(f.root.Fd()), int(f.lock.Fd()), int(r.Fd()), f.proof.Operation, f.id.Owner, f.id.Version, f.prefix)
}

func TestDelegatedInstallRetainsParentCustody(t *testing.T) {
	f := newDispatchFixture(t, "install")
	d, err := f.read(t)
	if err != nil {
		t.Fatal(err)
	}
	g, err := PrepareInstall(f.prefix, f.id, true, d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.PublishFile(f.id.BinaryRelative, strings.NewReader("binary"), f.id.BinaryDigest, 6, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err = g.PublishFile(f.id.ReceiptRelative, strings.NewReader("receipt"), f.id.ReceiptDigest, 7, 0o644); err != nil {
		t.Fatal(err)
	}
	if err = g.Activate(); err != nil {
		t.Fatal(err)
	}
	g.Close()
	d.Close()
	fresh, err := os.OpenFile(filepath.Join(f.prefix, ".symphony-lifecycle.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	if err = unix.Flock(int(fresh.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
		t.Fatal("child close released parent root lock")
	}
	raw, err := os.ReadFile(filepath.Join(f.prefix, ownershipRegistryPath))
	if err != nil {
		t.Fatal(err)
	}
	if err = validateDelegatedRegistry(raw, f.proof); err != nil {
		t.Fatalf("native mutation changed registry: %v", err)
	}
}

func TestDelegatedRetirementPreservesUseFence(t *testing.T) {
	f := newDispatchFixture(t, "uninstall")
	d, err := f.read(t)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	use, err := AcquireUse(f.prefix, f.id)
	if err != nil {
		t.Fatal(err)
	}
	if g, err := BeginRetirement(f.prefix, f.id, d); err == nil {
		g.Close()
		t.Fatal("reclaimed live owner")
	}
	if g, err := AcquireUse(f.prefix, f.id); err == nil {
		g.Close()
		t.Fatal("late start admitted")
	}
	use.Close()
	g, err := BeginRetirement(f.prefix, f.id, d)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	raw, err := g.ReadFile(f.id.ReceiptRelative, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if err = g.RemoveFile(f.id.BinaryRelative, f.id.BinaryDigest, f.id.BinarySize); err != nil {
		t.Fatal(err)
	}
	if err = g.RemoveFile(f.id.ReceiptRelative, testHash(raw), uint64(len(raw))); err != nil {
		t.Fatal(err)
	}
	if err = g.FinishRetirement(); err != nil {
		t.Fatal(err)
	}
}

func TestDelegationRejectsForeignClaimsAndChangedRegistry(t *testing.T) {
	t.Run("foreign retained", func(t *testing.T) {
		f := newDispatchFixture(t, "uninstall")
		c := f.registry.Claims[0]
		c.ProfileID = "other"
		c.Disposition = "retained"
		f.registry.Claims = append(f.registry.Claims, c)
		f.writeRegistry(t)
		if d, err := f.read(t); err == nil {
			d.Close()
			t.Fatal("foreign retained claim accepted")
		}
	})
	t.Run("changed after read", func(t *testing.T) {
		f := newDispatchFixture(t, "install")
		d, err := f.read(t)
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()
		old := f.registry.Digest
		f.registry.Previous = &old
		f.registry.Generation++
		f.writeRegistry(t)
		if g, err := PrepareInstall(f.prefix, f.id, true, d); err == nil {
			g.Close()
			t.Fatal("stale registry delegated")
		}
		if _, err := os.Stat(filepath.Join(f.prefix, ".symphony-package-retirement")); !os.IsNotExist(err) {
			t.Fatal("stale delegation created guard")
		}
	})
}

func TestDelegationRejectsFalseCustody(t *testing.T) {
	for _, name := range []string{"unlocked", "other description", "root inode", "registry digest", "component", "nonce", "operation"} {
		t.Run(name, func(t *testing.T) {
			f := newDispatchFixture(t, "install")
			switch name {
			case "unlocked":
				if err := unix.Flock(int(f.lock.Fd()), unix.LOCK_UN); err != nil {
					t.Fatal(err)
				}
			case "other description":
				originalLock := f.lock
				t.Cleanup(func() { originalLock.Close() })
				other, err := os.OpenFile(filepath.Join(f.prefix, ".symphony-lifecycle.lock"), os.O_RDWR, 0)
				if err != nil {
					t.Fatal(err)
				}
				defer other.Close()
				f.lock = other
			case "root inode":
				f.proof.RootInode++
			case "registry digest":
				f.proof.OwnershipRegistryDigest = "sha256:" + strings.Repeat("c", 64)
			case "component":
				f.proof.ComponentID = "other"
			case "nonce":
				f.proof.Nonce = "not-a-nonce"
			case "operation":
				f.proof.Operation = "purge"
			}
			if d, err := f.read(t); err == nil {
				d.Close()
				t.Fatal("false custody accepted")
			}
		})
	}
}

func TestDelegationFlagsAndStandaloneFence(t *testing.T) {
	if d, err := ReadDispatch(-1, -1, -1, "install", "owner", "1", "/ignored"); d != nil || err != nil {
		t.Fatal("absent flags must remain standalone")
	}
	for _, fds := range [][3]int{{3, -1, 5}, {3, 3, 5}, {0, 4, 5}} {
		if d, err := ReadDispatch(fds[0], fds[1], fds[2], "install", "owner", "1", "/ignored"); err == nil {
			d.Close()
			t.Fatal("invalid descriptors accepted")
		}
	}
	for _, raw := range []string{`{"protocol":"one","protocol":"two"}`, `{} {}`, `{"unknown":true}`} {
		var p DispatchProof
		if err := strictDispatchDecode([]byte(raw), &p); err == nil {
			t.Fatalf("ambiguous proof accepted: %s", raw)
		}
	}
	f := newDispatchFixture(t, "install")
	f.lock.Close()
	if g, err := PrepareInstall(f.prefix, f.id, true); err == nil {
		g.Close()
		t.Fatal("standalone mutation bypassed ownership markers")
	}
}

func TestDelegationChild(t *testing.T) {
	if os.Getenv("SYMPHONY_DISPATCH_TEST_CHILD") == "" {
		return
	}
	var id Identity
	if err := json.Unmarshal([]byte(os.Getenv("SYMPHONY_DISPATCH_TEST_ID")), &id); err != nil {
		t.Fatal(err)
	}
	prefix := os.Getenv("SYMPHONY_DISPATCH_TEST_ROOT")
	d, err := ReadDispatch(3, 4, 5, "install", id.Owner, id.Version, prefix)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	g, err := PrepareInstall(prefix, id, true, d)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
}

func TestDelegatedDescriptorsCrossExec(t *testing.T) {
	f := newDispatchFixture(t, "install")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(f.id)
	cmd := exec.Command(self, "-test.run=^TestDelegationChild$")
	cmd.Env = append(os.Environ(), "SYMPHONY_DISPATCH_TEST_CHILD=1", "SYMPHONY_DISPATCH_TEST_ROOT="+f.prefix, "SYMPHONY_DISPATCH_TEST_ID="+string(body))
	cmd.ExtraFiles = []*os.File{f.root, f.lock, f.proofFile(t)}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child failed: %v\n%s", err, out)
	}
	if _, phase, err := Inspect(f.prefix, f.id.Owner, f.id.Version); err != nil || phase != "installing" {
		t.Fatalf("child did not prepare exact install: %s %v", phase, err)
	}
}

func TestDelegationRequiresExactTypedFields(t *testing.T) {
	f := newDispatchFixture(t, "install")
	registryRaw, _ := json.Marshal(f.registry)
	proofRaw, _ := json.Marshal(f.proof)
	for _, test := range []struct {
		name string
		raw  []byte
		out  any
	}{
		{"null canonical", bytes.Replace(registryRaw, []byte(`"canonical":false`), []byte(`"canonical":null`), 1), &dispatchRegistry{}},
		{"null root device", bytes.Replace(proofRaw, []byte(`"root_device":`), []byte(`"root_device":null,"ignored_device":`), 1), &DispatchProof{}},
		{"case aliased field", bytes.Replace(proofRaw, []byte(`"protocol":`), []byte(`"Protocol":"ignored","protocol":`), 1), &DispatchProof{}},
		{"case aliased claim", bytes.Replace(registryRaw, []byte(`"claim_id":`), []byte(`"Claim_ID":"ignored","claim_id":`), 1), &dispatchRegistry{}},
		{"missing claim key", bytes.Replace(registryRaw, []byte(`"claim_kind":"profile",`), nil, 1), &dispatchRegistry{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := strictDispatchDecode(test.raw, test.out); err == nil {
				t.Fatal("ambiguous typed fields accepted")
			}
		})
	}
}

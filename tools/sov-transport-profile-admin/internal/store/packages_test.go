//go:build darwin || linux

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	c "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/contract"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func rawHash(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func packageFixture(t *testing.T, root string, role string, digit string) c.PackageRef {
	t.Helper()
	id := strings.Repeat(digit, 8) + "-" + strings.Repeat(digit, 4) + "-4" + strings.Repeat(digit, 3) + "-8" + strings.Repeat(digit, 3) + "-" + strings.Repeat(digit, 12)
	dir := filepath.Join(root, id, "0.1.0-dev")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	r := receiptV2{Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2, ComponentID: role, ComponentKind: "adapter", ModuleID: role, PackageID: role, Version: "0.1.0-dev", InstallScope: "prefix", PrefixMode: "installation_prefix", Files: []receiptV2File{{Path: "binary", Kind: "executable", Size: 7, Digest: rawHash([]byte("fixture"))}}, EntryPoints: []receiptV2EntryPoint{{EntryPointID: "main", Kind: "executable", Path: "binary", Protocols: []string{}}}, ProvidesCapabilities: []string{}, RequiresCapabilities: []string{}, CompatibleReceptors: []string{}, PlatformRequirements: []receiptV2Platform{{OS: osName, Architecture: runtime.GOARCH, Critical: true}}}
	if role == "carrier" {
		r.EntryPoints[0].Protocols = []string{"symphony.sov.ssh-endpoint.v2", "symphony.sov.ssh-invoke-input.v2", "symphony.sov.ssh-carrier-result.v2"}
	}
	b, _ := json.Marshal(r)
	var object map[string]json.RawMessage
	_ = json.Unmarshal(b, &object)
	delete(object, "receipt_digest")
	b, _ = c.Canonical(object)
	r.ReceiptDigest = rawHash(b)
	receipt, _ := c.Canonical(r)
	evidence := []byte("Offline fixture only; no vendor qualification")
	files := map[string][]byte{"binary": []byte("fixture"), "receipt.json": receipt, "qualification.txt": evidence}
	m := c.PackageManifest{Protocol: c.PackageProtocol, PackageID: id, Version: "0.1.0-dev", Role: role, Platform: runtime.GOOS + "-" + runtime.GOARCH, Executable: "binary", ReceiptPath: "receipt.json", ReceiptDigest: rawHash(receipt), QualificationPath: "qualification.txt", QualificationRef: rawHash(evidence), ReceiptComponentID: role, ReceiptVersion: "0.1.0-dev", ReceiptEntryPointID: "main", Files: []c.PackageFile{}}
	for _, name := range []string{"binary", "receipt.json", "qualification.txt"} {
		data := files[name]
		mode := os.FileMode(0644)
		if name == "binary" {
			mode = 0755
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, mode); err != nil {
			t.Fatal(err)
		}
		m.Files = append(m.Files, c.PackageFile{Path: name, Digest: rawHash(data), Size: strconv.Itoa(len(data)), Executable: name == "binary"})
	}
	m.Digest = c.Digest(c.PackageProtocol, m)
	b, _ = c.Canonical(m)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	return c.PackageRef{PackageID: id, Version: m.Version, ManifestDigest: m.Digest, ReceiptDigest: m.ReceiptDigest, ExecutableDigest: rawHash([]byte("fixture"))}
}
func enableFixture(t *testing.T) (*Store, c.Plan, string) {
	s, p, _ := fixture(t)
	root := t.TempDir()
	f, e := os.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	s.packages = &directory{file: f, owner: uint32(os.Geteuid())}
	config := p.Change.Configuration
	config.Platform = runtime.GOOS + "-" + runtime.GOARCH
	config.Adapter = packageFixture(t, root, "carrier", "6")
	config.SSH = packageFixture(t, root, "ssh", "7")
	config.Helper = packageFixture(t, root, "proxy", "8")
	p, e = c.MakePlan(p.Scope, p.ProfileID, p.OperationID, s.administrator, nil, p.Change, s.now())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	enabled, e := c.MakePlan(p.Scope, p.ProfileID, "99999999-9999-4999-8999-999999999999", s.administrator, &p.Desired, c.Change{Action: "enable"}, s.now())
	if e != nil {
		t.Fatal(e)
	}
	return s, enabled, root
}
func TestProtectedPackageClosureAndRecoveryDrift(t *testing.T) {
	for _, fault := range []string{"none", "state_published", "outcome_archived"} {
		t.Run(fault, func(t *testing.T) {
			s, p, root := enableFixture(t)
			if fault == "none" {
				if e := s.apply(context.Background(), p); e != nil {
					t.Fatal(e)
				}
				return
			}
			s.fault = func(at string) error {
				if at == fault {
					return errors.New("crash")
				}
				return nil
			}
			if s.apply(context.Background(), p) == nil {
				t.Fatal("fault not reached")
			}
			s.fault = nil
			file := filepath.Join(root, p.Desired.Profile.Configuration.Helper.PackageID, "0.1.0-dev", "binary")
			if e := os.WriteFile(file, []byte("drifted"), 0755); e != nil {
				t.Fatal(e)
			}
			pending, e := s.readPending("pending.json", s.dir)
			if e != nil || pending == nil {
				t.Fatal(e)
			}
			if s.recover(context.Background(), c.Recovery{OperationID: p.OperationID, PlanDigest: p.Digest, ExpectedPendingDigest: pending.Digest, Action: "complete"}) == nil {
				t.Fatal("package drift cleared pending admission block")
			}
		})
	}
	for _, change := range []string{"extra-file", "missing-evidence", "writable-ancestor", "forged-receipt"} {
		t.Run(change, func(t *testing.T) {
			s, p, root := enableFixture(t)
			ref := p.Desired.Profile.Configuration.Helper
			dir := filepath.Join(root, ref.PackageID, ref.Version)
			switch change {
			case "extra-file":
				_ = os.WriteFile(filepath.Join(dir, "extra"), []byte("extra"), 0644)
			case "missing-evidence":
				_ = os.Remove(filepath.Join(dir, "qualification.txt"))
			case "writable-ancestor":
				_ = os.Chmod(dir, 0777)
			case "forged-receipt":
				raw, _ := os.ReadFile(filepath.Join(dir, "manifest.json"))
				var m c.PackageManifest
				_ = c.Decode(raw, &m)
				fake := []byte(`{"unverified":true}`)
				_ = os.WriteFile(filepath.Join(dir, "receipt.json"), fake, 0644)
				for i := range m.Files {
					if m.Files[i].Path == m.ReceiptPath {
						m.Files[i].Digest = rawHash(fake)
						m.Files[i].Size = strconv.Itoa(len(fake))
					}
				}
				m.ReceiptDigest = rawHash(fake)
				m.Digest = c.Digest(c.PackageProtocol, m)
				raw, _ = c.Canonical(m)
				_ = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0644)
				ref.ManifestDigest = m.Digest
				ref.ReceiptDigest = m.ReceiptDigest
			}
			if verifyPackage(s.packages, ref, "proxy", runtime.GOOS+"-"+runtime.GOARCH) == nil {
				t.Fatal("untrusted package accepted")
			}
		})
	}
}

func TestCarrierReceiptCannotClaimV2FromV1(t *testing.T) {
	root := t.TempDir()
	ref := packageFixture(t, root, "carrier", "6")
	dir := filepath.Join(root, ref.PackageID, ref.Version)
	raw, e := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	var manifest c.PackageManifest
	if c.Decode(raw, &manifest) != nil {
		t.Fatal("manifest fixture")
	}
	raw, e = os.ReadFile(filepath.Join(dir, "receipt.json"))
	if e != nil {
		t.Fatal(e)
	}
	var receipt receiptV2
	if c.Decode(raw, &receipt) != nil {
		t.Fatal("receipt fixture")
	}
	receipt.EntryPoints[0].Protocols = []string{"symphony.sov.ssh-endpoint.v1"}
	raw, _ = c.Canonical(receipt)
	var body map[string]json.RawMessage
	_ = json.Unmarshal(raw, &body)
	delete(body, "receipt_digest")
	raw, _ = c.Canonical(body)
	receipt.ReceiptDigest = rawHash(raw)
	raw, _ = c.Canonical(receipt)
	if verifyReceipt(raw, manifest) == nil {
		t.Fatal("v1-only carrier admitted for v2 profile")
	}
}

func TestConnectionSnapshotLockAndRefusals(t *testing.T) {
	for _, scenario := range []string{"allowed", "root", "wrong-uid", "wrong-gid", "uid-sentinel", "gid-sentinel", "negative-gid", "stale", "scope", "disabled", "pending", "corrupt-pending", "package-drift", "callback-error"} {
		t.Run(scenario, func(t *testing.T) {
			s, plan, root := enableFixture(t)
			if err := s.apply(context.Background(), plan); err != nil {
				t.Fatal(err)
			}
			scope, digest := s.scope, plan.Desired.Digest
			uid, gid := 501, 20
			switch scenario {
			case "root":
				uid = 0
			case "wrong-uid":
				uid = 502
			case "wrong-gid":
				gid = 21
			case "uid-sentinel":
				uid = int(^uint32(0))
			case "gid-sentinel":
				gid = int(^uint32(0))
			case "negative-gid":
				gid = -1
			case "stale":
				digest = "sha256:" + strings.Repeat("f", 64)
			case "scope":
				scope.HostID = "ffffffff-ffff-4fff-8fff-ffffffffffff"
			case "disabled":
				p, e := c.MakePlan(s.scope, plan.ProfileID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", s.administrator, &plan.Desired, c.Change{Action: "disable"}, s.now())
				if e != nil {
					t.Fatal(e)
				}
				if e = s.apply(context.Background(), p); e != nil {
					t.Fatal(e)
				}
				digest = p.Desired.Digest
			case "pending":
				p, e := c.MakePlan(s.scope, plan.ProfileID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", s.administrator, &plan.Desired, c.Change{Action: "disable"}, s.now())
				if e != nil {
					t.Fatal(e)
				}
				s.fault = func(stage string) error {
					if stage == "intent_prepared" {
						return errors.New("interrupted")
					}
					return nil
				}
				if e = s.apply(context.Background(), p); e == nil {
					t.Fatal("fault not observed")
				}
			case "corrupt-pending":
				if e := s.dir.write("pending.json", []byte("corrupt interrupted state")); e != nil {
					t.Fatal(e)
				}
			case "package-drift":
				ref := plan.Desired.Profile.Configuration.Helper
				if e := os.WriteFile(filepath.Join(root, ref.PackageID, ref.Version, "binary"), []byte("drifted"), 0755); e != nil {
					t.Fatal(e)
				}
			}
			called := false
			callbackErr := errors.New("dispatch intent refused")
			err := s.withConnectionSnapshot(scope, plan.ProfileID, digest, uid, gid, func(snapshot c.Snapshot) error {
				called = true
				if snapshot.Digest != plan.Desired.Digest {
					t.Fatal("snapshot changed")
				}
				if release, e := s.dir.lock(true); e == nil {
					release()
					t.Fatal("writer admitted during read callback")
				}
				if scenario == "callback-error" {
					return callbackErr
				}
				return nil
			})
			wantCall := scenario == "allowed" || scenario == "callback-error"
			if called != wantCall || (err == nil) != (scenario == "allowed") {
				t.Fatalf("called=%v error=%v", called, err)
			}
			if scenario == "callback-error" && !errors.Is(err, callbackErr) {
				t.Fatal("callback failure lost")
			}
			release, e := s.dir.lock(true)
			if e != nil {
				t.Fatal("read lock leaked", e)
			}
			release()
		})
	}
}

func TestConnectionSnapshotUsesKernelIdentity(t *testing.T) {
	s, p, _ := enableFixture(t)
	if e := s.apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	t.Setenv("SUDO_UID", "501")
	t.Setenv("SUDO_GID", "20")
	t.Setenv("SUDO_USER", "permitted-user")
	called := false
	err := s.WithConnectionSnapshot(s.scope, p.ProfileID, p.Desired.Digest, func(c.Snapshot) error { called = true; return nil })
	expected := os.Geteuid() == 501 && os.Getegid() == 20
	if called != expected || (err == nil) != expected {
		t.Fatalf("environment changed kernel identity admission: called=%v error=%v", called, err)
	}
}

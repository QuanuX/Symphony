//go:build darwin || linux

package profile

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	v1 "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	c "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profilewire"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
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
func example() Endpoint {
	d := strings.Repeat("0", 64)
	key := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(append([]byte{0, 0, 0, 11, 's', 's', 'h', '-', 'e', 'd', '2', '5', '5', '1', '9', 0, 0, 0, 32}, make([]byte, 32)...))
	return Example(v1.Endpoint{Protocol: v1.EndpointProtocol, SSH: v1.Executable{"/usr/bin/ssh", d}, SSHExecution: "protected_system_path", Host: "ssh.example.invalid", Port: "22", User: "example", HostPublicKey: key, ClientPublicKey: key, AgentSocket: "/private/agent.sock", TargetLauncher: v1.Executable{"/remote/launcher", d}})
}
func fixture(t *testing.T) (*Store, Endpoint, string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("ordinary user contract fixture")
	}
	root := t.TempDir()
	for _, name := range []string{"profiles", "packages"} {
		if err := os.Mkdir(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "LOCK"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	d := &directory{file: f, owner: uint32(os.Geteuid())}
	e := example()
	s := &Store{dir: d, scope: e.Scope}
	s.profiles, err = d.child("profiles")
	if err != nil {
		t.Fatal(err)
	}
	s.packages, err = d.child("packages")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	cfg := &e.Snapshot.Profile.Configuration
	cfg.Adapter = packageFixture(t, filepath.Join(root, "packages"), "carrier", "6")
	cfg.SSH = packageFixture(t, filepath.Join(root, "packages"), "ssh", "7")
	cfg.Helper = packageFixture(t, filepath.Join(root, "packages"), "proxy", "8")
	e.Direct.SSH = v1.Executable{Path: filepath.Join(PackageRoot(), cfg.SSH.PackageID, cfg.SSH.Version, "binary"), SHA256: strings.TrimPrefix(cfg.SSH.ExecutableDigest, "sha256:")}
	e.Snapshot.Policy.Permitted = []c.Principal{{UID: strconv.Itoa(os.Geteuid()), GID: strconv.Itoa(os.Getegid())}}
	c.SealSnapshot(&e.Snapshot)
	save(t, s, e.Snapshot)
	return s, e, root
}
func save(t *testing.T, s *Store, v c.Snapshot) {
	t.Helper()
	b, _ := c.Canonical(v)
	if err := os.WriteFile(filepath.Join(s.dir.file.Name(), "profiles", v.Profile.ProfileID+".json"), b, 0644); err != nil {
		t.Fatal(err)
	}
}
func TestProfileAdmissionLockAndDrift(t *testing.T) {
	for _, scenario := range []string{"ok", "stale", "disabled", "removed", "policy", "pending", "package", "scope", "callback"} {
		t.Run(scenario, func(t *testing.T) {
			s, e, root := fixture(t)
			v := e.Snapshot
			switch scenario {
			case "stale":
				v.Profile.Generation = "2"
				v.Policy.ProfileGeneration = "2"
			case "disabled", "removed":
				v.Profile.State = scenario
			case "policy":
				v.Policy.Permitted[0].UID = "4294967294"
			case "pending":
				if err := os.WriteFile(filepath.Join(root, "pending.json"), nil, 0644); err != nil {
					t.Fatal(err)
				}
			case "package":
				ref := v.Profile.Configuration.Helper
				if err := os.WriteFile(filepath.Join(root, "packages", ref.PackageID, ref.Version, "binary"), []byte("drifted"), 0755); err != nil {
					t.Fatal(err)
				}
			case "scope":
				e.Scope.HostID = "ffffffff-ffff-4fff-8fff-ffffffffffff"
			}
			c.SealSnapshot(&v)
			save(t, s, v)
			called := false
			sentinel := errors.New("intent failure")
			err := s.With(e, func(_, _, _ string) error {
				called = true
				if release, err := s.dir.lock(true); err == nil {
					release()
					t.Fatal("writer overtook admission")
				}
				if scenario == "callback" {
					return sentinel
				}
				return nil
			})
			if called != (scenario == "ok" || scenario == "callback") || (err == nil) != (scenario == "ok") {
				t.Fatalf("callback=%v err=%v", called, err)
			}
			release, err := s.dir.lock(true)
			if err != nil {
				t.Fatal("lock leaked")
			}
			release()
		})
	}
}
func TestProfileStrictWireAndProxyQuoting(t *testing.T) {
	e := example()
	if Validate(e) != nil {
		t.Fatal("invalid example")
	}
	raw, _ := json.Marshal(e)
	var decoded Endpoint
	if Decode(raw, &decoded) != nil {
		t.Fatal("roundtrip")
	}
	for _, b := range []string{strings.Replace(string(raw), `"application":`, `"Application":`, 1), strings.Replace(string(raw), `"protocol":`, `"extra":null,"protocol":`, 1), string(raw) + ` {}`} {
		if Decode([]byte(b), &decoded) == nil {
			t.Fatal("open shape")
		}
	}
	// No v1 reader may acquire profile behavior.
	in, _ := json.Marshal(Input{Protocol: InputProtocol, Endpoint: e, Invocation: json.RawMessage(`{}`)})
	if _, err := v1.DecodeInput(in); err == nil {
		t.Fatal("v1 widened")
	}
	command, err := ProxyCommand("/tmp/a '$(false)`false`; wrapper", e)
	if err != nil {
		t.Fatal(err)
	}
	command = "printf '%s\\n' " + command
	out, err := exec.Command("/bin/sh", "-c", command).Output()
	if err != nil || !strings.HasPrefix(string(out), "/tmp/a '$(false)`false`; wrapper\n") {
		t.Fatal("proxy token was not literal", err, string(out))
	}
	e.SessionDirectory = "/private/session%h"
	if _, err = ProxyCommand("/bin/proxy", e); err == nil {
		t.Fatal("OpenSSH expansion admitted")
	}
}
func TestPrivateSessionAndJournalNoReplay(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ordinary user")
	}
	e := example()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e.SessionDirectory = filepath.Join(dir, "session")
	e.JournalDirectory = filepath.Join(dir, "journal")
	for _, p := range []string{e.SessionDirectory, e.JournalDirectory} {
		if err = os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	ready := Readiness{"symphony.sov.proxy-readiness.v1", e.Application, e.Snapshot.Profile.Configuration.SessionRef, e.Snapshot.Profile.Configuration.Helper.ExecutableDigest, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	b, _ := json.Marshal(ready)
	if err = os.WriteFile(filepath.Join(e.SessionDirectory, "readiness.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if CheckSession(e) != nil {
		t.Fatal("private readiness refused")
	}
	id := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	j, err := Begin(e, id, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	j.Close()
	if j, err = Begin(e, id, strings.Repeat("a", 64)); err == nil {
		j.Close()
		t.Fatal("dispatch replay")
	}
	ready.ExpiresAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	b, _ = json.Marshal(ready)
	_ = os.WriteFile(filepath.Join(e.SessionDirectory, "readiness.json"), b, 0600)
	if CheckSession(e) == nil {
		t.Fatal("expired readiness")
	}
}

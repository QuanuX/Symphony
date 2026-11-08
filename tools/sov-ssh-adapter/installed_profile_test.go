//go:build linux

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/sov-ssh-adapter/auditclient"
	a "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/connectionaudit"
	v1 "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	p "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profile"
	c "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profilewire"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type receiptV2File struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   uint64 `json:"size"`
	Digest string `json:"digest"`
}

type receiptV2EntryPoint struct {
	EntryPointID string   `json:"entry_point_id"`
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	Protocols    []string `json:"protocols"`
}

type receiptV2Platform struct {
	OS           string  `json:"os"`
	Architecture string  `json:"architecture"`
	KernelABI    *string `json:"kernel_abi"`
	Critical     bool    `json:"critical"`
}

type receiptV2 struct {
	Protocol             string                `json:"protocol"`
	FormatVersion        uint64                `json:"format_version"`
	ComponentID          string                `json:"component_id"`
	ComponentKind        string                `json:"component_kind"`
	ModuleID             string                `json:"module_id"`
	VectorID             *string               `json:"vector_id"`
	EngineID             *string               `json:"engine_id"`
	PackageID            string                `json:"package_id"`
	Version              string                `json:"version"`
	InstallScope         string                `json:"install_scope"`
	PrefixMode           string                `json:"prefix_mode"`
	Files                []receiptV2File       `json:"files"`
	EntryPoints          []receiptV2EntryPoint `json:"entry_points"`
	ProvidesCapabilities []string              `json:"provides_capabilities"`
	RequiresCapabilities []string              `json:"requires_capabilities"`
	CompatibleReceptors  []string              `json:"compatible_receptors"`
	PlatformRequirements []receiptV2Platform   `json:"platform_requirements"`
	ReceiptDigest        string                `json:"receipt_digest"`
}

func fixtureID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func fixtureWrite(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if e := os.WriteFile(path, data, mode); e != nil {
		t.Fatal(e)
	}
}
func fixtureRead(t *testing.T, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func fixturePackage(t *testing.T, role string, source []byte) c.PackageRef {
	// Ambient client configuration must never influence selected transport.
	fixtureWrite(t, "/home/carrier/.ssh/config", []byte("Host *\n  ProxyCommand /bin/false\n  User wrong\n"), 0644)
	id := fixtureID()
	dir := filepath.Join(p.PackageRoot(), id, "0.1.0-dev")
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(dir)) })
	digest := func(b []byte) string { return "sha256:" + v1.Digest(b) }
	r := receiptV2{Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2, ComponentID: role, ComponentKind: "adapter", ModuleID: role, PackageID: role, Version: "0.1.0-dev", InstallScope: "prefix", PrefixMode: "installation_prefix", Files: []receiptV2File{{"binary", "executable", uint64(len(source)), digest(source)}}, EntryPoints: []receiptV2EntryPoint{{"main", "executable", "binary", []string{p.EndpointProtocol, p.InputProtocol, p.ReplyProtocol}}}, ProvidesCapabilities: []string{}, RequiresCapabilities: []string{}, CompatibleReceptors: []string{}, PlatformRequirements: []receiptV2Platform{{OS: "linux", Architecture: runtime.GOARCH, Critical: true}}}
	raw, _ := json.Marshal(r)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	delete(fields, "receipt_digest")
	raw, _ = c.Canonical(fields)
	r.ReceiptDigest = digest(raw)
	receipt, _ := c.Canonical(r)
	qualification := []byte("Isolated OpenSSH fixture; no Cloudflare qualification")
	m := c.PackageManifest{Protocol: c.PackageProtocol, PackageID: id, Version: "0.1.0-dev", Role: role, Platform: "linux-" + runtime.GOARCH, Executable: "binary", ReceiptPath: "receipt.json", ReceiptDigest: digest(receipt), QualificationPath: "qualification.txt", QualificationRef: digest(qualification), ReceiptComponentID: role, ReceiptVersion: "0.1.0-dev", ReceiptEntryPointID: "main", Files: []c.PackageFile{}}
	for _, f := range []struct {
		name string
		data []byte
		mode os.FileMode
	}{{"binary", source, 0755}, {"receipt.json", receipt, 0644}, {"qualification.txt", qualification, 0644}} {
		fixtureWrite(t, filepath.Join(dir, f.name), f.data, f.mode)
		m.Files = append(m.Files, c.PackageFile{Path: f.name, Digest: digest(f.data), Size: strconv.Itoa(len(f.data)), Executable: f.mode == 0755})
	}
	m.Digest = c.Digest(c.PackageProtocol, m)
	raw, _ = c.Canonical(m)
	fixtureWrite(t, filepath.Join(dir, "manifest.json"), raw, 0644)
	return c.PackageRef{PackageID: id, Version: m.Version, ManifestDigest: m.Digest, ReceiptDigest: m.ReceiptDigest, ExecutableDigest: digest(source)}
}
func TestInstalledProfileRealOpenSSH(t *testing.T) {
	if os.Getenv("SYMPHONY_I3_ROOT_FIXTURE") != "1" || os.Geteuid() != 0 {
		t.Skip("explicit disposable Linux host with sshd and fixture user required")
	}
	uid, gid := uint32(1001), uint32(1001)
	binary := func(ref c.PackageRef) string {
		return filepath.Join(p.PackageRoot(), ref.PackageID, ref.Version, "binary")
	}
	carrier := fixturePackage(t, "carrier", fixtureRead(t, "/fixture/carrier"))
	ssh := fixturePackage(t, "ssh", fixtureRead(t, "/usr/bin/ssh"))
	proxy := fixturePackage(t, "proxy", []byte("#!/bin/sh\nset -eu\n[ \"$(id -u)\" = 1001 ]\n[ \"$#\" = 6 ]\n[ \"$1\" = --application ]\n[ \"$2\" = access.example.invalid ]\n[ \"$3\" = --session-directory ]\n[ \"$5\" = --session-ref ]\nprintf 'proxy-started\\n' >> /home/carrier/proxy-observation\nif [ -f /home/carrier/proxy-delay ]; then echo $$ > /home/carrier/proxy-pid; exec /bin/sleep 60; fi\nexec /usr/bin/nc 127.0.0.1 22222\n"))
	key := strings.Join(strings.Fields(string(fixtureRead(t, "/home/carrier/key.pub")))[:2], " ")
	hostkey := strings.Join(strings.Fields(string(fixtureRead(t, "/fixture/host.pub")))[:2], " ")
	direct := v1.Endpoint{Protocol: v1.EndpointProtocol, SSH: v1.Executable{binary(ssh), strings.TrimPrefix(ssh.ExecutableDigest, "sha256:")}, SSHExecution: "protected_system_path", Host: "ssh.example.invalid", Port: "22", User: "carrier", HostPublicKey: hostkey, ClientPublicKey: key, AgentSocket: "/home/carrier/agent.sock", TargetLauncher: v1.Executable{"/fixture/target", v1.Digest(fixtureRead(t, "/fixture/target"))}}
	e := p.Example(direct)
	e.Scope = c.Scope{HostID: fixtureID(), TOPSID: fixtureID()}
	e.Snapshot.Profile.Scope = e.Scope
	e.Snapshot.Policy.Scope = e.Scope
	e.Snapshot.Profile.ProfileID = fixtureID()
	e.Snapshot.Policy.ProfileID = e.Snapshot.Profile.ProfileID
	e.Snapshot.Profile.Configuration.Adapter = carrier
	e.Snapshot.Profile.Configuration.SSH = ssh
	e.Snapshot.Profile.Configuration.Helper = proxy
	e.Snapshot.Policy.Permitted = []c.Principal{{UID: "1001", GID: "1001"}}
	c.SealSnapshot(&e.Snapshot)
	e.SessionDirectory = "/home/carrier/private session 'quoted'"
	e.JournalDirectory = "/home/carrier/journal"
	for _, dir := range []string{e.SessionDirectory, e.JournalDirectory} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chown(dir, int(uid), int(gid)); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })
	}
	readiness, _ := json.Marshal(p.Readiness{Protocol: "symphony.sov.proxy-readiness.v1", Application: e.Application, SessionRef: e.Snapshot.Profile.Configuration.SessionRef, HelperDigest: proxy.ExecutableDigest, ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)})
	fixtureWrite(t, filepath.Join(e.SessionDirectory, "readiness.json"), readiness, 0600)
	_ = os.Chown(filepath.Join(e.SessionDirectory, "readiness.json"), int(uid), int(gid))
	store := filepath.Join(p.Root(), e.Scope.HostID, e.Scope.TOPSID)
	if err := os.MkdirAll(filepath.Join(store, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(p.Root(), e.Scope.HostID)) })
	scope, _ := json.Marshal(map[string]any{"protocol": "symphony.sov.transport-profile-scope.v1", "scope": e.Scope})
	fixtureWrite(t, filepath.Join(store, "scope.json"), scope, 0644)
	fixtureWrite(t, filepath.Join(store, "LOCK"), nil, 0644)
	publish := func(v c.Snapshot) {
		raw, _ := c.Canonical(v)
		fixtureWrite(t, filepath.Join(store, "profiles", v.Profile.ProfileID+".json"), raw, 0644)
	}
	publish(e.Snapshot)
	socket := auditclient.Socket(e.Scope.TOPSID)
	if err := os.MkdirAll(filepath.Dir(socket), 0755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(filepath.Dir(socket))) })
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_ = os.Chown(socket, 0, int(gid))
	_ = os.Chmod(socket, 0660)
	var refuseAudit atomic.Bool
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refuseAudit.Load() {
			http.Error(w, "unavailable", 503)
			return
		}
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 65537))
		var req a.Request
		if a.Decode(raw, &req) != nil {
			http.Error(w, "invalid", 400)
			return
		}
		candidate, err := a.Candidate(req, uid)
		if err != nil {
			http.Error(w, "invalid", 400)
			return
		}
		d, _ := stav.CandidateDigest(candidate)
		receipt := stav.Receipt{Schema: stav.SchemaReceipt, TOPSID: req.TOPSID, RequestID: req.RequestID, CandidateDigest: d, Disposition: "committed", ReasonCode: stav.ReasonReceiptCommitted, Commit: stav.CommitResult{EventID: fixtureID(), EventDigest: "sha256:" + strings.Repeat("c", 64), Sequence: 1, State: "committed", Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")}}
		_ = json.NewEncoder(w).Encode(a.Response{Protocol: a.Protocol, RequestDigest: a.Hash(req), Receipt: receipt})
	})}
	go server.Serve(listener)
	defer server.Close()
	deadline := 10 * time.Second
	invocation := func(endpoint p.Endpoint, id string) []byte {
		raw, _ := json.Marshal(map[string]any{"protocol": "symphony.sov.remote-invoke-input.v1", "request_id": id, "adapter": v1.Executable{binary(carrier), strings.TrimPrefix(carrier.ExecutableDigest, "sha256:")}, "selection": map[string]any{"transport": map[string]any{"arrangement": "controller", "family": "scoped_cli", "adapter_id": v1.AdapterID, "adapter_version": p.AdapterVersion, "endpoint_ref": p.Ref(endpoint)}, "runtime": map[string]any{"qxctl_sha256": strings.Repeat("a", 64)}, "resources": map[string]any{"deadline_unix_ms": strconv.FormatInt(time.Now().Add(deadline).UnixMilli(), 10)}}, "endpoint": map[string]any{"target_qxctl": v1.Executable{"/fixture/target", strings.Repeat("a", 64)}, "configuration_path": "/fixture/config", "configuration_sha256": strings.Repeat("a", 64)}})
		return raw
	}
	call := func(endpoint p.Endpoint, id string, asRoot bool) p.Reply {
		t.Helper()
		raw, _ := json.Marshal(p.Input{Protocol: p.InputProtocol, Endpoint: endpoint, Invocation: invocation(endpoint, id)})
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary(carrier))
		cmd.Stdin = bytes.NewReader(raw)
		cmd.Env = []string{"LANG=C", "SUDO_UID=1001"}
		if !asRoot {
			cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid}}
		}
		out, err := cmd.Output()
		if asRoot {
			if err == nil {
				t.Fatal("root connected")
			}
			return p.Reply{}
		}
		if err != nil {
			t.Fatalf("carrier exit: %v, output %s", err, out)
		}
		var result p.Reply
		if err = p.Decode(out, &result); err != nil {
			t.Fatal(err, string(out))
		}
		return result
	}
	id := fixtureID()
	result := call(e, id, false)
	if result.TransportOutcome != "response_received" || !bytes.Contains(result.TargetResult, []byte(`"fixture":true`)) {
		known, identity := "/fixture/known", "/fixture/identity.pub"
		fixtureWrite(t, known, []byte("sov-selected-target "+hostkey+"\n"), 0644)
		fixtureWrite(t, identity, []byte(key+"\n"), 0644)
		proxyCommand, _ := p.ProxyCommand(binary(proxy), e)
		args := sshOptions(direct, known, identity)
		for i, arg := range args {
			if arg == "ProxyCommand=none" {
				args[i] = "ProxyCommand=" + proxyCommand
			}
			if arg == "LogLevel=ERROR" {
				args[i] = "LogLevel=DEBUG3"
			}
		}
		debug := exec.Command(binary(ssh), append(args, "/fixture/target")...)
		debug.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid}}
		debug.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "HOME=" + e.SessionDirectory}
		diagnostic, diagnosticErr := debug.CombinedOutput()
		t.Logf("fixture SSH diagnostic: %v %s", diagnosticErr, diagnostic)
		t.Fatalf("OpenSSH exchange failed: %+v", result)
	}
	if result = call(e, id, false); result.TransportOutcome != "not_dispatched" {
		t.Fatal("same request replayed")
	}
	call(e, fixtureID(), true)
	disabled := e.Snapshot
	disabled.Profile.State = "disabled"
	disabled.Profile.Generation = "2"
	disabled.Policy.ProfileGeneration = "2"
	c.SealSnapshot(&disabled)
	publish(disabled)
	if result = call(e, fixtureID(), false); result.TransportOutcome != "not_dispatched" {
		t.Fatal("disabled profile dispatched")
	}
	publish(e.Snapshot)
	observation := func() string { b, _ := os.ReadFile("/home/carrier/proxy-observation"); return string(b) }
	before := observation()
	refuseAudit.Store(true)
	if result = call(e, fixtureID(), false); result.TransportOutcome != "not_dispatched" {
		t.Fatal("missing audit dispatched", result)
	}
	refuseAudit.Store(false)
	readinessPath := filepath.Join(e.SessionDirectory, "readiness.json")
	if err = os.Rename(readinessPath, readinessPath+".saved"); err != nil {
		t.Fatal(err)
	}
	if result = call(e, fixtureID(), false); result.TransportOutcome != "not_dispatched" || result.ReasonCode != "transport_profile.interaction_required" {
		t.Fatal("missing session dispatched", result)
	}
	if err = os.Rename(readinessPath+".saved", readinessPath); err != nil {
		t.Fatal(err)
	}
	fixtureWrite(t, filepath.Join(store, "pending.json"), []byte("corrupt intent"), 0644)
	if result = call(e, fixtureID(), false); result.TransportOutcome != "not_dispatched" {
		t.Fatal("unsettled store dispatched")
	}
	if err = os.Remove(filepath.Join(store, "pending.json")); err != nil {
		t.Fatal(err)
	}
	changed := e.Snapshot
	changed.Profile.Generation = "3"
	changed.Policy.ProfileGeneration = "3"
	c.SealSnapshot(&changed)
	publish(changed)
	if result = call(e, fixtureID(), false); result.TransportOutcome != "not_dispatched" {
		t.Fatal("old generation admitted after re-enable")
	}
	publish(e.Snapshot)
	if observation() != before {
		t.Fatal("refused request started proxy")
	}
	// A wrong selected public identity cannot use another key loaded in the agent.
	wrongClient := e
	wrongClient.Direct.ClientPublicKey = hostkey
	if result = call(wrongClient, fixtureID(), false); result.TransportOutcome != "response_lost" {
		t.Fatal("wrong client identity accepted", result)
	}
	// Local deadline must clean the SSH/proxy process group and retain uncertainty.
	fixtureWrite(t, "/home/carrier/proxy-delay", nil, 0644)
	deadline = 1500 * time.Millisecond
	started := time.Now()
	cancelID := fixtureID()
	if result = call(e, cancelID, false); result.TransportOutcome != "response_lost" {
		t.Fatal("cancellation claimed certainty", result)
	}
	if time.Since(started) > 5*time.Second {
		t.Fatal("cleanup exceeded bound")
	}
	pidText := strings.TrimSpace(string(fixtureRead(t, "/home/carrier/proxy-pid")))
	pid, err := strconv.Atoi(pidText)
	if err != nil {
		t.Fatal(err)
	}
	stat, _ := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if syscall.Kill(pid, 0) == nil && !strings.Contains(string(stat), ") Z ") {
		t.Fatal("proxy survived cancellation", pid)
	}
	record := string(fixtureRead(t, filepath.Join(e.JournalDirectory, cancelID+".jsonl")))
	for _, stage := range []string{"prepared", "audit_committed", "dispatch_intent", "started", "response_lost"} {
		if !strings.Contains(record, `"stage":"`+stage+`"`) {
			t.Fatal("missing journal stage", stage)
		}
	}
	if strings.Contains(record, "fixture\":true") {
		t.Fatal("target payload retained")
	}
	if err = os.Remove("/home/carrier/proxy-delay"); err != nil {
		t.Fatal(err)
	}
	deadline = 10 * time.Second
	wrong := e
	wrong.Direct.HostPublicKey = key
	wrong.Snapshot.Profile.Configuration.Endpoints[0].HostKeyDigest = "sha256:" + mustKeyDigest(t, key)
	c.SealSnapshot(&wrong.Snapshot)
	publish(wrong.Snapshot)
	if result = call(wrong, fixtureID(), false); result.TransportOutcome != "response_lost" {
		t.Fatal("wrong host key accepted", result)
	}
	t.Log("actual OpenSSH through controlled proxy as UID/GID 1001; root, replay, disabled/stale/pending, audit/session refusal, key isolation and bounded cancellation verified")
}
func mustKeyDigest(t *testing.T, key string) string {
	d, e := v1.PublicKeyDigest(key)
	if e != nil {
		t.Fatal(e)
	}
	return d
}

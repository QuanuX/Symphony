//go:build darwin || linux

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	c "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/contract"
	"github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/auditclient"
	"github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/store"
	a "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/transportaudit"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// This opt-in disposable-host test installs no policy. CI installs the exact
// built helper first; no production override of paths, owner or audit exists.
func TestProtectedProfileInstalledProcess(t *testing.T) {
	if os.Getenv("SYMPHONY_PROFILE_ROOT_FIXTURE") != "1" || os.Geteuid() != 0 {
		t.Skip("requires explicit disposable-root installed-helper fixture")
	}
	id := func() string {
		var b [16]byte
		if _, e := rand.Read(b[:]); e != nil {
			t.Fatal(e)
		}
		b[6] = b[6]&15 | 64
		b[8] = b[8]&63 | 128
		return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
	}
	scope := c.Scope{HostID: id(), TOPSID: id()}
	hostRoot := filepath.Join(store.Root(), scope.HostID)
	if _, e := os.Lstat(hostRoot); !os.IsNotExist(e) {
		t.Fatal("fixture destination exists")
	}
	dir := filepath.Join(hostRoot, scope.TOPSID)
	for _, child := range []string{"profiles", "history", "attempts"} {
		if e := os.MkdirAll(filepath.Join(dir, child), 0755); e != nil {
			t.Fatal(e)
		}
	}
	defer os.RemoveAll(hostRoot)
	raw, _ := json.Marshal(map[string]any{"protocol": "symphony.sov.transport-profile-scope.v1", "scope": scope})
	if e := os.WriteFile(filepath.Join(dir, "scope.json"), raw, 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "LOCK"), nil, 0644); e != nil {
		t.Fatal(e)
	}
	socket := auditclient.Socket(scope.TOPSID)
	socketRoot := filepath.Dir(filepath.Dir(socket))
	if _, e := os.Lstat(socketRoot); !os.IsNotExist(e) {
		t.Fatal("fixture audit destination exists")
	}
	if e := os.MkdirAll(filepath.Dir(socket), 0755); e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(socketRoot)
	listener, e := net.Listen("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	if e = os.Chmod(socket, 0600); e != nil {
		t.Fatal(e)
	}
	// Independent fake append sink: live kernel-root Unix transport with exact
	// STAV candidate receipts. This is not a deployed SSIAG/STAV qualification.
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, e := io.ReadAll(io.LimitReader(r.Body, 65537))
		var input a.Request
		if r.URL.Path != a.Endpoint || e != nil || a.Decode(raw, &input) != nil || input.TOPSID != scope.TOPSID {
			http.Error(w, "invalid", 400)
			return
		}
		candidate, e := a.Candidate(input, 0)
		if e != nil {
			http.Error(w, "invalid", 400)
			return
		}
		digest, e := stav.CandidateDigest(candidate)
		if e != nil {
			http.Error(w, "invalid", 400)
			return
		}
		receipt := stav.Receipt{Schema: stav.SchemaReceipt, TOPSID: scope.TOPSID, RequestID: input.RequestID, CandidateDigest: digest, Disposition: "committed", ReasonCode: stav.ReasonReceiptCommitted, Commit: stav.CommitResult{EventID: "55555555-5555-4555-8555-555555555555", EventDigest: "sha256:" + strings.Repeat("c", 64), Sequence: 1, State: "committed", Timestamp: "2026-11-07T12:00:00.000000000Z"}}
		_ = json.NewEncoder(w).Encode(a.Response{Protocol: a.Protocol, RequestDigest: a.Hash(input), Receipt: receipt})
	})}
	go server.Serve(listener)
	defer server.Close()
	call := func(input c.Request, privileged bool) c.Result {
		t.Helper()
		raw, _ := c.Canonical(input)
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, store.HelperPath(), "query")
		if privileged {
			command = exec.CommandContext(ctx, "/usr/bin/sudo", "-n", "--", store.HelperPath(), "administer")
		} else {
			command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
		}
		command.Dir = "/"
		command.Env = []string{"LANG=C", "LC_ALL=C"}
		command.Stdin = bytes.NewReader(raw)
		var diagnostic bytes.Buffer
		command.Stderr = &diagnostic
		result, e := command.Output()
		var decoded c.Result
		if e != nil || c.Decode(bytes.TrimSpace(result), &decoded) != nil || c.ValidateResult(decoded, input) != nil {
			t.Fatalf("installed process: %v, %s, %s", e, diagnostic.String(), result)
		}
		return decoded
	}
	input := c.Template()
	input.Scope = scope
	planned := call(input, false)
	if !planned.Completed || planned.Plan == nil {
		t.Fatal("ordinary-user plan unavailable")
	}
	applied := call(c.Request{Protocol: c.RequestProtocol, RequestID: id(), Operation: "apply", Scope: scope, Plan: planned.Plan}, true)
	if !applied.Completed || applied.Attempt == nil || applied.Attempt.Outcome != "published" {
		t.Fatal("root publication incomplete", applied)
	}
	inspected := call(c.Request{Protocol: c.RequestProtocol, RequestID: id(), Operation: "inspect", Scope: scope, ProfileID: input.ProfileID}, false)
	if inspected.Snapshot == nil || inspected.Snapshot.Profile.State != "disabled" {
		t.Fatal("unprivileged coherent inspection unavailable")
	}
	// Exact production helper rejects a forged sudo environment without UID 0.
	raw, _ = c.Canonical(c.Request{Protocol: c.RequestProtocol, RequestID: id(), Operation: "apply", Scope: scope, Plan: planned.Plan})
	cmd := exec.Command(store.HelperPath(), "administer")
	cmd.Dir = "/"
	cmd.Env = []string{"SUDO_UID=0", "SUDO_USER=root"}
	cmd.Stdin = bytes.NewReader(raw)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
	if e = cmd.Run(); e == nil {
		t.Fatal("unprivileged mutation accepted")
	}
}

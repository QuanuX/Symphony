package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	use "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/cloudflareuse"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func cloudflareFixture(t *testing.T, grant, invalidReceipt bool) (*Server, string, *foundationSink, use.Request) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("requires ordinary-user kernel credentials")
	}
	socket := shortSocketPath(t)
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	auth := serviceAuthentication(uid, gid)
	auth.Subjects = []config.SubjectConfig{{ID: "cloudflare-user", Kind: "symphony.identity.operator", UID: &uid, GID: &gid}}
	in := use.Request{Protocol: use.Protocol, RequestID: "11111111-1111-4111-8111-111111111111", TOPSID: testTOPSID, PlanDigest: "sha256:" + strings.Repeat("a", 64), AdapterDigest: "sha256:" + strings.Repeat("b", 64), ToolchainDigest: "sha256:" + strings.Repeat("c", 64), CredentialRef: "ssiag-native:api", Operation: "tunnels.list", AccountID: strings.Repeat("0", 32), DeadlineUnixMS: fmt.Sprint(time.Now().Add(50 * time.Second).UnixMilli())}
	if err := in.Validate(time.Now()); err != nil {
		t.Fatal("fixture validation", err)
	}
	encoded, _ := json.Marshal(in)
	var decoded use.Request
	if err := use.Decode(encoded, &decoded); err != nil {
		t.Fatal("fixture decoding", err)
	}
	authorization := &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 60, Grants: []config.AuthorizationGrant{}}
	if grant {
		authorization.Grants = append(authorization.Grants, config.AuthorizationGrant{ID: "cloudflare-list", SubjectID: "cloudflare-user", AuthorityBasis: "granted_permission", Operation: "symphony.sov.cloudflare.tunnels.list", Resource: in.Resource(), Audience: "sov-cloudflare", Scope: "tops:" + testTOPSID})
	}
	cfg := config.Config{Schema: "symphony.ssiag.config.v1", Mode: "user", TOPS: config.TOPSConfig{ID: testTOPSID, Name: "Test"}, Listen: config.ListenConfig{Network: "unix", Address: socket}, Authentication: auth, Authorization: authorization, Providers: []config.ProviderConfig{}}
	registry, _ := provider.New(nil)
	sink := &foundationSink{invalid: invalidReceipt, records: make(chan stavproducer.Record, 16)}
	s, e := NewWithAudit(cfg, registry, sink)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	waitForSocket(t, socket)
	t.Cleanup(func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	})
	return s, socket, sink, in
}
func cloudflareOpen(t *testing.T, path string, in any) (net.Conn, *bufio.ReadWriter, string) {
	t.Helper()
	c, e := net.DialTimeout("unix", path, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
	b, _ := json.Marshal(in)
	fmt.Fprintf(rw, "POST %s HTTP/1.1\r\nHost: localhost\r\nContent-Length: %d\r\n\r\n%s", use.Endpoint, len(b), b)
	_ = rw.Flush()
	first, e := rw.ReadString('\n')
	if e != nil {
		t.Fatal(e)
	}
	for {
		line, e := rw.ReadString('\n')
		if e != nil {
			t.Fatal(e)
		}
		if line == "\r\n" {
			break
		}
	}
	return c, rw, first
}
func TestCloudflareUseRequiresExactGrantAndCommittedAudit(t *testing.T) {
	for _, test := range []struct {
		name          string
		grant, broken bool
		status        string
	}{{"allowed", true, false, "101"}, {"denied", false, false, "403"}, {"wrong-receipt", true, true, "503"}} {
		t.Run(test.name, func(t *testing.T) {
			_, path, sink, in := cloudflareFixture(t, test.grant, test.broken)
			c, rw, status := cloudflareOpen(t, path, in)
			defer c.Close()
			if !strings.Contains(status, test.status) {
				t.Fatal(status)
			}
			if test.status != "101" {
				return
			}
			b, e := readUseFrame(rw)
			if e != nil {
				t.Fatal(e)
			}
			var admission map[string]any
			if json.Unmarshal(b, &admission) != nil || admission["request_digest"] != use.Hash(in) || admission["uid"] != fmt.Sprint(os.Geteuid()) {
				t.Fatal("wrong admission")
			}
			if e = writeUseFrame(rw, use.Completion{Protocol: use.Protocol, RequestDigest: use.Hash(in), Outcome: "completed"}); e != nil {
				t.Fatal(e)
			}
			b, e = readUseFrame(rw)
			if e != nil {
				t.Fatal(e)
			}
			var completion map[string]any
			_ = json.Unmarshal(b, &completion)
			if completion["outcome"] != "completed" {
				t.Fatal("outcome lost")
			}
			intent, outcome := <-sink.records, <-sink.records
			if intent.Kind != stavproducer.CloudflareUseIntent || outcome.Kind != stavproducer.CloudflareUseOutcome || intent.RequestID == outcome.RequestID || outcome.CorrelationID != intent.RequestID {
				t.Fatal("audit identity collision")
			}
		})
	}
}
func TestCloudflareUseRejectsForgedAndExpiredControls(t *testing.T) {
	s, path, _, in := cloudflareFixture(t, true, false)
	raw, _ := json.Marshal(in)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("POST", use.Endpoint, bytes.NewReader(raw)))
	if w.Code != 401 {
		t.Fatal("kernel identity bypass", w.Code)
	}
	for _, bad := range []map[string]any{{"subject": "forged"}, {"deadline_unix_ms": "1"}, {"credential_ref": "SECRET-SENTINEL"}, {"account_id": strings.Repeat("f", 32)}} {
		var changed map[string]any
		_ = json.Unmarshal(raw, &changed)
		for k, v := range bad {
			changed[k] = v
		}
		c, _, status := cloudflareOpen(t, path, changed)
		c.Close()
		if strings.Contains(status, "101") {
			t.Fatal("invalid admission", bad)
		}
	}
}
func TestCloudflareUsePinsPolicyAndAuditsLostCompletion(t *testing.T) {
	s, path, sink, in := cloudflareFixture(t, true, false)
	c, rw, status := cloudflareOpen(t, path, in)
	if !strings.Contains(status, "101") {
		t.Fatal(status)
	}
	if _, e := readUseFrame(rw); e != nil {
		t.Fatal(e)
	}
	if s.policy == nil {
		t.Fatal("missing owner")
	} // real pin is checked below through writer ordering
	finished := make(chan struct{})
	go func() {
		replacement := &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 60, Grants: []config.AuthorizationGrant{}}
		b, _ := json.Marshal(replacement)
		_ = s.policy.Replace(replacement, fmt.Sprintf("sha256:%x", sha256.Sum256(b)))
		close(finished)
	}()
	select {
	case <-finished:
		t.Fatal("policy snapshot released before cleanup")
	case <-time.After(30 * time.Millisecond):
	}
	c.Close()
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("lost connection kept policy pinned")
	}
	<-sink.records
	outcome := <-sink.records
	if outcome.Outcome != "unavailable" {
		t.Fatal("lost completion claimed success")
	}
}

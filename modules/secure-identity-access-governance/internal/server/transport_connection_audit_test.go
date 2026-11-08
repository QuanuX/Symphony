package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	a "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/connectionaudit"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestTransportConnectionAuditKernelAndReceipt(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("nonroot kernel peer fixture")
	}
	socket := shortSocketPath(t)
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	cfg := config.Config{Schema: "symphony.ssiag.config.v1", Mode: "user", TOPS: config.TOPSConfig{ID: testTOPSID, Name: "Test"}, Listen: config.ListenConfig{Network: "unix", Address: socket}, Authentication: serviceAuthentication(uid, gid), Providers: []config.ProviderConfig{}}
	registry, _ := provider.New(nil)
	sink := &foundationSink{records: make(chan stavproducer.Record, 8)}
	s, e := NewWithAudit(cfg, registry, sink)
	if e != nil {
		t.Fatal(e)
	}
	// Exercise system handler semantics over a real ordinary-user kernel peer.
	s.config.Mode = "system"
	input := a.Request{Protocol: a.Protocol, RequestID: "11111111-1111-4111-8111-111111111111", TOPSID: testTOPSID, ProfileID: "22222222-2222-4222-8222-222222222222", EndpointDigest: "sha256:" + strings.Repeat("a", 64), InvocationDigest: "sha256:" + strings.Repeat("b", 64)}
	raw, _ := json.Marshal(input)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", a.Endpoint, bytes.NewReader(raw))
	req.Header.Set("X-UID", "501")
	s.Handler().ServeHTTP(recorder, req)
	if recorder.Code != 401 {
		t.Fatal("unauthenticated peer admitted", recorder.Code)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	defer func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	}()
	waitForSocket(t, socket)
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}
	response, e := client.Post("http://localhost"+a.Endpoint, "application/json", bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	var result a.Response
	if e = json.NewDecoder(response.Body).Decode(&result); e != nil || response.StatusCode != 200 || result.ValidateFor(input, uid) != nil {
		t.Fatal("committed receipt mismatch", e, response.StatusCode, result)
	}
	record := <-sink.records
	if record.Kind != stavproducer.TransportConnectionIntent {
		t.Fatal("wrong event kind")
	}
	if result.ValidateFor(input, 0) == nil || result.ValidateFor(input, uid+1) == nil {
		t.Fatal("receipt transferred across principals")
	}
}

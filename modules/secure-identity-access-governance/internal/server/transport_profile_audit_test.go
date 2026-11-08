package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	a "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/transportaudit"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func profileAuditInput() a.Request {
	return a.Request{Protocol: a.Protocol, RequestID: "11111111-1111-4111-8111-111111111111", OperationID: "22222222-2222-4222-8222-222222222222", HostID: "33333333-3333-4333-8333-333333333333", TOPSID: testTOPSID, ProfileID: "44444444-4444-4444-8444-444444444444", Action: "create", PlanDigest: "sha256:" + strings.Repeat("a", 64), PriorDigest: "sha256:" + strings.Repeat("b", 64), DesiredDigest: "sha256:" + strings.Repeat("c", 64), Phase: "intent", Resolution: "pending"}
}
func TestTransportProfileAuditExactProducerAndReceipt(t *testing.T) {
	for _, phase := range []string{"intent", "published", "aborted"} {
		t.Run(phase, func(t *testing.T) {
			input := profileAuditInput()
			expected := stavproducer.TransportProfileIntent
			if phase != "intent" {
				input.Phase = "outcome"
				input.Resolution = phase
				input.ObservedDigest = input.DesiredDigest
				expected = stavproducer.TransportProfilePublished
				if phase == "aborted" {
					input.ObservedDigest = input.PriorDigest
					expected = stavproducer.TransportProfileAborted
				}
			}
			sink := &foundationSink{records: make(chan stavproducer.Record, 8)}
			s := &Server{config: config.Config{Mode: "system", TOPS: config.TOPSConfig{ID: testTOPSID}}, audit: sink}
			peer := peerauth.Peer{Credentials: peerauth.Credentials{UID: 0, GID: 0}}
			result, e := s.commitTransportProfileAudit(context.Background(), peer, input)
			if e != nil || result.Validate(input) != nil {
				t.Fatal("producer and independent consumer disagree", e)
			}
			record := <-sink.records
			if record.Kind != expected || record.Actor.ID != "symphony.host.owner.uid.0" {
				t.Fatal("incorrect root producer", record)
			}
			sink.invalid = true
			if _, e = s.commitTransportProfileAudit(context.Background(), peer, input); e == nil {
				t.Fatal("unrelated committed receipt accepted")
			}
			sink.invalid = false
			peer.Credentials.UID = 501
			if _, e = s.commitTransportProfileAudit(context.Background(), peer, input); e == nil {
				t.Fatal("nonroot authorized")
			}
			peer.Credentials.UID = 0
			s.config.Mode = "user"
			if _, e = s.commitTransportProfileAudit(context.Background(), peer, input); e == nil {
				t.Fatal("user scope accepted")
			}
			s.config.Mode = "system"
			input.TOPSID = "55555555-5555-4555-8555-555555555555"
			if _, e = s.commitTransportProfileAudit(context.Background(), peer, input); e == nil {
				t.Fatal("other TOPS accepted")
			}
		})
	}
}
func TestTransportProfileAuditKernelBoundary(t *testing.T) {
	socket := shortSocketPath(t)
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	cfg := config.Config{Schema: "symphony.ssiag.config.v1", Mode: "user", TOPS: config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"}, Listen: config.ListenConfig{Network: "unix", Address: socket}, Authentication: serviceAuthentication(uid, gid), Providers: []config.ProviderConfig{}}
	registry, _ := provider.New(nil)
	s, e := NewWithAudit(cfg, registry, &recordingAudit{})
	if e != nil {
		t.Fatal(e)
	}
	input := profileAuditInput()
	raw, _ := json.Marshal(input)
	direct := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, a.Endpoint, bytes.NewReader(raw))
	req.Header.Set("X-UID", "0")
	s.Handler().ServeHTTP(direct, req)
	if direct.Code != 401 {
		t.Fatal("missing kernel credentials accepted", direct.Code)
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
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	response, e := client.Post("http://unix"+a.Endpoint, "application/json", bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("real user socket elevated profile audit", response.StatusCode)
	}
}

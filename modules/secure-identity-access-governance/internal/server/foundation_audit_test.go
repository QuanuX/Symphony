package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationaudit"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

type foundationSink struct {
	invalid bool
	records chan stavproducer.Record
}

func (s *foundationSink) Submit(ctx context.Context, record stavproducer.Record) (stav.Receipt, error) {
	s.records <- record
	receipt, err := (&recordingAudit{}).Submit(ctx, record)
	if s.invalid {
		receipt.RequestID = "123e4567-e89b-42d3-a456-426614174009"
	}
	return receipt, err
}

func TestFoundationAuditRequiresKernelOwnerAndExactCommittedReceipt(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "committed", true: "wrong-receipt"}[invalid], func(t *testing.T) {
			socket := shortSocketPath(t)
			uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
			cfg := config.Config{Schema: "symphony.ssiag.config.v1", Mode: "user", TOPS: config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"}, Listen: config.ListenConfig{Network: "unix", Address: socket}, Authentication: serviceAuthentication(uid, gid), Providers: []config.ProviderConfig{}}
			registry, _ := provider.New(nil)
			sink := &foundationSink{invalid: invalid, records: make(chan stavproducer.Record, 4)}
			instance, err := NewWithAudit(cfg, registry, sink)
			if err != nil {
				t.Fatal(err)
			}
			request := foundationaudit.Request{Schema: foundationaudit.Schema, Component: "stav", Surface: "supervisor", Scope: "user", TOPSID: testTOPSID, OperationID: "test:stop", RequestID: "123e4567-e89b-42d3-a456-426614174001", CorrelationID: "123e4567-e89b-42d3-a456-426614174002", PlanDigest: "sha256:" + strings.Repeat("a", 64), PriorStateDigest: "sha256:" + strings.Repeat("b", 64), DesiredState: "native_installed_stopped", AuditMode: "ordinary", Phase: "intent"}
			payload, _ := json.Marshal(request)
			direct := httptest.NewRecorder()
			instance.Handler().ServeHTTP(direct, httptest.NewRequest(http.MethodPost, foundationaudit.Endpoint, bytes.NewReader(payload)))
			if direct.Code == 200 {
				t.Fatal("unverified caller reached audit bridge")
			}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- instance.Run(ctx) }()
			defer func() {
				cancel()
				if err := <-done; err != nil {
					t.Error(err)
				}
			}()
			waitForSocket(t, socket)
			transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
			response, err := client.Post("http://unix"+foundationaudit.Endpoint, "application/json", bytes.NewReader(payload))
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if invalid {
				if response.StatusCode != 503 {
					t.Fatalf("invalid receipt accepted: %s", data)
				}
				return
			}
			var result foundationaudit.Response
			if response.StatusCode != 200 || foundationaudit.Decode(data, &result) != nil || result.Validate(request, uid) != nil {
				t.Fatalf("exact receipt missing: %d %s", response.StatusCode, data)
			}
			record := <-sink.records
			if record.Actor.ID != resultActor(uid) || record.Kind != stavproducer.FoundationIntent || record.Outcome != "allowed" {
				t.Fatalf("wrong intent projection: %+v", record)
			}
			// Unknown actor/candidate fields cannot turn this endpoint into append proxy.
			bad := append(payload[:len(payload)-1], []byte(`,"actor":"forged"}`)...)
			response, err = client.Post("http://unix"+foundationaudit.Endpoint, "application/json", bytes.NewReader(bad))
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != 400 {
				t.Fatalf("caller-supplied actor accepted: %d", response.StatusCode)
			}
		})
	}
}
func resultActor(uid uint32) string { return fmt.Sprintf("symphony.host.owner.uid.%d", uid) }

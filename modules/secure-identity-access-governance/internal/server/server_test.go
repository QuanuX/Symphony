package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

type recordingAudit struct {
	record stavproducer.Record
}

func (audit *recordingAudit) Submit(_ context.Context, record stavproducer.Record) (stavprotocol.Receipt, error) {
	audit.record = record
	return stavprotocol.Receipt{}, nil
}

func TestAuthorizationOverKernelAuthenticatedSocketIsAudited(t *testing.T) {
	socket := shortSocketPath(t)
	probe, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	authentication := serviceAuthentication(uid, gid)
	authentication.Subjects = []config.SubjectConfig{{ID: "owner.primary", Kind: "owner", UID: &uid, GID: &gid}}
	cfg := config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: authentication,
		Authorization: &config.AuthorizationConfig{
			DefaultEffect: "deny", MaxCapabilitySeconds: 900,
			Grants: []config.AuthorizationGrant{{
				ID: "session-begin", SubjectID: "owner.primary", AuthorityBasis: "host_owner",
				Operation: "symphony.knowledge.session.begin", Resource: "symphony.knowledge.repository:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Audience: "qxctl", Scope: "tops:" + testTOPSID,
			}},
		},
		Providers: []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	audit := &recordingAudit{}
	instance, err := NewWithAudit(cfg, registry, audit)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- instance.Run(ctx) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if info, statErr := os.Stat(socket); statErr == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		select {
		case runErr := <-done:
			t.Fatalf("authorization server stopped before creating its socket: %v", runErr)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("socket %s was not created", socket)
		}
		time.Sleep(10 * time.Millisecond)
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	now := time.Now().UTC().Truncate(time.Second)
	payload, _ := json.Marshal(model.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: "request-1", CorrelationID: "correlation-1",
		Operation: "symphony.knowledge.session.begin", Resource: "symphony.knowledge.repository:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Audience: "qxctl", Scope: "tops:" + testTOPSID, RequestedAt: now, RequestedExpiresAt: now.Add(10 * time.Minute),
	})
	response, err := client.Post("http://unix/v1/authorization/decisions", "application/json", bytes.NewReader(payload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	defer response.Body.Close()
	var decision model.AuthorizationDecision
	if err := json.NewDecoder(response.Body).Decode(&decision); err != nil {
		cancel()
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || decision.Effect != "allow" || decision.Capability == nil {
		cancel()
		t.Fatalf("authorization response was not allowed: status=%d decision=%+v", response.StatusCode, decision)
	}
	if audit.record.Kind != stavproducer.PolicyDecision || audit.record.Outcome != "allowed" ||
		audit.record.Actor.ID != "owner.primary" || audit.record.Configuration.State != "digests" {
		cancel()
		t.Fatalf("safe STAV policy evidence was not submitted: %+v", audit.record)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestStatusOverUnixSocket(t *testing.T) {
	socket := shortSocketPath(t)
	probe, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers: []config.ProviderConfig{{
			Name: "native",
			Kind: "native-keyring",
		}},
	}
	registry, err := provider.New(cfg.Providers)
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()
	waitForSocket(t, socket)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	response, err := client.Get("http://unix/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var status model.Status
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.ProviderCount != 1 || status.TOPSID != testTOPSID {
		t.Fatalf("unexpected status: %+v", status)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRefusesRegularFileAtSocketPath(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "ssiag.sock")
	if err := os.WriteFile(socket, []byte("do not replace"), 0600); err != nil {
		t.Fatal(err)
	}
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers:      []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Run(context.Background()); err == nil {
		t.Fatal("expected non-socket collision error")
	}
}

func TestRefusesActiveSocketPath(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "ssiag.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	defer listener.Close()

	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers:      []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "active SSIAG socket") {
		t.Fatalf("expected active-socket collision error, got %v", err)
	}
	if info, err := os.Lstat(socket); err != nil || info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("active socket was removed: info=%v error=%v", info, err)
	}
}

func TestHandlerRejectsRequestWithoutKernelPeerContext(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "ssiag.sock")
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers:      []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	var failure model.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != "request.peer_authentication_failed" {
		t.Fatalf("unexpected error: %+v", failure)
	}
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeSocket != 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("socket %s was not created", path)
}

func shortSocketPath(t *testing.T) string {
	t.Helper()
	file, err := os.CreateTemp("/tmp", "symphony-ssiag-*.sock")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

func TestRunRejectsIdentityMismatch(t *testing.T) {
	socket := shortSocketPath(t)
	wrongUID := uint32(os.Geteuid() + 1)
	wrongGID := uint32(os.Getegid() + 1)
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(wrongUID, wrongGID),
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	err = server.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "process identity mismatch") {
		t.Fatalf("expected identity mismatch error, got %v", err)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("identity-mismatched process changed the socket path: %v", err)
	}
}

func serviceAuthentication(uid, gid uint32) *config.AuthenticationConfig {
	return &config.AuthenticationConfig{
		Mechanism: "unix_peer_credentials",
		Service: &config.SubjectConfig{
			ID: config.ServiceSubjectID, Kind: config.ServiceSubjectKind, UID: &uid, GID: &gid,
		},
		Subjects: []config.SubjectConfig{},
	}
}

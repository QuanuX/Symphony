package ssiagclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestClientReadsStatusAndProviders(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		payload := `{"schema":"symphony.ssiag.providers.v1","providers":[]}`
		if request.URL.Path == "/v1/status" {
			payload = `{"schema":"symphony.ssiag.status.v1","name":"secure-identity-access-governance","version":"dev","ready":true,"mode":"user","tops_id":"018f0c3a-7b2d-7e11-8c12-0242ac120002","tops_name":"Desk","transport":"unix","provider_count":0}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.TOPSID != testTOPSID {
		t.Fatalf("unexpected status: %+v", status)
	}
	providers, err := client.Providers(context.Background())
	if err != nil || len(providers.Providers) != 0 {
		t.Fatalf("unexpected providers: %+v error=%v", providers, err)
	}
}

func TestClientUsesClosedProviderTrustEndpoints(t *testing.T) {
	seen := make(map[string]string)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		seen[request.URL.Path] = request.Method
		operation := "show"
		if request.Method == http.MethodPost {
			operation = "verify"
			var candidate ProviderTrustVerificationRequest
			if err := json.NewDecoder(request.Body).Decode(&candidate); err != nil ||
				candidate.Protocol != "symphony.ssiag.provider-trust-verification-request.v1" ||
				candidate.AuthorityBasis != "granted_permission" {
				t.Fatalf("invalid provider verification request: %+v error=%v", candidate, err)
			}
		}
		payload := providerTrustJSON(t, operation, nil)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	if _, err := client.ProviderTrust(context.Background(), "native.keychain"); err != nil {
		t.Fatal(err)
	}
	request := ProviderTrustVerificationRequest{
		Protocol:  "symphony.ssiag.provider-trust-verification-request.v1",
		RequestID: testTOPSID, CorrelationID: "018f0c3a-7b2d-7e11-8c12-0242ac120003",
		AuthorityBasis: "granted_permission",
	}
	if _, err := client.VerifyProviderTrust(context.Background(), "native.keychain", request); err != nil {
		t.Fatal(err)
	}
	if seen["/v1/provider-trust/native.keychain"] != http.MethodGet ||
		seen["/v1/provider-trust/native.keychain/verifications"] != http.MethodPost {
		t.Fatalf("provider trust routes drifted: %#v", seen)
	}
}

func TestClientRejectsUnsafeProviderTrustEvidence(t *testing.T) {
	tests := map[string]func(map[string]any){
		"result digest drift": func(value map[string]any) { value["result_digest"] = "sha256:" + strings.Repeat("0", 64) },
		"operational access":  func(value map[string]any) { value["operational_access_enabled"] = true },
		"uppercase installation digest": func(value map[string]any) {
			value["installation_digest"] = "sha256:" + strings.Repeat("A", 64)
		},
		"uppercase executable digest": func(value map[string]any) {
			value["executable_digest"] = "sha256:" + strings.Repeat("B", 64)
		},
		"raw reason": func(value map[string]any) {
			value["checks"] = []any{map[string]any{"check_id": "receipt", "outcome": "failed", "reason_code": "raw stderr leaked"}}
		},
		"secret member": func(value map[string]any) { value["credential"] = "forbidden" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			payload := providerTrustJSON(t, "show", mutate)
			transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
			})
			client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
			if _, err := client.ProviderTrust(context.Background(), "native.keychain"); err == nil {
				t.Fatal("unsafe provider trust evidence was accepted")
			}
		})
	}
}

func TestClientRejectsDuplicateProviderTrustMembers(t *testing.T) {
	payload := strings.Replace(providerTrustJSON(t, "show", nil), `"operation":"show"`, `"operation":"show","operation":"show"`, 1)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	if _, err := client.ProviderTrust(context.Background(), "native.keychain"); err == nil || !strings.Contains(err.Error(), "duplicate JSON member") {
		t.Fatalf("duplicate provider trust member was not rejected: %v", err)
	}
}

func providerTrustJSON(t *testing.T, operation string, mutate func(map[string]any)) string {
	t.Helper()
	mode := "snapshot"
	if operation == "verify" {
		mode = "fresh"
	}
	value := map[string]any{
		"protocol": "symphony.ssiag.provider-trust-result.v1", "operation": operation,
		"tops_id": testTOPSID, "provider_name": "native.keychain", "provider_kind": "native-keyring",
		"declaration_state": "declared", "trust_state": "verified", "verification_mode": mode,
		"adapter_identifier": "macos-keychain", "adapter_version": "0.1.0", "provider_protocol": "symphony.ssiag.provider.v1",
		"capabilities": []any{"capability-discovery", "metadata"}, "exportable": false, "interactive": true,
		"installation_digest": "sha256:" + strings.Repeat("a", 64), "executable_digest": "sha256:" + strings.Repeat("b", 64),
		"checks":                     []any{map[string]any{"check_id": "receipt", "outcome": "passed", "reason_code": "symphony.ssiag.provider.receipt_verified"}},
		"mutual_trust":               map[string]any{"foundation_verified_adapter": true, "adapter_verified_foundation": true},
		"operational_access_enabled": false, "provider_operations_enabled": false, "secret_channel_enabled": false,
		"observed_at": "2026-08-13T12:00:00Z", "read_only": true, "caller_class_used": false, "canonical": false,
	}
	if mutate != nil {
		mutate(value)
	}
	if _, supplied := value["result_digest"]; !supplied {
		canonical, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(canonical)
		value["result_digest"] = "sha256:" + hex.EncodeToString(digest[:])
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestClientUsesClosedPolicyAdministrationProtocols(t *testing.T) {
	seen := make(map[string]bool)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		seen[request.URL.Path] = true
		payload := `{"protocol":"symphony.ssiag.policy-result.v1","operation":"status","tops_id":"` + testTOPSID + `","source":"config","generation":0,"policy_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","state_digest":"absent","recovery_required":false,"changed":false,"recovered":false,"observed_at":"2026-08-12T12:00:00Z","read_only":true,"caller_class_used":false,"canonical":false}`
		if request.URL.Path == "/v1/policy/proposals" {
			var candidate PolicyProposalRequest
			if err := json.NewDecoder(request.Body).Decode(&candidate); err != nil || candidate.Protocol != "symphony.ssiag.policy-proposal-request.v1" {
				t.Fatalf("invalid proposal request: %+v error=%v", candidate, err)
			}
			payload = `{"protocol":"symphony.ssiag.policy-proposal.v1","operation_id":"operation-1","request_id":"request-1","correlation_id":"correlation-1","tops_id":"` + testTOPSID + `","subject":{"id":"owner","kind":"owner","authority":"unix_peer_credentials"},"authority_basis":"host_owner","expected_policy_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","change":"reset","desired_policy":null,"desired_policy_digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","config_digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","created_at":"2026-08-12T12:00:00Z","expires_at":"2026-08-12T12:05:00Z","caller_class_used":false,"canonical":false,"applied":false,"proposal_digest":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}`
		} else if request.URL.Path == "/v1/policy/apply" {
			payload = strings.Replace(payload, `"operation":"status"`, `"operation":"apply"`, 1)
		} else if request.URL.Path == "/v1/policy/recover" {
			payload = strings.Replace(payload, `"operation":"status"`, `"operation":"recover"`, 1)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	if _, err := client.PolicyStatus(context.Background()); err != nil {
		t.Fatal(err)
	}
	proposal, err := client.ProposePolicy(context.Background(), PolicyProposalRequest{Protocol: "symphony.ssiag.policy-proposal-request.v1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ApplyPolicy(context.Background(), PolicyApplyRequest{Protocol: "symphony.ssiag.policy-apply-request.v1", Proposal: proposal}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RecoverPolicy(context.Background(), PolicyRecoveryRequest{Protocol: "symphony.ssiag.policy-recovery-request.v1", Discover: true}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/policy/status", "/v1/policy/proposals", "/v1/policy/apply", "/v1/policy/recover"} {
		if !seen[path] {
			t.Fatalf("policy client did not call %s", path)
		}
	}
}

func TestReadBoundedJSONRejectsSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "policy.json")
	if err := os.WriteFile(target, []byte(`{"default_effect":"deny","max_capability_seconds":60,"grants":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	var policy AuthorizationPolicy
	if err := ReadBoundedJSON(link, &policy); err == nil {
		t.Fatal("symlinked policy input was accepted")
	}
	if err := ReadBoundedJSON(target, &policy); err != nil || policy.Grants == nil {
		t.Fatalf("valid bounded policy input failed: %+v %v", policy, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestSocketIsTOPSIsolated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	path, err := SocketForTOPS("user", testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, "runtime", "symphony", testTOPSID, "ssiag", "ssiag.sock") {
		t.Fatalf("unexpected socket %q", path)
	}
}

func TestSocketOverrideMustBeAbsolute(t *testing.T) {
	t.Setenv("SYMPHONY_SSIAG_SOCKET", "relative.sock")
	if _, err := SocketForTOPS("user", testTOPSID); err == nil {
		t.Fatal("expected absolute socket error")
	}
}

func TestSocketRejectsDisplayNameIdentity(t *testing.T) {
	for _, invalid := range []string{
		"trading-desk",
		"00000000-0000-0000-0000-000000000000",
		"018f0c3a-7b2d-0e11-8c12-0242ac120002",
		"018f0c3a-7b2d-7e11-7c12-0242ac120002",
	} {
		if _, err := SocketForTOPS("user", invalid); err == nil {
			t.Fatalf("expected identity %q to be rejected", invalid)
		}
	}
}

func TestClientRejectsUnknownResponseMembers(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		payload := `{"schema":"symphony.ssiag.status.v1","name":"secure-identity-access-governance","version":"dev","ready":true,"mode":"user","tops_id":"018f0c3a-7b2d-7e11-8c12-0242ac120002","tops_name":"Desk","transport":"unix","provider_count":0,"credential":"must-not-be-ignored"}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	if _, err := client.Status(context.Background()); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-member error, got %v", err)
	}
}

func TestClientRejectsOversizedResponse(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", maxResponseBytes+1))), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	if _, err := client.Status(context.Background()); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected response-size error, got %v", err)
	}
}

func TestNewForTOPS(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))

	configPath, err := ConfigForTOPS("user", testTOPSID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatal(err)
	}

	socket, err := canonicalSocketForTOPS("user", testTOPSID)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Config lacking service identity.
	payloadNoIdentity := fmt.Sprintf(`{"schema":"symphony.ssiag.config.v1","mode":"user","tops":{"id":"%s","name":"Desk"},"listen":{"network":"unix","address":"%s"},"authentication":{"mechanism":"unix_peer_credentials","subjects":[]},"providers":[]}`, testTOPSID, socket)
	if err := os.WriteFile(configPath, []byte(payloadNoIdentity), 0600); err != nil {
		t.Fatal(err)
	}

	_, err = NewForTOPS("user", testTOPSID, 2*time.Second)
	if err == nil {
		t.Fatal("expected error for config lacking service identity")
	}

	// 2. A real enrolled shape, including authentication and providers, is accepted.
	payloadWithIdentity := validConfigJSON(socket)
	if err := os.WriteFile(configPath, []byte(payloadWithIdentity), 0600); err != nil {
		t.Fatal(err)
	}

	client, err := NewForTOPS("user", testTOPSID, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// 3. Unsafe config permissions (group writable)
	if err := os.Chmod(configPath, 0620); err != nil {
		t.Fatal(err)
	}
	_, err = NewForTOPS("user", testTOPSID, 2*time.Second)
	if err == nil {
		t.Fatal("expected error for group-writable configuration file")
	}
	_ = os.Chmod(configPath, 0600) // restore

	// 4. Config is symlink
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	realConfig := filepath.Join(home, "real-config.json")
	if err := os.WriteFile(realConfig, []byte(payloadWithIdentity), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realConfig, configPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err = NewForTOPS("user", testTOPSID, 2*time.Second)
	if err == nil {
		t.Fatal("expected error for symlinked configuration file")
	}
}

func TestPeerVerificationFailsBeforeHTTPBytes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	configPath, _ := ConfigForTOPS("user", testTOPSID)
	socket, _ := canonicalSocketForTOPS("user", testTOPSID)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(validConfigJSON(socket)), 0o600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("Unix sockets unavailable: %v", err)
	}
	defer listener.Close()
	readResult := make(chan int, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			readResult <- -1
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		buffer := make([]byte, 1)
		n, _ := conn.Read(buffer)
		readResult <- n
	}()
	client, err := newForTOPS("user", testTOPSID, time.Second, func(net.Conn, uint32, uint32) error {
		return errors.New("deliberate peer mismatch")
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "deliberate peer mismatch") {
		t.Fatalf("expected peer mismatch, got %v", err)
	}
	if n := <-readResult; n != 0 {
		t.Fatalf("server received %d HTTP bytes before peer verification", n)
	}
}

func TestSocketOverrideRetainsConfiguredPeerIdentity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	configPath, _ := ConfigForTOPS("user", testTOPSID)
	canonicalSocket, _ := canonicalSocketForTOPS("user", testTOPSID)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(validConfigJSON(canonicalSocket)), 0o600); err != nil {
		t.Fatal(err)
	}

	override := filepath.Join(home, "override.sock")
	listener, err := net.Listen("unix", override)
	if err != nil {
		t.Skipf("Unix sockets unavailable: %v", err)
	}
	defer listener.Close()
	t.Setenv("SYMPHONY_SSIAG_SOCKET", override)

	done := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
		close(done)
	}()
	called := false
	client, err := newForTOPS("user", testTOPSID, time.Second, func(_ net.Conn, uid, gid uint32) error {
		called = true
		if uid != uint32(os.Geteuid()) || gid != uint32(os.Getegid()) {
			t.Fatalf("override changed expected identity to uid=%d gid=%d", uid, gid)
		}
		return errors.New("verification sentinel")
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client.Status(context.Background())
	<-done
	if !called {
		t.Fatal("socket override bypassed peer verification")
	}
}

func TestAuthenticatedClientReadsStatusOverUnixSocket(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	configPath, _ := ConfigForTOPS("user", testTOPSID)
	socket, _ := canonicalSocketForTOPS("user", testTOPSID)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(validConfigJSON(socket)), 0o600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("Unix sockets unavailable: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"schema":"symphony.ssiag.status.v1","name":"secure-identity-access-governance","version":"test","ready":true,"mode":"user","tops_id":"`+testTOPSID+`","tops_name":"Desk","transport":"unix","provider_count":0}`)
	})}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		<-done
	})
	client, err := NewForTOPS("user", testTOPSID, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.TOPSID != testTOPSID {
		t.Fatalf("unexpected authenticated status: %+v", status)
	}
}

func TestClientRejectsNonSocketEndpointBeforeVerification(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	configPath, _ := ConfigForTOPS("user", testTOPSID)
	socket, _ := canonicalSocketForTOPS("user", testTOPSID)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(validConfigJSON(socket)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socket, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	client, err := newForTOPS("user", testTOPSID, time.Second, func(net.Conn, uint32, uint32) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not a Unix socket") {
		t.Fatalf("expected non-socket error, got %v", err)
	}
	if called {
		t.Fatal("peer verifier was called for a non-socket endpoint")
	}
}

func TestConfigTOPSAndScopeBinding(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	configPath, _ := ConfigForTOPS("user", testTOPSID)
	socket, _ := canonicalSocketForTOPS("user", testTOPSID)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := strings.Replace(validConfigJSON(socket), `"mode":"user"`, `"mode":"system"`, 1)
	if err := os.WriteFile(configPath, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewForTOPS("user", testTOPSID, time.Second); err == nil || !strings.Contains(err.Error(), "does not match requested TOPS and scope") {
		t.Fatalf("expected scope-binding error, got %v", err)
	}
}

func validConfigJSON(socket string) string {
	return fmt.Sprintf(`{"schema":"symphony.ssiag.config.v1","mode":"user","tops":{"id":"%s","name":"Desk"},"listen":{"network":"unix","address":"%s"},"authentication":{"mechanism":"unix_peer_credentials","service":{"id":"%s","kind":"%s","uid":%d,"gid":%d},"subjects":[]},"providers":[]}`,
		testTOPSID, socket, serviceSubjectID, serviceSubjectKind, os.Geteuid(), os.Getegid())
}

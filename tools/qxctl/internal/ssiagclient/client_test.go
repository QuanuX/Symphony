package ssiagclient

import (
	"context"
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

package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ssiagTestTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestSSIAGStatusJSONFailsClosed(t *testing.T) {
	tests := map[string]string{
		"wrong identity": `{"schema":"symphony.ssiag.status.v1","name":"secure-identity-access-governance","version":"dev","ready":true,"mode":"user","tops_id":"018f0c3a-7b2d-7e11-8c12-0242ac120003","tops_name":"Wrong","transport":"unix","provider_count":0}`,
		"not ready":      `{"schema":"symphony.ssiag.status.v1","name":"secure-identity-access-governance","version":"dev","ready":false,"mode":"user","tops_id":"018f0c3a-7b2d-7e11-8c12-0242ac120002","tops_name":"Desk","transport":"unix","provider_count":0}`,
		"wrong scope":    `{"schema":"symphony.ssiag.status.v1","name":"secure-identity-access-governance","version":"dev","ready":true,"mode":"system","tops_id":"018f0c3a-7b2d-7e11-8c12-0242ac120002","tops_name":"Desk","transport":"unix","provider_count":0}`,
	}
	for name, status := range tests {
		t.Run(name, func(t *testing.T) {
			serveSSIAGTestSocket(t, status)
			if err := executeCommand([]string{"ssiag", "status", "--tops-id", ssiagTestTOPSID, "--json"}); err == nil {
				t.Fatal("expected status validation error")
			}
		})
	}
}

func TestSSIAGProvidersBindsServerIdentityBeforeQuery(t *testing.T) {
	status := `{"schema":"symphony.ssiag.status.v1","name":"secure-identity-access-governance","version":"dev","ready":true,"mode":"user","tops_id":"018f0c3a-7b2d-7e11-8c12-0242ac120003","tops_name":"Wrong","transport":"unix","provider_count":0}`
	serveSSIAGTestSocket(t, status)
	if err := executeCommand([]string{"ssiag", "providers", "--tops-id", ssiagTestTOPSID, "--json"}); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func TestSSIAGRejectsTrailingArguments(t *testing.T) {
	if err := executeCommand([]string{"ssiag", "status", "--tops-id", ssiagTestTOPSID, "extra"}); err == nil || !strings.Contains(err.Error(), "unexpected SSIAG arguments") {
		t.Fatalf("expected trailing-argument error, got %v", err)
	}
}

func serveSSIAGTestSocket(t *testing.T, status string) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "ssiag.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/v1/status" {
			_, _ = writer.Write([]byte(status))
			return
		}
		_, _ = writer.Write([]byte(`{"schema":"symphony.ssiag.providers.v1","providers":[]}`))
	})
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Setenv("SYMPHONY_SSIAG_SOCKET", socket)
	t.Cleanup(func() {
		_ = server.Close()
		_ = listener.Close()
		_ = os.Remove(socket)
	})
}

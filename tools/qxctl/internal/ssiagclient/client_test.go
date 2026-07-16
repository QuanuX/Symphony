package ssiagclient

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
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
	if path != filepath.Join(home, "runtime", "symphony", testTOPSID, "ssiag.sock") {
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

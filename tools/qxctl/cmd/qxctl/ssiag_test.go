package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, operation func() error) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = writer
	operationErr := operation()
	_ = writer.Close()
	os.Stdout = original
	data, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if operationErr != nil {
		t.Fatal(operationErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(data)
}

func TestSSIAGLifecycleGrantPlanUsesStableProfileResource(t *testing.T) {
	options := ssiagOptions{
		topsID: ssiagTestTOPSID, scope: "user", profileID: "default",
		subjectID: "host-owner", authorityBasis: "host_owner", jsonOutput: true,
	}
	first := captureStdout(t, func() error { return runSSIAGLifecycleGrantPlan(options) })
	second := captureStdout(t, func() error { return runSSIAGLifecycleGrantPlan(options) })
	if first != second {
		t.Fatal("lifecycle grant plan is not deterministic")
	}
	var plan ssiagLifecycleGrantPlan
	decoder := json.NewDecoder(bytes.NewBufferString(first))
	if err := decoder.Decode(&plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Grants) != 14 || plan.ApplyEnabled || plan.Canonical || !validTaggedDigest(plan.PlanDigest) {
		t.Fatalf("unexpected lifecycle grant plan: %+v", plan)
	}
	wantResource := lifecycleResource(ssiagTestTOPSID, "default", "changing-evidence-does-not-change-policy")
	wantCatalogResource := lifecycleCatalogResource(ssiagTestTOPSID)
	if plan.Resource != wantResource || plan.CatalogResource != wantCatalogResource {
		t.Fatalf("grant-plan resources are not exact and stable: %+v", plan)
	}
	for _, grant := range plan.Grants {
		expectedResource := wantResource
		if grant.Operation == "symphony.knowledge.lifecycle.profile.list" {
			expectedResource = wantCatalogResource
		}
		if grant.Resource != expectedResource || grant.SubjectID != "host-owner" || grant.Scope != "tops:"+ssiagTestTOPSID {
			t.Fatalf("grant escaped exact stable target: %+v", grant)
		}
	}
}

func TestLifecycleCatalogResourceCannotCollideWithProfileIdentity(t *testing.T) {
	if catalog, profile := lifecycleCatalogResource(ssiagTestTOPSID), lifecycleResource(ssiagTestTOPSID, "profiles", ""); catalog == profile {
		t.Fatal("profile catalog resource collided with a valid profile identity")
	}
}

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

package ssiagclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const (
	testProviderName = "native.keychain"
	testDigestA      = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testDigestB      = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	testDigestC      = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	testOperationID  = "018f0c3a-7b2d-7e11-8c12-0242ac120003"
)

func TestProviderBindingClientUsesExactRoutesAndBodies(t *testing.T) {
	seen := make(map[string]string)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		seen[request.URL.Path] = request.Method
		var payload string
		switch request.URL.Path {
		case "/v1/provider-installations/native.keychain":
			payload = providerInventoryJSON(t, nil)
		case "/v1/provider-bindings/native.keychain":
			payload = providerBindingStatusJSON(t, nil)
		case "/v1/provider-bindings/native.keychain/plans":
			requireExactRequest(t, request, map[string]string{
				"installation_id": testDigestA, "expected_state_digest": "absent", "reason": "activate exact installation",
			})
			payload = providerBindingPlanJSON(t, nil)
		case "/v1/provider-bindings/native.keychain/apply":
			requireExactRequest(t, request, map[string]string{"plan_digest": testDigestB, "expected_state_digest": "absent"})
			payload = providerBindingResultJSON(t, "apply", nil)
		case "/v1/provider-bindings/native.keychain/attempts/" + testOperationID:
			payload = providerBindingResultJSON(t, "apply-status", nil)
		case "/v1/provider-bindings/native.keychain/recover":
			requireExactRequest(t, request, map[string]string{"expected_state_digest": "absent", "reason": "resume exact attempt"})
			payload = providerBindingResultJSON(t, "recover", nil)
		default:
			t.Fatalf("unexpected provider binding route %s", request.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	ctx := context.Background()
	if _, err := client.ProviderInstallations(ctx, testProviderName); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ProviderBindingStatus(ctx, testProviderName); err != nil {
		t.Fatal(err)
	}
	if _, err := client.PlanProviderBinding(ctx, testProviderName, ProviderBindingPlanRequest{
		InstallationID: testDigestA, ExpectedStateDigest: "absent", Reason: "activate exact installation",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ApplyProviderBinding(ctx, testProviderName, ProviderBindingApplyRequest{
		PlanDigest: testDigestB, ExpectedStateDigest: "absent",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ProviderBindingApplyStatus(ctx, testProviderName, testOperationID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RecoverProviderBinding(ctx, testProviderName, ProviderBindingRecoveryRequest{
		ExpectedStateDigest: "absent", Reason: "resume exact attempt",
	}); err != nil {
		t.Fatal(err)
	}
	for path, method := range map[string]string{
		"/v1/provider-installations/native.keychain":                        http.MethodGet,
		"/v1/provider-bindings/native.keychain":                             http.MethodGet,
		"/v1/provider-bindings/native.keychain/plans":                       http.MethodPost,
		"/v1/provider-bindings/native.keychain/apply":                       http.MethodPost,
		"/v1/provider-bindings/native.keychain/attempts/" + testOperationID: http.MethodGet,
		"/v1/provider-bindings/native.keychain/recover":                     http.MethodPost,
	} {
		if seen[path] != method {
			t.Errorf("route %s method=%q, want %q", path, seen[path], method)
		}
	}
}

func TestProviderBindingClientRejectsUnsafeEvidence(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"operational access": func(value map[string]any) { value["operational_access_enabled"] = true },
		"secret channel":     func(value map[string]any) { value["secret_channel_enabled"] = true },
		"caller class":       func(value map[string]any) { value["caller_class_used"] = true },
		"digest drift":       func(value map[string]any) { value["result_digest"] = testDigestC },
		"unknown member":     func(value map[string]any) { value["credential"] = "forbidden" },
		"uppercase digest":   func(value map[string]any) { value["state_digest"] = "sha256:" + strings.Repeat("A", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			payload := providerBindingStatusJSON(t, mutate)
			transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
			})
			client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
			if _, err := client.ProviderBindingStatus(context.Background(), testProviderName); err == nil {
				t.Fatal("unsafe provider binding status was accepted")
			}
		})
	}
}

func TestProviderBindingClientRejectsSelectorsOutsideOpaqueContract(t *testing.T) {
	client := &Client{}
	for name, request := range map[string]ProviderBindingPlanRequest{
		"path":    {InstallationID: "/tmp/provider", ExpectedStateDigest: "absent", Reason: "bind"},
		"version": {InstallationID: "1.2.3", ExpectedStateDigest: "absent", Reason: "bind"},
		"newline": {InstallationID: testDigestA, ExpectedStateDigest: "absent", Reason: "bind\nsecret"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := client.PlanProviderBinding(context.Background(), testProviderName, request); err == nil {
				t.Fatal("unsafe selector or reason was accepted")
			}
		})
	}
}

func TestProviderBindingReasonMatchesCanonicalUnicodeBound(t *testing.T) {
	if !validReason(strings.Repeat("界", 1024)) {
		t.Fatal("canonical 1024-character UTF-8 reason was rejected")
	}
	for name, value := range map[string]string{
		"too_long": strings.Repeat("界", 1025),
		"newline":  "unsafe\nreason",
		"invalid":  string([]byte{0xff}),
	} {
		if validReason(value) {
			t.Fatalf("%s reason was accepted", name)
		}
	}
}

func TestProviderBindingClientRejectsCyclicActionGraph(t *testing.T) {
	payload := providerBindingPlanJSON(t, func(value map[string]any) {
		value["actions"] = []any{
			map[string]any{"action_id": "first", "kind": "bind", "direction": "forward", "depends_on": []any{"second"}},
			map[string]any{"action_id": "second", "kind": "bind", "direction": "forward", "depends_on": []any{"first"}},
		}
	})
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
	})
	client := &Client{httpClient: &http.Client{Transport: transport}, baseURL: "http://unix"}
	_, err := client.PlanProviderBinding(context.Background(), testProviderName, ProviderBindingPlanRequest{
		InstallationID: testDigestA, ExpectedStateDigest: "absent", Reason: "activate exact installation",
	})
	if err == nil || !strings.Contains(err.Error(), "action graph") {
		t.Fatalf("cyclic provider binding action graph was accepted: %v", err)
	}
}

func requireExactRequest(t *testing.T, request *http.Request, want map[string]string) {
	t.Helper()
	var got map[string]json.RawMessage
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("request member count=%d, want=%d: %#v", len(got), len(want), got)
	}
	for key, value := range want {
		var decoded string
		if err := json.Unmarshal(got[key], &decoded); err != nil || decoded != value {
			t.Fatalf("request %s=%q error=%v, want %q", key, decoded, err, value)
		}
	}
}

func providerInventoryJSON(t *testing.T, mutate func(map[string]any)) string {
	return digestJSON(t, map[string]any{
		"protocol": ProviderInstallationInventoryProtocol, "tops_id": testTOPSID, "scope": "user",
		"provider_name": testProviderName, "provider_kind": "native-keyring",
		"installations": []any{map[string]any{
			"installation_id": testDigestA, "adapter_identifier": "macos-keychain", "adapter_version": "0.1.0",
			"provider_protocol": "symphony.ssiag.provider.v1", "command_protocol": "symphony.ssiag.provider.control.v1",
			"receipt_digest": testDigestA, "executable_digest": testDigestB, "foundation_version": "0.1.0",
			"foundation_receipt_digest": testDigestB, "foundation_executable_digest": testDigestC,
			"compatibility_state": "exact", "reason_code": "symphony.ssiag.provider.installation.exact",
		}},
		"observed_at": "2026-08-14T12:00:00Z", "operational_access_enabled": false,
		"provider_operations_enabled": false, "secret_channel_enabled": false, "read_only": true, "canonical": false,
	}, "inventory_digest", mutate)
}

func providerBindingStatusJSON(t *testing.T, mutate func(map[string]any)) string {
	return digestJSON(t, map[string]any{
		"protocol": ProviderBindingStatusProtocol, "tops_id": testTOPSID, "scope": "user",
		"provider_name": testProviderName, "provider_kind": "native-keyring", "binding_state": "bound", "generation": 1,
		"installation_id": testDigestA, "previous_installation_id": "not_applicable", "state_digest": testDigestB,
		"attempt_state": "none", "attempt_operation_id": "not_applicable", "attempt_digest": "not_applicable",
		"recovery_required": false, "reason_code": "symphony.ssiag.provider.binding.observed",
		"observed_at": "2026-08-14T12:00:00Z", "operational_access_enabled": false,
		"provider_operations_enabled": false, "secret_channel_enabled": false, "read_only": true,
		"caller_class_used": false, "canonical": false,
	}, "result_digest", mutate)
}

func providerBindingPlanJSON(t *testing.T, mutate func(map[string]any)) string {
	return digestJSON(t, map[string]any{
		"protocol": ProviderBindingPlanProtocol, "plan_id": testOperationID, "tops_id": testTOPSID, "scope": "user",
		"provider_name": testProviderName, "provider_kind": "native-keyring", "desired_state": "bound",
		"installation_id": testDigestA, "expected_state_digest": "absent", "current_state_digest": "absent",
		"inventory_digest": testDigestC,
		"actions":          []any{map[string]any{"action_id": "bind-exact", "kind": "bind", "direction": "forward", "depends_on": []any{}}},
		"applicable":       true, "changed": true, "recovery_required": false, "reason": "activate exact installation",
		"expires_at": "2026-08-14T12:05:00Z", "operational_access_enabled": false,
		"provider_operations_enabled": false, "secret_channel_enabled": false, "caller_class_used": false,
		"canonical": false,
	}, "plan_digest", mutate)
}

func providerBindingResultJSON(t *testing.T, operation string, mutate func(map[string]any)) string {
	return digestJSON(t, map[string]any{
		"protocol": ProviderBindingResultProtocol, "operation": operation, "operation_id": testOperationID,
		"tops_id": testTOPSID, "scope": "user", "provider_name": testProviderName, "provider_kind": "native-keyring",
		"binding_state": "bound", "generation": 1, "installation_id": testDigestA,
		"previous_installation_id": "not_applicable", "state_digest": testDigestB, "attempt_state": "committed",
		"attempt_digest": testDigestC, "receipt_digest": testDigestA, "changed": operation != "apply-status",
		"recovered": operation == "recover", "recovery_required": false,
		"reason_code": "symphony.ssiag.provider.binding.succeeded", "observed_at": "2026-08-14T12:00:00Z",
		"operational_access_enabled": false, "provider_operations_enabled": false, "secret_channel_enabled": false,
		"caller_class_used": false, "canonical": false,
	}, "result_digest", mutate)
}

func digestJSON(t *testing.T, value map[string]any, field string, mutate func(map[string]any)) string {
	t.Helper()
	if mutate != nil {
		mutate(value)
	}
	if _, supplied := value[field]; !supplied {
		canonical, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		value[field] = sha256Tagged(canonical)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

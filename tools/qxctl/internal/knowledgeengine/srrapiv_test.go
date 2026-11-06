package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func srrapivTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func srrapivTestResult(t *testing.T, operation string, payload []byte, inner map[string]any) map[string]any {
	t.Helper()
	p, err := srrapivObject(payload, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]any{"protocol": SRRAPIVOutputProtocol, "operation": operation, "engine_version": SRRAPIVVersion,
		"request_digest": digestBytes(srrapivTestJSON(t, p)), "result": inner}
	result["result_digest"] = digestBytes(srrapivTestJSON(t, result))
	return result
}

func TestSRRAPIVTransportAndCorrespondence(t *testing.T) {
	payload := []byte(`{"protocol":"symphony.srrapiv.request.v1","params":{"price":"123.45","context":"18446744073709551615"}}`)
	contract := []byte(`{"input":{"type":"object","additionalProperties":false,"required":["protocol","params"],"properties":{"protocol":{"const":"symphony.srrapiv.request.v1"},"params":{"type":"object","additionalProperties":true}}},"result":{"type":"object","additionalProperties":true}}`)
	inner := map[string]any{"ok": false, "errors": []any{map[string]any{"code": "srrapiv.binding.invalid", "path": "/binding"}}}
	base := srrapivTestResult(t, "runtime_invoke", payload, inner)
	if err := ValidateSRRAPIVResult("runtime_invoke", SRRAPIVVersion, payload, srrapivTestJSON(t, base), contract); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"protocol", "operation", "engine_version", "request_digest", "result", "result_digest"} {
		t.Run(field, func(t *testing.T) {
			mutated := srrapivTestResult(t, "runtime_invoke", payload, inner)
			mutated[field] = "tampered"
			if err := ValidateSRRAPIVResult("runtime_invoke", SRRAPIVVersion, payload, srrapivTestJSON(t, mutated), contract); err == nil {
				t.Fatalf("accepted tampered %s", field)
			}
		})
	}
	base["extra"] = true
	delete(base, "result_digest")
	base["result_digest"] = digestBytes(srrapivTestJSON(t, base))
	if ValidateSRRAPIVResult("runtime_invoke", SRRAPIVVersion, payload, srrapivTestJSON(t, base), contract) == nil {
		t.Fatal("accepted extra field with valid digest")
	}
	for _, invalid := range []string{
		`{"protocol":"symphony.srrapiv.request.v1","params":{"price":1.5}}`,
		`{"protocol":"symphony.srrapiv.request.v1","params":{"context":9007199254740992}}`,
		`{"protocol":"symphony.srrapiv.request.v1","protocol":"symphony.srrapiv.request.v1","params":{}}`,
		`{"protocol":"symphony.srrapiv.request.v1","params":{"invalid":"\ud800"}}`,
	} {
		if _, err := srrapivObject([]byte(invalid), maxRequestBytes); err == nil {
			t.Fatal("unsafe JSON accepted")
		}
	}
}

func TestSRRAPIVTransportShape(t *testing.T) {
	raw := []byte(`{"type":"object","additionalProperties":false,"required":["generation","params"],"properties":{"generation":{"type":"integer","minimum":1,"maximum":3},"params":{"type":"object","additionalProperties":{"anyOf":[{"type":"string"},{"type":"boolean"}]}}}}`)
	schema, err := srrapivObject(raw, 4096)
	if err != nil {
		t.Fatal(err)
	}
	good := map[string]any{"generation": int64(1), "params": map[string]any{"price": "1.25", "flag": true}}
	if !srrapivTransportShape(schema, good, 0) {
		t.Fatal("valid native payload shape rejected")
	}
	for _, v := range []map[string]any{
		{"generation": int64(0), "params": map[string]any{}},
		{"generation": int64(4), "params": map[string]any{}},
		{"generation": int64(1), "params": map[string]any{"price": int64(2)}},
		{"generation": int64(1), "params": map[string]any{}, "extra": true},
	} {
		if srrapivTransportShape(schema, v, 0) {
			t.Fatal("invalid transport shape accepted")
		}
	}
	schema["$ref"] = "https://invalid.example/schema"
	if srrapivTransportShape(schema, good, 0) {
		t.Fatal("unadmitted schema keyword accepted")
	}
}

func TestSRRAPIVInputFileBoundary(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "request.json")
	values := make([]any, 40000)
	for i := range values {
		values[i] = int64(1)
	}
	raw := srrapivTestJSON(t, map[string]any{"protocol": SRRAPIVInputProtocol, "values": values})
	if err := os.WriteFile(input, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSRRAPIVPayload(input); err != nil {
		t.Fatalf("declared native value budget unavailable: %v", err)
	}
	link := filepath.Join(directory, "link.json")
	if err := os.Symlink(input, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSRRAPIVPayload(link); err == nil {
		t.Fatal("followed input symlink")
	}
	if err := os.WriteFile(input, []byte(`{"a":1,"a":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSRRAPIVPayload(input); err == nil {
		t.Fatal("duplicate input keys accepted")
	}
}

func TestSRRAPIVExactSelection(t *testing.T) {
	for _, version := range []string{"", "latest", "0.1.1-dev", "13.7.0.0"} {
		if _, err := InspectSRRAPIV("/private-marker", version, "api_describe"); err == nil || strings.Contains(err.Error(), "private-marker") {
			t.Fatal("version selection did not fail safely")
		}
	}
	if _, err := InspectSRRAPIV("/private-marker", SRRAPIVVersion, "unrecognized"); err == nil || strings.Contains(err.Error(), "private-marker") {
		t.Fatal("operation selection did not fail safely")
	}
}

func TestSRRAPIVReviewedResourcePins(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "modules", SRRAPIVModuleID)
	ownerBytes, err := os.ReadFile(filepath.Join(root, "OWNER-INTERFACE.json"))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := srrapivObject(ownerBytes, maxRequestBytes)
	if err != nil || digestBytes(srrapivTestJSON(t, owner)) != srrapivInterfaceDigest {
		t.Fatal("native owner declaration differs from reviewed client pin")
	}
	for name, expected := range srrapivInterfaceResources {
		raw, err := os.ReadFile(filepath.Join(root, "schemas", "v1", name))
		if err != nil || digestBytes(raw) != expected {
			t.Fatalf("native resource differs from reviewed client pin: %s", name)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "schemas", "v1", "srrapiv.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := srrapivObject(raw, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	requests := schema["requests"].(map[string]any)
	results := schema["results"].(map[string]any)
	if len(requests) != len(SRRAPIVOperations) || len(results) != len(SRRAPIVOperations) {
		t.Fatal("native/client operation census differs")
	}
	for _, op := range SRRAPIVOperations {
		if requests[op.Operation] == nil || results[op.Operation] == nil {
			t.Fatalf("native schema missing %s", op.Operation)
		}
	}
}

func TestSRRAPIVInstalledNative(t *testing.T) {
	prefix := os.Getenv("SRRAPIV_TEST_PREFIX")
	if prefix == "" {
		t.Skip("select the exact installed native SRRAPIV prefix")
	}
	for _, op := range SRRAPIVOperations {
		for _, template := range []bool{false, true} {
			if _, _, err := SRRAPIVResource(prefix, SRRAPIVVersion, op.Operation, template); err != nil {
				t.Fatalf("%s resource template=%t: %v", op.Operation, template, err)
			}
		}
	}
	response, err := InvokeSRRAPIV(context.Background(), prefix, SRRAPIVVersion, t.TempDir(), "api_describe", []byte(`{"protocol":"symphony.srrapiv.request.v1"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if json.Unmarshal(response.Result, &result) != nil || response.EngineID != SRRAPIVEngineID || response.Operation != "api_describe" || result["protocol"] != SRRAPIVOutputProtocol {
		t.Fatal("native description lost exact owner identity")
	}
}

// This exercises the finite process client's process-group cleanup against the
// real detached resident owner. The fixture must be an offline start config;
// no SDK initialization, login, subscription or trading method is invoked.
func TestSRRAPIVInstalledResidentLifecycle(t *testing.T) {
	prefix, configFile := os.Getenv("SRRAPIV_TEST_PREFIX"), os.Getenv("SRRAPIV_TEST_CONFIG")
	if prefix == "" || configFile == "" {
		t.Skip("select the exact native prefix and offline start fixture")
	}
	config, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	base := "/tmp"
	if runtime.GOOS == "darwin" {
		base = "/private/tmp"
	}
	directory, err := os.MkdirTemp(base, "srrapiv-qxctl-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	privateConfig := filepath.Join(directory, "config.json")
	if err = os.WriteFile(privateConfig, config, 0600); err != nil {
		t.Fatal(err)
	}
	runtimeDir := filepath.Join(directory, "runtime")
	run := func(operation string, payload map[string]any) (map[string]any, error) {
		payload["protocol"] = SRRAPIVInputProtocol
		payload["runtime_dir"] = runtimeDir
		response, err := InvokeSRRAPIV(context.Background(), prefix, SRRAPIVVersion, directory, operation, srrapivTestJSON(t, payload))
		if err != nil {
			return nil, err
		}
		result, err := srrapivObject(response.Result, maxResponseBytes)
		if err != nil {
			return nil, err
		}
		inner, ok := result["result"].(map[string]any)
		if !ok {
			return nil, srrapivInvalid()
		}
		return inner, nil
	}
	started, err := run("runtime_start", map[string]any{"config_file": privateConfig})
	if err != nil {
		t.Fatal(err)
	}
	generation, ok := started["generation"].(string)
	if !ok || generation == "" {
		t.Fatal("start lost owner generation")
	}
	cleanStop := false
	defer func() {
		if !cleanStop {
			_, _ = run("runtime_stop", map[string]any{"expected_generation": generation})
		}
	}()
	status, err := run("runtime_status", map[string]any{})
	if err != nil || status["generation"] != generation || status["state"] != "running" {
		t.Fatalf("resident did not survive finite client exit: %v %v", status, err)
	}
	if _, err = run("runtime_status", map[string]any{"expected_generation": "stale-generation"}); err == nil {
		t.Fatal("stale generation was accepted")
	}
	if _, err = run("runtime_attempts", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if _, err = run("runtime_events", map[string]any{"expected_generation": generation}); err != nil {
		t.Fatal(err)
	}
	stopped, err := run("runtime_stop", map[string]any{"expected_generation": generation})
	if err != nil || (stopped["state"] != "stopped" && stopped["state"] != "stopping") {
		t.Fatalf("native stop failed: %v %v", stopped, err)
	}
	if stopped["state"] == "stopping" {
		deadline := time.Now().Add(5 * time.Second)
		complete := false
		for time.Now().Before(deadline) {
			status, err := run("runtime_status", map[string]any{"expected_generation": generation})
			if err != nil {
				t.Fatal(err)
			}
			terminal, _ := status["stop_result"].(map[string]any)
			if terminal["state"] == "stopped" {
				complete = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !complete {
			t.Fatal("offline native stop did not complete")
		}
	}
	cleanStop = true
}

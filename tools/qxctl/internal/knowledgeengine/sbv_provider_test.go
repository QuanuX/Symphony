package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSBVProviderEvidenceCorrespondence(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	selection := map[string]any{"protocol": "symphony.sbv.native-provider-selection.v1", "library": map[string]any{"path": "/provider.dylib", "expected_sha256": hash}, "id": "example", "version": "1", "role": "strategy", "concurrency": "serialized_instance", "parameters": map[string]any{"label": "native line separator"}, "extensions": map[string]any{}, "input_profile": "symphony.sbv.provider-databento-mbo-event.v1", "dependencies": map[string]any{"capture": "declared_artifacts", "description": "explicit dependency", "artifacts": []any{map[string]any{"path": "/weights.dat", "expected_sha256": hash}}}}
	canonical, err := sbvNativeCanonical(selection)
	if err != nil {
		t.Fatal(err)
	}
	descriptor := `{"protocol":"symphony.sbv.native-provider-descriptor.v1","id":"example","version":"1","reproducibility":"deterministic_declared","cancellation":"cooperative_polling","config_schema":{"type":"object","properties":{"threshold":{"minimum":0.5}}}}`
	evidence := map[string]any{"protocol": "symphony.sbv.native-provider-evidence.v1", "abi_version": "1", "library": map[string]any{"path": "/provider.dylib", "expected_sha256": hash, "bytes": "1234"}, "selection_sha256": digestBytes(canonical)[7:], "descriptor_sha256": digestBytes([]byte(descriptor))[7:], "descriptor_json": descriptor, "id": "example", "version": "1", "role": "strategy", "concurrency": "serialized_instance", "reproducibility": "deterministic_declared", "cancellation": "cooperative_polling", "dependencies": map[string]any{"capture": "declared_artifacts", "description": "explicit dependency", "artifacts": []any{map[string]any{"path": "/weights.dat", "expected_sha256": hash, "bytes": "42"}}}, "configuration_validation": "provider owned", "loading": map[string]any{"profile": "verified copy", "symbol": "symphony_sbv_provider_api_v1", "visibility": "local", "dependency_resolution": "platform loader", "trust": "caller selected native code", "isolation": false}}
	request := map[string]any{"provider": selection, "extensions": map[string]any{"label": "caller choice"}}
	result := map[string]any{"protocol": "symphony.sbv.provider-inspect.v1", "engine_version": SBVAdministrationInterfaceVersion, "provider": evidence, "extensions": request["extensions"]}
	raw, _ := json.Marshal(result)
	if err := validateSBVResult("provider_inspect", request, raw); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(map[string]any){
		func(r map[string]any) { r["extensions"] = map[string]any{} },
		func(r map[string]any) { r["engine_version"] = "0.19.0-dev" },
		func(r map[string]any) { r["provider"].(map[string]any)["id"] = "another" },
		func(r map[string]any) { r["provider"].(map[string]any)["selection_sha256"] = hash },
		func(r map[string]any) { r["provider"].(map[string]any)["descriptor_json"] = "{}" },
		func(r map[string]any) { r["provider"].(map[string]any)["reproducibility"] = "uncaptured" },
		func(r map[string]any) { r["provider"].(map[string]any)["library"].(map[string]any)["bytes"] = "01" },
		func(r map[string]any) {
			r["provider"].(map[string]any)["dependencies"].(map[string]any)["artifacts"].([]any)[0].(map[string]any)["expected_sha256"] = "wrong"
		},
		func(r map[string]any) { r["provider"].(map[string]any)["loading"].(map[string]any)["isolation"] = true },
		func(r map[string]any) { delete(r["provider"].(map[string]any), "configuration_validation") },
	} {
		var changed map[string]any
		if err := json.Unmarshal(raw, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(changed)
		b, _ := json.Marshal(changed)
		if validateSBVResult("provider_inspect", request, b) == nil {
			t.Fatalf("changed provider evidence admitted: %s", b)
		}
	}
}

func TestSBVInstalledNativeProvider(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_PROVIDER_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("set exact installation and emitted native provider fixture directory")
	}
	read := func(path string) map[string]any {
		t.Helper()
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		x, e := sqavObject(b, 128<<20)
		if e != nil {
			t.Fatal(e)
		}
		return x
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p map[string]any) Response {
		t.Helper()
		raw, e := json.Marshal(p)
		if e != nil {
			t.Fatal(e)
		}
		r, e := InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, op, raw)
		if e != nil {
			t.Fatal(op, e)
		}
		return r
	}
	generate := read(filepath.Join(fixture, "generate-request.json"))
	generate["output_path"] = filepath.Join(dir, "census.json")
	inspected := invoke("provider_inspect", map[string]any{"protocol": "symphony.sbv.provider-inspect-input.v1", "provider": generate["provider"], "extensions": map[string]any{}})
	inspect, err := sqavObject(inspected.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "provider_inspect")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := sqavObject(raw, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !sqvTransportShape(schema["output"], inspect, 0) {
		t.Fatal("provider inspection differs from installed schema")
	}
	invoke("generate_census", generate)
	census := read(generate["output_path"].(string))
	data := func(r map[string]any, k string) any {
		return r["sections"].(map[string]any)[k].(map[string]any)["data"]
	}
	evaluate := read(filepath.Join(fixture, "evaluate-request.json"))
	evaluate["census"] = map[string]any{"path": generate["output_path"], "expected_sha256": census["content_sha256"], "pointer": ""}
	raw, err = SBVSchema(prefix, SBVAdministrationInterfaceVersion, "evaluate")
	if err != nil {
		t.Fatal(err)
	}
	schema, err = sqavObject(raw, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := schema["$defs"].(map[string]any)
	for _, mode := range []string{"serialized_instance", "per_worker_instances", "shared_reentrant_instance"} {
		evaluate["model"].(map[string]any)["parameters"].(map[string]any)["provider"].(map[string]any)["concurrency"] = mode
		evaluate["output_path"] = filepath.Join(dir, mode+".json")
		invoke("evaluate", evaluate)
		result := read(evaluate["output_path"].(string))
		if !reflect.DeepEqual(data(census, "census"), data(result, "census")) {
			t.Fatal("model changed provider census")
		}
		for _, name := range []string{"census"} {
			if !sqvTransportShape(defs["census_evidence"], data(result, name), 0) {
				t.Fatal("provider census schema mismatch")
			}
		}
		if !sqvTransportShape(defs["native_model_evidence"], data(result, "provider"), 0) {
			t.Fatal("native model evidence schema mismatch")
		}
		if !sqvTransportShape(sbvBindOpenObjects(defs["native_provider_census"], data(result, "census").(map[string]any)["declaration"], 0), data(result, "census").(map[string]any)["declaration"], 0) {
			t.Fatal("native provider declaration schema mismatch")
		}
		for _, row := range data(result, "execution").([]any) {
			if !sqvTransportShape(defs["model_outcome"], row, 0) {
				t.Fatal("provider outcome schema mismatch")
			}
		}
		invoke("result_query", map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": evaluate["output_path"], "expected_sha256": result["content_sha256"], "pointer": "/sections/provider", "limit": "256", "cursor": ""})
	}
}

func TestSBVProviderOpenConfiguration(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"provider": map[string]any{"type": "object", "properties": map[string]any{"parameters": map[string]any{"type": "object", "additionalProperties": true}}, "required": []any{"parameters"}, "additionalProperties": false}}, "required": []any{"provider"}, "additionalProperties": false}
	for _, value := range []any{map[string]any{"nested": map[string]any{"weight": "1/3"}, "new-choice": []any{true, "arbitrary"}}, map[string]any{}} {
		if !sbvInputShape(schema, map[string]any{"provider": map[string]any{"parameters": value}}) {
			t.Fatal("provider object rejected")
		}
	}
	for _, value := range []any{nil, []any{}, "not-object", true} {
		if sbvInputShape(schema, map[string]any{"provider": map[string]any{"parameters": value}}) {
			t.Fatal("nonobject configuration admitted")
		}
	}
	if sqvTransportShape(map[string]any{"type": "object", "additionalProperties": true}, map[string]any{"x": "1"}, 0) {
		t.Fatal("shared owner schema unexpectedly loosened")
	}
}

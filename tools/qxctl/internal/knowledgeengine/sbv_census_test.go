package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSBVInstalledCensusContinuity(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_CENSUS_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("set exact installation and emitted census fixture directory")
	}
	read := func(path string) map[string]any {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		x, err := sqavObject(b, 128<<20)
		if err != nil {
			t.Fatal(err)
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
	invoke := func(op string, request map[string]any) error {
		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		_, err = InvokeSBV(context.Background(), prefix, "0.22.0-dev", cwd, op, b)
		return err
	}
	run := read(filepath.Join(fixture, "run-request.json"))
	run["output_path"] = filepath.Join(dir, "run.json")
	if err := invoke("run", run); err != nil {
		t.Fatal(err)
	}
	native := read(run["output_path"].(string))
	section := func(result map[string]any, name string) any {
		return result["sections"].(map[string]any)[name].(map[string]any)["data"]
	}
	request := read(filepath.Join(fixture, "evaluate-request.json"))
	request["census"] = map[string]any{"path": run["output_path"], "expected_sha256": native["content_sha256"], "pointer": ""}
	request["output_path"] = filepath.Join(dir, "evaluate.json")
	if err := invoke("evaluate", request); err != nil {
		t.Fatal(err)
	}
	first := read(request["output_path"].(string))
	if !reflect.DeepEqual(section(native, "signals"), section(first, "signals")) ||
		!reflect.DeepEqual(section(native, "census"), section(first, "census")) {
		t.Fatal("model evaluation changed retained native census")
	}
	raw, err := SBVSchema(prefix, "0.22.0-dev", "evaluate")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := sqavObject(raw, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := schema["$defs"].(map[string]any)
	for name, definition := range map[string]string{"census": "census_evidence", "census_reference": "census_reference"} {
		if !sqvTransportShape(defs[definition], section(first, name), 0) {
			t.Fatalf("undiscoverable %s schema", name)
		}
	}
	request["census"] = map[string]any{"path": request["output_path"], "expected_sha256": first["content_sha256"], "pointer": ""}
	request["output_path"] = filepath.Join(dir, "second-model.json")
	request["model"] = map[string]any{"protocol": "symphony.sbv.model-selection.v1", "id": "observed_trade_levels", "version": "1", "horizon_ns": "1100", "parameters": map[string]any{"levels_per_side": "1", "include_anchor": true, "thin_support": "unavailable"}}
	if err := os.Remove(run["output_path"].(string)); err != nil {
		t.Fatal(err)
	}
	if err := invoke("evaluate", request); err != nil {
		t.Fatal(err)
	}
	second := read(request["output_path"].(string))
	if !reflect.DeepEqual(section(first, "census"), section(second, "census")) {
		t.Fatal("reevaluation requires missing ancestor or changes census")
	}
	query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": request["output_path"], "expected_sha256": second["content_sha256"], "pointer": "/sections/census/data/census_sha256", "limit": "1", "cursor": ""}
	if err := invoke("result_query", query); err != nil {
		t.Fatal(err)
	}
	request["census"].(map[string]any)["pointer"] = "/sections/signals/data"
	request["output_path"] = filepath.Join(dir, "invalid.json")
	if err := invoke("evaluate", request); err == nil {
		t.Fatal("bare signal array admitted without source census identity")
	}
	if _, err := os.Stat(request["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("invalid reference published")
	}
}

func TestSBVInstalledEconomicComposition(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_COMPOSITION_FIXTURES")
	if prefix == "" || fixture == "" {
		t.Skip("set exact installation and native-emitted composition fixtures")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := SBVSchema(prefix, "0.22.0-dev", "compose_economics")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := sqavObject(raw, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := schema["$defs"].(map[string]any)
	for _, kind := range []string{"independent", "joint"} {
		raw, err := os.ReadFile(filepath.Join(fixture, kind+"-request.json"))
		if err != nil {
			t.Fatal(err)
		}
		request, err := sqavObject(raw, maxRequestBytes)
		if err != nil {
			t.Fatal(err)
		}
		request["output_path"] = filepath.Join(dir, kind+".json")
		raw, err = json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = InvokeSBV(context.Background(), prefix, "0.22.0-dev", cwd, "compose_economics", raw); err != nil {
			t.Fatal(err)
		}
		raw, err = os.ReadFile(request["output_path"].(string))
		if err != nil {
			t.Fatal(err)
		}
		result, err := sqavObject(raw, 128<<20)
		if err != nil {
			t.Fatal(err)
		}
		sections := result["sections"].(map[string]any)
		data := func(name string) any { return sections[name].(map[string]any)["data"] }
		if !sqvTransportShape(defs["economic_composition"], data("distributions"), 0) {
			t.Fatal("composition shape differs from discoverable schema")
		}
		for _, row := range data("studies").([]any) {
			if !sqvTransportShape(defs["composed_economic_study"], row, 0) {
				t.Fatal("composition study shape differs from discoverable schema")
			}
		}
		for _, row := range data("composition_sources").([]any) {
			if !sqvTransportShape(defs["census_evidence"], row.(map[string]any)["census"], 0) {
				t.Fatal("composition did not retain source census")
			}
		}
	}
}

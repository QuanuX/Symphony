package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVAdaptiveShapes(t *testing.T) {
	catalogue := partitionedCatalogue(t, "admin.schema.json")
	templates := partitionedCatalogue(t, "admin.templates.json")
	requests := catalogue["requests"].(map[string]any)
	variants := templates["variants"].(map[string]any)
	for _, spec := range []struct{ op, variant string }{{"experiment", "adaptive"}, {"compare", "selection"}} {
		t.Run(spec.op, func(t *testing.T) {
			p := recoveryClone(t, variants[spec.op].(map[string]any)[spec.variant].(map[string]any))
			if !sbvInputShape(requests[spec.op].(map[string]any), p) {
				t.Fatal("template rejected")
			}
			if spec.op == "experiment" {
				p["proposal_events"] = []any{map[string]any{"event_id": "e", "proposal_id": "p", "state": "proposed", "trial_id": nil, "reason": "", "parameters": map[string]any{}, "lineage": map[string]any{}}}
				if !sbvInputShape(requests[spec.op].(map[string]any), p) {
					t.Fatal("proposal rejected")
				}
				p["revision"].(map[string]any)["index"] = "-1"
			} else {
				p["selection"].(map[string]any)["tie_break"] = "hidden_default"
			}
			if sbvInputShape(requests[spec.op].(map[string]any), p) {
				t.Fatal("malformed control admitted")
			}
		})
	}
}

func TestSBVInstalledAdaptive(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_ADAPTIVE_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("installed native and adaptive fixture required")
	}
	raw, err := os.ReadFile(filepath.Join(fixtures, "nested-selection-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p["directory"] = filepath.Join(dir, "journal")
	if err = os.Mkdir(p["directory"].(string), 0700); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	schema, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "experiment")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sbvDecodeOwnedResource(schema)
	if err != nil {
		t.Fatal(err)
	}
	trialShape := contract["$defs"].(map[string]any)["experiment_trial"]
	invoke := func(name string) map[string]any {
		t.Helper()
		p["output_path"] = filepath.Join(dir, name+".json")
		request, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		response, err := InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, "experiment", request)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := sqavObject(response.Result, maxResponseBytes)
		if err != nil {
			t.Fatal(err)
		}
		if receipt["status"] != "completed" {
			t.Fatalf("incomplete receipt: %s", response.Result)
		}
		bytes, err := os.ReadFile(p["output_path"].(string))
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err = json.Unmarshal(bytes, &result); err != nil {
			t.Fatal(err)
		}
		search := result["sections"].(map[string]any)["search"].(map[string]any)["data"].(map[string]any)
		for _, row := range search["trials"].([]any) {
			if !sbvSchemaShape(trialShape, row, 0) {
				t.Fatal("trial violates installed output definition")
			}
		}
		return search
	}
	first := invoke("first")
	if first["executed_this_invocation"] != "8" {
		t.Fatal(first)
	}
	resumed := invoke("resume")
	if resumed["executed_this_invocation"] != "0" || resumed["reused_receipts"] != "8" {
		t.Fatal(resumed)
	}
	p["revision"] = map[string]any{"index": "1", "previous_plan_sha256": first["plan_sha256"]}
	p["workers"] = "1"
	evolved := invoke("evolved")
	if evolved["executed_this_invocation"] != "0" || evolved["reused_receipts"] != "8" {
		t.Fatal(evolved)
	}
	for _, s := range []struct{ op, variant string }{{"experiment", "adaptive"}, {"compare", "selection"}} {
		if _, _, err = SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, s.op, s.variant); err != nil {
			t.Fatal(err)
		}
	}
}

package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledTemporalSplit(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_SPLIT_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed SBV and split fixtures required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(fixtures, "split-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	p["output_path"] = filepath.Join(dir, "split.json")
	invoke := func(op string, input map[string]any) (Response, error) {
		b, e := json.Marshal(input)
		if e != nil {
			t.Fatal(e)
		}
		return InvokeSBV(context.Background(), prefix, "0.17.0-dev", cwd, op, b)
	}
	response, err := invoke("split", p)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := invoke("result_inspect", map[string]any{"protocol": "symphony.sbv.result-inspect-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"]})
	if err != nil {
		t.Fatal(err)
	}
	raw, err = ReadSBVExport(p["output_path"].(string), inspection.Result)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sqavObject(raw, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	sections := artifact["sections"].(map[string]any)
	schema, err := SBVSchema(prefix, "0.17.0-dev", "split")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	for _, pair := range [][2]string{{"observations", "split_observation"}, {"folds", "temporal_fold"}} {
		for _, row := range sections[pair[0]].(map[string]any)["data"].([]any) {
			if !sqvTransportShape(defs[pair[1]], row, 0) {
				t.Fatal("split result schema mismatch", pair)
			}
		}
	}
	query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": "/sections/folds/data/0/train_indices/0", "limit": "1", "cursor": ""}
	response, err = invoke("result_query", query)
	if err != nil {
		t.Fatal(err)
	}
	page, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if page["nodes"].([]any)[0].(map[string]any)["value"] != "0" {
		t.Fatal("wrong terminal train row")
	}
	_, template, err := SBVResource(prefix, "0.17.0-dev", "split", true)
	if err != nil {
		t.Fatal(err)
	}
	var tmpl map[string]any
	if err = json.Unmarshal(template, &tmpl); err != nil {
		t.Fatal(err)
	}
	if tmpl["chronology"] != "past_only" {
		t.Fatal("template policy unavailable")
	}
	for _, key := range []string{"retain_rows", "plan", "available_pointer"} {
		bad := make(map[string]any)
		for k, v := range p {
			bad[k] = v
		}
		bad[key] = 42
		bad["output_path"] = filepath.Join(dir, "invalid-"+key+".json")
		if _, err = invoke("split", bad); err == nil {
			t.Fatal("malformed split accepted", key)
		}
		if _, err = os.Stat(bad["output_path"].(string)); !os.IsNotExist(err) {
			t.Fatal("rejected split wrote artifact")
		}
	}
}

func TestSBVInstalledCombinatorialSplit(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_COMBINATORIAL_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed SBV and combinatorial fixtures required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(fixtures, "combinatorial-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	p["output_path"] = filepath.Join(dir, "folds.json")
	invoke := func(input map[string]any) (Response, error) {
		b, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		return InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, "split", b)
	}
	response, err := invoke(p)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(p["output_path"].(string))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sqavObject(raw, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	if artifact["content_sha256"] != receipt["content_sha256"] {
		t.Fatal("artifact receipt mismatch")
	}
	schema, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "split")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	sections := artifact["sections"].(map[string]any)
	for _, pair := range [][2]string{{"groups", "combinatorial_group"}, {"folds", "temporal_fold"}} {
		for _, row := range sections[pair[0]].(map[string]any)["data"].([]any) {
			if !sqvTransportShape(defs[pair[1]], row, 0) {
				t.Fatal("combinatorial result schema mismatch", pair)
			}
		}
	}
	selection := sections["combination_selection"].(map[string]any)["data"]
	if !sqvTransportShape(defs["combination_selection"], selection, 0) {
		t.Fatal("selection shape")
	}
	if selection.(map[string]any)["total_combinations"] != "15" {
		t.Fatal("wrong fold space")
	}
	// Installed input validation exposes every new plan field.
	for _, change := range []map[string]any{{"fold_offset": 2}, {"fold_count": 3}, {"groups": "invalid"}, {"test_group_count": 2}, {"fit_cutoff_ns": 1}, {"undeclared": true}} {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var bad map[string]any
		if err = json.Unmarshal(b, &bad); err != nil {
			t.Fatal(err)
		}
		plan := bad["plan"].(map[string]any)
		for k, v := range change {
			plan[k] = v
		}
		bad["output_path"] = filepath.Join(dir, "invalid.json")
		if _, err = invoke(bad); err == nil {
			t.Fatal("invalid plan accepted", change)
		}
		if _, err = os.Stat(bad["output_path"].(string)); !os.IsNotExist(err) {
			t.Fatal("rejected request wrote artifact")
		}
	}
}

func TestSBVInstalledFitPredict(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_FIT_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed SBV and fit fixtures required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var fitted map[string]any
	for _, op := range []string{"fit", "predict"} {
		raw, err := os.ReadFile(filepath.Join(fixtures, op+"-request.json"))
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		if err = json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		p["output_path"] = filepath.Join(dir, op+".json")
		if op == "predict" {
			p["model"] = map[string]any{"path": filepath.Join(dir, "fit.json"), "expected_sha256": fitted["content_sha256"], "pointer": "/sections/model/data"}
		}
		invoke := func(input map[string]any) (Response, error) {
			b, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			return InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, op, b)
		}
		response, err := invoke(p)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := sqavObject(response.Result, maxResponseBytes)
		if err != nil {
			t.Fatal(err)
		}
		raw, err = os.ReadFile(p["output_path"].(string))
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := sqavObject(raw, 128<<20)
		if err != nil {
			t.Fatal(err)
		}
		if artifact["content_sha256"] != receipt["content_sha256"] {
			t.Fatal("receipt mismatch")
		}
		schema, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, op)
		if err != nil {
			t.Fatal(err)
		}
		contract, err := sqavObject(schema, maxRequestBytes)
		if err != nil {
			t.Fatal(err)
		}
		defs := contract["$defs"].(map[string]any)
		sections := artifact["sections"].(map[string]any)
		if op == "fit" {
			fitted = artifact
			if !sqvTransportShape(defs["linear_model"], sections["model"].(map[string]any)["data"], 0) {
				t.Fatal("model shape")
			}
		} else {
			for _, row := range sections["predictions"].(map[string]any)["data"].([]any) {
				if !sqvTransportShape(defs["linear_prediction"], row, 0) {
					t.Fatal("prediction shape")
				}
			}
			overlap := sections["training_overlap"].(map[string]any)["data"]
			if !sqvTransportShape(defs["linear_training_overlap"], overlap, 0) || overlap.(map[string]any)["reused_count"] != "3" {
				t.Fatal("overlap evidence")
			}
		}
		_, template, err := SBVResource(prefix, SBVAdministrationInterfaceVersion, op, true)
		if err != nil || len(template) == 0 {
			t.Fatal("missing template", err)
		}
		for _, key := range []string{"source", "features", "target", "weights", "id_pointer"} {
			b, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			var bad map[string]any
			if err = json.Unmarshal(b, &bad); err != nil {
				t.Fatal(err)
			}
			bad[key] = 42
			bad["output_path"] = filepath.Join(dir, "invalid-"+op+"-"+key+".json")
			if _, err = invoke(bad); err == nil {
				t.Fatal("invalid model input accepted", op, key)
			}
			if _, err = os.Stat(bad["output_path"].(string)); !os.IsNotExist(err) {
				t.Fatal("rejected input wrote output")
			}
		}
	}
}

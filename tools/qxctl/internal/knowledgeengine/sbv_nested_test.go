package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSBVNestedSplitShapes(t *testing.T) {
	catalogue := partitionedCatalogue(t, "admin.schema.json")
	templates := partitionedCatalogue(t, "admin.templates.json")
	shape := catalogue["requests"].(map[string]any)["split"].(map[string]any)
	variants := templates["variants"].(map[string]any)["split"].(map[string]any)
	for _, name := range []string{"nested", "nested_partitioned"} {
		t.Run(name, func(t *testing.T) {
			p := recoveryClone(t, variants[name].(map[string]any))
			if !sbvInputShape(shape, p) {
				t.Fatal("nested template rejected")
			}
			inner := p["plan"].(map[string]any)["inner"].([]any)[0].(map[string]any)
			inner["population"] = "source"
			policies := inner["policies"].(map[string]any)
			policies["purge"] = "none"
			policies["availability"] = "ignore"
			policies["chronology"] = "unrestricted"
			if !sbvInputShape(shape, p) {
				t.Fatal("user override rejected")
			}
			policies["automatic_holdout_gate"] = true
			if sbvInputShape(shape, p) {
				t.Fatal("unowned policy admitted")
			}
		})
	}
	p := recoveryClone(t, variants["nested"].(map[string]any))
	inner := p["plan"].(map[string]any)["inner"].([]any)[0].(map[string]any)
	folds := make([]any, 129)
	for i := range folds {
		folds[i] = map[string]any{"id": "id", "train_start_ns": "0", "train_end_ns": "1", "test_start_ns": "1", "test_end_ns": "2", "fit_cutoff_ns": "1"}
	}
	// IDs are native semantic admission; schema must not impose a 128-fold quota.
	inner["plan"] = map[string]any{"kind": "explicit", "folds": folds}
	if !sbvInputShape(shape, p) {
		t.Fatal("schema imposed a fold count policy")
	}
}

func TestSBVInstalledNestedSplit(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_NESTED_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed native and nested fixture required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(fixtures, "nested-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	p["output_path"] = filepath.Join(dir, "nested.json")
	invoke := func(op string, input map[string]any) Response {
		t.Helper()
		b, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		out, err := InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, op, b)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	out := invoke("split", p)
	receipt, err := sqavObject(out.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	inspection := invoke("result_inspect", map[string]any{"protocol": "symphony.sbv.result-inspect-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"]})
	exported, err := ReadSBVExport(p["output_path"].(string), inspection.Result)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sqavObject(exported, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(fixtures, "nested-expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]any
	if err = json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(artifact, expected) {
		t.Fatal("native/qxctl complete results differ")
	}
	schema, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "split")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	foldSchema := contract["$defs"].(map[string]any)["temporal_fold"]
	folds := artifact["sections"].(map[string]any)["folds"].(map[string]any)["data"].([]any)
	for _, f := range folds {
		if !sbvSchemaShape(foldSchema, f, 0) {
			t.Fatal("nested row violates installed definition")
		}
	}
	query := invoke("result_query", map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": "/sections/folds/data/1/nesting/population", "limit": "1", "cursor": ""})
	page, err := sqavObject(query.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if page["nodes"].([]any)[0].(map[string]any)["value"] != "outer_train" {
		t.Fatal("nested policy not queryable")
	}
	for _, v := range []string{"nested", "nested_partitioned"} {
		if _, _, err = SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, "split", v); err != nil {
			t.Fatal(err)
		}
	}
}

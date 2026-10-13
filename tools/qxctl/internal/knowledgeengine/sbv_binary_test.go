package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVBinaryShapes(t *testing.T) {
	c := partitionedCatalogue(t, "admin.schema.json")
	templates := partitionedCatalogue(t, "admin.templates.json")
	variants := templates["variants"].(map[string]any)["analyze"].(map[string]any)
	shape := c["requests"].(map[string]any)["analyze"].(map[string]any)
	for _, name := range []string{"binary", "binary_partitioned"} {
		t.Run(name, func(t *testing.T) {
			p := recoveryClone(t, variants[name].(map[string]any))
			if !sbvInputShape(shape, p) {
				t.Fatal("binary template rejected")
			}
			for _, key := range []string{"label_pointer", "label_type", "positive_label", "negative_label", "unknown_label", "weight_normalization", "thresholds", "bin_edges", "tie_credit", "retain_rows", "label_description"} {
				bad := recoveryClone(t, p)
				delete(bad["binary"].(map[string]any), key)
				if sbvInputShape(shape, bad) {
					t.Fatalf("missing %s admitted", key)
				}
			}
			for _, pair := range [][2]string{{"label_type", "implicit"}, {"unknown_label", "impute"}, {"weight_normalization", "automatic"}} {
				bad := recoveryClone(t, p)
				bad["binary"].(map[string]any)[pair[0]] = pair[1]
				if sbvInputShape(shape, bad) {
					t.Fatal("invalid policy admitted")
				}
			}
		})
	}
}

func TestSBVInstalledBinary(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_BINARY_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("installed native binary fixtures required")
	}
	raw, err := os.ReadFile(filepath.Join(fixtures, "request.json"))
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
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p map[string]any) map[string]any {
		t.Helper()
		b, e := json.Marshal(p)
		if e != nil {
			t.Fatal(e)
		}
		response, e := InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, op, b)
		if e != nil {
			t.Fatal(e)
		}
		v, e := sqavObject(response.Result, maxResponseBytes)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	p["output_path"] = filepath.Join(dir, "analysis.json")
	receipt := invoke("analyze", p)
	if receipt["status"] != "completed" {
		t.Fatal(receipt)
	}
	raw, err = os.ReadFile(p["output_path"].(string))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	schema, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "analyze")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sbvDecodeOwnedResource(schema)
	if err != nil {
		t.Fatal(err)
	}
	shape := contract["$defs"].(map[string]any)["series_study"]
	studies := result["sections"].(map[string]any)["studies"].(map[string]any)["data"].([]any)
	for _, study := range studies {
		if !sbvSchemaShape(shape, study, 0) {
			t.Fatalf("installed binary study contract: %v", study)
		}
	}
	for _, s := range []struct{ pointer, want string }{{"/sections/studies/data/1/data/brier_score/numerator", "1"}, {"/sections/studies/data/1/data/brier_score/denominator", "8"}, {"/sections/studies/data/2/data/average_precision/value/numerator", "9"}, {"/sections/binary_input/data/choices/weight_normalization", "by_total"}} {
		q := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": s.pointer, "limit": "1", "cursor": ""}
		page := invoke("result_query", q)
		if page["nodes"].([]any)[0].(map[string]any)["value"] != s.want {
			t.Fatal("inaccessible binary result", s.pointer)
		}
	}
	for _, variant := range []string{"binary", "binary_partitioned"} {
		if _, _, err = SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, "analyze", variant); err != nil {
			t.Fatal(err)
		}
	}
	// Native admission remains authoritative for cross-field scientific choices.
	p["output_path"] = filepath.Join(dir, "invalid.json")
	p["binary"].(map[string]any)["positive_label"] = p["binary"].(map[string]any)["negative_label"]
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, "analyze", b); err == nil {
		t.Fatal("identical labels accepted")
	}
	if _, err = os.Stat(p["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("invalid output published")
	}
}

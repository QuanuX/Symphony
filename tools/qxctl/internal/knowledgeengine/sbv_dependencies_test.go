package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledDependencies(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_DEPENDENCIES_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed SBV and dependency fixtures required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(fixtures, "experiment-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	journal := filepath.Join(dir, "journal")
	if err = os.Mkdir(journal, 0700); err != nil {
		t.Fatal(err)
	}
	p["directory"] = journal
	schema, err := SBVSchema(prefix, "0.21.0-dev", "experiment")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	for _, run := range []string{"first", "resume"} {
		p["output_path"] = filepath.Join(dir, run+".json")
		payload, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		response, err := InvokeSBV(context.Background(), prefix, "0.21.0-dev", cwd, "experiment", payload)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := sqavObject(response.Result, maxResponseBytes)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(p["output_path"].(string))
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := sqavObject(b, 128<<20)
		if err != nil {
			t.Fatal(err)
		}
		search := artifact["sections"].(map[string]any)["search"].(map[string]any)["data"].(map[string]any)
		want := "5"
		if run == "resume" {
			want = "0"
		}
		if search["executed_this_invocation"] != want || search["wave_count"] != "3" {
			t.Fatal("unexpected graph execution")
		}
		if run == "resume" && search["reused_receipts"] != "5" {
			t.Fatal("completion not reused")
		}
		for _, row := range search["trials"].([]any) {
			if !sqvTransportShape(defs["experiment_trial"], row, 0) {
				t.Fatal("graph ledger schema mismatch")
			}
		}
		query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": "/sections/search/data/trials/0/bindings/1/reference/pointer", "limit": "1", "cursor": ""}
		qb, err := json.Marshal(query)
		if err != nil {
			t.Fatal(err)
		}
		response, err = InvokeSBV(context.Background(), prefix, "0.21.0-dev", cwd, "result_query", qb)
		if err != nil {
			t.Fatal(err)
		}
		page, err := sqavObject(response.Result, maxResponseBytes)
		if err != nil {
			t.Fatal(err)
		}
		if page["nodes"].([]any)[0].(map[string]any)["value"] != "/sections/model/data" {
			t.Fatal("dependency lineage inaccessible")
		}
	}
}

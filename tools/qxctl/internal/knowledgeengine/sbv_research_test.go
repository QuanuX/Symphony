package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledResearch(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_RESEARCH_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed SBV and research fixtures required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p map[string]any) (Response, error) {
		b, e := json.Marshal(p)
		if e != nil {
			t.Fatal(e)
		}
		return InvokeSBV(context.Background(), prefix, "0.10.0-dev", cwd, op, b)
	}
	for _, op := range []string{"resample", "experiment"} {
		t.Run(op, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(fixtures, op+"-request.json"))
			if err != nil {
				t.Fatal(err)
			}
			var p map[string]any
			if err = json.Unmarshal(raw, &p); err != nil {
				t.Fatal(err)
			}
			p["output_path"] = filepath.Join(dir, op+".json")
			if op == "experiment" {
				p["directory"] = filepath.Join(dir, "trial-state")
				if err = os.Mkdir(p["directory"].(string), 0700); err != nil {
					t.Fatal(err)
				}
			}
			response, err := invoke(op, p)
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
			sections := artifact["sections"].(map[string]any)
			schema, err := SBVSchema(prefix, "0.10.0-dev", op)
			if err != nil {
				t.Fatal(err)
			}
			contract, err := sqavObject(schema, maxRequestBytes)
			if err != nil {
				t.Fatal(err)
			}
			defs := contract["$defs"].(map[string]any)
			var rows []any
			var def, pointer, want string
			if op == "resample" {
				rows = sections["resamples"].(map[string]any)["data"].([]any)
				def = "bootstrap_replicate"
				pointer = "/sections/resamples/data/0/replicate"
				want = "0"
			} else {
				rows = sections["search"].(map[string]any)["data"].(map[string]any)["trials"].([]any)
				def = "experiment_trial"
				pointer = "/sections/search/data/executed_this_invocation"
				want = "2"
			}
			for _, row := range rows {
				if !sqvTransportShape(defs[def], row, 0) {
					t.Fatal("new result schema mismatch")
				}
			}
			query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": pointer, "limit": "1", "cursor": ""}
			response, err = invoke("result_query", query)
			if err != nil {
				t.Fatal(err)
			}
			page, err := sqavObject(response.Result, maxResponseBytes)
			if err != nil {
				t.Fatal(err)
			}
			if page["nodes"].([]any)[0].(map[string]any)["value"] != want {
				t.Fatal("research output inaccessible")
			}
			if op == "experiment" {
				p["output_path"] = filepath.Join(dir, "reconciled.json")
				response, err = invoke(op, p)
				if err != nil {
					t.Fatal(err)
				}
				receipt, err = sqavObject(response.Result, maxResponseBytes)
				if err != nil {
					t.Fatal(err)
				}
				query["path"], query["expected_sha256"], query["pointer"] = p["output_path"], receipt["content_sha256"], "/sections/search/data/reused_receipts"
				response, err = invoke("result_query", query)
				if err != nil {
					t.Fatal(err)
				}
				page, err = sqavObject(response.Result, maxResponseBytes)
				if err != nil {
					t.Fatal(err)
				}
				if page["nodes"].([]any)[0].(map[string]any)["value"] != "2" {
					t.Fatal("completed trials not reconciled")
				}
			}
		})
	}
}

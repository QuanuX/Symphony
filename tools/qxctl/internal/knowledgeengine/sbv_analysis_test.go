package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledAnalysis(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_ANALYSIS_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed SBV and native analysis/comparison fixtures required")
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
		return InvokeSBV(context.Background(), prefix, "0.14.0-dev", cwd, op, b)
	}
	for _, op := range []string{"analyze", "compare"} {
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
			response, err := invoke(op, p)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := sqavObject(response.Result, maxResponseBytes)
			if err != nil {
				t.Fatal(err)
			}
			bytes, err := os.ReadFile(p["output_path"].(string))
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := sqavObject(bytes, 128<<20)
			if err != nil {
				t.Fatal(err)
			}
			schema, err := SBVSchema(prefix, "0.14.0-dev", op)
			if err != nil {
				t.Fatal(err)
			}
			contract, err := sqavObject(schema, maxRequestBytes)
			if err != nil {
				t.Fatal(err)
			}
			sections := artifact["sections"].(map[string]any)
			defs := contract["$defs"].(map[string]any)
			if op == "analyze" {
				for _, row := range sections["series"].(map[string]any)["data"].([]any) {
					if !sqvTransportShape(defs["series_observation"], row, 0) {
						t.Fatal("series row contract")
					}
				}
				for _, row := range sections["studies"].(map[string]any)["data"].([]any) {
					if !sqvTransportShape(defs["series_study"], row, 0) {
						t.Fatal("study row contract")
					}
				}
			} else {
				for _, row := range sections["comparisons"].(map[string]any)["data"].(map[string]any)["candidates"].([]any) {
					if !sqvTransportShape(defs["comparison_candidate"], row, 0) {
						t.Fatal("candidate row contract")
					}
				}
			}
			pointer, want := "/sections/studies/data/3/data/downside_second_moment/numerator", "27"
			if op == "compare" {
				pointer, want = "/sections/comparisons/data/weighted_order/0", "a"
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
				t.Fatal("new evidence unreachable")
			}
			inspect := map[string]any{"protocol": "symphony.sbv.result-inspect-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"]}
			response, err = invoke("result_inspect", inspect)
			if err != nil {
				t.Fatal(err)
			}
			exported, err := ReadSBVExport(p["output_path"].(string), response.Result)
			if err != nil {
				t.Fatal(err)
			}
			if string(exported) != string(bytes) {
				t.Fatal("export changed source")
			}
			if _, err = invoke(op, p); err == nil {
				t.Fatal("overwrote immutable result")
			}
		})
	}
}

package knowledgeengine

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledResearchHistory(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_HISTORY_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("exact installed SBV and history fixtures required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(fixtures, "history-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	p["output_path"] = filepath.Join(dir, "history.json")
	payload, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	response, err := InvokeSBV(context.Background(), prefix, "0.18.0-dev", cwd, "research_history", payload)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(p["output_path"].(string))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(fixtures, "history-expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatal("installed research history differs from native fixture")
	}
	artifact, err := sqavObject(actual, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := SBVSchema(prefix, "0.18.0-dev", "research_history")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	sections := artifact["sections"].(map[string]any)
	for section, def := range map[string]string{"research_usage": "research_usage_entry", "observation_usage": "research_observation_usage", "selection_history": "research_selection_record"} {
		for _, row := range sections[section].(map[string]any)["data"].([]any) {
			if !sqvTransportShape(defs[def], row, 0) {
				t.Fatalf("%s schema mismatch", section)
			}
		}
	}
	query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": p["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": "/sections/research_usage/data/1/previously_fitted_observation_count", "limit": "1", "cursor": ""}
	qb, _ := json.Marshal(query)
	response, err = InvokeSBV(context.Background(), prefix, "0.18.0-dev", cwd, "result_query", qb)
	if err != nil {
		t.Fatal(err)
	}
	page, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if page["nodes"].([]any)[0].(map[string]any)["value"] != "1" {
		t.Fatal("research usage inaccessible")
	}
	p["duplicate_artifacts"] = "invented"
	p["output_path"] = filepath.Join(dir, "bad.json")
	payload, _ = json.Marshal(p)
	if _, err = InvokeSBV(context.Background(), prefix, "0.18.0-dev", cwd, "research_history", payload); err == nil {
		t.Fatal("invalid history policy accepted")
	}
	if _, err = os.Stat(p["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("rejected history persisted")
	}
}

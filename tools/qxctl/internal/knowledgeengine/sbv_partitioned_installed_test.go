package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// This opt-in boundary preserves every request, response and workspace under a
// new caller-selected evidence root. It performs no provider or network calls.
func TestSBVInstalledPartitionedReceipts(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SBV_RESOURCE_PREFIX")
	input := os.Getenv("SYMPHONY_SBV_PARTITIONED_SOURCE_REQUEST")
	root := os.Getenv("SYMPHONY_SBV_PARTITIONED_EVIDENCE_ROOT")
	if prefix == "" || input == "" || root == "" {
		t.Skip("select exact installed prefix, existing source request and new evidence root")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		t.Fatal("absolute canonical evidence root required")
	}
	if _, err := InspectSBV(prefix, SBVAdministrationInterfaceVersion); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	source, err := sqavObject(raw, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal("evidence root must be new; no existing evidence is overwritten", err)
	}
	write := func(name string, raw []byte) {
		t.Helper()
		f, err := os.OpenFile(filepath.Join(root, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := f.Write(raw)
		closeErr := f.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatal(writeErr, closeErr)
		}
	}
	template := func(op string) map[string]any {
		t.Helper()
		_, raw, err := SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, op, "partitioned")
		if err != nil {
			t.Fatal(err)
		}
		p, err := sqavObject(raw, maxRequestBytes)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"source_path", "source_sha256", "dataset", "retained_source", "dataset_limits", "memory_budget_bytes"} {
			delete(p, key)
			if value, exists := source[key]; exists {
				p[key] = value
			}
		}
		p["workers"] = "1"
		p["replay"] = map[string]any{"before_ns": "0", "after_ns": "0", "retain_events": false}
		p["extensions"] = map[string]any{"recovery_canonical_probe": "<>&\u2028\u2029\\u2028"}
		return p
	}
	output := func(name string) map[string]any {
		return map[string]any{"kind": "partitioned", "bundle_path": filepath.Join(root, name+"-bundle"), "workspace_path": filepath.Join(root, name+"-workspace"), "write_options": map[string]any{"page_bytes": "1048576", "index_fanout": "64"}}
	}
	invoke := func(name, op string, p map[string]any) map[string]any {
		t.Helper()
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		write(name+"-request.json", raw)
		r, err := InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, root, op, raw)
		write(name+"-response.json", r.Result)
		if err != nil {
			write(name+"-error.txt", []byte(err.Error()))
			t.Fatal(op, err)
		}
		result, err := sqavObject(r.Result, maxResponseBytes)
		if err != nil {
			t.Fatal(err)
		}
		combined, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, op)
		if err != nil {
			t.Fatal(err)
		}
		contract, err := sqavObject(combined, maxRequestBytes)
		if err != nil {
			t.Fatal(err)
		}
		shape, ok := contract["output"].(map[string]any)
		if !ok || !sbvInputShape(shape, result) {
			t.Fatal("native response does not match registered result schema", op)
		}
		return result
	}
	run := template("run")
	run["output"] = output("run")
	run["criteria"] = map[string]any{"rule": "spaced_trades", "spacing_ns": "0", "min_trade_size": "1", "direction": "any", "max_signals": "1"}
	receipt := invoke("01-run", "run", run)
	if receipt["status"] == "recovery_required" || receipt["summary"].(map[string]any)["data"].(map[string]any)["signal_count"] != "1" {
		t.Fatal("selected sample did not produce exactly one signal", receipt)
	}
	ref := receipt["storage"].(map[string]any)["reference"]
	readOptions := map[string]any{"max_page_bytes": nil, "cache_bytes": "8388608"}
	verified := invoke("02-verify", "bundle_verify", map[string]any{"protocol": "symphony.sbv.bundle-verify-input.v1", "reference": ref, "read_options": readOptions, "extensions": map[string]any{}})
	if verified["status"] != "complete" {
		t.Fatal("full bundle closure did not complete")
	}
	evaluate := template("evaluate")
	evaluate["output"] = output("valid-probability")
	evaluate["census"] = map[string]any{"kind": "bundle", "reference": ref, "selector": map[string]any{"kind": "node_id", "node_id": "0"}, "read_options": readOptions}
	ratio := func(n, d string) map[string]any { return map[string]any{"numerator": n, "denominator": d} }
	evaluate["model"] = map[string]any{"protocol": "symphony.sbv.model-selection.v1", "id": "external_outcomes", "version": "1", "horizon_ns": "10", "parameters": map[string]any{
		"producer": map[string]any{"id": "installed-boundary", "version": "1", "artifact_sha256": "", "reproducibility": "uncaptured"}, "measure": "probability", "conditioning": "execution", "calibration_reference": "fixture:uncalibrated",
		"outcomes": []any{map[string]any{"signal_id": "signal-0", "support": []any{map[string]any{"price_nanos": "100", "weight": ratio("1", "1")}}, "execution_probability": map[string]any{"status": "supplied", "value": ratio("1", "2")}, "evidence_reference": "fixture:probability-domain"}},
	}}
	valid := invoke("03-valid-model", "evaluate", evaluate)
	if valid["status"] == "recovery_required" {
		t.Fatal("valid external probability baseline did not complete", valid)
	}
	evaluate["output"] = output("invalid-probability")
	evaluate["model"].(map[string]any)["parameters"].(map[string]any)["outcomes"].([]any)[0].(map[string]any)["support"].([]any)[0].(map[string]any)["weight"] = ratio("2", "1")
	failure := invoke("04-model-domain-recovery", "evaluate", evaluate)
	if failure["status"] != "recovery_required" || failure["cause"].(map[string]any)["category"] != "contract" {
		t.Fatal("invalid probability did not return typed contract recovery", failure)
	}
	recovery := failure["recovery"].(map[string]any)
	if recovery["phase"] != "produce" || recovery["workspace_created"] != true || recovery["final_storage"] != nil {
		t.Fatal("unexpected probability rejection publication state", failure)
	}
	selected := evaluate["output"].(map[string]any)
	if _, err := os.Stat(selected["workspace_path"].(string)); err != nil {
		t.Fatal("caller workspace was not retained", err)
	}
	if _, err := os.Lstat(filepath.Join(selected["bundle_path"].(string), "manifest.json")); !os.IsNotExist(err) {
		t.Fatal("invalid probability published a final result", err)
	}
	write("validation.txt", []byte(fmt.Sprintf("installed_version=%s\nrun_signals=1\nfull_bundle_closure=passed\nexternal_probability_baseline=passed\nexternal_probability_domain_recovery=passed\nevidence_and_workspaces_retained=true\nprovider_calls=0\n", SBVAdministrationInterfaceVersion)))
	t.Logf("native success, full closure and request-bound model-domain recovery preserved at %s", root)
}

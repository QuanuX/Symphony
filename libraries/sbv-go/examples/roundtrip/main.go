// This independent consumer demonstrates a caller-authored model and proposal
// producer. The 1/3 and 2/3 probabilities are synthetic declarations, not fitted
// predictions. Native SBV alone validates and evaluates them.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	sbv "github.com/QuanuX/Symphony/libraries/sbv-go"
)

type M = map[string]any

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(path string) M {
	raw, e := os.ReadFile(path)
	must(e)
	var m M
	must(json.Unmarshal(raw, &m))
	return m
}
func bytes(v any) json.RawMessage { b, e := json.Marshal(v); must(e); return b }
func write(path string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(path, append(b, '\n'), 0600))
}
func rat(n, d string) M { return M{"numerator": n, "denominator": d} }
func main() {
	if len(os.Args) != 4 {
		panic("usage: roundtrip config.json source-result.json fresh-output-directory")
	}
	var config sbv.Config
	raw, e := os.ReadFile(os.Args[1])
	must(e)
	must(json.Unmarshal(raw, &config))
	c, e := sbv.New(config)
	must(e)
	root, e := filepath.Abs(os.Args[3])
	must(e)
	must(os.Mkdir(root, 0700))
	root, e = filepath.EvalSymlinks(root)
	must(e)
	ctx := context.Background()
	sourcePath, e := filepath.Abs(os.Args[2])
	must(e)
	source := read(sourcePath)
	sha := source["content_sha256"].(string)
	sourceRef := M{"path": sourcePath, "expected_sha256": sha, "pointer": ""}
	sections := source["sections"].(map[string]any)
	prov := sections["provenance"].(map[string]any)["data"].(map[string]any)
	data, e := os.ReadFile(prov["source_path"].(string))
	must(e)
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != prov["source_sha256"] {
		panic("retained source hash mismatch")
	}
	data = nil
	call := func(label, op string, p M) M {
		write(filepath.Join(root, label+".request.json"), p)
		raw, e := c.Invoke(ctx, op, bytes(p))
		must(e)
		var v M
		must(json.Unmarshal(raw, &v))
		write(filepath.Join(root, label+".receipt.json"), v)
		return v
	}
	export := M{"protocol": "symphony.sbv.result-inspect-input.v1", "path": sourcePath, "expected_sha256": sha}
	var signalRows []sbv.Row
	done, e := c.Rows(ctx, "result_export", bytes(export), sbv.StreamOptions{}, sbv.Table{ArrayPointer: "/sections/signals/data", Columns: []sbv.Column{{Name: "id", Pointer: "/signal_id", Type: "string"}, {Name: "price", Pointer: "/anchor_price_nanos", Type: "int64"}}}, func(row sbv.Row) error { signalRows = append(signalRows, row); return nil })
	must(e)
	if !done.Complete {
		panic("incomplete source")
	}
	var replay []sbv.Row
	replayDone, e := c.Rows(ctx, "result_export", bytes(export), sbv.StreamOptions{}, sbv.Table{ArrayPointer: "/sections/replay/data/events", Columns: []sbv.Column{{Name: "ordinal", Pointer: "/source_ordinal", Type: "uint64"}, {Name: "receive", Pointer: "/ts_recv", Type: "uint64"}, {Name: "price", Pointer: "/price_nanos", Type: "int64"}}}, func(row sbv.Row) error { replay = append(replay, row); return nil })
	must(e)
	if len(replay) == 0 {
		panic("fixture must retain replay")
	}
	write(filepath.Join(root, "replay-rows.json"), replay)
	// Repackage the same native logical artifact, then consume the scalable
	// node stream with the same typed row contract and owned-copy lifetime.
	sourceBytes, e := os.ReadFile(sourcePath)
	must(e)
	fileHash := sha256.Sum256(sourceBytes)
	imported := call("bundle-import", "bundle_import", M{"protocol": "symphony.sbv.bundle-import-input.v1", "input_path": sourcePath, "expected_file_sha256": hex.EncodeToString(fileHash[:]), "bundle_path": filepath.Join(root, "bundle"), "write_options": M{"page_bytes": "4096", "index_fanout": "8"}, "extensions": M{}})
	bundleRequest := M{"protocol": "symphony.sbv.bundle-export-input.v1", "reference": imported["reference"], "read_options": M{"max_page_bytes": nil, "cache_bytes": "0"}, "output_path": filepath.Join(root, "bundle-stream.ndjson"), "format": "ndjson", "extensions": M{}}
	var bundleRows []sbv.Row
	bundleDone, e := c.Rows(ctx, "bundle_export", bytes(bundleRequest), sbv.StreamOptions{}, sbv.Table{ArrayPointer: "/sections/replay/data/events", Columns: []sbv.Column{{Name: "ordinal", Pointer: "/source_ordinal", Type: "uint64"}, {Name: "receive", Pointer: "/ts_recv", Type: "uint64"}, {Name: "price", Pointer: "/price_nanos", Type: "int64"}}}, func(row sbv.Row) error { bundleRows = append(bundleRows, row); return nil })
	must(e)
	if !reflect.DeepEqual(bundleRows, replay) || bundleDone.ContentSHA256 != done.ContentSHA256 || bundleDone.Nodes != done.Nodes {
		panic("legacy/partitioned stream mismatch")
	}
	bundleRequest["output_path"] = filepath.Join(root, "cancelled-consumer.ndjson")
	stop := errors.New("caller stopped consuming")
	stopped, e := c.Stream(ctx, "bundle_export", bytes(bundleRequest), sbv.StreamOptions{}, func(sbv.Node) error { return stop })
	if !errors.Is(e, stop) || stopped.Complete {
		panic("consumer cancellation did not remain incomplete")
	}
	// All time/column selection is requested from the existing C++ operation.
	var firstTS string
	must(json.Unmarshal(replay[0].Cells[1].Value, &firstTS))
	selection := call("selection", "result_select", M{"protocol": "symphony.sbv.result-select-input.v1", "path": sourcePath, "expected_sha256": sha, "pointer": "/sections/replay/data/events", "filters": []any{M{"pointer": "/ts_recv", "type": "integer", "op": "eq", "value": firstTS, "missing": "reject"}}, "order_by": []any{}, "columns": []any{M{"name": "ordinal", "pointer": "/source_ordinal"}, M{"name": "receive", "pointer": "/ts_recv"}}, "limit": "256", "cursor": ""})
	if selection["protocol"] != "symphony.sbv.result-select.v1" {
		panic("selection response")
	}
	producerBytes, e := os.ReadFile(os.Args[0])
	must(e)
	producerHash := sha256.Sum256(producerBytes)
	producer := M{"id": "example.external.synthetic-model", "version": "1", "artifact_sha256": hex.EncodeToString(producerHash[:]), "reproducibility": "deterministic_declared"}
	resource, e := c.Resource(ctx, "evaluate", "template", "")
	must(e)
	var template M
	must(json.Unmarshal(resource, &template))
	request := template["template"].(map[string]any)
	request["source_path"] = prov["source_path"]
	request["source_sha256"] = prov["source_sha256"]
	request["dataset"] = prov["dataset"]
	request["census"] = sourceRef
	request["replay"] = M{"before_ns": "0", "after_ns": "1000", "retain_events": true}
	request["studies"] = []any{"signal_summary", "model_summary"}
	request["workers"] = "2"
	journal := filepath.Join(root, "journal")
	must(os.Mkdir(journal, 0700))
	var trials []any
	for _, weight := range []string{"1", "2"} {
		outcomes := []any{}
		for _, row := range signalRows {
			var id, price string
			must(json.Unmarshal(row.Cells[0].Value, &id))
			must(json.Unmarshal(row.Cells[1].Value, &price))
			outcomes = append(outcomes, M{"signal_id": id, "support": []any{M{"price_nanos": price, "weight": rat("1", "1")}}, "execution_probability": M{"status": "supplied", "value": rat(weight, "3")}, "evidence_reference": "example://synthetic-external-model"})
		}
		request["model"] = M{"protocol": "symphony.sbv.model-selection.v1", "id": "external_outcomes", "version": "1", "horizon_ns": "1000", "parameters": M{"producer": producer, "measure": "probability", "conditioning": "execution", "calibration_reference": "example://not-calibrated", "outcomes": outcomes}}
		delete(request, "output_path")
		var child M
		must(json.Unmarshal(bytes(request), &child))
		trials = append(trials, M{"id": "proposal-" + weight, "state": "ready", "operation": "evaluate", "request": child, "reason": "", "parameters": M{"synthetic_probability": rat(weight, "3")}, "lineage": M{"producer": producer, "source_content_sha256": sha, "purpose": "synthetic external model round-trip; not calibrated"}, "depends_on": []any{}, "bindings": []any{}})
	}
	plan := M{"protocol": "symphony.sbv.experiment-input.v1", "output_path": filepath.Join(root, "experiment.json"), "directory": journal, "experiment_id": "external-proposals", "trials": trials, "workers": "2", "on_failure": "continue", "extensions": M{"proposal_source": "independent-go-consumer"}}
	call("experiment", "experiment", plan)
	plan["output_path"] = filepath.Join(root, "resume.json")
	call("resume", "experiment", plan)
	resume := read(plan["output_path"].(string))
	search := resume["sections"].(map[string]any)["search"].(map[string]any)["data"].(map[string]any)
	if search["executed_this_invocation"] != "0" || search["reused_receipts"] != "2" {
		panic("unexpected resume")
	}
	var candidates []any
	for i := 1; i <= 2; i++ {
		result := read(filepath.Join(journal, "proposal-"+strconv.Itoa(i)+".result.json"))
		s := result["sections"].(map[string]any)
		if !reflect.DeepEqual(s["signals"], sections["signals"]) {
			panic("census changed")
		}
		for _, r := range s["execution"].(map[string]any)["data"].([]any) {
			row := r.(map[string]any)
			if row["calibration"] != "not_verified" || !reflect.DeepEqual(row["fill_probability"].(map[string]any)["value"], rat(strconv.Itoa(i), "3")) {
				panic("external probability altered")
			}
		}
		candidates = append(candidates, M{"id": "proposal-" + strconv.Itoa(i), "state": "completed", "reason": "", "path": filepath.Join(journal, "proposal-"+strconv.Itoa(i)+".result.json"), "expected_sha256": result["content_sha256"], "parameters": M{"synthetic_probability": rat(strconv.Itoa(i), "3")}, "lineage": M{"producer": producer, "source_content_sha256": sha}})
	}
	comparisonPath := filepath.Join(root, "comparison.json")
	call("compare", "compare", M{"protocol": "symphony.sbv.compare-input.v1", "output_path": comparisonPath, "candidates": candidates, "objectives": []any{M{"id": "synthetic_probability", "pointer": "/sections/execution/data/0/fill_probability/value", "type": "rational", "direction": "maximize", "weight": rat("1", "1"), "scale": rat("1", "1"), "unit": "declared_probability"}}, "methods": []any{"weighted_sum", "pareto"}, "missing": "exclude_candidate", "comparison_description": "Synthetic round-trip objective only; larger supplied probability is not evidence of a better market model", "extensions": M{}})
	comparison := read(comparisonPath)
	ranking := comparison["sections"].(map[string]any)["comparisons"].(map[string]any)["data"].(map[string]any)
	selected := ""
	for _, r := range ranking["candidates"].([]any) {
		row := r.(map[string]any)
		if row["weighted_rank"] == "1" {
			selected = row["id"].(string)
		}
	}
	if selected != "proposal-2" {
		panic("external selection feedback mismatch")
	}
	write(filepath.Join(root, "report.json"), M{"status": "passed", "source": sourceRef, "source_stream": done, "replay_stream": replayDone, "bundle_stream": bundleDone, "consumer_cancellation": "incomplete_and_reaped", "signals": len(signalRows), "replay_rows": len(replay), "external_trials": "2", "resume_executed": "0", "selected_proposal": selected, "comparison_content_sha256": comparison["content_sha256"], "calibration": "not_verified", "provider_requests": "0", "spend_usd": "0"})
	fmt.Println("external model/proposal round-trip passed")
}

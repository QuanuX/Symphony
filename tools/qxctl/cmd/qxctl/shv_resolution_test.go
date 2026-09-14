package main

import (
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"strings"
	"testing"
)

func TestSHVResolutionRouting(t *testing.T) {
	h := "sha256:" + strings.Repeat("1", 64)
	input := json.RawMessage(`{"dependencies":{"source_revision_digest":"` + h + `","source_engine":{"engine_id":"symphony-shv-source","version":"0.1.0-dev","executable_digest":"` + h + `"},"kernel_engine":{"engine_id":"symphony-shv","version":"0.2.0-dev","executable_digest":"` + h + `"},"captures":[{"capture_id":"body","capture_digest":"` + h + `","content_digest":"` + h + `","bytes":3}],"mapping_digest":"` + h + `","catalogue_digest":"` + h + `"},"subject_ids":["a"]}`)
	part, e := knowledgeengine.ExpectedSHVPartition("partition_build", input)
	if e != nil {
		t.Fatal(e)
	}
	var p map[string]json.RawMessage
	_ = json.Unmarshal(part, &p)
	manifest := func(entries []any) json.RawMessage {
		t.Helper()
		m, e := knowledgeengine.ExpectedSHVPartition("manifest_build", jobMarshal(map[string]any{"entries": entries, "required_references": []any{map[string]any{"partition_digest": p["digest"], "subject_id": "a"}}}))
		if e != nil {
			t.Fatal(e)
		}
		return m
	}
	missing := manifest([]any{map[string]any{"partition_digest": p["digest"], "partition": nil}})
	t.Run("fill_only_declared", func(t *testing.T) {
		plan, e := prepareSHVResolution(jobMarshal(map[string]any{"manifest": missing, "candidates": []any{part}}))
		if e != nil || len(plan.resolved) != 1 || len(plan.unused) != 0 {
			t.Fatal(plan, e)
		}
		result, e := knowledgeengine.ExpectedSHVPartition("manifest_build", plan.resultInput)
		if e != nil {
			t.Fatal(e)
		}
		if !refreshEqual(result, manifest([]any{map[string]any{"partition_digest": p["digest"], "partition": part}})) {
			t.Fatal("result differs")
		}
	})
	t.Run("unlisted_stays_unlisted", func(t *testing.T) {
		m := manifest([]any{})
		plan, e := prepareSHVResolution(jobMarshal(map[string]any{"manifest": m, "candidates": []any{part}}))
		if e != nil || len(plan.resolved) != 0 || len(plan.unused) != 1 {
			t.Fatal(plan, e)
		}
		result, e := knowledgeengine.ExpectedSHVPartition("manifest_build", plan.resultInput)
		if e != nil || !refreshEqual(result, m) {
			t.Fatal(e, string(result))
		}
	})
	t.Run("reject_invalid_inventory", func(t *testing.T) {
		for _, c := range []any{nil, []any{part, part}, []any{map[string]any{"digest": h}}} {
			if _, e := prepareSHVResolution(jobMarshal(map[string]any{"manifest": missing, "candidates": c})); e == nil {
				t.Fatal("invalid candidate inventory accepted")
			}
		}
	})
}
func TestSHVResolutionErrors(t *testing.T) {
	out, status := invokeCLI(t, "shv", "partition", "resolution", "run", "--json")
	var d map[string]any
	if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
		t.Fatal(status, out)
	}
	for _, raw := range []string{`null`, `{}`, `{"manifest":{},"candidates":[],"extra":1}`} {
		if _, e := prepareSHVResolution(json.RawMessage(raw)); e == nil {
			t.Fatal("invalid resolution accepted")
		}
	}
}

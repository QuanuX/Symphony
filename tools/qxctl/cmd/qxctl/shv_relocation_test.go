package main

import (
	"encoding/json"
	"testing"
)

func TestSHVRelocationAdmission(t *testing.T) {
	for _, s := range []string{"relative", "/", "/tmp/../other", "/tmp/"} {
		if relocationRoot(s) {
			t.Fatal("unsafe root accepted", s)
		}
	}
	if !relocationRoot("/private/tmp/data") {
		t.Fatal("absolute root rejected")
	}
	for _, raw := range []string{`null`, `{}`, `{"parent_job_root":"relative","parent_job_id":"j","parent_checkpoint_digest":"bad","source_roots":[]}`} {
		if validateJobOrigin(json.RawMessage(raw), nil) == nil {
			t.Fatal("invalid lineage accepted")
		}
	}
	for _, raw := range []string{`{"protocol":"symphony.qxctl.shv-materialization.v2"}`, `null`} {
		if _, _, e := jobValidate(json.RawMessage(raw), "child"); e == nil {
			t.Fatal("invalid derived job accepted")
		}
	}
}
func TestSHVRelocationCommandErrors(t *testing.T) {
	out, status := invokeCLI(t, "shv", "materialization", "relocation", "run", "--json")
	var d map[string]any
	if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
		t.Fatal(status, out)
	}
}

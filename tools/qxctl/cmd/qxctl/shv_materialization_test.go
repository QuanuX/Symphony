package main

import (
	"encoding/json"
	"testing"
)

func TestSHVMaterializationErrors(t *testing.T) {
	for _, op := range []string{"prepare", "run", "resume", "status", "export"} {
		t.Run(op, func(t *testing.T) {
			out, status := invokeCLI(t, "shv", "materialization", op, "--json")
			var d map[string]any
			if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
				t.Fatal(status, out)
			}
		})
	}
}
func TestSHVMaterializationRejectsBadState(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `{"protocol":"symphony.qxctl.shv-materialization.v1","job_id":"j","installation":{},"tasks":[],"required_references":[],"digest":"bad"}`} {
		if _, _, e := jobValidate(json.RawMessage(raw), "j"); e == nil {
			t.Fatal("bad job accepted")
		}
	}
}

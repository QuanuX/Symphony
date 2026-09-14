package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSHVRefreshStrictInput(t *testing.T) {
	for name, raw := range map[string]string{"duplicate": `{"a":1,"a":2}`, "unicode": `{"a":"\ud800"}`, "extra": `{"a":1,"b":2}`, "missing": `{}`, "null": `null`, "trailing": `{"a":1} {}`} {
		t.Run(name, func(t *testing.T) {
			if _, e := refreshObject(json.RawMessage(raw), "a"); e == nil {
				t.Fatal("malformed input accepted")
			}
		})
	}
	if _, e := refreshObject(json.RawMessage(`{"a":1}`), "a"); e != nil {
		t.Fatal(e)
	}
}
func TestSHVRefreshMachineBoundary(t *testing.T) {
	for _, op := range []string{"build", "verify", "schema", "template"} {
		t.Run(op, func(t *testing.T) {
			out, status := invokeCLI(t, "shv", "refresh", op, "--json")
			var d map[string]any
			if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
				t.Fatal(status, out)
			}
		})
	}
	out, status := invokeCLI(t, "shv", "refresh", "build", "--source-root", "--json")
	if status == 0 || strings.Contains(out, `"protocol":"`) {
		t.Fatal("flag value incorrectly treated as output intent", status, out)
	}
}
func TestSHVRefreshReplayBudget(t *testing.T) {
	// Pretty output can exceed the read bound even when canonical bytes fit.
	raw := json.RawMessage(`{"rows":[` + strings.Repeat(`{},`, 190000) + `{}]}`)
	if len(raw) > 1<<20 {
		t.Fatal("fixture must fit compact input")
	}
	if e := refreshReplayBound(raw); e == nil {
		t.Fatal("unreadable pretty output accepted")
	}
	if e := refreshReplayBound(json.RawMessage(`{"rows":[]}`)); e != nil {
		t.Fatal(e)
	}
}

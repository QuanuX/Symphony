package main

import (
	"encoding/json"
	"testing"
)

func TestSHVInventoryAdmission(t *testing.T) {
	row := map[string]any{"id": "item", "subject_id": "subject", "manufacturer": "Caller", "model": "M", "hardware_class": "custom", "locator": nil, "evidence_id": nil}
	for _, input := range []any{nil, map[string]any{}, map[string]any{"profile": nil, "roster": nil, "evidence": []any{}}, map[string]any{"profile": nil, "roster": []any{row, row}, "evidence": []any{}}} {
		if _, _, _, e := inventoryRequest(jobMarshal(input)); e == nil {
			t.Fatal("invalid inventory admitted")
		}
	}
	if _, r, _, e := inventoryRequest(jobMarshal(map[string]any{"profile": nil, "roster": []any{row}, "evidence": []any{}})); e != nil || len(r) != 1 {
		t.Fatal(e)
	}
	row["evidence_id"] = "missing"
	if _, _, _, e := inventoryRequest(jobMarshal(map[string]any{"profile": nil, "roster": []any{row}, "evidence": []any{}})); e == nil {
		t.Fatal("unlisted evidence")
	}
}
func TestSHVInventoryStageChanges(t *testing.T) {
	before := jobMarshal(map[string]any{"rows": []any{map[string]any{"id": "removed", "stage": "expected"}, map[string]any{"id": "same", "stage": "discovered"}}})
	after := jobMarshal(map[string]any{"rows": []any{map[string]any{"id": "added", "stage": "interpreted"}, map[string]any{"id": "same", "stage": "discovered"}}})
	changes, e := inventoryChanges(before, after)
	if e != nil || len(changes) != 2 {
		t.Fatal(changes, e)
	}
}
func TestSHVInventoryErrors(t *testing.T) {
	for _, op := range []string{"run", "compare"} {
		out, status := invokeCLI(t, "shv", "inventory", op, "--json")
		var d map[string]any
		if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
			t.Fatal(status, out)
		}
	}
}

func TestSHVInventoryJSONIntent(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"shv", "inventory", "run", "--input", "--json"},
		{"shv", "inventory", "compare", "--json=false"},
	} {
		if scvJSONRequested(root, args) {
			t.Fatal("JSON intent inferred from a value", args)
		}
	}
	if !scvJSONRequested(root, []string{"shv", "inventory", "run", "--json"}) {
		t.Fatal("explicit JSON intent lost")
	}
}

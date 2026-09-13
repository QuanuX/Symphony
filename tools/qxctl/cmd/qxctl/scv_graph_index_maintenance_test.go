package main

import (
	"encoding/json"
	"errors"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"testing"
)

func maintenanceCLIModel(t *testing.T) (*graphIndexRunner, map[string]any, graphIndexTargetObserver, *int) {
	t.Helper()
	raw, err := os.ReadFile("../../../../modules/scv-graph-duckdb-connector/tests/fixtures/maintenance-wire.json")
	if err != nil {
		t.Fatal(err)
	}
	all := graphIndexCLIMap(t, raw)
	p := all["plan"].(map[string]any)
	in := p["input"].(map[string]any)
	var source, target, owner knowledgeengine.Installation
	decode := func(v any, dst any) {
		if err := json.Unmarshal(graphIndexCLIRaw(t, v), dst); err != nil {
			t.Fatal(err)
		}
	}
	decode(in["source_connector"], &source)
	decode(in["target_connector"], &target)
	plan := p["result"].(map[string]any)
	first := plan["selected"].([]any)[0].(map[string]any)["source"].(map[string]any)["intent"].(map[string]any)["snapshot"].(map[string]any)
	decode(first["owner"], &owner)
	r := &graphIndexRunner{connector: source, options: graphIndexOptions{namespace: in["namespace"].(string), scv: scvOptions{topsID: in["tops_id"].(string)}}}
	calls := 0
	r.inspectOwner = func(string, string, string) (knowledgeengine.Installation, error) { return owner, nil }
	// Semantic transport is modeled here; the installed acceptance invokes the
	// actual owner and validates its full native result schema.
	r.owner = func(_ knowledgeengine.Installation, _ json.RawMessage, _ string) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{"modeled_owner":true}`), nil
	}
	r.invoke = func(op string, payload map[string]any) (json.RawMessage, error) {
		name := "plan"
		if op == "inventory" {
			name = "inventory"
		}
		if op != "inventory" && op != "transfer_plan" {
			t.Fatal("planning invoked mutation", op)
		}
		copy := graphIndexCLIMap(t, raw)[name].(map[string]any)["result"].(map[string]any)
		copy["input"] = payload
		return graphIndexCLIRaw(t, graphIndexCLISeal(t, copy)), nil
	}
	input := map[string]any{"expected_revision": in["expected_revision"], "operation_ids": in["operation_ids"], "target": map[string]any{"prefix": target.Prefix, "version": target.Version, "root": in["target_root"]}, "capacity": in["capacity"]}
	observe := func(map[string]any) (map[string]any, error) {
		return map[string]any{"installation": target, "root": in["target_root"], "state": "empty"}, nil
	}
	return r, input, observe, &calls
}
func TestSCVGraphIndexPlanOwnerAndTargetBlockers(t *testing.T) {
	r, input, observe, calls := maintenanceCLIModel(t)
	raw, err := runGraphIndexMaintenance(r, "transfer-plan", input, observe)
	if err != nil {
		t.Fatal(err)
	}
	result := graphIndexCLIMap(t, raw)
	expected := graphIndexCLISeal(t, graphIndexCLIMap(t, raw))
	if result["digest"] != expected["digest"] {
		t.Fatal("maintenance wrapper seal is not canonical")
	}
	if result["disposition"] != "ready" || *calls != 2 {
		t.Fatal("missing semantic replay", result["disposition"], *calls)
	}
	r.inspectOwner = func(string, string, string) (knowledgeengine.Installation, error) {
		return knowledgeengine.Installation{}, errors.New("missing original owner")
	}
	raw, err = runGraphIndexMaintenance(r, "transfer-plan", input, observe)
	if err != nil {
		t.Fatal(err)
	}
	result = graphIndexCLIMap(t, raw)
	if result["disposition"] != "blocked" || len(result["blockers"].([]any)) != 2 {
		t.Fatal("missing owner not explicit")
	}
	r, input, observe, _ = maintenanceCLIModel(t)
	notEmpty := func(v map[string]any) (map[string]any, error) {
		m, e := observe(v)
		m["state"] = "not_empty"
		return m, e
	}
	raw, err = runGraphIndexMaintenance(r, "transfer-plan", input, notEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if graphIndexCLIMap(t, raw)["disposition"] != "blocked" {
		t.Fatal("nonempty target accepted")
	}
}
func TestSCVGraphIndexPlanReobservesSourceAndTarget(t *testing.T) {
	r, input, observe, _ := maintenanceCLIModel(t)
	invoke := r.invoke
	r.invoke = func(op string, p map[string]any) (json.RawMessage, error) {
		if op == "inventory" {
			return nil, errors.New("revision changed")
		}
		return invoke(op, p)
	}
	if _, err := runGraphIndexMaintenance(r, "transfer-plan", input, observe); err == nil {
		t.Fatal("stale source accepted")
	}
	r, input, observe, _ = maintenanceCLIModel(t)
	count := 0
	changed := func(v map[string]any) (map[string]any, error) {
		m, e := observe(v)
		count++
		if count > 1 {
			m["state"] = "not_empty"
		}
		return m, e
	}
	if _, err := runGraphIndexMaintenance(r, "transfer-plan", input, changed); err == nil {
		t.Fatal("changed target accepted")
	}
}
func TestSCVGraphIndexInventoryDoesNotRequireSemanticOwner(t *testing.T) {
	r, _, _, calls := maintenanceCLIModel(t)
	r.inspectOwner = func(string, string, string) (knowledgeengine.Installation, error) {
		t.Fatal("inventory inspected semantic owner")
		return knowledgeengine.Installation{}, nil
	}
	input := map[string]any{"expected_revision": nil, "cursor": nil, "limit": json.Number("1")}
	raw, err := runGraphIndexMaintenance(r, "inventory", input, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := graphIndexCLIMap(t, raw)
	if result["disposition"] != "observed" || *calls != 0 {
		t.Fatal("inventory overstates semantics")
	}
	r.connector.Version = "0.1.0-dev"
	if _, err = runGraphIndexMaintenance(r, "inventory", input, nil); err == nil {
		t.Fatal("legacy release silently upgraded")
	}
}

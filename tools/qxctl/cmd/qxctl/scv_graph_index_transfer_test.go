package main

import (
	"encoding/json"
	"errors"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvtransfer"
	"os"
	"path/filepath"
	"testing"
)

func transferModel(t *testing.T) *transferRunner {
	t.Helper()
	source, _, _, _ := maintenanceCLIModel(t)
	raw, err := source.invoke("transfer_plan", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	// Read the original mechanical fixture, preserving its validated input.
	raw, err = os.ReadFile("../../../../modules/scv-graph-duckdb-connector/tests/fixtures/maintenance-wire.json")
	if err != nil {
		t.Fatal(err)
	}
	all := graphIndexCLIMap(t, raw)
	plan := all["plan"].(map[string]any)["result"].(map[string]any)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	os.Chmod(root, 0o700)
	pin := plan["input"].(map[string]any)
	pin["target_root"] = root
	plan = graphIndexCLISeal(t, plan)
	source.options.root = "/source"
	source.invoke = func(op string, input map[string]any) (json.RawMessage, error) {
		if op != "transfer_plan" {
			t.Fatal(op)
		}
		return graphIndexCLIRaw(t, plan), nil
	}
	intent := scvtransfer.Intent{Protocol: scvtransfer.Protocol, SourceRoot: "/source", TargetRoot: root, Plan: graphIndexCLIRaw(t, plan)}
	raw, err = scvtransfer.Seal(intent)
	if err != nil {
		t.Fatal(err)
	}
	intent, err = scvtransfer.ReadIntent(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &transferRunner{source: source, intent: intent, plan: plan, hook: func(string) error { return nil }}
}
func TestSCVGraphIndexTransferTargetIdentity(t *testing.T) {
	r := transferModel(t)
	for _, v := range r.plan["selected"].([]any) {
		item := v.(map[string]any)
		got, err := r.targetRecord(item)
		if err != nil {
			t.Fatal(err)
		}
		source := item["source"].(map[string]any)
		intent := got["intent"].(map[string]any)
		if intent["digest"] != item["target_intent_digest"] || got["state"] != source["state"] {
			t.Fatal("lineage changed")
		}
		for _, key := range []string{"operation_id", "validation_query_time"} {
			if intent[key] != source["intent"].(map[string]any)[key] {
				t.Fatal(key)
			}
		}
		raw := graphIndexCLIRaw(t, got)
		in := map[string]any{"tops_id": r.source.options.scv.topsID, "namespace": r.source.options.namespace, "operation_id": intent["operation_id"]}
		if err = knowledgeengine.ValidateSCVGraphIndexResult("status", graphIndexCLIRaw(t, in), raw); err != nil {
			t.Fatal(err)
		}
	}
}
func TestSCVGraphIndexTransferPreflightNoReservation(t *testing.T) {
	for _, which := range []string{"source", "owner"} {
		t.Run(which, func(t *testing.T) {
			r := transferModel(t)
			if which == "source" {
				r.source.invoke = func(string, map[string]any) (json.RawMessage, error) { return nil, errors.New("stale source") }
			} else {
				r.source.inspectOwner = func(string, string, string) (knowledgeengine.Installation, error) {
					return knowledgeengine.Installation{}, errors.New("owner missing")
				}
			}
			if _, err := r.execute(true); err == nil {
				t.Fatal("preflight accepted")
			}
			files, err := os.ReadDir(r.intent.TargetRoot)
			if err != nil || len(files) != 0 {
				t.Fatal("preflight mutated target", err)
			}
		})
	}
}
func TestSCVGraphIndexTransferOutputIdentity(t *testing.T) {
	r := transferModel(t)
	j := &scvtransfer.Journal{Intent: r.intent}
	var raw json.RawMessage
	if err := transferOutput(j, nil, errors.New("interrupted"), true, &raw); err != nil {
		t.Fatal(err)
	}
	result := graphIndexCLIMap(t, raw)
	if result["digest"] != graphIndexCLISeal(t, result)["digest"] || result["status"] != "incomplete" {
		t.Fatal("output seal/status differs")
	}
	if err := transferOutput(j, nil, nil, false, &raw); err != nil {
		t.Fatal(err)
	}
	if graphIndexCLIMap(t, raw)["status"] != "recorded_incomplete" {
		t.Fatal("journal observation claims current execution")
	}
}
func TestSCVGraphIndexTransferRootSeparation(t *testing.T) {
	for _, child := range []string{"/root/child", "/root/.hidden"} {
		if !withinRoot("/root", child) {
			t.Fatal(child)
		}
	}
	for _, child := range []string{"/root-peer", "/other"} {
		if withinRoot("/root", child) {
			t.Fatal(child)
		}
	}
}

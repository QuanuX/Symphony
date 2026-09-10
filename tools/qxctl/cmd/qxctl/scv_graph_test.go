package main

import (
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvgraph"
	"os"
	"path/filepath"
	"testing"
)

func projectionOptions(t *testing.T) (scvOptions, string) {
	t.Helper()
	prefix := os.Getenv("SYMPHONY_SCV_ACCEPTANCE_PREFIX")
	if prefix == "" {
		t.Skip("requires receipt-owned installed SCV acceptance prefix")
	}
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return scvOptions{domain: "scv", prefix: prefix, version: "0.1.0-dev", repository: directory, stateRoot: directory,
		topsID: "00000000-0000-4000-8000-000000000099", operationID: "projection-test", jsonOutput: true}, directory
}
func projectionInput(t *testing.T, options *scvOptions, value any) {
	t.Helper()
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	options.input = filepath.Join(options.stateRoot, "projection-input.json")
	if err := os.WriteFile(options.input, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}
func TestSCVProjectionRejectsForgedGraphBeforeIntent(t *testing.T) {
	options, _ := projectionOptions(t)
	// Self-consistent digest is insufficient: the real C++ graph owner must
	// reject absent capture/claim/interpretation structure before persistence.
	forged := map[string]any{"protocol": "symphony.scv.graph.v1", "domain": "scv"}
	digest, err := knowledgeengine.SCVDigest(forged)
	if err != nil {
		t.Fatal(err)
	}
	forged["digest"] = digest
	projectionInput(t, &options, map[string]any{"graph": forged, "expected_graph_digest": nil, "expected_generation": 0})
	if err := runSCVProjection("select", options, "research"); err == nil {
		t.Fatal("forged graph accepted")
	}
	store, err := scvgraph.New(options.stateRoot, options.topsID, options.domain, "research")
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Operations) != 0 || state.GraphDigest != nil {
		t.Fatal("invalid owner result reached mutation intent")
	}
}
func TestSCVProjectionRequiresAuditedAuthorization(t *testing.T) {
	options, directory := projectionOptions(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(directory, "isolated-config"))
	t.Setenv("XDG_STATE_HOME", directory)
	t.Setenv("SYMPHONY_SSIAG_SOCKET", "")
	desired := map[string]any{"source_id": "fixture-docs", "provider_id": "fixture", "family_id": "schv", "publisher": "Fixture",
		"authority_role": "user_declared", "scope": "Synthetic fixture", "continuity_evidence": []string{"fixture"},
		"locators": []any{map[string]any{"locator_id": "docs", "uri": "https://example.invalid/fixture.md", "role": "preferred", "format": "markdown", "selector": "fixture-v1"}}}
	plan, err := invokeSCV(options, "source_plan", map[string]any{"operation_id": "fixture", "current": nil, "desired": desired, "reason": "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Source json.RawMessage `json:"source"`
	}
	if err := json.Unmarshal(plan, &p); err != nil {
		t.Fatal(err)
	}
	capture, err := invokeSCV(options, "capture_import", map[string]any{"source": p.Source, "locator_id": "docs", "resolved_uri": "https://example.invalid/fixture.md",
		"redirects": []string{}, "observed_at": "2026-09-10T12:00:00Z", "upstream_revision": nil, "media_type": "text/markdown", "body": "# Fixture\n",
		"completeness": "complete", "issues": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	knowledge, err := invokeSCV(options, "knowledge_interpret", map[string]any{"captures": []json.RawMessage{capture}, "claims": []any{}, "interpreter_version": "fixture-v1",
		"selection_policy": map[string]any{"policy_id": "fixture", "max_age_seconds": nil, "allowed_statement_kinds": []string{"documented_fact"}, "partial_capture": "exclude"}})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := invokeSCV(options, "graph_build", map[string]any{"knowledge": []json.RawMessage{knowledge}})
	if err != nil {
		t.Fatal(err)
	}
	projectionInput(t, &options, map[string]any{"graph": graph, "expected_graph_digest": nil, "expected_generation": 0})
	if err := runSCVProjection("select", options, "research"); err == nil {
		t.Fatal("graph selected without trusted audited authority")
	} else {
		t.Logf("expected authorization failure: %v", err)
	}
	store, err := scvgraph.New(options.stateRoot, options.topsID, options.domain, "research")
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if state.GraphDigest != nil || state.Generation != 0 || state.Operations[options.operationID].Status != "prepared" {
		t.Fatalf("unavailable authority did not preserve pending exact intent and absent graph: %#v", state)
	}
}

package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
)

func TestInstalledSCVCompositionArtifactsRetainExactOwnerAndRejectEarlierAdmission(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.6.0-dev SCV owners")
	}
	store := workflowStore(t)
	runner, err := newWorkflowRunner(scvOptions{domain: "scv", prefix: prefix, version: "0.6.0-dev", repository: store.Root}, true)
	if err != nil {
		t.Fatal(err)
	}
	retained := map[string]map[string]any{}
	inputs := map[string]map[string]any{}
	importArtifact := func(operation string, input map[string]any) map[string]any {
		t.Helper()
		raw, err := runner.artifact(store, "import", map[string]any{"operation": operation, "input": input, "result": nil})
		if err != nil {
			t.Fatal(operation, err)
		}
		record := workflowValue(t, raw)
		shown, err := runner.artifact(store, "show", map[string]any{"record_digest": record["digest"]})
		if err != nil || !scvworkflow.Same(raw, shown) {
			t.Fatal("exact owner replay", operation, err)
		}
		retained[operation] = record
		inputs[operation] = input
		return record["artifact"].(map[string]any)
	}
	policy := map[string]any{"policy_id": "caller-fixture", "max_age_seconds": nil, "partial_capture": "exclude", "allowed_statement_kinds": []any{"documented_fact"}}
	fixture := map[string]any{"fixture_id": "empty-selection", "captures": []any{}, "bindings": []any{}, "selection_policy": policy}
	draft := map[string]any{"protocol": "symphony.scv.provider-pack.v1", "pack_id": "empty-portable-provider", "pack_version": "fixture-1", "authored_by": "Synthetic fixture author", "provenance": []any{"No provider facts or conformance asserted"},
		"provider": map[string]any{"provider_id": "fixture-independent", "family_id": "schv", "display_name": "Synthetic unlisted provider", "sources": []any{map[string]any{"source_id": "fixture-docs", "provider_id": "fixture-independent", "family_id": "schv", "publisher": "Synthetic fixture", "authority_role": "user_declared", "scope": "Empty mapping test source", "locators": []any{map[string]any{"locator_id": "docs", "uri": "https://example.invalid/docs", "role": "reference", "format": "json", "selector": "fixture-1"}}, "continuity_evidence": []any{}}}}, "profiles": []any{}, "structured_profiles": []any{}, "fixtures": []any{map[string]any{"fixture_id": "empty-selection", "label": "No facts from empty selection", "authored_by": "Fixture expectation author", "rationale": "Zero selected captures must create zero claims", "input_digest": nil, "expected_claims": []any{}, "expected_extractions": []any{}}}}
	pack := importArtifact("provider_pack_prepare", map[string]any{"pack": draft, "fixtures": []any{fixture}})
	evaluated := importArtifact("provider_pack_evaluate", map[string]any{"pack": pack, "captures": []any{}, "bindings": []any{}, "selection_policy": policy, "fixtures": []any{fixture}})
	resolution := map[string]any{"kind": "caller_decision", "reference": "fixture:missing-proof", "description": "Caller supplies missing evidence"}
	requirement := map[string]any{"requirement_id": "capacity", "importance": "required", "operator": "gte", "right": map[string]any{"kind": "literal", "value": map[string]any{"type": "integer", "value": 1, "unit": "units"}}, "resolution": resolution}
	recipe := map[string]any{"recipe_id": "unimplemented", "provider_id": "fixture-independent", "bindings": []any{}, "prerequisites": []any{}, "requires_interfaces": []any{}, "supplies_interfaces": []any{}, "guarantee_changes": []any{}, "implementation": map[string]any{"status": "unimplemented", "reference": nil}, "resolution": resolution}
	explorationInput := map[string]any{"interpretations": []any{}, "additional_knowledge": []any{}, "provider_packs": []any{evaluated}, "query_time": "2026-09-11T00:00:00Z", "requirements": []any{requirement}, "slots": []any{map[string]any{"slot_id": "compute", "allowed_provider_ids": []any{"fixture-independent"}, "recipes": []any{recipe}}}, "allowed_guarantee_changes": []any{}, "counterfactuals": []any{}, "bounds": map[string]any{"max_candidates": 1}}
	exploration := importArtifact("composition_explore", explorationInput)
	importArtifact("composition_reassess", map[string]any{"before": exploration, "after": exploration})
	for operation, record := range retained {
		if record["kind"] == "" || record["installation"].(map[string]any)["Version"] != "0.6.0-dev" {
			t.Fatal("lost kind/version")
		}
		forged := workflowClone(t, record["artifact"].(map[string]any))
		forged["digest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		if _, err := runner.artifact(store, "import", map[string]any{"operation": operation, "input": inputs[operation], "result": forged}); err == nil {
			t.Fatal("accepted forged retained result", operation)
		}
		earlier := *runner
		earlier.installation.Version = "0.5.0-dev"
		if _, err := earlier.artifact(store, "import", map[string]any{"operation": operation, "input": inputs[operation], "result": nil}); err == nil {
			t.Fatal("earlier exact release admitted new artifact", operation)
		}
		// Show takes the original installation from the immutable record even
		// when the current caller has selected another version.
		shown, err := earlier.artifact(store, "show", map[string]any{"record_digest": record["digest"]})
		expected, _ := json.Marshal(record)
		if err != nil || !scvworkflow.Same(expected, shown) {
			t.Fatal("show substituted current version", operation, err)
		}
	}
}

package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Deliberately synthetic, prevalidated-structure input for the pure inventory
// projection. Full consumer acceptance uses independently emitted native results.
func obligationStructureFixture() map[string]any {
	check := func(id string) map[string]any {
		return map[string]any{"specification": map[string]any{"check_id": id}, "status": "unresolved", "claim_ids": []any{"premise"}}
	}
	return map[string]any{"digest": "sha256:" + strings.Repeat("a", 64),
		"input": map[string]any{"slots": []any{map[string]any{"slot_id": "worker", "recipes": []any{map[string]any{"recipe_id": "recipe", "implementation": map[string]any{"status": "unknown"}}}}}},
		"scenarios": []any{map[string]any{"scenario_id": "baseline", "candidates": []any{map[string]any{
			"candidate_id": "sha256:" + strings.Repeat("b", 64),
			"choices":      []any{map[string]any{"slot_id": "worker", "recipe_id": "recipe", "provider_id": "synthetic"}},
			"connections":  []any{map[string]any{"connection_id": "prerequisites:worker", "checks": []any{check("alpha"), check("beta")}}},
			"unbound_requirements": []any{
				map[string]any{"requirement_id": "missing", "reason": "missing_binding", "claim_references": []any{}},
				map[string]any{"requirement_id": "ambiguous", "reason": "ambiguous_binding", "claim_references": []any{map[string]any{"claim_id": "premise"}, map[string]any{"claim_id": "premise"}}}},
			"interfaces": []any{map[string]any{"slot_id": "worker", "recipe_id": "recipe", "interface_ref": "synthetic.interface.v1", "declared_match": false}},
			// Legacy summaries deliberately carry no check identity; they are not read.
			"obligations": []any{map[string]any{"kind": "check_unresolved"}, map[string]any{"kind": "check_unresolved"}},
		}}}},
		"evidence_obligations": []any{map[string]any{"pack_evaluation_digest": "sha256:" + strings.Repeat("c", 64), "fixture_id": "independent-case", "status": "not_run", "claim_ids": []any{"premise"}}}}
}

func TestSCVObligationDistinctPrerequisitesAndTypedTargets(t *testing.T) {
	composition := obligationStructureFixture()
	items, err := scvObligationInventory(composition)
	if err != nil || len(items) != 7 {
		t.Fatal("expected two checks, two bindings, interface, implementation and fixture", len(items), err)
	}
	counts := map[string]int{}
	checks := map[string]string{}
	last := ""
	for _, raw := range items {
		item := raw.(map[string]any)
		id := item["obligation_id"].(string)
		if id <= last {
			t.Fatal("inventory must have distinct sorted identities")
		}
		last = id
		kind := item["kind"].(string)
		counts[kind]++
		target := item["target"].(map[string]any)
		if len(target) != 10 {
			t.Fatal("target lost explicit nullable fields")
		}
		if kind == "check" {
			checks[target["check_id"].(string)] = id
			if target["connection_id"] != "prerequisites:worker" || target["requirement_id"] != nil || target["slot_id"] != nil {
				t.Fatal("precise check identity was replaced with legacy summary scope")
			}
		}
		if kind == "binding" && target["requirement_id"] == "ambiguous" && !scvEqual(item["claim_ids"], []any{"premise"}) {
			t.Fatal("ambiguous repeated claim IDs inflated the inventory")
		}
		if kind == "fixture" && (target["scenario_id"] != nil || target["candidate_id"] != nil || item["prior_status"] != "not_run") {
			t.Fatal("global unrun fixture became candidate-local evidence")
		}
	}
	if checks["alpha"] == "" || checks["beta"] == "" || checks["alpha"] == checks["beta"] || counts["implementation"] != 1 {
		t.Fatal("identical anonymous prerequisite summaries collapsed")
	}
	composition["digest"] = "sha256:" + strings.Repeat("d", 64)
	changed, err := scvObligationInventory(composition)
	if err != nil {
		t.Fatal(err)
	}
	oldIDs := map[string]bool{}
	for _, raw := range items {
		oldIDs[raw.(map[string]any)["obligation_id"].(string)] = true
	}
	for _, raw := range changed {
		if oldIDs[raw.(map[string]any)["obligation_id"].(string)] {
			t.Fatal("obligation identity silently crossed composition revisions")
		}
	}
}

func obligationTestProvenance() map[string]any {
	return map[string]any{"kind": "source", "producer": "explicit synthetic producer", "reference": "opaque:unretrieved",
		"content_digest": nil, "recorded_at": "2026-09-11T12:00:00Z", "description": "Caller submitted a reference; no content was retrieved or verified."}
}

func TestSCVObligationProvenanceBoundsAndNoAuthorityFields(t *testing.T) {
	if err := scvObligationProvenance(obligationTestProvenance()); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(map[string]any){
		"verification_claim": func(v map[string]any) { v["verified"] = true },
		"wrong_kind":         func(v map[string]any) { v["kind"] = "certification" },
		"null_producer":      func(v map[string]any) { v["producer"] = nil },
		"utf8_byte_bound":    func(v map[string]any) { v["reference"] = strings.Repeat("é", 257) },
		"control_text":       func(v map[string]any) { v["description"] = "line\ncommand" },
		"empty_description":  func(v map[string]any) { v["description"] = "" },
		"description_bound":  func(v map[string]any) { v["description"] = strings.Repeat("a", 2049) },
		"bad_digest":         func(v map[string]any) { v["content_digest"] = "sha256:" + strings.Repeat("A", 64) },
		"digest_number":      func(v map[string]any) { v["content_digest"] = 10 },
		"year_zero":          func(v map[string]any) { v["recorded_at"] = "0000-01-01T00:00:00Z" },
		"invalid_day":        func(v map[string]any) { v["recorded_at"] = "2026-02-29T00:00:00Z" },
		"offset":             func(v map[string]any) { v["recorded_at"] = "2026-09-11T12:00:00+00:00" },
	} {
		t.Run(name, func(t *testing.T) {
			v := obligationTestProvenance()
			change(v)
			if err := scvObligationProvenance(v); err == nil {
				t.Fatal("invalid/overclaiming provenance accepted")
			}
		})
	}
}

func obligationValidate(t *testing.T, operation string, input, result map[string]any, expected bool) {
	t.Helper()
	result = scvTestSeal(t, result)
	err := validateSCVObligation(operation, input, result)
	if (err == nil) != expected {
		t.Fatalf("%s expected acceptance %v: %v", operation, expected, err)
	}
}

func TestSCVObligationNativeParityAndResealedTampering(t *testing.T) {
	path := os.Getenv("SYMPHONY_SCV_OBLIGATION_FIXTURE")
	if path == "" {
		path = "testdata/scv-obligation.v1.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := scvObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	oi, or := fixture["obligations_input"].(map[string]any), fixture["obligations_result"].(map[string]any)
	fi, fr := fixture["followup_input"].(map[string]any), fixture["followup_result"].(map[string]any)
	obligationValidate(t, "composition_obligations", oi, or, true)
	obligationValidate(t, "composition_followup", fi, fr, true)
	checkCount := 0
	for _, raw := range or["obligations"].([]any) {
		if raw.(map[string]any)["kind"] == "check" {
			checkCount++
		}
	}
	if checkCount != 3 {
		t.Fatal("native fixture must exercise two distinct prerequisites and a common check")
	}
	for name, change := range map[string]func(map[string]any){
		"dropped_target": func(v map[string]any) { v["obligations"] = v["obligations"].([]any)[1:] },
		"wrong_obligation_id": func(v map[string]any) {
			v["obligations"].([]any)[0].(map[string]any)["obligation_id"] = "sha256:" + strings.Repeat("e", 64)
		},
		"fabricated_check_id": func(v map[string]any) {
			v["obligations"].([]any)[0].(map[string]any)["target"].(map[string]any)["check_id"] = "invented"
		},
		"broadened_description": func(v map[string]any) {
			v["obligations"].([]any)[0].(map[string]any)["description"] = "Runtime certified"
		},
		"malformed_domain": func(v map[string]any) { v["domain"] = []any{} },
	} {
		t.Run(name, func(t *testing.T) {
			v := scvTestClone(t, or)
			change(v)
			obligationValidate(t, "composition_obligations", oi, v, false)
		})
	}
	for name, change := range map[string]func(map[string]any){
		"fabricated_outcome": func(v map[string]any) { v["entries"].([]any)[0].(map[string]any)["outcome"] = "resolved" },
		"provenance_verified": func(v map[string]any) {
			v["entries"].([]any)[0].(map[string]any)["provenance_validation"] = "content_verified"
		},
		"causation_claim":       func(v map[string]any) { v["entries"].([]any)[0].(map[string]any)["causation"] = "established" },
		"forged_current_status": func(v map[string]any) { v["entries"].([]any)[0].(map[string]any)["current_status"] = "contradicted" },
		"candidate_changes":     func(v map[string]any) { v["candidate_changes"].([]any)[0].(map[string]any)["affected"] = false },
		"wrong_axes":            func(v map[string]any) { v["change_axes"].(map[string]any)["evidence"] = false },
		"dropped_submission":    func(v map[string]any) { v["entries"] = []any{} },
		"reordered_entries":     func(v map[string]any) { a := v["entries"].([]any); a[0], a[len(a)-1] = a[len(a)-1], a[0] },
		"false_limits":          func(v map[string]any) { v["limitations"] = []any{"No limitations; runtime certified."} },
	} {
		t.Run(name, func(t *testing.T) {
			v := scvTestClone(t, fr)
			change(v)
			obligationValidate(t, "composition_followup", fi, v, false)
		})
	}
}

func TestInstalledSCVObligationConsumerAndCallerContinuity(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_OBLIGATION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.8.0-dev owner")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	owner := func(operation string, input any) map[string]any {
		t.Helper()
		raw, err := SCVCanonical(input)
		if err != nil {
			t.Fatal(err)
		}
		response, err := InvokeSCVDomain(context.Background(), "scv", prefix, "0.8.0-dev", cwd, operation, raw)
		if err != nil {
			t.Fatal(operation, err)
		}
		v, err := scvObject(response.Result)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	knowledge := owner("knowledge_interpret", map[string]any{"captures": []any{}, "claims": []any{map[string]any{"claim_id": "limit", "subject": "fixture-service", "predicate": "maximum", "scope": map[string]any{"plan": "fixture"}, "statement_kind": "user_assertion", "value": map[string]any{"type": "integer", "value": 10, "unit": "units"}, "evidence": []any{}, "dependencies": []any{}}}, "interpreter_version": "synthetic-obligation-fixture", "selection_policy": map[string]any{"policy_id": "fixture", "max_age_seconds": 60, "partial_capture": "exclude", "allowed_statement_kinds": []any{"user_assertion"}}})
	input := compositionTestInput(t)
	input["additional_knowledge"] = []any{knowledge}
	before := owner("composition_explore", input)
	inventory := owner("composition_obligations", map[string]any{"composition": before})
	var target string
	for _, raw := range inventory["obligations"].([]any) {
		item := raw.(map[string]any)
		if item["kind"] == "check" {
			target = item["obligation_id"].(string)
			break
		}
	}
	if target == "" {
		t.Fatal("conditional caller assertion lost its obligation")
	}
	submissions := []any{map[string]any{"submission_id": "followup", "obligation_id": target, "provenance": obligationTestProvenance()}}
	fi := map[string]any{"before": before, "after": before, "submissions": submissions}
	result := owner("composition_followup", fi)
	entry := result["entries"].([]any)[0].(map[string]any)
	if entry["current_status"] != "conditional" || entry["outcome"] != "criterion_not_satisfied" {
		t.Fatal("a submitted reference promoted a caller assertion")
	}
	for name, change := range map[string]func(map[string]any){
		"requirement": func(v map[string]any) { v["requirements"] = []any{compositionTestRequirement(7)} },
		"recipe": func(v map[string]any) {
			v["slots"].([]any)[0].(map[string]any)["recipes"].([]any)[0].(map[string]any)["implementation"] = map[string]any{"status": "unimplemented", "reference": nil}
		},
		"bounds": func(v map[string]any) { v["bounds"] = map[string]any{"max_candidates": 1} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := scvTestClone(t, input)
			change(changed)
			after := owner("composition_explore", changed)
			request := map[string]any{"before": before, "after": after, "submissions": submissions}
			forged := scvTestClone(t, result)
			forged["input"] = request
			forged["after_digest"] = after["digest"]
			obligationValidate(t, "composition_followup", request, forged, false)
			bytes, _ := json.Marshal(request)
			if _, err := InvokeSCVDomain(context.Background(), "scv", prefix, "0.8.0-dev", cwd, "composition_followup", bytes); err == nil {
				t.Fatal("native owner accepted a changed caller problem")
			}
		})
	}
}

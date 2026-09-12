package knowledgeengine

import (
	"fmt"
	"sort"
)

const scvObligationLimit = 26496

var scvObligationDescriptions = map[string]string{
	"check":          "Review the exact native check and selected evidence; criterion state is not runtime compatibility.",
	"binding":        "Supply one unambiguous scoped binding for this caller requirement; mapping changes require a separate composition.",
	"interface":      "Verify or implement the exact declared interface contract; a reference alone does not establish connectivity.",
	"implementation": "Verify the declared implementation and its runtime behavior; availability is caller asserted.",
	"fixture":        "Review the exact provider-pack fixture and authored mapping; fixture status is not provider impossibility.",
}

var scvObligationLimitations = []string{
	"inventory covers obligations in the exact replayed finite composition only; it does not establish global completeness or prioritize work",
	"obligation identities bind the composition digest and precise target; opaque references are never executed or treated as evidence",
	"implementation and interface declarations require separate verification; check satisfaction is not runtime certification",
}

var scvFollowupLimitations = []string{
	"submissions retain opaque provenance references only; no content, producer identity, authorization or execution is verified",
	"criterion state comes only from the independently replayed after composition; a satisfied criterion does not prove the submission caused it",
	"requirements, recipes, selections, bounds and evidence policy remain fixed; query-time differences are reported without asserting chronology or causation",
	"non-check obligations require separate verification; no submission globally resolves an obligation or certifies runtime compatibility",
}

func scvObligationTarget() map[string]any {
	return map[string]any{"scenario_id": nil, "candidate_id": nil, "connection_id": nil, "check_id": nil,
		"requirement_id": nil, "slot_id": nil, "recipe_id": nil, "interface_ref": nil,
		"pack_evaluation_digest": nil, "fixture_id": nil}
}

// The composition must have passed validateSCVComposition before this function.
// This derives targets from exact native checks, never anonymous obligation prose.
func scvObligationInventory(composition map[string]any) ([]any, error) {
	items := []any{}
	seen := map[string]bool{}
	add := func(kind string, target map[string]any, status, claims any) error {
		id, err := SCVDigest(map[string]any{"composition_digest": composition["digest"], "kind": kind, "target": target})
		if err != nil {
			return err
		}
		if seen[id] || len(items) >= scvObligationLimit {
			return fmt.Errorf("duplicate or excessive obligation inventory")
		}
		seen[id] = true
		items = append(items, map[string]any{"obligation_id": id, "kind": kind, "target": target,
			"prior_status": status, "claim_ids": claims, "description": scvObligationDescriptions[kind]})
		return nil
	}
	recipes := map[string]map[string]map[string]any{}
	for _, raw := range composition["input"].(map[string]any)["slots"].([]any) {
		slot := raw.(map[string]any)
		selected := map[string]map[string]any{}
		for _, raw := range slot["recipes"].([]any) {
			r := raw.(map[string]any)
			selected[r["recipe_id"].(string)] = r
		}
		recipes[slot["slot_id"].(string)] = selected
	}
	for _, raw := range composition["scenarios"].([]any) {
		scenario := raw.(map[string]any)
		for _, raw := range scenario["candidates"].([]any) {
			candidate := raw.(map[string]any)
			target := func() map[string]any {
				t := scvObligationTarget()
				t["scenario_id"], t["candidate_id"] = scenario["scenario_id"], candidate["candidate_id"]
				return t
			}
			for _, raw := range candidate["connections"].([]any) {
				connection := raw.(map[string]any)
				for _, raw := range connection["checks"].([]any) {
					check := raw.(map[string]any)
					if check["status"] == "satisfied" {
						continue
					}
					t := target()
					t["connection_id"], t["check_id"] = connection["connection_id"], check["specification"].(map[string]any)["check_id"]
					if err := add("check", t, check["status"], check["claim_ids"]); err != nil {
						return nil, err
					}
				}
			}
			for _, raw := range candidate["unbound_requirements"].([]any) {
				unbound := raw.(map[string]any)
				t := target()
				t["requirement_id"] = unbound["requirement_id"]
				ids := map[string]bool{}
				for _, raw := range unbound["claim_references"].([]any) {
					ids[raw.(map[string]any)["claim_id"].(string)] = true
				}
				claims := []string{}
				for id := range ids {
					claims = append(claims, id)
				}
				sort.Strings(claims)
				if err := add("binding", t, unbound["reason"], claims); err != nil {
					return nil, err
				}
			}
			for _, raw := range candidate["interfaces"].([]any) {
				item := raw.(map[string]any)
				if item["declared_match"] == true {
					continue
				}
				t := target()
				for _, key := range []string{"slot_id", "recipe_id", "interface_ref"} {
					t[key] = item[key]
				}
				if err := add("interface", t, "missing", []any{}); err != nil {
					return nil, err
				}
			}
			for _, raw := range candidate["choices"].([]any) {
				choice := raw.(map[string]any)
				t := target()
				t["slot_id"], t["recipe_id"] = choice["slot_id"], choice["recipe_id"]
				r := recipes[choice["slot_id"].(string)][choice["recipe_id"].(string)]
				if r == nil {
					return nil, fmt.Errorf("obligation choice lacks exact recipe")
				}
				if err := add("implementation", t, r["implementation"].(map[string]any)["status"], []any{}); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, raw := range composition["evidence_obligations"].([]any) {
		item := raw.(map[string]any)
		t := scvObligationTarget()
		t["pack_evaluation_digest"], t["fixture_id"] = item["pack_evaluation_digest"], item["fixture_id"]
		if err := add("fixture", t, item["status"], item["claim_ids"]); err != nil {
			return nil, err
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].(map[string]any)["obligation_id"].(string) < items[j].(map[string]any)["obligation_id"].(string)
	})
	return items, nil
}

func scvObligationComposition(value any, domain any) (map[string]any, error) {
	composition, ok := value.(map[string]any)
	if !ok || composition["domain"] != domain {
		return nil, fmt.Errorf("obligation composition owner mismatch")
	}
	input, ok := composition["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("obligation composition lacks input")
	}
	if err := validateSCVComposition("composition_explore", input, composition); err != nil {
		return nil, err
	}
	return composition, nil
}

func scvObligationProvenance(value any) error {
	p, err := compObject(value, "kind", "producer", "reference", "content_digest", "recorded_at", "description")
	if err != nil {
		return err
	}
	kind, err := compText(p["kind"])
	if err != nil || !compContains([]string{"source", "observation", "adapter", "caller_decision"}, kind) {
		return fmt.Errorf("invalid obligation provenance kind")
	}
	for _, key := range []string{"producer", "reference"} {
		if _, err := compText(p[key]); err != nil {
			return err
		}
	}
	if p["content_digest"] != nil {
		d, ok := p["content_digest"].(string)
		if !ok || !taggedSHA256(d) {
			return fmt.Errorf("invalid claimed provenance digest")
		}
	}
	if _, err = scvCorpusTime(p["recorded_at"]); err != nil {
		return err
	}
	if p["recorded_at"].(string)[:4] == "0000" {
		return fmt.Errorf("provenance time excludes year zero")
	}
	s, ok := p["description"].(string)
	if !ok || len(s) == 0 || len(s) > 2048 {
		return fmt.Errorf("invalid obligation provenance description")
	}
	for _, c := range []byte(s) {
		if c < 32 || c == 127 {
			return fmt.Errorf("obligation provenance control text")
		}
	}
	return nil
}

// Only called after the full composition consumer validates both inputs.
func scvObligationFindCheck(composition, target map[string]any) (map[string]any, error) {
	for _, raw := range composition["scenarios"].([]any) {
		scenario := raw.(map[string]any)
		if scenario["scenario_id"] != target["scenario_id"] {
			continue
		}
		for _, raw := range scenario["candidates"].([]any) {
			candidate := raw.(map[string]any)
			if candidate["candidate_id"] != target["candidate_id"] {
				continue
			}
			for _, raw := range candidate["connections"].([]any) {
				connection := raw.(map[string]any)
				if connection["connection_id"] != target["connection_id"] {
					continue
				}
				for _, raw := range connection["checks"].([]any) {
					check := raw.(map[string]any)
					if check["specification"].(map[string]any)["check_id"] == target["check_id"] {
						return check, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("obligation exact check absent from composition")
}

func validateSCVObligation(operation string, input, result map[string]any) error {
	if err := scvSeal(result, "digest"); err != nil {
		return err
	}
	domain, ok := result["domain"].(string)
	if !ok || !scvCoverageAdmits(domain, domain) {
		return fmt.Errorf("invalid obligation owner domain")
	}
	if !scvEqual(input, result["input"]) {
		return fmt.Errorf("obligation retained input mismatch")
	}
	if operation == "composition_obligations" {
		if !scvCorpusFields(input, "composition") || !scvCorpusFields(result, "protocol", "domain", "input", "composition_digest", "obligations", "limitations", "digest") || result["protocol"] != "symphony.scv.composition-obligations.v1" {
			return fmt.Errorf("invalid obligation inventory fields")
		}
		composition, err := scvObligationComposition(input["composition"], result["domain"])
		if err != nil {
			return err
		}
		items, err := scvObligationInventory(composition)
		if err != nil {
			return err
		}
		if result["composition_digest"] != composition["digest"] || !scvEqual(result["obligations"], items) || !scvEqual(result["limitations"], scvObligationLimitations) {
			return fmt.Errorf("obligation inventory identity or attribution mismatch")
		}
		return nil
	}
	if operation != "composition_followup" || !scvCorpusFields(input, "before", "after", "submissions") || !scvCorpusFields(result, "protocol", "domain", "input", "before_digest", "after_digest", "change_axes", "candidate_changes", "entries", "limitations", "digest") || result["protocol"] != "symphony.scv.composition-followup.v1" {
		return fmt.Errorf("invalid obligation followup fields")
	}
	return scvValidateObligationFollowup(input, result)
}

func scvValidateObligationFollowup(input, result map[string]any) error {
	before, err := scvObligationComposition(input["before"], result["domain"])
	if err != nil {
		return err
	}
	after, err := scvObligationComposition(input["after"], result["domain"])
	if err != nil {
		return err
	}
	a, b := before["input"].(map[string]any), after["input"].(map[string]any)
	for _, key := range []string{"requirements", "slots", "allowed_guarantee_changes", "counterfactuals", "bounds"} {
		if !scvEqual(a[key], b[key]) {
			return fmt.Errorf("obligation followup changes caller %s", key)
		}
	}
	be := before["evidence_evaluation"].(map[string]any)["graph_evaluation"].(map[string]any)
	ae := after["evidence_evaluation"].(map[string]any)["graph_evaluation"].(map[string]any)
	if !scvEqual(be["selection_policy"], ae["selection_policy"]) {
		return fmt.Errorf("obligation followup changes evidence policy")
	}
	// Reuse the independent reassessment checker without trusting any fabricated
	// top-level reassessment envelope or performing another native invocation.
	reassessment := map[string]any{"protocol": "symphony.scv.composition-reassessment.v1", "domain": result["domain"],
		"input": map[string]any{"before": before, "after": after}, "before_digest": result["before_digest"], "after_digest": result["after_digest"],
		"change_axes": result["change_axes"], "candidates": result["candidate_changes"], "search_changed": !scvEqual(before["search"], after["search"]),
		"limitations": []any{"Internal consumer reassessment check; no new native result."}}
	reassessment["digest"], err = SCVDigest(reassessment)
	if err != nil {
		return err
	}
	if err = compValidateReassessment(reassessment["input"].(map[string]any), reassessment); err != nil {
		return err
	}
	items, err := scvObligationInventory(before)
	if err != nil {
		return err
	}
	inventory := map[string]map[string]any{}
	for _, raw := range items {
		item := raw.(map[string]any)
		inventory[item["obligation_id"].(string)] = item
	}
	submissions, err := compArray(input["submissions"], 32, 1)
	if err != nil {
		return err
	}
	seenIDs, seenTargets := map[string]bool{}, map[string]bool{}
	expected := []any{}
	for _, raw := range submissions {
		submission, err := compObject(raw, "submission_id", "obligation_id", "provenance")
		if err != nil {
			return err
		}
		id, err := compText(submission["submission_id"])
		if err != nil || seenIDs[id] {
			return fmt.Errorf("duplicate or invalid obligation submission")
		}
		digest, ok := submission["obligation_id"].(string)
		if !ok || !taggedSHA256(digest) || inventory[digest] == nil || seenTargets[digest] {
			return fmt.Errorf("unknown or repeated exact obligation")
		}
		seenIDs[id], seenTargets[digest] = true, true
		if err := scvObligationProvenance(submission["provenance"]); err != nil {
			return err
		}
		item := inventory[digest]
		var current any
		outcome := "requires_separate_verification"
		if item["kind"] == "check" {
			target := item["target"].(map[string]any)
			old, err := scvObligationFindCheck(before, target)
			if err != nil {
				return err
			}
			updated, err := scvObligationFindCheck(after, target)
			if err != nil || !scvEqual(old["specification"], updated["specification"]) {
				return fmt.Errorf("obligation followup changes exact check")
			}
			current, outcome = updated["status"], "criterion_not_satisfied"
			if current == "satisfied" {
				outcome = "criterion_satisfied"
			}
		}
		expected = append(expected, map[string]any{"submission_id": id, "obligation_id": digest, "kind": item["kind"],
			"target": item["target"], "prior_status": item["prior_status"], "current_status": current, "outcome": outcome,
			"claim_ids": item["claim_ids"], "provenance_validation": "reference_only", "causation": "not_established"})
	}
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].(map[string]any)["submission_id"].(string) < expected[j].(map[string]any)["submission_id"].(string)
	})
	if !scvEqual(result["entries"], expected) || !scvEqual(result["limitations"], scvFollowupLimitations) {
		return fmt.Errorf("obligation followup outcome or provenance attribution mismatch")
	}
	return nil
}

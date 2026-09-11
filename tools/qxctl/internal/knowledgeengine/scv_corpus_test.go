package knowledgeengine

import (
	"encoding/json"
	"testing"
)

func corpusBoundarySeal(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	delete(value, "digest")
	digest, err := SCVDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	value["digest"] = digest
	return value
}
func corpusBoundaryClone(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	raw, err := SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	copy, err := scvObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	return copy
}
func corpusBoundaryIndex(t *testing.T, id, observed, completeness string) map[string]any {
	t.Helper()
	source, _ := SCVDigest(map[string]any{"fixture_source": id})
	body, _ := SCVDigest(map[string]any{"fixture_body": id})
	capture, _ := SCVDigest(map[string]any{"fixture_capture": id, "observed": observed, "completeness": completeness})
	issues := []any{}
	if completeness != "complete" {
		issues = append(issues, "fixture incomplete acquisition")
	}
	return corpusBoundarySeal(t, map[string]any{"protocol": "symphony.scv.capture-index.v1", "domain": "scv",
		"source_id": id, "provider_id": "cf", "family_id": "scev", "source_digest": source, "source_generation": 1,
		"locator_id": "main", "requested_uri": "https://fixture.invalid/" + id, "capture_digest": capture,
		"body_digest": body, "byte_size": 4, "observed_at": observed, "upstream_revision": nil,
		"media_type": "text/plain", "completeness": completeness, "issues": issues})
}
func corpusBoundaryMember(id string, latest, last any) map[string]any {
	return map[string]any{"member_id": id, "latest_attempt": latest, "last_complete": last}
}
func corpusBoundarySnapshot(t *testing.T, generation int, parent any, members ...map[string]any) map[string]any {
	t.Helper()
	list := []any{}
	indexed := map[string]map[string]any{}
	for _, member := range members {
		list = append(list, member)
		indexed[member["member_id"].(string)] = member
	}
	return corpusBoundarySeal(t, map[string]any{"protocol": "symphony.scv.corpus.v1", "domain": "scv", "corpus_id": "fixture",
		"generation": generation, "parent_digest": parent, "snapshot_time": "2026-09-11T00:00:00Z",
		"members": list, "coverage": scvCorpusCoverage(indexed)})
}
func corpusBoundaryFixtures(t *testing.T) (map[string]any, map[string]any, map[string]any) {
	t.Helper()
	a := corpusBoundaryIndex(t, "a", "2026-09-10T00:00:00Z", "complete")
	b := corpusBoundaryIndex(t, "b", "2026-09-10T00:00:00Z", "complete")
	failed := corpusBoundaryIndex(t, "b", "2026-09-11T00:00:00Z", "failed")
	before := corpusBoundarySnapshot(t, 1, nil, corpusBoundaryMember("a", a, a), corpusBoundaryMember("b", b, b))
	after := corpusBoundarySnapshot(t, 2, before["digest"], corpusBoundaryMember("a", a, a), corpusBoundaryMember("b", failed, b))
	input := map[string]any{"corpus_id": "fixture", "previous": before, "snapshot_time": after["snapshot_time"],
		"attempts": []any{map[string]any{"member_id": "b", "capture": failed}, map[string]any{"member_id": "a", "capture": a}}}
	return input, before, after
}
func corpusBoundaryValidate(t *testing.T, operation string, input, output map[string]any, valid bool) {
	t.Helper()
	payload, err := SCVCanonical(input)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := SCVCanonical(corpusBoundarySeal(t, output))
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateSCVResult(operation, payload, raw)
	if valid && err != nil {
		t.Fatal(err)
	}
	if !valid && err == nil {
		t.Fatal("re-sealed result changed exact request/evidence attribution")
	}
}
func corpusBoundaryQuery(t *testing.T, corpus map[string]any, ids []any, selection string) (map[string]any, map[string]any) {
	t.Helper()
	input := map[string]any{"corpus": corpus, "query_time": "2026-09-11T00:00:00Z", "member_ids": ids,
		"selection": selection, "max_age_seconds": nil}
	all := corpus["members"].([]any)
	selected := map[string]map[string]any{}
	for _, raw := range all {
		member := raw.(map[string]any)
		include := len(ids) == 0
		for _, id := range ids {
			include = include || id == member["member_id"]
		}
		if include {
			selected[member["member_id"].(string)] = member
		}
	}
	results := []any{}
	for _, id := range scvCorpusKeys(selected) {
		member := selected[id]
		latest := member["latest_attempt"].(map[string]any)
		chosen := member[selection]
		status, freshness := "unavailable", "not_selected"
		var same any
		if chosen != nil {
			index := chosen.(map[string]any)
			status, freshness = index["completeness"].(string), "current"
			same = scvEqual(index["source_digest"], latest["source_digest"])
		}
		results = append(results, map[string]any{"member_id": id, "latest_attempt_digest": latest["capture_digest"], "selected": chosen,
			"status": status, "freshness": freshness, "source_revision_matches_latest": same, "reasons": []any{}})
	}
	result := map[string]any{"protocol": "symphony.scv.corpus-query.v1", "domain": "scv", "corpus_digest": corpus["digest"],
		"query_time": input["query_time"], "selection": selection, "max_age_seconds": nil, "members": results, "coverage": scvCorpusCoverage(selected)}
	return input, result
}

func TestSCVCorpusBuildBindsRetainedCompleteAndCoverage(t *testing.T) {
	input, _, output := corpusBoundaryFixtures(t)
	corpusBoundaryValidate(t, "corpus_build", input, output, true)
	for name, mutate := range map[string]func(map[string]any){
		"lost_complete": func(v map[string]any) { v["members"].([]any)[1].(map[string]any)["last_complete"] = nil },
		"invented_complete": func(v map[string]any) {
			v["members"].([]any)[1].(map[string]any)["last_complete"] = corpusBoundaryIndex(t, "b", "2026-09-10T01:00:00Z", "complete")
		},
		"misleading_coverage": func(v map[string]any) { v["coverage"].(map[string]any)["complete_attempts"] = 2 },
		"duplicated_member":   func(v map[string]any) { v["members"].([]any)[1] = v["members"].([]any)[0] },
		"wrong_generation":    func(v map[string]any) { v["generation"] = 1 },
		"wrong_parent":        func(v map[string]any) { v["parent_digest"] = nil },
		"member_scalar":       func(v map[string]any) { v["members"].([]any)[0] = true },
		"missing_attempt":     func(v map[string]any) { delete(v["members"].([]any)[0].(map[string]any), "latest_attempt") },
	} {
		t.Run(name, func(t *testing.T) {
			value := corpusBoundaryClone(t, output)
			mutate(value)
			corpusBoundaryValidate(t, "corpus_build", input, value, false)
		})
	}
}
func TestSCVCorpusQueryBindsExactRequestedSubset(t *testing.T) {
	_, _, corpus := corpusBoundaryFixtures(t)
	input, output := corpusBoundaryQuery(t, corpus, []any{"b"}, "last_complete")
	corpusBoundaryValidate(t, "corpus_query", input, output, true)
	for name, mutate := range map[string]func(map[string]any){
		"wrong_valid_member": func(v map[string]any) {
			_, other := corpusBoundaryQuery(t, corpus, []any{"a"}, "last_complete")
			v["members"] = other["members"]
			v["coverage"] = other["coverage"]
		},
		"omitted_requested_member": func(v map[string]any) { v["members"] = []any{} },
		"extra_requested_member": func(v map[string]any) {
			_, all := corpusBoundaryQuery(t, corpus, []any{}, "last_complete")
			v["members"] = all["members"]
		},
		"wrong_selected_slot": func(v map[string]any) {
			v["members"].([]any)[0].(map[string]any)["selected"] = corpus["members"].([]any)[1].(map[string]any)["latest_attempt"]
		},
		"wrong_latest_digest": func(v map[string]any) {
			v["members"].([]any)[0].(map[string]any)["latest_attempt_digest"] = corpus["members"].([]any)[0].(map[string]any)["latest_attempt"].(map[string]any)["capture_digest"]
		},
		"wrong_status": func(v map[string]any) { v["members"].([]any)[0].(map[string]any)["status"] = "failed" },
		"wrong_source_agreement": func(v map[string]any) {
			v["members"].([]any)[0].(map[string]any)["source_revision_matches_latest"] = false
		},
		"wrong_coverage":  func(v map[string]any) { v["coverage"].(map[string]any)["requested_members"] = 2 },
		"invalid_reasons": func(v map[string]any) { v["members"].([]any)[0].(map[string]any)["reasons"] = []any{nil} },
		"scalar_member":   func(v map[string]any) { v["members"] = []any{true} },
	} {
		t.Run(name, func(t *testing.T) {
			value := corpusBoundaryClone(t, output)
			mutate(value)
			corpusBoundaryValidate(t, "corpus_query", input, value, false)
		})
	}
	allInput, allOutput := corpusBoundaryQuery(t, corpus, []any{}, "latest_attempt")
	corpusBoundaryValidate(t, "corpus_query", allInput, allOutput, true)
	bad := corpusBoundaryClone(t, allOutput)
	list := bad["members"].([]any)
	list[0], list[1] = list[1], list[0]
	corpusBoundaryValidate(t, "corpus_query", allInput, bad, false)
	for _, ids := range []any{[]any{"b", "b"}, []any{"missing"}, []any{true}, true} {
		badInput := corpusBoundaryClone(t, input)
		badInput["member_ids"] = ids
		corpusBoundaryValidate(t, "corpus_query", badInput, output, false)
	}
}
func TestSCVCorpusQueryPreservesTemporalQualification(t *testing.T) {
	_, _, corpus := corpusBoundaryFixtures(t)
	input, output := corpusBoundaryQuery(t, corpus, []any{"b"}, "last_complete")
	input["max_age_seconds"], output["max_age_seconds"] = 60, 60
	output["members"].([]any)[0].(map[string]any)["freshness"] = "expired"
	corpusBoundaryValidate(t, "corpus_query", input, output, true)
	wrong := corpusBoundaryClone(t, output)
	wrong["members"].([]any)[0].(map[string]any)["freshness"] = "current"
	corpusBoundaryValidate(t, "corpus_query", input, wrong, false)
	input["query_time"], output["query_time"] = "2026-09-10T00:01:00Z", "2026-09-10T00:01:00Z"
	output["members"].([]any)[0].(map[string]any)["freshness"] = "current"
	corpusBoundaryValidate(t, "corpus_query", input, output, true)
	input["query_time"], output["query_time"] = "2026-09-09T00:00:00Z", "2026-09-09T00:00:00Z"
	output["members"].([]any)[0].(map[string]any)["freshness"] = "future"
	corpusBoundaryValidate(t, "corpus_query", input, output, true)
}
func TestSCVCorpusDiffBindsMembershipAndChangeAxes(t *testing.T) {
	a := corpusBoundaryIndex(t, "a", "2026-09-10T00:00:00Z", "complete")
	b := corpusBoundaryIndex(t, "b", "2026-09-10T00:00:00Z", "complete")
	newA := corpusBoundaryIndex(t, "a", "2026-09-11T00:00:00Z", "complete")
	c := corpusBoundaryIndex(t, "c", "2026-09-11T00:00:00Z", "complete")
	before := corpusBoundarySnapshot(t, 1, nil, corpusBoundaryMember("a", a, a), corpusBoundaryMember("b", b, b))
	after := corpusBoundarySnapshot(t, 2, before["digest"], corpusBoundaryMember("a", newA, newA), corpusBoundaryMember("c", c, c))
	input := map[string]any{"before": before, "after": after}
	output := map[string]any{"protocol": "symphony.scv.corpus-diff.v1", "domain": "scv", "before_digest": before["digest"], "after_digest": after["digest"],
		"added_member_ids": []any{"c"}, "removed_member_ids": []any{"b"}, "affected_member_ids": []any{"a", "b", "c"},
		"changes": []any{map[string]any{"member_id": "a", "body_changed": false, "source_changed": false, "observation_changed": true, "coverage_changed": false, "last_complete_changed": true}}}
	corpusBoundaryValidate(t, "corpus_diff", input, output, true)
	for name, mutate := range map[string]func(map[string]any){
		"wrong_addition":    func(v map[string]any) { v["added_member_ids"] = []any{"b"} },
		"lost_removal":      func(v map[string]any) { v["removed_member_ids"] = []any{} },
		"wrong_axis":        func(v map[string]any) { v["changes"].([]any)[0].(map[string]any)["body_changed"] = true },
		"extra_axis":        func(v map[string]any) { v["changes"].([]any)[0].(map[string]any)["retired"] = true },
		"duplicated_change": func(v map[string]any) { v["changes"] = append(v["changes"].([]any), v["changes"].([]any)[0]) },
		"missing_affected":  func(v map[string]any) { v["affected_member_ids"] = []any{"a"} },
		"malformed_change":  func(v map[string]any) { v["changes"] = []any{nil} },
	} {
		t.Run(name, func(t *testing.T) {
			value := corpusBoundaryClone(t, output)
			mutate(value)
			corpusBoundaryValidate(t, "corpus_diff", input, value, false)
		})
	}
}
func TestSCVCorpusEmptySelectionsAndMalformedInputs(t *testing.T) {
	empty := corpusBoundarySnapshot(t, 1, nil)
	input := map[string]any{"corpus_id": "fixture", "previous": nil, "snapshot_time": empty["snapshot_time"], "attempts": []any{}}
	corpusBoundaryValidate(t, "corpus_build", input, empty, true)
	queryInput, queryOutput := corpusBoundaryQuery(t, empty, []any{}, "last_complete")
	corpusBoundaryValidate(t, "corpus_query", queryInput, queryOutput, true)
	for _, previous := range []any{true, []any{}, map[string]any{}, json.Number("2")} {
		bad := corpusBoundaryClone(t, input)
		bad["previous"] = previous
		corpusBoundaryValidate(t, "corpus_build", bad, empty, false)
	}
	bad := corpusBoundaryClone(t, input)
	bad["attempts"] = []any{nil}
	corpusBoundaryValidate(t, "corpus_build", bad, empty, false)
}

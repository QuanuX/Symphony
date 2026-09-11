package knowledgeengine

import (
	"fmt"
	"sort"
	"strings"
)

// Coverage verifies accounting over exact retained selections. Provider facts,
// profile interpretation, and native replay remain with the compiled owner.
func validateSCVProviderCoverage(input, result map[string]any) error {
	if !scvCorpusFields(input, "provider", "corpus_query", "interpretations") ||
		!scvCorpusFields(result, "protocol", "domain", "input", "corpus_query_result", "sources", "unlisted_members", "unselected_bindings", "summary", "limitations", "digest") || !scvEqual(input, result["input"]) {
		return fmt.Errorf("provider coverage changed exact input or result fields")
	}
	if err := scvSeal(result, "digest"); err != nil {
		return err
	}
	domain, ok := result["domain"].(string)
	if !ok || !scvCoverageAdmits(domain, domain) {
		return fmt.Errorf("invalid coverage owner")
	}
	limits, ok := result["limitations"].([]any)
	if !ok || len(limits) == 0 {
		return fmt.Errorf("missing bounded coverage limitations")
	}
	for _, raw := range limits {
		if s, ok := raw.(string); !ok || s == "" || len(s) > 4096 {
			return fmt.Errorf("invalid coverage limitation")
		}
	}
	provider, ok := input["provider"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing provider declaration")
	}
	declarationInput := map[string]any{}
	for _, key := range []string{"provider_id", "family_id", "display_name", "sources"} {
		declarationInput[key] = provider[key]
	}
	if err := scvCoverageValidateNested("provider_onboard", declarationInput, provider); err != nil {
		return err
	}
	if !scvCoverageSourceScope(domain, provider) {
		return fmt.Errorf("provider outside coverage owner")
	}
	queryInput, ok := input["corpus_query"].(map[string]any)
	if !ok || !scvCorpusFields(queryInput, "corpus", "query_time", "member_ids", "selection", "max_age_seconds") {
		return fmt.Errorf("invalid coverage corpus query")
	}
	query, ok := result["corpus_query_result"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing coverage corpus result")
	}
	if err := scvCoverageValidateNested("corpus_query", queryInput, query); err != nil {
		return err
	}
	corpus, all, err := scvCorpusSnapshot(queryInput["corpus"])
	if err != nil {
		return err
	}
	corpusDomain, _ := corpus["domain"].(string)
	if !scvCoverageAdmits(domain, corpusDomain) {
		return fmt.Errorf("corpus outside coverage owner")
	}
	byTuple := map[string]map[string]any{}
	for _, member := range all {
		index := member["latest_attempt"].(map[string]any)
		for _, field := range []string{"family_id", "provider_id", "source_id", "locator_id"} {
			if _, err := scvCorpusID(index[field]); err != nil {
				return fmt.Errorf("invalid coverage corpus identity: %s", field)
			}
		}
		key := scvCoverageTuple(index)
		if byTuple[key] != nil {
			return fmt.Errorf("duplicate corpus source locator identity")
		}
		byTuple[key] = member
	}
	selected, err := scvMapList(query["members"], "member_id")
	if err != nil {
		return err
	}
	selectedCaptures := map[string][]map[string]any{}
	for _, member := range selected {
		if index, ok := member["selected"].(map[string]any); ok {
			d, _ := index["capture_digest"].(string)
			selectedCaptures[d] = append(selectedCaptures[d], member)
		}
	}
	sources, ok := provider["sources"].([]any)
	if !ok || len(sources) == 0 || len(sources) > 32 {
		return fmt.Errorf("invalid provider source inventory")
	}
	declarations := map[string]map[string]any{}
	rows := map[string]map[string]any{}
	declaredMembers := map[string]string{}
	status := map[string]any{"complete": 0, "partial": 0, "failed": 0, "unavailable": 0}
	freshness := map[string]any{"current": 0, "expired": 0, "future": 0, "not_selected": 0}
	summary := map[string]any{"declared_sources": len(sources), "declared_locators": 0, "selected_declared_members": 0, "unselected_declared_locators": 0, "selected_unlisted_members": 0, "replayed_selected_captures": 0, "selected_profile_bindings": 0, "matched_rule_attempts": 0, "unresolved_rule_attempts": 0, "unselected_profile_bindings": 0, "selected_declared_status": status, "selected_declared_freshness": freshness}
	inc := func(object map[string]any, key string, n int) error {
		old, ok := object[key].(int)
		if !ok {
			return fmt.Errorf("unknown coverage partition %s", key)
		}
		object[key] = old + n
		return nil
	}
	for _, raw := range sources {
		source, ok := raw.(map[string]any)
		if !ok || !scvCorpusFields(source, "source_id", "provider_id", "family_id", "publisher", "authority_role", "scope", "locators", "continuity_evidence") {
			return fmt.Errorf("invalid declared source")
		}
		sid, err := scvCorpusID(source["source_id"])
		if err != nil || declarations[sid] != nil {
			return fmt.Errorf("duplicate or invalid declared source")
		}
		if !scvEqual(source["provider_id"], provider["provider_id"]) || !scvEqual(source["family_id"], provider["family_id"]) {
			return fmt.Errorf("declared source provider mismatch")
		}
		declarations[sid] = source
		locators, ok := source["locators"].([]any)
		if !ok || len(locators) == 0 || len(locators) > 16 {
			return fmt.Errorf("invalid declared locators")
		}
		for _, raw := range locators {
			locator, ok := raw.(map[string]any)
			if !ok || !scvCorpusFields(locator, "locator_id", "uri", "role", "format", "selector") {
				return fmt.Errorf("invalid declared locator")
			}
			lid, err := scvCorpusID(locator["locator_id"])
			key := sid + "\x00" + lid
			if err != nil || rows[key] != nil {
				return fmt.Errorf("duplicate or invalid declared locator")
			}
			row := map[string]any{"source_id": sid, "locator_id": lid, "member_id": nil, "selection_status": "not_selected", "selected_capture_digest": nil, "declaration_match": "not_available", "declaration_difference_fields": []any{}, "interpretations": []any{}}
			tuple := map[string]any{"family_id": source["family_id"], "provider_id": source["provider_id"], "source_id": sid, "locator_id": lid}
			if member := byTuple[scvCoverageTuple(tuple)]; member != nil {
				id := member["member_id"].(string)
				row["member_id"] = id
				declaredMembers[id] = key
				if chosen := selected[id]; chosen != nil {
					_ = inc(summary, "selected_declared_members", 1)
					if err := inc(status, chosen["status"].(string), 1); err != nil {
						return err
					}
					if err := inc(freshness, chosen["freshness"].(string), 1); err != nil {
						return err
					}
					row["selection_status"] = "unavailable"
					if index, ok := chosen["selected"].(map[string]any); ok {
						row["selection_status"] = "selected"
						row["selected_capture_digest"] = index["capture_digest"]
					}
				}
			}
			if row["selection_status"] == "not_selected" {
				_ = inc(summary, "unselected_declared_locators", 1)
			}
			_ = inc(summary, "declared_locators", 1)
			rows[key] = row
		}
	}
	unlisted := []any{}
	for _, id := range scvCorpusKeys(selected) {
		if declaredMembers[id] == "" {
			index := all[id]["latest_attempt"].(map[string]any)
			reason := "locator_not_declared"
			if !scvEqual(index["provider_id"], provider["provider_id"]) || !scvEqual(index["family_id"], provider["family_id"]) {
				reason = "provider_not_declared"
			} else if declarations[index["source_id"].(string)] == nil {
				reason = "source_not_declared"
			}
			unlisted = append(unlisted, map[string]any{"member_id": id, "family_id": index["family_id"], "provider_id": index["provider_id"], "source_id": index["source_id"], "locator_id": index["locator_id"], "reason": reason})
		}
	}
	wrappers, ok := input["interpretations"].([]any)
	if !ok || len(wrappers) > 16 {
		return fmt.Errorf("invalid coverage interpretation count")
	}
	seen := map[string]bool{}
	replayed := map[string]bool{}
	unselected := []any{}
	for _, raw := range wrappers {
		wrapper, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid coverage interpretation")
		}
		knowledge, ok := wrapper["knowledge"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing coverage knowledge")
		}
		wi := map[string]any{"captures": knowledge["captures"], "profiles": wrapper["profiles"], "bindings": wrapper["bindings"], "selection_policy": knowledge["selection_policy"]}
		if err := scvCoverageValidateNested("provider_interpret", wi, wrapper); err != nil {
			return err
		}
		wd, _ := wrapper["digest"].(string)
		if seen[wd] {
			return fmt.Errorf("duplicate coverage interpretation")
		}
		seen[wd] = true
		captures, err := scvMapList(knowledge["captures"], "digest")
		if err != nil {
			return err
		}
		for cd, capture := range captures {
			source, ok := capture["source"].(map[string]any)
			if !ok || !scvCoverageSourceScope(domain, source) {
				return fmt.Errorf("capture outside coverage owner")
			}
			ci := map[string]any{}
			for key, value := range capture {
				if key != "protocol" && key != "body_digest" && key != "byte_size" && key != "digest" {
					ci[key] = value
				}
			}
			if err := scvCoverageValidateNested("capture_import", ci, capture); err != nil {
				return err
			}
			if err := scvSeal(source, "digest"); err != nil {
				return err
			}
			for _, member := range selectedCaptures[cd] {
				if err := scvCoverageValidateNested("capture_index", map[string]any{"capture": capture}, member["selected"].(map[string]any)); err != nil {
					return err
				}
				if key := declaredMembers[member["member_id"].(string)]; key != "" {
					row := rows[key]
					desired := declarations[row["source_id"].(string)]
					differences := []any{}
					keys := []string{}
					for key := range desired {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					for _, key := range keys {
						if !scvEqual(desired[key], source[key]) {
							differences = append(differences, key)
						}
					}
					row["declaration_match"] = "matches"
					if len(differences) > 0 {
						row["declaration_match"] = "differs"
					}
					row["declaration_difference_fields"] = differences
					replayed[cd] = true
				}
			}
		}
		for _, raw := range wrapper["bindings"].([]any) {
			binding := raw.(map[string]any)
			pd := binding["profile_digest"].(string)
			cd := binding["capture_digest"].(string)
			chosen := selectedCaptures[cd]
			key := ""
			if len(chosen) > 0 {
				key = declaredMembers[chosen[0]["member_id"].(string)]
			}
			if key == "" {
				reason := "capture_not_selected"
				if len(chosen) > 0 {
					reason = "selected_member_not_declared"
				}
				unselected = append(unselected, map[string]any{"interpretation_digest": wd, "profile_digest": pd, "capture_digest": cd, "reason": reason})
				continue
			}
			row := rows[key]
			matched, unresolved := []any{}, []any{}
			for _, raw := range wrapper["extractions"].([]any) {
				e := raw.(map[string]any)
				if e["profile_digest"] == pd && e["capture_digest"] == cd {
					if e["status"] == "matched" {
						matched = append(matched, e["rule_id"])
					} else {
						unresolved = append(unresolved, e["rule_id"])
					}
				}
			}
			sort.Slice(matched, func(i, j int) bool { return matched[i].(string) < matched[j].(string) })
			sort.Slice(unresolved, func(i, j int) bool { return unresolved[i].(string) < unresolved[j].(string) })
			row["interpretations"] = append(row["interpretations"].([]any), map[string]any{"interpretation_digest": wd, "profile_digest": pd, "capture_digest": cd, "matched_rule_ids": matched, "unresolved_rule_ids": unresolved})
			_ = inc(summary, "selected_profile_bindings", 1)
			_ = inc(summary, "matched_rule_attempts", len(matched))
			_ = inc(summary, "unresolved_rule_attempts", len(unresolved))
		}
	}
	expectedRows := []any{}
	for _, key := range scvCorpusKeys(rows) {
		row := rows[key]
		scvCoverageSortBindings(row["interpretations"].([]any))
		expectedRows = append(expectedRows, row)
	}
	scvCoverageSortBindings(unselected)
	summary["selected_unlisted_members"] = len(unlisted)
	summary["replayed_selected_captures"] = len(replayed)
	summary["unselected_profile_bindings"] = len(unselected)
	for key, expected := range map[string]any{"sources": expectedRows, "unlisted_members": unlisted, "unselected_bindings": unselected, "summary": summary} {
		if !scvEqual(expected, result[key]) {
			return fmt.Errorf("provider coverage changed %s attribution", key)
		}
	}
	return nil
}

func scvCoverageValidateNested(operation string, input, result map[string]any) error {
	i, err := SCVCanonical(input)
	if err != nil {
		return err
	}
	r, err := SCVCanonical(result)
	if err != nil {
		return err
	}
	return ValidateSCVResult(operation, i, r)
}
func scvCoverageTuple(index map[string]any) string {
	raw, _ := SCVCanonical(scvCorpusTuple(index))
	return string(raw)
}
func scvCoverageAdmits(domain, owner string) bool {
	known := false
	for _, d := range SCVDomains() {
		if d == owner {
			known = true
		}
	}
	return known && (domain == "scv" || domain == owner || ((domain == "schv" || domain == "scev") && strings.HasPrefix(owner, domain+"-")))
}
func scvCoverageSourceScope(domain string, source map[string]any) bool {
	family, fok := source["family_id"].(string)
	provider, pok := source["provider_id"].(string)
	return fok && pok && (domain == "scv" || domain == family || domain == family+"-"+provider)
}
func scvCoverageSortBindings(values []any) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i].(map[string]any), values[j].(map[string]any)
		for _, k := range []string{"interpretation_digest", "profile_digest", "capture_digest", "reason"} {
			x, _ := a[k].(string)
			y, _ := b[k].(string)
			if x != y {
				return x < y
			}
		}
		return false
	})
}

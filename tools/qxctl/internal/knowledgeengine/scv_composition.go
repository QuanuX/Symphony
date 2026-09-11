package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type scvCompRecipe struct {
	slot, id, provider, status string
	raw                        map[string]any
	bindings                   map[string]map[string]any
	prerequisites              []any
	needs, supplies, changes   []string
	resolution                 map[string]any
}
type scvCompModel struct {
	requirements                 map[string]map[string]any
	scenarios                    map[string]map[string]map[string]any
	options                      map[string][]scvCompRecipe
	excluded                     []any
	requested, eligible, maximum int
	slots                        []string
}

func compObject(v any, keys ...string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok || !scvCorpusFields(m, keys...) {
		return nil, fmt.Errorf("invalid composition fields")
	}
	return m, nil
}
func compText(v any) (string, error) {
	s, ok := v.(string)
	if !ok || len(s) == 0 || len(s) > 512 {
		return "", fmt.Errorf("invalid composition text")
	}
	for _, c := range []byte(s) {
		if c < 32 || c == 127 {
			return "", fmt.Errorf("composition control text")
		}
	}
	return s, nil
}
func compArray(v any, max, min int) ([]any, error) {
	a, ok := v.([]any)
	if !ok || len(a) < min || len(a) > max {
		return nil, fmt.Errorf("composition array outside bounds")
	}
	return a, nil
}
func compStrings(v any, max, min int) ([]string, error) {
	a, e := compArray(v, max, min)
	if e != nil {
		return nil, e
	}
	out := []string{}
	seen := map[string]bool{}
	for _, x := range a {
		s, e := compText(x)
		if e != nil || seen[s] {
			return nil, fmt.Errorf("invalid/duplicate composition string")
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out, nil
}
func compContains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func compResolution(v any) (map[string]any, error) {
	m, e := compObject(v, "kind", "reference", "description")
	if e != nil {
		return nil, e
	}
	kind, e := compText(m["kind"])
	if e != nil || !compContains([]string{"source", "observation", "adapter", "caller_decision"}, kind) {
		return nil, fmt.Errorf("invalid resolution kind")
	}
	if _, e = compText(m["reference"]); e != nil {
		return nil, e
	}
	s, ok := m["description"].(string)
	if !ok || s == "" || len(s) > 2048 {
		return nil, fmt.Errorf("invalid resolution description")
	}
	for _, c := range []byte(s) {
		if c < 32 || c == 127 {
			return nil, fmt.Errorf("resolution control text")
		}
	}
	return m, nil
}
func compReference(v any, kind bool) error {
	keys := []string{"claim_id", "subject", "scope"}
	if kind {
		keys = append(keys, "kind")
	}
	m, e := compObject(v, keys...)
	if e != nil {
		return e
	}
	for _, k := range []string{"claim_id", "subject"} {
		if value, ok := m[k].(string); !ok || value == "" || len(value) > 512 || strings.IndexByte(value, 0) >= 0 {
			return fmt.Errorf("invalid scoped claim text")
		}
	}
	scope, ok := m["scope"].(map[string]any)
	if !ok || len(scope) > 16 {
		return fmt.Errorf("invalid claim scope")
	}
	for k, v := range scope {
		s, ok := v.(string)
		if k == "" || len(k) > 128 || !ok || len(s) > 512 {
			return fmt.Errorf("invalid claim scope qualifier")
		}
	}
	return nil
}
func compOperand(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid right operand")
	}
	if m["kind"] == "claim" {
		return compReference(m, true)
	}
	if !scvCorpusFields(m, "kind", "value") || m["kind"] != "literal" {
		return fmt.Errorf("invalid right operand")
	}
	tv, ok := m["value"].(map[string]any)
	if !ok || !scvValidTypedValue(tv) {
		return fmt.Errorf("invalid typed literal")
	}
	return nil
}
func compCheck(v any) error {
	m, e := compObject(v, "check_id", "importance", "left", "operator", "right")
	if e != nil {
		return e
	}
	if _, e = compText(m["check_id"]); e != nil {
		return e
	}
	if m["importance"] != "required" && m["importance"] != "optional" {
		return fmt.Errorf("invalid check importance")
	}
	if m["operator"] != "eq" && m["operator"] != "gte" && m["operator"] != "lte" {
		return fmt.Errorf("invalid check operator")
	}
	if e = compReference(m["left"], false); e != nil {
		return e
	}
	return compOperand(m["right"])
}
func compRequirements(v any) (map[string]map[string]any, error) {
	a, e := compArray(v, 32, 1)
	if e != nil {
		return nil, e
	}
	out := map[string]map[string]any{}
	for _, v := range a {
		m, e := compObject(v, "requirement_id", "importance", "operator", "right", "resolution")
		if e != nil {
			return nil, e
		}
		id, e := compText(m["requirement_id"])
		if e != nil || out[id] != nil {
			return nil, fmt.Errorf("duplicate/invalid common requirement")
		}
		if _, e = compResolution(m["resolution"]); e != nil {
			return nil, e
		}
		test := compMakeCheck(m, map[string]any{"claim_id": "unbound", "subject": "unbound", "scope": map[string]any{}})
		if e = compCheck(test); e != nil {
			return nil, e
		}
		out[id] = m
	}
	return out, nil
}
func compMakeCheck(r, claim map[string]any) map[string]any {
	return map[string]any{"check_id": r["requirement_id"], "importance": r["importance"], "operator": r["operator"], "right": r["right"], "left": claim}
}
func compConnection(id string, checks []any) map[string]any {
	return map[string]any{"connection_id": id, "from_subject": "caller-composition", "to_subject": "caller-requirements", "checks": checks}
}
func compSort(values []any) {
	sort.Slice(values, func(i, j int) bool {
		a, _ := SCVCanonical(values[i])
		b, _ := SCVCanonical(values[j])
		return string(a) < string(b)
	})
}
func compModel(input map[string]any) (scvCompModel, error) {
	m := scvCompModel{scenarios: map[string]map[string]map[string]any{}, options: map[string][]scvCompRecipe{}, excluded: []any{}, requested: 1, eligible: 1}
	var err error
	if !scvCorpusFields(input, "interpretations", "additional_knowledge", "provider_packs", "query_time", "requirements", "slots", "allowed_guarantee_changes", "counterfactuals", "bounds") {
		return m, fmt.Errorf("invalid exploration fields")
	}
	if m.requirements, err = compRequirements(input["requirements"]); err != nil {
		return m, err
	}
	m.scenarios["baseline"] = m.requirements
	counter, err := compArray(input["counterfactuals"], 4, 0)
	if err != nil {
		return m, err
	}
	for _, raw := range counter {
		c, e := compObject(raw, "counterfactual_id", "requirements")
		if e != nil {
			return m, e
		}
		id, e := compText(c["counterfactual_id"])
		if e != nil || m.scenarios[id] != nil {
			return m, fmt.Errorf("invalid counterfactual identity")
		}
		rs, e := compRequirements(c["requirements"])
		if e != nil || len(rs) != len(m.requirements) {
			return m, fmt.Errorf("counterfactual requirement mismatch")
		}
		for id := range m.requirements {
			if rs[id] == nil {
				return m, fmt.Errorf("counterfactual requirement mismatch")
			}
		}
		m.scenarios[id] = rs
	}
	allowed, err := compStrings(input["allowed_guarantee_changes"], 32, 0)
	if err != nil {
		return m, err
	}
	bounds, err := compObject(input["bounds"], "max_candidates")
	if err != nil {
		return m, err
	}
	n, err := scvCorpusInteger(bounds["max_candidates"], 1, 32)
	if err != nil {
		return m, err
	}
	m.maximum = int(n)
	slots, err := compArray(input["slots"], 4, 1)
	if err != nil {
		return m, err
	}
	seenSlots := map[string]bool{}
	for _, raw := range slots {
		s, e := compObject(raw, "slot_id", "allowed_provider_ids", "recipes")
		if e != nil {
			return m, e
		}
		sid, e := compText(s["slot_id"])
		if e != nil || len(sid) > 128 || seenSlots[sid] {
			return m, fmt.Errorf("duplicate/invalid slot")
		}
		seenSlots[sid] = true
		m.slots = append(m.slots, sid)
		m.options[sid] = []scvCompRecipe{}
		providers, e := compStrings(s["allowed_provider_ids"], 16, 1)
		if e != nil {
			return m, e
		}
		recipes, e := compArray(s["recipes"], 8, 1)
		if e != nil {
			return m, e
		}
		m.requested *= len(recipes)
		seenRecipes := map[string]bool{}
		for _, raw := range recipes {
			r, e := compObject(raw, "recipe_id", "provider_id", "bindings", "prerequisites", "requires_interfaces", "supplies_interfaces", "guarantee_changes", "implementation", "resolution")
			if e != nil {
				return m, e
			}
			rid, e := compText(r["recipe_id"])
			if e != nil || seenRecipes[rid] {
				return m, fmt.Errorf("duplicate/invalid recipe")
			}
			seenRecipes[rid] = true
			provider, e := compText(r["provider_id"])
			if e != nil {
				return m, e
			}
			recipe := scvCompRecipe{slot: sid, id: rid, provider: provider, raw: r, bindings: map[string]map[string]any{}}
			if recipe.resolution, e = compResolution(r["resolution"]); e != nil {
				return m, e
			}
			if recipe.needs, e = compStrings(r["requires_interfaces"], 16, 0); e != nil {
				return m, e
			}
			if recipe.supplies, e = compStrings(r["supplies_interfaces"], 16, 0); e != nil {
				return m, e
			}
			if recipe.changes, e = compStrings(r["guarantee_changes"], 16, 0); e != nil {
				return m, e
			}
			if recipe.prerequisites, e = compArray(r["prerequisites"], 16, 0); e != nil {
				return m, e
			}
			seenChecks := map[string]bool{}
			for _, v := range recipe.prerequisites {
				if e = compCheck(v); e != nil {
					return m, e
				}
				id := v.(map[string]any)["check_id"].(string)
				if seenChecks[id] {
					return m, fmt.Errorf("duplicate prerequisite check")
				}
				seenChecks[id] = true
			}
			imp, e := compObject(r["implementation"], "status", "reference")
			if e != nil {
				return m, e
			}
			recipe.status, e = compText(imp["status"])
			if e != nil || !compContains([]string{"available", "unimplemented", "unknown"}, recipe.status) {
				return m, fmt.Errorf("invalid implementation state")
			}
			if imp["reference"] != nil {
				if _, e = compText(imp["reference"]); e != nil {
					return m, e
				}
			} else if recipe.status == "available" {
				return m, fmt.Errorf("available implementation lacks reference")
			}
			bs, e := compArray(r["bindings"], 32, 0)
			if e != nil {
				return m, e
			}
			for _, raw := range bs {
				b, e := compObject(raw, "requirement_id", "claim")
				if e != nil {
					return m, e
				}
				id, e := compText(b["requirement_id"])
				if e != nil || m.requirements[id] == nil || recipe.bindings[id] != nil {
					return m, fmt.Errorf("unknown/duplicate recipe binding")
				}
				if e = compReference(b["claim"], false); e != nil {
					return m, e
				}
				recipe.bindings[id] = b["claim"].(map[string]any)
			}
			reasons := []any{}
			if !compContains(providers, provider) {
				reasons = append(reasons, "provider_not_selected")
			}
			for _, id := range recipe.changes {
				if !compContains(allowed, id) {
					reasons = append(reasons, "guarantee_change_not_permitted:"+id)
				}
			}
			if len(reasons) > 0 {
				m.excluded = append(m.excluded, map[string]any{"slot_id": sid, "recipe_id": rid, "reasons": reasons})
			} else {
				m.options[sid] = append(m.options[sid], recipe)
			}
		}
		sort.Slice(m.options[sid], func(i, j int) bool { return m.options[sid][i].id < m.options[sid][j].id })
		m.eligible *= len(m.options[sid])
	}
	sort.Strings(m.slots)
	sort.Slice(m.excluded, func(i, j int) bool {
		a, b := m.excluded[i].(map[string]any), m.excluded[j].(map[string]any)
		if a["slot_id"] != b["slot_id"] {
			return a["slot_id"].(string) < b["slot_id"].(string)
		}
		return a["recipe_id"].(string) < b["recipe_id"].(string)
	})
	return m, nil
}
func compProducts(m scvCompModel) [][]scvCompRecipe {
	out := [][]scvCompRecipe{}
	var walk func(int, []scvCompRecipe)
	walk = func(depth int, path []scvCompRecipe) {
		if len(out) >= m.maximum {
			return
		}
		if depth == len(m.slots) {
			out = append(out, append([]scvCompRecipe{}, path...))
			return
		}
		for _, r := range m.options[m.slots[depth]] {
			walk(depth+1, append(path, r))
			if len(out) >= m.maximum {
				return
			}
		}
	}
	walk(0, nil)
	return out
}
func compObligation(kind string, requirement, slot, recipe any, claims any, how map[string]any, detail string) map[string]any {
	return map[string]any{"kind": kind, "requirement_id": requirement, "slot_id": slot, "recipe_id": recipe, "claim_ids": claims, "resolution": how, "detail": detail}
}
func compAggregate(status map[string]bool) string {
	if len(status) == 0 {
		return "unresolved"
	}
	for _, s := range []string{"contradicted", "unresolved", "conditional"} {
		if status[s] {
			return s
		}
	}
	return "satisfied"
}
func compValidateCandidate(raw any, selected []scvCompRecipe, reqs map[string]map[string]any, findings map[string]map[string]any, evidenceObligations []any) error {
	result, e := compObject(raw, "candidate_id", "choices", "status", "implementation_status", "connections", "unbound_requirements", "interfaces", "claim_ids", "obligations", "evidence_obligation_refs")
	if e != nil {
		return e
	}
	choices, checks, connections, unbound, obligations := []any{}, []any{}, []any{}, []any{}, []any{}
	bindings := map[string][]map[string]any{}
	supplied := map[string]bool{}
	statuses, claimIDs := map[string]bool{}, map[string]bool{}
	unknown, unimplemented := false, false
	for _, r := range selected {
		choices = append(choices, map[string]any{"slot_id": r.slot, "recipe_id": r.id, "provider_id": r.provider})
		for id, ref := range r.bindings {
			bindings[id] = append(bindings[id], ref)
			claimIDs[ref["claim_id"].(string)] = true
		}
		for _, id := range r.supplies {
			supplied[id] = true
		}
		unknown = unknown || r.status == "unknown"
		unimplemented = unimplemented || r.status == "unimplemented"
		kind := "implementation_gap"
		if r.status == "available" {
			kind = "verify_implementation"
		}
		obligations = append(obligations, compObligation(kind, nil, r.slot, r.id, []any{}, r.resolution, "implementation status/reference is caller declared; verify exact adapter behavior and runtime installation"))
		connections = append(connections, compConnection("prerequisites:"+r.slot, r.prerequisites))
	}
	for _, id := range scvCorpusKeys(reqs) {
		r := reqs[id]
		// Preserve provenance even when a missing or ambiguous left binding
		// prevents an ordinary check from being evaluated.
		right := r["right"].(map[string]any)
		if right["kind"] == "claim" {
			claimIDs[right["claim_id"].(string)] = true
		}
		options := bindings[id]
		if len(options) != 1 {
			reason := "missing_binding"
			if len(options) > 1 {
				reason = "ambiguous_binding"
			}
			refs, ids := []any{}, []any{}
			for _, ref := range options {
				refs = append(refs, ref)
				ids = append(ids, ref["claim_id"])
			}
			unbound = append(unbound, map[string]any{"requirement_id": id, "reason": reason, "claim_references": refs})
			obligations = append(obligations, compObligation(reason, id, nil, nil, ids, r["resolution"].(map[string]any), "supply one unambiguous scoped claim binding for this common requirement"))
			if r["importance"] == "required" {
				statuses["unresolved"] = true
			}
		} else {
			checks = append(checks, compMakeCheck(r, options[0]))
		}
	}
	connections = append(connections, compConnection("requirements", checks))
	if e = scvValidateConnections(connections, result["connections"], findings); e != nil {
		return e
	}
	evaluated, e := compArray(result["connections"], 5, 1)
	if e != nil {
		return e
	}
	last := ""
	for _, raw := range evaluated {
		c := raw.(map[string]any)
		cid := c["connection_id"].(string)
		if cid <= last {
			return fmt.Errorf("reordered composition connections")
		}
		last = cid
		for _, raw := range c["checks"].([]any) {
			item := raw.(map[string]any)
			spec := item["specification"].(map[string]any)
			status := item["status"].(string)
			for _, rawID := range item["claim_ids"].([]any) {
				claimIDs[rawID.(string)] = true
			}
			if spec["importance"] == "required" {
				statuses[status] = true
			}
			if status == "satisfied" {
				continue
			}
			id := spec["check_id"].(string)
			var how map[string]any
			var rid, slot, recipe any
			if cid == "requirements" {
				rid = id
				how = reqs[id]["resolution"].(map[string]any)
			} else {
				for _, r := range selected {
					if cid == "prerequisites:"+r.slot {
						slot, recipe, how = r.slot, r.id, r.resolution
					}
				}
			}
			if how == nil {
				return fmt.Errorf("unknown prerequisite result")
			}
			obligations = append(obligations, compObligation("check_"+status, rid, slot, recipe, item["claim_ids"], how, "inspect the native check and exact evidence qualification; a contradiction is scoped to this requirement and selected evidence"))
		}
	}
	interfaces := []any{}
	for _, r := range selected {
		for _, id := range r.needs {
			matched := supplied[id]
			interfaces = append(interfaces, map[string]any{"slot_id": r.slot, "recipe_id": r.id, "interface_ref": id, "declared_match": matched})
			if !matched {
				statuses["unresolved"] = true
				obligations = append(obligations, compObligation("missing_interface", nil, r.slot, r.id, []any{}, r.resolution, "no selected recipe supplies exact interface contract: "+id))
			}
		}
	}
	compSort(obligations)
	ids := []string{}
	for id := range claimIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	implementation := "available"
	if unknown {
		implementation = "unknown"
	}
	if unimplemented {
		implementation = "unimplemented"
	}
	digest, e := SCVDigest(map[string]any{"choices": choices})
	if e != nil {
		return e
	}
	evidenceRefs := []any{}
	for _, raw := range evidenceObligations {
		issue := raw.(map[string]any)
		relevant := false
		for _, id := range issue["claim_ids"].([]string) {
			relevant = relevant || claimIDs[id]
		}
		if relevant {
			evidenceRefs = append(evidenceRefs, map[string]any{"pack_evaluation_digest": issue["pack_evaluation_digest"], "fixture_id": issue["fixture_id"], "status": issue["status"]})
		}
	}
	expected := map[string]any{"evidence_obligation_refs": evidenceRefs, "candidate_id": digest, "choices": choices, "status": compAggregate(statuses), "implementation_status": implementation, "connections": result["connections"], "unbound_requirements": unbound, "interfaces": interfaces, "claim_ids": ids, "obligations": obligations}
	if !scvEqual(result, expected) {
		return fmt.Errorf("composition candidate attribution mismatch")
	}
	return nil
}
func validateSCVComposition(operation string, input, result map[string]any) error {
	if operation == "composition_reassess" {
		return compValidateReassessment(input, result)
	}
	if operation != "composition_explore" {
		return fmt.Errorf("unsupported composition operation")
	}
	if !scvCorpusFields(result, "protocol", "domain", "input", "evidence_evaluation", "evidence_obligations", "search", "scenarios", "limitations", "digest") || result["protocol"] != "symphony.scv.composition-exploration.v1" || !scvEqual(input, result["input"]) {
		return fmt.Errorf("composition exact input/result mismatch")
	}
	if e := scvSeal(result, "digest"); e != nil {
		return e
	}
	domain, ok := result["domain"].(string)
	if !ok || !scvCoverageAdmits(domain, domain) {
		return fmt.Errorf("invalid composition domain")
	}
	limits, e := compStrings(result["limitations"], 16, 1)
	if e != nil || len(limits) == 0 {
		return fmt.Errorf("missing composition limitations")
	}
	model, e := compModel(input)
	if e != nil {
		return e
	}
	interpretations, e := compArray(input["interpretations"], 16, 0)
	if e != nil {
		return e
	}
	additional, e := compArray(input["additional_knowledge"], 16, 0)
	if e != nil {
		return e
	}
	extra := append([]any{}, additional...)
	packs, e := compArray(input["provider_packs"], 16, 0)
	if e != nil {
		return e
	}
	seen := map[string]bool{}
	evidenceObligations := []any{}
	for _, raw := range packs {
		p, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid selected provider pack")
		}
		pi, ok := p["input"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing pack input")
		}
		if e = validateSCVProviderPack("provider_pack_evaluate", pi, p); e != nil {
			return e
		}
		d, _ := p["digest"].(string)
		if seen[d] {
			return fmt.Errorf("duplicate pack selection")
		}
		seen[d] = true
		extra = append(extra, p["knowledge"])
		knowledge, ok := p["knowledge"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing pack knowledge")
		}
		claims, e := scvMapList(knowledge["claims"], "claim_id")
		if e != nil {
			return e
		}
		ids := scvCorpusKeys(claims)
		fixtures, e := compArray(p["fixture_results"], 16, 0)
		if e != nil {
			return e
		}
		for _, raw := range fixtures {
			f, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid pack fixture result")
			}
			if f["status"] != "passed" {
				evidenceObligations = append(evidenceObligations, map[string]any{"pack_evaluation_digest": p["digest"], "fixture_id": f["fixture_id"], "status": f["status"], "claim_ids": ids, "detail": "review the retained conformance fixture and authored mapping; failure or absence is not provider impossibility"})
			}
		}
	}
	compSort(evidenceObligations)
	if !scvEqual(evidenceObligations, result["evidence_obligations"]) {
		return fmt.Errorf("selected pack conformance obligations changed")
	}
	ev, ok := result["evidence_evaluation"].(map[string]any)
	if !ok || ev["domain"] != domain {
		return fmt.Errorf("invalid shared evidence evaluation")
	}
	ei := map[string]any{"interpretations": interpretations, "additional_knowledge": extra, "query_time": input["query_time"], "connections": []any{}}
	if e = scvCoverageValidateNested("connection_evaluate", ei, ev); e != nil {
		return e
	}
	assessment, ok := ev["graph_evaluation"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing evidence assessment")
	}
	findings, e := scvFindingMap(assessment)
	if e != nil {
		return e
	}
	products := compProducts(model)
	stop := "exhausted"
	if model.eligible == 0 {
		stop = "no_eligible_recipe"
	} else if len(products) < model.eligible {
		stop = "candidate_limit"
	}
	search := map[string]any{"requested_combinations": model.requested, "eligible_combinations": model.eligible, "evaluated_candidates": len(products), "exhaustive": len(products) == model.eligible, "stop_reason": stop, "excluded_recipes": model.excluded}
	if !scvEqual(search, result["search"]) {
		return fmt.Errorf("composition search space or exhaustion mismatch")
	}
	scenarios, e := compArray(result["scenarios"], 5, 1)
	if e != nil || len(scenarios) != len(model.scenarios) {
		return fmt.Errorf("composition scenario coverage mismatch")
	}
	scenarioKeys := []string{}
	for id := range model.scenarios {
		scenarioKeys = append(scenarioKeys, id)
	}
	sort.Strings(scenarioKeys)
	for i, raw := range scenarios {
		s, e := compObject(raw, "scenario_id", "changed_requirement_ids", "candidates")
		if e != nil || s["scenario_id"] != scenarioKeys[i] {
			return fmt.Errorf("composition scenario identity/order mismatch")
		}
		reqs := model.scenarios[scenarioKeys[i]]
		changed := []string{}
		for _, id := range scvCorpusKeys(reqs) {
			if !scvEqual(reqs[id], model.requirements[id]) {
				changed = append(changed, id)
			}
		}
		if !scvEqual(s["changed_requirement_ids"], changed) {
			return fmt.Errorf("counterfactual changes misattributed")
		}
		cs, e := compArray(s["candidates"], 32, 0)
		if e != nil || len(cs) != len(products) {
			return fmt.Errorf("candidate enumeration incomplete")
		}
		for j, c := range cs {
			if e = compValidateCandidate(c, products[j], reqs, findings, evidenceObligations); e != nil {
				return e
			}
		}
	}
	return nil
}
func compResultCandidates(result map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, raw := range result["scenarios"].([]any) {
		s := raw.(map[string]any)
		for _, raw := range s["candidates"].([]any) {
			c := raw.(map[string]any)
			key, _ := SCVCanonical([]any{s["scenario_id"], c["candidate_id"]})
			out[string(key)] = c
		}
	}
	return out
}
func compRecipeProjection(input map[string]any, selection bool) []any {
	slots, _ := scvMapList(input["slots"], "slot_id")
	key := "recipes"
	if selection {
		key = "allowed_provider_ids"
	}
	out := []any{}
	for _, id := range scvCorpusKeys(slots) {
		out = append(out, map[string]any{"slot_id": id, key: slots[id][key]})
	}
	return out
}
func compValidateReassessment(input, result map[string]any) error {
	if !scvCorpusFields(input, "before", "after") || !scvCorpusFields(result, "protocol", "domain", "input", "before_digest", "after_digest", "change_axes", "candidates", "search_changed", "limitations", "digest") || result["protocol"] != "symphony.scv.composition-reassessment.v1" || !scvEqual(result["input"], input) {
		return fmt.Errorf("invalid composition reassessment fields")
	}
	if e := scvSeal(result, "digest"); e != nil {
		return e
	}
	before, bok := input["before"].(map[string]any)
	after, aok := input["after"].(map[string]any)
	if !bok || !aok {
		return fmt.Errorf("missing composition results")
	}
	for _, v := range []map[string]any{before, after} {
		i, ok := v["input"].(map[string]any)
		if !ok || v["domain"] != result["domain"] {
			return fmt.Errorf("composition reassessment owner mismatch")
		}
		if e := validateSCVComposition("composition_explore", i, v); e != nil {
			return e
		}
	}
	if result["before_digest"] != before["digest"] || result["after_digest"] != after["digest"] {
		return fmt.Errorf("changed composition revision")
	}
	a, b := before["input"].(map[string]any), after["input"].(map[string]any)
	be, ae := before["evidence_evaluation"].(map[string]any)["graph_evaluation"].(map[string]any), after["evidence_evaluation"].(map[string]any)["graph_evaluation"].(map[string]any)
	evidence := false
	for _, key := range []string{"interpretations", "additional_knowledge", "provider_packs"} {
		evidence = evidence || !scvEqual(a[key], b[key])
	}
	axes := map[string]any{"evidence": evidence, "policy": !scvEqual(be["selection_policy"], ae["selection_policy"]), "requirements": !scvEqual(a["requirements"], b["requirements"]) || !scvEqual(a["counterfactuals"], b["counterfactuals"]), "query_time": !scvEqual(a["query_time"], b["query_time"]), "recipes": !scvEqual(compRecipeProjection(a, false), compRecipeProjection(b, false)), "selections": !scvEqual(compRecipeProjection(a, true), compRecipeProjection(b, true)) || !scvEqual(a["allowed_guarantee_changes"], b["allowed_guarantee_changes"]), "bounds": !scvEqual(a["bounds"], b["bounds"])}
	if !scvEqual(result["change_axes"], axes) || result["search_changed"] != (!scvEqual(before["search"], after["search"])) {
		return fmt.Errorf("composition change axes mismatch")
	}
	old, current := compResultCandidates(before), compResultCandidates(after)
	oldFindings, _ := scvFindingMap(be)
	newFindings, _ := scvFindingMap(ae)
	keys := map[string]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range current {
		keys[k] = true
	}
	ordered := []string{}
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)
	changes := []any{}
	for _, key := range ordered {
		x, y := old[key], current[key]
		changed := !scvEqual(x, y)
		affected := changed
		refs := map[string]bool{}
		for _, c := range []map[string]any{x, y} {
			if c != nil {
				for _, rawID := range c["claim_ids"].([]any) {
					previous, current := map[string]bool{}, map[string]bool{}
					scvFindingClosure(oldFindings, rawID.(string), previous)
					scvFindingClosure(newFindings, rawID.(string), current)
					for id := range previous {
						refs[id] = true
					}
					for id := range current {
						refs[id] = true
					}
				}
			}
		}
		for id := range refs {
			if !scvEqual(oldFindings[id], newFindings[id]) {
				affected = true
			}
		}
		var identity []any
		if err := json.Unmarshal([]byte(key), &identity); err != nil {
			return err
		}
		var bs, as any
		if x != nil {
			bs = x["status"]
		}
		if y != nil {
			as = y["status"]
		}
		changes = append(changes, map[string]any{"scenario_id": identity[0], "candidate_id": identity[1], "before_status": bs, "after_status": as, "changed": changed, "affected": affected})
	}
	if !scvEqual(changes, result["candidates"]) {
		return fmt.Errorf("composition changed candidate attribution")
	}
	if _, e := compStrings(result["limitations"], 16, 1); e != nil {
		return e
	}
	return nil
}

package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The consumer checks seals, exact input binding and observable comparison
// results independently, including finite token extraction and retained anchors.
// Semantic interpretation and graph support remain C++ owner responsibilities;
// retained interpretation/evaluation artifacts are replayed there.
func validateSCVInterpretationResult(operation string, input, result map[string]any) error {
	fields := map[string][]string{
		"provider_interpret":  {"protocol", "domain", "profiles", "bindings", "knowledge", "extractions", "limitations", "digest"},
		"connection_evaluate": {"protocol", "domain", "input", "graph_digest", "graph_evaluation", "connections", "limitations", "digest"},
		"connection_reassess": {"protocol", "domain", "before_digest", "after_digest", "change_axes", "checks", "limitations", "digest"},
	}
	if !scvCorpusFields(result, fields[operation]...) {
		return fmt.Errorf("unexpected interpretation result fields")
	}
	domain, domainOK := result["domain"].(string)
	allowedDomain := false
	if domainOK {
		for _, d := range SCVDomains() {
			if domain == d {
				allowedDomain = true
			}
		}
	}
	if !allowedDomain {
		return fmt.Errorf("invalid interpretation result domain")
	}
	if err := scvSeal(result, "digest"); err != nil {
		return err
	}
	limits, ok := result["limitations"].([]any)
	if !ok || len(limits) == 0 {
		return fmt.Errorf("interpretation result lacks bounded limitations")
	}
	switch operation {
	case "provider_interpret":
		knowledge, ok := result["knowledge"].(map[string]any)
		if !ok || knowledge["protocol"] != "symphony.scv.knowledge.v1" || !scvTextEqual(knowledge["domain"], result["domain"]) {
			return fmt.Errorf("interpretation knowledge identity mismatch")
		}
		if err := scvSeal(knowledge, "digest"); err != nil {
			return err
		}
		for _, key := range []string{"profiles", "bindings"} {
			if !scvUnorderedEqual(input[key], result[key]) {
				return fmt.Errorf("interpretation changed %s", key)
			}
		}
		if !scvUnorderedEqual(input["captures"], knowledge["captures"]) || !scvPolicyEqual(input["selection_policy"], knowledge["selection_policy"]) {
			return fmt.Errorf("interpretation changed captured evidence or policy")
		}
		return scvValidateExtractions(result, knowledge)
	case "connection_evaluate":
		if !scvEqual(input, result["input"]) {
			return fmt.Errorf("connection evaluation changed selected input")
		}
		evaluation, ok := result["graph_evaluation"].(map[string]any)
		if !ok || evaluation["protocol"] != "symphony.scv.evaluate-result.v1" || !scvTextEqual(evaluation["domain"], result["domain"]) || !scvTextEqual(evaluation["graph_digest"], result["graph_digest"]) || !scvTextEqual(evaluation["query_time"], input["query_time"]) {
			return fmt.Errorf("connection graph evaluation binding mismatch")
		}
		if err := scvSeal(evaluation, "digest"); err != nil {
			return err
		}
		findings, err := scvFindingMap(evaluation)
		if err != nil {
			return err
		}
		if digest, ok := result["graph_digest"].(string); !ok || !taggedSHA256(digest) {
			return fmt.Errorf("invalid graph identity")
		}
		if _, err := scvCorpusTime(input["query_time"]); err != nil {
			return err
		}
		if err := scvBindFindingClaims(input, evaluation, findings); err != nil {
			return err
		}
		return scvValidateConnections(input["connections"], result["connections"], findings)
	case "connection_reassess":
		before, bok := input["before"].(map[string]any)
		after, aok := input["after"].(map[string]any)
		if !bok || !aok || !scvTextEqual(result["before_digest"], before["digest"]) || !scvTextEqual(result["after_digest"], after["digest"]) {
			return fmt.Errorf("connection reassessment revision binding mismatch")
		}
		for _, v := range []map[string]any{before, after} {
			if v["protocol"] != "symphony.scv.connection-evaluation.v1" || !scvTextEqual(v["domain"], result["domain"]) {
				return fmt.Errorf("connection reassessment owner mismatch")
			}
			submitted, ok := v["input"].(map[string]any)
			if !ok {
				return fmt.Errorf("missing evaluation input")
			}
			if err := validateSCVInterpretationResult("connection_evaluate", submitted, v); err != nil {
				return err
			}
		}
		return scvValidateReassessment(before, after, result)
	}
	return fmt.Errorf("unsupported interpretation operation")
}

func scvTextEqual(a, b any) bool {
	x, xok := a.(string)
	y, yok := b.(string)
	return xok && yok && len(x) > 0 && len(x) <= 512 && x == y && !strings.ContainsRune(x, 0)
}
func scvValidTypedValue(value map[string]any) bool {
	if !scvCorpusFields(value, "type", "value", "unit") {
		return false
	}
	if value["unit"] != nil {
		u, ok := value["unit"].(string)
		if !ok || u == "" || len(u) > 128 || strings.ContainsRune(u, 0) {
			return false
		}
	}
	switch value["type"] {
	case "integer":
		n, ok := value["value"].(json.Number)
		if !ok {
			return false
		}
		i, err := n.Int64()
		return err == nil && i >= -9007199254740991 && i <= 9007199254740991 && strconv.FormatInt(i, 10) == n.String()
	case "decimal":
		s, ok := value["value"].(string)
		return ok && len(s) <= 128 && s != "-0" && scvCanonicalDecimal.MatchString(s)
	case "boolean":
		_, ok := value["value"].(bool)
		return ok
	case "string", "reference":
		s, ok := value["value"].(string)
		return ok && s != "" && len(s) <= 4096 && !strings.ContainsRune(s, 0)
	}
	return false
}
func scvBindFindingClaims(input, evaluation map[string]any, findings map[string]map[string]any) error {
	knowledge := []map[string]any{}
	wrappers, ok := input["interpretations"].([]any)
	if !ok {
		return fmt.Errorf("missing selected interpretation wrappers")
	}
	for _, raw := range wrappers {
		w, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid selected interpretation")
		}
		k, ok := w["knowledge"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing retained knowledge")
		}
		replayInput := map[string]any{"captures": k["captures"], "profiles": w["profiles"], "bindings": w["bindings"], "selection_policy": k["selection_policy"]}
		if err := validateSCVInterpretationResult("provider_interpret", replayInput, w); err != nil {
			return err
		}
		knowledge = append(knowledge, k)
	}
	additional, ok := input["additional_knowledge"].([]any)
	if !ok {
		return fmt.Errorf("missing selected additional knowledge")
	}
	for _, raw := range additional {
		k, ok := raw.(map[string]any)
		if !ok || k["protocol"] != "symphony.scv.knowledge.v1" {
			return fmt.Errorf("invalid additional knowledge")
		}
		if err := scvSeal(k, "digest"); err != nil {
			return err
		}
		knowledge = append(knowledge, k)
	}
	expected := map[string]map[string]any{}
	for _, k := range knowledge {
		if !scvEqual(k["selection_policy"], evaluation["selection_policy"]) {
			return fmt.Errorf("graph evaluation changed selected knowledge policy")
		}
		claims, err := scvMapList(k["claims"], "claim_id")
		if err != nil {
			return err
		}
		for id, claim := range claims {
			value, ok := claim["value"].(map[string]any)
			if !ok || !scvValidTypedValue(value) {
				return fmt.Errorf("invalid selected typed claim")
			}
			if prior := expected[id]; prior != nil && !scvEqual(prior, claim) {
				return fmt.Errorf("selected claim identity collision")
			}
			expected[id] = claim
		}
	}
	if len(expected) != len(findings) {
		return fmt.Errorf("graph evaluation changed selected claim coverage")
	}
	for id, claim := range expected {
		f := findings[id]
		if f == nil || !scvEqual(f["claim"], claim) {
			return fmt.Errorf("graph evaluation changed selected claim content")
		}
	}
	return nil
}
func scvUnorderedEqual(a, b any) bool {
	x, xok := a.([]any)
	y, yok := b.([]any)
	if !xok || !yok || len(x) != len(y) {
		return false
	}
	encode := func(values []any) []string {
		out := []string{}
		for _, v := range values {
			raw, err := SCVCanonical(v)
			if err != nil {
				return nil
			}
			out = append(out, string(raw))
		}
		sort.Strings(out)
		return out
	}
	return scvEqual(encode(x), encode(y))
}
func scvPolicyEqual(a, b any) bool {
	x, xok := a.(map[string]any)
	y, yok := b.(map[string]any)
	if !xok || !yok || len(x) != len(y) {
		return false
	}
	for k, v := range x {
		if k == "allowed_statement_kinds" {
			if !scvUnorderedEqual(v, y[k]) {
				return false
			}
		} else if !scvEqual(v, y[k]) {
			return false
		}
	}
	return true
}
func scvMapList(raw any, key string) (map[string]map[string]any, error) {
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("expected %s array", key)
	}
	out := map[string]map[string]any{}
	for _, v := range values {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid %s object", key)
		}
		id, ok := m[key].(string)
		if !ok || id == "" || out[id] != nil {
			return nil, fmt.Errorf("invalid or duplicate %s", key)
		}
		out[id] = m
	}
	return out, nil
}

var scvCanonicalDecimal = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]*[1-9])?$`)

func scvOccurrences(body, needle string) int {
	if needle == "" {
		return 0
	}
	count := 0
	for start := 0; start < len(body); {
		offset := strings.Index(body[start:], needle)
		if offset < 0 {
			break
		}
		count++
		if count == 2 {
			return count
		}
		start += offset + 1
	}
	return count
}
func scvExtractedValue(rule, capture, profile, policy map[string]any) (map[string]any, bool) {
	if capture["completeness"] == "failed" || (capture["completeness"] == "partial" && policy["partial_capture"] == "exclude") {
		return nil, false
	}
	media, _ := profile["media_types"].([]any)
	supported := false
	for _, m := range media {
		if scvTextEqual(m, capture["media_type"]) {
			supported = true
		}
	}
	if !supported {
		return nil, false
	}
	body, _ := capture["body"].(string)
	contexts, _ := rule["context"].([]any)
	for _, raw := range contexts {
		anchor, ok := raw.(string)
		if !ok || scvOccurrences(body, anchor) != 1 {
			return nil, false
		}
	}
	ex, ok := rule["extractor"].(map[string]any)
	if !ok {
		return nil, false
	}
	if ex["kind"] == "literal" {
		quote, ok := ex["quote"].(string)
		value, vok := ex["value"].(map[string]any)
		return value, ok && vok && scvOccurrences(body, quote) == 1
	}
	prefix, pok := ex["prefix"].(string)
	suffix, sok := ex["suffix"].(string)
	if ex["kind"] != "delimited" || !pok || !sok || prefix == "" || suffix == "" {
		return nil, false
	}
	count, begin, end := 0, 0, 0
	for cursor := 0; cursor < len(body); {
		i := strings.Index(body[cursor:], prefix)
		if i < 0 {
			break
		}
		start := cursor + i
		close := strings.Index(body[start+len(prefix):], suffix)
		if close >= 0 {
			begin = start
			end = start + len(prefix) + close
			count++
			if count > 1 {
				return nil, false
			}
		}
		cursor = start + 1
	}
	if count != 1 || end+len(suffix)-begin > 4096 {
		return nil, false
	}
	token := body[begin+len(prefix) : end]
	if len(token) == 0 || strings.ContainsRune(token, 0) {
		return nil, false
	}
	var value any = token
	switch ex["type"] {
	case "integer":
		n, err := strconv.ParseInt(token, 10, 64)
		if err != nil || n < -9007199254740991 || n > 9007199254740991 || strconv.FormatInt(n, 10) != token {
			return nil, false
		}
		value = json.Number(token)
	case "decimal":
		if len(token) > 128 || token == "-0" || !scvCanonicalDecimal.MatchString(token) {
			return nil, false
		}
	case "string":
	default:
		return nil, false
	}
	return map[string]any{"type": ex["type"], "value": value, "unit": ex["unit"]}, true
}
func scvValidateExtractions(result, knowledge map[string]any) error {
	profiles, err := scvMapList(result["profiles"], "digest")
	if err != nil {
		return err
	}
	captures, err := scvMapList(knowledge["captures"], "digest")
	if err != nil {
		return err
	}
	claims, err := scvMapList(knowledge["claims"], "claim_id")
	if err != nil {
		return err
	}
	bindings, ok := result["bindings"].([]any)
	if !ok {
		return fmt.Errorf("missing profile bindings")
	}
	expected := map[string]map[string]any{}
	bound := map[string]string{}
	for _, raw := range bindings {
		b, ok := raw.(map[string]any)
		if !ok || !scvCorpusFields(b, "profile_digest", "capture_digest") {
			return fmt.Errorf("invalid profile binding")
		}
		pd, pok := b["profile_digest"].(string)
		cd, cok := b["capture_digest"].(string)
		p, c := profiles[pd], captures[cd]
		if !pok || !cok || p == nil || c == nil || bound[pd] != "" {
			return fmt.Errorf("profile binding does not resolve exactly")
		}
		bound[pd] = cd
		if err := scvSeal(p, "digest"); err != nil {
			return err
		}
		if err := scvSeal(c, "digest"); err != nil {
			return err
		}
		source, sok := c["source"].(map[string]any)
		if !sok || !scvTextEqual(source["source_id"], p["source_id"]) || !scvTextEqual(source["provider_id"], p["provider_id"]) || !scvTextEqual(c["locator_id"], p["locator_id"]) {
			return fmt.Errorf("profile source identity mismatch")
		}
		rules, err := scvMapList(p["rules"], "rule_id")
		if err != nil {
			return err
		}
		for id, rule := range rules {
			expected[pd+"\x00"+id] = rule
		}
	}
	extractions, ok := result["extractions"].([]any)
	if !ok || len(extractions) != len(expected) {
		return fmt.Errorf("extraction coverage mismatch")
	}
	seen := map[string]bool{}
	matched := map[string]bool{}
	for _, raw := range extractions {
		e, ok := raw.(map[string]any)
		if !ok || !scvCorpusFields(e, "profile_digest", "capture_digest", "rule_id", "claim_id", "status", "reasons") {
			return fmt.Errorf("invalid extraction fields")
		}
		pd, pok := e["profile_digest"].(string)
		rid, rok := e["rule_id"].(string)
		key := pd + "\x00" + rid
		rule := expected[key]
		if !pok || !rok || rule == nil || seen[key] || e["capture_digest"] != bound[pd] || !scvTextEqual(e["claim_id"], rule["claim_id"]) {
			return fmt.Errorf("extraction changed selected rule identity")
		}
		seen[key] = true
		reasons, ok := e["reasons"].([]any)
		if !ok {
			return fmt.Errorf("extraction reasons missing")
		}
		id, _ := e["claim_id"].(string)
		policy, _ := knowledge["selection_policy"].(map[string]any)
		expectedValue, extractionMatched := scvExtractedValue(rule, captures[bound[pd]], profiles[pd], policy)
		if e["status"] == "unresolved" {
			if extractionMatched {
				return fmt.Errorf("matching extraction was reported unresolved")
			}
			if len(reasons) == 0 || claims[id] != nil {
				return fmt.Errorf("unresolved extraction became a claim")
			}
			continue
		}
		claim := claims[id]
		if !extractionMatched || claim == nil || !scvEqual(claim["value"], expectedValue) {
			return fmt.Errorf("extracted value does not match selected source token")
		}
		if e["status"] != "matched" || claim == nil || matched[id] {
			return fmt.Errorf("extraction status/claim mismatch")
		}
		if !scvValidTypedValue(expectedValue) {
			return fmt.Errorf("invalid extracted typed value")
		}
		anchors := map[string]bool{}
		contexts, _ := rule["context"].([]any)
		for _, raw := range contexts {
			anchor, ok := raw.(string)
			if !ok {
				return fmt.Errorf("invalid context anchor")
			}
			anchors[anchor] = true
		}
		selectedExtractor, _ := rule["extractor"].(map[string]any)
		if selectedExtractor["kind"] == "literal" {
			quote, _ := selectedExtractor["quote"].(string)
			anchors[quote] = true
		} else {
			prefix, _ := selectedExtractor["prefix"].(string)
			suffix, _ := selectedExtractor["suffix"].(string)
			anchors[prefix+fmt.Sprint(expectedValue["value"])+suffix] = true
		}
		expectedEvidence := []any{}
		for quote := range anchors {
			expectedEvidence = append(expectedEvidence, map[string]any{"capture_digest": bound[pd], "quote": quote})
		}
		if !scvUnorderedEqual(expectedEvidence, claim["evidence"]) {
			return fmt.Errorf("extracted claim changed complete context/token evidence anchors")
		}
		matched[id] = true
		for _, field := range []string{"claim_id", "subject", "predicate", "scope", "statement_kind"} {
			if !scvEqual(claim[field], rule[field]) {
				return fmt.Errorf("extraction changed claim %s", field)
			}
		}
		if !scvUnorderedEqual(claim["dependencies"], rule["dependencies"]) {
			return fmt.Errorf("extraction changed claim dependencies")
		}
		evidence, ok := claim["evidence"].([]any)
		if !ok || len(evidence) == 0 {
			return fmt.Errorf("extracted claim lacks evidence")
		}
		capture := captures[bound[pd]]
		body, _ := capture["body"].(string)
		for _, v := range evidence {
			a, ok := v.(map[string]any)
			if !ok || a["capture_digest"] != bound[pd] {
				return fmt.Errorf("extracted claim changed capture")
			}
			quote, ok := a["quote"].(string)
			if !ok || quote == "" || !strings.Contains(body, quote) {
				return fmt.Errorf("extracted evidence absent from capture")
			}
		}
		extractor, ok := rule["extractor"].(map[string]any)
		if !ok {
			return fmt.Errorf("invalid selected extractor")
		}
		if extractor["kind"] == "literal" && !scvEqual(extractor["value"], claim["value"]) {
			return fmt.Errorf("literal extraction changed selected value")
		}
	}
	if len(matched) != len(claims) {
		return fmt.Errorf("interpretation introduced unselected claims")
	}
	return nil
}
func scvFindingMap(evaluation map[string]any) (map[string]map[string]any, error) {
	values, ok := evaluation["findings"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing graph findings")
	}
	out := map[string]map[string]any{}
	for _, raw := range values {
		f, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid finding")
		}
		c, ok := f["claim"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("missing finding claim")
		}
		id, ok := c["claim_id"].(string)
		if !ok || out[id] != nil {
			return nil, fmt.Errorf("duplicate/missing finding claim")
		}
		out[id] = f
	}
	return out, nil
}
func scvOperand(raw map[string]any, findings map[string]map[string]any) (map[string]any, bool, bool, []string, error) {
	id, ok := raw["claim_id"].(string)
	if !ok || id == "" {
		return nil, false, false, nil, fmt.Errorf("invalid operand claim identity")
	}
	f := findings[id]
	if f == nil {
		return nil, false, false, []string{"claim_missing:" + id}, nil
	}
	c, ok := f["claim"].(map[string]any)
	if !ok {
		return nil, false, false, nil, fmt.Errorf("invalid operand finding")
	}
	value, ok := c["value"].(map[string]any)
	if !ok || !scvValidTypedValue(value) {
		return nil, false, false, nil, fmt.Errorf("invalid operand typed value")
	}
	status, ok := f["status"].(string)
	if !ok {
		return nil, false, false, nil, fmt.Errorf("invalid operand finding status")
	}
	reasons := []string{}
	eligible := true
	if !scvEqual(c["subject"], raw["subject"]) {
		eligible = false
		reasons = append(reasons, "subject_mismatch:"+id)
	}
	if !scvEqual(c["scope"], raw["scope"]) {
		eligible = false
		reasons = append(reasons, "scope_mismatch:"+id)
	}
	if status != "supported" && status != "conditional" {
		eligible = false
		reasons = append(reasons, "claim_not_eligible:"+id+":"+status)
	}
	qualified := status == "conditional" || (c["statement_kind"] != "documented_fact" && c["statement_kind"] != "requirement")
	if qualified {
		reasons = append(reasons, "conditional_evidence:"+id)
	}
	// Retain the typed value even when ineligible: type/unit diagnostics are
	// independent of scope, freshness and conditional-support qualification.
	return value, qualified, eligible, reasons, nil
}
func scvComparison(left, right map[string]any, operator string) (bool, bool) {
	if !scvValidTypedValue(left) || !scvValidTypedValue(right) || !scvEqual(left["type"], right["type"]) || !scvEqual(left["unit"], right["unit"]) {
		return false, false
	}
	if operator == "eq" {
		return scvEqual(left["value"], right["value"]), true
	}
	if operator != "gte" && operator != "lte" {
		return false, false
	}
	if left["type"] != "integer" && left["type"] != "decimal" {
		return false, false
	}
	a, aok := new(big.Rat).SetString(fmt.Sprint(left["value"]))
	b, bok := new(big.Rat).SetString(fmt.Sprint(right["value"]))
	if !aok || !bok {
		return false, false
	}
	if operator == "gte" {
		return a.Cmp(b) >= 0, true
	}
	return a.Cmp(b) <= 0, true
}
func scvCheckOutcome(spec map[string]any, findings map[string]map[string]any) (string, any, []string, []string, error) {
	left, lok := spec["left"].(map[string]any)
	right, rok := spec["right"].(map[string]any)
	if !lok || !rok {
		return "", nil, nil, nil, fmt.Errorf("invalid check operands")
	}
	op, ok := spec["operator"].(string)
	if !ok || (op != "eq" && op != "gte" && op != "lte") {
		return "", nil, nil, nil, fmt.Errorf("invalid check comparison operator")
	}
	lv, lq, lu, lr, err := scvOperand(left, findings)
	if err != nil {
		return "", nil, nil, nil, err
	}
	ids := map[string]bool{left["claim_id"].(string): true}
	reasons := map[string]bool{}
	for _, reason := range lr {
		reasons[reason] = true
	}
	var rv map[string]any
	rq, ru := false, true
	if right["kind"] == "claim" {
		var rr []string
		rv, rq, ru, rr, err = scvOperand(right, findings)
		if err != nil {
			return "", nil, nil, nil, err
		}
		ids[right["claim_id"].(string)] = true
		for _, reason := range rr {
			reasons[reason] = true
		}
	} else if right["kind"] == "literal" {
		rv, ok = right["value"].(map[string]any)
		if !ok || !scvValidTypedValue(rv) {
			return "", nil, nil, nil, fmt.Errorf("invalid typed literal")
		}
	} else {
		return "", nil, nil, nil, fmt.Errorf("invalid right operand kind")
	}
	unresolved := !lu || !ru
	var comparison any
	if lv != nil && rv != nil {
		if !scvEqual(lv["type"], rv["type"]) {
			unresolved = true
			reasons["value_type_mismatch"] = true
		} else if !scvEqual(lv["unit"], rv["unit"]) {
			unresolved = true
			reasons["unit_mismatch"] = true
		} else if op != "eq" && lv["type"] != "integer" && lv["type"] != "decimal" {
			unresolved = true
			reasons["ordering_requires_numeric_type"] = true
		}
		if !unresolved {
			matched, valid := scvComparison(lv, rv, op)
			if !valid {
				return "", nil, nil, nil, fmt.Errorf("invalid typed comparison")
			}
			comparison = matched
			if matched {
				reasons["comparison_matched"] = true
			} else {
				reasons["comparison_not_matched"] = true
			}
		}
	}
	status := "contradicted"
	if unresolved {
		status = "unresolved"
	} else if lq || rq {
		status = "conditional"
	} else if comparison == true {
		status = "satisfied"
	}
	sortedKeys := func(values map[string]bool) []string {
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return keys
	}
	return status, comparison, sortedKeys(ids), sortedKeys(reasons), nil
}
func scvValidateConnections(input, output any, findings map[string]map[string]any) error {
	specs, err := scvMapList(input, "connection_id")
	if err != nil {
		return err
	}
	results, err := scvMapList(output, "connection_id")
	if err != nil {
		return err
	}
	if len(specs) != len(results) {
		return fmt.Errorf("connection coverage mismatch")
	}
	for id, spec := range specs {
		result := results[id]
		if result == nil || !scvCorpusFields(result, "connection_id", "from_subject", "to_subject", "status", "checks") || !scvTextEqual(result["from_subject"], spec["from_subject"]) || !scvTextEqual(result["to_subject"], spec["to_subject"]) {
			return fmt.Errorf("connection identity mismatch")
		}
		checks, err := scvMapList(spec["checks"], "check_id")
		if err != nil {
			return err
		}
		out, ok := result["checks"].([]any)
		if !ok || len(out) != len(checks) {
			return fmt.Errorf("check coverage mismatch")
		}
		seen := map[string]bool{}
		counts := map[string]int{}
		required := 0
		for _, raw := range out {
			c, ok := raw.(map[string]any)
			if !ok || !scvCorpusFields(c, "specification", "status", "reasons", "claim_ids", "comparison") {
				return fmt.Errorf("invalid connection check")
			}
			s, ok := c["specification"].(map[string]any)
			if !ok {
				return fmt.Errorf("missing check specification")
			}
			cid, _ := s["check_id"].(string)
			if checks[cid] == nil || seen[cid] || !scvEqual(s, checks[cid]) {
				return fmt.Errorf("check input binding mismatch")
			}
			seen[cid] = true
			status, comparison, ids, reasons, err := scvCheckOutcome(s, findings)
			if err != nil {
				return err
			}
			if c["status"] != status || !scvEqual(c["comparison"], comparison) || !scvEqual(c["claim_ids"], ids) {
				return fmt.Errorf("connection comparison/status mismatch for %s", cid)
			}
			if !scvEqual(c["reasons"], reasons) {
				return fmt.Errorf("connection reason inventory mismatch for %s", cid)
			}
			if s["importance"] == "required" {
				required++
				counts[status]++
			}
		}
		status := "satisfied"
		if counts["contradicted"] > 0 {
			status = "contradicted"
		} else if counts["unresolved"] > 0 || required == 0 {
			status = "unresolved"
		} else if counts["conditional"] > 0 {
			status = "conditional"
		}
		if result["status"] != status {
			return fmt.Errorf("connection aggregation changed required/optional policy")
		}
	}
	return nil
}
func scvCheckMap(evaluation map[string]any) (map[string]map[string]any, error) {
	connections, err := scvMapList(evaluation["connections"], "connection_id")
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]any{}
	for cid, c := range connections {
		checks, ok := c["checks"].([]any)
		if !ok {
			return nil, fmt.Errorf("missing checks")
		}
		for _, raw := range checks {
			r, ok := raw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid check")
			}
			s, ok := r["specification"].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid check specification")
			}
			id, ok := s["check_id"].(string)
			if !ok {
				return nil, fmt.Errorf("missing check identity")
			}
			out[cid+"\x00"+id] = r
		}
	}
	return out, nil
}
func scvFindingClosure(findings map[string]map[string]any, id string, visited map[string]bool) {
	if visited[id] {
		return
	}
	visited[id] = true
	finding := findings[id]
	if finding == nil {
		return
	}
	claim, _ := finding["claim"].(map[string]any)
	visit := func(raw any) {
		deps, _ := raw.([]any)
		for _, rawDep := range deps {
			dep, _ := rawDep.(map[string]any)
			child, _ := dep["claim_id"].(string)
			if child != "" {
				scvFindingClosure(findings, child, visited)
			}
		}
	}
	visit(claim["dependencies"])
	supports, _ := claim["alternative_supports"].([]any)
	for _, raw := range supports {
		support, _ := raw.(map[string]any)
		visit(support["dependencies"])
	}
}
func scvRequirements(raw any) any {
	connections, ok := raw.([]any)
	if !ok {
		return nil
	}
	output := []any{}
	for _, raw := range connections {
		c, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		checks, ok := c["checks"].([]any)
		if !ok {
			return nil
		}
		sorted := append([]any{}, checks...)
		sort.Slice(sorted, func(i, j int) bool {
			a, _ := SCVCanonical(sorted[i])
			b, _ := SCVCanonical(sorted[j])
			return string(a) < string(b)
		})
		copy := map[string]any{}
		for k, v := range c {
			copy[k] = v
		}
		copy["checks"] = sorted
		output = append(output, copy)
	}
	sort.Slice(output, func(i, j int) bool {
		a, _ := SCVCanonical(output[i])
		b, _ := SCVCanonical(output[j])
		return string(a) < string(b)
	})
	return output
}
func scvInterpretationSelection(input map[string]any) (any, any, any, error) {
	captures, profiles, additional := map[string]bool{}, map[string]bool{}, map[string]bool{}
	bindings := []any{}
	captureSet := func(knowledge map[string]any) error {
		list, ok := knowledge["captures"].([]any)
		if !ok {
			return fmt.Errorf("missing selected captures")
		}
		for _, raw := range list {
			c, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid retained capture")
			}
			id, ok := c["digest"].(string)
			if !ok {
				return fmt.Errorf("missing capture identity")
			}
			captures[id] = true
		}
		return nil
	}
	wrappers, ok := input["interpretations"].([]any)
	if !ok {
		return nil, nil, nil, fmt.Errorf("missing selected interpretations")
	}
	for _, raw := range wrappers {
		w, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, nil, fmt.Errorf("invalid interpretation wrapper")
		}
		knowledge, ok := w["knowledge"].(map[string]any)
		if !ok {
			return nil, nil, nil, fmt.Errorf("missing knowledge")
		}
		if err := captureSet(knowledge); err != nil {
			return nil, nil, nil, err
		}
		ps, err := scvMapList(w["profiles"], "digest")
		if err != nil {
			return nil, nil, nil, err
		}
		for id := range ps {
			profiles[id] = true
		}
		bs, ok := w["bindings"].([]any)
		if !ok {
			return nil, nil, nil, fmt.Errorf("missing selected bindings")
		}
		for _, raw := range bs {
			b, ok := raw.(map[string]any)
			if !ok {
				return nil, nil, nil, fmt.Errorf("invalid binding")
			}
			pd, _ := b["profile_digest"].(string)
			profile := ps[pd]
			if profile == nil {
				return nil, nil, nil, fmt.Errorf("unresolved profile binding")
			}
			bindings = append(bindings, map[string]any{"profile_id": profile["profile_id"], "capture_digest": b["capture_digest"]})
		}
	}
	extras, ok := input["additional_knowledge"].([]any)
	if !ok {
		return nil, nil, nil, fmt.Errorf("missing additional knowledge")
	}
	for _, raw := range extras {
		k, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, nil, fmt.Errorf("invalid additional knowledge")
		}
		if err := captureSet(k); err != nil {
			return nil, nil, nil, err
		}
		id, _ := k["digest"].(string)
		additional[id] = true
	}
	keys := func(m map[string]bool) []string {
		out := []string{}
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	sort.Slice(bindings, func(i, j int) bool {
		a, _ := SCVCanonical(bindings[i])
		b, _ := SCVCanonical(bindings[j])
		return string(a) < string(b)
	})
	return map[string]any{"captures": keys(captures), "bindings": bindings}, keys(profiles), keys(additional), nil
}
func scvValidateReassessment(before, after, result map[string]any) error {
	old, err := scvCheckMap(before)
	if err != nil {
		return err
	}
	current, err := scvCheckMap(after)
	if err != nil {
		return err
	}
	bf, _ := before["graph_evaluation"].(map[string]any)
	af, _ := after["graph_evaluation"].(map[string]any)
	oldFindings, err := scvFindingMap(bf)
	if err != nil {
		return err
	}
	newFindings, err := scvFindingMap(af)
	if err != nil {
		return err
	}
	keys := map[string]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range current {
		keys[k] = true
	}
	checks, ok := result["checks"].([]any)
	if !ok || len(checks) != len(keys) {
		return fmt.Errorf("reassessment check coverage mismatch")
	}
	seen := map[string]bool{}
	for _, raw := range checks {
		c, ok := raw.(map[string]any)
		if !ok || !scvCorpusFields(c, "connection_id", "check_id", "before_status", "after_status", "changed", "affected") {
			return fmt.Errorf("invalid reassessment check")
		}
		cid, cok := c["connection_id"].(string)
		id, iok := c["check_id"].(string)
		key := cid + "\x00" + id
		if !cok || !iok || !keys[key] || seen[key] {
			return fmt.Errorf("reassessment check identity mismatch")
		}
		seen[key] = true
		a, b := old[key], current[key]
		var oldStatus, newStatus any
		if a != nil {
			oldStatus = a["status"]
		}
		if b != nil {
			newStatus = b["status"]
		}
		changed := !scvEqual(a, b)
		affected := changed
		for _, check := range []map[string]any{a, b} {
			if check == nil {
				continue
			}
			ids, ok := check["claim_ids"].([]any)
			if !ok {
				return fmt.Errorf("missing reassessment claim IDs")
			}
			for _, rawID := range ids {
				claimID, ok := rawID.(string)
				if !ok {
					return fmt.Errorf("invalid reassessment claim ID")
				}
				oldClosure, newClosure := map[string]bool{}, map[string]bool{}
				scvFindingClosure(oldFindings, claimID, oldClosure)
				scvFindingClosure(newFindings, claimID, newClosure)
				for id := range newClosure {
					oldClosure[id] = true
				}
				for id := range oldClosure {
					if !scvEqual(oldFindings[id], newFindings[id]) {
						affected = true
					}
				}
			}
		}
		if !scvEqual(c["before_status"], oldStatus) || !scvEqual(c["after_status"], newStatus) || c["changed"] != changed || c["affected"] != affected {
			return fmt.Errorf("reassessment changed exact finding attribution")
		}
	}
	axes, ok := result["change_axes"].(map[string]any)
	if !ok || !scvCorpusFields(axes, "captures", "profiles", "additional_knowledge", "selection_policy", "requirements", "query_time") {
		return fmt.Errorf("invalid reassessment axes")
	}
	for _, v := range axes {
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("invalid change axis")
		}
	}
	bi, _ := before["input"].(map[string]any)
	ai, _ := after["input"].(map[string]any)
	bc, bp, ba, err := scvInterpretationSelection(bi)
	if err != nil {
		return err
	}
	ac, ap, aa, err := scvInterpretationSelection(ai)
	if err != nil {
		return err
	}
	if axes["query_time"] != (!scvEqual(bi["query_time"], ai["query_time"])) || axes["requirements"] != (!scvEqual(scvRequirements(bi["connections"]), scvRequirements(ai["connections"]))) ||
		axes["additional_knowledge"] != (!scvEqual(ba, aa)) || axes["captures"] != (!scvEqual(bc, ac)) || axes["profiles"] != (!scvEqual(bp, ap)) || axes["selection_policy"] != (!scvEqual(bf["selection_policy"], af["selection_policy"])) {
		return fmt.Errorf("reassessment input axes mismatch")
	}
	return nil
}

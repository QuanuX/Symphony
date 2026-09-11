package knowledgeengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// The consumer independently checks selected source bytes, typed extraction,
// full claim/evidence attribution, fixture expectations and conformance counts.
// Native structural graph interpretation remains subject to C++ exact replay.
func validateSCVProviderPack(operation string, input, result map[string]any) error {
	if err := scvSeal(result, "digest"); err != nil {
		return err
	}
	if operation == "provider_pack_prepare" {
		if !scvCorpusFields(input, "pack", "fixtures") {
			return fmt.Errorf("invalid pack preparation input")
		}
		draft, ok := input["pack"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing authored pack draft")
		}
		if !scvPackDraftFields(draft) {
			return fmt.Errorf("invalid pack draft fields")
		}
		expected := scvPackCopy(draft)
		manifests, err := scvPackMapArray(expected["fixtures"], "fixture_id", 16)
		if err != nil {
			return err
		}
		fixtures, err := scvPackFixtures(input["fixtures"])
		if err != nil {
			return err
		}
		for id, f := range fixtures {
			m := manifests[id]
			if m == nil {
				return fmt.Errorf("undeclared fixture input")
			}
			dg, err := SCVDigest(f)
			if err != nil {
				return err
			}
			if m["input_digest"] != nil && m["input_digest"] != dg {
				return fmt.Errorf("fixture input differs from authored digest")
			}
			m["input_digest"] = dg
		}
		for _, m := range manifests {
			if dg, ok := m["input_digest"].(string); !ok || !taggedSHA256(dg) {
				return fmt.Errorf("fixture requires input or exact digest")
			}
		}
		expected["digest"], err = SCVDigest(expected)
		if err != nil {
			return err
		}
		if !scvEqual(expected, result) {
			return fmt.Errorf("preparation changed authored pack content")
		}
		ps, err := scvPackProfiles(result)
		if err != nil {
			return err
		}
		for _, f := range fixtures {
			if _, _, err := scvPackSelection(result, ps, f, "scv"); err != nil {
				return err
			}
		}
		return nil
	}
	if operation != "provider_pack_evaluate" || !scvCorpusFields(input, "pack", "captures", "bindings", "selection_policy", "fixtures") || !scvCorpusFields(result, "protocol", "domain", "input", "knowledge", "extractions", "fixture_results", "conformance", "limitations", "digest") || result["protocol"] != "symphony.scv.provider-pack-evaluation.v1" || !scvEqual(input, result["input"]) {
		return fmt.Errorf("pack evaluation input/result binding mismatch")
	}
	domain, ok := result["domain"].(string)
	if !ok || !scvCoverageAdmits(domain, domain) {
		return fmt.Errorf("invalid pack owner")
	}
	pack, ok := input["pack"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing retained pack")
	}
	ps, err := scvPackProfiles(pack)
	if err != nil {
		return err
	}
	provider, _ := pack["provider"].(map[string]any)
	if !scvCoverageSourceScope(domain, provider) {
		return fmt.Errorf("pack provider outside owner scope")
	}
	claims, extractions, err := scvPackSelection(pack, ps, input, domain)
	if err != nil {
		return err
	}
	k, ok := result["knowledge"].(map[string]any)
	if !ok || !scvCorpusFields(k, "protocol", "domain", "interpreter_version", "selection_policy", "captures", "claims", "native_nodes", "native_edges", "limitations", "digest") || k["protocol"] != "symphony.scv.knowledge.v1" || k["domain"] != domain || k["interpreter_version"] != "provider-pack-v1:"+pack["digest"].(string) || !scvUnorderedEqual(k["captures"], input["captures"]) || !scvPolicyEqual(k["selection_policy"], input["selection_policy"]) || !scvEqual(k["claims"], claims) || !scvEqual(result["extractions"], extractions) {
		return fmt.Errorf("pack changed extracted knowledge, evidence or policy")
	}
	if err := scvSeal(k, "digest"); err != nil {
		return err
	}
	fixtures, err := scvPackFixtures(input["fixtures"])
	if err != nil {
		return err
	}
	manifests, err := scvPackMapArray(pack["fixtures"], "fixture_id", 16)
	if err != nil {
		return err
	}
	cases := []any{}
	counts := map[string]any{"passed": 0, "failed": 0, "not_run": 0}
	for id, m := range manifests {
		var ac, ae, axes any
		status := "not_run"
		if f := fixtures[id]; f != nil {
			dg, err := SCVDigest(f)
			if err != nil || dg != m["input_digest"] {
				return fmt.Errorf("selected fixture input digest mismatch")
			}
			cs, es, err := scvPackSelection(pack, ps, f, domain)
			if err != nil {
				return err
			}
			projected := []any{}
			for _, raw := range cs {
				projected = append(projected, scvPackProjection(raw.(map[string]any)))
			}
			ac = scvPackSorted(projected)
			ae = es
			dc := !scvEqual(ac, scvPackSorted(m["expected_claims"].([]any)))
			de := !scvEqual(ae, scvPackSorted(m["expected_extractions"].([]any)))
			axes = map[string]any{"claims": dc, "extractions": de}
			status = "passed"
			if dc || de {
				status = "failed"
			}
		}
		counts[status] = counts[status].(int) + 1
		cases = append(cases, map[string]any{"fixture_id": id, "status": status, "expected_claims": m["expected_claims"], "expected_extractions": m["expected_extractions"], "actual_claims": ac, "actual_extractions": ae, "difference_axes": axes})
	}
	for id := range fixtures {
		if manifests[id] == nil {
			return fmt.Errorf("selected fixture not declared")
		}
	}
	if !scvEqual(result["fixture_results"], scvPackSorted(cases)) || !scvEqual(result["conformance"], counts) {
		return fmt.Errorf("pack changed independently authored fixture outcomes or accounting")
	}
	limits, ok := result["limitations"].([]any)
	if !ok || len(limits) == 0 || len(limits) > 16 {
		return fmt.Errorf("missing pack qualification")
	}
	for _, l := range limits {
		if !scvPackText(l, 4096, false) {
			return fmt.Errorf("invalid pack qualification")
		}
	}
	return nil
}
func scvPackDraftFields(p map[string]any) bool {
	return scvCorpusFields(p, "protocol", "pack_id", "pack_version", "authored_by", "provenance", "provider", "profiles", "structured_profiles", "fixtures")
}
func scvPackCopy(m map[string]any) map[string]any {
	b, _ := SCVCanonical(m)
	var out map[string]any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	_ = d.Decode(&out)
	return out
}
func scvPackText(v any, n int, empty bool) bool {
	s, ok := v.(string)
	return ok && (empty || s != "") && len(s) <= n && !strings.ContainsRune(s, 0)
}
func scvPackSorted(a []any) []any {
	r := append([]any{}, a...)
	sort.Slice(r, func(i, j int) bool {
		x, _ := SCVCanonical(r[i])
		y, _ := SCVCanonical(r[j])
		return bytes.Compare(x, y) < 0
	})
	return r
}
func scvPackMapArray(v any, key string, n int) (map[string]map[string]any, error) {
	a, ok := v.([]any)
	if !ok || len(a) > n {
		return nil, fmt.Errorf("pack array outside bound")
	}
	out := map[string]map[string]any{}
	for _, raw := range a {
		m, ok := raw.(map[string]any)
		if !ok || !scvPackText(m[key], 512, false) {
			return nil, fmt.Errorf("pack entry identity invalid")
		}
		id := m[key].(string)
		if out[id] != nil {
			return nil, fmt.Errorf("duplicate pack identity")
		}
		out[id] = m
	}
	return out, nil
}
func scvPackFixtures(v any) (map[string]map[string]any, error) {
	out, err := scvPackMapArray(v, "fixture_id", 16)
	if err != nil {
		return nil, err
	}
	n := 0
	for _, f := range out {
		if !scvCorpusFields(f, "fixture_id", "captures", "bindings", "selection_policy") {
			return nil, fmt.Errorf("invalid fixture input fields")
		}
		a, ok := f["captures"].([]any)
		if !ok {
			return nil, fmt.Errorf("invalid fixture captures")
		}
		n += len(a)
	}
	if n > 16 {
		return nil, fmt.Errorf("fixture capture aggregate bound")
	}
	return out, nil
}
func scvPackProjection(m map[string]any) map[string]any {
	o := map[string]any{}
	for _, k := range []string{"claim_id", "subject", "predicate", "scope", "statement_kind", "value", "dependencies"} {
		o[k] = m[k]
	}
	return o
}
func scvPackProfiles(p map[string]any) (map[string]map[string]any, error) {
	if !scvCorpusFields(p, "protocol", "pack_id", "pack_version", "authored_by", "provenance", "provider", "profiles", "structured_profiles", "fixtures", "digest") || p["protocol"] != "symphony.scv.provider-pack.v1" {
		return nil, fmt.Errorf("invalid pack fields/protocol")
	}
	if err := scvSeal(p, "digest"); err != nil {
		return nil, err
	}
	for _, k := range []string{"pack_id", "pack_version"} {
		if !scvPackText(p[k], 512, false) {
			return nil, fmt.Errorf("invalid pack identity")
		}
	}
	if !scvPackText(p["authored_by"], 1024, false) {
		return nil, fmt.Errorf("pack lacks author")
	}
	prov, ok := p["provenance"].([]any)
	if !ok || len(prov) == 0 || len(prov) > 16 {
		return nil, fmt.Errorf("pack lacks bounded provenance")
	}
	for _, v := range prov {
		if !scvPackText(v, 4096, false) {
			return nil, fmt.Errorf("invalid pack provenance")
		}
	}
	provider, ok := p["provider"].(map[string]any)
	if !ok || !scvCorpusFields(provider, "provider_id", "family_id", "display_name", "sources") {
		return nil, fmt.Errorf("invalid pack provider")
	}
	sources, err := scvPackMapArray(provider["sources"], "source_id", 32)
	if err != nil || len(sources) == 0 {
		return nil, fmt.Errorf("invalid provider sources")
	}
	ps := map[string]map[string]any{}
	claims := map[string]bool{}
	ruleCount := 0
	for _, group := range []string{"profiles", "structured_profiles"} {
		a, ok := p[group].([]any)
		if !ok || len(a) > 16 {
			return nil, fmt.Errorf("invalid pack profile array")
		}
		for _, raw := range a {
			q, ok := raw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid pack profile")
			}
			structured := group == "structured_profiles"
			keys := []string{"protocol", "profile_id", "profile_version", "provider_id", "source_id", "locator_id", "media_types", "authored_by", "rationale", "rules"}
			if !structured {
				keys = append(keys, "digest")
			}
			if !scvCorpusFields(q, keys...) || !scvPackText(q["profile_id"], 512, false) || !scvPackText(q["profile_version"], 512, false) || !scvPackText(q["authored_by"], 1024, false) || !scvPackText(q["rationale"], 4096, false) {
				return nil, fmt.Errorf("invalid profile metadata")
			}
			if structured {
				if q["protocol"] != "symphony.scv.structured-profile.v1" {
					return nil, fmt.Errorf("invalid structured profile protocol")
				}
			} else {
				if q["protocol"] != "symphony.scv.interpretation-profile.v1" {
					return nil, fmt.Errorf("invalid text profile protocol")
				}
				if err := scvSeal(q, "digest"); err != nil {
					return nil, err
				}
			}
			id := q["profile_id"].(string)
			if ps[id] != nil || len(ps) >= 16 {
				return nil, fmt.Errorf("duplicate/excessive profile identity")
			}
			ps[id] = q
			sid, sok := q["source_id"].(string)
			s := sources[sid]
			if !sok || s == nil || !scvEqual(q["provider_id"], provider["provider_id"]) {
				return nil, fmt.Errorf("profile source/provider mismatch")
			}
			ls, err := scvPackMapArray(s["locators"], "locator_id", 16)
			lid, lok := q["locator_id"].(string)
			if err != nil || !lok || ls[lid] == nil {
				return nil, fmt.Errorf("profile locator undeclared")
			}
			media, ok := q["media_types"].([]any)
			if !ok || len(media) == 0 || len(media) > 16 {
				return nil, fmt.Errorf("profile media invalid")
			}
			seen := map[string]bool{}
			for _, m := range media {
				if !scvPackText(m, 128, false) || seen[m.(string)] {
					return nil, fmt.Errorf("profile media invalid")
				}
				seen[m.(string)] = true
			}
			rules, err := scvPackMapArray(q["rules"], "rule_id", 128)
			if err != nil {
				return nil, err
			}
			for _, rule := range rules {
				ruleCount++
				if ruleCount > 128 || !scvCorpusFields(rule, "rule_id", "claim_id", "subject", "predicate", "scope", "statement_kind", "dependencies", "context", "extractor") || !scvPackText(rule["claim_id"], 512, false) {
					return nil, fmt.Errorf("rule fields/aggregate bound")
				}
				if err := scvPackClaimMetadata(rule); err != nil {
					return nil, err
				}
				cid := rule["claim_id"].(string)
				if claims[cid] {
					return nil, fmt.Errorf("duplicate generated claim")
				}
				claims[cid] = true
				x, ok := rule["extractor"].(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid extractor")
				}
				context, ok := rule["context"].([]any)
				if !ok || len(context) > 8 {
					return nil, fmt.Errorf("context bound")
				}
				if structured {
					if !scvCorpusFields(x, "kind", "pointer", "type", "unit") || x["kind"] != "json_pointer" {
						return nil, fmt.Errorf("invalid pointer extractor")
					}
					if _, err := scvPackPointer(x["pointer"]); err != nil {
						return nil, err
					}
					if !scvPackTypeDescriptor(x) {
						return nil, fmt.Errorf("invalid pointer type/unit")
					}
					for _, raw := range context {
						c, ok := raw.(map[string]any)
						if !ok || !scvCorpusFields(c, "pointer", "value") {
							return nil, fmt.Errorf("invalid typed context")
						}
						if _, err := scvPackPointer(c["pointer"]); err != nil {
							return nil, err
						}
						v, ok := c["value"].(map[string]any)
						if !ok || !scvValidTypedValue(v) {
							return nil, fmt.Errorf("invalid context value")
						}
					}
				} else {
					for _, c := range context {
						if !scvPackText(c, 4096, false) {
							return nil, fmt.Errorf("invalid text context")
						}
					}
					if x["kind"] == "literal" {
						v, ok := x["value"].(map[string]any)
						if !scvCorpusFields(x, "kind", "quote", "value") || !scvPackText(x["quote"], 4096, false) || !ok || !scvValidTypedValue(v) {
							return nil, fmt.Errorf("invalid literal extractor")
						}
					} else if x["kind"] == "delimited" {
						if !scvCorpusFields(x, "kind", "prefix", "suffix", "type", "unit") || !scvPackText(x["prefix"], 4096, false) || !scvPackText(x["suffix"], 4096, false) || !scvPackTypeDescriptor(x) || (x["type"] != "integer" && x["type"] != "decimal" && x["type"] != "string") {
							return nil, fmt.Errorf("invalid delimited extractor")
						}
					} else {
						return nil, fmt.Errorf("unsupported text extractor")
					}
				}
			}
		}
	}
	ms, err := scvPackMapArray(p["fixtures"], "fixture_id", 16)
	if err != nil || len(ms) == 0 {
		return nil, fmt.Errorf("pack requires conformance manifest")
	}
	for _, m := range ms {
		if !scvCorpusFields(m, "fixture_id", "label", "authored_by", "rationale", "input_digest", "expected_claims", "expected_extractions") || !scvPackText(m["label"], 1024, false) || !scvPackText(m["authored_by"], 1024, false) || !scvPackText(m["rationale"], 4096, false) {
			return nil, fmt.Errorf("invalid fixture metadata")
		}
		dg, ok := m["input_digest"].(string)
		if !ok || !taggedSHA256(dg) {
			return nil, fmt.Errorf("invalid fixture digest")
		}
		cs, err := scvPackMapArray(m["expected_claims"], "claim_id", 128)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			if err := scvPackClaimMetadata(c); err != nil {
				return nil, err
			}
			v, ok := c["value"].(map[string]any)
			if !scvCorpusFields(c, "claim_id", "subject", "predicate", "scope", "statement_kind", "value", "dependencies") || !ok || !scvValidTypedValue(v) {
				return nil, fmt.Errorf("invalid fixture expected claim")
			}
		}
		es, ok := m["expected_extractions"].([]any)
		if !ok || len(es) > 128 {
			return nil, fmt.Errorf("invalid fixture expected extractions")
		}
		for _, raw := range es {
			e, ok := raw.(map[string]any)
			if !ok || !scvCorpusFields(e, "profile_id", "capture_digest", "rule_id", "claim_id", "status", "reasons") {
				return nil, fmt.Errorf("invalid fixture extraction fields")
			}
			if e["status"] != "matched" && e["status"] != "unresolved" {
				return nil, fmt.Errorf("invalid fixture extraction status")
			}
			if _, ok := e["reasons"].([]any); !ok {
				return nil, fmt.Errorf("invalid fixture reasons")
			}
		}
	}
	return ps, nil
}
func scvPackTypeDescriptor(x map[string]any) bool {
	switch x["type"] {
	case "integer", "decimal", "boolean", "string", "reference":
	default:
		return false
	}
	return x["unit"] == nil || scvPackText(x["unit"], 128, false)
}
func scvPackPointer(v any) ([]string, error) {
	if !scvPackText(v, 1024, true) {
		return nil, fmt.Errorf("invalid pointer")
	}
	s := v.(string)
	if s == "" {
		return []string{}, nil
	}
	if s[0] != '/' {
		return nil, fmt.Errorf("pointer must start with slash")
	}
	parts := strings.Split(s[1:], "/")
	if len(parts) > 32 {
		return nil, fmt.Errorf("pointer depth bound")
	}
	for i, p := range parts {
		var b strings.Builder
		for j := 0; j < len(p); j++ {
			if p[j] != '~' {
				b.WriteByte(p[j])
				continue
			}
			j++
			if j == len(p) || (p[j] != '0' && p[j] != '1') {
				return nil, fmt.Errorf("pointer escape invalid")
			}
			if p[j] == '0' {
				b.WriteByte('~')
			} else {
				b.WriteByte('/')
			}
		}
		parts[i] = b.String()
	}
	return parts, nil
}

type scvPackNode struct {
	kind, raw string
	value     any
	reference bool
}

func scvPackDocument(body string) (map[string]scvPackNode, error) {
	if err := scvPackJSONUnicode(body); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	count := 0
	var check func(int) error
	check = func(depth int) error {
		count++
		if depth > 32 || count > 8192 {
			return fmt.Errorf("document bounds")
		}
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				s, ok := key.(string)
				if !ok || seen[s] {
					return fmt.Errorf("duplicate document key")
				}
				seen[s] = true
				if err := check(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for dec.More() {
				if err := check(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected document delimiter")
		}
		_, err = dec.Token()
		return err
	}
	if err := check(0); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing document data")
	}
	nodes := map[string]scvPackNode{}
	var walk func(json.RawMessage, string) error
	walk = func(raw json.RawMessage, path string) error {
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 {
			return fmt.Errorf("empty node")
		}
		n := scvPackNode{raw: string(raw)}
		switch raw[0] {
		case '{':
			n.kind = "object"
			var m map[string]json.RawMessage
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			_, n.reference = m["$ref"]
			for k, v := range m {
				if err := walk(v, path+"/"+strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1")); err != nil {
					return err
				}
			}
		case '[':
			n.kind = "array"
			var a []json.RawMessage
			if err := json.Unmarshal(raw, &a); err != nil {
				return err
			}
			for i, v := range a {
				if err := walk(v, path+"/"+strconv.Itoa(i)); err != nil {
					return err
				}
			}
		case '"':
			n.kind = "string"
			var s string
			if err := json.Unmarshal(raw, &s); err != nil {
				return err
			}
			n.value = s
		case 't', 'f':
			n.kind = "boolean"
			n.value = string(raw) == "true"
		case 'n':
			n.kind = "null"
		default:
			n.kind = "number"
			n.value = json.Number(raw)
		}
		nodes[path] = n
		return nil
	}
	if err := walk(json.RawMessage(body), ""); err != nil {
		return nil, err
	}
	return nodes, nil
}
func scvPackSelect(nodes map[string]scvPackNode, p any) (scvPackNode, string) {
	parts, err := scvPackPointer(p)
	if err != nil {
		return scvPackNode{}, "pointer_missing"
	}
	path := ""
	for i := 0; ; i++ {
		n, ok := nodes[path]
		if !ok {
			return n, "pointer_missing"
		}
		if n.reference {
			return n, "reference_unresolved"
		}
		if i == len(parts) {
			return n, ""
		}
		if n.kind != "array" && n.kind != "object" {
			return n, "pointer_missing"
		}
		if n.kind == "array" {
			x := parts[i]
			if x == "" || (len(x) > 1 && x[0] == '0') || strings.IndexFunc(x, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
				return n, "array_index_invalid"
			}
		}
		path += "/" + strings.ReplaceAll(strings.ReplaceAll(parts[i], "~", "~0"), "/", "~1")
	}
}
func scvPackNodeValue(n scvPackNode, x map[string]any) (map[string]any, bool) {
	var value any
	switch x["type"] {
	case "integer":
		if n.kind != "number" || strings.ContainsAny(n.raw, ".eE") {
			return nil, false
		}
		i, err := strconv.ParseInt(n.raw, 10, 64)
		if err != nil || i < -9007199254740991 || i > 9007199254740991 {
			return nil, false
		}
		value = json.Number(strconv.FormatInt(i, 10))
	case "decimal":
		if n.kind != "number" {
			return nil, false
		}
		s := n.raw
		exp := 0
		if k := strings.IndexAny(s, "eE"); k >= 0 {
			var err error
			exp, err = strconv.Atoi(s[k+1:])
			if err != nil || exp < -256 || exp > 256 {
				return nil, false
			}
			s = s[:k]
		}
		if strings.Contains(s, ".") {
			s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
		}
		scale := 0
		if k := strings.IndexByte(s, '.'); k >= 0 {
			scale = len(s) - k - 1
		}
		scale -= exp
		if scale < 0 {
			scale = 0
		}
		if scale > 512 {
			return nil, false
		}
		r, ok := new(big.Rat).SetString(n.raw)
		if !ok {
			return nil, false
		}
		s = r.FloatString(scale)
		if strings.Contains(s, ".") {
			s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
		}
		if len(s) > 128 {
			return nil, false
		}
		value = s
	case "boolean":
		if n.kind != "boolean" {
			return nil, false
		}
		value = n.value
	case "string", "reference":
		if n.kind != "string" || !scvPackText(n.value, 4096, false) {
			return nil, false
		}
		value = n.value
	default:
		return nil, false
	}
	return map[string]any{"type": x["type"], "value": value, "unit": x["unit"]}, true
}
func scvPackStringSet(s map[string]bool) []any {
	keys := []string{}
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []any{}
	for _, k := range keys {
		out = append(out, k)
	}
	return out
}
func scvPackTextExtraction(rule, cap, profile, policy map[string]any) (map[string]any, map[string]bool, map[string]bool) {
	reasons := map[string]bool{}
	anchors := map[string]bool{}
	body, _ := cap["body"].(string)
	context, _ := rule["context"].([]any)
	for i, raw := range context {
		s := raw.(string)
		count := scvOccurrences(body, s)
		if count != 1 {
			reason := "context_missing:"
			if count > 1 {
				reason = "context_ambiguous:"
			}
			reasons[reason+strconv.Itoa(i)] = true
		} else {
			anchors[s] = true
		}
	}
	x := rule["extractor"].(map[string]any)
	var value map[string]any
	if x["kind"] == "literal" {
		quote := x["quote"].(string)
		n := scvOccurrences(body, quote)
		if n != 1 {
			reason := "value_missing"
			if n > 1 {
				reason = "value_ambiguous"
			}
			reasons[reason] = true
		} else {
			anchors[quote] = true
			value = x["value"].(map[string]any)
		}
	} else {
		prefix, suffix := x["prefix"].(string), x["suffix"].(string)
		count, begin, end := 0, 0, 0
		for start := 0; start < len(body); {
			offset := strings.Index(body[start:], prefix)
			if offset < 0 {
				break
			}
			cursor := start + offset
			close := strings.Index(body[cursor+len(prefix):], suffix)
			if close >= 0 {
				begin = cursor
				end = cursor + len(prefix) + close
				count++
				if count >= 2 {
					break
				}
			}
			start = cursor + 1
		}
		if count != 1 {
			reason := "value_missing"
			if count > 1 {
				reason = "value_ambiguous"
			}
			reasons[reason] = true
		} else if end+len(suffix)-begin > 4096 {
			reasons["value_quote_exceeds_evidence_bound"] = true
		} else {
			// Check the token independently of any failed surrounding context.
			q := scvPackCopy(rule)
			q["context"] = []any{}
			v, ok := scvExtractedValue(q, cap, profile, policy)
			if !ok {
				reasons["invalid_token"] = true
			} else {
				value = v
				anchors[body[begin:end+len(suffix)]] = true
			}
		}
	}
	return value, reasons, anchors
}
func scvPackSelection(pack map[string]any, profiles map[string]map[string]any, input map[string]any, domain string) ([]any, []any, error) {
	caps, err := scvPackMapArray(input["captures"], "digest", 16)
	if err != nil {
		return nil, nil, err
	}
	policy, ok := input["selection_policy"].(map[string]any)
	if !ok || !scvCorpusFields(policy, "policy_id", "max_age_seconds", "partial_capture", "allowed_statement_kinds") {
		return nil, nil, fmt.Errorf("invalid pack selected policy")
	}
	provider := pack["provider"].(map[string]any)
	sources, _ := scvPackMapArray(provider["sources"], "source_id", 32)
	for _, c := range caps {
		source, ok := c["source"].(map[string]any)
		if !ok || !scvCoverageSourceScope(domain, source) {
			return nil, nil, fmt.Errorf("capture outside pack owner")
		}
		if err := scvSeal(source, "digest"); err != nil {
			return nil, nil, err
		}
		ci := map[string]any{}
		for k, v := range c {
			if k != "protocol" && k != "digest" && k != "body_digest" && k != "byte_size" {
				ci[k] = v
			}
		}
		if err := scvCoverageValidateNested("capture_import", ci, c); err != nil {
			return nil, nil, err
		}
	}
	bindings, ok := input["bindings"].([]any)
	if !ok || len(bindings) > 16 {
		return nil, nil, fmt.Errorf("invalid pack bindings")
	}
	boundP, boundC := map[string]bool{}, map[string]bool{}
	claims, extractions := []any{}, []any{}
	for _, raw := range scvPackSorted(bindings) {
		b, ok := raw.(map[string]any)
		if !ok || !scvCorpusFields(b, "profile_id", "capture_digest") {
			return nil, nil, fmt.Errorf("invalid binding fields")
		}
		pid, pok := b["profile_id"].(string)
		cd, cok := b["capture_digest"].(string)
		p, c := profiles[pid], caps[cd]
		if !pok || !cok || p == nil || c == nil || boundP[pid] {
			return nil, nil, fmt.Errorf("binding selects invalid profile/capture")
		}
		boundP[pid] = true
		boundC[cd] = true
		source := c["source"].(map[string]any)
		if !scvEqual(p["provider_id"], source["provider_id"]) || !scvEqual(p["source_id"], source["source_id"]) || !scvEqual(p["locator_id"], c["locator_id"]) {
			return nil, nil, fmt.Errorf("binding source mismatch")
		}
		desired := sources[p["source_id"].(string)]
		stage := ""
		for k, v := range desired {
			if !scvEqual(v, source[k]) {
				stage = "source_declaration_differs"
			}
		}
		if stage == "" {
			if c["completeness"] == "failed" {
				stage = "capture_failed"
			} else if c["completeness"] == "partial" && policy["partial_capture"] == "exclude" {
				stage = "partial_capture_excluded"
			} else {
				supported := false
				for _, m := range p["media_types"].([]any) {
					if scvEqual(m, c["media_type"]) {
						supported = true
					}
				}
				if !supported {
					stage = "unsupported_media_type"
				}
			}
		}
		structured := p["protocol"] == "symphony.scv.structured-profile.v1"
		var nodes map[string]scvPackNode
		if stage == "" && structured {
			body, _ := c["body"].(string)
			nodes, err = scvPackDocument(body)
			if err != nil {
				stage = "invalid_json"
			}
		}
		for _, rawRule := range p["rules"].([]any) {
			rule := rawRule.(map[string]any)
			reasons, anchors := map[string]bool{}, map[string]bool{}
			var value map[string]any
			if stage != "" {
				reasons[stage] = true
			} else if !structured {
				value, reasons, anchors = scvPackTextExtraction(rule, c, p, policy)
			} else {
				for i, rawContext := range rule["context"].([]any) {
					context := rawContext.(map[string]any)
					n, reason := scvPackSelect(nodes, context["pointer"])
					suffix := ":" + strconv.Itoa(i)
					if reason != "" {
						reasons["context_"+reason+suffix] = true
					} else {
						want := context["value"].(map[string]any)
						v, ok := scvPackNodeValue(n, want)
						if !ok {
							reasons["context_type_mismatch"+suffix] = true
						} else if !scvEqual(v, want) {
							reasons["context_value_differs"+suffix] = true
						} else if len(n.raw) > 4096 {
							reasons["context_quote_exceeds_bound"+suffix] = true
						} else {
							anchors[n.raw] = true
						}
					}
				}
				x := rule["extractor"].(map[string]any)
				n, reason := scvPackSelect(nodes, x["pointer"])
				if reason != "" {
					reasons[reason] = true
				} else if len(n.raw) > 4096 {
					reasons["value_quote_exceeds_evidence_bound"] = true
				} else {
					v, ok := scvPackNodeValue(n, x)
					if !ok {
						reasons["value_type_mismatch"] = true
					} else {
						value = v
						anchors[n.raw] = true
					}
				}
			}
			status := "unresolved"
			if len(reasons) == 0 {
				status = "matched"
				claim := scvPackProjection(rule)
				claim["value"] = value
				claim["dependencies"] = scvPackSorted(rule["dependencies"].([]any))
				claim["valid_from"] = nil
				claim["valid_until"] = nil
				claim["alternative_supports"] = []any{}
				claim["scope_dependencies"] = []any{}
				ev := []any{}
				for a := range anchors {
					ev = append(ev, map[string]any{"capture_digest": cd, "quote": a})
				}
				claim["evidence"] = scvPackSorted(ev)
				claims = append(claims, claim)
				if c["completeness"] == "partial" {
					reasons["partial_capture_qualified"] = true
				}
			}
			extractions = append(extractions, map[string]any{"profile_id": pid, "capture_digest": cd, "rule_id": rule["rule_id"], "claim_id": rule["claim_id"], "status": status, "reasons": scvPackStringSet(reasons)})
		}
	}
	if len(boundC) != len(caps) {
		return nil, nil, fmt.Errorf("unbound pack capture")
	}
	// Ordinary knowledge orders claims by identity, independently of other fields.
	sort.Slice(claims, func(i, j int) bool {
		return claims[i].(map[string]any)["claim_id"].(string) < claims[j].(map[string]any)["claim_id"].(string)
	})
	return claims, scvPackSorted(extractions), nil
}

func scvPackClaimMetadata(c map[string]any) error {
	for _, key := range []string{"claim_id", "subject", "predicate"} {
		if !scvPackText(c[key], 512, false) {
			return fmt.Errorf("invalid pack claim identity")
		}
	}
	switch c["statement_kind"] {
	case "documented_fact", "requirement", "recommendation", "observation", "user_assertion", "inference", "hypothesis":
	default:
		return fmt.Errorf("invalid pack statement kind")
	}
	scope, ok := c["scope"].(map[string]any)
	if !ok || len(scope) > 16 {
		return fmt.Errorf("invalid pack scope")
	}
	for k, v := range scope {
		if k == "" || len(k) > 128 || !scvPackText(v, 512, true) {
			return fmt.Errorf("invalid pack scope qualifier")
		}
	}
	deps, ok := c["dependencies"].([]any)
	if !ok || len(deps) > 16 {
		return fmt.Errorf("invalid pack dependencies")
	}
	seen := map[string]bool{}
	for _, raw := range deps {
		d, ok := raw.(map[string]any)
		if !ok || !scvCorpusFields(d, "claim_id", "role") || !scvPackText(d["claim_id"], 512, false) {
			return fmt.Errorf("invalid dependency")
		}
		switch d["role"] {
		case "support", "requires", "scope":
		default:
			return fmt.Errorf("invalid dependency role")
		}
		key, _ := SCVDigest(d)
		if seen[key] {
			return fmt.Errorf("duplicate dependency")
		}
		seen[key] = true
	}
	return nil
}

// encoding/json replaces malformed UTF-16 escapes. The native parser rejects
// them, so check escape pairs before using Decoder for structural validation.
func scvPackJSONUnicode(body string) error {
	quoted := false
	for i := 0; i < len(body); i++ {
		if body[i] == '"' {
			quoted = !quoted
			continue
		}
		if !quoted || body[i] != '\\' {
			continue
		}
		i++
		if i >= len(body) {
			return fmt.Errorf("truncated JSON escape")
		}
		if body[i] != 'u' {
			continue
		}
		if i+4 >= len(body) {
			return fmt.Errorf("truncated unicode escape")
		}
		code, err := strconv.ParseUint(body[i+1:i+5], 16, 16)
		if err != nil {
			return err
		}
		i += 4
		if code >= 0xdc00 && code <= 0xdfff {
			return fmt.Errorf("unpaired low surrogate")
		}
		if code >= 0xd800 && code <= 0xdbff {
			if i+6 >= len(body) || body[i+1:i+3] != `\u` {
				return fmt.Errorf("unpaired high surrogate")
			}
			low, err := strconv.ParseUint(body[i+3:i+7], 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return fmt.Errorf("invalid low surrogate")
			}
			i += 6
		}
	}
	return nil
}

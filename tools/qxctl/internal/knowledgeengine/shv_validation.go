package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// JSON/canonical primitives are shared mechanics only; all correspondence and
// hardware/exchange checks below are separately authored for this contract.
func shvObject(raw []byte) (map[string]any, error) {
	if err := ValidateSCVBundleText(raw); err != nil {
		return nil, err
	}
	return scvObject(raw)
}
func shvMap(v any) map[string]any                     { m, _ := v.(map[string]any); return m }
func shvList(v any) []any                             { a, _ := v.([]any); return a }
func shvText(v any) string                            { s, _ := v.(string); return s }
func shvFields(m map[string]any, keys ...string) bool { return graphIndexFields(m, keys...) }
func shvBoundedText(v any, max int) bool {
	s, ok := v.(string)
	if !ok || s == "" || len(s) > max {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func shvFail() error {
	return fmt.Errorf("SHV result does not match the exact request/evidence contract")
}
func shvSealed(m map[string]any) error           { return scvSeal(m, "digest") }
func shvSealNew(m map[string]any) map[string]any { d, _ := SCVDigest(m); m["digest"] = d; return m }
func shvDate(v any) bool {
	s := shvText(v)
	d, e := time.Parse("2006-01-02", s)
	return e == nil && d.Format("2006-01-02") == s
}

var shvIdentifier = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func shvID(v any) bool { return shvIdentifier.MatchString(shvText(v)) }
func shvStrings(v any, sorted bool) bool {
	a, ok := v.([]any)
	if !ok {
		return false
	}
	seen := map[string]bool{}
	last := ""
	for _, x := range a {
		s, ok := x.(string)
		if !ok || s == "" || seen[s] || (sorted && last != "" && s <= last) {
			return false
		}
		seen[s] = true
		last = s
	}
	return true
}
func shvValue(v any) bool {
	switch x := v.(type) {
	case string:
		return true
	case json.Number:
		_, e := x.Int64()
		return e == nil
	case []any:
		return shvStrings(x, false)
	}
	return false
}
func shvRequirementValue(v any) bool {
	switch x := v.(type) {
	case string:
		return shvBoundedText(x, 4096)
	case json.Number:
		return shvValue(x)
	case []any:
		if len(x) > 32 || !shvStrings(x, false) {
			return false
		}
		for _, v := range x {
			if !shvBoundedText(v, 128) {
				return false
			}
		}
		return true
	}
	return false
}
func shvInterval(v any) bool {
	if v == nil {
		return true
	}
	m := shvMap(v)
	return shvFields(m, "from", "through") && shvDate(m["from"]) && shvDate(m["through"]) && shvText(m["from"]) <= shvText(m["through"])
}
func shvSummary(v any) bool {
	m := shvMap(v)
	return shvFields(m, "id", "manufacturer", "model", "hardware_class", "introduced") && shvID(m["id"]) && shvBoundedText(m["manufacturer"], 256) && shvBoundedText(m["model"], 256) && shvID(m["hardware_class"]) && shvInterval(m["introduced"])
}
func shvSelector(m map[string]any, subject map[string]any, asOf string, depth int, nodes *int) (string, error) {
	*nodes++
	if depth > 16 || *nodes > 128 {
		return "", shvFail()
	}
	op := shvText(m["op"])
	switch op {
	case "all":
		if !shvFields(m, "op") {
			break
		}
		return "included", nil
	case "ids", "class", "manufacturer":
		if !shvFields(m, "op", "values") || len(shvList(m["values"])) > 128 || !shvStrings(m["values"], false) {
			break
		}
		for _, v := range shvList(m["values"]) {
			if (op == "manufacturer" && !shvBoundedText(v, 256)) || (op != "manufacturer" && !shvID(v)) {
				return "", shvFail()
			}
		}
		field := map[string]string{"ids": "id", "class": "hardware_class", "manufacturer": "manufacturer"}[op]
		for _, v := range shvList(m["values"]) {
			if v == subject[field] {
				return "included", nil
			}
		}
		return "excluded", nil
	case "date":
		if !shvFields(m, "op", "basis", "from", "through") || m["basis"] != "model_introduction" || !shvDate(m["from"]) || !shvDate(m["through"]) || shvText(m["from"]) > shvText(m["through"]) || shvText(m["through"]) > asOf {
			break
		}
		if subject["introduced"] == nil {
			return "unresolved", nil
		}
		iv := shvMap(subject["introduced"])
		a, b := shvText(iv["from"]), shvText(iv["through"])
		low, high := shvText(m["from"]), shvText(m["through"])
		if b < low || a > high {
			return "excluded", nil
		}
		if a >= low && b <= high {
			return "included", nil
		}
		return "unresolved", nil
	case "not":
		if !shvFields(m, "op", "arg") {
			break
		}
		s, e := shvSelector(shvMap(m["arg"]), subject, asOf, depth+1, nodes)
		if s == "included" {
			s = "excluded"
		} else if s == "excluded" {
			s = "included"
		}
		return s, e
	case "and", "or":
		a, ok := m["args"].([]any)
		if !shvFields(m, "op", "args") || !ok || len(a) == 0 || len(a) > 32 {
			break
		}
		inc, exc, unknown := false, false, false
		for _, v := range a {
			s, e := shvSelector(shvMap(v), subject, asOf, depth+1, nodes)
			if e != nil {
				return "", e
			}
			inc = inc || s == "included"
			exc = exc || s == "excluded"
			unknown = unknown || s == "unresolved"
		}
		if op == "and" {
			if exc {
				return "excluded", nil
			}
			if unknown {
				return "unresolved", nil
			}
			return "included", nil
		}
		if inc {
			return "included", nil
		}
		if unknown {
			return "unresolved", nil
		}
		return "excluded", nil
	}
	return "", shvFail()
}
func shvCoverage(p map[string]any) (map[string]any, error) {
	profile := shvMap(p["profile"])
	a, ok := p["subjects"].([]any)
	if !shvFields(p, "profile", "subjects") || !ok || len(a) > 128 || !shvFields(profile, "protocol", "as_of", "selector", "digest") || profile["protocol"] != "symphony.shv.coverage-profile.v1" || !shvDate(profile["as_of"]) {
		return nil, shvFail()
	}
	if e := shvSealed(profile); e != nil {
		return nil, e
	}
	decisions := []any{}
	counts := map[string]any{"included": 0, "excluded": 0, "unresolved": 0}
	seen := map[string]bool{}
	// Validate AST even for an empty inventory.
	n := 0
	if _, e := shvSelector(shvMap(profile["selector"]), map[string]any{}, shvText(profile["as_of"]), 0, &n); e != nil {
		return nil, e
	}
	for _, v := range a {
		s := shvMap(v)
		id := shvText(s["id"])
		if !shvSummary(v) || seen[id] {
			return nil, shvFail()
		}
		seen[id] = true
		n = 0
		status, e := shvSelector(shvMap(profile["selector"]), s, shvText(profile["as_of"]), 0, &n)
		if e != nil {
			return nil, e
		}
		counts[status] = counts[status].(int) + 1
		decisions = append(decisions, map[string]any{"subject_id": id, "status": status})
	}
	sort.Slice(decisions, func(i, j int) bool {
		return shvText(shvMap(decisions[i])["subject_id"]) < shvText(shvMap(decisions[j])["subject_id"])
	})
	return shvSealNew(map[string]any{"protocol": "symphony.shv.coverage-result.v1", "profile": profile, "subjects": a, "decisions": decisions, "counts": counts}), nil
}
func shvCatalogue(v any) error { return shvCatalogueVersion(v, SHVVersion) }
func shvCatalogueVersion(v any, version string) error {
	if !shvKernelVersion(version) {
		return shvFail()
	}
	c := shvMap(v)
	if !shvFields(c, "protocol", "sources", "subjects", "mapping", "digest") || c["protocol"] != "symphony.shv.catalogue.v1" {
		return shvFail()
	}
	if e := shvSealed(c); e != nil {
		return e
	}
	sources, ok := c["sources"].([]any)
	subjects, ok2 := c["subjects"].([]any)
	mapping, ok3 := c["mapping"].([]any)
	if !ok || !ok2 || !ok3 || len(sources) > 8 || len(subjects) > 32 || len(subjects) != len(mapping) {
		return shvFail()
	}
	sourceIDs, paths := map[string]bool{}, map[string]bool{}
	var total int64
	for _, v := range sources {
		s := shvMap(v)
		n, ok := s["bytes"].(json.Number)
		size, e := n.Int64()
		id, path := shvText(s["id"]), shvText(s["path"])
		if !shvFields(s, "id", "path", "bytes", "digest", "format") || !shvID(id) || sourceIDs[id] || paths[path] || !safeRelativePath(path) || !ok || e != nil || size < 0 || size > 1<<20 || !taggedSHA256(shvText(s["digest"])) || (s["format"] != "html" && s["format"] != "opaque") {
			return shvFail()
		}
		total += size
		sourceIDs[id] = true
		paths[path] = true
	}
	if total > 4<<20 {
		return shvFail()
	}
	maps := map[string]map[string]any{}
	for _, v := range mapping {
		m := shvMap(v)
		fields, ok := m["fields"].([]any)
		id := shvText(m["id"])
		if !shvMappingShape(m, version) || !shvID(id) || maps[id] != nil || !shvBoundedText(m["heading_section"], 128) || !ok || len(fields) > 16 || !sourceIDs[shvText(m["source_id"])] {
			return shvFail()
		}
		maps[id] = m
	}
	last := ""
	count := 0
	for _, v := range subjects {
		s := shvMap(v)
		id := shvText(s["id"])
		m := maps[id]
		a, ok := s["assertions"].([]any)
		if !shvFields(s, "id", "manufacturer", "model", "hardware_class", "introduced", "assertions") || !shvID(id) || id <= last || m == nil || !ok || len(a) != len(shvList(m["fields"])) || !shvInterval(s["introduced"]) {
			return shvFail()
		}
		last = id
		for _, key := range []string{"manufacturer", "model", "hardware_class"} {
			if !shvBoundedText(s[key], 256) || (key == "hardware_class" && !shvID(s[key])) || s[key] != m[key] {
				return shvFail()
			}
		}
		fieldMap := map[string]map[string]any{}
		for _, f := range shvList(m["fields"]) {
			x := shvMap(f)
			pred := shvText(x["predicate"])
			if !shvFieldShape(x, m, version) || !shvID(pred) || fieldMap[pred] != nil || !shvBoundedText(x["qualifier"], 256) {
				return shvFail()
			}
			fieldMap[pred] = x
		}
		prev := ""
		var intro any
		for _, v := range a {
			x := shvMap(v)
			pred := shvText(x["predicate"])
			f := fieldMap[pred]
			if !shvValue(x["value"]) && !(version == SHVTableVersion && (f["value_type"] == "quarter_20yy" || f["value_type"] == "table_rows")) {
				return shvFail()
			}
			if !shvFields(x, "predicate", "value", "qualifier", "source_id") || pred <= prev || f == nil || x["source_id"] != m["source_id"] || !scvEqual(x["qualifier"], f["qualifier"]) {
				return shvFail()
			}
			prev = pred
			switch f["value_type"] {
			case "string":
				if _, ok := x["value"].(string); !ok {
					return shvFail()
				}
			case "integer":
				if _, ok := x["value"].(json.Number); !ok {
					return shvFail()
				}
			case "tokens":
				if !shvStrings(x["value"], true) {
					return shvFail()
				}
			case "date":
				if !shvDate(x["value"]) {
					return shvFail()
				}
			case "quarter_20yy", "table_rows":
				if version != SHVTableVersion || m["interpretation_profile"] != "scoped_tables.v1" || !shvStructuredValue(x["value"], shvText(f["value_type"]), f) {
					return shvFail()
				}
			default:
				return shvFail()
			}
			if pred == "model_introduction" && f["value_type"] == "date" {
				intro = map[string]any{"from": x["value"], "through": x["value"]}
			}
			if pred == "model_introduction" && f["value_type"] == "quarter_20yy" {
				q := shvMap(x["value"])
				intro = map[string]any{"from": q["from"], "through": q["through"]}
			}
		}
		if !scvEqual(intro, s["introduced"]) {
			return shvFail()
		}
		count += len(a)
	}
	if count > 256 {
		return shvFail()
	}
	return nil
}
func shvSelect(rows []any, ids []any) ([]any, []any, error) {
	if !shvStrings(ids, false) {
		return nil, nil, shvFail()
	}
	lookup := map[string]any{}
	for _, v := range rows {
		lookup[shvText(shvMap(v)["id"])] = v
	}
	out, missing := []any{}, []any{}
	if len(ids) == 0 {
		return rows, missing, nil
	}
	for _, v := range ids {
		if x, ok := lookup[shvText(v)]; ok {
			out = append(out, x)
		} else {
			missing = append(missing, v)
		}
	}
	return out, missing, nil
}
func shvQuery(p map[string]any) (map[string]any, error) { return shvQueryVersion(p, SHVVersion) }
func shvQueryVersion(p map[string]any, version string) (map[string]any, error) {
	c := shvMap(p["catalogue"])
	if e := shvCatalogueVersion(c, version); e != nil {
		return nil, e
	}
	ids, ok := p["subject_ids"].([]any)
	if !ok || len(ids) > 128 {
		return nil, shvFail()
	}
	for _, id := range ids {
		if !shvID(id) {
			return nil, shvFail()
		}
	}
	rows, missing, e := shvSelect(shvList(c["subjects"]), ids)
	if e != nil {
		return nil, e
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.query-result.v1", "catalogue_digest": c["digest"], "subject_ids": ids, "subjects": rows, "missing_subject_ids": missing}), nil
}
func shvEvaluation(p map[string]any) (map[string]any, error) {
	return shvEvaluationVersion(p, SHVVersion)
}
func shvEvaluationVersion(p map[string]any, version string) (map[string]any, error) {
	c := shvMap(p["catalogue"])
	if e := shvCatalogueVersion(c, version); e != nil {
		return nil, e
	}
	ids, ok := p["subject_ids"].([]any)
	reqs, ok2 := p["requirements"].([]any)
	if !ok || !ok2 || len(ids) > 128 || len(reqs) > 32 {
		return nil, shvFail()
	}
	for _, id := range ids {
		if !shvID(id) {
			return nil, shvFail()
		}
	}
	rows, missing, e := shvSelect(shvList(c["subjects"]), ids)
	if e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	for _, v := range reqs {
		r := shvMap(v)
		id := shvText(r["id"])
		if !shvFields(r, "id", "predicate", "operator", "value", "qualifier") || !shvID(id) || seen[id] || !shvID(r["predicate"]) || !shvRequirementValue(r["value"]) || !shvBoundedText(r["qualifier"], 256) {
			return nil, shvFail()
		}
		seen[id] = true
		switch r["operator"] {
		case "eq":
		case "gte":
			if _, ok := r["value"].(json.Number); !ok {
				return nil, shvFail()
			}
		case "contains":
			if _, ok := r["value"].(string); !ok || r["qualifier"] != "finite_supported_set" {
				return nil, shvFail()
			}
		default:
			return nil, shvFail()
		}
	}
	findings := []any{}
	for _, v := range rows {
		s := shvMap(v)
		for _, rv := range reqs {
			r := shvMap(rv)
			status := "unresolved"
			evidence := []any{}
			for _, av := range shvList(s["assertions"]) {
				a := shvMap(av)
				if a["predicate"] != r["predicate"] || !scvEqual(a["qualifier"], r["qualifier"]) {
					continue
				}
				if shvMap(a["value"]) != nil {
					return nil, shvFail()
				}
				pass := false
				switch r["operator"] {
				case "eq":
					if fmt.Sprintf("%T", a["value"]) != fmt.Sprintf("%T", r["value"]) {
						return nil, shvFail()
					}
					pass = scvEqual(a["value"], r["value"])
				case "gte":
					x, ok := a["value"].(json.Number)
					if !ok {
						return nil, shvFail()
					}
					y := r["value"].(json.Number)
					xi, _ := x.Int64()
					yi, _ := y.Int64()
					pass = xi >= yi
				case "contains":
					if !shvStrings(a["value"], true) {
						return nil, shvFail()
					}
					for _, x := range shvList(a["value"]) {
						pass = pass || x == r["value"]
					}
				}
				status = "contradicted"
				if pass {
					status = "supported"
				}
				evidence = append(evidence, map[string]any{"predicate": a["predicate"], "source_id": a["source_id"], "value": a["value"], "qualifier": a["qualifier"]})
			}
			findings = append(findings, map[string]any{"subject_id": s["id"], "requirement_id": r["id"], "status": status, "evidence": evidence})
		}
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.evaluation.v1", "catalogue_digest": c["digest"], "subject_ids": ids, "requirements": reqs, "findings": findings, "missing_subject_ids": missing}), nil
}
func shvProjection(c map[string]any) (map[string]any, error) {
	return shvProjectionVersion(c, SHVVersion)
}
func shvProjectionVersion(c map[string]any, version string) (map[string]any, error) {
	if e := shvCatalogueVersion(c, version); e != nil {
		return nil, e
	}
	nodes, edges := []any{}, []any{}
	for _, v := range shvList(c["sources"]) {
		s := shvMap(v)
		nodes = append(nodes, map[string]any{"id": "source:" + shvText(s["id"]), "labels": []any{"evidence_source"}, "properties": s})
	}
	for _, v := range shvList(c["subjects"]) {
		s := shvMap(v)
		id := shvText(s["id"])
		nodes = append(nodes, map[string]any{"id": "subject:" + id, "labels": []any{"hardware_subject"}, "properties": s})
		for _, av := range shvList(s["assertions"]) {
			a := shvMap(av)
			edges = append(edges, map[string]any{"id": "assertion:" + id + ":" + shvText(a["predicate"]), "from": "subject:" + id, "to": "source:" + shvText(a["source_id"]), "label": "supported_by_source", "properties": a})
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return shvText(shvMap(nodes[i])["id"]) < shvText(shvMap(nodes[j])["id"]) })
	sort.Slice(edges, func(i, j int) bool { return shvText(shvMap(edges[i])["id"]) < shvText(shvMap(edges[j])["id"]) })
	return shvSealNew(map[string]any{"protocol": "symphony.graph.exchange.v1", "owner": map[string]any{"engine_id": "symphony-shv", "engine_version": version, "artifact_protocol": c["protocol"], "artifact_digest": c["digest"]}, "owner_artifact": c, "nodes": nodes, "edges": edges}), nil
}
func shvGraph(g map[string]any) error {
	raw, err := SCVCanonical(g)
	if err != nil {
		return err
	}
	if err = validateJSONObject(raw, maxRequestBytes); err != nil {
		return err
	}
	if !shvFields(g, "protocol", "owner", "owner_artifact", "nodes", "edges", "digest") || g["protocol"] != "symphony.graph.exchange.v1" {
		return shvFail()
	}
	if e := shvSealed(g); e != nil {
		return e
	}
	o := shvMap(g["owner"])
	a := shvMap(g["owner_artifact"])
	if !shvFields(o, "engine_id", "engine_version", "artifact_protocol", "artifact_digest") || !shvBoundedText(o["engine_id"], 128) || !shvBoundedText(o["engine_version"], 128) || !shvBoundedText(o["artifact_protocol"], 128) || o["artifact_protocol"] != a["protocol"] || o["artifact_digest"] != a["digest"] {
		return shvFail()
	}
	if e := shvSealed(a); e != nil {
		return e
	}
	nodes, ok := g["nodes"].([]any)
	edges, ok2 := g["edges"].([]any)
	if !ok || !ok2 || len(nodes) > 1024 || len(edges) > 2048 {
		return shvFail()
	}
	seen := map[string]bool{}
	last := ""
	for _, v := range nodes {
		n := shvMap(v)
		id := shvText(n["id"])
		if !shvFields(n, "id", "labels", "properties") || !shvBoundedText(id, 512) || id <= last || len(shvList(n["labels"])) > 32 || !shvStrings(n["labels"], true) || shvMap(n["properties"]) == nil {
			return shvFail()
		}
		for _, label := range shvList(n["labels"]) {
			if !shvBoundedText(label, 4096) {
				return shvFail()
			}
		}
		seen[id] = true
		last = id
	}
	last = ""
	for _, v := range edges {
		e := shvMap(v)
		id := shvText(e["id"])
		if !shvFields(e, "id", "from", "to", "label", "properties") || !shvBoundedText(id, 512) || id <= last || !seen[shvText(e["from"])] || !seen[shvText(e["to"])] || !shvBoundedText(e["label"], 128) || shvMap(e["properties"]) == nil {
			return shvFail()
		}
		last = id
	}
	return nil
}
func shvAdapterResult(op string, p map[string]any) (map[string]any, error) {
	g := shvMap(p["graph"])
	if e := shvGraph(g); e != nil {
		return nil, e
	}
	if op == "roundtrip" {
		if !shvFields(p, "graph") {
			return nil, shvFail()
		}
		return shvSealNew(map[string]any{"protocol": "symphony.graph.adapter-result.v1", "backend": "portable-reference", "operation": op, "graph": g}), nil
	}
	if !shvFields(p, "graph", "kind", "ids") || (p["kind"] != "nodes" && p["kind"] != "edges") {
		return nil, shvFail()
	}
	ids, ok := p["ids"].([]any)
	if !ok {
		return nil, shvFail()
	}
	if len(ids) > 2048 {
		return nil, shvFail()
	}
	for _, id := range ids {
		if !shvBoundedText(id, 512) {
			return nil, shvFail()
		}
	}
	rows, missing, e := shvSelect(shvList(g[shvText(p["kind"])]), ids)
	sort.Slice(rows, func(i, j int) bool { return shvText(shvMap(rows[i])["id"]) < shvText(shvMap(rows[j])["id"]) })
	if e != nil {
		return nil, e
	}
	return shvSealNew(map[string]any{"protocol": "symphony.graph.adapter-result.v1", "backend": "portable-reference", "operation": op, "graph_digest": g["digest"], "kind": p["kind"], "ids": ids, "rows": rows, "missing_ids": missing}), nil
}

// ValidateSHVResult independently rederives caller/result correspondence. The
// portable adapter verifies graph structure only, never primary-source truth.
func ValidateSHVResult(op string, input, raw []byte, adapter bool) error {
	return ValidateSHVResultVersion(op, input, raw, adapter, SHVVersion)
}
func ValidateSHVResultVersion(op string, input, raw []byte, adapter bool, version string) error {
	if (adapter && version != SHVVersion) || (!adapter && !shvKernelVersion(version)) {
		return shvFail()
	}
	p, e := shvObject(input)
	if e != nil {
		return e
	}
	r, e := shvObject(raw)
	if e != nil {
		return e
	}
	protocol, ok := SHVResultProtocol(op, adapter)
	if !ok || r["protocol"] != protocol {
		return shvFail()
	}
	if op == "inspect" {
		return shvDescriptorVersion(p, r, adapter, version)
	}
	if e = shvSealed(r); e != nil {
		return e
	}
	var expected map[string]any
	if adapter {
		expected, e = shvAdapterResult(op, p)
	} else {
		switch op {
		case "coverage_default":
			if !shvFields(p, "as_of") || !shvDate(p["as_of"]) || shvText(p["as_of"]) < "2018-01-01" {
				return shvFail()
			}
			expected = shvSealNew(map[string]any{"protocol": protocol, "as_of": p["as_of"], "selector": map[string]any{"op": "date", "basis": "model_introduction", "from": "2018-01-01", "through": p["as_of"]}})
		case "coverage_plan":
			expected, e = shvCoverage(p)
		case "catalogue_build":
			return shvValidateBuildVersion(p, r, version)
		case "catalogue_query":
			if !shvFields(p, "source_root", "catalogue", "subject_ids") {
				return shvFail()
			}
			expected, e = shvQueryVersion(p, version)
		case "evaluate":
			if !shvFields(p, "source_root", "catalogue", "subject_ids", "requirements") {
				return shvFail()
			}
			expected, e = shvEvaluationVersion(p, version)
		case "graph_project":
			if !shvFields(p, "source_root", "catalogue") {
				return shvFail()
			}
			expected, e = shvProjectionVersion(shvMap(p["catalogue"]), version)
		case "graph_validate":
			if !shvFields(p, "source_root", "graph") {
				return shvFail()
			}
			g := shvMap(p["graph"])
			c := shvMap(g["owner_artifact"])
			projection, err := shvProjectionVersion(c, version)
			if err != nil {
				return err
			}
			if !scvEqual(projection, g) {
				return shvFail()
			}
			expected = shvSealNew(map[string]any{"protocol": protocol, "graph_digest": g["digest"], "catalogue_digest": c["digest"], "valid": true})
		}
	}
	if e != nil {
		return e
	}
	if !adapter && op != "coverage_default" && op != "coverage_plan" {
		c := shvMap(p["catalogue"])
		if op == "graph_validate" {
			c = shvMap(shvMap(p["graph"])["owner_artifact"])
		}
		if err := shvValidateBuildVersion(map[string]any{"source_root": p["source_root"], "sources": c["sources"], "subjects": c["mapping"]}, c, version); err != nil {
			return err
		}
	}
	if !scvEqual(expected, r) {
		return shvFail()
	}
	return nil
}
func shvDescriptor(p, r map[string]any, adapter bool) error {
	return shvDescriptorVersion(p, r, adapter, SHVVersion)
}
func shvDescriptorVersion(p, r map[string]any, adapter bool, version string) error {
	if len(p) != 0 || !shvFields(r, "protocol", "format_version", "module_id", "engine_id", "vector_id", "engine_version", "process_protocols", "contract_versions", "operations", "limits", "supported_scopes", "language", "thermal_path", "canonical_apply_enabled", "session_mutation_enabled", "network_listener", "descriptor_digest") {
		return shvFail()
	}
	if e := scvSeal(r, "descriptor_digest"); e != nil {
		return e
	}
	s := shvEngineSpec
	if adapter {
		s = shvAdapterSpec
	}
	if r["module_id"] != s.moduleID || r["engine_id"] != s.engineID || r["vector_id"] != "shv" || r["engine_version"] != version || r["format_version"] != json.Number("2") || r["language"] != "C++26" || r["thermal_path"] != "freezing" || r["canonical_apply_enabled"] != false || r["session_mutation_enabled"] != false || r["network_listener"] != false || !scvEqual(r["process_protocols"], []any{processProtocol}) || !scvEqual(r["supported_scopes"], []any{"user"}) {
		return shvFail()
	}
	limits := map[string]any{"request_bytes": 1048576, "response_bytes": 4194304, "json_depth": 64, "json_values": 32768, "path_bytes": 4096, "snapshot_files": 1024, "snapshot_file_bytes": 4194304, "deadline_ahead_ms": 300000}
	contracts := []any{"knowledge/shv/SPEC.md@v1", "symphony.shv.catalogue.v1", "symphony.graph.exchange.v1"}
	if adapter {
		contracts = []any{"shv-graph-adapter/SPEC.md@v1"}
	}
	if !scvEqual(r["limits"], limits) || !scvEqual(r["contract_versions"], contracts) {
		return shvFail()
	}
	ops, ok := r["operations"].([]any)
	count := len(shvOutputs)
	if adapter {
		count = 3
	}
	if !ok || len(ops) != count {
		return shvFail()
	}
	seen := map[string]bool{}
	domain := "shv"
	feature := "ssfv:symphony:shv-engine"
	if adapter {
		domain = "shv-graph-adapter"
		feature = "ssfv:symphony:shv-graph-adapter"
	}
	for _, v := range ops {
		o := shvMap(v)
		name := shvText(o["operation_name"])
		out, ok := SHVResultProtocol(name, adapter)
		interaction := "invoke"
		if name == "inspect" {
			interaction = "inspect"
		}
		if name == "catalogue_query" || (adapter && name == "query") {
			interaction = "query"
		}
		if !shvFields(o, "engine_operation_id", "operation_name", "availability", "feature_ids", "administrative_interactions", "administration_disposition", "input_protocol", "output_protocol", "mutability", "idempotency", "expected_state_required", "authorization_requirement", "recovery_operation_id", "direct_invocation", "thermal_path") || !scvEqual(o["administrative_interactions"], []any{interaction}) || o["administration_disposition"] != "qxctl_required" || o["mutability"] != "read_only" || o["idempotency"] != "idempotent" || o["recovery_operation_id"] != nil || o["direct_invocation"] != "supported" {
			return shvFail()
		}
		if !ok || seen[name] || o["engine_operation_id"] != "engop:symphony:"+domain+"."+strings.ReplaceAll(name, "_", ".") || o["availability"] != "implemented" || !scvEqual(o["feature_ids"], []any{feature}) || o["input_protocol"] != SHVInputProtocol(name, adapter) || o["output_protocol"] != out || o["authorization_requirement"] != "none" || o["expected_state_required"] != false || o["thermal_path"] != "freezing" {
			return shvFail()
		}
		seen[name] = true
	}
	return nil
}

package knowledgeengine

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func shvDiagnosticError(v any, failed bool) bool {
	if !failed {
		return v == nil
	}
	e := shvMap(v)
	// Native detail is informational. Go independently verifies failure versus
	// success and the affected field, rather than inferring meaning from prose.
	return shvFields(e, "code", "message") && shvBoundedText(e["code"], 128) && shvBoundedText(e["message"], 8192)
}
func shvDiagnosticSources(v any) error {
	sources, ok := partArray(v, 8)
	if !ok {
		return shvFail()
	}
	ids, paths := map[string]bool{}, map[string]bool{}
	var total int64
	for _, v := range sources {
		s := shvMap(v)
		id, path := shvText(s["id"]), shvText(s["path"])
		n, e := storeInteger(s["bytes"], 0, 1<<20)
		if !shvFields(s, "id", "path", "bytes", "digest", "format") || !shvID(id) || ids[id] || !safeRelativePath(path) || paths[path] || !storeDigest(s["digest"]) || e != nil || (s["format"] != "html" && s["format"] != "opaque") {
			return shvFail()
		}
		ids[id] = true
		paths[path] = true
		total += n
	}
	if total > 4<<20 {
		return shvFail()
	}
	return nil
}
func shvDiagnosticValue(p, m, f map[string]any, raw string, format any) (any, error) {
	if !shvMappingShape(m, SHVDocumentVersion) || !shvFieldShape(f, m, SHVDocumentVersion) {
		return nil, shvFail()
	}
	if m["interpretation_profile"] == "pdf_opn.v1" {
		if e := validateSHVDocumentSubject(p, m, map[string]any{"assertions": []any{}}, SHVDocumentVersion); e != nil {
			return nil, e
		}
		for _, v := range shvList(shvMap(shvMap(m["document"])["extraction"])["rows"]) {
			row := shvMap(v)
			if row["model"] == m["model"] {
				return row["opn"], nil
			}
		}
		return nil, shvFail()
	}
	if format != "html" {
		return nil, shvFail()
	}
	if m["interpretation_profile"] == "scoped_tables.v1" {
		rows, e := shvHTMLTable(raw, shvText(m["model"]), shvText(m["heading_section"]), shvText(f["section"]))
		if e != nil {
			return nil, e
		}
		return shvTableValue(rows, f)
	}
	pairs, e := shvHTMLPairs(raw, shvText(m["model"]), shvText(m["heading_section"]), shvText(m["field_section"]))
	if e != nil {
		return nil, e
	}
	for i, pair := range pairs {
		if pair[0] == f["label"] {
			if i+1 >= len(pairs) || pairs[i+1][0] != f["next_label"] {
				return nil, shvFail()
			}
			return shvSourceValue(pair[1], shvText(f["value_type"]))
		}
	}
	return nil, shvFail()
}
func shvExtractionDiagnose(p, r map[string]any) (map[string]any, error) {
	if !shvFields(p, "source_root", "sources", "subjects") || !shvProfileRoot(p["source_root"]) || shvDiagnosticSources(p["sources"]) != nil || shvProfileMappings(p["subjects"], false) != nil {
		return nil, shvFail()
	}
	sourceResults, ok := partArray(r["sources"], 8)
	if !ok || len(sourceResults) != len(shvList(p["sources"])) {
		return nil, shvFail()
	}
	sources := []any{}
	raws := map[string]string{}
	formats := map[string]any{}
	verified := map[string]bool{}
	for i, v := range shvList(p["sources"]) {
		s := shvMap(v)
		id := shvText(s["id"])
		formats[id] = s["format"]
		raw, e := readNoFollowRelative("/", strings.TrimPrefix(filepath.Join(shvText(p["source_root"]), shvText(s["path"])), "/"), 1<<20)
		n, _ := s["bytes"].(json.Number).Int64()
		valid := e == nil && int64(len(raw)) == n && digestBytes(raw) == s["digest"]
		verified[id] = valid
		raws[id] = string(raw)
		actual := shvMap(sourceResults[i])
		if !shvDiagnosticError(actual["error"], !valid) {
			return nil, shvFail()
		}
		status := "unverified"
		if valid {
			status = "verified"
		}
		sources = append(sources, map[string]any{"source_id": id, "status": status, "error": actual["error"]})
	}
	subjectResults, ok := partArray(r["subjects"], 32)
	if !ok || len(subjectResults) != len(shvList(p["subjects"])) {
		return nil, shvFail()
	}
	subjects := []any{}
	counts := map[string]int{"extracted": 0, "failed": 0, "unavailable": 0}
	for i, v := range shvList(p["subjects"]) {
		m := shvMap(v)
		sid := shvText(m["source_id"])
		if _, ok := formats[sid]; !ok {
			return nil, shvFail()
		}
		if _, portable := shvMap(m["document"])["decoder_binding"]; portable {
			return nil, shvFail()
		}
		fields := []any{}
		actual, ok := partArray(shvMap(subjectResults[i])["fields"], 16)
		if !ok || len(actual) != len(shvList(m["fields"])) {
			return nil, shvFail()
		}
		for j, fv := range shvList(m["fields"]) {
			f := shvMap(fv)
			status := "unavailable"
			var assertion any
			var e error
			if verified[sid] {
				var value any
				value, e = shvDiagnosticValue(p, m, f, raws[sid], formats[sid])
				status = "failed"
				if e == nil {
					status = "extracted"
					assertion = map[string]any{"predicate": f["predicate"], "source_id": sid, "qualifier": f["qualifier"], "value": value}
				}
			}
			a := shvMap(actual[j])
			if !shvDiagnosticError(a["error"], status == "failed") {
				return nil, shvFail()
			}
			counts[status]++
			fields = append(fields, map[string]any{"predicate": f["predicate"], "status": status, "assertion": assertion, "error": a["error"]})
		}
		subjects = append(subjects, map[string]any{"subject_id": m["id"], "source_id": sid, "fields": fields})
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.extraction-diagnostics.v1", "input": p, "reader": map[string]any{"engine_id": "symphony-shv", "version": SHVDocumentVersion, "mode": "compiled_exact_contract"}, "sources": sources, "subjects": subjects, "counts": counts, "evidence_scope": "individual_field_source_replay", "canonical_apply_enabled": false}), nil
}
func shvReferencePointer(v any, pointer string) (any, error) {
	if !strings.HasPrefix(pointer, "/") {
		return nil, shvFail()
	}
	for _, key := range strings.Split(pointer[1:], "/") {
		for i := 0; i < len(key); i++ {
			if key[i] == '~' {
				if i+1 >= len(key) || (key[i+1] != '0' && key[i+1] != '1') {
					return nil, shvFail()
				}
				i++
			}
		}
		key = strings.NewReplacer("~1", "/", "~0", "~").Replace(key)
		switch x := v.(type) {
		case map[string]any:
			var ok bool
			v, ok = x[key]
			if !ok {
				return nil, shvFail()
			}
		case []any:
			i, e := strconv.Atoi(key)
			if e != nil || i < 0 || i >= len(x) || strconv.Itoa(i) != key {
				return nil, shvFail()
			}
			v = x[i]
		default:
			return nil, shvFail()
		}
	}
	return v, nil
}
func shvReferencesAnalyze(p map[string]any) (map[string]any, error) {
	if !shvFields(p, "objects", "edges", "root_ids", "candidate_ids") {
		return nil, shvFail()
	}
	rows, ok := partArray(p["objects"], 128)
	if !ok {
		return nil, shvFail()
	}
	objects := map[string]map[string]any{}
	uninspected := []string{}
	for _, v := range rows {
		o := shvMap(v)
		id := shvText(o["id"])
		if !shvFields(o, "id", "kind", "digest", "document") || !shvID(id) || !shvID(o["kind"]) || !storeDigest(o["digest"]) || objects[id] != nil {
			return nil, shvFail()
		}
		if o["document"] != nil {
			d := shvMap(o["document"])
			if d == nil {
				return nil, shvFail()
			}
			if x, has := d["digest"]; has {
				if shvSealed(d) != nil || x != o["digest"] {
					return nil, shvFail()
				}
			} else {
				h, e := SCVDigest(d)
				if e != nil || h != o["digest"] {
					return nil, shvFail()
				}
			}
		} else {
			uninspected = append(uninspected, id)
		}
		objects[id] = o
	}
	sort.Strings(uninspected)
	for _, key := range []string{"root_ids", "candidate_ids"} {
		a, ok := partArray(p[key], 128)
		if !ok || !shvStrings(a, false) {
			return nil, shvFail()
		}
		for _, v := range a {
			if objects[shvText(v)] == nil {
				return nil, shvFail()
			}
		}
	}
	edges, ok := partArray(p["edges"], 512)
	if !ok {
		return nil, shvFail()
	}
	edges = append([]any{}, edges...)
	seen := map[string]bool{}
	adj := map[string]map[string]bool{}
	for _, v := range edges {
		e := shvMap(v)
		from, to, pointer := shvText(e["from"]), shvText(e["to"]), shvText(e["pointer"])
		if !shvFields(e, "from", "to", "pointer") || objects[from] == nil || objects[to] == nil || !shvBoundedText(pointer, 4096) || objects[from]["document"] == nil {
			return nil, shvFail()
		}
		key := from + "\x00" + pointer + "\x00" + to
		if seen[key] {
			return nil, shvFail()
		}
		seen[key] = true
		value, err := shvReferencePointer(objects[from]["document"], pointer)
		if err != nil || value != objects[to]["digest"] {
			return nil, shvFail()
		}
		if adj[from] == nil {
			adj[from] = map[string]bool{}
		}
		adj[from][to] = true
	}
	sort.Slice(edges, func(i, j int) bool {
		a, b := shvMap(edges[i]), shvMap(edges[j])
		for _, k := range []string{"from", "pointer", "to"} {
			if a[k] != b[k] {
				return shvText(a[k]) < shvText(b[k])
			}
		}
		return false
	})
	queue := []string{}
	for _, v := range shvList(p["root_ids"]) {
		queue = append(queue, shvText(v))
	}
	sort.Strings(queue)
	paths := map[string][]string{}
	for _, id := range queue {
		paths[id] = []string{id}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		next := []string{}
		for to := range adj[id] {
			next = append(next, to)
		}
		sort.Strings(next)
		for _, to := range next {
			if paths[to] == nil {
				paths[to] = append(append([]string{}, paths[id]...), to)
				queue = append(queue, to)
			}
		}
	}
	reachable := []string{}
	for id := range paths {
		reachable = append(reachable, id)
	}
	sort.Strings(reachable)
	candidates := []any{}
	for _, v := range shvList(p["candidate_ids"]) {
		id := shvText(v)
		status := "not_reachable_in_supplied_graph"
		var path any
		if paths[id] != nil {
			status = "reachable"
			path = paths[id]
		}
		incoming := []any{}
		for _, e := range edges {
			if shvMap(e)["to"] == id {
				incoming = append(incoming, e)
			}
		}
		candidates = append(candidates, map[string]any{"id": id, "status": status, "path": path, "incoming_edges": incoming})
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.reference-analysis.v1", "input": p, "edges": edges, "reachable_ids": reachable, "candidates": candidates, "uninspected_object_ids": uninspected, "evidence_scope": "caller_selected_reference_edges", "deletion_authorized": false, "canonical_apply_enabled": false}), nil
}

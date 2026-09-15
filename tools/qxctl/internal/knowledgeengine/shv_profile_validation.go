package knowledgeengine

import (
	"path/filepath"
	"sort"
)

func shvProfileType(v any) bool {
	return v == "string" || v == "integer" || v == "date" || v == "tokens" || v == "quarter_20yy" || v == "table_rows"
}
func shvProfileRoot(v any) bool {
	s := shvText(v)
	return shvBoundedText(v, 4096) && filepath.IsAbs(s) && filepath.Clean(s) == s
}
func shvProfileCompile(d map[string]any) (map[string]any, error) {
	if !shvFields(d, "id", "revision", "hardware_class", "metrics", "extensions") || !shvID(d["id"]) || !shvID(d["revision"]) || !shvID(d["hardware_class"]) || shvMap(d["extensions"]) == nil {
		return nil, shvFail()
	}
	metrics, ok := partArray(d["metrics"], 16)
	if !ok {
		return nil, shvFail()
	}
	seen := map[string]bool{}
	for _, v := range metrics {
		m := shvMap(v)
		id := shvText(m["predicate"])
		_, required := m["required"].(bool)
		if !shvFields(m, "predicate", "value_type", "qualifier", "required", "description", "extensions") || !shvID(id) || seen[id] || !shvProfileType(m["value_type"]) || !shvBoundedText(m["qualifier"], 256) || !shvBoundedText(m["description"], 4096) || !required || shvMap(m["extensions"]) == nil {
			return nil, shvFail()
		}
		seen[id] = true
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.class-profile.v1", "definition": d}), nil
}
func shvProfileCheck(v any) error {
	p := shvMap(v)
	if !shvFields(p, "protocol", "definition", "digest") {
		return shvFail()
	}
	want, e := shvProfileCompile(shvMap(p["definition"]))
	if e != nil || !scvEqual(want, p) {
		return shvFail()
	}
	return nil
}
func shvProfileMappings(v any, portable bool) error {
	rows, ok := partArray(v, 32)
	if !ok {
		return shvFail()
	}
	ids := map[string]bool{}
	count := 0
	for _, v := range rows {
		m := shvMap(v)
		pdf := m["interpretation_profile"] == "pdf_opn.v1"
		_, tagged := m["interpretation_profile"]
		tables := tagged && !pdf
		if pdf {
			if !shvFields(m, "id", "manufacturer", "model", "hardware_class", "source_id", "heading_section", "interpretation_profile", "document", "fields") || m["heading_section"] != "table8" {
				return shvFail()
			}
			d := shvMap(m["document"])
			_, binding := d["decoder_binding"]
			if portable || binding {
				if !shvFields(d, "decoder_binding", "extraction") || !shvID(d["decoder_binding"]) {
					return shvFail()
				}
			} else if !shvFields(d, "decoder_root", "extraction") || !shvProfileRoot(d["decoder_root"]) {
				return shvFail()
			}
			if shvMap(d["extraction"]) == nil {
				return shvFail()
			}
		} else if tables {
			if !shvFields(m, "id", "manufacturer", "model", "hardware_class", "source_id", "heading_section", "interpretation_profile", "fields") || m["interpretation_profile"] != "scoped_tables.v1" {
				return shvFail()
			}
		} else if !shvFields(m, "id", "manufacturer", "model", "hardware_class", "source_id", "heading_section", "field_section", "fields") || !shvBoundedText(m["field_section"], 128) {
			return shvFail()
		}
		id := shvText(m["id"])
		fs, ok := partArray(m["fields"], 16)
		if !shvID(id) || ids[id] || !shvID(m["source_id"]) || !shvID(m["hardware_class"]) || !shvBoundedText(m["manufacturer"], 256) || !shvBoundedText(m["model"], 256) || !shvBoundedText(m["heading_section"], 128) || !ok {
			return shvFail()
		}
		ids[id] = true
		count += len(fs)
		seen := map[string]bool{}
		for _, fv := range fs {
			f := shvMap(fv)
			pred := shvText(f["predicate"])
			if !shvID(pred) || seen[pred] || !shvProfileType(f["value_type"]) || !shvBoundedText(f["qualifier"], 256) {
				return shvFail()
			}
			seen[pred] = true
			if tables && f["value_type"] == "table_rows" {
				if !shvFields(f, "predicate", "section", "columns", "value_type", "qualifier") || !shvTableColumns(f["columns"]) {
					return shvFail()
				}
			} else {
				keys := []string{"predicate", "label", "next_label", "value_type", "qualifier"}
				if tables {
					keys = append(keys, "section")
				}
				if !shvFields(f, keys...) || !shvBoundedText(f["label"], 256) || ((!tables || f["next_label"] != nil) && !shvBoundedText(f["next_label"], 256)) {
					return shvFail()
				}
			}
			if tables && !shvBoundedText(f["section"], 128) {
				return shvFail()
			}
			if !tables && (f["value_type"] == "quarter_20yy" || f["value_type"] == "table_rows") {
				return shvFail()
			}
			if pdf && (f["label"] != "OPN" || f["next_label"] != "Model" || f["value_type"] != "string" || f["qualifier"] != "issuer=AMD;namespace=opn;profile=1") {
				return shvFail()
			}
			if pred == "model_introduction" && f["value_type"] != "date" && !(tables && f["value_type"] == "quarter_20yy") {
				return shvFail()
			}
		}
	}
	if count > 256 {
		return shvFail()
	}
	return nil
}
func shvProfileDiagnose(p map[string]any) (map[string]any, error) {
	if !shvFields(p, "profile", "mapping") || shvProfileCheck(p["profile"]) != nil || shvProfileMappings(p["mapping"], false) != nil {
		return nil, shvFail()
	}
	d := shvMap(shvMap(p["profile"])["definition"])
	rows := []any{}
	counts := map[string]int{"conformant": 0, "incomplete": 0, "not_applicable": 0}
	for _, v := range shvList(p["mapping"]) {
		m := shvMap(v)
		findings := []any{}
		extensions := []string{}
		status := "not_applicable"
		if m["hardware_class"] == d["hardware_class"] {
			status = "conformant"
			actual := map[string]map[string]any{}
			known := map[string]bool{}
			for _, f := range shvList(m["fields"]) {
				x := shvMap(f)
				actual[shvText(x["predicate"])] = x
			}
			for _, v := range shvList(d["metrics"]) {
				metric := shvMap(v)
				pred := shvText(metric["predicate"])
				known[pred] = true
				f, exists := actual[pred]
				var observed any
				problems := []string{}
				state := "unmapped_optional"
				if !exists {
					if metric["required"] == true {
						state = "unmapped_required"
					}
				} else {
					observed = map[string]any{"value_type": f["value_type"], "qualifier": f["qualifier"]}
					for _, k := range []string{"value_type", "qualifier"} {
						if f[k] != metric[k] {
							problems = append(problems, k)
						}
					}
					state = "matched"
					if len(problems) > 0 {
						state = "mismatch"
					}
				}
				if state == "unmapped_required" || state == "mismatch" {
					status = "incomplete"
				}
				findings = append(findings, map[string]any{"predicate": pred, "status": state, "expected": metric, "observed": observed, "differences": problems})
			}
			for pred := range actual {
				if !known[pred] {
					extensions = append(extensions, pred)
				}
			}
			sort.Strings(extensions)
		}
		counts[status]++
		rows = append(rows, map[string]any{"subject_id": m["id"], "status": status, "findings": findings, "extension_predicates": extensions})
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.mapping-diagnostics.v1", "input": p, "subjects": rows, "counts": counts, "evidence_scope": "mapping_declarations_only"}), nil
}
func shvUniverseBuild(p map[string]any) (map[string]any, error) {
	if !shvFields(p, "id", "revision", "kernel_version", "coverage", "profiles", "sources", "mapping", "locators", "extensions") || !shvID(p["id"]) || !shvID(p["revision"]) || p["kernel_version"] != SHVDocumentVersion || shvMap(p["extensions"]) == nil || shvProfileMappings(p["mapping"], true) != nil {
		return nil, shvFail()
	}
	if _, e := shvCoverage(map[string]any{"profile": p["coverage"], "subjects": []any{}}); e != nil {
		return nil, e
	}
	profiles, ok := partArray(p["profiles"], 16)
	if !ok {
		return nil, shvFail()
	}
	seen := map[string]bool{}
	for _, pr := range profiles {
		id := shvText(shvMap(shvMap(pr)["definition"])["id"])
		if shvProfileCheck(pr) != nil || seen[id] {
			return nil, shvFail()
		}
		seen[id] = true
	}
	sources, ok := partArray(p["sources"], 8)
	if !ok {
		return nil, shvFail()
	}
	ids, paths := map[string]bool{}, map[string]bool{}
	var total int64
	for _, v := range sources {
		s := shvMap(v)
		id, path := shvText(s["id"]), shvText(s["path"])
		n, e := storeInteger(s["bytes"], 0, 1<<20)
		if !shvFields(s, "id", "path", "bytes", "digest", "format") || !shvID(id) || ids[id] || !safeRelativePath(path) || paths[path] || !storeDigest(s["digest"]) || e != nil || (s["format"] != "html" && s["format"] != "opaque") {
			return nil, shvFail()
		}
		ids[id] = true
		paths[path] = true
		total += n
	}
	if total > 4<<20 {
		return nil, shvFail()
	}
	for _, v := range shvList(p["mapping"]) {
		if !ids[shvText(shvMap(v)["source_id"])] {
			return nil, shvFail()
		}
	}
	locators, ok := partArray(p["locators"], 8)
	if !ok {
		return nil, shvFail()
	}
	seen = map[string]bool{}
	for _, v := range locators {
		l := shvMap(v)
		id := shvText(l["source_id"])
		if !shvFields(l, "source_id", "uri", "upstream_revision") || !ids[id] || seen[id] || !shvBoundedText(l["uri"], 4096) || (l["upstream_revision"] != nil && !shvBoundedText(l["upstream_revision"], 256)) {
			return nil, shvFail()
		}
		seen[id] = true
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.universe.v1", "definition": p}), nil
}
func shvUniverseBind(p, r map[string]any) (map[string]any, error) {
	if !shvFields(p, "universe", "bindings") {
		return nil, shvFail()
	}
	u := shvMap(p["universe"])
	want, e := shvUniverseBuild(shvMap(u["definition"]))
	if e != nil || !scvEqual(u, want) {
		return nil, shvFail()
	}
	d := shvMap(u["definition"])
	b := shvMap(p["bindings"])
	decoders := shvMap(b["decoders"])
	if !shvFields(b, "source_root", "decoders") || !shvProfileRoot(b["source_root"]) || decoders == nil {
		return nil, shvFail()
	}
	raw, _ := SCVCanonical(map[string]any{"mapping": d["mapping"]})
	copy, _ := shvObject(raw)
	mapping := shvList(copy["mapping"])
	used := map[string]bool{}
	for _, v := range mapping {
		m := shvMap(v)
		if m["interpretation_profile"] == "pdf_opn.v1" {
			doc := shvMap(m["document"])
			key := shvText(doc["decoder_binding"])
			if !shvProfileRoot(decoders[key]) {
				return nil, shvFail()
			}
			used[key] = true
			delete(doc, "decoder_binding")
			doc["decoder_root"] = decoders[key]
		}
	}
	if len(used) != len(decoders) {
		return nil, shvFail()
	}
	input := map[string]any{"source_root": b["source_root"], "sources": d["sources"], "subjects": mapping}
	c := shvMap(r["catalogue"])
	if e = shvValidateBuildVersion(input, c, SHVDocumentVersion); e != nil {
		return nil, e
	}
	summaries := []any{}
	for _, v := range shvList(c["subjects"]) {
		s := shvMap(v)
		summaries = append(summaries, map[string]any{"id": s["id"], "manufacturer": s["manufacturer"], "model": s["model"], "hardware_class": s["hardware_class"], "introduced": s["introduced"]})
	}
	coverage, e := shvCoverage(map[string]any{"profile": d["coverage"], "subjects": summaries})
	if e != nil {
		return nil, e
	}
	conformance := []any{}
	declared := map[string]bool{}
	classes := map[string]bool{}
	for _, pr := range shvList(d["profiles"]) {
		declared[shvText(shvMap(shvMap(pr)["definition"])["hardware_class"])] = true
		x, e := shvProfileDiagnose(map[string]any{"profile": pr, "mapping": mapping})
		if e != nil {
			return nil, e
		}
		conformance = append(conformance, x)
	}
	for _, v := range mapping {
		classes[shvText(shvMap(v)["hardware_class"])] = true
	}
	unprofiled := []string{}
	for c := range classes {
		if !declared[c] {
			unprofiled = append(unprofiled, c)
		}
	}
	sort.Strings(unprofiled)
	return shvSealNew(map[string]any{"protocol": "symphony.shv.universe-binding.v1", "input": p, "reader": map[string]any{"engine_id": "symphony-shv", "version": SHVDocumentVersion, "mode": "compiled_exact_contract"}, "catalogue_input": input, "catalogue": c, "coverage": coverage, "conformance": conformance, "unprofiled_classes": unprofiled, "canonical_apply_enabled": false}), nil
}
func ValidateSHVProfileResult(op string, input, result []byte) error {
	return ValidateSHVProfileResultVersion(op, input, result, SHVProfileVersion)
}
func ValidateSHVProfileResultVersion(op string, input, result []byte, version string) error {
	if shvProfileAdmission[version] == nil {
		return shvFail()
	}
	if !shvProfileAdmission[version][op] {
		return shvFail()
	}
	p, e := shvObject(input)
	if e != nil {
		return e
	}
	r, e := shvObject(result)
	if e != nil {
		return e
	}
	if op == "inspect" {
		return shvProfileDescriptor(p, r, version)
	}
	var want map[string]any
	switch op {
	case "extraction_diagnose":
		want, e = shvExtractionDiagnose(p, r)
	case "references_analyze":
		want, e = shvReferencesAnalyze(p)
	case "profile_compile":
		want, e = shvProfileCompile(p)
	case "mapping_diagnose":
		want, e = shvProfileDiagnose(p)
	case "universe_build":
		want, e = shvUniverseBuild(p)
	case "universe_bind":
		want, e = shvUniverseBind(p, r)
	default:
		return shvFail()
	}
	if e != nil {
		return e
	}
	if !scvEqual(want, r) {
		return shvFail()
	}
	return nil
}

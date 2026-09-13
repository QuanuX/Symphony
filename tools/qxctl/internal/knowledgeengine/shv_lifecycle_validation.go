package knowledgeengine

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var shvLifecycleHost = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`)

func shvLifecycleURI(v any) bool {
	s, ok := v.(string)
	if !ok || len(s) > 4096 {
		return false
	}
	tail := ""
	if strings.HasPrefix(s, "https://") {
		tail = s[8:]
	} else if strings.HasPrefix(s, "http://") {
		tail = s[7:]
	} else {
		return false
	}
	for _, c := range s {
		if c < 33 || c > 126 || strings.ContainsRune("\\#\"<>", c) {
			return false
		}
	}
	authority := strings.SplitN(strings.SplitN(tail, "/", 2)[0], "?", 2)[0]
	parts := strings.Split(authority, ":")
	if len(parts) > 2 || !shvLifecycleHost.MatchString(parts[0]) {
		return false
	}
	if len(parts) == 2 {
		n, e := strconv.Atoi(parts[1])
		if e != nil || n < 1 || n > 65535 || strconv.Itoa(n) != parts[1] {
			return false
		}
	}
	return true
}
func shvLifecycleInteger(v any, min, max int64) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	x, e := n.Int64()
	return x, e == nil && x >= min && x <= max
}
func shvLifecycleDefinition(v any) error {
	d := shvMap(v)
	subjects, ok := d["subject_ids"].([]any)
	locators, ok2 := d["locators"].([]any)
	if !shvFields(d, "source_id", "publisher", "authority_role", "subject_ids", "locators") || !shvID(d["source_id"]) || !shvBoundedText(d["publisher"], 256) || !shvBoundedText(d["authority_role"], 256) || !ok || len(subjects) < 1 || len(subjects) > 32 || !shvStrings(subjects, false) || !ok2 || len(locators) < 1 || len(locators) > 16 {
		return shvFail()
	}
	for _, id := range subjects {
		if !shvID(id) {
			return shvFail()
		}
	}
	ids, uris := map[string]bool{}, map[string]bool{}
	for _, v := range locators {
		l := shvMap(v)
		id, uri := shvText(l["id"]), shvText(l["uri"])
		if !shvFields(l, "id", "uri", "format") || !shvID(id) || ids[id] || uris[uri] || !shvLifecycleURI(uri) || (l["format"] != "html" && l["format"] != "opaque") {
			return shvFail()
		}
		ids[id] = true
		uris[uri] = true
	}
	return nil
}
func shvLifecycleSource(v any) error {
	s := shvMap(v)
	n, ok := shvLifecycleInteger(s["generation"], 1, 32)
	if !shvFields(s, "protocol", "definition", "generation", "previous_digest", "digest") || s["protocol"] != "symphony.shv.source-revision.v1" || !ok || (n == 1 && s["previous_digest"] != nil) || (n > 1 && !taggedSHA256(shvText(s["previous_digest"]))) {
		return shvFail()
	}
	if e := shvLifecycleDefinition(s["definition"]); e != nil {
		return e
	}
	return shvSealed(s)
}
func shvLifecyclePlan(p map[string]any) (map[string]any, error) {
	if !shvFields(p, "operation_id", "current", "desired", "reason") || !shvID(p["operation_id"]) || !shvBoundedText(p["reason"], 4096) {
		return nil, shvFail()
	}
	if e := shvLifecycleDefinition(p["desired"]); e != nil {
		return nil, e
	}
	generation := int64(1)
	kind := "onboard"
	var previous any
	if p["current"] != nil {
		if e := shvLifecycleSource(p["current"]); e != nil {
			return nil, e
		}
		c := shvMap(p["current"])
		old, d := shvMap(c["definition"]), shvMap(p["desired"])
		n, _ := shvLifecycleInteger(c["generation"], 1, 32)
		if n == 32 || old["source_id"] != d["source_id"] || scvEqual(old, d) {
			return nil, shvFail()
		}
		generation = n + 1
		previous = c["digest"]
		kind = "relocation"
		for _, key := range []string{"publisher", "authority_role", "subject_ids"} {
			if !scvEqual(old[key], d[key]) {
				kind = "authority_change"
			}
		}
	}
	source := shvSealNew(map[string]any{"protocol": "symphony.shv.source-revision.v1", "definition": p["desired"], "generation": generation, "previous_digest": previous})
	return shvSealNew(map[string]any{"protocol": "symphony.shv.source-plan.v1", "operation_id": p["operation_id"], "expected_state_digest": previous, "change_kind": kind, "reason": p["reason"], "source": source}), nil
}
func shvLifecycleCapture(p map[string]any) (map[string]any, error) {
	if !shvFields(p, "source_root", "source", "locator_id", "resolved_uri", "redirect_chain", "observed_at", "upstream_revision", "manifest", "completeness", "issues") {
		return nil, shvFail()
	}
	root := shvText(p["source_root"])
	if !shvBoundedText(root, 4096) || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, shvFail()
	}
	if e := shvLifecycleSource(p["source"]); e != nil {
		return nil, e
	}
	var locator map[string]any
	for _, v := range shvList(shvMap(shvMap(p["source"])["definition"])["locators"]) {
		l := shvMap(v)
		if l["id"] == p["locator_id"] {
			locator = l
		}
	}
	if locator == nil || !shvLifecycleURI(p["resolved_uri"]) {
		return nil, shvFail()
	}
	chain, ok := p["redirect_chain"].([]any)
	if !ok || len(chain) > 8 || !shvStrings(chain, false) {
		return nil, shvFail()
	}
	last := locator["uri"]
	for _, u := range chain {
		if !shvLifecycleURI(u) || u == locator["uri"] {
			return nil, shvFail()
		}
		last = u
	}
	if p["resolved_uri"] != last {
		return nil, shvFail()
	}
	stamp := shvText(p["observed_at"])
	ts, e := time.Parse("2006-01-02T15:04:05Z", stamp)
	if e != nil || ts.Format("2006-01-02T15:04:05Z") != stamp || ts.Year() < 1 {
		return nil, shvFail()
	}
	if p["upstream_revision"] != nil {
		r := shvMap(p["upstream_revision"])
		if !shvFields(r, "scheme", "value") || !shvBoundedText(r["scheme"], 128) || !shvBoundedText(r["value"], 512) {
			return nil, shvFail()
		}
	}
	issues, ok := p["issues"].([]any)
	if !ok || len(issues) > 32 || !shvStrings(issues, false) || (p["completeness"] != "complete" && p["completeness"] != "partial") || (p["completeness"] == "partial" && len(issues) == 0) {
		return nil, shvFail()
	}
	for _, v := range issues {
		if !shvBoundedText(v, 512) {
			return nil, shvFail()
		}
	}
	m := shvMap(p["manifest"])
	size, ok := shvLifecycleInteger(m["bytes"], 0, 1<<20)
	path := shvText(m["path"])
	if !shvFields(m, "id", "path", "bytes", "digest", "format") || !shvID(m["id"]) || !ok || !safeRelativePath(path) || len(path) > 4096 || !taggedSHA256(shvText(m["digest"])) || m["format"] != locator["format"] {
		return nil, shvFail()
	}
	data, e := readNoFollowRelative("/", strings.TrimPrefix(filepath.Join(root, path), "/"), 1<<20)
	if e != nil {
		return nil, e
	}
	if int64(len(data)) != size || digestBytes(data) != m["digest"] {
		return nil, shvFail()
	}
	out := map[string]any{"protocol": "symphony.shv.source-capture.v1"}
	for k, v := range p {
		if k != "source_root" {
			out[k] = v
		}
	}
	return shvSealNew(out), nil
}
func shvLifecycleReplay(root any, v any) error {
	c := shvMap(v)
	if !shvFields(c, "protocol", "source", "locator_id", "resolved_uri", "redirect_chain", "observed_at", "upstream_revision", "manifest", "completeness", "issues", "digest") || c["protocol"] != "symphony.shv.source-capture.v1" {
		return shvFail()
	}
	if e := shvSealed(c); e != nil {
		return e
	}
	p := map[string]any{"source_root": root}
	for k, v := range c {
		if k != "protocol" && k != "digest" {
			p[k] = v
		}
	}
	expected, e := shvLifecycleCapture(p)
	if e != nil {
		return e
	}
	if !scvEqual(expected, c) {
		return shvFail()
	}
	return nil
}
func shvLifecycleGraph(root any, captures any) (map[string]any, error) {
	a, ok := captures.([]any)
	if !ok || len(a) > 8 {
		return nil, shvFail()
	}
	r := shvText(root)
	if !shvBoundedText(r, 4096) || !filepath.IsAbs(r) || filepath.Clean(r) != r {
		return nil, shvFail()
	}
	nodes, edges := []any{}, []any{}
	ids, digests, sources := map[string]bool{}, map[string]bool{}, map[string]bool{}
	total := int64(0)
	for _, v := range a {
		size, valid := shvLifecycleInteger(shvMap(shvMap(v)["manifest"])["bytes"], 0, 1<<20)
		if !valid {
			return nil, shvFail()
		}
		total += size
	}
	if total > 4<<20 {
		return nil, shvFail()
	}
	for _, v := range a {
		if e := shvLifecycleReplay(root, v); e != nil {
			return nil, e
		}
		c := shvMap(v)
		m := shvMap(c["manifest"])
		id, d := shvText(m["id"]), shvText(c["digest"])
		if ids[id] || digests[d] {
			return nil, shvFail()
		}
		ids[id] = true
		digests[d] = true
		s := shvMap(c["source"])
		sd := shvText(s["digest"])
		sid := "source-revision:" + strings.TrimPrefix(sd, "sha256:")
		if !sources[sd] {
			nodes = append(nodes, map[string]any{"id": sid, "labels": []any{"source_revision"}, "properties": s})
			sources[sd] = true
		}
		nodes = append(nodes, map[string]any{"id": "capture:" + id, "labels": []any{"source_capture"}, "properties": c})
		edges = append(edges, map[string]any{"id": "capture-source:" + id, "from": "capture:" + id, "to": sid, "label": "captured_under", "properties": map[string]any{"source_id": shvMap(s["definition"])["source_id"], "source_digest": sd}})
	}
	sort.Slice(nodes, func(i, j int) bool { return shvText(shvMap(nodes[i])["id"]) < shvText(shvMap(nodes[j])["id"]) })
	sort.Slice(edges, func(i, j int) bool { return shvText(shvMap(edges[i])["id"]) < shvText(shvMap(edges[j])["id"]) })
	bundle := shvSealNew(map[string]any{"protocol": "symphony.shv.source-bundle.v1", "captures": a})
	return shvSealNew(map[string]any{"protocol": "symphony.graph.exchange.v1", "owner": map[string]any{"engine_id": "symphony-shv-source", "engine_version": SHVSourceVersion, "artifact_protocol": bundle["protocol"], "artifact_digest": bundle["digest"]}, "owner_artifact": bundle, "nodes": nodes, "edges": edges}), nil
}
func shvLifecycleExpected(op string, p map[string]any) (map[string]any, error) {
	switch op {
	case "source_plan":
		return shvLifecyclePlan(p)
	case "source_reduce":
		plan := shvMap(p["plan"])
		if !shvFields(p, "current", "plan") || !shvFields(plan, "protocol", "operation_id", "expected_state_digest", "change_kind", "reason", "source", "digest") {
			return nil, shvFail()
		}
		expected, e := shvLifecyclePlan(map[string]any{"current": p["current"], "operation_id": plan["operation_id"], "desired": shvMap(plan["source"])["definition"], "reason": plan["reason"]})
		if e != nil {
			return nil, e
		}
		if !scvEqual(expected, plan) {
			return nil, shvFail()
		}
		return shvSealNew(map[string]any{"protocol": "symphony.shv.source-transition.v1", "operation_id": plan["operation_id"], "expected_state_digest": plan["expected_state_digest"], "source": plan["source"]}), nil
	case "source_status":
		a, ok := p["history"].([]any)
		if !shvFields(p, "history") || !ok || len(a) < 1 || len(a) > 32 {
			return nil, shvFail()
		}
		ds := []any{}
		var prev map[string]any
		for i, v := range a {
			if e := shvLifecycleSource(v); e != nil {
				return nil, e
			}
			s := shvMap(v)
			n, _ := shvLifecycleInteger(s["generation"], 1, 32)
			if n != int64(i+1) {
				return nil, shvFail()
			}
			if prev != nil && (s["previous_digest"] != prev["digest"] || shvMap(s["definition"])["source_id"] != shvMap(prev["definition"])["source_id"] || scvEqual(s["definition"], prev["definition"])) {
				return nil, shvFail()
			}
			ds = append(ds, s["digest"])
			prev = s
		}
		return shvSealNew(map[string]any{"protocol": "symphony.shv.source-status.v1", "source": prev, "history_digests": ds}), nil
	case "capture_import":
		return shvLifecycleCapture(p)
	case "capture_compare":
		if !shvFields(p, "source_root", "previous", "current") {
			return nil, shvFail()
		}
		for _, k := range []string{"previous", "current"} {
			if e := shvLifecycleReplay(p["source_root"], p[k]); e != nil {
				return nil, e
			}
		}
		a, b := shvMap(p["previous"]), shvMap(p["current"])
		sa, sb := shvMap(a["source"]), shvMap(b["source"])
		ma, mb := shvMap(a["manifest"]), shvMap(b["manifest"])
		same := shvMap(sa["definition"])["source_id"] == shvMap(sb["definition"])["source_id"]
		changes := []any{}
		pairs := []struct {
			name    string
			changed bool
		}{{"source_identity", !same}, {"source_revision", sa["digest"] != sb["digest"]}, {"body", ma["digest"] != mb["digest"] || !scvEqual(ma["bytes"], mb["bytes"])}, {"representation", ma["format"] != mb["format"]}, {"acquisition_route", a["locator_id"] != b["locator_id"] || a["resolved_uri"] != b["resolved_uri"] || !scvEqual(a["redirect_chain"], b["redirect_chain"])}, {"upstream_revision", !scvEqual(a["upstream_revision"], b["upstream_revision"])}, {"completeness", a["completeness"] != b["completeness"] || !scvEqual(a["issues"], b["issues"])}, {"observation_time", a["observed_at"] != b["observed_at"]}, {"manifest_identity", ma["id"] != mb["id"] || ma["path"] != mb["path"]}}
		for _, v := range pairs {
			if v.changed {
				changes = append(changes, v.name)
			}
		}
		return shvSealNew(map[string]any{"protocol": "symphony.shv.capture-comparison.v1", "previous_digest": a["digest"], "current_digest": b["digest"], "same_logical_source": same, "changes": changes}), nil
	case "graph_project":
		if !shvFields(p, "source_root", "captures") {
			return nil, shvFail()
		}
		return shvLifecycleGraph(p["source_root"], p["captures"])
	case "graph_validate":
		if !shvFields(p, "source_root", "graph") {
			return nil, shvFail()
		}
		g := shvMap(p["graph"])
		bundle := shvMap(g["owner_artifact"])
		expected, e := shvLifecycleGraph(p["source_root"], bundle["captures"])
		if e != nil {
			return nil, e
		}
		if !scvEqual(expected, g) {
			return nil, shvFail()
		}
		return shvSealNew(map[string]any{"protocol": "symphony.shv.source-graph-validation.v1", "graph_digest": g["digest"], "bundle_digest": bundle["digest"], "valid": true}), nil
	}
	return nil, shvFail()
}

// ValidateSHVSourceResult verifies supplied lineage and actual capture bytes;
// it does not authenticate the caller's authority or activate a source registry.
func ValidateSHVSourceResult(op string, input, raw []byte) error {
	p, e := shvObject(input)
	if e != nil {
		return e
	}
	r, e := shvObject(raw)
	if e != nil {
		return e
	}
	protocol, ok := SHVSourceResultProtocol(op)
	if !ok || r["protocol"] != protocol {
		return shvFail()
	}
	if op == "inspect" {
		return shvLifecycleDescriptor(p, r)
	}
	expected, e := shvLifecycleExpected(op, p)
	if e != nil {
		return e
	}
	if !scvEqual(expected, r) {
		return shvFail()
	}
	return nil
}

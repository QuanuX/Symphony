package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func lifecycleFixture(t *testing.T) (map[string]any, map[string]any, map[string]any, map[string]any) {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	body := []byte{0, 255, 1, 2}
	if e = os.WriteFile(filepath.Join(root, "body.bin"), body, 0600); e != nil {
		t.Fatal(e)
	}
	d := map[string]any{"source_id": "family-doc", "publisher": "Independent vendor", "authority_role": "caller-declared family document", "subject_ids": []any{"cpu-a", "cpu-b"}, "locators": []any{map[string]any{"id": "primary", "uri": "https://Vendor.example/family", "format": "opaque"}}}
	s := shvReseal(t, map[string]any{"protocol": "symphony.shv.source-revision.v1", "definition": d, "generation": 1, "previous_digest": nil})
	planInput := shvClone(t, map[string]any{"operation_id": "onboard-1", "current": nil, "desired": d, "reason": "Include independently selected family source"})
	plan := shvReseal(t, map[string]any{"protocol": "symphony.shv.source-plan.v1", "operation_id": "onboard-1", "expected_state_digest": nil, "change_kind": "onboard", "reason": planInput["reason"], "source": s})
	captureInput := shvClone(t, map[string]any{"source_root": root, "source": s, "locator_id": "primary", "resolved_uri": "https://cdn.example/family", "redirect_chain": []any{"https://cdn.example/family"}, "observed_at": "2026-09-13T12:34:56Z", "upstream_revision": nil, "manifest": map[string]any{"id": "snapshot-1", "path": "body.bin", "bytes": len(body), "digest": digestBytes(body), "format": "opaque"}, "completeness": "partial", "issues": []any{"publisher completeness unknown"}})
	capture := shvClone(t, captureInput)
	delete(capture, "source_root")
	capture["protocol"] = "symphony.shv.source-capture.v1"
	capture = shvReseal(t, capture)
	return planInput, plan, captureInput, capture
}
func TestSHVLifecycleLineage(t *testing.T) {
	p, plan, _, _ := lifecycleFixture(t)
	if e := ValidateSHVSourceResult("source_plan", shvRaw(t, p), shvRaw(t, plan)); e != nil {
		t.Fatal(e)
	}
	bad := shvClone(t, plan)
	bad["change_kind"] = "authority_change"
	bad = shvReseal(t, bad)
	if ValidateSHVSourceResult("source_plan", shvRaw(t, p), shvRaw(t, bad)) == nil {
		t.Fatal("resealed false classification")
	}
	current := shvMap(plan["source"])
	desired := shvClone(t, shvMap(current["definition"]))
	shvMap(shvList(desired["locators"])[0])["uri"] = "https://other.example/moved"
	move := shvClone(t, map[string]any{"current": current, "desired": desired, "operation_id": "move", "reason": "relocate"})
	r, e := shvLifecycleExpected("source_plan", move)
	if e != nil || r["change_kind"] != "relocation" {
		t.Fatal(e, r)
	}
	r = shvClone(t, r)
	shvMap(move["desired"])["publisher"] = "New authority"
	r2, e := shvLifecycleExpected("source_plan", shvClone(t, move))
	if e != nil || r2["change_kind"] != "authority_change" {
		t.Fatal(e, r2)
	}
	transition := shvReseal(t, map[string]any{"protocol": "symphony.shv.source-transition.v1", "operation_id": "move", "expected_state_digest": current["digest"], "source": r["source"]})
	reduce := map[string]any{"current": current, "plan": r}
	if e = ValidateSHVSourceResult("source_reduce", shvRaw(t, reduce), shvRaw(t, transition)); e != nil {
		t.Fatal(e)
	}
	for _, field := range []string{"expected_state_digest", "change_kind", "source"} {
		t.Run(field, func(t *testing.T) {
			bad := shvClone(t, r)
			if field == "source" {
				s := shvMap(bad[field])
				s["previous_digest"] = digestBytes([]byte("wrong predecessor"))
				bad[field] = shvReseal(t, s)
			} else {
				bad[field] = "forged"
			}
			bad = shvReseal(t, bad)
			if _, e := shvLifecycleExpected("source_reduce", map[string]any{"current": current, "plan": bad}); e == nil {
				t.Fatal("forged lineage accepted")
			}
		})
	}
	history := map[string]any{"history": []any{current, r["source"]}}
	if _, e = shvLifecycleExpected("source_status", history); e != nil {
		t.Fatal(e)
	}
	for _, h := range [][]any{{r["source"]}, {current, current}, {r["source"], current}} {
		if _, e = shvLifecycleExpected("source_status", map[string]any{"history": h}); e == nil {
			t.Fatal("noncontiguous history accepted")
		}
	}
	p["current"] = current
	if _, e = shvLifecycleExpected("source_plan", p); e == nil {
		t.Fatal("unchanged source accepted")
	}
}
func TestSHVLifecycleCaptureReplayAndProvenance(t *testing.T) {
	_, _, p, c := lifecycleFixture(t)
	if e := ValidateSHVSourceResult("capture_import", shvRaw(t, p), shvRaw(t, c)); e != nil {
		t.Fatal(e)
	}
	changes := map[string]any{"resolved_uri": "https://unrecorded.example/x", "redirect_chain": []any{"https://Vendor.example/family"}, "observed_at": "2026-02-30T12:00:00Z", "upstream_revision": map[string]any{"scheme": "etag"}, "issues": []any{}, "source_root": shvText(p["source_root"]) + "/../bad"}
	for k, v := range changes {
		t.Run(k, func(t *testing.T) {
			bad := shvClone(t, p)
			bad[k] = v
			if _, e := shvLifecycleExpected("capture_import", bad); e == nil {
				t.Fatal("bad provenance accepted")
			}
		})
	}
	forged := shvClone(t, c)
	forged["observed_at"] = "2026-09-13T12:34:57Z"
	forged = shvReseal(t, forged)
	if ValidateSHVSourceResult("capture_import", shvRaw(t, p), shvRaw(t, forged)) == nil {
		t.Fatal("resealed observation substitution")
	}
	cmp := map[string]any{"source_root": p["source_root"], "previous": c, "current": forged}
	r, e := shvLifecycleExpected("capture_compare", cmp)
	if e != nil || !scvEqual(r["changes"], []any{"observation_time"}) {
		t.Fatal(e, r)
	}
	path := filepath.Join(shvText(p["source_root"]), "body.bin")
	if e = os.WriteFile(path, []byte{0, 255, 1, 3}, 0600); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVSourceResult("capture_import", shvRaw(t, p), shvRaw(t, c)) == nil {
		t.Fatal("changed actual capture bytes accepted")
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	if e = os.Symlink("elsewhere", path); e != nil {
		t.Fatal(e)
	}
	if _, e = shvLifecycleExpected("capture_import", p); e == nil {
		t.Fatal("symlink accepted")
	}
}
func TestSHVLifecycleGraphCorrespondence(t *testing.T) {
	_, _, p, c := lifecycleFixture(t)
	input := map[string]any{"source_root": p["source_root"], "captures": []any{c}}
	g, e := shvLifecycleExpected("graph_project", input)
	if e != nil {
		t.Fatal(e)
	}
	if len(shvList(g["nodes"])) != 2 || len(shvList(g["edges"])) != 1 {
		t.Fatal("missing source/capture provenance")
	}
	q := map[string]any{"source_root": p["source_root"], "graph": shvClone(t, g)}
	r, e := shvLifecycleExpected("graph_validate", q)
	if e != nil {
		t.Fatal(e)
	}
	bad := shvClone(t, r)
	bad["valid"] = false
	bad = shvReseal(t, bad)
	if ValidateSHVSourceResult("graph_validate", shvRaw(t, q), shvRaw(t, bad)) == nil {
		t.Fatal("resealed false validation result")
	}
	for _, field := range []string{"owner", "edges", "nodes"} {
		t.Run(field, func(t *testing.T) {
			bad := shvClone(t, g)
			switch field {
			case "owner":
				shvMap(bad["owner"])["engine_version"] = "0.2.0"
			case "edges":
				shvMap(shvList(bad["edges"])[0])["label"] = "hardware_supported"
			case "nodes":
				shvMap(shvList(bad["nodes"])[0])["properties"] = map[string]any{"truth": true}
			}
			bad = shvReseal(t, bad)
			if _, e := shvLifecycleExpected("graph_validate", map[string]any{"source_root": p["source_root"], "graph": bad}); e == nil {
				t.Fatal("resealed graph forgery")
			}
		})
	}
	if _, e = shvLifecycleExpected("graph_project", map[string]any{"source_root": p["source_root"], "captures": []any{c, c}}); e == nil {
		t.Fatal("duplicate capture accepted")
	}
}
func TestSHVLifecycleURIAndExactVersion(t *testing.T) {
	for _, s := range []string{"https://Example.com.:443/a?x=1", "http://a", "http://a:65535/a%20b"} {
		if !shvLifecycleURI(s) {
			t.Fatal("valid URI", s)
		}
	}
	for _, s := range []string{"HTTPS://a", "https://a:01", "https://a:65536", "https://[::1]", "https://u@a", "https://a%20b", "https://a/#x", "https://a/é", "https://a/\\x", "https://a/\"", "https://a/<x>"} {
		if shvLifecycleURI(s) {
			t.Fatal("bad URI", s)
		}
	}
	for _, v := range []string{"", "0.1.0", "0.2.0-dev"} {
		if _, e := InspectSHVSource(t.TempDir(), v); e == nil {
			t.Fatal("implicit version selection")
		}
	}
}
func TestSHVLifecycleInstalled(t *testing.T) {
	prefix := os.Getenv("SHV_SOURCE_TEST_PREFIX")
	if prefix == "" {
		t.Skip("explicit installed SHV source prefix required")
	}
	p, plan, capInput, _ := lifecycleFixture(t)
	run := func(op string, p map[string]any) map[string]any {
		t.Helper()
		r, e := InvokeSHVSource(context.Background(), prefix, SHVSourceVersion, shvText(capInput["source_root"]), op, shvRaw(t, p))
		if e != nil {
			t.Fatal(op, e)
		}
		out, e := shvObject(r.Result)
		if e != nil {
			t.Fatal(e)
		}
		return out
	}
	descriptor := run("inspect", map[string]any{})
	for _, field := range []string{"engine_version", "canonical_apply_enabled"} {
		bad := shvClone(t, descriptor)
		if field == "engine_version" {
			bad[field] = "0.2.0-dev"
		} else {
			bad[field] = true
		}
		delete(bad, "descriptor_digest")
		d, e := SCVDigest(bad)
		if e != nil {
			t.Fatal(e)
		}
		bad["descriptor_digest"] = d
		if ValidateSHVSourceResult("inspect", []byte(`{}`), shvRaw(t, bad)) == nil {
			t.Fatal("resealed descriptor drift accepted", field)
		}
	}
	got := run("source_plan", p)
	if !scvEqual(got, plan) {
		t.Fatal("native plan differs")
	}
	run("source_reduce", map[string]any{"current": nil, "plan": got})
	run("source_status", map[string]any{"history": []any{got["source"]}})
	c := run("capture_import", capInput)
	run("capture_compare", map[string]any{"source_root": capInput["source_root"], "previous": c, "current": c})
	g := run("graph_project", map[string]any{"source_root": capInput["source_root"], "captures": []any{c}})
	run("graph_validate", map[string]any{"source_root": capInput["source_root"], "graph": g})
	for _, template := range []bool{false, true} {
		selection := ""
		if template {
			selection = "capture_import"
		}
		raw, e := SHVSourceDiscovery(prefix, SHVSourceVersion, template, selection)
		if e != nil || !json.Valid(raw) {
			t.Fatal(e, string(raw))
		}
	}
}

func TestSHVLifecycleComparisonOrderAndLimits(t *testing.T) {
	_, _, p, a := lifecycleFixture(t)
	b := shvClone(t, a)
	s := shvMap(b["source"])
	d := shvMap(s["definition"])
	d["source_id"] = "different-family"
	l := shvMap(shvList(d["locators"])[0])
	l["id"] = "new"
	l["uri"] = "https://new.example/source"
	l["format"] = "html"
	b["source"] = shvReseal(t, s)
	b["locator_id"] = "new"
	b["resolved_uri"] = l["uri"]
	b["redirect_chain"] = []any{}
	b["observed_at"] = "2026-09-14T00:00:00Z"
	b["upstream_revision"] = map[string]any{"scheme": "publisher-revision", "value": "r2"}
	b["completeness"] = "complete"
	b["issues"] = []any{}
	m := shvMap(b["manifest"])
	body := []byte("<html>different representation</html>")
	m["id"] = "second"
	m["path"] = "second.html"
	m["bytes"] = len(body)
	m["digest"] = digestBytes(body)
	m["format"] = "html"
	if e := os.WriteFile(filepath.Join(shvText(p["source_root"]), "second.html"), body, 0600); e != nil {
		t.Fatal(e)
	}
	b = shvReseal(t, b)
	r, e := shvLifecycleExpected("capture_compare", map[string]any{"source_root": p["source_root"], "previous": a, "current": b})
	if e != nil {
		t.Fatal(e)
	}
	want := []any{"source_identity", "source_revision", "body", "representation", "acquisition_route", "upstream_revision", "completeness", "observation_time", "manifest_identity"}
	if !scvEqual(r["changes"], want) || r["same_logical_source"] != false {
		t.Fatal(r)
	}
	current := shvClone(t, shvMap(a["source"]))
	current["generation"] = json.Number("32")
	current["previous_digest"] = digestBytes([]byte("prior"))
	current = shvReseal(t, current)
	desired := shvClone(t, shvMap(current["definition"]))
	desired["publisher"] = "changed"
	if _, e = shvLifecycleExpected("source_plan", map[string]any{"operation_id": "exhausted", "reason": "change", "current": current, "desired": desired}); e == nil {
		t.Fatal("generation bound ignored")
	}
	desired = shvClone(t, shvMap(shvMap(a["source"])["definition"]))
	desired["locators"] = append(shvList(desired["locators"]), shvList(desired["locators"])[0])
	if shvLifecycleDefinition(desired) == nil {
		t.Fatal("duplicate locator accepted")
	}
}

func TestSHVLifecycleGraphReadBudgetPreflight(t *testing.T) {
	_, _, p, c := lifecycleFixture(t)
	if _, e := shvLifecycleExpected("graph_project", map[string]any{"source_root": "/bad\nroot", "captures": []any{}}); e == nil {
		t.Fatal("empty graph accepted control-bearing root")
	}
	for _, root := range []string{"//", shvText(p["source_root"]) + "/", shvText(p["source_root"]) + "/."} {
		if _, e := shvLifecycleExpected("graph_project", map[string]any{"source_root": root, "captures": []any{}}); e == nil {
			t.Fatal("nonnormalized root accepted", root)
		}
	}
	if _, e := shvLifecycleExpected("graph_project", map[string]any{"source_root": "/", "captures": []any{}}); e != nil {
		t.Fatal("filesystem root rejected", e)
	}
	// Five declared MiB must reject the aggregate before attempting nonexistent files.
	captures := []any{}
	for i := 0; i < 5; i++ {
		x := shvClone(t, c)
		m := shvMap(x["manifest"])
		m["bytes"] = json.Number("1048576")
		m["path"] = "missing"
		captures = append(captures, shvReseal(t, x))
	}
	_, e := shvLifecycleExpected("graph_project", map[string]any{"source_root": p["source_root"], "captures": captures})
	if e == nil || e.Error() != shvFail().Error() {
		t.Fatalf("aggregate byte budget was not rejected before file access: %v", e)
	}
}

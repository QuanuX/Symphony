package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func shvRaw(t *testing.T, v any) []byte {
	t.Helper()
	b, e := SCVCanonical(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func shvClone(t *testing.T, v map[string]any) map[string]any {
	t.Helper()
	m, e := shvObject(shvRaw(t, v))
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func shvReseal(t *testing.T, v map[string]any) map[string]any {
	t.Helper()
	delete(v, "digest")
	return shvClone(t, shvSealNew(v))
}
func shvFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw := `<html><div id="overview"><h1>Acme™ A</h1></div><script><dt>Modes</dt><dd>4P</dd></script><article id="specs"><dl><dt>Modes</dt><dd>1P / 2P</dd><dt>Cores</dt><dd>16</dd><dt>Launch Date</dt><dd>03/15/2021</dd><dt>End</dt><dd>yes</dd></dl></article></html>`
	if e := os.WriteFile(filepath.Join(root, "source.html"), []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
	source := map[string]any{"id": "acme", "path": "source.html", "bytes": len(raw), "digest": digestBytes([]byte(raw)), "format": "html"}
	fields := []any{map[string]any{"predicate": "socket_modes", "label": "Modes", "next_label": "Cores", "value_type": "tokens", "qualifier": "finite_supported_set"}, map[string]any{"predicate": "cores", "label": "Cores", "next_label": "Launch Date", "value_type": "integer", "qualifier": "declared"}, map[string]any{"predicate": "model_introduction", "label": "Launch Date", "next_label": "End", "value_type": "date", "qualifier": "declared"}}
	spec := map[string]any{"id": "a", "manufacturer": "Acme", "model": "Acme™ A", "hardware_class": "cpu", "source_id": "acme", "heading_section": "div#overview", "field_section": "article#specs", "fields": fields}
	subject := map[string]any{"id": "a", "manufacturer": "Acme", "model": "Acme™ A", "hardware_class": "cpu", "introduced": map[string]any{"from": "2021-03-15", "through": "2021-03-15"}, "assertions": []any{map[string]any{"predicate": "cores", "value": 16, "qualifier": "declared", "source_id": "acme"}, map[string]any{"predicate": "model_introduction", "value": "2021-03-15", "qualifier": "declared", "source_id": "acme"}, map[string]any{"predicate": "socket_modes", "value": []any{"1P", "2P"}, "qualifier": "finite_supported_set", "source_id": "acme"}}}
	input := shvClone(t, map[string]any{"source_root": root, "sources": []any{source}, "subjects": []any{spec}})
	catalogue := shvClone(t, shvSealNew(map[string]any{"protocol": "symphony.shv.catalogue.v1", "sources": []any{source}, "subjects": []any{subject}, "mapping": []any{spec}}))
	return input, catalogue
}
func TestSHVSourceConsumerRejectsResealedFalseValues(t *testing.T) {
	p, c := shvFixture(t)
	if e := ValidateSHVResult("catalogue_build", shvRaw(t, p), shvRaw(t, c), false); e != nil {
		t.Fatal(e)
	}
	bad := shvClone(t, c)
	shvMap(shvList(shvMap(shvList(bad["subjects"])[0])["assertions"])[0])["value"] = json.Number("32")
	bad = shvReseal(t, bad)
	if e := ValidateSHVResult("catalogue_build", shvRaw(t, p), shvRaw(t, bad), false); e == nil {
		t.Fatal("resealed false source-derived cores accepted")
	}
	query := map[string]any{"source_root": p["source_root"], "catalogue": bad, "subject_ids": []any{}}
	result, e := shvQuery(query)
	if e != nil {
		t.Fatal(e)
	}
	if e = ValidateSHVResult("catalogue_query", shvRaw(t, query), shvRaw(t, result), false); e == nil {
		t.Fatal("resealed forged retained catalogue accepted by consumer")
	}
	if e = os.WriteFile(filepath.Join(shvText(p["source_root"]), "source.html"), []byte("different bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = ValidateSHVResult("catalogue_build", shvRaw(t, p), shvRaw(t, c), false); e == nil {
		t.Fatal("source mutation accepted")
	}
}
func TestSHVCoverageIndependentBoundaryCases(t *testing.T) {
	profile := shvReseal(t, map[string]any{"protocol": "symphony.shv.coverage-profile.v1", "as_of": "2026-09-13", "selector": map[string]any{"op": "date", "basis": "model_introduction", "from": "2018-01-01", "through": "2026-09-13"}})
	summary := func(id string, interval any) any {
		return map[string]any{"id": id, "manufacturer": "custom", "model": id, "hardware_class": "custom", "introduced": interval}
	}
	subjects := []any{summary("older", map[string]any{"from": "2017-01-01", "through": "2017-12-31"}), summary("boundary", map[string]any{"from": "2018-01-01", "through": "2018-01-01"}), summary("straddle", map[string]any{"from": "2017-12-01", "through": "2018-02-01"}), summary("unknown", nil)}
	p := shvClone(t, map[string]any{"profile": profile, "subjects": subjects})
	r, e := shvCoverage(p)
	if e != nil {
		t.Fatal(e)
	}
	want := []any{map[string]any{"subject_id": "boundary", "status": "included"}, map[string]any{"subject_id": "older", "status": "excluded"}, map[string]any{"subject_id": "straddle", "status": "unresolved"}, map[string]any{"subject_id": "unknown", "status": "unresolved"}}
	if !scvEqual(r["decisions"], want) {
		t.Fatal("wrong independently expected date decisions")
	}
	if e = ValidateSHVResult("coverage_plan", shvRaw(t, p), shvRaw(t, r), false); e != nil {
		t.Fatal(e)
	}
	bad := shvClone(t, r)
	shvMap(shvList(bad["decisions"])[3])["status"] = "included"
	bad = shvReseal(t, bad)
	if ValidateSHVResult("coverage_plan", shvRaw(t, p), shvRaw(t, bad), false) == nil {
		t.Fatal("resealed unknown-to-included forgery accepted")
	}
	oldSelector := profile["selector"]
	profile["selector"] = map[string]any{"op": "or", "args": []any{oldSelector, map[string]any{"op": "ids", "values": []any{"older"}}}}
	profile = shvReseal(t, profile)
	p["profile"] = profile
	r, e = shvCoverage(p)
	if e != nil || shvMap(shvList(r["decisions"])[1])["status"] != "included" {
		t.Fatal("caller older inclusion failed", e)
	}
	profile["selector"] = map[string]any{"op": "date", "basis": "manufacture", "from": "2018-01-01", "through": "2026-09-13"}
	p["profile"] = shvReseal(t, profile)
	if _, e = shvCoverage(p); e == nil {
		t.Fatal("unsupported date basis replaced")
	}
}
func TestSHVEvaluationAndProjectionCorrespondence(t *testing.T) {
	p, c := shvFixture(t)
	reqs := []any{map[string]any{"id": "four", "predicate": "socket_modes", "operator": "contains", "value": "4P", "qualifier": "finite_supported_set"}, map[string]any{"id": "board", "predicate": "board_bios_os", "operator": "eq", "value": "yes", "qualifier": "declared"}}
	payload := shvClone(t, map[string]any{"source_root": p["source_root"], "catalogue": c, "subject_ids": []any{"missing", "a"}, "requirements": reqs})
	r, e := shvEvaluation(payload)
	if e != nil {
		t.Fatal(e)
	}
	findings := shvList(r["findings"])
	if shvMap(findings[0])["status"] != "contradicted" || shvMap(findings[1])["status"] != "unresolved" {
		t.Fatal("incorrect evidence expectations")
	}
	if e = ValidateSHVResult("evaluate", shvRaw(t, payload), shvRaw(t, r), false); e != nil {
		t.Fatal(e)
	}
	bad := shvClone(t, r)
	shvMap(shvList(bad["findings"])[0])["status"] = "supported"
	if ValidateSHVResult("evaluate", shvRaw(t, payload), shvRaw(t, shvReseal(t, bad)), false) == nil {
		t.Fatal("resealed false positive accepted")
	}
	g, e := shvProjection(c)
	if e != nil {
		t.Fatal(e)
	}
	gp := map[string]any{"source_root": p["source_root"], "catalogue": c}
	if e = ValidateSHVResult("graph_project", shvRaw(t, gp), shvRaw(t, g), false); e != nil {
		t.Fatal(e)
	}
	bad = shvClone(t, g)
	shvMap(shvList(bad["nodes"])[0])["labels"] = []any{"hardware_subject"}
	if ValidateSHVResult("graph_project", shvRaw(t, gp), shvRaw(t, shvReseal(t, bad)), false) == nil {
		t.Fatal("resealed source/subject substitution accepted")
	}
}
func TestSHVAdapterIndependentStructuralContract(t *testing.T) {
	artifact := shvReseal(t, map[string]any{"protocol": "example.custom.v1", "payload": "opaque"})
	g := shvReseal(t, map[string]any{"protocol": "symphony.graph.exchange.v1", "owner": map[string]any{"engine_id": "custom-owner", "engine_version": "exact", "artifact_protocol": artifact["protocol"], "artifact_digest": artifact["digest"]}, "owner_artifact": artifact, "nodes": []any{map[string]any{"id": "a", "labels": []any{}, "properties": map[string]any{"text": "é < > & \u2028"}}, map[string]any{"id": "b", "labels": []any{"custom"}, "properties": map[string]any{}}}, "edges": []any{}})
	p := map[string]any{"graph": g, "kind": "nodes", "ids": []any{"b", "absent", "a"}}
	r, e := shvAdapterResult("query", p)
	if e != nil {
		t.Fatal(e)
	}
	if shvMap(shvList(r["rows"])[0])["id"] != "a" {
		t.Fatal("adapter query must preserve graph row order")
	}
	if e = ValidateSHVResult("query", shvRaw(t, p), shvRaw(t, r), true); e != nil {
		t.Fatal(e)
	}
	bad := shvClone(t, r)
	bad["missing_ids"] = []any{}
	if ValidateSHVResult("query", shvRaw(t, p), shvRaw(t, shvReseal(t, bad)), true) == nil {
		t.Fatal("resealed missing row omission accepted")
	}
	broken := shvClone(t, g)
	broken["edges"] = []any{map[string]any{"id": "edge", "from": "a", "to": "absent", "label": "link", "properties": map[string]any{}}}
	if shvGraph(shvReseal(t, broken)) == nil {
		t.Fatal("dangling endpoint accepted")
	}
}
func TestSHVRawAndExactReceiptBoundary(t *testing.T) {
	if _, e := InvokeSHV(context.Background(), "/absent", SHVVersion, "/", "coverage_plan", []byte(`{"x":"\ud800"}`)); e == nil || strings.Contains(e.Error(), "receipt") {
		t.Fatal("raw Unicode must fail before receipt lookup", e)
	}
	prefix := t.TempDir()
	spec := shvEngineSpec
	spec.expectedFiles = func(v string) map[string]struct{} {
		return map[string]struct{}{filepath.ToSlash(filepath.Join("libexec", "symphony", spec.moduleID, v, spec.engineID)): {}, filepath.ToSlash(filepath.Join("share", "symphony", "contracts", spec.moduleID, v, "SPEC.md")): {}}
	}
	_, receipt := createInstalledV2Fixture(t, spec, prefix, SHVVersion)
	inst, e := InspectSHV(prefix, SHVVersion)
	if e != nil || inst.ReceiptDigest != receipt.ReceiptDigest {
		t.Fatal(e)
	}
	for _, v := range []string{"", "latest", "0.2.0-dev"} {
		if _, e = InspectSHV(prefix, v); e == nil {
			t.Fatal("implicit/upgraded version accepted")
		}
	}
	if e = os.WriteFile(filepath.Join(prefix, "share/symphony/contracts/shv-engine", SHVVersion, "SPEC.md"), []byte("changed"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = InspectSHV(prefix, SHVVersion); e == nil {
		t.Fatal("receipt-owned nonexecutable change accepted")
	}
}
func TestSHVInstalledNativeOperations(t *testing.T) {
	prefix := os.Getenv("SHV_TEST_PREFIX")
	adapter := os.Getenv("SHV_ADAPTER_TEST_PREFIX")
	if prefix == "" || adapter == "" {
		t.Skip("requires explicit installed SHV and adapter prefixes")
	}
	p, c := shvFixture(t)
	cwd, _ := os.Getwd()
	invoke := func(op string, payload any) map[string]any {
		t.Helper()
		r, e := InvokeSHV(context.Background(), prefix, SHVVersion, cwd, op, shvRaw(t, payload))
		if e != nil {
			t.Fatalf("%s: %v", op, e)
		}
		m, e := shvObject(r.Result)
		if e != nil {
			t.Fatal(e)
		}
		return m
	}
	invoke("inspect", map[string]any{})
	profile := invoke("coverage_default", map[string]any{"as_of": "2026-09-13"})
	invoke("coverage_plan", map[string]any{"profile": profile, "subjects": []any{map[string]any{"id": "old", "manufacturer": "Custom", "model": "old", "hardware_class": "cpu", "introduced": nil}}})
	actual := invoke("catalogue_build", p)
	if !scvEqual(actual, c) {
		t.Fatal("native build differs from independent Go fixture")
	}
	invoke("catalogue_query", map[string]any{"source_root": p["source_root"], "catalogue": c, "subject_ids": []any{"a", "missing"}})
	invoke("evaluate", map[string]any{"source_root": p["source_root"], "catalogue": c, "subject_ids": []any{}, "requirements": []any{map[string]any{"id": "cores", "predicate": "cores", "operator": "gte", "value": 16, "qualifier": "declared"}}})
	g := invoke("graph_project", map[string]any{"source_root": p["source_root"], "catalogue": c})
	invoke("graph_validate", map[string]any{"source_root": p["source_root"], "graph": g})
	for _, op := range []string{"inspect", "roundtrip", "query"} {
		payload := map[string]any{}
		if op != "inspect" {
			payload["graph"] = g
		}
		if op == "query" {
			payload["kind"] = "nodes"
			payload["ids"] = []any{"subject:a", "absent", "source:acme"}
		}
		if _, e := InvokeSHVGraphAdapter(context.Background(), adapter, SHVVersion, cwd, op, shvRaw(t, payload)); e != nil {
			t.Fatalf("adapter %s: %v", op, e)
		}
	}
	for _, a := range []bool{false, true} {
		path := prefix
		if a {
			path = adapter
		}
		if _, e := SHVDiscovery(path, SHVVersion, a, false, ""); e != nil {
			t.Fatal(e)
		}
		if _, e := SHVDiscovery(path, SHVVersion, a, true, "inspect"); e != nil {
			t.Fatal(e)
		}
	}
}

func TestSHVHTMLSectionAndTagBoundaries(t *testing.T) {
	good := `<nav><h1>Not the product</h1><dt>Modes</dt><dd>8P</dd></nav><div id="heading"><h1>Product</h1></div><article id="fields"><dl><dt>Modes</dt><dd>1P / 2P</dd><dt>End</dt><dd>yes</dd></dl></article>`
	pairs, e := shvHTMLPairs(good, "Product", "div#heading", "article#fields")
	if e != nil || len(pairs) != 2 || pairs[0][1] != "1P / 2P" {
		t.Fatal("explicit sections must ignore navigation", e)
	}
	cases := map[string]string{
		"fake-heading-tag":       strings.ReplaceAll(strings.ReplaceAll(good, "<h1>Product</h1>", "<h1-fake>Product</h1-fake>"), "Not the product", "Product"),
		"duplicate-section":      good + `<div id="heading"><h1>Product</h1></div>`,
		"nested-sections":        `<article id="fields"><div id="heading"><h1>Product</h1></div><dt>Modes</dt><dd>1P</dd></article>`,
		"duplicate-id-attribute": strings.Replace(good, `id="heading"`, `id="heading" id="other"`, 1),
		"script-prefix-close":    `<script></scriptx>` + good,
		"self-closing-section":   strings.Replace(good, `<div id="heading">`, `<div id="heading"/>`, 1),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, e := shvHTMLPairs(raw, "Product", "div#heading", "article#fields"); e == nil {
				t.Fatal("ambiguous HTML admitted")
			}
		})
	}
	nested := `<div id="heading"><h1>Product</h1><div id="fields"><article id="fields"><h1>Ignored field title</h1><dt>Modes</dt><dd>1P</dd><dt>End</dt><dd>yes</dd></article></div></div>`
	if pairs, e := shvHTMLPairs(nested, "Product", "div#heading", "article#fields"); e != nil || len(pairs) != 2 {
		t.Fatal("explicit field section nested within product heading ancestor should work", e)
	}
	raw := `<script></scriptx><h1>False heading</h1><dt>Modes</dt><dd>4P</dd></script>` + good
	if _, e = shvHTMLPairs(raw, "Product", "div#heading", "article#fields"); e != nil {
		t.Fatal("script close prefix must not terminate ignored block", e)
	}
}

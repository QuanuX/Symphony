package knowledgeengine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func shvTableFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	body := `<div id="overview"><h1>Family A</h1><section id="summary"><table><tr><th>Launch</th><td>Q1'23</td></tr><tr><th>Modes</th><td>2P / 1P</td></tr><tr><th>End</th><td>yes</td></tr></table></section><section id="memory"><table><tr><th>Mode</th><th>Rate</th><th>Capacity</th></tr><tr><td>1 DPC</td><td>4800</td><td>4 TB</td></tr><tr><td>2 DPC</td><td>4400</td><td>8 TB</td></tr></table></section></div>`
	if e = os.WriteFile(filepath.Join(root, "source.html"), []byte(body), 0600); e != nil {
		t.Fatal(e)
	}
	source := map[string]any{"id": "source", "path": "source.html", "bytes": len(body), "digest": digestBytes([]byte(body)), "format": "html"}
	columns := []any{"Mode", "Rate", "Capacity"}
	matrix := map[string]any{"columns": columns, "rows": []any{[]any{"1 DPC", "4800", "4 TB"}, []any{"2 DPC", "4400", "8 TB"}}}
	mapping := map[string]any{"id": "family-a", "manufacturer": "Acme", "model": "Family A", "hardware_class": "cpu_family", "source_id": "source", "heading_section": "div#overview", "interpretation_profile": "scoped_tables.v1", "fields": []any{map[string]any{"predicate": "model_introduction", "section": "section#summary", "label": "Launch", "next_label": "Modes", "value_type": "quarter_20yy", "qualifier": "declared"}, map[string]any{"predicate": "socket_modes", "section": "section#summary", "label": "Modes", "next_label": "End", "value_type": "tokens", "qualifier": "finite_supported_set"}, map[string]any{"predicate": "memory_modes", "section": "section#memory", "columns": columns, "value_type": "table_rows", "qualifier": "declared"}}}
	quarter := map[string]any{"precision": "quarter", "source_text": "Q1'23", "from": "2023-01-01", "through": "2023-03-31"}
	subject := map[string]any{"id": "family-a", "manufacturer": "Acme", "model": "Family A", "hardware_class": "cpu_family", "introduced": map[string]any{"from": "2023-01-01", "through": "2023-03-31"}, "assertions": []any{map[string]any{"predicate": "memory_modes", "value": matrix, "qualifier": "declared", "source_id": "source"}, map[string]any{"predicate": "model_introduction", "value": quarter, "qualifier": "declared", "source_id": "source"}, map[string]any{"predicate": "socket_modes", "value": []any{"1P", "2P"}, "qualifier": "finite_supported_set", "source_id": "source"}}}
	p := shvClone(t, map[string]any{"source_root": root, "sources": []any{source}, "subjects": []any{mapping}})
	c := shvReseal(t, map[string]any{"protocol": "symphony.shv.catalogue.v1", "sources": []any{source}, "mapping": []any{mapping}, "subjects": []any{subject}})
	return p, c
}
func TestSHVTablesVersionAndLegacy(t *testing.T) {
	p, c := shvTableFixture(t)
	if e := ValidateSHVResultVersion("catalogue_build", shvRaw(t, p), shvRaw(t, c), false, SHVTableVersion); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVResult("catalogue_build", shvRaw(t, p), shvRaw(t, c), false) == nil {
		t.Fatal("old wrapper accepted a tagged catalogue")
	}
	old, legacy := shvFixture(t)
	for _, version := range []string{SHVVersion, SHVTableVersion} {
		if e := ValidateSHVResultVersion("catalogue_build", shvRaw(t, old), shvRaw(t, legacy), false, version); e != nil {
			t.Fatal(version, e)
		}
	}
	if _, e := InspectSHVGraphAdapter(t.TempDir(), SHVTableVersion); e == nil {
		t.Fatal("kernel release leaked into adapter version")
	}
	if _, e := InspectSHV(t.TempDir(), "0.3.0-dev"); e == nil {
		t.Fatal("unknown release accepted")
	}
}
func TestSHVTablesReplayForgeries(t *testing.T) {
	p, c := shvTableFixture(t)
	for _, name := range []string{"mode_mix", "quarter_date", "quarter_text", "columns", "row_drop"} {
		t.Run(name, func(t *testing.T) {
			bad := shvClone(t, c)
			a := shvList(shvMap(shvList(bad["subjects"])[0])["assertions"])
			matrix := shvMap(shvMap(a[0])["value"])
			quarter := shvMap(shvMap(a[1])["value"])
			switch name {
			case "mode_mix":
				shvList(shvList(matrix["rows"])[0])[2] = "8 TB"
			case "quarter_date":
				quarter["through"] = "2023-03-30"
			case "quarter_text":
				quarter["source_text"] = "Q1'22"
			case "columns":
				matrix["columns"] = []any{"Mode", "Capacity", "Rate"}
			case "row_drop":
				matrix["rows"] = shvList(matrix["rows"])[:1]
			}
			bad = shvReseal(t, bad)
			if ValidateSHVResultVersion("catalogue_build", shvRaw(t, p), shvRaw(t, bad), false, SHVTableVersion) == nil {
				t.Fatal("resealed interpreted forgery accepted")
			}
		})
	}
	if e := os.WriteFile(filepath.Join(shvText(p["source_root"]), "source.html"), []byte("changed"), 0600); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVResultVersion("catalogue_build", shvRaw(t, p), shvRaw(t, c), false, SHVTableVersion) == nil {
		t.Fatal("changed capture accepted")
	}
}
func TestSHVTablesParserBoundaries(t *testing.T) {
	wrap := func(body string) string { return `<div id="h"><h1>A</h1><section id="s">` + body + `</section></div>` }
	good := `<table><tr><th>Mode</th><th>Value</th></tr><tr><td>A</td><td>1 <b>&amp;</b> 2</td></tr><tr><td>B</td><td></td></tr></table>`
	rows, e := shvHTMLTable(wrap(good), "A", "div#h", "section#s")
	if e != nil {
		t.Fatal(e)
	}
	field := map[string]any{"value_type": "table_rows", "columns": []any{"Mode", "Value"}}
	v, e := shvTableValue(rows, field)
	if e != nil || !scvEqual(shvMap(v)["rows"], []any{[]any{"A", "1 & 2"}, []any{"B", ""}}) {
		t.Fatal(e, v)
	}
	cases := map[string]string{"nested_table": strings.Replace(good, "<td>A</td>", "<td><table></table></td>", 1), "nested_cell": strings.Replace(good, "<td>A</td>", "<td><td>A</td></td>", 1), "nested_row": strings.Replace(good, "<td>A</td>", "<tr><td>A</td></tr>", 1), "unpaired": strings.Replace(good, "</td>", "</th>", 1), "rowspan": strings.Replace(good, "<td>A", "<td RoWsPaN>A", 1), "colspan": strings.Replace(good, "<td>A", "<td colspan=\"1\">A", 1), "second_table": good + good, "empty_table": "<table></table>", "too_many_rows": "<table>" + strings.Repeat("<tr><td>x</td></tr>", 257) + "</table>", "too_many_columns": "<table><tr>" + strings.Repeat("<td>x</td>", 17) + "</tr></table>", "raw_buffer_bound": strings.Replace(good, "<td>A</td>", "<td>"+strings.Repeat(" ", 65537)+"</td>", 1), "cell_bound": strings.Replace(good, "<td>A</td>", "<td>"+strings.Repeat("x", 4097)+"</td>", 1)}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, e := shvHTMLTable(wrap(body), "A", "div#h", "section#s"); e == nil {
				t.Fatal("malformed selected table accepted")
			}
		})
	}
	for _, body := range []string{strings.Replace(good, "<td>B</td>", "<td>A</td>", 1), strings.Replace(good, "<td>B</td>", "<td></td>", 1), strings.Replace(good, "<td>B</td>", "<th>B</th>", 1)} {
		rows, e := shvHTMLTable(wrap(body), "A", "div#h", "section#s")
		if e != nil {
			t.Fatal(e)
		}
		if _, e = shvTableValue(rows, field); e == nil {
			t.Fatal("bad row identity or data cell accepted")
		}
	}
	// Attribute-like text inside another attribute never acts as a spanning cell.
	body := strings.Replace(good, "<td>A", "<td title=\"colspan=2\">A", 1)
	if _, e := shvHTMLTable(wrap(body), "A", "div#h", "section#s"); e != nil {
		t.Fatal(e)
	}
}
func TestSHVTablesQuarterAndCoverage(t *testing.T) {
	for text, want := range map[string]string{"Q1'00": "2000-03-31", "Q2'24": "2024-06-30", "Q3'99": "2099-09-30", "Q4'99": "2099-12-31"} {
		q, e := shvQuarter(text)
		if e != nil || q["through"] != want {
			t.Fatal(text, e, q)
		}
	}
	for _, s := range []string{"Q0'23", "Q5'23", "Q1’23", "Q1'2023", "Q1'3", "Q1'23 ", "q1'23"} {
		if _, e := shvQuarter(s); e == nil {
			t.Fatal("ambiguous quarter accepted", s)
		}
	}
	_, c := shvTableFixture(t)
	s := shvClone(t, shvMap(shvList(c["subjects"])[0]))
	delete(s, "assertions")
	profile := shvReseal(t, map[string]any{"protocol": "symphony.shv.coverage-profile.v1", "as_of": "2026-09-13", "selector": map[string]any{"op": "date", "basis": "model_introduction", "from": "2023-02-01", "through": "2026-09-13"}})
	r, e := shvCoverage(map[string]any{"profile": profile, "subjects": []any{s}})
	if e != nil || shvMap(shvList(r["decisions"])[0])["status"] != "unresolved" {
		t.Fatal(e, r)
	}
}
func TestSHVTablesQueryEvaluationGraph(t *testing.T) {
	p, c := shvTableFixture(t)
	q := map[string]any{"source_root": p["source_root"], "catalogue": c, "subject_ids": []any{}}
	r, e := shvQueryVersion(q, SHVTableVersion)
	if e != nil {
		t.Fatal(e)
	}
	if e = ValidateSHVResultVersion("catalogue_query", shvRaw(t, q), shvRaw(t, r), false, SHVTableVersion); e != nil {
		t.Fatal(e)
	}
	req := map[string]any{"id": "r", "predicate": "memory_modes", "operator": "eq", "value": "4800", "qualifier": "declared"}
	ev := shvClone(t, q)
	ev["requirements"] = []any{req}
	if _, e = shvEvaluationVersion(ev, SHVTableVersion); e == nil {
		t.Fatal("structured table treated as scalar contradiction")
	}
	req["predicate"] = "model_introduction"
	if _, e = shvEvaluationVersion(ev, SHVTableVersion); e == nil {
		t.Fatal("quarter treated as scalar")
	}
	graph, e := shvProjectionVersion(c, SHVTableVersion)
	if e != nil {
		t.Fatal(e)
	}
	gp := map[string]any{"source_root": p["source_root"], "catalogue": c}
	if e = ValidateSHVResultVersion("graph_project", shvRaw(t, gp), shvRaw(t, graph), false, SHVTableVersion); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVResultVersion("graph_project", shvRaw(t, gp), shvRaw(t, graph), false, SHVVersion) == nil {
		t.Fatal("old kernel accepted new graph")
	}
	ar, e := shvAdapterResult("roundtrip", map[string]any{"graph": shvClone(t, graph)})
	if e != nil {
		t.Fatal(e)
	}
	if e = ValidateSHVResultVersion("roundtrip", shvRaw(t, map[string]any{"graph": graph}), shvRaw(t, ar), true, SHVVersion); e != nil {
		t.Fatal(e)
	}
	bad := shvClone(t, graph)
	shvMap(bad["owner"])["engine_version"] = SHVVersion
	bad = shvReseal(t, bad)
	validation := shvReseal(t, map[string]any{"protocol": "symphony.shv.graph-validation.v1", "graph_digest": bad["digest"], "catalogue_digest": c["digest"], "valid": true})
	if ValidateSHVResultVersion("graph_validate", shvRaw(t, map[string]any{"source_root": p["source_root"], "graph": bad}), shvRaw(t, validation), false, SHVTableVersion) == nil {
		t.Fatal("wrong owner version accepted")
	}
}
func TestSHVTablesInstalled(t *testing.T) {
	prefix := os.Getenv("SHV_TABLE_TEST_PREFIX")
	if prefix == "" {
		t.Skip("explicit installed 0.2.0-dev kernel required")
	}
	p, c := shvTableFixture(t)
	run := func(op string, p map[string]any) map[string]any {
		t.Helper()
		r, e := InvokeSHV(context.Background(), prefix, SHVTableVersion, shvText(p["source_root"]), op, shvRaw(t, p))
		if e != nil {
			t.Fatal(op, e)
		}
		out, e := shvObject(r.Result)
		if e != nil {
			t.Fatal(e)
		}
		return out
	}
	// inspect has no caller working-directory payload, so invoke from a real temporary cwd.
	if _, e := InvokeSHV(context.Background(), prefix, SHVTableVersion, shvText(p["source_root"]), "inspect", []byte(`{}`)); e != nil {
		t.Fatal(e)
	}
	profileResponse, e := InvokeSHV(context.Background(), prefix, SHVTableVersion, shvText(p["source_root"]), "coverage_default", shvRaw(t, map[string]any{"as_of": "2026-09-13"}))
	if e != nil {
		t.Fatal(e)
	}
	profile, e := shvObject(profileResponse.Result)
	if e != nil {
		t.Fatal(e)
	}
	summary := shvClone(t, shvMap(shvList(c["subjects"])[0]))
	delete(summary, "assertions")
	if _, e = InvokeSHV(context.Background(), prefix, SHVTableVersion, shvText(p["source_root"]), "coverage_plan", shvRaw(t, map[string]any{"profile": profile, "subjects": []any{summary}})); e != nil {
		t.Fatal(e)
	}
	got := run("catalogue_build", p)
	if !scvEqual(got, c) {
		t.Fatal("native interpreted catalogue differs")
	}
	run("catalogue_query", map[string]any{"source_root": p["source_root"], "catalogue": c, "subject_ids": []any{}})
	run("evaluate", map[string]any{"source_root": p["source_root"], "catalogue": c, "subject_ids": []any{}, "requirements": []any{map[string]any{"id": "mode", "predicate": "socket_modes", "operator": "contains", "value": "2P", "qualifier": "finite_supported_set"}}})
	if _, e := InvokeSHV(context.Background(), prefix, SHVTableVersion, shvText(p["source_root"]), "evaluate", shvRaw(t, map[string]any{"source_root": p["source_root"], "catalogue": c, "subject_ids": []any{}, "requirements": []any{map[string]any{"id": "unsupported", "predicate": "memory_modes", "operator": "eq", "value": "4800", "qualifier": "declared"}}})); e == nil {
		t.Fatal("native compared structured table as scalar")
	}
	g := run("graph_project", map[string]any{"source_root": p["source_root"], "catalogue": c})
	run("graph_validate", map[string]any{"source_root": p["source_root"], "graph": g})
	old, legacy := shvFixture(t)
	if got := run("catalogue_build", old); !scvEqual(got, legacy) {
		t.Fatal("legacy native interpretation changed")
	}
	for _, templates := range []bool{false, true} {
		selection := ""
		if templates {
			selection = "catalogue_build"
		}
		if _, e := SHVDiscovery(prefix, SHVTableVersion, false, templates, selection); e != nil {
			t.Fatal(e)
		}
	}
}

func TestSHVTablesFinalScalarAndProfileBounds(t *testing.T) {
	rows := []shvTableRow{{{"th", "First"}, {"td", "1"}}, {{"th", "Lanes"}, {"td", "80"}}}
	f := map[string]any{"value_type": "integer", "label": "Lanes", "next_label": nil}
	v, e := shvTableValue(rows, f)
	if e != nil || !scvEqual(v, 80) {
		t.Fatal(e, v)
	}
	f["label"] = "First"
	if _, e = shvTableValue(rows, f); e == nil {
		t.Fatal("null next label accepted nonfinal row")
	}
	scalarBad := [][]shvTableRow{
		{{{"td", "Lanes"}, {"td", "80"}}},
		{{{"th", "Lanes"}, {"td", "80"}}, {{"th", "Lanes"}, {"td", "81"}}},
	}
	for _, bad := range scalarBad {
		f["label"] = "Lanes"
		if _, e := shvTableValue(bad, f); e == nil {
			t.Fatal("scalar table shape/duplicate accepted")
		}
	}
	f["label"] = "Lanes"
	f["next_label"] = "Invented"
	if _, e = shvTableValue(rows, f); e == nil {
		t.Fatal("invented sentinel accepted")
	}
	p, c := shvTableFixture(t)
	m := shvMap(shvList(c["mapping"])[0])
	field := shvMap(shvList(m["fields"])[0])
	field["next_label"] = nil
	shvList(shvMap(shvList(p["subjects"])[0])["fields"])[0] = field
	// The first row cannot claim final position, even with a resealed catalogue.
	c = shvReseal(t, c)
	if ValidateSHVResultVersion("catalogue_build", shvRaw(t, p), shvRaw(t, c), false, SHVTableVersion) == nil {
		t.Fatal("resealed last-row claim accepted")
	}
	old, legacy := shvFixture(t)
	for _, v := range shvList(shvMap(shvList(legacy["mapping"])[0])["fields"]) {
		f := shvMap(v)
		if f["predicate"] == "model_introduction" {
			f["value_type"] = "string"
		}
	}
	for _, v := range shvList(shvMap(shvList(legacy["subjects"])[0])["assertions"]) {
		a := shvMap(v)
		if a["predicate"] == "model_introduction" {
			a["value"] = "03/15/2021"
		}
	}
	shvMap(shvList(legacy["subjects"])[0])["introduced"] = nil
	legacy = shvReseal(t, legacy)
	old["subjects"] = legacy["mapping"]
	if ValidateSHVResult("catalogue_build", shvRaw(t, old), shvRaw(t, legacy), false) == nil {
		t.Fatal("legacy non-date model introduction accepted")
	}
}

func TestSHVTablesQualificationAndGroupIntegrity(t *testing.T) {
	wrap := func(s string) string { return `<div id="h"><h1>A</h1><section id="s">` + s + `</section></div>` }
	cell := `<tr><th>Key</th><td>value</td></tr>`
	good := `<table><thead>` + cell + `</thead><tbody>` + cell + `</tbody><tfoot>` + cell + `</tfoot></table>`
	if _, e := shvHTMLTable(wrap(good), "A", "div#h", "section#s"); e != nil {
		t.Fatal(e)
	}
	cases := map[string]string{
		"discarded_qualification": `<table>only in reduced mode` + cell + `</table>`,
		"qualification_element":   `<table><p>only in reduced mode</p>` + cell + `</table>`,
		"unclosed_group":          `<table><tbody>` + cell + `</table>`,
		"nested_groups":           `<table><tbody><thead>` + cell + `</thead></tbody></table>`,
		"mismatched_group":        `<table><tbody>` + cell + `</thead></table>`,
		"closing_attributes":      `<table>` + strings.Replace(cell, `</td>`, `</td hidden>`, 1) + `</table>`,
		"closing_slash":           `<table>` + strings.Replace(cell, `</td>`, `</td/>`, 1) + `</table>`,
		"buffer_separators":       `<table><tr><td>` + strings.Repeat(`<b>`, 65537) + `</td></tr></table>`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, e := shvHTMLTable(wrap(body), "A", "div#h", "section#s"); e == nil {
				t.Fatal("qualification or structure discarded")
			}
		})
	}
	ignored := `<table><!-- ignored --><script>untrusted instructions</script><style>body{}</style>` + cell + `</table>`
	if _, e := shvHTMLTable(wrap(ignored), "A", "div#h", "section#s"); e != nil {
		t.Fatal(e)
	}
}

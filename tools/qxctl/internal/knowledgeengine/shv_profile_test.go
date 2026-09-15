package knowledgeengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func profileFixture(t *testing.T, class string) map[string]any {
	t.Helper()
	p, e := shvProfileCompile(shvClone(t, map[string]any{"id": "caller", "revision": "r1", "hardware_class": class, "metrics": []any{map[string]any{"predicate": "cores", "value_type": "integer", "qualifier": "declared", "required": true, "description": "Caller metric", "extensions": map[string]any{}}}, "extensions": map[string]any{}}))
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestSHVProfileDeclarationEvidenceBoundary(t *testing.T) {
	p, _ := shvFixture(t)
	profile := profileFixture(t, "cpu")
	input := map[string]any{"profile": profile, "mapping": p["subjects"]}
	got, e := shvProfileDiagnose(input)
	if e != nil {
		t.Fatal(e)
	}
	row := shvMap(shvList(got["subjects"])[0])
	if row["status"] != "conformant" || got["evidence_scope"] != "mapping_declarations_only" {
		t.Fatal(got)
	}
	if !scvEqual(row["extension_predicates"], []string{"model_introduction", "socket_modes"}) {
		t.Fatal(row)
	}
	// No coercion: even a correctly typed value with a different unit is incomplete.
	changed := shvClone(t, input)
	m := shvMap(shvList(changed["mapping"])[0])
	shvMap(shvList(m["fields"])[1])["qualifier"] = "unit=other"
	x, e := shvProfileDiagnose(changed)
	if e != nil || shvMap(shvList(x["subjects"])[0])["status"] != "incomplete" {
		t.Fatal(x, e)
	}
	// Resealing a fabricated complete result must not conceal the mismatch.
	forged := shvClone(t, got)
	forged["input"] = changed
	forged = shvReseal(t, forged)
	if ValidateSHVProfileResult("mapping_diagnose", shvRaw(t, changed), shvRaw(t, forged)) == nil {
		t.Fatal("fabricated conformance accepted")
	}
	// A custom class is selectable; wrong classes are not silently assigned a profile.
	custom := map[string]any{"profile": profileFixture(t, "custom"), "mapping": p["subjects"]}
	x, e = shvProfileDiagnose(custom)
	if e != nil || shvMap(shvList(x["subjects"])[0])["status"] != "not_applicable" {
		t.Fatal(x, e)
	}
}
func TestSHVProfileUniverseSourceCorrespondence(t *testing.T) {
	p, c := shvTableFixture(t)
	definition := shvClone(t, map[string]any{"id": "selected", "revision": "r1", "kernel_version": SHVDocumentVersion, "coverage": shvSealNew(map[string]any{"protocol": "symphony.shv.coverage-profile.v1", "as_of": "2026-09-14", "selector": map[string]any{"op": "class", "values": []any{"unselected"}}}), "profiles": []any{}, "sources": p["sources"], "mapping": p["subjects"], "locators": []any{}, "extensions": map[string]any{}})
	u, e := shvUniverseBuild(definition)
	if e != nil {
		t.Fatal(e)
	}
	input := map[string]any{"universe": u, "bindings": map[string]any{"source_root": p["source_root"], "decoders": map[string]any{}}}
	got, e := shvUniverseBind(input, map[string]any{"catalogue": c})
	if e != nil {
		t.Fatal(e)
	}
	if !scvEqual(shvMap(got["coverage"])["counts"], map[string]int{"included": 0, "excluded": 1, "unresolved": 0}) || len(shvList(shvMap(got["catalogue"])["subjects"])) != 1 {
		t.Fatal("coverage unexpectedly erased the catalogue", got)
	}
	if e = ValidateSHVProfileResult("universe_bind", shvRaw(t, input), shvRaw(t, got)); e != nil {
		t.Fatal(e)
	}
	for name, edit := range map[string]func(map[string]any){
		"hidden_filter":       func(r map[string]any) { r["coverage"] = shvSealNew(map[string]any{}) },
		"engine_substitution": func(r map[string]any) { shvMap(r["reader"])["version"] = "0.2.0-dev" },
		"claimed_authority":   func(r map[string]any) { r["canonical_apply_enabled"] = true },
		"false_assertion": func(r map[string]any) {
			cat := shvMap(r["catalogue"])
			shvMap(shvList(shvMap(shvList(cat["subjects"])[0])["assertions"])[0])["qualifier"] = "forged"
			r["catalogue"] = shvReseal(t, cat)
		},
	} {
		t.Run(name, func(t *testing.T) {
			bad := shvClone(t, got)
			edit(bad)
			bad = shvReseal(t, bad)
			if ValidateSHVProfileResult("universe_bind", shvRaw(t, input), shvRaw(t, bad)) == nil {
				t.Fatal("resealed false result accepted")
			}
		})
	}
	if e = os.WriteFile(filepath.Join(shvText(p["source_root"]), "source.html"), []byte("changed"), 0600); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVProfileResult("universe_bind", shvRaw(t, input), shvRaw(t, got)) == nil {
		t.Fatal("changed evidence accepted")
	}
}
func TestSHVProfileInputStrictness(t *testing.T) {
	p := profileFixture(t, "cpu")
	definition := shvMap(p["definition"])
	for name, edit := range map[string]func(map[string]any){"unknown_control": func(d map[string]any) { d["extra"] = true }, "duplicate_metric": func(d map[string]any) { d["metrics"] = append(shvList(d["metrics"]), shvList(d["metrics"])[0]) }, "invented_type": func(d map[string]any) { shvMap(shvList(d["metrics"])[0])["value_type"] = "float" }, "wrong_required": func(d map[string]any) { shvMap(shvList(d["metrics"])[0])["required"] = "yes" }} {
		t.Run(name, func(t *testing.T) {
			d := shvClone(t, definition)
			edit(d)
			if _, e := shvProfileCompile(d); e == nil {
				t.Fatal("ambiguous profile accepted")
			}
		})
	}
	d := shvClone(t, definition)
	d["metrics"] = []any{}
	d["revision"] = "retired"
	if _, e := shvProfileCompile(d); e != nil {
		t.Fatal(e)
	}
	if e := shvProfileCheck(p); e != nil {
		t.Fatal("retiring caller fields changed history", e)
	}
	if _, e := InspectSHVProfile(t.TempDir(), "latest"); e == nil {
		t.Fatal("implicit upgrade accepted")
	}
}
func TestSHVProfileNativeCorrespondence(t *testing.T) {
	dir := os.Getenv("SHV_PROFILE_CASES")
	if dir == "" {
		t.Skip("set SHV_PROFILE_CASES to retained native/installed acceptance cases")
	}
	paths, e := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9].json"))
	if e != nil || len(paths) == 0 {
		t.Fatal(paths, e)
	}
	checked := 0
	for _, path := range paths {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		c, e := shvObject(raw)
		if e != nil {
			t.Fatal(e)
		}
		if c["good"] != true {
			continue
		}
		t.Run(filepath.Base(path), func(t *testing.T) {
			op := shvText(c["operation"])
			input, result := shvRaw(t, c["input"]), shvRaw(t, c["result"])
			if e := ValidateSHVProfileResult(op, input, result); e != nil {
				t.Fatal(e)
			}
			if op == "inspect" {
				return
			}
			bad := shvClone(t, shvMap(c["result"]))
			bad["invented_control"] = json.Number("1")
			bad = shvReseal(t, bad)
			if ValidateSHVProfileResult(op, input, shvRaw(t, bad)) == nil {
				t.Fatal("resealed extension to closed result accepted")
			}
		})
		checked++
	}
	if checked < 30 {
		t.Fatal("incomplete native correspondence campaign", checked)
	}
}

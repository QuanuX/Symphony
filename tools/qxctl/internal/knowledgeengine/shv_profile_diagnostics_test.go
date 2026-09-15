package knowledgeengine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSHVProfileReferencePaths(t *testing.T) {
	source := map[string]any{"id": "source", "kind": "capture", "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "document": nil}
	snapshot := shvReseal(t, map[string]any{"source_digest": source["digest"]})
	head := shvReseal(t, map[string]any{"snapshots": []any{snapshot["digest"], snapshot["digest"]}})
	input := shvClone(t, map[string]any{"objects": []any{source, map[string]any{"id": "snapshot", "kind": "graph_snapshot", "digest": snapshot["digest"], "document": snapshot}, map[string]any{"id": "history", "kind": "catalogue_history", "digest": head["digest"], "document": head}}, "edges": []any{map[string]any{"from": "history", "to": "snapshot", "pointer": "/snapshots/0"}, map[string]any{"from": "history", "to": "snapshot", "pointer": "/snapshots/1"}, map[string]any{"from": "snapshot", "to": "source", "pointer": "/source_digest"}}, "root_ids": []any{"history"}, "candidate_ids": []any{"source", "snapshot"}})
	r, e := shvReferencesAnalyze(input)
	if e != nil {
		t.Fatal(e)
	}
	if !scvEqual(shvMap(shvList(r["candidates"])[0])["path"], []any{"history", "snapshot", "source"}) || r["deletion_authorized"] != false {
		t.Fatal(r)
	}
	if len(shvList(shvMap(shvList(r["candidates"])[1])["incoming_edges"])) != 2 {
		t.Fatal("shared history reference lost")
	}
	for name, edit := range map[string]func(map[string]any){"wrong_digest": func(p map[string]any) {
		shvMap(shvList(p["objects"])[0])["digest"] = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	}, "noncanonical_index": func(p map[string]any) { shvMap(shvList(p["edges"])[0])["pointer"] = "/snapshots/00" }, "uninspected_source": func(p map[string]any) { shvMap(shvList(p["edges"])[0])["from"] = "source" }, "unselected_root": func(p map[string]any) { p["root_ids"] = []any{"unknown"} }} {
		t.Run(name, func(t *testing.T) {
			bad := shvClone(t, input)
			edit(bad)
			if _, e := shvReferencesAnalyze(bad); e == nil {
				t.Fatal("invalid reference accepted")
			}
		})
	}
	bad := shvClone(t, r)
	bad["deletion_authorized"] = true
	bad = shvReseal(t, bad)
	if ValidateSHVProfileResultVersion("references_analyze", shvRaw(t, input), shvRaw(t, bad), SHVProfileDiagnosticsVersion) == nil {
		t.Fatal("invented deletion authority accepted")
	}
	if ValidateSHVProfileResult("references_analyze", shvRaw(t, input), shvRaw(t, r)) == nil {
		t.Fatal("older owner admitted new operation")
	}
}
func TestSHVProfileDiagnosticsInstalledCorrespondence(t *testing.T) {
	dir := os.Getenv("SHV_DIAGNOSTIC_CASES")
	if dir == "" {
		t.Skip("set SHV_DIAGNOSTIC_CASES to the installed diagnostic campaign")
	}
	paths, e := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9].json"))
	if e != nil || len(paths) == 0 {
		t.Fatal(paths, e)
	}
	version := SHVProfileDiagnosticsVersion
	if selected := os.Getenv("SHV_DIAGNOSTIC_VERSION"); selected != "" {
		version = selected
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
		checked++
		t.Run(filepath.Base(path), func(t *testing.T) {
			op := shvText(c["operation"])
			p, r := shvRaw(t, c["input"]), shvRaw(t, c["result"])
			if e := ValidateSHVProfileResultVersion(op, p, r, version); e != nil {
				t.Fatal(e)
			}
			if op == "inspect" {
				return
			}
			for _, key := range []string{"extra", "canonical_apply_enabled"} {
				bad := shvClone(t, shvMap(c["result"]))
				bad[key] = true
				bad = shvReseal(t, bad)
				if ValidateSHVProfileResultVersion(op, p, shvRaw(t, bad), version) == nil {
					t.Fatal("resealed false result accepted", key)
				}
			}
			if op == "extraction_diagnose" {
				forged := shvClone(t, shvMap(c["result"]))
				for _, subject := range shvList(forged["subjects"]) {
					for _, item := range shvList(shvMap(subject)["fields"]) {
						field := shvMap(item)
						if field["status"] == "extracted" {
							shvMap(field["assertion"])["value"] = "invented value"
							forged = shvReseal(t, forged)
							if ValidateSHVProfileResultVersion(op, p, shvRaw(t, forged), version) == nil {
								t.Fatal("resealed false extraction accepted")
							}
							break
						}
					}
				}

				bad := shvClone(t, shvMap(c["result"]))
				bad["counts"] = map[string]any{"extracted": 0, "failed": 0, "unavailable": 0}
				bad = shvReseal(t, bad)
				if !scvEqual(bad, c["result"]) && ValidateSHVProfileResultVersion(op, p, shvRaw(t, bad), version) == nil {
					t.Fatal("invented counts accepted")
				}
			}
		})
	}
	if checked < 15 {
		t.Fatal("incomplete diagnostic campaign", checked)
	}
}

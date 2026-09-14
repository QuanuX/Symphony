package knowledgeengine

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSHVStoreIndependentNativeCorrespondence(t *testing.T) {
	raw, e := os.ReadFile("testdata/shv-store/native-results.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures []struct {
		Operation string          `json:"operation"`
		Input     json.RawMessage `json:"input"`
		Result    json.RawMessage `json:"result"`
	}
	if e = json.Unmarshal(raw, &fixtures); e != nil {
		t.Fatal(e)
	}
	for _, f := range fixtures {
		t.Run(f.Operation, func(t *testing.T) {
			if e := ValidateSHVStoreResult(f.Operation, f.Input, f.Result); e != nil {
				t.Fatal(e)
			}
		})
		mutations := map[string]func(map[string]any){"extra-field": func(r map[string]any) { r["unreviewed"] = true }}
		if f.Operation == "inspect" {
			mutations["authority-escalation"] = func(r map[string]any) { r["canonical_apply_enabled"] = true }
			mutations["missing-operation"] = func(r map[string]any) { r["operations"] = shvList(r["operations"])[1:] }
		} else {
			mutations["wrong-count"] = func(r map[string]any) { shvMap(r["counts"])["nodes"] = json.Number("0") }
			if f.Operation == "prepare" || f.Operation == "commit" {
				mutations["wrong-intent"] = func(r map[string]any) {
					i := shvMap(r["intent"])
					i["operation_id"] = "other"
					delete(i, "digest")
					r["intent"] = shvSealNew(i)
				}
			}
			if f.Operation == "query" {
				mutations["missing-row"] = func(r map[string]any) { r["rows"] = []any{} }
				mutations["forged-page"] = func(r map[string]any) { r["next_cursor"] = nil }
			}
		}
		for name, change := range mutations {
			t.Run(f.Operation+"/"+name, func(t *testing.T) {
				r, _ := shvObject(f.Result)
				change(r)
				field := "digest"
				if f.Operation == "inspect" {
					field = "descriptor_digest"
				}
				delete(r, field)
				d, _ := SCVDigest(r)
				r[field] = d
				bad, _ := json.Marshal(r)
				if ValidateSHVStoreResult(f.Operation, f.Input, bad) == nil {
					t.Fatal("resealed mismatch accepted")
				}
			})
		}
	}
}
func TestSHVStoreRejectsInputAmbiguity(t *testing.T) {
	raw, _ := os.ReadFile("testdata/shv-store/native-results.json")
	var f []map[string]json.RawMessage
	json.Unmarshal(raw, &f)
	p, _ := shvObject(f[3]["input"])
	for _, change := range []func(map[string]any){func(v map[string]any) { v["backend"] = "other" }, func(v map[string]any) { v["limit"] = json.Number("1.5") }, func(v map[string]any) { v["tops_id"] = "not-a-tops" }, func(v map[string]any) { v["kind"] = "nodes"; v["filters"] = map[string]any{"label": "x"} }, func(v map[string]any) { v["cursor"] = map[string]any{"query_digest": "x", "after_key": "y"} }} {
		raw, _ := json.Marshal(p)
		v, _ := shvObject(raw)
		change(v)
		if shvStoreInput("query", v) == nil {
			t.Fatal("bad input accepted")
		}
	}
}

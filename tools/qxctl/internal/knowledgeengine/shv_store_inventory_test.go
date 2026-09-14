package knowledgeengine

import (
	"encoding/json"
	"os"
	"testing"
)

func inventoryFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	b, e := os.ReadFile("testdata/shv-store/inventory.json")
	if e != nil {
		t.Fatal(e)
	}
	v, e := shvObject(b)
	if e != nil {
		t.Fatal(e)
	}
	return shvMap(v["input"]), shvMap(v["result"])
}
func TestSHVStoreInventoryCorrespondence(t *testing.T) {
	p, r := inventoryFixture(t)
	a, _ := json.Marshal(p)
	b, _ := json.Marshal(r)
	if e := ValidateSHVStoreResultVersion("inventory", a, b, SHVStoreInventoryVersion); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVStoreResultVersion("inventory", a, b, SHVStoreVersion) == nil {
		t.Fatal("legacy release gained unsupported operation")
	}
	cases := map[string]func(map[string]any){
		"missing-page-record": func(r map[string]any) { r["records"] = []any{} },
		"forged-continuation": func(r map[string]any) { r["next_cursor"] = nil },
		"scope":               func(r map[string]any) { shvMap(r["manifest"])["namespace"] = "other" },
		"false-reference-count": func(r map[string]any) {
			shvMap(shvList(shvMap(r["manifest"])["snapshots"])[0])["committed_operations"] = json.Number("2")
		},
		"wrong-capacity": func(r map[string]any) { shvMap(shvMap(r["manifest"])["capacity"])["intents"] = json.Number("129") },
		"reordered-entries": func(r map[string]any) {
			m := shvMap(r["manifest"])
			es := shvList(m["entries"])
			es[0], es[1] = es[1], es[0]
		},
		"unknown-writer": func(r map[string]any) {
			shvMap(shvMap(shvList(shvMap(r["manifest"])["entries"])[0])["connector"])["Version"] = "99.0.0"
		},
		"extra-field": func(r map[string]any) { r["policy"] = true },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			p, r := inventoryFixture(t)
			change(r)
			m := shvMap(r["manifest"])
			delete(m, "digest")
			r["manifest"] = shvSealNew(m)
			delete(r, "digest")
			r = shvSealNew(r)
			a, _ := json.Marshal(p)
			b, _ := json.Marshal(r)
			if ValidateSHVStoreResultVersion("inventory", a, b, SHVStoreInventoryVersion) == nil {
				t.Fatal("resealed inventory drift accepted")
			}
		})
	}
}
func TestSHVStoreInventoryInputBounds(t *testing.T) {
	for _, change := range []func(map[string]any){func(p map[string]any) { p["limit"] = json.Number("17") }, func(p map[string]any) { p["limit"] = json.Number("1.5") }, func(p map[string]any) { p["cursor"] = map[string]any{"revision": "bad", "after_operation_id": "a"} }, func(p map[string]any) { p["expected_revision"] = "latest" }, func(p map[string]any) { p["prune"] = true }} {
		p, _ := inventoryFixture(t)
		change(p)
		if shvStoreInventoryInput(p) == nil {
			t.Fatal("ambiguous or unbounded inventory input accepted")
		}
	}
}

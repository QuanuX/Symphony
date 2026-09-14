package knowledgeengine

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func transferFixture(t *testing.T) map[string]any {
	t.Helper()
	b, e := os.ReadFile("testdata/shv-store/transfer-plan.json")
	if e != nil {
		t.Fatal(e)
	}
	p, e := shvObject(b)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestSHVTransferIndependentPlan(t *testing.T) {
	r := transferFixture(t)
	input := pubTestRaw(t, r["input"])
	raw := pubTestRaw(t, r)
	if e := ValidateSHVStoreResultVersion("transfer_plan", input, raw, SHVStoreTransferVersion); e != nil {
		t.Fatal(e)
	}
	for _, v := range []string{SHVStoreVersion, SHVStoreInventoryVersion} {
		if ValidateSHVStoreResultVersion("transfer_plan", input, raw, v) == nil {
			t.Fatal("older reader gained planning")
		}
	}
	for name, change := range map[string]func(map[string]any){
		"lineage": func(r map[string]any) {
			shvMap(shvList(r["selected"])[0])["target_snapshot_digest"] = "sha256:" + strings.Repeat("0", 64)
		},
		"blocker":      func(r map[string]any) { r["blockers"] = []any{"snapshot_capacity"} },
		"requirements": func(r map[string]any) { shvMap(r["requirements"])["snapshots"] = json.Number("2") },
		"selection":    func(r map[string]any) { a := shvList(r["selected"]); a[0], a[1] = a[1], a[0] },
		"exclusion":    func(r map[string]any) { r["excluded_operation_ids"] = []any{"committed"} },
	} {
		t.Run(name, func(t *testing.T) {
			r := transferFixture(t)
			p := pubTestRaw(t, r["input"])
			change(r)
			delete(r, "digest")
			r = shvSealNew(r)
			if ValidateSHVStoreResultVersion("transfer_plan", p, pubTestRaw(t, r), SHVStoreTransferVersion) == nil {
				t.Fatal("resealed divergence accepted")
			}
		})
	}
}
func TestSHVPublicationExplicitWriterAdmission(t *testing.T) {
	for _, writer := range []string{SHVStoreVersion, SHVStoreInventoryVersion, SHVStoreTransferVersion} {
		b, e := os.ReadFile("testdata/shv-store/publication-definition.json")
		if e != nil {
			t.Fatal(e)
		}
		d, e := shvObject(b)
		if e != nil {
			t.Fatal(e)
		}
		in := map[string]any{"operation_id": "test", "current": nil, "desired": d, "reason": "caller choice"}
		members := shvList(d["members"])
		if len(members) == 0 {
			t.Fatal("fixture must exercise a real binding")
		}
		for _, v := range members {
			x := shvMap(v)
			inst := shvMap(x["store_installation"])
			old := shvText(inst["Version"])
			inst["Version"] = writer
			for _, k := range []string{"ReceiptPath", "ExecutablePath"} {
				inst[k] = strings.ReplaceAll(shvText(inst[k]), old, writer)
			}
			shvMap(x["store"])["version"] = writer
		}
		p, e := pubPlanVersion(in, SHVPublicationTransferVersion)
		if e != nil {
			t.Fatal(writer, e)
		}
		if e = ValidateSHVPublicationResultVersion("publication_plan", pubTestRaw(t, in), pubTestRaw(t, p), SHVPublicationTransferVersion); e != nil {
			t.Fatal(e)
		}
		e = ValidateSHVPublicationResult("publication_plan", pubTestRaw(t, in), pubTestRaw(t, p))
		if (e == nil) != (writer == SHVStoreVersion) {
			t.Fatal("legacy publisher admission changed", writer, e)
		}
	}
}

func TestSHVInventoryLegacyReaderAdmission(t *testing.T) {
	r := transferFixture(t)
	p := map[string]any{"tops_id": shvMap(r["input"])["tops_id"], "namespace": shvMap(r["input"])["namespace"], "expected_revision": nil, "cursor": nil, "limit": json.Number("16")}
	records := []any{}
	for _, v := range shvList(r["selected"]) {
		records = append(records, shvMap(v)["source"])
	}
	out := shvSealNew(map[string]any{"protocol": "symphony.shv.graph-store-inventory.v1", "backend": "duckdb", "input": p, "manifest": r["manifest"], "records": records, "next_cursor": nil, "physical_bytes": map[string]any{"database": json.Number("1"), "wal": json.Number("0")}})
	if e := ValidateSHVStoreResultVersion("inventory", pubTestRaw(t, p), pubTestRaw(t, out), SHVStoreTransferVersion); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVStoreResultVersion("inventory", pubTestRaw(t, p), pubTestRaw(t, out), SHVStoreInventoryVersion) == nil {
		t.Fatal("legacy reader admitted future writer")
	}
}

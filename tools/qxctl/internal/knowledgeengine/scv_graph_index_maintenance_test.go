package knowledgeengine

import (
	"encoding/json"
	"os"
	"testing"
)

func maintenanceFixture(t *testing.T, name string) (map[string]any, map[string]any) {
	t.Helper()
	raw, err := os.ReadFile("../../../../modules/scv-graph-duckdb-connector/tests/fixtures/maintenance-wire.json")
	if err != nil {
		t.Fatal(err)
	}
	all, err := scvObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := all[name].(map[string]any)
	return v["input"].(map[string]any), v["result"].(map[string]any)
}
func maintenanceSeal(t *testing.T, m map[string]any) {
	t.Helper()
	delete(m, "digest")
	d, err := SCVDigest(m)
	if err != nil {
		t.Fatal(err)
	}
	m["digest"] = d
}
func TestSCVGraphIndexInventoryIndependentValidation(t *testing.T) {
	input, result := maintenanceFixture(t, "inventory")
	if err := ValidateSCVGraphIndexResult("inventory", graphIndexTestRaw(t, input), graphIndexTestRaw(t, result)); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"drop_page":         func(r map[string]any) { r["records"] = []any{} },
		"omit_continuation": func(r map[string]any) { r["next_cursor"] = nil },
		"wrong_cursor":      func(r map[string]any) { r["next_cursor"].(map[string]any)["after_operation_id"] = "missing" },
		"summary_projection": func(r map[string]any) {
			m := r["manifest"].(map[string]any)
			m["entries"].([]any)[0].(map[string]any)["projection_digest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
			maintenanceSeal(t, m)
		},
		"shared_references": func(r map[string]any) {
			m := r["manifest"].(map[string]any)
			m["snapshots"].([]any)[0].(map[string]any)["operation_ids"] = []any{"wrong"}
			maintenanceSeal(t, m)
		},
		"capacity": func(r map[string]any) {
			m := r["manifest"].(map[string]any)
			m["capacity"].(map[string]any)["intents"] = 129
			maintenanceSeal(t, m)
		},
		"unverified_page": func(r map[string]any) {
			s := r["records"].([]any)[0].(map[string]any)
			s["index_verified"] = true
			maintenanceSeal(t, s)
		},
		"negative_physical": func(r map[string]any) { r["physical_bytes"].(map[string]any)["wal"] = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			i, r := maintenanceFixture(t, "inventory")
			mutate(r)
			maintenanceSeal(t, r)
			if ValidateSCVGraphIndexResult("inventory", graphIndexTestRaw(t, i), graphIndexTestRaw(t, r)) == nil {
				t.Fatal("accepted resealed inventory corruption")
			}
		})
	}
}
func TestSCVGraphIndexTransferIndependentValidation(t *testing.T) {
	input, result := maintenanceFixture(t, "plan")
	if err := ValidateSCVGraphIndexResult("transfer_plan", graphIndexTestRaw(t, input), graphIndexTestRaw(t, result)); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"target_lineage": func(r map[string]any) {
			r["selected"].([]any)[0].(map[string]any)["target_intent_digest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		},
		"selection_order":     func(r map[string]any) { a := r["selected"].([]any); a[0], a[1] = a[1], a[0] },
		"capacity_accounting": func(r map[string]any) { r["requirements"].(map[string]any)["snapshots"] = 0 },
		"unselected":          func(r map[string]any) { r["excluded_operation_ids"] = []any{"alias"} },
		"invented_blocker":    func(r map[string]any) { r["blockers"] = []any{"provider_preference"}; r["disposition"] = "blocked" },
		"stale_revision": func(r map[string]any) {
			r["manifest"].(map[string]any)["global_revision"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
			maintenanceSeal(t, r["manifest"].(map[string]any))
		},
		"unknown_field": func(r map[string]any) { r["execute"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			i, r := maintenanceFixture(t, "plan")
			mutate(r)
			maintenanceSeal(t, r)
			if ValidateSCVGraphIndexResult("transfer_plan", graphIndexTestRaw(t, i), graphIndexTestRaw(t, r)) == nil {
				t.Fatal("accepted resealed transfer corruption")
			}
		})
	}
}
func TestSCVGraphIndexMaintenanceMalformedTypes(t *testing.T) {
	for _, field := range []string{"manifest", "records", "physical_bytes"} {
		i, r := maintenanceFixture(t, "inventory")
		r[field] = json.Number("3")
		maintenanceSeal(t, r)
		if ValidateSCVGraphIndexResult("inventory", graphIndexTestRaw(t, i), graphIndexTestRaw(t, r)) == nil {
			t.Fatal(field)
		}
	}
	for _, field := range []string{"selected", "requirements", "blockers"} {
		i, r := maintenanceFixture(t, "plan")
		r[field] = nil
		maintenanceSeal(t, r)
		if ValidateSCVGraphIndexResult("transfer_plan", graphIndexTestRaw(t, i), graphIndexTestRaw(t, r)) == nil {
			t.Fatal(field)
		}
	}
}

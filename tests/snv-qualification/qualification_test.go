package main

import "testing"

func nativeFixture() object {
	return object{"owner": "local-observer", "owner_version": "0.1.0-dev", "observation_id": "capture", "acquisition_route": "native_fixed_sources", "node_ref": nil, "physical_inventory_complete": false, "allocation_verified": false, "node_association": "caller_supplied_unverified", "subject_ids": []any{}, "fields": []any{object{"field": "memory_available", "status": "observed", "value": float64(0)}}, "platform": object{"os": "linux"}, "status": "complete", "capture": object{"atomic_inventory": false, "started_unix_ms": float64(100), "finished_unix_ms": float64(101)}}
}
func TestObserverEvidenceBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		change    func(object)
		os        string
		wantError bool
	}{
		{"observed zero remains evidence", func(object) {}, "linux", false},
		{"partial denied remains null", func(v object) {
			v["status"] = "partial"
			v["fields"] = []any{object{"field": "memory_available", "status": "permission_denied", "value": nil}}
		}, "linux", false},
		{"unseen cannot become zero", func(v object) {
			v["fields"] = []any{object{"field": "memory_available", "status": "permission_denied", "value": float64(0)}}
		}, "linux", true},
		{"synthetic cannot certify native", func(v object) { v["acquisition_route"] = "sdk_supplied_reader" }, "linux", true},
		{"identity cannot be invented", func(v object) { v["subject_ids"] = []any{"invented"} }, "linux", true},
		{"extra fields refused", func(v object) {
			v["fields"] = append(v["fields"].([]any), object{"field": "cpu_present", "status": "observed", "value": float64(1)})
		}, "linux", true},
		{"wrong selected field refused", func(v object) { v["fields"].([]any)[0].(map[string]any)["field"] = "cpu_present" }, "linux", true},
		{"unknown field status refused", func(v object) {
			v["fields"] = []any{object{"field": "memory_available", "status": "unknown", "value": nil}}
		}, "linux", true},
		{"complete cannot conceal unseen", func(v object) {
			v["fields"] = []any{object{"field": "memory_available", "status": "unavailable", "value": nil}}
		}, "linux", true},
		{"capture order checked", func(v object) { v["capture"].(map[string]any)["finished_unix_ms"] = float64(99) }, "linux", true},
		{"macOS cannot claim Linux", func(object) {}, "darwin", true},
		{"macOS explicit unavailable", func(v object) {
			v["status"] = "unavailable"
			v["platform"] = object{"os": nil}
			v["fields"] = []any{object{"field": "memory_available", "status": "unavailable", "value": nil}}
		}, "darwin", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := nativeFixture()
			tt.change(v)
			err := observer(v, []string{"memory_available"}, "capture", tt.os)
			if (err != nil) != tt.wantError {
				t.Fatalf("error=%v want refusal=%v", err, tt.wantError)
			}
		})
	}
}

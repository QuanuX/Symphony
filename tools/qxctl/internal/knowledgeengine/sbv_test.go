package knowledgeengine

import (
	"encoding/json"
	"testing"
)

func TestSBVQueryCorrespondence(t *testing.T) {
	p := map[string]any{"expected_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "pointer": "/a", "limit": "2", "cursor": ""}
	query, _ := SCVCanonical(map[string]any{"snapshot": p["expected_sha256"], "pointer": p["pointer"], "limit": p["limit"]})
	hash := digestBytes(query)[7:]
	r := map[string]any{"protocol": "symphony.sbv.result-query.v1", "content_sha256": p["expected_sha256"], "pointer": "/a", "query_sha256": hash, "offset": "0", "total": "1", "complete": true, "next_cursor": "", "nodes": []any{map[string]any{"pointer": "/a/0", "type": "string", "value": "9007199254740993", "children": "0"}}}
	raw, _ := json.Marshal(r)
	if err := validateSBVResult("result_query", p, raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pointer", "query_sha256", "content_sha256", "offset", "total", "next_cursor"} {
		bad := map[string]any{}
		for k, v := range r {
			bad[k] = v
		}
		bad[key] = "wrong"
		b, _ := json.Marshal(bad)
		if validateSBVResult("result_query", p, b) == nil {
			t.Fatal("accepted changed " + key)
		}
	}
}
func TestSBVExactReleaseAndCapabilities(t *testing.T) {
	if _, err := InspectSBV("/private/tmp", "0.1.0-dev"); err == nil {
		t.Fatal("implicit release upgrade")
	}
	p := map[string]any{}
	if validateSBVResult("capabilities", p, []byte(`{"protocol":"symphony.sbv.capabilities.v1","engine_version":"0.13.0-dev","cpu":{"available":true},"cuda":{"available":true},"tensor":{"available":false},"live":{"available":false}}`)) == nil {
		t.Fatal("unexpected accelerator capability accepted")
	}
}

func TestSBVDatasetUserLimitEvidence(t *testing.T) {
	p := map[string]any{"directory": "/private/tmp/example", "instance_id": "00000000000000000000000000000001", "source_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "source_path": "/example.dbn", "dataset": "GLBX.MDP3", "memory_budget_bytes": nil, "residency": "pageable", "max_concurrent_jobs": "1", "worker_budget": "1", "idle_timeout_ms": "0"}
	r := map[string]any{}
	for k, v := range p {
		r[k] = v
	}
	r["protocol"] = "symphony.sbv.dataset-load.v1"
	r["engine_version"] = SBVAdministrationInterfaceVersion
	r["state"] = "ready"
	for _, k := range []string{"events", "decoded_bytes", "load_buffer_bytes", "source_reads", "decode_passes", "active_jobs", "active_workers", "completed_jobs", "failed_jobs"} {
		r[k] = "1"
	}
	limits := map[string]any{"max_source_bytes": nil, "max_source_events": nil, "max_metadata_bytes": nil}
	r["dataset_limits"] = limits
	verify := func(want bool) {
		t.Helper()
		b, _ := json.Marshal(r)
		if (validateSBVResult("dataset_load", p, b) == nil) != want {
			t.Fatalf("dataset user-limit evidence accepted=%v wanted=%v: %s", !want, want, b)
		}
	}
	verify(true)
	limits["max_source_bytes"] = "1099511627776"
	verify(false)
	p["dataset_limits"] = map[string]any{"max_source_bytes": "1099511627776", "max_source_events": nil, "max_metadata_bytes": nil}
	verify(true)
	for _, v := range []any{"0", "01", "18446744073709551616", 123} {
		limits["max_source_events"] = v
		verify(false)
	}
	limits["max_source_events"] = nil
	r["memory_budget_bytes"] = "1099511627776"
	verify(false)
	p["memory_budget_bytes"] = "1099511627776"
	verify(true)
}

package knowledgeengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func partitionedFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	output := map[string]any{"kind": "partitioned", "bundle_path": "/private/results/final", "workspace_path": "/private/workspaces/selected", "write_options": map[string]any{"page_bytes": "8192", "index_fanout": "3"}}
	request := map[string]any{"protocol": "symphony.sbv.generate-census-input.v1", "output": output, "extensions": map[string]any{"exact": "<>&\u2028\u2029\\u2028"}}
	ref := map[string]any{"manifest_path": "/private/results/final/manifest.json", "manifest_sha256": strings.Repeat("a", 64), "content_sha256": strings.Repeat("b", 64)}
	receipt := map[string]any{"protocol": "symphony.sbv.generate-census.v1", "status": "partial", "content_sha256": ref["content_sha256"], "summary": map[string]any{"status": "available", "reason": "", "data": map[string]any{"signal_count": "65537", "closed_census": true}},
		"storage": map[string]any{"kind": "partitioned", "reference": ref, "workspace_path": output["workspace_path"], "workspace_retained": true, "manifest_bytes": "1024", "logical_body_bytes": "268435456", "logical_body_nodes": "5000000", "logical_body_values": "3000000", "exported_value_nodes": "3000001", "page_files_created": "4096", "page_bytes_created": "100000000", "write_options": output["write_options"]}}
	return request, receipt
}

func partitionedValidate(t *testing.T, op string, p, m map[string]any, admitted bool) {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	handled, err := ValidateSBVPartitionedResult(op, p, raw)
	if !handled || (err == nil) != admitted {
		t.Fatalf("%s handled=%v admitted=%v wanted=%v error=%v", op, handled, err == nil, admitted, err)
	}
}

func partitionedRecoveryFixture(t *testing.T) (map[string]any, map[string]any) {
	p, _ := partitionedFixture(t)
	m := map[string]any{"protocol": "symphony.sbv.generate-census.v1", "status": "recovery_required", "code": "sbv.partitioned_result_incomplete", "cause": map[string]any{"category": "contract", "code": "sbv.contract"}, "request_sha256": recoveryHash(t, p), "automatic_retry": false, "rollback_performed": false,
		"recovery": map[string]any{"output": p["output"], "phase": "produce", "workspace_created": true, "workspace_creation_durable": true, "automatic_cleanup": false, "final_storage": nil}}
	return p, m
}

func TestSBVPartitionedOutputAndSuccess(t *testing.T) {
	p, m := partitionedFixture(t)
	partitionedValidate(t, "generate_census", p, m, true)
	for name, change := range map[string]func(map[string]any){
		"legacy_path":                    func(r map[string]any) { r["path"] = "/private/pretend-file.json" },
		"wrong_protocol":                 func(r map[string]any) { r["protocol"] = "symphony.sbv.evaluate.v1" },
		"complete_is_not_logical_status": func(r map[string]any) { r["status"] = "complete" },
		"wrong_content":                  func(r map[string]any) { r["content_sha256"] = strings.Repeat("c", 64) },
		"wrong_manifest": func(r map[string]any) {
			r["storage"].(map[string]any)["reference"].(map[string]any)["manifest_path"] = "/private/unrelated/manifest.json"
		},
		"scratch_removed":   func(r map[string]any) { r["storage"].(map[string]any)["workspace_retained"] = false },
		"scratch_relocated": func(r map[string]any) { r["storage"].(map[string]any)["workspace_path"] = "/private/unrelated" },
		"wrong_layout": func(r map[string]any) {
			r["storage"].(map[string]any)["write_options"].(map[string]any)["page_bytes"] = "512"
		},
		"wrong_count":         func(r map[string]any) { r["storage"].(map[string]any)["exported_value_nodes"] = "3000000" },
		"unknown_storage":     func(r map[string]any) { r["storage"].(map[string]any)["unregistered"] = "x" },
		"unavailable_summary": func(r map[string]any) { r["summary"].(map[string]any)["status"] = "unavailable" },
		"numeric_summary":     func(r map[string]any) { r["summary"].(map[string]any)["data"].(map[string]any)["signal_count"] = 65537 },
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, m)
			change(r)
			partitionedValidate(t, "generate_census", p, r, false)
		})
	}
	for name, change := range map[string]func(map[string]any){
		"both":              func(r map[string]any) { r["output_path"] = "/private/legacy.json" },
		"neither":           func(r map[string]any) { delete(r, "output") },
		"null":              func(r map[string]any) { r["output"] = nil },
		"missing_workspace": func(r map[string]any) { delete(r["output"].(map[string]any), "workspace_path") },
		"same_path":         func(r map[string]any) { r["output"].(map[string]any)["workspace_path"] = "/private/results/final" },
		"ancestor":          func(r map[string]any) { r["output"].(map[string]any)["workspace_path"] = "/private/results" },
		"descendant": func(r map[string]any) {
			r["output"].(map[string]any)["workspace_path"] = "/private/results/final/scratch"
		},
		"cleanup": func(r map[string]any) { r["output"].(map[string]any)["automatic_cleanup"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, p)
			change(r)
			if sbvPartitionedRequestPreflight("generate_census", r) == nil {
				t.Fatal("invalid output accepted")
			}
		})
	}
	legacy := recoveryClone(t, p)
	delete(legacy, "output")
	legacy["output_path"] = "/private/legacy.json"
	if err := sbvPartitionedRequestPreflight("generate_census", legacy); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"protocol":"symphony.sbv.generate-census.v1","path":"/private/legacy.json","status":"partial"}`)
	if handled, err := ValidateSBVPartitionedResult("generate_census", legacy, raw); handled || err != nil {
		t.Fatal("legacy receipt intercepted", handled, err)
	}
	partitionedValidate(t, "generate_census", legacy, m, false)
}

func TestSBVPartitionedReceiptDepthReserve(t *testing.T) {
	for _, layers := range []int{57, 58} {
		p, m := partitionedFixture(t)
		var value any = "leaf"
		for range layers {
			value = []any{value}
		}
		m["summary"].(map[string]any)["data"].(map[string]any)["nested"] = value
		raw, err := json.Marshal(m)
		if err != nil || validateJSONObject(raw, maxResponseBytes) != nil {
			t.Fatal("fixture must fit the ordinary process depth profile", err)
		}
		// Root=0, summary=1, data=2, nested=3: the deepest leaf is
		// 60/61 respectively. Only the first leaves all four frame levels.
		partitionedValidate(t, "generate_census", p, m, layers == 57)
	}
}

func TestSBVPartitionedRecoveryCause(t *testing.T) {
	for _, cause := range []map[string]any{
		{"category": "contract", "code": "sbv.contract"},
		{"category": "contract", "code": "logical.invalid_value"},
		{"category": "deadline", "code": "deadline.exceeded"},
		{"category": "deadline", "code": "request.deadline_expired"},
		{"category": "storage", "code": "bundle.output_exists"},
		{"category": "allocation", "code": "sbv.allocation_failed"},
		{"category": "unexpected", "code": "sbv.invalid_native_error_code"},
		{"category": "unexpected", "code": "bundle.interrupted"},
		{"category": "contract", "code": "native.module-safe_code.1"},
	} {
		p, m := partitionedRecoveryFixture(t)
		m["cause"] = cause
		partitionedValidate(t, "generate_census", p, m, true)
	}
	for _, cause := range []any{
		nil, "native private message",
		map[string]any{"category": "contract"},
		map[string]any{"category": "other", "code": "native.code"},
		map[string]any{"category": "deadline", "code": "sbv.contract"},
		map[string]any{"category": "contract", "code": "deadline.exceeded"},
		map[string]any{"category": "contract", "code": "bundle.output_exists"},
		map[string]any{"category": "storage", "code": "logical.invalid_value"},
		map[string]any{"category": "unexpected", "code": "sbv.allocation_failed"},
		map[string]any{"category": "contract", "code": "private\nmessage"},
		map[string]any{"category": "contract", "code": "Native.Code"},
		map[string]any{"category": "contract", "code": strings.Repeat("a", 129)},
		map[string]any{"category": "contract", "code": "native.code", "message": "private diagnostics"},
	} {
		p, m := partitionedRecoveryFixture(t)
		m["cause"] = cause
		partitionedValidate(t, "generate_census", p, m, false)
	}
	p, m := partitionedRecoveryFixture(t)
	delete(m, "cause")
	partitionedValidate(t, "generate_census", p, m, false)
}

func TestSBVPartitionedRecoveryStages(t *testing.T) {
	p, m := partitionedRecoveryFixture(t)
	partitionedValidate(t, "generate_census", p, m, true)
	for _, created := range []bool{false, true} {
		r := recoveryClone(t, m)
		d := r["recovery"].(map[string]any)
		d["phase"], d["workspace_created"], d["workspace_creation_durable"] = "create_workspace", created, false
		partitionedValidate(t, "generate_census", p, r, true)
	}
	final := map[string]any{"bundle_path": "/private/results/final", "manifest_path": "/private/results/final/manifest.json", "phase": "sync_manifest", "manifest_published": true, "durable": false, "page_files_created": "10", "page_bytes_created": "81920", "reference": map[string]any{"manifest_path": "/private/results/final/manifest.json", "manifest_sha256": strings.Repeat("a", 64), "content_sha256": strings.Repeat("b", 64)}}
	d := m["recovery"].(map[string]any)
	d["phase"], d["final_storage"] = "finalize_result", final
	partitionedValidate(t, "generate_census", p, m, true)
	for name, change := range map[string]func(map[string]any){
		"request_hash": func(r map[string]any) { r["request_sha256"] = strings.Repeat("c", 64) },
		"cause":        func(r map[string]any) { r["cause"] = "private native diagnostic" },
		"retry":        func(r map[string]any) { r["automatic_retry"] = true },
		"rollback":     func(r map[string]any) { r["rollback_performed"] = true },
		"wrong_output": func(r map[string]any) {
			r["recovery"].(map[string]any)["output"].(map[string]any)["workspace_path"] = "/private/other"
		},
		"workspace_not_created": func(r map[string]any) { r["recovery"].(map[string]any)["workspace_created"] = false },
		"workspace_not_synced":  func(r map[string]any) { r["recovery"].(map[string]any)["workspace_creation_durable"] = false },
		"cleanup":               func(r map[string]any) { r["recovery"].(map[string]any)["automatic_cleanup"] = true },
		"private_manifest_not_final": func(r map[string]any) {
			r["recovery"].(map[string]any)["final_storage"].(map[string]any)["bundle_path"] = "/private/workspaces/selected/spool"
		},
		"storage_before_finalize": func(r map[string]any) { r["recovery"].(map[string]any)["phase"] = "produce" },
		"published_without_reference": func(r map[string]any) {
			r["recovery"].(map[string]any)["final_storage"].(map[string]any)["reference"] = nil
		},
		"false_durable": func(r map[string]any) {
			s := r["recovery"].(map[string]any)["final_storage"].(map[string]any)
			s["phase"], s["durable"] = "publish_manifest", true
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, m)
			change(r)
			partitionedValidate(t, "generate_census", p, r, false)
		})
	}
	// Constructor errors may provide initial writer facts without claiming that
	// the final directory or manifest was successfully created.
	final["phase"], final["manifest_published"], final["reference"] = "create_directory", false, nil
	final["page_files_created"], final["page_bytes_created"] = "0", "0"
	partitionedValidate(t, "generate_census", p, m, true)
	d["final_storage"] = nil
	partitionedValidate(t, "generate_census", p, m, true)
}

func TestSBVPartitionedResidentRecoveryIdentities(t *testing.T) {
	child, recovery := partitionedRecoveryFixture(t)
	parent := map[string]any{"protocol": "symphony.sbv.dataset-execute-input.v1", "directory": "/private/socket", "instance_id": strings.Repeat("1", 32), "operation": "generate_census", "request": recoveryClone(t, child), "output": child["output"]}
	delete(parent["request"].(map[string]any), "output")
	result := map[string]any{"protocol": "symphony.sbv.dataset-execute.v1", "status": "recovery_required", "code": "sbv.dataset_execution_incomplete", "request_sha256": recoveryHash(t, parent), "automatic_retry": false, "rollback_performed": false, "operation": "generate_census", "child_recovery": recovery}
	partitionedValidate(t, "dataset_execute", parent, result, true)
	for name, change := range map[string]func(map[string]any){
		"child_hash_is_not_parent": func(r map[string]any) { r["child_recovery"].(map[string]any)["request_sha256"] = r["request_sha256"] },
		"parent_hash_is_not_child": func(r map[string]any) { r["request_sha256"] = r["child_recovery"].(map[string]any)["request_sha256"] },
		"child_protocol_not_renamed": func(r map[string]any) {
			r["child_recovery"].(map[string]any)["protocol"] = "symphony.sbv.dataset-execute.v1"
		},
		"operation":           func(r map[string]any) { r["operation"] = "evaluate" },
		"child_false_success": func(r map[string]any) { r["child_recovery"].(map[string]any)["status"] = "completed" },
		"nested_retry":        func(r map[string]any) { r["child_recovery"].(map[string]any)["automatic_retry"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, result)
			change(r)
			partitionedValidate(t, "dataset_execute", parent, r, false)
		})
	}
	for _, key := range []string{"output_path", "output"} {
		p := recoveryClone(t, parent)
		p["request"].(map[string]any)[key] = nil
		if _, err := sbvExpandedResidentRequest(p); err == nil {
			t.Fatal("child output form admitted", key)
		}
	}
	expanded, err := sbvExpandedResidentRequest(parent)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryHash(t, expanded) != recoveryHash(t, child) {
		t.Fatal("child expansion changed request identity")
	}
	if _, present := parent["request"].(map[string]any)["output"]; present {
		t.Fatal("expansion mutated original parent")
	}
	_, success := partitionedFixture(t)
	success["protocol"] = "symphony.sbv.dataset-execute.v1"
	partitionedValidate(t, "dataset_execute", parent, success, true)
}

func partitionedSourceFixture() map[string]any {
	return map[string]any{"kind": "bundle", "reference": map[string]any{"manifest_path": "/private/input/manifest.json", "manifest_sha256": strings.Repeat("a", 64), "content_sha256": strings.Repeat("b", 64)}, "selector": map[string]any{"kind": "pointer", "pointer": "/sections/user_extensions/data/rows"}, "read_options": map[string]any{"max_page_bytes": nil, "cache_bytes": "0"}}
}

func TestSBVPartitionedTypedSourcesAndSelections(t *testing.T) {
	source := partitionedSourceFixture()
	if !sbvBundleSource(source) {
		t.Fatal("valid typed reference rejected")
	}
	for name, change := range map[string]func(map[string]any){
		"raw_json_deferred": func(r map[string]any) { r["kind"] = "json_file" },
		"missing_options":   func(r map[string]any) { delete(r, "read_options") },
		"raw_cache_number":  func(r map[string]any) { r["read_options"].(map[string]any)["cache_bytes"] = 0 },
		"zero_page":         func(r map[string]any) { r["read_options"].(map[string]any)["max_page_bytes"] = "0" },
		"invalid_pointer":   func(r map[string]any) { r["selector"].(map[string]any)["pointer"] = "/bad~2" },
		"manifest_digest":   func(r map[string]any) { r["reference"].(map[string]any)["manifest_sha256"] = "wrong" },
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, source)
			change(r)
			if sbvBundleSource(r) {
				t.Fatal("invalid source admitted")
			}
		})
	}
	selectors := []map[string]any{{"kind": "all"}, {"kind": "range", "first": "65536", "count": nil}, {"kind": "range", "first": "18446744073709551615", "count": "0"}, {"kind": "ids", "ids": []any{}, "order": "supplied"}, {"kind": "ids", "ids": []any{"z", "a"}, "order": "census"}, {"kind": "referenced_ids", "source": source, "order": "supplied"}}
	for _, selector := range selectors {
		if !sbvSignalSelection(selector) {
			t.Fatal(selector)
		}
	}
	for _, selector := range []map[string]any{{"kind": "range", "first": "18446744073709551615", "count": "1"}, {"kind": "ids", "ids": []any{"a", "a"}, "order": "supplied"}, {"kind": "ids", "ids": []any{"a"}, "order": "silent_default"}, {"kind": "all", "limit": "4096"}} {
		if sbvSignalSelection(selector) {
			t.Fatal("invalid selection", selector)
		}
	}
	p, _ := partitionedFixture(t)
	p["protocol"] = "symphony.sbv.economics-input.v1"
	p["source"] = source
	p["selections"] = map[string]any{"kind": "referenced_rows", "source": source}
	if err := sbvPartitionedRequestPreflight("economics", p); err != nil {
		t.Fatal(err)
	}
	p["selections"] = map[string]any{"kind": "apply", "signals": selectors[0], "economics": map[string]any{"transform": map[string]any{}, "mode": "support_only", "nonexecution_pnl": nil}}
	if err := sbvPartitionedRequestPreflight("economics", p); err != nil {
		t.Fatal(err)
	}
	p["protocol"] = "symphony.sbv.evaluate-input.v1"
	p["census"] = source
	p["model"] = map[string]any{"id": "external_outcomes", "parameters": map[string]any{"outcomes_source": source}}
	if err := sbvPartitionedRequestPreflight("evaluate", p); err != nil {
		t.Fatal(err)
	}
	p["model"].(map[string]any)["parameters"].(map[string]any)["outcomes"] = []any{}
	if sbvPartitionedRequestPreflight("evaluate", p) == nil {
		t.Fatal("ambiguous external model inputs")
	}
	component := map[string]any{"source": source, "outcome_pointer": "/sections/economics/data/0"}
	p["protocol"] = "symphony.sbv.compose-economics-input.v1"
	p["components"] = []any{component}
	if err := sbvPartitionedRequestPreflight("compose_economics", p); err != nil {
		t.Fatal(err)
	}
	component["outcome_selector"] = map[string]any{"kind": "node_id", "node_id": "123"}
	if sbvPartitionedRequestPreflight("compose_economics", p) == nil {
		t.Fatal("unimplemented economic outcome selector")
	}
}

func TestSBVPartitionedReceiptControlHeadroom(t *testing.T) {
	p, m := partitionedFixture(t)
	data := m["summary"].(map[string]any)["data"].(map[string]any)
	rows := make([]any, 18)
	for i := range rows {
		rows[i] = strings.Repeat("x", 60000)
	}
	data["selected_details"] = rows
	raw, _ := json.Marshal(m)
	if _, err := sqavObject(raw, maxResponseBytes); err != nil {
		t.Fatalf("fixture must fit public process frame: %v", err)
	}
	partitionedValidate(t, "generate_census", p, m, false)
	data["selected_details"] = rows[:17]
	partitionedValidate(t, "generate_census", p, m, true)
	data["selected_details"] = make([]any, 32650)
	raw, _ = json.Marshal(m)
	if _, err := sqavObject(raw, maxResponseBytes); err != nil {
		t.Fatalf("fixture must fit original counted-value profile: %v", err)
	}
	partitionedValidate(t, "generate_census", p, m, false)
}

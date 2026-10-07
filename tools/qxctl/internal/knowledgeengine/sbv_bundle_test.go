package knowledgeengine

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func bundleValidate(t *testing.T, op string, request, result map[string]any, admitted bool) {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateSBVBundleResult(op, request, raw); (err == nil) != admitted {
		t.Fatalf("%s admission=%v, wanted=%v: %v", op, err == nil, admitted, err)
	}
}

func TestSBVBundleExportReceiptCorrespondence(t *testing.T) {
	request, result, _ := sbvBundleFixture(t, []byte("{\"v\":\"exact\"}\n"), "json")
	bundleValidate(t, "bundle_export", request, result, true)
	for name, change := range map[string]func(map[string]any){
		"unknown":            func(r map[string]any) { r["unknown"] = "x" },
		"source":             func(r map[string]any) { r["reference"].(map[string]any)["manifest_sha256"] = strings.Repeat("c", 64) },
		"path":               func(r map[string]any) { r["output_path"] = "/different/output.json" },
		"zero_bytes":         func(r map[string]any) { r["bytes"] = "0" },
		"noncanonical_bytes": func(r map[string]any) { r["bytes"] = "01" },
		"uint_overflow":      func(r map[string]any) { r["bytes"] = "18446744073709551616" },
		"numeric":            func(r map[string]any) { r["bytes"] = 10 },
		"counts":             func(r map[string]any) { r["exported_value_nodes"] = "7" },
		"hash":               func(r map[string]any) { r["file_sha256"] = strings.Repeat("A", 64) },
		"closure":            func(r map[string]any) { r["verification_extent"] = "accessed_pages" },
		"authorship":         func(r map[string]any) { r["source_authorship"] = "verified" },
		"suffix":             func(r map[string]any) { r["completion_suffix"] = "}\nextra" },
		"extensions":         func(r map[string]any) { r["extensions"].(map[string]any)["lost"] = "echo" },
		"stats":              func(r map[string]any) { delete(r["io_stats"].(map[string]any), "cache_hits") },
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, result)
			change(r)
			bundleValidate(t, "bundle_export", request, r, false)
		})
	}
	_, result, _ = sbvBundleFixture(t, []byte("placeholder\n"), "ndjson")
	request["format"], request["output_path"], request["reference"] = result["format"], result["output_path"], result["reference"]
	bundleValidate(t, "bundle_export", request, result, true)
	for _, suffix := range []string{strings.Replace(result["completion_suffix"].(string), `"nodes":"6"`, `"nodes":"7"`, 1), strings.Replace(result["completion_suffix"].(string), `"event":"end"`, `"event": "end"`, 1), strings.Replace(result["completion_suffix"].(string), strings.Repeat("a", 64), strings.Repeat("c", 64), 1)} {
		r := recoveryClone(t, result)
		r["completion_suffix"] = suffix
		bundleValidate(t, "bundle_export", request, r, false)
	}
}

func bundleQueryFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	reference := map[string]any{"manifest_path": "/private/bundle/manifest.json", "manifest_sha256": strings.Repeat("a", 64), "content_sha256": strings.Repeat("b", 64)}
	selector := map[string]any{"kind": "node_id", "node_id": "10"}
	request := map[string]any{"protocol": "symphony.sbv.bundle-query-input.v1", "reference": reference, "read_options": map[string]any{"max_page_bytes": nil, "cache_bytes": "0"}, "selector": selector, "cursor": nil, "row_limit": "1", "byte_limit": "65536", "extensions": map[string]any{}}
	cursor := map[string]any{"protocol": "symphony.sbv.bundle-cursor.v1", "reference": reference, "selector": selector, "selected_node_id": "10", "order": sbvBundleOrder, "access_profile": sbvBundleAccess, "next_offset": "1"}
	result := map[string]any{"protocol": "symphony.sbv.bundle-query.v1", "status": "complete", "source_authorship": "not_verified", "verification_extent": "accessed_pages", "reference": reference, "extensions": map[string]any{}, "selector": selector, "selected_node_id": "10", "selected_kind": "array", "order": sbvBundleOrder, "access_profile": sbvBundleAccess, "offset": "0", "total": "3", "complete": false, "next_cursor": cursor,
		"nodes": []any{map[string]any{"node_id": "11", "parent_id": "10", "edge": "0", "kind": "scalar", "value": "9007199254740993", "children": "0"}}, "io_stats": map[string]any{"files_read": "2", "bytes_read": "100", "cache_hits": "0", "maximum_file_bytes": "60"}}
	return request, result
}

func TestSBVBundleQueryCursorAndProgress(t *testing.T) {
	request, result := bundleQueryFixture(t)
	bundleValidate(t, "bundle_query", request, result, true)
	for name, change := range map[string]func(map[string]any){
		"empty_progress": func(r map[string]any) { r["nodes"] = []any{}; r["next_cursor"].(map[string]any)["next_offset"] = "0" },
		"false_complete": func(r map[string]any) { r["complete"] = true; r["next_cursor"] = nil },
		"source_cursor": func(r map[string]any) {
			r["next_cursor"].(map[string]any)["reference"].(map[string]any)["content_sha256"] = strings.Repeat("c", 64)
		},
		"cursor_order":  func(r map[string]any) { r["next_cursor"].(map[string]any)["order"] = "different" },
		"cursor_offset": func(r map[string]any) { r["next_cursor"].(map[string]any)["next_offset"] = "2" },
		"selected_node": func(r map[string]any) { r["selected_node_id"] = "9" },
		"edge":          func(r map[string]any) { r["nodes"].([]any)[0].(map[string]any)["edge"] = "01" },
		"parent":        func(r map[string]any) { r["nodes"].([]any)[0].(map[string]any)["parent_id"] = "9" },
		"node":          func(r map[string]any) { r["nodes"].([]any)[0].(map[string]any)["node_id"] = "10" },
		"kind":          func(r map[string]any) { r["nodes"].([]any)[0].(map[string]any)["kind"] = "object" },
		"children":      func(r map[string]any) { r["nodes"].([]any)[0].(map[string]any)["children"] = "1" },
		"extent":        func(r map[string]any) { r["verification_extent"] = "full_logical_closure" },
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, result)
			change(r)
			bundleValidate(t, "bundle_query", request, r, false)
		})
	}
	// A cursor carries source and traversal semantics, never a fixed page size.
	request["cursor"] = result["next_cursor"]
	request["row_limit"] = "2"
	request["byte_limit"] = "100000"
	result["offset"], result["complete"], result["next_cursor"] = "1", true, nil
	result["nodes"] = []any{map[string]any{"node_id": "12", "parent_id": "10", "edge": "1", "kind": "object", "value": map[string]any{}, "children": "0"}, map[string]any{"node_id": "13", "parent_id": "10", "edge": "2", "kind": "array", "value": []any{}, "children": "0"}}
	bundleValidate(t, "bundle_query", request, result, true)
	request["byte_limit"] = "100"
	bundleValidate(t, "bundle_query", request, result, false)
	request["byte_limit"] = "100000"
	request["cursor"].(map[string]any)["reference"].(map[string]any)["manifest_sha256"] = strings.Repeat("c", 64)
	// Clone to avoid fixture aliases hiding an independently stale reference.
	r := recoveryClone(t, result)
	r["reference"].(map[string]any)["manifest_sha256"] = strings.Repeat("a", 64)
	bundleValidate(t, "bundle_query", request, r, false)
}

func TestSBVBundleQueryScalarAndObjectEdges(t *testing.T) {
	request, result := bundleQueryFixture(t)
	selector := map[string]any{"kind": "pointer", "pointer": "/newline\nkey/~0/~1"}
	request["selector"], result["selector"] = selector, selector
	result["selected_kind"], result["total"], result["complete"], result["next_cursor"] = "scalar", "1", true, nil
	result["nodes"] = []any{map[string]any{"node_id": "10", "parent_id": "2", "edge": "strange\nkey", "kind": "scalar", "value": nil, "children": "0"}}
	bundleValidate(t, "bundle_query", request, result, true)
	selector["pointer"] = "/invalid~2"
	bundleValidate(t, "bundle_query", request, result, false)
	selector["pointer"] = "/object"
	result["selected_kind"], result["total"], request["row_limit"] = "object", "2", "2"
	result["nodes"] = []any{map[string]any{"node_id": "11", "parent_id": "10", "edge": "", "kind": "scalar", "value": false, "children": "0"}, map[string]any{"node_id": "12", "parent_id": "10", "edge": "a\n", "kind": "scalar", "value": "exact", "children": "0"}}
	bundleValidate(t, "bundle_query", request, result, true)
	result["nodes"].([]any)[1].(map[string]any)["edge"] = ""
	bundleValidate(t, "bundle_query", request, result, false)
}

func TestSBVBundleRecoveryAndRequestBinding(t *testing.T) {
	request, receipt, _ := sbvBundleFixture(t, []byte("{\"x\":null}\n"), "json")
	request["extensions"] = map[string]any{"unicode": "<>&\u2028\u2029\\u2028"}
	binary := map[string]any{}
	for _, key := range []string{"reference", "format", "bytes", "file_sha256", "completion_suffix"} {
		binary[key] = receipt[key]
	}
	result := map[string]any{"protocol": "symphony.sbv.bundle-export.v1", "status": "recovery_required", "code": "sbv.bundle_export_incomplete", "request_sha256": recoveryHash(t, request), "automatic_retry": false, "rollback_performed": false, "recovery": map[string]any{"output_path": request["output_path"], "stage_path": nil, "phase": "sync_export", "published": true, "durable": false, "expected_binary": binary}}
	bundleValidate(t, "bundle_export", request, result, true)
	for name, change := range map[string]func(map[string]any){
		"hash":             func(r map[string]any) { r["request_sha256"] = strings.Repeat("f", 64) },
		"retry":            func(r map[string]any) { r["automatic_retry"] = true },
		"freeform":         func(r map[string]any) { r["cause"] = "sensitive" },
		"missing_binary":   func(r map[string]any) { r["recovery"].(map[string]any)["expected_binary"] = nil },
		"phase":            func(r map[string]any) { r["recovery"].(map[string]any)["phase"] = "write_export" },
		"unpublished_sync": func(r map[string]any) { r["recovery"].(map[string]any)["published"] = false },
		"stage_parent":     func(r map[string]any) { r["recovery"].(map[string]any)["stage_path"] = "/unrelated/stage" },
		"binary_source": func(r map[string]any) {
			r["recovery"].(map[string]any)["expected_binary"].(map[string]any)["reference"].(map[string]any)["content_sha256"] = strings.Repeat("c", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := recoveryClone(t, result)
			change(r)
			bundleValidate(t, "bundle_export", request, r, false)
		})
	}
	r := result["recovery"].(map[string]any)
	r["published"], r["expected_binary"], r["phase"] = false, nil, "create_stage"
	bundleValidate(t, "bundle_export", request, result, true)
	r["durable"] = true
	bundleValidate(t, "bundle_export", request, result, false)
	// Import recovery exposes a discoverable reference only when known.
	request = map[string]any{"protocol": "symphony.sbv.bundle-import-input.v1", "input_path": "/private/input.json", "expected_file_sha256": strings.Repeat("c", 64), "bundle_path": "/private/bundle", "write_options": map[string]any{"page_bytes": "4096", "index_fanout": "4"}, "extensions": map[string]any{}}
	result = map[string]any{"protocol": "symphony.sbv.bundle-import.v1", "status": "recovery_required", "code": "sbv.bundle_import_incomplete", "request_sha256": recoveryHash(t, request), "automatic_retry": false, "rollback_performed": false, "recovery": map[string]any{"bundle_path": "/private/bundle", "manifest_path": "/private/bundle/manifest.json", "phase": "write_pages", "manifest_published": false, "durable": false, "page_files_created": "3", "page_bytes_created": "8192", "reference": nil}}
	bundleValidate(t, "bundle_import", request, result, true)
	r = result["recovery"].(map[string]any)
	r["phase"], r["manifest_published"] = "sync_manifest", true
	bundleValidate(t, "bundle_import", request, result, false)
	r["reference"] = map[string]any{"manifest_path": "/private/bundle/manifest.json", "manifest_sha256": strings.Repeat("d", 64), "content_sha256": strings.Repeat("e", 64)}
	bundleValidate(t, "bundle_import", request, result, true)
}

func TestSBVBundleExtensionsPreflight(t *testing.T) {
	// Count keys and values exactly, retaining the same bounded control reserve
	// as native code while imposing no aggregate bundle limit.
	for _, n := range []int{maxJSONValues - 512 - 3, maxJSONValues - 512 - 2} {
		values := make([]any, n)
		request := map[string]any{"extensions": map[string]any{"values": values}}
		err := sbvBundleControlPreflight("bundle_import", request)
		if (err == nil) != (n == maxJSONValues-512-3) {
			t.Fatal(strconv.Itoa(n), err)
		}
	}
	if sbvBundleControlPreflight("bundle_export", map[string]any{"extensions": map[string]any{"nested": []any{map[string]any{"numeric": 1}}}}) == nil {
		t.Fatal("numeric extension admitted")
	}
}

func TestSBVBundleImportInspectVerifyReceipts(t *testing.T) {
	request, export, _ := sbvBundleFixture(t, []byte("{\"x\":null}\n"), "json")
	base := recoveryClone(t, export)
	for _, key := range []string{"output_path", "format", "bytes", "file_sha256", "completion_suffix"} {
		delete(base, key)
	}
	options := map[string]any{"page_bytes": "512", "index_fanout": "2"}
	base["protocol"] = "symphony.sbv.bundle-verify.v1"
	bundleValidate(t, "bundle_verify", request, base, true)
	base["verification_extent"] = "manifest_only"
	bundleValidate(t, "bundle_verify", request, base, false)
	base["protocol"] = "symphony.sbv.bundle-inspect.v1"
	base["logical_protocol"], base["storage_protocol"] = "symphony.sbv.result.v1", "symphony.sbv.partitioned-result.v1"
	base["canonicalization"] = "nlohmann_compact_utf8_sorted_keys_no_numeric_literals.v1"
	base["root_node_id"], base["root_children"], base["write_options"] = "0", "5", options
	bundleValidate(t, "bundle_inspect", request, base, true)
	base["root_children"] = "6"
	bundleValidate(t, "bundle_inspect", request, base, false)
	base["root_children"] = "5"
	options["page_bytes"] = "511"
	bundleValidate(t, "bundle_inspect", request, base, false)
	options["page_bytes"] = "512"
	for _, key := range []string{"logical_protocol", "storage_protocol", "canonicalization", "root_node_id", "root_children"} {
		delete(base, key)
	}
	request = map[string]any{"protocol": "symphony.sbv.bundle-import-input.v1", "input_path": "/private/input.json", "expected_file_sha256": strings.Repeat("c", 64), "bundle_path": "/private/bundle", "write_options": options, "extensions": map[string]any{}}
	base["protocol"], base["verification_extent"] = "symphony.sbv.bundle-import.v1", "full_logical_closure"
	base["reference"].(map[string]any)["manifest_path"] = "/private/bundle/manifest.json"
	base["input_path"], base["input_file_sha256"], base["input_bytes"], base["bundle_path"], base["manifest_bytes"], base["page_files_created"], base["page_bytes_created"] = "/private/input.json", request["expected_file_sha256"], "500", "/private/bundle", "200", "0", "0"
	bundleValidate(t, "bundle_import", request, base, true)
	for _, key := range []string{"input_path", "input_file_sha256", "bundle_path"} {
		r := recoveryClone(t, base)
		r[key] = "wrong"
		bundleValidate(t, "bundle_import", request, r, false)
	}
	r := recoveryClone(t, base)
	delete(r, "io_stats")
	bundleValidate(t, "bundle_import", request, r, false)
	r = recoveryClone(t, base)
	r["logical_body_nodes"] = "18446744073709551615"
	bundleValidate(t, "bundle_import", request, r, false)
}

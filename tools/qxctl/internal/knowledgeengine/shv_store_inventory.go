package knowledgeengine

import (
	"fmt"
	"sort"
)

func storeInventorySummary(status map[string]any) map[string]any {
	intent := status["intent"].(map[string]any)
	snapshot := intent["snapshot"].(map[string]any)
	return map[string]any{"operation_id": intent["operation_id"], "intent_digest": intent["digest"], "snapshot_digest": snapshot["digest"], "state": status["state"], "graph_digest": snapshot["graph"].(map[string]any)["digest"], "connector": snapshot["connector"], "counts": status["counts"], "projection_digest": status["projection_digest"]}
}
func validateStoreInventoryManifest(value any, input map[string]any) (map[string]any, []any, error) {
	fail := fmt.Errorf("invalid graph index inventory manifest")
	m, ok := value.(map[string]any)
	if !ok || !shvFields(m, "protocol", "tops_id", "namespace", "entries", "snapshots", "global_revision", "global_counts", "capacity", "digest") || m["protocol"] != "symphony.shv.graph-store-inventory-manifest.v1" || !storeScope(m) || m["tops_id"] != input["tops_id"] || m["namespace"] != input["namespace"] || scvSeal(m, "digest") != nil || !indexDigest(m["global_revision"]) {
		return nil, nil, fail
	}
	if input["expected_revision"] != nil && (!indexDigest(input["expected_revision"]) || input["expected_revision"] != m["digest"]) {
		return nil, nil, fmt.Errorf("inventory changed expected revision")
	}
	entries, ok := m["entries"].([]any)
	if !ok || len(entries) > 128 {
		return nil, nil, fail
	}
	refs := map[string]map[string]any{}
	summaries := map[string]map[string]any{}
	previous := ""
	for _, item := range entries {
		e, ok := item.(map[string]any)
		if !ok || !shvFields(e, "operation_id", "intent_digest", "snapshot_digest", "state", "graph_digest", "connector", "counts", "projection_digest") {
			return nil, nil, fail
		}
		id, ok := e["operation_id"].(string)
		if !ok || !graphIndexOperation.MatchString(id) || (previous != "" && id <= previous) {
			return nil, nil, fail
		}
		previous = id
		for _, key := range []string{"intent_digest", "snapshot_digest", "graph_digest", "projection_digest"} {
			if !indexDigest(e[key]) {
				return nil, nil, fail
			}
		}
		if e["state"] != "prepared" && e["state"] != "committed" {
			return nil, nil, fail
		}
		writer := shvText(shvMap(e["connector"])["Version"])
		if (writer != SHVStoreVersion && writer != SHVStoreInventoryVersion) || storeInstallationVersion(e["connector"], writer) != nil {
			return nil, nil, fail
		}
		counts, ok := e["counts"].(map[string]any)
		if !ok || !shvFields(counts, "nodes", "edges") {
			return nil, nil, fail
		}
		for key, max := range map[string]int64{"nodes": 1024, "edges": 2048} {
			if _, ok := indexCount(counts[key], max); !ok {
				return nil, nil, fail
			}
		}
		sd := e["snapshot_digest"].(string)
		if prior, ok := summaries[sd]; ok {
			for _, key := range []string{"graph_digest", "connector", "counts", "projection_digest"} {
				if !scvEqual(prior[key], e[key]) {
					return nil, nil, fmt.Errorf("shared snapshot summary differs")
				}
			}
		} else {
			summaries[sd] = e
			refs[sd] = map[string]any{"snapshot_digest": sd, "operation_ids": []string{}, "committed_operations": 0}
		}
		ref := refs[sd]
		ref["operation_ids"] = append(ref["operation_ids"].([]string), id)
		if e["state"] == "committed" {
			ref["committed_operations"] = ref["committed_operations"].(int) + 1
		}
	}
	keys := []string{}
	for key := range refs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	expected := []any{}
	published := int64(0)
	for _, key := range keys {
		expected = append(expected, refs[key])
		if refs[key]["committed_operations"].(int) > 0 {
			published++
		}
	}
	if !scvEqual(expected, m["snapshots"]) || !scvEqual(m["capacity"], map[string]int{"intents": 128, "snapshots": 128}) {
		return nil, nil, fail
	}
	counts, ok := m["global_counts"].(map[string]any)
	if !ok || !shvFields(counts, "intents", "snapshots") {
		return nil, nil, fail
	}
	ni, oi := indexCount(counts["intents"], 128)
	ns, os := indexCount(counts["snapshots"], 128)
	if !oi || !os || ni < int64(len(entries)) || ns < published || ns > ni {
		return nil, nil, fail
	}
	return m, entries, nil
}
func validateStoreInventoryRecord(v any, manifest map[string]any, entry any) error {
	r, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("inventory record must be status")
	}
	e := entry.(map[string]any)
	payload, _ := SCVCanonical(map[string]any{"tops_id": manifest["tops_id"], "namespace": manifest["namespace"], "operation_id": e["operation_id"]})
	raw, err := SCVCanonical(r)
	if err != nil {
		return err
	}
	if err = ValidateSHVStoreResultVersion("status", payload, raw, shvText(shvMap(e["connector"])["Version"])); err != nil {
		return err
	}
	if !scvEqual(storeInventorySummary(r), e) {
		return fmt.Errorf("inventory record changed manifest summary")
	}
	return nil
}
func validateSHVStoreInventory(operation string, input, result map[string]any) error {
	fail := fmt.Errorf("invalid graph index maintenance result")
	if !storeScope(input) || result["backend"] != "duckdb" || !scvEqual(result["input"], input) || scvSeal(result, "digest") != nil {
		return fail
	}
	m, entries, err := validateStoreInventoryManifest(result["manifest"], input)
	if err != nil {
		return err
	}
	if operation == "inventory" {
		if !shvFields(input, "tops_id", "namespace", "expected_revision", "cursor", "limit") || !shvFields(result, "protocol", "backend", "input", "manifest", "records", "next_cursor", "physical_bytes", "digest") || result["protocol"] != "symphony.shv.graph-store-inventory.v1" {
			return fail
		}
		limit, ok := indexCount(input["limit"], 16)
		if !ok || limit < 1 {
			return fail
		}
		start := 0
		if input["cursor"] != nil {
			c, ok := input["cursor"].(map[string]any)
			if !ok || !shvFields(c, "revision", "after_operation_id") || c["revision"] != m["digest"] {
				return fail
			}
			found := false
			for i, e := range entries {
				if c["after_operation_id"] == e.(map[string]any)["operation_id"] {
					start = i + 1
					found = true
					break
				}
			}
			if !found {
				return fail
			}
		}
		end := start + int(limit)
		if end > len(entries) {
			end = len(entries)
		}
		rows, ok := result["records"].([]any)
		if !ok || len(rows) != end-start {
			return fail
		}
		for i, row := range rows {
			if err = validateStoreInventoryRecord(row, m, entries[start+i]); err != nil {
				return err
			}
		}
		var next any
		if end < len(entries) {
			next = map[string]any{"revision": m["digest"], "after_operation_id": entries[end-1].(map[string]any)["operation_id"]}
		}
		if !scvEqual(result["next_cursor"], next) {
			return fail
		}
		physical, ok := result["physical_bytes"].(map[string]any)
		if !ok || !shvFields(physical, "database", "wal") {
			return fail
		}
		for _, key := range []string{"database", "wal"} {
			if _, ok := indexCount(physical[key], 1<<53-1); !ok {
				return fail
			}
		}
		return nil
	}
	return fail
}

func shvStoreInventoryInput(p map[string]any) error {
	if !storeScope(p) || !shvFields(p, "tops_id", "namespace", "expected_revision", "cursor", "limit") {
		return shvFail()
	}
	if p["expected_revision"] != nil && !storeDigest(p["expected_revision"]) {
		return shvFail()
	}
	if _, e := storeInteger(p["limit"], 1, 16); e != nil {
		return e
	}
	if p["cursor"] != nil {
		c := shvMap(p["cursor"])
		if !shvFields(c, "revision", "after_operation_id") || !storeDigest(c["revision"]) || !graphIndexOperation.MatchString(shvText(c["after_operation_id"])) {
			return shvFail()
		}
	}
	return nil
}

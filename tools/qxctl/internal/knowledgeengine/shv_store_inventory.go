package knowledgeengine

import (
	"fmt"
	"path/filepath"
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
		if (writer != SHVStoreVersion && writer != SHVStoreInventoryVersion && writer != SHVStoreTransferVersion) || storeInstallationVersion(e["connector"], writer) != nil {
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
	if operation != "transfer_plan" || !shvFields(input, "tops_id", "namespace", "expected_revision", "operation_ids", "source_connector", "target_connector", "target_root", "capacity") || !indexDigest(input["expected_revision"]) || !shvFields(result, "protocol", "backend", "input", "manifest", "selected", "excluded_operation_ids", "requirements", "blockers", "disposition", "digest") || result["protocol"] != "symphony.shv.graph-store-transfer-plan.v1" {
		return fail
	}
	err = shvStoreTransferInput(input)
	if err != nil {
		return fail
	}
	root, ok := graphIndexText(input["target_root"], 4096)
	if !ok || !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return fail
	}
	capacity, ok := input["capacity"].(map[string]any)
	if !ok || !shvFields(capacity, "intents", "snapshots") {
		return fail
	}
	ci, oi := indexCount(capacity["intents"], 128)
	cs, os := indexCount(capacity["snapshots"], 128)
	if !oi || !os {
		return fail
	}
	ids, ok := input["operation_ids"].([]any)
	if !ok || len(ids) == 0 || len(ids) > 16 {
		return fail
	}
	selected, ok := result["selected"].([]any)
	if !ok || len(selected) != len(ids) {
		return fail
	}
	byID := map[string]any{}
	for _, entry := range entries {
		byID[entry.(map[string]any)["operation_id"].(string)] = entry
	}
	seen := map[string]bool{}
	published := map[string]bool{}
	for i, item := range ids {
		id, ok := item.(string)
		if !ok || seen[id] || byID[id] == nil {
			return fail
		}
		seen[id] = true
		row, ok := selected[i].(map[string]any)
		if !ok || !shvFields(row, "source", "target_snapshot_digest", "target_intent_digest") {
			return fail
		}
		if err = validateStoreInventoryRecord(row["source"], m, byID[id]); err != nil {
			return err
		}
		raw, _ := SCVCanonical(row["source"])
		status, _ := scvObject(raw)
		intent := status["intent"].(map[string]any)
		snapshot := intent["snapshot"].(map[string]any)
		delete(snapshot, "digest")
		snapshot["connector"] = input["target_connector"]
		sd, _ := SCVDigest(snapshot)
		snapshot["digest"] = sd
		delete(intent, "digest")
		td, _ := SCVDigest(intent)
		if row["target_snapshot_digest"] != sd || row["target_intent_digest"] != td {
			return fmt.Errorf("transfer changed target lineage")
		}
		if status["state"] == "committed" {
			published[sd] = true
		}
	}
	excluded := []string{}
	for _, e := range entries {
		id := e.(map[string]any)["operation_id"].(string)
		if !seen[id] {
			excluded = append(excluded, id)
		}
	}
	blockers := []string{}
	if int64(len(ids)) > ci {
		blockers = append(blockers, "intent_capacity")
	}
	if int64(len(published)) > cs {
		blockers = append(blockers, "snapshot_capacity")
	}
	disposition := "ready"
	if len(blockers) > 0 {
		disposition = "blocked"
	}
	if !scvEqual(result["excluded_operation_ids"], excluded) || !scvEqual(result["requirements"], map[string]int{"intents": len(ids), "snapshots": len(published)}) || !scvEqual(result["blockers"], blockers) || result["disposition"] != disposition {
		return fail
	}
	return nil
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

func shvStoreTransferInput(p map[string]any) error {
	if !storeScope(p) || !shvFields(p, "tops_id", "namespace", "expected_revision", "operation_ids", "source_connector", "target_connector", "target_root", "capacity") || !storeDigest(p["expected_revision"]) {
		return shvFail()
	}
	if storeInstallationVersion(p["source_connector"], SHVStoreTransferVersion) != nil {
		return shvFail()
	}
	v := shvText(shvMap(p["target_connector"])["Version"])
	if (v != SHVStoreVersion && v != SHVStoreInventoryVersion && v != SHVStoreTransferVersion) || storeInstallationVersion(p["target_connector"], v) != nil {
		return shvFail()
	}
	root := shvText(p["target_root"])
	if root == "/" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return shvFail()
	}
	c := shvMap(p["capacity"])
	if !shvFields(c, "intents", "snapshots") {
		return shvFail()
	}
	for _, k := range []string{"intents", "snapshots"} {
		if _, e := storeInteger(c[k], 0, 128); e != nil {
			return e
		}
	}
	ids := shvList(p["operation_ids"])
	if len(ids) < 1 || len(ids) > 16 {
		return shvFail()
	}
	seen := map[string]bool{}
	for _, v := range ids {
		id := shvText(v)
		if !graphIndexOperation.MatchString(id) || seen[id] {
			return shvFail()
		}
		seen[id] = true
	}
	return nil
}

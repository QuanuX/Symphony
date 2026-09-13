package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
)

func indexCount(v any, max int64) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, e := n.Int64()
	return i, e == nil && i >= 0 && i <= max
}
func indexDigest(v any) bool { s, ok := v.(string); return ok && taggedSHA256(s) }
func indexSummary(status map[string]any) map[string]any {
	intent := status["intent"].(map[string]any)
	snapshot := intent["snapshot"].(map[string]any)
	return map[string]any{"operation_id": intent["operation_id"], "intent_digest": intent["digest"], "snapshot_digest": snapshot["digest"], "state": status["state"], "graph_digest": snapshot["graph"].(map[string]any)["digest"], "owner": snapshot["owner"], "connector": snapshot["connector"], "validation_query_time": intent["validation_query_time"], "counts": status["counts"], "projection_digest": status["projection_digest"]}
}
func validateIndexManifest(value any, input map[string]any) (map[string]any, []any, error) {
	fail := fmt.Errorf("invalid graph index inventory manifest")
	m, ok := value.(map[string]any)
	if !ok || !graphIndexFields(m, "protocol", "tops_id", "namespace", "entries", "snapshots", "global_revision", "global_counts", "capacity", "digest") || m["protocol"] != "symphony.scv.graph-index-inventory-manifest.v1" || !graphIndexScope(m) || m["tops_id"] != input["tops_id"] || m["namespace"] != input["namespace"] || scvSeal(m, "digest") != nil || !indexDigest(m["global_revision"]) {
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
		if !ok || !graphIndexFields(e, "operation_id", "intent_digest", "snapshot_digest", "state", "graph_digest", "owner", "connector", "validation_query_time", "counts", "projection_digest") {
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
		if !graphIndexTime(e["validation_query_time"]) {
			return nil, nil, fail
		}
		if _, err := graphIndexInstallation(e["owner"], false); err != nil {
			return nil, nil, err
		}
		if _, err := graphIndexInstallation(e["connector"], true); err != nil {
			return nil, nil, err
		}
		counts, ok := e["counts"].(map[string]any)
		if !ok || !graphIndexFields(counts, "claims", "nodes", "edges") {
			return nil, nil, fail
		}
		for key, max := range map[string]int64{"claims": 128, "nodes": 512, "edges": 1024} {
			if _, ok := indexCount(counts[key], max); !ok {
				return nil, nil, fail
			}
		}
		sd := e["snapshot_digest"].(string)
		if prior, ok := summaries[sd]; ok {
			for _, key := range []string{"graph_digest", "owner", "connector", "counts", "projection_digest"} {
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
	if !ok || !graphIndexFields(counts, "intents", "snapshots") {
		return nil, nil, fail
	}
	ni, oi := indexCount(counts["intents"], 128)
	ns, os := indexCount(counts["snapshots"], 128)
	if !oi || !os || ni < int64(len(entries)) || ns < published || ns > ni {
		return nil, nil, fail
	}
	return m, entries, nil
}
func validateIndexRecord(v any, manifest map[string]any, entry any) error {
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
	if err = ValidateSCVGraphIndexResult("status", payload, raw); err != nil {
		return err
	}
	if !scvEqual(indexSummary(r), e) {
		return fmt.Errorf("inventory record changed manifest summary")
	}
	return nil
}
func validateGraphIndexMaintenance(operation string, input, result map[string]any) error {
	fail := fmt.Errorf("invalid graph index maintenance result")
	if !graphIndexScope(input) || result["backend"] != "duckdb" || !scvEqual(result["input"], input) || scvSeal(result, "digest") != nil {
		return fail
	}
	m, entries, err := validateIndexManifest(result["manifest"], input)
	if err != nil {
		return err
	}
	if operation == "inventory" {
		if !graphIndexFields(input, "tops_id", "namespace", "expected_revision", "cursor", "limit") || !graphIndexFields(result, "protocol", "backend", "input", "manifest", "records", "next_cursor", "physical_bytes", "digest") || result["protocol"] != "symphony.scv.graph-index-inventory.v1" {
			return fail
		}
		limit, ok := indexCount(input["limit"], 16)
		if !ok || limit < 1 {
			return fail
		}
		start := 0
		if input["cursor"] != nil {
			c, ok := input["cursor"].(map[string]any)
			if !ok || !graphIndexFields(c, "revision", "after_operation_id") || c["revision"] != m["digest"] {
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
			if err = validateIndexRecord(row, m, entries[start+i]); err != nil {
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
		if !ok || !graphIndexFields(physical, "database", "wal") {
			return fail
		}
		for _, key := range []string{"database", "wal"} {
			if _, ok := indexCount(physical[key], 1<<53-1); !ok {
				return fail
			}
		}
		return nil
	}
	if operation != "transfer_plan" || !graphIndexFields(input, "tops_id", "namespace", "expected_revision", "operation_ids", "source_connector", "target_connector", "target_root", "capacity") || !indexDigest(input["expected_revision"]) || !graphIndexFields(result, "protocol", "backend", "input", "manifest", "selected", "excluded_operation_ids", "requirements", "blockers", "disposition", "digest") || result["protocol"] != "symphony.scv.graph-index-transfer-plan.v1" {
		return fail
	}
	source, err := graphIndexInstallation(input["source_connector"], true)
	if err != nil || source.Version != SCVGraphIndexPlanningVersion {
		return fail
	}
	if _, err = graphIndexInstallation(input["target_connector"], true); err != nil {
		return err
	}
	root, ok := graphIndexText(input["target_root"], 4096)
	if !ok || !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return fail
	}
	capacity, ok := input["capacity"].(map[string]any)
	if !ok || !graphIndexFields(capacity, "intents", "snapshots") {
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
		if !ok || !graphIndexFields(row, "source", "target_snapshot_digest", "target_intent_digest") {
			return fail
		}
		if err = validateIndexRecord(row["source"], m, byID[id]); err != nil {
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

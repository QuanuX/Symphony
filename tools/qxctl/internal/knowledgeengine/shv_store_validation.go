package knowledgeengine

import (
	"encoding/json"
	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"path/filepath"
	"regexp"
)

func storeScope(p map[string]any) bool {
	return stavprotocol.ValidateTOPSID(shvText(p["tops_id"])) == nil && graphIndexNamespace.MatchString(shvText(p["namespace"]))
}
func storeInstallation(v any) error { return storeInstallationVersion(v, SHVStoreVersion) }
func storeInstallationVersion(v any, version string) error {
	m := shvMap(v)
	if !shvFields(m, "Role", "ModuleID", "EngineID", "Version", "Prefix", "ReceiptPath", "ReceiptDigest", "ReceiptProtocol", "ExecutablePath", "ExecutableDigest") {
		return shvFail()
	}
	prefix := shvText(m["Prefix"])
	mod := shvStoreSpec.moduleID
	if prefix == "/" || !filepath.IsAbs(prefix) || filepath.Clean(prefix) != prefix || m["Role"] != mod || m["ModuleID"] != mod || m["EngineID"] != shvStoreSpec.engineID || m["Version"] != version || m["ReceiptProtocol"] != receiptProtocolV2 || m["ReceiptPath"] != prefix+"/share/symphony/receipts/"+mod+"/"+version+"/install-receipt.json" || m["ExecutablePath"] != prefix+"/libexec/symphony/"+mod+"/"+version+"/"+shvStoreSpec.engineID || !storeDigest(m["ReceiptDigest"]) || !storeDigest(m["ExecutableDigest"]) {
		return shvFail()
	}
	return nil
}
func shvStoreInput(op string, p map[string]any) error {
	return shvStoreInputVersion(op, p, SHVStoreVersion)
}
func shvStoreInputVersion(op string, p map[string]any, version string) error {
	if op == "inventory" {
		if version != SHVStoreInventoryVersion {
			return shvFail()
		}
		return shvStoreInventoryInput(p)
	}
	if op == "inspect" {
		if !shvFields(p) {
			return shvFail()
		}
		return nil
	}
	if !storeScope(p) {
		return shvFail()
	}
	switch op {
	case "prepare":
		if !shvFields(p, "tops_id", "namespace", "operation_id", "graph", "connector") || !graphIndexOperation.MatchString(shvText(p["operation_id"])) || storeInstallationVersion(p["connector"], version) != nil {
			return shvFail()
		}
		return shvGraph(shvMap(p["graph"]))
	case "commit":
		if !shvFields(p, "tops_id", "namespace", "operation_id", "expected_intent_digest") || !graphIndexOperation.MatchString(shvText(p["operation_id"])) || !storeDigest(p["expected_intent_digest"]) {
			return shvFail()
		}
	case "status":
		if !shvFields(p, "tops_id", "namespace", "operation_id") || !graphIndexOperation.MatchString(shvText(p["operation_id"])) {
			return shvFail()
		}
	case "export":
		if !shvFields(p, "tops_id", "namespace", "snapshot_digest") || !storeDigest(p["snapshot_digest"]) {
			return shvFail()
		}
	case "query":
		if !shvFields(p, "tops_id", "namespace", "snapshot_digest", "kind", "filters", "cursor", "limit") || !storeDigest(p["snapshot_digest"]) || (p["kind"] != "nodes" && p["kind"] != "edges") {
			return shvFail()
		}
		n, e := storeInteger(p["limit"], 1, 128)
		if e != nil || n < 1 {
			return shvFail()
		}
		f := shvMap(p["filters"])
		if f == nil {
			return shvFail()
		}
		for k, v := range f {
			if k != "id" && (p["kind"] != "edges" || (k != "from" && k != "to" && k != "label")) {
				return shvFail()
			}
			if !shvBoundedText(v, 65536) {
				return shvFail()
			}
		}
		if p["cursor"] != nil {
			c := shvMap(p["cursor"])
			if !shvFields(c, "query_digest", "after_key") || !storeDigest(c["query_digest"]) || !shvBoundedText(c["after_key"], 512) {
				return shvFail()
			}
		}
	default:
		return shvFail()
	}
	return nil
}
func storeSnapshot(s map[string]any) error { return storeSnapshotVersion(s, SHVStoreVersion) }
func storeSnapshotVersion(s map[string]any, version string) error {
	if !shvFields(s, "protocol", "backend", "mapping_version", "tops_id", "namespace", "graph", "connector", "digest") || s["protocol"] != "symphony.shv.graph-store-snapshot.v1" || s["backend"] != "duckdb" || s["mapping_version"] != "1" || !storeScope(s) || storeInstallationVersion(s["connector"], version) != nil || shvSealed(s) != nil {
		return shvFail()
	}
	return shvGraph(shvMap(s["graph"]))
}
func storeProjection(s map[string]any) (map[string]any, map[string]any) {
	g := shvMap(s["graph"])
	projection := map[string]any{}
	counts := map[string]any{}
	for _, kind := range []string{"nodes", "edges"} {
		rows := []any{}
		for _, v := range shvList(g[kind]) {
			r := shvMap(v)
			rows = append(rows, map[string]any{"key": r["id"], "value": v})
		}
		projection[kind] = rows
		counts[kind] = len(rows)
	}
	return projection, counts
}
func ValidateSHVStoreResult(op string, input, result []byte) error {
	return ValidateSHVStoreResultVersion(op, input, result, SHVStoreVersion)
}
func ValidateSHVStoreResultVersion(op string, input, result []byte, version string) error {
	if version != SHVStoreVersion && version != SHVStoreInventoryVersion {
		return shvFail()
	}
	p, e := shvObject(input)
	if e != nil {
		return e
	}
	if e = shvStoreInputVersion(op, p, version); e != nil {
		return e
	}
	r, e := shvObject(result)
	if e != nil {
		return e
	}
	if op == "inspect" {
		return shvStoreDescriptor(p, r, version)
	}
	if op == "inventory" {
		return validateSHVStoreInventory(op, p, r)
	}
	if shvSealed(r) != nil {
		return shvFail()
	}
	protocol, ok := SHVStoreResultProtocol(op)
	if !ok || r["protocol"] != protocol || r["backend"] != "duckdb" {
		return shvFail()
	}
	s := shvMap(r["snapshot"])
	if op == "prepare" || op == "commit" || op == "status" {
		if !shvFields(r, "protocol", "backend", "intent", "state", "snapshot_digest", "projection_digest", "counts", "index_verified", "digest") {
			return shvFail()
		}
		i := shvMap(r["intent"])
		s = shvMap(i["snapshot"])
		if !shvFields(i, "protocol", "operation_id", "snapshot", "digest") || i["protocol"] != "symphony.shv.graph-store-intent.v1" || shvSealed(i) != nil || i["operation_id"] != p["operation_id"] || r["snapshot_digest"] != s["digest"] || (r["state"] != "prepared" && r["state"] != "committed") || r["index_verified"] != (r["state"] == "committed") {
			return shvFail()
		}
		if op == "commit" && (p["expected_intent_digest"] != i["digest"] || r["state"] != "committed") {
			return shvFail()
		}
		if op == "prepare" && (!scvEqual(p["graph"], s["graph"]) || !scvEqual(p["connector"], s["connector"])) {
			return shvFail()
		}
	} else if op == "export" {
		if !shvFields(r, "protocol", "backend", "snapshot", "projection_digest", "counts", "digest") {
			return shvFail()
		}
	} else {
		if !shvFields(r, "protocol", "backend", "snapshot", "projection_digest", "counts", "input", "rows", "matched_count", "next_cursor", "digest") {
			return shvFail()
		}
	}
	if storeSnapshotVersion(s, version) != nil || s["tops_id"] != p["tops_id"] || s["namespace"] != p["namespace"] {
		return shvFail()
	}
	if (op == "export" || op == "query") && s["digest"] != p["snapshot_digest"] {
		return shvFail()
	}
	projected, counts := storeProjection(s)
	pd, e := SCVDigest(projected)
	if e != nil || r["projection_digest"] != pd || !scvEqual(r["counts"], counts) {
		return shvFail()
	}
	if op == "query" {
		if !scvEqual(r["input"], p) {
			return shvFail()
		}
		identity := map[string]any{}
		for k, v := range p {
			if k != "cursor" && k != "limit" {
				identity[k] = v
			}
		}
		qd, _ := SCVDigest(identity)
		matches := []any{}
		for _, v := range shvList(projected[shvText(p["kind"])]) {
			row := shvMap(shvMap(v)["value"])
			match := true
			for k, x := range shvMap(p["filters"]) {
				if !scvEqual(row[k], x) {
					match = false
				}
			}
			if match {
				matches = append(matches, v)
			}
		}
		offset := 0
		if p["cursor"] != nil {
			c := shvMap(p["cursor"])
			if c["query_digest"] != qd {
				return shvFail()
			}
			found := false
			for j, v := range matches {
				if shvMap(v)["key"] == c["after_key"] {
					offset = j + 1
					found = true
				}
			}
			if !found {
				return shvFail()
			}
		}
		limit, _ := storeInteger(p["limit"], 1, 128)
		end := min(offset+int(limit), len(matches))
		var next any
		if end < len(matches) {
			next = map[string]any{"query_digest": qd, "after_key": shvMap(matches[end-1])["key"]}
		}
		if !scvEqual(r["rows"], matches[offset:end]) || !scvEqual(r["matched_count"], len(matches)) || !scvEqual(r["next_cursor"], next) {
			return shvFail()
		}
	}
	return nil
}
func storeDigest(v any) bool { return storeDigestPattern.MatchString(shvText(v)) }
func storeInteger(v any, lo, hi int64) (int64, error) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, shvFail()
	}
	x, e := n.Int64()
	if e != nil || x < lo || x > hi {
		return 0, shvFail()
	}
	return x, nil
}

var storeDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

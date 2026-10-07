package knowledgeengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

const sbvBundleOrder = "canonical_immediate_children.v1"
const sbvBundleAccess = "all_admitted_fields.v1"

// The native wrapper reserves the same existing control-frame headroom before
// writes. This is a reply-capacity check, never a bundle/data-size policy.
func sbvBundleControlPreflight(op string, request map[string]any) error {
	if !strings.HasPrefix(op, "bundle_") {
		return nil
	}
	extensions, ok := request["extensions"].(map[string]any)
	if !ok || !sbvBundlePortable(extensions) {
		return fmt.Errorf("SBV bundle portable extensions required")
	}
	var count func(any) int
	count = func(v any) int {
		n := 1
		switch x := v.(type) {
		case map[string]any:
			for _, child := range x {
				n += 1 + count(child)
			}
		case []any:
			for _, child := range x {
				n += count(child)
			}
		}
		return n
	}
	if count(extensions)+512 > maxJSONValues {
		return fmt.Errorf("SBV bundle extensions leave insufficient response metadata capacity")
	}
	return nil
}

func sbvBundleUint(v any) (uint64, bool) {
	s, ok := v.(string)
	n, err := strconv.ParseUint(s, 10, 64)
	return n, ok && err == nil && strconv.FormatUint(n, 10) == s
}

func sbvBundlePortable(v any) bool {
	switch x := v.(type) {
	case nil, string, bool:
		return true
	case []any:
		for _, child := range x {
			if !sbvBundlePortable(child) {
				return false
			}
		}
	case map[string]any:
		for _, child := range x {
			if !sbvBundlePortable(child) {
				return false
			}
		}
	default:
		return false
	}
	return true
}

func sbvBundleReference(v any) bool {
	m, ok := v.(map[string]any)
	return ok && sbvRecoveryFields(m, "manifest_path", "manifest_sha256", "content_sha256") &&
		sbvRecoveryPath(m["manifest_path"]) && sbvRecoverySHA(m["manifest_sha256"]) && sbvRecoverySHA(m["content_sha256"])
}

func sbvBundleSelector(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	if m["kind"] == "node_id" {
		_, ok = sbvBundleUint(m["node_id"])
		return ok && sbvRecoveryFields(m, "kind", "node_id")
	}
	p, ok := m["pointer"].(string)
	if !ok || m["kind"] != "pointer" || !sbvRecoveryFields(m, "kind", "pointer") || (p != "" && !strings.HasPrefix(p, "/")) {
		return false
	}
	for i := 0; i < len(p); i++ {
		if p[i] == '~' {
			if i+1 >= len(p) || (p[i+1] != '0' && p[i+1] != '1') {
				return false
			}
			i++
		}
	}
	return true
}

func sbvBundleCursor(v, reference, selector any, node any, offset uint64) bool {
	m, ok := v.(map[string]any)
	if !ok || !sbvRecoveryFields(m, "protocol", "reference", "selector", "selected_node_id", "order", "access_profile", "next_offset") ||
		m["protocol"] != "symphony.sbv.bundle-cursor.v1" || !reflect.DeepEqual(m["reference"], reference) ||
		!reflect.DeepEqual(m["selector"], selector) || m["selected_node_id"] != node ||
		m["order"] != sbvBundleOrder || m["access_profile"] != sbvBundleAccess {
		return false
	}
	n, ok := sbvBundleUint(m["next_offset"])
	return ok && n > 0 && n == offset
}

func sbvBundleCounts(m map[string]any) bool {
	bytes, b := sbvBundleUint(m["logical_body_bytes"])
	nodes, n := sbvBundleUint(m["logical_body_nodes"])
	values, v := sbvBundleUint(m["logical_body_values"])
	exported, e := sbvBundleUint(m["exported_value_nodes"])
	return b && n && v && e && bytes > 0 && values >= 5 && nodes >= values && nodes <= ^uint64(0)-2 &&
		values < ^uint64(0) && exported == values+1
}

func sbvBundleStats(v any) bool {
	m, ok := v.(map[string]any)
	if !ok || !sbvRecoveryFields(m, "files_read", "bytes_read", "cache_hits", "maximum_file_bytes") {
		return false
	}
	for _, value := range m {
		if _, ok := sbvBundleUint(value); !ok {
			return false
		}
	}
	return true
}

func sbvBundleWriteOptions(v any) bool {
	m, ok := v.(map[string]any)
	if !ok || !sbvRecoveryFields(m, "page_bytes", "index_fanout") {
		return false
	}
	bytes, b := sbvBundleUint(m["page_bytes"])
	fanout, f := sbvBundleUint(m["index_fanout"])
	return b && f && bytes >= 512 && fanout >= 2
}

func sbvBundleSuffix(format, suffix any, reference any, expectedNodes any) bool {
	s, ok := suffix.(string)
	if !ok || len(s) == 0 || len(s) > maxStringBytes {
		return false
	}
	if format == "json" {
		return s == "}\n"
	}
	if format != "ndjson" || !strings.HasSuffix(s, "\n") {
		return false
	}
	m, err := sqavObject([]byte(s), maxStringBytes)
	r, ok := reference.(map[string]any)
	if err != nil || !ok || !sbvRecoveryFields(m, "event", "status", "nodes", "content_sha256", "manifest_sha256") ||
		m["event"] != "end" || m["status"] != "complete" || m["content_sha256"] != r["content_sha256"] ||
		m["manifest_sha256"] != r["manifest_sha256"] {
		return false
	}
	if _, ok := sbvBundleUint(m["nodes"]); !ok || (expectedNodes != nil && m["nodes"] != expectedNodes) {
		return false
	}
	canonical, err := sbvNativeCanonical(m)
	return err == nil && string(canonical)+"\n" == s
}

// ValidateSBVBundleResult checks a bounded operation result and exact request
// correspondence. Native traversal remains responsible for the bundle's
// mathematical/structural identity; a quick query never claims full closure.
func ValidateSBVBundleResult(op string, request map[string]any, raw []byte) error {
	bad := func() error { return fmt.Errorf("SBV bundle result contract mismatch") }
	m, err := sqavObject(raw, maxResponseBytes)
	if err != nil || !sbvBundlePortable(m) || m["protocol"] != "symphony.sbv."+strings.ReplaceAll(op, "_", "-")+".v1" {
		return bad()
	}
	if m["status"] == "recovery_required" {
		if !sbvBundleRecovery(op, request, m) {
			return bad()
		}
		return nil
	}
	common := []string{"protocol", "status", "source_authorship", "verification_extent", "reference", "extensions"}
	if m["status"] != "complete" || m["source_authorship"] != "not_verified" ||
		!sbvBundleReference(m["reference"]) || !reflect.DeepEqual(m["extensions"], request["extensions"]) {
		return bad()
	}
	if _, ok := m["extensions"].(map[string]any); !ok {
		return bad()
	}
	if op != "bundle_import" && !reflect.DeepEqual(m["reference"], request["reference"]) {
		return bad()
	}
	if op == "bundle_query" {
		fields := append(common, "selector", "selected_node_id", "selected_kind", "order", "access_profile", "offset", "total", "complete", "next_cursor", "nodes", "io_stats")
		budget, ok := sbvBundleUint(request["byte_limit"])
		if !ok || uint64(len(raw)) > budget || !sbvRecoveryFields(m, fields...) ||
			m["verification_extent"] != "accessed_pages" || !sbvBundleStats(m["io_stats"]) || !sbvBundleQuery(request, m) {
			return bad()
		}
		return nil
	}
	common = append(common, "logical_body_bytes", "logical_body_nodes", "logical_body_values", "exported_value_nodes")
	if !sbvBundleCounts(m) {
		return bad()
	}
	switch op {
	case "bundle_import":
		fields := append(common, "input_path", "input_file_sha256", "input_bytes", "bundle_path", "manifest_bytes", "page_files_created", "page_bytes_created", "write_options", "io_stats")
		if !sbvRecoveryFields(m, fields...) || m["verification_extent"] != "full_logical_closure" ||
			!sbvBundleStats(m["io_stats"]) ||
			!sbvRecoveryPath(m["input_path"]) || m["input_path"] != request["input_path"] ||
			!sbvRecoveryPath(m["bundle_path"]) || m["bundle_path"] != request["bundle_path"] ||
			m["input_file_sha256"] != request["expected_file_sha256"] || !sbvRecoverySHA(m["input_file_sha256"]) ||
			!sbvBundleWriteOptions(m["write_options"]) || !reflect.DeepEqual(m["write_options"], request["write_options"]) {
			return bad()
		}
		for _, key := range []string{"input_bytes", "manifest_bytes"} {
			if !sbvRecoveryPositive(m[key]) {
				return bad()
			}
		}
		for _, key := range []string{"page_files_created", "page_bytes_created"} {
			if _, ok := sbvBundleUint(m[key]); !ok {
				return bad()
			}
		}
		if m["reference"].(map[string]any)["manifest_path"] != filepath.Join(m["bundle_path"].(string), "manifest.json") {
			return bad()
		}
	case "bundle_inspect":
		fields := append(common, "logical_protocol", "storage_protocol", "canonicalization", "root_node_id", "root_children", "write_options", "io_stats")
		if !sbvRecoveryFields(m, fields...) || m["verification_extent"] != "manifest_only" ||
			m["logical_protocol"] != "symphony.sbv.result.v1" || m["storage_protocol"] != "symphony.sbv.partitioned-result.v1" ||
			m["canonicalization"] != "nlohmann_compact_utf8_sorted_keys_no_numeric_literals.v1" ||
			m["root_node_id"] != "0" || m["root_children"] != "5" || !sbvBundleWriteOptions(m["write_options"]) || !sbvBundleStats(m["io_stats"]) {
			return bad()
		}
	case "bundle_verify":
		if !sbvRecoveryFields(m, append(common, "io_stats")...) || m["verification_extent"] != "full_logical_closure" || !sbvBundleStats(m["io_stats"]) {
			return bad()
		}
	case "bundle_export":
		fields := append(common, "output_path", "format", "bytes", "file_sha256", "completion_suffix", "io_stats")
		if !sbvRecoveryFields(m, fields...) || m["verification_extent"] != "full_logical_closure" ||
			!sbvRecoveryPath(m["output_path"]) || m["output_path"] != request["output_path"] || m["format"] != request["format"] ||
			!sbvRecoveryPositive(m["bytes"]) || !sbvRecoverySHA(m["file_sha256"]) ||
			!sbvBundleSuffix(m["format"], m["completion_suffix"], m["reference"], m["exported_value_nodes"]) || !sbvBundleStats(m["io_stats"]) {
			return bad()
		}
	default:
		return bad()
	}
	return nil
}

func sbvBundleQuery(request, m map[string]any) bool {
	selected, s := sbvBundleUint(m["selected_node_id"])
	offset, o := sbvBundleUint(m["offset"])
	total, t := sbvBundleUint(m["total"])
	limit, l := sbvBundleUint(request["row_limit"])
	complete, c := m["complete"].(bool)
	rows, r := m["nodes"].([]any)
	if !s || !o || !t || !l || !c || !r || limit == 0 || offset > total ||
		uint64(len(rows)) > limit || uint64(len(rows)) > total-offset ||
		!sbvBundleSelector(m["selector"]) || !reflect.DeepEqual(m["selector"], request["selector"]) ||
		m["order"] != sbvBundleOrder || m["access_profile"] != sbvBundleAccess {
		return false
	}
	selector := m["selector"].(map[string]any)
	if selector["kind"] == "node_id" && selector["node_id"] != m["selected_node_id"] {
		return false
	}
	if request["cursor"] == nil {
		if offset != 0 {
			return false
		}
	} else if !sbvBundleCursor(request["cursor"], m["reference"], m["selector"], m["selected_node_id"], offset) {
		return false
	}
	next := offset + uint64(len(rows))
	if complete != (next == total) || (complete && m["next_cursor"] != nil) ||
		(!complete && (len(rows) == 0 || !sbvBundleCursor(m["next_cursor"], m["reference"], m["selector"], m["selected_node_id"], next))) {
		return false
	}
	kind, ok := m["selected_kind"].(string)
	if !ok || (kind != "object" && kind != "array" && kind != "scalar") ||
		(kind == "scalar" && total != 1) || (selected == 0 && (kind != "object" || total != 5)) {
		return false
	}
	var previous uint64
	var previousEdge string
	for i, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok || !sbvRecoveryFields(row, "node_id", "parent_id", "edge", "kind", "value", "children") {
			return false
		}
		id, idOK := sbvBundleUint(row["node_id"])
		children, childOK := sbvBundleUint(row["children"])
		if !idOK || !childOK || (i > 0 && id <= previous) {
			return false
		}
		previous = id
		if kind == "scalar" && row["kind"] != "scalar" {
			return false
		}
		if kind != "scalar" && row["parent_id"] != m["selected_node_id"] {
			return false
		}
		if id == 0 {
			if row["parent_id"] != nil || row["edge"] != nil {
				return false
			}
		} else {
			parent, parentOK := sbvBundleUint(row["parent_id"])
			_, edgeOK := row["edge"].(string)
			if !parentOK || parent >= id || !edgeOK || (kind != "scalar" && parent != selected) {
				return false
			}
		}
		if kind == "scalar" && id != selected {
			return false
		}
		if kind == "array" {
			edge, ok := sbvBundleUint(row["edge"])
			if !ok || edge != offset+uint64(i) {
				return false
			}
		}
		if kind == "object" {
			edge, ok := row["edge"].(string)
			if !ok || (i > 0 && edge <= previousEdge) {
				return false
			}
			previousEdge = edge
		}
		switch row["kind"] {
		case "object":
			value, ok := row["value"].(map[string]any)
			if !ok || len(value) != 0 {
				return false
			}
		case "array":
			value, ok := row["value"].([]any)
			if !ok || len(value) != 0 {
				return false
			}
		case "scalar":
			switch row["value"].(type) {
			case nil, string, bool:
			default:
				return false
			}
			if children != 0 {
				return false
			}
		default:
			return false
		}
		if id == 1 && (row["parent_id"] != "0" || row["edge"] != "content_sha256" || row["kind"] != "scalar" || row["value"] != m["reference"].(map[string]any)["content_sha256"]) {
			return false
		}
	}
	return true
}

func sbvBundleRecovery(op string, request, m map[string]any) bool {
	if (op != "bundle_import" && op != "bundle_export") ||
		!sbvRecoveryFields(m, "protocol", "status", "code", "request_sha256", "automatic_retry", "rollback_performed", "recovery") ||
		m["code"] != "sbv."+op+"_incomplete" || m["automatic_retry"] != false || m["rollback_performed"] != false {
		return false
	}
	canonical, err := sbvNativeCanonical(request)
	if err != nil {
		return false
	}
	hash := sha256.Sum256(canonical)
	if m["request_sha256"] != hex.EncodeToString(hash[:]) {
		return false
	}
	r, ok := m["recovery"].(map[string]any)
	if !ok {
		return false
	}
	durable, d := r["durable"].(bool)
	if !d {
		return false
	}
	if op == "bundle_import" {
		published, p := r["manifest_published"].(bool)
		if !p || !sbvRecoveryFields(r, "bundle_path", "manifest_path", "phase", "manifest_published", "durable", "page_files_created", "page_bytes_created", "reference") ||
			!sbvRecoveryPath(r["bundle_path"]) || r["bundle_path"] != request["bundle_path"] ||
			r["manifest_path"] != filepath.Join(r["bundle_path"].(string), "manifest.json") ||
			(durable && (!published || r["phase"] != "sync_manifest")) || (published && r["reference"] == nil) {
			return false
		}
		switch r["phase"] {
		case "create_directory", "write_pages", "publish_manifest", "sync_manifest":
		default:
			return false
		}
		if (published && r["phase"] != "publish_manifest" && r["phase"] != "sync_manifest") ||
			(r["phase"] == "sync_manifest" && !published) ||
			((r["phase"] == "publish_manifest" || r["phase"] == "sync_manifest") && r["reference"] == nil) {
			return false
		}
		for _, key := range []string{"page_files_created", "page_bytes_created"} {
			if _, ok := sbvBundleUint(r[key]); !ok {
				return false
			}
		}
		return r["reference"] == nil || (sbvBundleReference(r["reference"]) && r["reference"].(map[string]any)["manifest_path"] == r["manifest_path"])
	}
	published, p := r["published"].(bool)
	if !p || !sbvRecoveryFields(r, "output_path", "stage_path", "phase", "published", "durable", "expected_binary") ||
		!sbvRecoveryPath(r["output_path"]) || r["output_path"] != request["output_path"] ||
		(durable && (!published || r["phase"] != "sync_export")) || (published && r["expected_binary"] == nil) {
		return false
	}
	switch r["phase"] {
	case "create_stage", "write_export", "publish_export", "sync_export":
	default:
		return false
	}
	if (published && r["phase"] != "publish_export" && r["phase"] != "sync_export") ||
		(r["phase"] == "sync_export" && !published) ||
		((r["phase"] == "publish_export" || r["phase"] == "sync_export") && r["expected_binary"] == nil) {
		return false
	}
	if r["stage_path"] != nil && (!sbvRecoveryPath(r["stage_path"]) ||
		filepath.Dir(r["stage_path"].(string)) != filepath.Dir(r["output_path"].(string)) || r["stage_path"] == r["output_path"]) {
		return false
	}
	if r["expected_binary"] == nil {
		return true
	}
	b, ok := r["expected_binary"].(map[string]any)
	return ok && sbvRecoveryFields(b, "reference", "format", "bytes", "file_sha256", "completion_suffix") &&
		sbvBundleReference(b["reference"]) && reflect.DeepEqual(b["reference"], request["reference"]) && b["format"] == request["format"] &&
		sbvRecoveryPositive(b["bytes"]) && sbvRecoverySHA(b["file_sha256"]) && sbvBundleSuffix(b["format"], b["completion_suffix"], b["reference"], nil)
}

package knowledgeengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

func sbvPartitionedProducer(op string) bool {
	switch op {
	case "run", "generate_census", "evaluate", "economics", "compose_economics":
		return true
	}
	return false
}

func sbvPartitionedOutput(value any) bool {
	m, ok := value.(map[string]any)
	if !ok || !sbvRecoveryFields(m, "kind", "bundle_path", "workspace_path", "write_options") || m["kind"] != "partitioned" ||
		!sbvRecoveryPath(m["bundle_path"]) || !sbvRecoveryPath(m["workspace_path"]) || !sbvBundleWriteOptions(m["write_options"]) {
		return false
	}
	// Native additionally compares actual no-follow directory ancestry/aliases.
	// This syntactic check does not claim filesystem isolation.
	bundle, workspace := m["bundle_path"].(string), m["workspace_path"].(string)
	return bundle != workspace && !strings.HasPrefix(bundle, workspace+"/") && !strings.HasPrefix(workspace, bundle+"/")
}

func sbvBundleSource(value any) bool {
	m, ok := value.(map[string]any)
	if !ok || !sbvRecoveryFields(m, "kind", "reference", "selector", "read_options") || m["kind"] != "bundle" ||
		!sbvBundleReference(m["reference"]) || !sbvBundleSelector(m["selector"]) {
		return false
	}
	options, ok := m["read_options"].(map[string]any)
	if !ok || !sbvRecoveryFields(options, "max_page_bytes", "cache_bytes") {
		return false
	}
	if options["max_page_bytes"] != nil && !sbvRecoveryPositive(options["max_page_bytes"]) {
		return false
	}
	_, ok = sbvBundleUint(options["cache_bytes"])
	return ok
}

func sbvSignalSelection(value any) bool {
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	switch m["kind"] {
	case "all":
		return sbvRecoveryFields(m, "kind")
	case "range":
		first, ok := sbvBundleUint(m["first"])
		if !ok || !sbvRecoveryFields(m, "kind", "first", "count") {
			return false
		}
		if m["count"] == nil {
			return true
		}
		count, ok := sbvBundleUint(m["count"])
		return ok && count <= ^uint64(0)-first
	case "ids":
		ids, ok := m["ids"].([]any)
		if !ok || !sbvRecoveryFields(m, "kind", "ids", "order") || (m["order"] != "supplied" && m["order"] != "census") {
			return false
		}
		seen := map[string]bool{}
		for _, value := range ids {
			id, ok := value.(string)
			if !ok || id == "" || len(id) > 256 || seen[id] {
				return false
			}
			seen[id] = true
		}
		return true
	case "referenced_ids":
		return sbvRecoveryFields(m, "kind", "source", "order") && (m["order"] == "supplied" || m["order"] == "census") && sbvBundleSource(m["source"])
	}
	return false
}

// This validates new control alternatives, not source contents, mathematics,
// physical paths, or a provider's claims. Native owns those admissions.
func sbvPartitionedRequestPreflight(op string, p map[string]any) error {
	bad := func() error { return fmt.Errorf("SBV partitioned control selection mismatch") }
	if !sbvPartitionedProducer(op) && op != "dataset_execute" {
		return nil
	}
	if p["protocol"] != "symphony.sbv."+strings.ReplaceAll(op, "_", "-")+"-input.v1" {
		return bad()
	}
	output, partitioned := p["output"]
	_, legacy := p["output_path"]
	if partitioned == legacy || (partitioned && !sbvPartitionedOutput(output)) {
		return bad()
	}
	if op == "dataset_execute" {
		child, err := sbvExpandedResidentRequest(p)
		if err != nil {
			return err
		}
		childOp, _ := p["operation"].(string)
		if partitioned && childOp != "run" && childOp != "generate_census" && childOp != "evaluate" {
			return bad()
		}
		return sbvPartitionedRequestPreflight(childOp, child)
	}
	checkTagged := func(value any) bool {
		m, ok := value.(map[string]any)
		if !ok {
			return false
		}
		_, tagged := m["kind"]
		return !tagged || sbvBundleSource(value)
	}
	if op == "evaluate" {
		if !checkTagged(p["census"]) {
			return bad()
		}
		if model, ok := p["model"].(map[string]any); ok {
			if params, ok := model["parameters"].(map[string]any); ok {
				if source, present := params["outcomes_source"]; present {
					_, inline := params["outcomes"]
					if model["id"] != "external_outcomes" || inline || !sbvBundleSource(source) {
						return bad()
					}
				}
			}
		}
	}
	if op == "economics" {
		if source, present := p["source"]; present && !checkTagged(source) {
			return bad()
		}
		if selections, ok := p["selections"].(map[string]any); ok {
			switch selections["kind"] {
			case "referenced_rows":
				if !sbvRecoveryFields(selections, "kind", "source") || !sbvBundleSource(selections["source"]) {
					return bad()
				}
			case "apply":
				economic, ok := selections["economics"].(map[string]any)
				if !ok || !sbvRecoveryFields(selections, "kind", "signals", "economics") || !sbvSignalSelection(selections["signals"]) ||
					!sbvRecoveryFields(economic, "transform", "mode", "nonexecution_pnl") {
					return bad()
				}
			default:
				return bad()
			}
		}
	}
	if op == "compose_economics" {
		if rows, ok := p["components"].([]any); ok {
			for _, raw := range rows {
				row, ok := raw.(map[string]any)
				if !ok || !checkTagged(row["source"]) {
					return bad()
				}
				if _, proposed := row["outcome_selector"]; proposed {
					return bad()
				}
				if _, present := row["outcome_pointer"].(string); !present {
					return bad()
				}
			}
		}
	}
	return nil
}

func sbvExpandedResidentRequest(p map[string]any) (map[string]any, error) {
	child, ok := p["request"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("SBV resident child request required")
	}
	_, old := child["output_path"]
	_, new := child["output"]
	if old || new {
		return nil, fmt.Errorf("SBV resident child must omit both output forms")
	}
	output, partitioned := p["output"]
	path, legacy := p["output_path"]
	if partitioned == legacy {
		return nil, fmt.Errorf("SBV resident parent must select exactly one output")
	}
	out := make(map[string]any, len(child)+1)
	for key, value := range child {
		out[key] = value
	}
	if partitioned {
		out["output"] = output
	} else {
		out["output_path"] = path
	}
	return out, nil
}

func sbvRequestHashMatches(p map[string]any, hash any) bool {
	raw, err := sbvNativeCanonical(p)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(raw)
	return hash == hex.EncodeToString(sum[:])
}

func sbvPartitionedReceiptDepth(value any, depth int) bool {
	// Native leaves four nesting levels for resident and process envelopes.
	// This applies to the small receipt, never the partitioned logical result.
	if depth > maxJSONDepth-4 {
		return false
	}
	switch v := value.(type) {
	case map[string]any:
		for _, child := range v {
			if !sbvPartitionedReceiptDepth(child, depth+1) {
				return false
			}
		}
	case []any:
		for _, child := range v {
			if !sbvPartitionedReceiptDepth(child, depth+1) {
				return false
			}
		}
	}
	return true
}

func sbvPartitionedCause(value any) bool {
	cause, ok := value.(map[string]any)
	if !ok || !sbvRecoveryFields(cause, "category", "code") {
		return false
	}
	category, c := cause["category"].(string)
	code, k := cause["code"].(string)
	if !c || !k || len(code) == 0 || len(code) > 128 || code[0] < 'a' || code[0] > 'z' {
		return false
	}
	for _, ch := range code {
		if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '.' || ch == '-') {
			return false
		}
	}
	switch category {
	case "contract", "storage", "deadline", "allocation", "unexpected":
	default:
		return false
	}
	switch code {
	case "deadline.exceeded", "request.deadline_expired":
		return category == "deadline"
	case "sbv.contract", "sbv.length_error":
		return category == "contract"
	case "sbv.allocation_failed":
		return category == "allocation"
	case "sbv.unexpected_exception", "sbv.unknown_exception", "sbv.invalid_native_error_code", "bundle.interrupted":
		return category == "unexpected"
	}
	if strings.HasPrefix(code, "bundle.") {
		return category == "storage"
	}
	if strings.HasPrefix(code, "logical.") {
		return category == "contract"
	}
	// Other safe native codes remain observable without guessing their domain.
	// The fixed category vocabulary and closed shape still exclude raw causes.
	return true
}

func sbvPartitionedRecovery(op string, p, m map[string]any) bool {
	if m["automatic_retry"] != false || m["rollback_performed"] != false || !sbvRequestHashMatches(p, m["request_sha256"]) {
		return false
	}
	if op == "dataset_execute" {
		if !sbvRecoveryFields(m, "protocol", "status", "code", "request_sha256", "automatic_retry", "rollback_performed", "operation", "child_recovery") ||
			m["code"] != "sbv.dataset_execution_incomplete" || !reflect.DeepEqual(m["operation"], p["operation"]) {
			return false
		}
		childOp, ok := m["operation"].(string)
		if !ok || (childOp != "run" && childOp != "generate_census" && childOp != "evaluate") {
			return false
		}
		child, err := sbvExpandedResidentRequest(p)
		recovery, ok := m["child_recovery"].(map[string]any)
		return err == nil && ok && recovery["status"] == "recovery_required" && recovery["protocol"] == "symphony.sbv."+strings.ReplaceAll(childOp, "_", "-")+".v1" && sbvPartitionedRecovery(childOp, child, recovery)
	}
	if !sbvPartitionedProducer(op) || !sbvRecoveryFields(m, "protocol", "status", "code", "cause", "request_sha256", "automatic_retry", "rollback_performed", "recovery") || m["code"] != "sbv.partitioned_result_incomplete" || !sbvPartitionedCause(m["cause"]) {
		return false
	}
	r, ok := m["recovery"].(map[string]any)
	if !ok || !sbvRecoveryFields(r, "output", "phase", "workspace_created", "workspace_creation_durable", "automatic_cleanup", "final_storage") ||
		!reflect.DeepEqual(r["output"], p["output"]) || !sbvPartitionedOutput(r["output"]) || r["automatic_cleanup"] != false {
		return false
	}
	created, c := r["workspace_created"].(bool)
	durable, d := r["workspace_creation_durable"].(bool)
	if !c || !d || (durable && !created) {
		return false
	}
	switch r["phase"] {
	case "create_workspace":
		return r["final_storage"] == nil
	case "produce":
		return created && durable && r["final_storage"] == nil
	case "finalize_result":
		return created && durable && (r["final_storage"] == nil || sbvBundleWriterRecovery(r["final_storage"], p["output"].(map[string]any)["bundle_path"]))
	}
	return false
}

// ValidateSBVPartitionedResult handles only the explicitly selected new output
// alternative. Legacy result receipts continue through their original validator.
func ValidateSBVPartitionedResult(op string, p map[string]any, raw []byte) (bool, error) {
	_, selected := p["output"]
	if !sbvPartitionedProducer(op) && op != "dataset_execute" {
		return false, nil
	}
	m, err := sqavObject(raw, maxResponseBytes)
	if err != nil {
		return selected, err
	}
	_, hasStorage := m["storage"]
	newRecovery := m["code"] == "sbv.partitioned_result_incomplete" || m["code"] == "sbv.dataset_execution_incomplete"
	if !selected && !hasStorage && !newRecovery {
		return false, nil
	}
	bad := func() (bool, error) { return true, fmt.Errorf("SBV partitioned result contract mismatch") }
	// New producer receipts share the smaller resident frame profile even when
	// invoked directly. This reserve concerns only control framing, never the
	// portable result's aggregate bytes/nodes/signals.
	if validateJSONObjectWithValueLimit(raw, maxRequestBytes-2048, maxJSONValues-64) != nil || !sbvPartitionedReceiptDepth(m, 0) {
		return bad()
	}
	if !selected || sbvPartitionedRequestPreflight(op, p) != nil || !sbvBundlePortable(m) || m["protocol"] != "symphony.sbv."+strings.ReplaceAll(op, "_", "-")+".v1" {
		return bad()
	}
	if m["status"] == "recovery_required" {
		if !sbvPartitionedRecovery(op, p, m) {
			return bad()
		}
		return true, nil
	}
	if !sbvRecoveryFields(m, "protocol", "status", "content_sha256", "storage", "summary") || (m["status"] != "partial" && m["status"] != "completed") || !sbvRecoverySHA(m["content_sha256"]) {
		return bad()
	}
	s, ok := m["storage"].(map[string]any)
	output := p["output"].(map[string]any)
	if !ok || !sbvRecoveryFields(s, "kind", "reference", "workspace_path", "workspace_retained", "manifest_bytes", "logical_body_bytes", "logical_body_nodes", "logical_body_values", "exported_value_nodes", "page_files_created", "page_bytes_created", "write_options") ||
		s["kind"] != "partitioned" || !sbvBundleReference(s["reference"]) || !sbvBundleCounts(s) || !sbvRecoveryPositive(s["manifest_bytes"]) ||
		s["workspace_path"] != output["workspace_path"] || s["workspace_retained"] != true || !reflect.DeepEqual(s["write_options"], output["write_options"]) {
		return bad()
	}
	ref := s["reference"].(map[string]any)
	if ref["content_sha256"] != m["content_sha256"] || ref["manifest_path"] != filepath.Join(output["bundle_path"].(string), "manifest.json") {
		return bad()
	}
	for _, key := range []string{"page_files_created", "page_bytes_created"} {
		if _, ok := sbvBundleUint(s[key]); !ok {
			return bad()
		}
	}
	summary, ok := m["summary"].(map[string]any)
	if !ok || !sbvRecoveryFields(summary, "status", "reason", "data") || summary["status"] != "available" {
		return bad()
	}
	if _, ok := summary["reason"].(string); !ok {
		return bad()
	}
	if _, ok := summary["data"].(map[string]any); !ok {
		return bad()
	}
	return true, nil
}

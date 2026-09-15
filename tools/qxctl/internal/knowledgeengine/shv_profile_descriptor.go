package knowledgeengine

import (
	"encoding/json"
	"strings"
)

func shvProfileDescriptor(p, r map[string]any, version string) error {
	if len(p) != 0 || !shvFields(r, "protocol", "format_version", "module_id", "engine_id", "vector_id", "engine_version", "process_protocols", "contract_versions", "operations", "limits", "supported_scopes", "language", "thermal_path", "canonical_apply_enabled", "session_mutation_enabled", "network_listener", "descriptor_digest") {
		return shvFail()
	}
	if e := scvSeal(r, "descriptor_digest"); e != nil {
		return e
	}
	s := shvProfileSpec
	if r["module_id"] != s.moduleID || r["engine_id"] != s.engineID || r["vector_id"] != "shv" || r["engine_version"] != version || r["format_version"] != json.Number("2") || r["language"] != "C++26" || r["thermal_path"] != "freezing" || r["canonical_apply_enabled"] != false || r["session_mutation_enabled"] != false || r["network_listener"] != false || !scvEqual(r["process_protocols"], []any{processProtocol}) || !scvEqual(r["supported_scopes"], []any{"user"}) {
		return shvFail()
	}
	limits := map[string]any{"request_bytes": 1048576, "response_bytes": 4194304, "json_depth": 64, "json_values": 32768, "path_bytes": 4096, "snapshot_files": 1024, "snapshot_file_bytes": 4194304, "deadline_ahead_ms": 300000}
	contracts := []any{"knowledge/shv/PROFILES.md@v1"}
	if !scvEqual(r["limits"], limits) || !scvEqual(r["contract_versions"], contracts) {
		return shvFail()
	}
	ops, ok := r["operations"].([]any)
	count := len(shvProfileOutputs)
	if !ok || len(ops) != count {
		return shvFail()
	}
	seen := map[string]bool{}
	domain := "shv-profile"
	feature := "ssfv:symphony:shv-profile-engine"
	for _, v := range ops {
		o := shvMap(v)
		name := shvText(o["operation_name"])
		out, ok := SHVProfileResultProtocol(name)
		interaction := "invoke"
		if name == "inspect" {
			interaction = "inspect"
		}
		if name == "mapping_diagnose" {
			interaction = "query"
		}
		interactions := []any{interaction}
		if !shvFields(o, "engine_operation_id", "operation_name", "availability", "feature_ids", "administrative_interactions", "administration_disposition", "input_protocol", "output_protocol", "mutability", "idempotency", "expected_state_required", "authorization_requirement", "recovery_operation_id", "direct_invocation", "thermal_path") || !scvEqual(o["administrative_interactions"], interactions) || o["administration_disposition"] != "qxctl_required" || o["mutability"] != "read_only" || o["idempotency"] != "idempotent" || o["recovery_operation_id"] != nil || o["direct_invocation"] != "supported" {
			return shvFail()
		}
		if !ok || seen[name] || o["engine_operation_id"] != "engop:symphony:"+domain+"."+strings.ReplaceAll(name, "_", ".") || o["availability"] != "implemented" || !scvEqual(o["feature_ids"], []any{feature}) || o["input_protocol"] != SHVProfileInputProtocol(name) || o["output_protocol"] != out || o["authorization_requirement"] != "none" || o["expected_state_required"] != false || o["thermal_path"] != "freezing" {
			return shvFail()
		}
		seen[name] = true
	}
	return nil
}

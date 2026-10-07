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

// ValidateSBVRecovery admits the operation-scoped recovery result before the
// usual published-artifact receipt. A transport success delivers this evidence;
// recovery_required is deliberately not successful artifact publication. The
// CLI renders the complete admitted payload and uses its own nonzero evidence
// exit. The shared process error contract and other vectors are unchanged.
func ValidateSBVRecovery(op string, request map[string]any, raw []byte) (bool, error) {
	value, err := sqavObject(raw, maxResponseBytes)
	if err != nil {
		return false, err
	}
	if value["status"] != "recovery_required" {
		return false, nil
	}
	bad := func() (bool, error) {
		return true, fmt.Errorf("SBV recovery evidence contract mismatch")
	}
	if op != "source_retain" && op != "source_export" {
		return bad()
	}
	fields := []string{"protocol", "status", "code", "stage", "request_sha256", "output_path", "automatic_retry", "rollback_performed", "recovery"}
	if op == "source_export" {
		fields = append(fields, "receipt_path")
	}
	if !sbvRecoveryFields(value, fields...) ||
		value["protocol"] != "symphony.sbv."+strings.ReplaceAll(op, "_", "-")+".v1" ||
		value["automatic_retry"] != false || value["rollback_performed"] != false ||
		!sbvRecoveryPath(value["output_path"]) || value["output_path"] != request["output_path"] {
		return bad()
	}
	canonical, err := sbvNativeCanonical(request)
	if err != nil {
		return bad()
	}
	hash := sha256.Sum256(canonical)
	if value["request_sha256"] != hex.EncodeToString(hash[:]) {
		return bad()
	}
	recovery, ok := value["recovery"].(map[string]any)
	if !ok {
		return bad()
	}
	if op == "source_export" {
		if !sbvRecoveryFields(recovery, "binary_publication_confirmed", "expected_binary") ||
			value["code"] != "sbv.source_export_unpublished" ||
			!sbvRecoveryPath(value["receipt_path"]) || value["receipt_path"] != request["receipt_path"] ||
			value["receipt_path"] == value["output_path"] {
			return bad()
		}
		confirmed, ok := recovery["binary_publication_confirmed"].(bool)
		if !ok || (confirmed && value["stage"] != "receipt_publication") ||
			(!confirmed && value["stage"] != "binary_publication") {
			return bad()
		}
		binary, ok := recovery["expected_binary"].(map[string]any)
		if !ok || !sbvRecoveryFields(binary, "protocol", "path", "bytes", "sha256", "kind", "reference", "capture_reference", "metadata_reference", "batch_content_id", "retention_receipt") ||
			binary["protocol"] != "symphony.sbv.source-binary-receipt.v1" || binary["path"] != value["output_path"] ||
			binary["kind"] != request["kind"] || !sbvRecoverySHA(binary["sha256"]) ||
			!sbvRecoveryPositive(binary["bytes"]) || !sbvRecoveryTagged(binary["capture_reference"], "sqac1-sha256-") ||
			!sbvRecoveryTagged(binary["metadata_reference"], "sqmv1-sha256-") || !sbvRecoverySHA(binary["batch_content_id"]) {
			return bad()
		}
		if binary["kind"] != "original" && binary["kind"] != "capture" && binary["kind"] != "manifest" {
			return bad()
		}
		selected, ok := request["retained_source"].(map[string]any)
		if !ok || !sbvRecoveryReference(binary["reference"]) || !reflect.DeepEqual(binary["reference"], selected["reference"]) {
			return bad()
		}
		receipt, ok := binary["retention_receipt"].(map[string]any)
		if !ok || !sbvRecoveryReceipt(receipt) || receipt["content_id"] != binary["batch_content_id"] {
			return bad()
		}
		selection, ok := selected["delivery"].(map[string]any)
		if !ok || selection["processed_ack"] != "none" || receipt["batch_sequence"] != selection["first_sequence"] {
			return bad()
		}
		return true, nil
	}
	if !sbvRecoveryFields(recovery, "retention_confirmed", "root", "options", "metadata", "batch", "receipt") {
		return bad()
	}
	confirmed, ok := recovery["retention_confirmed"].(bool)
	selected, selectionOK := request["retention"].(map[string]any)
	if !ok || !selectionOK || !sbvRecoveryPath(recovery["root"]) || recovery["root"] != selected["root"] {
		return bad()
	}
	if confirmed {
		if value["code"] != "sbv.source_retention_unpublished" || value["stage"] != "result_publication" || recovery["receipt"] == nil {
			return bad()
		}
	} else {
		if value["code"] != "sbv.source_owner.outcome_uncertain" || recovery["receipt"] != nil {
			return bad()
		}
		switch value["stage"] {
		case "store_create":
			if selected["mode"] != "create" {
				return bad()
			}
		case "store_open":
			if selected["mode"] != "open" {
				return bad()
			}
		case "store_append":
		default:
			return bad()
		}
	}
	options, ok := recovery["options"].(map[string]any)
	if !ok || !sbvRecoveryFields(options, "partition", "producer_generation", "store_generation", "first_sequence", "limits") ||
		!sbvRecoveryText(options["partition"], 4096) || !sbvRecoveryGeneration(options["producer_generation"]) ||
		!sbvRecoveryGeneration(options["store_generation"]) || !sbvRecoveryUnsigned(options["first_sequence"]) {
		return bad()
	}
	for _, key := range []string{"partition", "producer_generation", "store_generation", "first_sequence"} {
		if options[key] != selected[key] {
			return bad()
		}
	}
	limits, ok := options["limits"].(map[string]any)
	inputLimits, inputOK := request["limits"].(map[string]any)
	if !ok || !inputOK || !sbvRecoveryFields(limits, "max_frame_bytes", "max_store_bytes", "max_batches") ||
		!reflect.DeepEqual(limits, inputLimits["store"]) {
		return bad()
	}
	for _, v := range limits {
		if !sbvRecoveryPositive(v) {
			return bad()
		}
	}
	metadata, ok := recovery["metadata"].(map[string]any)
	if !ok || !sbvRecoveryMetadata(metadata) {
		return bad()
	}
	description := metadata["description"].(map[string]any)
	source, ok := request["source"].(map[string]any)
	if !ok || description["dataset_id"] != source["dataset"] {
		return bad()
	}
	batch, ok := recovery["batch"].(map[string]any)
	if !ok || !sbvRecoveryFields(batch, "descriptor", "content_id") || !sbvRecoverySHA(batch["content_id"]) {
		return bad()
	}
	descriptor, ok := batch["descriptor"].(map[string]any)
	if !ok || !sbvRecoveryFields(descriptor, "binding", "partition", "source_binding", "source_position", "producer_generation", "batch_sequence", "record_count") ||
		descriptor["partition"] != options["partition"] || descriptor["producer_generation"] != options["producer_generation"] ||
		descriptor["batch_sequence"] != selected["batch_sequence"] || descriptor["record_count"] != "1" ||
		!sbvRecoveryUnsigned(descriptor["batch_sequence"]) || !sbvRecoveryTagged(descriptor["source_binding"], "sqas1-sha256-") ||
		!sbvRecoveryTagged(descriptor["source_position"], "sqac1-sha256-") {
		return bad()
	}
	sequence, _ := strconv.ParseUint(descriptor["batch_sequence"].(string), 10, 64)
	first, _ := strconv.ParseUint(options["first_sequence"].(string), 10, 64)
	if sequence < first {
		return bad()
	}
	binding, ok := descriptor["binding"].(map[string]any)
	if !ok || !sbvRecoveryFields(binding, "metadata_ref", "dataset_revision", "schema_version", "layout_version", "access_scope") || binding["metadata_ref"] != metadata["reference"] {
		return bad()
	}
	for _, key := range []string{"dataset_revision", "schema_version", "layout_version", "access_scope"} {
		if binding[key] != description[key] {
			return bad()
		}
	}
	if confirmed {
		receipt, ok := recovery["receipt"].(map[string]any)
		if !ok || !sbvRecoveryReceipt(receipt) || receipt["store_generation"] != options["store_generation"] ||
			receipt["producer_generation"] != options["producer_generation"] || receipt["batch_sequence"] != descriptor["batch_sequence"] ||
			receipt["content_id"] != batch["content_id"] {
			return bad()
		}
	}
	return true, nil
}

func sbvRecoveryFields(value map[string]any, names ...string) bool {
	if len(value) != len(names) {
		return false
	}
	for _, key := range names {
		if _, present := value[key]; !present {
			return false
		}
	}
	return true
}

func sbvRecoveryUnsigned(value any) bool {
	s, ok := value.(string)
	n, err := strconv.ParseUint(s, 10, 64)
	return ok && err == nil && strconv.FormatUint(n, 10) == s
}

func sbvRecoveryPositive(value any) bool { return value != "0" && sbvRecoveryUnsigned(value) }

func sbvRecoveryText(value any, maximum int) bool {
	s, ok := value.(string)
	return ok && len(s) > 0 && len(s) <= maximum
}

func sbvRecoverySHA(value any) bool {
	s, ok := value.(string)
	return ok && sbvSHA.MatchString(s)
}

func sbvRecoveryTagged(value any, prefix string) bool {
	s, ok := value.(string)
	return ok && strings.HasPrefix(s, prefix) && sbvRecoverySHA(strings.TrimPrefix(s, prefix))
}

func sbvRecoveryGeneration(value any) bool {
	s, ok := value.(string)
	if !ok || len(s) != 32 || s == strings.Repeat("0", 32) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && strings.ToLower(s) == s
}

func sbvRecoveryPath(value any) bool {
	s, ok := value.(string)
	return ok && len(s) > 1 && len(s) <= 4096 && filepath.IsAbs(s) && filepath.Clean(s) == s &&
		!strings.ContainsAny(s, "\\\x00\r\n\t") && !strings.ContainsFunc(s, func(r rune) bool { return r < 32 || r == 127 })
}

func sbvRecoveryReference(value any) bool {
	r, ok := value.(map[string]any)
	if !ok || !sbvRecoveryFields(r, "path", "expected_sha256", "pointer") ||
		!sbvRecoveryPath(r["path"]) || !sbvRecoverySHA(r["expected_sha256"]) {
		return false
	}
	pointer, ok := r["pointer"].(string)
	if !ok || (pointer != "" && !strings.HasPrefix(pointer, "/")) {
		return false
	}
	for i := 0; i < len(pointer); i++ {
		if pointer[i] == '~' {
			if i+1 >= len(pointer) || (pointer[i+1] != '0' && pointer[i+1] != '1') {
				return false
			}
			i++
		}
	}
	return true
}

func sbvRecoveryReceipt(r map[string]any) bool {
	return sbvRecoveryFields(r, "store_generation", "producer_generation", "batch_sequence", "frame_bytes", "content_id", "frame_sha256", "commit_sha256") &&
		sbvRecoveryGeneration(r["store_generation"]) && sbvRecoveryGeneration(r["producer_generation"]) &&
		sbvRecoveryUnsigned(r["batch_sequence"]) && sbvRecoveryPositive(r["frame_bytes"]) &&
		sbvRecoverySHA(r["content_id"]) && sbvRecoverySHA(r["frame_sha256"]) && sbvRecoverySHA(r["commit_sha256"])
}

func sbvRecoveryMetadata(metadata map[string]any) bool {
	if !sbvRecoveryFields(metadata, "reference", "encoded_sha256", "bytes", "description") ||
		!sbvRecoveryTagged(metadata["reference"], "sqmv1-sha256-") || !sbvRecoverySHA(metadata["encoded_sha256"]) ||
		!sbvRecoveryPositive(metadata["bytes"]) {
		return false
	}
	description, ok := metadata["description"].(map[string]any)
	if !ok || !sbvRecoveryFields(description, "dataset_id", "dataset_revision", "schema_version", "layout_version", "access_scope", "producer_ref", "evidence") {
		return false
	}
	for _, key := range []string{"dataset_id", "dataset_revision", "schema_version", "layout_version", "access_scope", "producer_ref"} {
		if !sbvRecoveryText(description[key], 4096) {
			return false
		}
	}
	evidence, ok := description["evidence"].([]any)
	if !ok || len(evidence) < 3 || len(evidence) > 128 {
		return false
	}
	counts := map[string]int{}
	for _, row := range evidence {
		item, ok := row.(map[string]any)
		if !ok || !sbvRecoveryFields(item, "role", "producer_ref", "evidence_ref") ||
			!sbvRecoveryText(item["producer_ref"], 4096) || !sbvRecoveryText(item["evidence_ref"], 4096) {
			return false
		}
		role, ok := item["role"].(string)
		if !ok {
			return false
		}
		switch role {
		case "schema", "layout", "access", "source", "time", "coverage", "lineage", "units":
			counts[role]++
		default:
			return false
		}
	}
	return counts["schema"] == 1 && counts["layout"] == 1 && counts["access"] == 1
}

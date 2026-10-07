package knowledgeengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func recoveryClone(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	result, err := sqavObject(raw, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func recoveryHash(t *testing.T, request map[string]any) string {
	t.Helper()
	raw, err := sbvNativeCanonical(request)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func recoveryFixture(t *testing.T, export bool) (string, map[string]any, map[string]any) {
	t.Helper()
	hash := strings.Repeat("a", 64)
	producer, store := strings.Repeat("1", 32), strings.Repeat("2", 32)
	partition := "selected-<>&-\u2028-\u2029-\\u2028"
	options := map[string]any{"partition": partition, "producer_generation": producer, "store_generation": store,
		"first_sequence": "7", "limits": map[string]any{"max_frame_bytes": "8388608", "max_store_bytes": "33554432", "max_batches": "4"}}
	receipt := map[string]any{"store_generation": store, "producer_generation": producer, "batch_sequence": "7", "frame_bytes": "1000",
		"content_id": hash, "frame_sha256": strings.Repeat("b", 64), "commit_sha256": strings.Repeat("c", 64)}
	metadata := map[string]any{"reference": "sqmv1-sha256-" + hash, "encoded_sha256": hash, "bytes": "1024",
		"description": map[string]any{"dataset_id": "GLBX.MDP3", "dataset_revision": "selected-revision", "schema_version": "sqav-capture-v1",
			"layout_version": "sqav-capture-v1", "access_scope": "private:selected", "producer_ref": "selected-producer",
			"evidence": []any{
				map[string]any{"role": "schema", "producer_ref": "selected-producer", "evidence_ref": "sqav-capture-v1"},
				map[string]any{"role": "layout", "producer_ref": "selected-producer", "evidence_ref": "sqav-capture-v1"},
				map[string]any{"role": "access", "producer_ref": "selected-producer", "evidence_ref": "caller-selected-access"}}}}
	batch := map[string]any{"content_id": hash, "descriptor": map[string]any{
		"binding":   map[string]any{"metadata_ref": metadata["reference"], "dataset_revision": "selected-revision", "schema_version": "sqav-capture-v1", "layout_version": "sqav-capture-v1", "access_scope": "private:selected"},
		"partition": partition, "source_binding": "sqas1-sha256-" + hash, "source_position": "sqac1-sha256-" + hash,
		"producer_generation": producer, "batch_sequence": "7", "record_count": "1"}}
	request := map[string]any{"protocol": "symphony.sbv.source-retain-input.v1", "output_path": "/private/results/new.json",
		"source": map[string]any{"path": "/private/source.dbn", "expected_sha256": hash, "dataset": "GLBX.MDP3"},
		"limits": map[string]any{"store": options["limits"]},
		"retention": map[string]any{"mode": "create", "root": "/private/selected-store", "partition": partition,
			"producer_generation": producer, "store_generation": store, "first_sequence": "7", "batch_sequence": "7"},
		"extensions": map[string]any{"declared": "explicit\tvalue\nwith controls escaped"}}
	result := map[string]any{"protocol": "symphony.sbv.source-retain.v1", "status": "recovery_required",
		"code": "sbv.source_retention_unpublished", "stage": "result_publication", "output_path": request["output_path"],
		"automatic_retry": false, "rollback_performed": false,
		"recovery": map[string]any{"retention_confirmed": true, "root": "/private/selected-store", "options": options,
			"metadata": metadata, "batch": batch, "receipt": receipt}}
	op := "source_retain"
	if export {
		op = "source_export"
		reference := map[string]any{"path": "/private/results/source.json", "expected_sha256": hash, "pointer": "/sections/source/data"}
		request = map[string]any{"protocol": "symphony.sbv.source-export-input.v1", "output_path": "/private/results/original.dbn",
			"receipt_path": "/private/results/export.json", "kind": "original", "extensions": map[string]any{},
			"retained_source": map[string]any{"reference": reference, "delivery": map[string]any{"processed_ack": "none", "first_sequence": "7"}}}
		result = map[string]any{"protocol": "symphony.sbv.source-export.v1", "status": "recovery_required",
			"code": "sbv.source_export_unpublished", "stage": "receipt_publication", "output_path": request["output_path"], "receipt_path": request["receipt_path"],
			"automatic_retry": false, "rollback_performed": false,
			"recovery": map[string]any{"binary_publication_confirmed": true, "expected_binary": map[string]any{
				"protocol": "symphony.sbv.source-binary-receipt.v1", "path": request["output_path"], "bytes": "900", "sha256": hash, "kind": "original", "reference": reference,
				"capture_reference": "sqac1-sha256-" + hash, "metadata_reference": metadata["reference"], "batch_content_id": hash, "retention_receipt": receipt}}}
	}
	result["request_sha256"] = recoveryHash(t, request)
	return op, request, result
}

func recoveryValidate(t *testing.T, op string, request, value map[string]any, want bool) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	handled, err := ValidateSBVRecovery(op, request, raw)
	if !handled || (err == nil) != want {
		t.Fatalf("handled=%v error=%v wanted admission=%v", handled, err, want)
	}
}

func TestSBVRetainRecoveryCorrespondence(t *testing.T) {
	op, request, value := recoveryFixture(t, false)
	recoveryValidate(t, op, request, value, true)
	for _, stage := range []string{"store_create", "store_open", "store_append"} {
		t.Run(stage, func(t *testing.T) {
			input, result := recoveryClone(t, request), recoveryClone(t, value)
			if stage == "store_open" {
				input["retention"].(map[string]any)["mode"] = "open"
			}
			result["request_sha256"] = recoveryHash(t, input)
			result["code"], result["stage"] = "sbv.source_owner.outcome_uncertain", stage
			recovery := result["recovery"].(map[string]any)
			recovery["retention_confirmed"], recovery["receipt"] = false, nil
			recoveryValidate(t, op, input, result, true)
		})
	}
	mutations := map[string]func(map[string]any){
		"unknown stage":         func(v map[string]any) { v["stage"] = "untrusted-secret" },
		"unknown code":          func(v map[string]any) { v["code"] = "sbv.caller-secret" },
		"freeform cause":        func(v map[string]any) { v["cause"] = "credential or arbitrary native error" },
		"missing field":         func(v map[string]any) { delete(v, "automatic_retry") },
		"automatic retry":       func(v map[string]any) { v["automatic_retry"] = true },
		"claimed rollback":      func(v map[string]any) { v["rollback_performed"] = true },
		"different request":     func(v map[string]any) { v["request_sha256"] = strings.Repeat("b", 64) },
		"different destination": func(v map[string]any) { v["output_path"] = "/private/elsewhere.json" },
		"different store":       func(v map[string]any) { v["recovery"].(map[string]any)["root"] = "/private/other-store" },
		"missing receipt":       func(v map[string]any) { v["recovery"].(map[string]any)["receipt"] = nil },
		"wrong options": func(v map[string]any) {
			v["recovery"].(map[string]any)["options"].(map[string]any)["first_sequence"] = "6"
		},
		"wrong limits": func(v map[string]any) {
			v["recovery"].(map[string]any)["options"].(map[string]any)["limits"].(map[string]any)["max_batches"] = "5"
		},
		"wrong generation": func(v map[string]any) {
			v["recovery"].(map[string]any)["receipt"].(map[string]any)["store_generation"] = strings.Repeat("3", 32)
		},
		"wrong content": func(v map[string]any) {
			v["recovery"].(map[string]any)["receipt"].(map[string]any)["content_id"] = strings.Repeat("b", 64)
		},
		"wrong sequence": func(v map[string]any) {
			v["recovery"].(map[string]any)["receipt"].(map[string]any)["batch_sequence"] = "8"
		},
		"receipt untyped field": func(v map[string]any) {
			v["recovery"].(map[string]any)["receipt"].(map[string]any)["message"] = "secret"
		},
		"wrong metadata binding": func(v map[string]any) {
			v["recovery"].(map[string]any)["batch"].(map[string]any)["descriptor"].(map[string]any)["binding"].(map[string]any)["metadata_ref"] = "sqmv1-sha256-" + strings.Repeat("b", 64)
		},
		"wrong dataset": func(v map[string]any) {
			v["recovery"].(map[string]any)["metadata"].(map[string]any)["description"].(map[string]any)["dataset_id"] = "wrong"
		},
		"untyped metadata field": func(v map[string]any) {
			v["recovery"].(map[string]any)["metadata"].(map[string]any)["message"] = "secret"
		},
		"unknown evidence role": func(v map[string]any) {
			v["recovery"].(map[string]any)["metadata"].(map[string]any)["description"].(map[string]any)["evidence"].([]any)[0].(map[string]any)["role"] = "arbitrary"
		},
		"integer overflow": func(v map[string]any) {
			v["recovery"].(map[string]any)["receipt"].(map[string]any)["frame_bytes"] = "18446744073709551616"
		},
		"json number": func(v map[string]any) {
			v["recovery"].(map[string]any)["receipt"].(map[string]any)["frame_bytes"] = int64(1000)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			bad := recoveryClone(t, value)
			mutate(bad)
			recoveryValidate(t, op, request, bad, false)
		})
	}
}

func TestSBVExportRecoveryCorrespondence(t *testing.T) {
	op, request, value := recoveryFixture(t, true)
	recoveryValidate(t, op, request, value, true)
	uncertain := recoveryClone(t, value)
	uncertain["stage"] = "binary_publication"
	uncertain["recovery"].(map[string]any)["binary_publication_confirmed"] = false
	recoveryValidate(t, op, request, uncertain, true)
	mutations := map[string]func(map[string]any){
		"receipt mismatch":       func(v map[string]any) { v["receipt_path"] = "/private/other.json" },
		"stage mismatch":         func(v map[string]any) { v["stage"] = "binary_publication" },
		"untyped recovery field": func(v map[string]any) { v["recovery"].(map[string]any)["cause"] = "arbitrary native output" },
		"binary destination": func(v map[string]any) {
			v["recovery"].(map[string]any)["expected_binary"].(map[string]any)["path"] = "/private/other.dbn"
		},
		"source reference": func(v map[string]any) {
			v["recovery"].(map[string]any)["expected_binary"].(map[string]any)["reference"].(map[string]any)["expected_sha256"] = strings.Repeat("b", 64)
		},
		"kind mismatch": func(v map[string]any) {
			v["recovery"].(map[string]any)["expected_binary"].(map[string]any)["kind"] = "capture"
		},
		"frame identity": func(v map[string]any) {
			v["recovery"].(map[string]any)["expected_binary"].(map[string]any)["batch_content_id"] = strings.Repeat("b", 64)
		},
		"capture reference": func(v map[string]any) {
			v["recovery"].(map[string]any)["expected_binary"].(map[string]any)["capture_reference"] = "other:" + strings.Repeat("a", 64)
		},
		"manifest reference": func(v map[string]any) {
			v["recovery"].(map[string]any)["expected_binary"].(map[string]any)["metadata_reference"] = "other:" + strings.Repeat("a", 64)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			bad := recoveryClone(t, value)
			mutate(bad)
			recoveryValidate(t, op, request, bad, false)
		})
	}
	badRequest := recoveryClone(t, request)
	badRequest["retained_source"].(map[string]any)["delivery"].(map[string]any)["processed_ack"] = "dataset_admitted"
	bad := recoveryClone(t, value)
	bad["request_sha256"] = recoveryHash(t, badRequest)
	recoveryValidate(t, op, badRequest, bad, false)
}

func TestSBVRecoveryNativeCanonicalAndNormalFallback(t *testing.T) {
	// This exact byte string is independent of either JSON encoder's HTML and
	// U+2028/U+2029 policies. Literal backslash-u is distinct from the code point.
	request := map[string]any{"text": "<>&\u2028\u2029\\u2028\n\t"}
	expected := []byte("{\"text\":\"<>&\u2028\u2029\\\\u2028\\n\\t\"}")
	actual, err := sbvNativeCanonical(request)
	if err != nil || string(actual) != string(expected) {
		t.Fatalf("native JSON mismatch: %q error=%v", actual, err)
	}
	for _, raw := range []string{`{"status":"completed"}`, `{"protocol":"symphony.sbv.run.v1"}`} {
		handled, err := ValidateSBVRecovery("source_retain", request, []byte(raw))
		if handled || err != nil {
			t.Fatal("ordinary result consumed by recovery branch", handled, err)
		}
	}
	_, input, value := recoveryFixture(t, false)
	recoveryValidate(t, "run", input, value, false)
	if _, err := ValidateSBVRecovery("source_retain", input, []byte(`{"status":"recovery_required",`)); err == nil {
		t.Fatal("malformed response accepted")
	}
}

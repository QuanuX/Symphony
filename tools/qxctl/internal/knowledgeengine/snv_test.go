package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func snvRepository(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return path
}
func snvReadObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	object, err := ParseSNVObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	return object
}
func snvEncode(t *testing.T, object any) []byte {
	t.Helper()
	raw, err := SCVCanonical(object)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func snvClone(t *testing.T, object map[string]any) map[string]any {
	t.Helper()
	result, err := ParseSNVObject(snvEncode(t, object))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSNVObjectStrictBoundsAndUnicode(t *testing.T) {
	for _, raw := range []string{`{"x":1,"x":2}`, `{"x":1.0}`, `{"x":9007199254740992}`, `{"x":"\ud800"}`, `{"x":"\udfff"}`, `{} {}`, `[]`, "{\"x\":\"\xff\"}"} {
		if _, err := ParseSNVObject([]byte(raw)); err == nil {
			t.Fatalf("Accepted invalid SNV JSON %q", raw)
		}
	}
	input, err := ParseSNVObject([]byte(`{"x":-0,"large":"18446744073709551615","unicode":"a\u2028b\u2029c","literal":"\\u2028"}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded := snvEncode(t, input)
	if !strings.Contains(string(encoded), "a\u2028b\u2029c") || !strings.Contains(string(encoded), `"x":0`) || !strings.Contains(string(encoded), `"literal":"\\u2028"`) {
		t.Fatal("Exact native JSON representation changed")
	}
	document := []byte(`{"items":[` + strings.Repeat("0,", 32768) + `0]}`)
	if _, err := ParseSNVObject(document); err != nil {
		t.Fatal("SNV explicit larger JSON profile rejected", err)
	}
	above := []byte(`{"items":[` + strings.Repeat("0,", snvJSONValues) + `0]}`)
	if _, err := ParseSNVObject(above); err == nil {
		t.Fatal("SNV value bound ignored")
	}
}

func TestSNVResultRejectsUnboundOwner(t *testing.T) {
	for _, owner := range []string{"sniv", "snrv"} {
		t.Run(owner, func(t *testing.T) {
			operation := map[string]string{"sniv": "identity_validate", "snrv": "resources_validate"}[owner]
			module := filepath.Join(snvRepository(t), "modules", owner+"-engine")
			input := snvReadObject(t, filepath.Join(module, "tests", "fixtures", operation+".json"))
			collection := snvReadObject(t, filepath.Join(module, "schemas", "v1", "admin.schema.json"))
			contract := map[string]any{"input": collection["requests"].(map[string]any)[operation], "result": collection["results"].(map[string]any)[operation], "$defs": collection["$defs"]}
			payload := snvEncode(t, input)
			result := map[string]any{"protocol": "symphony." + owner + "." + strings.ReplaceAll(operation, "_", "-") + ".v1", "owner": owner, "owner_version": "0.1.0-dev", "source_digest": digestBytes(payload), "subject_ids": []any{"node-1"}, "status": "valid", "proposed_records": []any{}, "findings": []any{}}
			if owner == "sniv" {
				result["incarnation_ids"] = []any{"incarnation-1"}
				result["records"] = map[string]any{"physical_records": input["physical_records"], "participations": input["participations"]}
			} else {
				result["records"] = input["inventories"]
				result["resource_changes"] = []any{}
				result["identity_handoff"] = map[string]any{"required": false, "authority": "sniv", "physical_identity_changed": false, "subject_rebind": nil}
			}
			if err := ValidateSNVResult(owner, operation, payload, snvEncode(t, result), snvEncode(t, contract)); err != nil {
				t.Fatal("Valid native result correspondence rejected", err)
			}
			for _, mutation := range []struct {
				key   string
				value any
			}{{"owner", "foreign"}, {"owner_version", "0.2.0-dev"}, {"source_digest", "sha256:" + strings.Repeat("a", 64)}, {"subject_ids", []any{"invented-node"}}, {"subject_ids", []any{"node-1", "node-1"}}, {"protocol", "unsupported"}, {"extra", true}} {
				changed := snvClone(t, result)
				changed[mutation.key] = mutation.value
				if err := ValidateSNVResult(owner, operation, payload, snvEncode(t, changed), snvEncode(t, contract)); err == nil {
					t.Fatalf("Accepted changed native result %s", mutation.key)
				}
			}
		})
	}
}

func TestSNVDescriptorRejectsUnsupportedOperation(t *testing.T) {
	declaration := snvReadObject(t, filepath.Join(snvRepository(t), "modules/sniv-engine/OWNER-INTERFACE.json"))
	descriptor := map[string]any{"protocol": "symphony.knowledge.engine-descriptor.v2", "format_version": int64(2), "module_id": "sniv-engine", "engine_id": "symphony-sniv", "vector_id": "sniv", "engine_version": "0.1.0-dev", "process_protocols": []any{processProtocol}, "contract_versions": declaration["contract_versions"], "operations": declaration["operations"], "embedded_dependencies": declaration["embedded_dependencies"], "limits": map[string]any{"request_bytes": int64(maxRequestBytes), "response_bytes": int64(maxResponseBytes), "json_depth": int64(64), "json_values": int64(snvJSONValues), "records": int64(2048), "deadline_ahead_ms": int64(300000)}, "supported_scopes": []any{"user"}, "language": "C++26", "thermal_path": "freezing", "canonical_apply_enabled": false, "session_mutation_enabled": false, "network_listener": false}
	descriptor["descriptor_digest"] = digestBytes(snvEncode(t, descriptor))
	contract := snvEncode(t, declaration)
	if err := ValidateSNVDescriptor("sniv", "0.1.0-dev", snvEncode(t, descriptor), contract); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(map[string]any){func(m map[string]any) { m["operations"].([]any)[0].(map[string]any)["operation_name"] = "invented" }, func(m map[string]any) { m["operations"].([]any)[0].(map[string]any)["input_protocol"] = "future" }, func(m map[string]any) { m["network_listener"] = true }, func(m map[string]any) {
		m["embedded_dependencies"] = []any{map[string]any{"engine_id": "other", "version": "9"}}
	}, func(m map[string]any) { m["limits"].(map[string]any)["records"] = int64(4096) }} {
		changed := snvClone(t, descriptor)
		delete(changed, "descriptor_digest")
		mutation(changed)
		changed["descriptor_digest"] = digestBytes(snvEncode(t, changed))
		if err := ValidateSNVDescriptor("sniv", "0.1.0-dev", snvEncode(t, changed), contract); err == nil {
			t.Fatal("Accepted forged internally sealed native descriptor")
		}
	}
}

func TestSNVParentSubjectProjectionRejectsUnboundIDs(t *testing.T) {
	input := snvReadObject(t, filepath.Join(snvRepository(t), "modules/sniv-engine/tests/fixtures/identity_validate.json"))
	source := snvEncode(t, input)
	bundle := map[string]any{"bundle_id": "projection-fixture", "artifacts": []any{map[string]any{"owner": "sniv", "owner_version": "0.1.0-dev", "source_utf8": string(source), "source_digest": digestBytes(source)}}}
	request := map[string]any{"bundle": bundle}
	result := map[string]any{"owner": "symphony-snv", "bundle_id": bundle["bundle_id"], "bundle_digest": digestBytes(snvEncode(t, bundle)), "views": map[string]any{"sniv": map[string]any{"owner": "sniv", "owner_version": "0.1.0-dev", "source_digest": digestBytes(source), "subject_ids": []any{"node-1"}}}, "subject_ids": map[string]any{"sniv": []any{"incarnation-1", "node-1"}}}
	if err := snvParentCorrespondence("snv_inspect", request, result); err != nil {
		t.Fatal("Native typed projection from supplied identifiers rejected", err)
	}
	for _, subjects := range [][]any{{"incarnation-1"}, {"node-1", "invented-subject"}, {"node-1", "node-1"}} {
		changed := snvClone(t, result)
		changed["subject_ids"].(map[string]any)["sniv"] = subjects
		if err := snvParentCorrespondence("snv_inspect", request, changed); err == nil {
			t.Fatal("Accepted omitted, invented, or duplicate parent subject projection")
		}
	}
	changed := snvClone(t, result)
	changed["subject_ids"].(map[string]any)["unknown-owner"] = []any{}
	if err := snvParentCorrespondence("snv_inspect", request, changed); err == nil {
		t.Fatal("Accepted extra parent subject owner")
	}
}

func snvCopyInstallation(t *testing.T, source string) string {
	t.Helper()
	target := t.TempDir()
	if err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("Unexpected installation entry %s", relative)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, raw, info.Mode().Perm())
	}); err != nil {
		t.Fatal(err)
	}
	return target
}
func TestSNVInterfaceResourcesRejectTampering(t *testing.T) {
	prefix := os.Getenv("SNV_TEST_PREFIX")
	if prefix == "" {
		t.Skip("SNV_TEST_PREFIX selects actual installed package gate")
	}
	for _, owner := range []string{"sniv", "snrv", "sciv", "scnv", "snv"} {
		if _, err := InspectSNV(prefix, "0.1.0-dev", owner); err != nil {
			t.Fatal(owner, err)
		}
	}
	for _, relative := range []string{"share/symphony/sniv-engine/0.1.0-dev/schemas/v1/admin.templates.json", "share/symphony/sniv-engine/0.1.0-dev/OWNER-INTERFACE.json", "libexec/symphony/sniv-engine/0.1.0-dev/symphony-sniv"} {
		copy := snvCopyInstallation(t, prefix)
		path := filepath.Join(copy, relative)
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.WriteString("\nchanged"); err != nil {
			t.Fatal(err)
		}
		file.Close()
		if _, err := InspectSNV(copy, "0.1.0-dev", "sniv"); err == nil {
			t.Fatal("Accepted tampered receipt-owned SNV installation", relative)
		}
	}
	copy := snvCopyInstallation(t, prefix)
	path := filepath.Join(copy, "share/symphony/sniv-engine/0.1.0-dev/schemas/v1/admin.templates.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "template")
	if err = os.WriteFile(outside, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSNV(copy, "0.1.0-dev", "sniv"); err == nil {
		t.Fatal("Accepted symlinked receipt-owned schema")
	}
}
func TestSNVInstalledConsumer(t *testing.T) {
	prefix := os.Getenv("SNV_TEST_PREFIX")
	if prefix == "" {
		t.Skip("SNV_TEST_PREFIX selects actual installed package gate")
	}
	for _, owner := range []string{"sniv", "snrv", "sciv", "scnv", "snv"} {
		if _, err := InvokeSNV(context.Background(), prefix, "0.1.0-dev", snvRepository(t), owner, "descriptor", []byte("{}")); err != nil {
			t.Fatal(owner, "descriptor", err)
		}
		if _, err := InspectSNV(prefix, "latest", owner); err == nil {
			t.Fatal("Floating SNV release accepted")
		}
	}
	for _, op := range SNVOperations {
		// The optional collector has its own installation selector and gate.
		// A core-only prefix must remain sufficient for the five core owners.
		if op.Owner == "local-observer" {
			continue
		}
		t.Run(op.Owner+"/"+op.Operation+"/resources", func(t *testing.T) {
			for _, templates := range []bool{false, true} {
				_, raw, err := SNVResource(prefix, "0.1.0-dev", op.Owner, op.Operation, templates)
				if err != nil || !json.Valid(raw) {
					t.Fatal(op.Operation, "resource", err)
				}
			}
		})
	}
	invoke := func(t *testing.T, owner, operation string, input map[string]any) map[string]any {
		t.Helper()
		response, err := InvokeSNV(context.Background(), prefix, "0.1.0-dev", snvRepository(t), owner, operation, snvEncode(t, input))
		if err != nil {
			t.Fatal(owner, operation, err)
		}
		result, err := ParseSNVObject(response.Result)
		if err != nil {
			t.Fatal(owner, operation, "result", err)
		}
		return result
	}
	inspection := snvReadObject(t, filepath.Join(snvRepository(t), "modules/snv-engine/tests/fixtures/snv_inspect.json"))
	bundle := inspection["bundle"].(map[string]any)
	for _, raw := range bundle["artifacts"].([]any) {
		artifact := raw.(map[string]any)
		owner, operation := artifact["owner"].(string), artifact["operation"].(string)
		t.Run(owner+"/"+operation, func(t *testing.T) {
			input, err := ParseSNVObject([]byte(artifact["source_utf8"].(string)))
			if err != nil {
				t.Fatal(err)
			}
			invoke(t, owner, operation, input)
		})
	}
	t.Run("sciv/sciv_transition", func(t *testing.T) {
		invoke(t, "sciv", "sciv_transition", snvReadObject(t, filepath.Join(snvRepository(t), "modules/sciv-engine/schemas/v1/sciv-transition.template.json")))
	})
	t.Run("scnv/names_resolve", func(t *testing.T) {
		invoke(t, "scnv", "names_resolve", snvReadObject(t, filepath.Join(snvRepository(t), "modules/scnv-engine/tests/fixtures/names_resolve.json")))
	})
	t.Run("snv/native_owner_results_and_exports", func(t *testing.T) {
		invoke(t, "snv", "snv_inspect", inspection)
		for _, mode := range []string{"projection", "history", "diff"} {
			input := snvReadObject(t, filepath.Join(snvRepository(t), "modules/snv-engine/tests/fixtures/snv_inspect.json"))
			input["mode"] = mode
			if mode == "diff" {
				input["baseline"] = bundle
			}
			invoke(t, "snv", "snv_inspect", input)
		}
		inspection["mode"] = "export_manifest"
		manifest := invoke(t, "snv", "snv_inspect", inspection)["export_manifest"].(map[string]any)
		chunks := []any{}
		inspection["mode"] = "export_chunk"
		for i := range manifest["chunks"].([]any) {
			inspection["offset"] = int64(i)
			chunks = append(chunks, invoke(t, "snv", "snv_inspect", inspection)["export_chunk"])
		}
		for _, mode := range []string{"bundle", "import"} {
			input := map[string]any{"protocol": "symphony.snv.evidence-plan-input.v1", "operation_id": "consumer-evidence", "mode": mode, "bundle": bundle, "export_records": nil}
			if mode == "import" {
				input["bundle"] = nil
				input["export_records"] = map[string]any{"manifest": manifest, "chunks": chunks}
			}
			invoke(t, "snv", "snv_evidence_plan", input)
		}
		input := map[string]any{"protocol": "symphony.snv.state-plan-input.v1", "view": map[string]any{"tops_id": "consumer-tops", "view_id": "consumer-view"}, "operation_id": "consumer-select", "expected_state_digest": nil, "prior_head": nil, "change_kind": "select", "bundle": bundle, "reason": "installed consumer gate", "migration": nil}
		plan := invoke(t, "snv", "snv_state_plan", input)
		invoke(t, "snv", "snv_state_reduce", map[string]any{"protocol": "symphony.snv.state-reduce-input.v1", "input": input, "plan": plan})
	})
}

func TestSNVExportConsumerBindsChunks(t *testing.T) {
	// SHA-256("abc") is a published fixed representation vector; each digest
	// names bytes, while the surrounding manifest has a separate JSON seal.
	const rawDigest = "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	manifest := map[string]any{"protocol": "symphony.snv.export-manifest.v1", "writer": "symphony-snv", "writer_version": "0.1.0-dev", "bundle_digest": rawDigest, "byte_count": int64(3), "encoding": "hex", "chunk_size": int64(16384), "chunks": []any{map[string]any{"index": int64(0), "byte_count": int64(3), "digest": rawDigest}}, "embedded_dependencies": []any{}}
	manifest["digest"] = digestBytes(snvEncode(t, manifest))
	if err := snvVerifySeals(manifest, 0); err != nil {
		t.Fatal("Raw chunk digests mistaken for JSON seals", err)
	}
	if err := snvExportCorrespondence([]byte("abc"), manifest, nil); err != nil {
		t.Fatal(err)
	}
	changed := snvClone(t, manifest)
	changed["chunks"].([]any)[0].(map[string]any)["digest"] = "sha256:" + strings.Repeat("a", 64)
	delete(changed, "digest")
	changed["digest"] = digestBytes(snvEncode(t, changed))
	if err := snvVerifySeals(changed, 0); err != nil {
		t.Fatal(err)
	}
	if err := snvExportCorrespondence([]byte("abc"), changed, nil); err == nil {
		t.Fatal("Self-consistent manifest seal bypassed wrong chunk bytes")
	}
	chunk := map[string]any{"protocol": "symphony.snv.export-chunk.v1", "manifest_digest": manifest["digest"], "index": int64(0), "hex": "616263", "byte_count": int64(3), "bundle_digest": rawDigest, "digest": rawDigest}
	if err := snvVerifySeals(chunk, 0); err != nil {
		t.Fatal("Raw exported chunk mistaken for a JSON seal", err)
	}
	if err := snvExportCorrespondence([]byte("abc"), nil, chunk); err != nil {
		t.Fatal(err)
	}
	chunk["hex"] = "616264"
	if err := snvExportCorrespondence([]byte("abc"), nil, chunk); err == nil {
		t.Fatal("Wrong export chunk bytes admitted")
	}
	chunk["hex"] = "616263"
	chunk["digest"] = "sha256:" + strings.Repeat("0", 64)
	if err := snvExportCorrespondence([]byte("abc"), nil, chunk); err == nil {
		t.Fatal("Wrong raw chunk digest admitted")
	}
}

func TestSNVSchemaPrimitiveTypeUnions(t *testing.T) {
	rule := map[string]any{"type": []any{"integer", "null"}, "minimum": int64(0), "maximum": int64(9007199254740991)}
	for _, value := range []any{nil, int64(0), int64(9007199254740991)} {
		if !snvTransportShape(rule, value, rule, 0) {
			t.Fatal("Admitted nullable integer rejected", value)
		}
	}
	for _, value := range []any{int64(-1), int64(9007199254740992), "0", false} {
		if snvTransportShape(rule, value, rule, 0) {
			t.Fatal("Invalid nullable integer admitted", value)
		}
	}
	for _, types := range [][]any{{}, {"integer", "integer"}, {"integer", "unknown"}} {
		invalid := map[string]any{"type": types}
		if snvTransportShape(invalid, int64(0), invalid, 0) {
			t.Fatal("Unsupported type union admitted")
		}
	}
}

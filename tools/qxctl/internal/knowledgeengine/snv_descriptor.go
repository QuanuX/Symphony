package knowledgeengine

import (
	"context"
	"fmt"
	"reflect"
)

func invokeSNVDescriptor(ctx context.Context, prefix, version, cwd, owner string, payload []byte) (Response, error) {
	input, err := snvObject(payload, maxRequestBytes)
	if err != nil || len(input) != 0 {
		return Response{}, fmt.Errorf("SNV descriptor requires an empty payload")
	}
	inst, err := InspectSNV(prefix, version, owner)
	if err != nil {
		return Response{}, err
	}
	declaration, err := snvOwnedResource(inst, "share/symphony/"+inst.ModuleID+"/"+version+"/OWNER-INTERFACE.json", "")
	if err != nil {
		return Response{}, err
	}
	result, invokeErr := invokeResolved(ctx, snvEngineSpec(owner), inst.ExecutablePath, version, cwd, "descriptor", []byte("{}"))
	after, err := InspectSNV(prefix, version, owner)
	if err != nil || after != inst {
		return Response{}, fmt.Errorf("SNV installation changed during descriptor invocation")
	}
	if invokeErr != nil {
		return Response{}, invokeErr
	}
	if err := ValidateSNVDescriptor(owner, version, result.Result, declaration); err != nil {
		return Response{}, err
	}
	return result, nil
}

// ValidateSNVDescriptor checks the native descriptor against the exact compiled
// declaration already admitted from receipt-owned resources.
func ValidateSNVDescriptor(owner, version string, result, declaration []byte) error {
	descriptor, err := snvObject(result, maxResponseBytes)
	if err != nil {
		return err
	}
	contract, err := snvObject(declaration, maxRequestBytes)
	if err != nil {
		return err
	}
	fields := []string{"protocol", "format_version", "module_id", "engine_id", "vector_id", "engine_version", "process_protocols", "contract_versions", "operations", "embedded_dependencies", "limits", "supported_scopes", "language", "thermal_path", "canonical_apply_enabled", "session_mutation_enabled", "network_listener", "descriptor_digest"}
	if len(descriptor) != len(fields) {
		return fmt.Errorf("SNV descriptor fields differ from exact contract")
	}
	for _, key := range fields {
		if _, okay := descriptor[key]; !okay {
			return fmt.Errorf("SNV descriptor field missing")
		}
	}
	spec := snvEngineSpec(owner)
	var expectedVector any = owner
	recordLimit := int64(2048)
	if owner == "local-observer" {
		expectedVector = nil
		recordLimit = 4
	}
	if descriptor["protocol"] != "symphony.knowledge.engine-descriptor.v2" || descriptor["format_version"] != int64(2) || descriptor["module_id"] != spec.moduleID || descriptor["engine_id"] != spec.engineID || descriptor["vector_id"] != expectedVector || descriptor["engine_version"] != version || descriptor["language"] != "C++26" || descriptor["thermal_path"] != "freezing" || descriptor["canonical_apply_enabled"] != false || descriptor["session_mutation_enabled"] != false || descriptor["network_listener"] != false || !reflect.DeepEqual(descriptor["process_protocols"], []any{processProtocol}) || !reflect.DeepEqual(descriptor["supported_scopes"], []any{"user"}) {
		return fmt.Errorf("SNV descriptor identity or execution scope mismatch")
	}
	expectedOperations, okay := contract["operations"].([]any)
	if !okay {
		return fmt.Errorf("SNV compiled operation declarations are missing")
	}
	observed, okay := descriptor["operations"].([]any)
	if !okay || len(observed) != len(expectedOperations) {
		return fmt.Errorf("SNV descriptor operation census mismatch")
	}
	expected := map[string]any{}
	for _, value := range expectedOperations {
		operation, okay := value.(map[string]any)
		if !okay {
			return fmt.Errorf("Invalid SNV declaration operation")
		}
		name, okay := operation["operation_name"].(string)
		if !okay || expected[name] != nil {
			return fmt.Errorf("Invalid SNV declaration operation identity")
		}
		expected[name] = operation
	}
	seen := map[string]bool{}
	for _, value := range observed {
		operation, okay := value.(map[string]any)
		if !okay {
			return fmt.Errorf("Invalid native SNV operation")
		}
		name, okay := operation["operation_name"].(string)
		if !okay || seen[name] || expected[name] == nil || !reflect.DeepEqual(operation, expected[name]) {
			return fmt.Errorf("SNV descriptor differs from compiled operation admission")
		}
		seen[name] = true
	}
	limits, okay := descriptor["limits"].(map[string]any)
	if !okay || !reflect.DeepEqual(limits, map[string]any{"request_bytes": int64(maxRequestBytes), "response_bytes": int64(maxResponseBytes), "json_depth": int64(maxJSONDepth), "json_values": int64(snvJSONValues), "records": recordLimit, "deadline_ahead_ms": int64(300000)}) {
		return fmt.Errorf("SNV descriptor limits mismatch")
	}
	if !reflect.DeepEqual(descriptor["embedded_dependencies"], contract["embedded_dependencies"]) {
		return fmt.Errorf("SNV embedded dependency declaration mismatch")
	}
	if !snvSameTextSet(descriptor["contract_versions"], contract["contract_versions"]) {
		return fmt.Errorf("SNV descriptor contract versions mismatch")
	}
	claimed := descriptor["descriptor_digest"]
	delete(descriptor, "descriptor_digest")
	canonical, err := SCVCanonical(descriptor)
	if err != nil || claimed != digestBytes(canonical) {
		return fmt.Errorf("SNV descriptor digest mismatch")
	}
	return nil
}
func snvSameTextSet(a, b any) bool {
	left, leftOkay := a.([]any)
	right, rightOkay := b.([]any)
	if !leftOkay || !rightOkay || len(left) != len(right) || !snvUniqueTexts(left) || !snvUniqueTexts(right) {
		return false
	}
	seen := map[string]bool{}
	for _, value := range left {
		seen[value.(string)] = true
	}
	for _, value := range right {
		if !seen[value.(string)] {
			return false
		}
	}
	return true
}

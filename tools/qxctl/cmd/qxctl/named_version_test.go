package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNamedVersionCommandGrammar(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"propose", "seal", "alias", "lookup", "status", "recover"} {
		command, _, err := root.Find([]string{"sav", "named-version", operation})
		if err != nil || command == nil || command.Name() != operation {
			t.Fatalf("Named Version command %s is not reachable: %v", operation, err)
		}
	}
}

func TestNamedVersionResourceBindsEveryInput(t *testing.T) {
	basePayload := map[string]any{
		"tops_id": "tops", "operation": "named_version_prepare", "expected_registry_digest": "absent",
		"named_version":         map[string]any{"named_version_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		"prepared_operation_id": nil, "proposal_digest": nil, "alias": nil, "selector": nil,
	}
	base := namedVersionResource(basePayload)
	if !strings.HasPrefix(base, "symphony.knowledge.named-version:") || len(base) != 97 ||
		base != namedVersionResource(basePayload) {
		t.Fatalf("Named Version resource is unstable: %q", base)
	}
	for field, replacement := range map[string]any{
		"tops_id": "other", "operation": "named_version_seal", "expected_registry_digest": "discover",
		"prepared_operation_id": "prepare-1", "proposal_digest": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"alias": "stable", "selector": map[string]any{"kind": "alias", "value": "stable"},
	} {
		variant := map[string]any{}
		for key, value := range basePayload {
			variant[key] = value
		}
		variant[field] = replacement
		if namedVersionResource(variant) == base {
			t.Fatalf("Named Version resource did not bind %s", field)
		}
	}
	variant := map[string]any{}
	for key, value := range basePayload {
		variant[key] = value
	}
	variant["named_version"] = map[string]any{"named_version_digest": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}
	if namedVersionResource(variant) == base {
		t.Fatal("Named Version resource did not bind the candidate digest")
	}
}

func TestValidateNamedVersionResultRejectsAuthorityAndDigestDrift(t *testing.T) {
	result := map[string]any{
		"protocol": namedVersionResultProtocol, "format_version": 1,
		"operation": "named_version_status",
		"compatibility": map[string]any{
			"mode": "full", "process_protocol": "symphony.knowledge.engine-process.v1",
			"registry_read_version": 1, "registry_write_version": 1, "missing_capabilities": []string{},
		},
		"registry_present": false, "registry_digest": nil, "version_count": 0, "alias_count": 0,
		"proposal_digest": nil, "artifact": nil, "selected_alias": nil,
		"changed": false, "recovered": false, "repair_actions": []string{}, "read_only": true,
		"canonical_apply_enabled": false, "canonical": false, "stav_append_enabled": false,
	}
	digest, err := maintenanceObjectDigest(result, "result_digest")
	if err != nil {
		t.Fatal(err)
	}
	result["result_digest"] = digest
	encoded, _ := json.Marshal(result)
	if _, err := validateNamedVersionResult(encoded, "named_version_status"); err != nil {
		t.Fatal(err)
	}
	result["canonical_apply_enabled"] = true
	encoded, _ = json.Marshal(result)
	if _, err := validateNamedVersionResult(encoded, "named_version_status"); err == nil {
		t.Fatal("canonical apply escalation was accepted")
	}
	result["canonical_apply_enabled"] = false
	result["stav_append_enabled"] = true
	encoded, _ = json.Marshal(result)
	if _, err := validateNamedVersionResult(encoded, "named_version_status"); err == nil {
		t.Fatal("unratified STAV append escalation was accepted")
	}
	result["stav_append_enabled"] = false
	result["result_digest"] = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	encoded, _ = json.Marshal(result)
	if _, err := validateNamedVersionResult(encoded, "named_version_status"); err == nil {
		t.Fatal("result digest drift was accepted")
	}
}

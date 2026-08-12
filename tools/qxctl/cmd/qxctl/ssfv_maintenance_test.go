package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSSFVMaintenanceCommandGrammar(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"begin", "status", "checkpoint", "close", "recover"} {
		command, _, err := root.Find([]string{"knowledge", "session", "features", operation})
		if err != nil || command == nil || command.Name() != operation {
			t.Fatalf("maintenance command %s is not reachable: %v", operation, err)
		}
	}
}

func TestSSFVMaintenanceResourceBindsEveryInput(t *testing.T) {
	base := ssfvMaintenanceResource("tops", "/repo", "ssfv_maintenance_begin", "absent",
		"session", "snapshot", "inventory")
	if base != ssfvMaintenanceResource("tops", "/repo", "ssfv_maintenance_begin", "absent",
		"session", "snapshot", "inventory") ||
		!strings.HasPrefix(base, "symphony.knowledge.ssfv-maintenance:") || len(base) != 100 {
		t.Fatalf("maintenance resource is not stable: %q", base)
	}
	variants := []string{
		ssfvMaintenanceResource("other", "/repo", "ssfv_maintenance_begin", "absent", "session", "snapshot", "inventory"),
		ssfvMaintenanceResource("tops", "/other", "ssfv_maintenance_begin", "absent", "session", "snapshot", "inventory"),
		ssfvMaintenanceResource("tops", "/repo", "ssfv_maintenance_close", "absent", "session", "snapshot", "inventory"),
		ssfvMaintenanceResource("tops", "/repo", "ssfv_maintenance_begin", "other", "session", "snapshot", "inventory"),
		ssfvMaintenanceResource("tops", "/repo", "ssfv_maintenance_begin", "absent", "other", "snapshot", "inventory"),
		ssfvMaintenanceResource("tops", "/repo", "ssfv_maintenance_begin", "absent", "session", "other", "inventory"),
		ssfvMaintenanceResource("tops", "/repo", "ssfv_maintenance_begin", "absent", "session", "snapshot", "other"),
	}
	for _, variant := range variants {
		if variant == base {
			t.Fatal("resource did not bind one of its authority inputs")
		}
	}
}

func TestMaintenanceDigestCanonicalizesNestedObjects(t *testing.T) {
	left := map[string]any{"z": map[string]any{"b": 2, "a": 1}, "a": true}
	right := map[string]any{"a": true, "z": map[string]any{"a": 1, "b": 2}}
	first, err := maintenanceObjectDigest(left, "evidence_digest")
	if err != nil {
		t.Fatal(err)
	}
	second, err := maintenanceObjectDigest(right, "evidence_digest")
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !validTaggedDigest(first) {
		t.Fatalf("nested canonicalization is unstable: %s != %s", first, second)
	}
}

func TestValidateSSFVMaintenanceResultRejectsApplyAndDigestDrift(t *testing.T) {
	journal := map[string]any{
		"protocol": "symphony.knowledge.ssfv-maintenance-journal.v1",
		"state":    "open", "review_state": "current", "canonical": false,
	}
	digest, err := maintenanceObjectDigest(journal, "journal_digest")
	if err != nil {
		t.Fatal(err)
	}
	journal["journal_digest"] = digest
	result := map[string]any{
		"protocol": ssfvMaintenanceResultProtocol, "format_version": 1,
		"operation": "ssfv_maintenance_status", "compatibility": map[string]any{"mode": "full"},
		"journal_present": true, "journal": journal, "journal_digest": digest,
		"effective_state": "open", "review_state": "current", "changed": false,
		"recovered": false, "repair_actions": []string{}, "read_only": true,
		"canonical_apply_enabled": false, "canonical": false,
	}
	encoded, _ := json.Marshal(result)
	if _, err := validateSSFVMaintenanceResult(encoded, "ssfv_maintenance_status"); err != nil {
		t.Fatal(err)
	}
	result["canonical_apply_enabled"] = true
	encoded, _ = json.Marshal(result)
	if _, err := validateSSFVMaintenanceResult(encoded, "ssfv_maintenance_status"); err == nil {
		t.Fatal("canonical apply escalation was accepted")
	}
	result["canonical_apply_enabled"] = false
	journal["state"] = "closed"
	encoded, _ = json.Marshal(result)
	if _, err := validateSSFVMaintenanceResult(encoded, "ssfv_maintenance_status"); err == nil {
		t.Fatal("journal digest drift was accepted")
	}
}

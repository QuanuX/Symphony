package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func validReconciliationResult(t *testing.T, operation string, present bool) json.RawMessage {
	t.Helper()
	var journal any
	var digest any
	if present {
		digest = "sha256:" + strings.Repeat("1", 64)
		journal = map[string]any{
			"protocol":       "symphony.knowledge.reconciliation-journal.v1",
			"journal_digest": digest,
			"canonical":      false,
		}
	}
	readOnly := operation == "compatibility" || operation == "status"
	value := map[string]any{
		"protocol":  "symphony.knowledge.reconciliation-result.v1",
		"operation": operation,
		"compatibility": map[string]any{
			"mode":                  "full",
			"process_protocol":      "symphony.knowledge.engine-process.v1",
			"journal_read_version":  1,
			"journal_write_version": 1,
			"shared_capabilities":   []string{"content-snapshot-v1"},
			"missing_capabilities":  []string{},
			"reasons":               []string{"compatible"},
		},
		"journal_present":         present,
		"journal":                 journal,
		"journal_digest":          digest,
		"changed":                 false,
		"recovered":               false,
		"repair_actions":          []string{},
		"read_only":               readOnly,
		"canonical_apply_enabled": false,
		"canonical":               false,
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestValidateReconciliationResult(t *testing.T) {
	for _, operation := range []string{
		"compatibility", "begin", "status", "checkpoint", "close", "recover",
	} {
		present := operation != "compatibility"
		if _, err := validateReconciliationResult(
			validReconciliationResult(t, operation, present), operation); err != nil {
			t.Fatalf("%s valid result rejected: %v", operation, err)
		}
	}
}

func TestValidateReconciliationResultRejectsUnsafeOrInconsistentState(t *testing.T) {
	base := validReconciliationResult(t, "status", true)
	var value map[string]any
	if err := json.Unmarshal(base, &value); err != nil {
		t.Fatal(err)
	}
	value["canonical"] = true
	data, _ := json.Marshal(value)
	if _, err := validateReconciliationResult(data, "status"); err == nil {
		t.Fatal("canonical reconciliation result was accepted")
	}

	if err := json.Unmarshal(base, &value); err != nil {
		t.Fatal(err)
	}
	value["journal_digest"] = "sha256:" + strings.Repeat("2", 64)
	data, _ = json.Marshal(value)
	if _, err := validateReconciliationResult(data, "status"); err == nil {
		t.Fatal("mismatched journal digest was accepted")
	}

	if err := json.Unmarshal(base, &value); err != nil {
		t.Fatal(err)
	}
	value["unexpected"] = true
	data, _ = json.Marshal(value)
	if _, err := validateReconciliationResult(data, "status"); err == nil {
		t.Fatal("unknown reconciliation result field was accepted")
	}

	if err := json.Unmarshal(base, &value); err != nil {
		t.Fatal(err)
	}
	value["repair_actions"] = []string{"unsafe\nrepair"}
	data, _ = json.Marshal(value)
	if _, err := validateReconciliationResult(data, "status"); err == nil {
		t.Fatal("unsafe repair guidance was accepted")
	}
}

package main

import (
	"encoding/json"
	"testing"
)

func TestValidateSKVIResultHonorsVersionedEntryCapacity(t *testing.T) {
	for _, item := range []struct {
		version   string
		operation string
		protocol  string
		limit     int
	}{
		{version: "0.1.0-dev", operation: "check", protocol: "symphony.skvi.check-result.v1", limit: 1024},
		{version: "0.2.0-dev", operation: "check", protocol: "symphony.skvi.check-result.v2", limit: 2048},
		{version: "0.1.0-dev", operation: "project", protocol: "symphony.skvi.projection.v1", limit: 1024},
		{version: "0.2.0-dev", operation: "project", protocol: "symphony.skvi.projection.v2", limit: 2048},
	} {
		t.Run(item.protocol, func(t *testing.T) {
			makeResult := func(count int) json.RawMessage {
				var result map[string]any
				if item.operation == "check" {
					result = map[string]any{
						"protocol": item.protocol, "entries_checked": count,
						"read_only": true, "canonical_apply_enabled": false,
						"summary": map[string]any{"state": "valid", "violation": 0},
					}
				} else {
					entries := make([]map[string]any, count)
					for i := range entries {
						entries[i] = map[string]any{}
					}
					result = map[string]any{
						"protocol": item.protocol, "module_id": "skvi-engine",
						"engine_id": "symphony-skvi", "vector_id": "skvi",
						"entry_count": count, "entries": entries,
						"projection_digest": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
						"noncanonical":      true, "rebuildable": true,
					}
				}
				encoded, err := json.Marshal(result)
				if err != nil {
					t.Fatal(err)
				}
				return encoded
			}
			if _, err := validateSKVIResult(item.version, item.operation, makeResult(item.limit)); err != nil {
				t.Fatalf("accepted ceiling %d rejected: %v", item.limit, err)
			}
			if _, err := validateSKVIResult(item.version, item.operation, makeResult(item.limit+1)); err == nil {
				t.Fatalf("over-ceiling count %d accepted", item.limit+1)
			}
			otherVersion := "0.1.0-dev"
			if item.version == otherVersion {
				otherVersion = "0.2.0-dev"
			}
			if _, err := validateSKVIResult(otherVersion, item.operation, makeResult(item.limit)); err == nil {
				t.Fatalf("result protocol %s accepted for installed SKVI %s", item.protocol, otherVersion)
			}
		})
	}
}

func TestSKVICheckValidityIsPresentationIndependent(t *testing.T) {
	valid, err := skviCheckValid(json.RawMessage(`{"summary":{"state":"valid","violation":0}}`))
	if err != nil || !valid {
		t.Fatalf("valid result rejected: valid=%t err=%v", valid, err)
	}
	valid, err = skviCheckValid(json.RawMessage(`{"summary":{"state":"invalid","violation":2}}`))
	if err != nil || valid {
		t.Fatalf("invalid result accepted: valid=%t err=%v", valid, err)
	}
	if _, err := skviCheckValid(json.RawMessage(`{"summary":{}}`)); err == nil {
		t.Fatal("incomplete check result accepted")
	}
}

func TestPrintSKVIResultRejectsInvalidPlainCheck(t *testing.T) {
	result := json.RawMessage(`{
		"entries_checked":1,
		"relationships_checked":0,
		"index":{"digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		"summary":{"pass":1,"warning":0,"violation":1,"state":"invalid"}
	}`)
	if err := printSKVIResult("check", result); err == nil {
		t.Fatal("plain invalid check did not return an error")
	}
}

func TestValidateSKVIResultRejectsSafetyEscalation(t *testing.T) {
	missingSafetyAssertion := json.RawMessage(`{
		"readiness":"read_check_propose_project",
		"canonical_apply_enabled":false,
		"engine_decides_membership":false,
		"descriptor":{"engine_id":"symphony-skvi","canonical_apply_enabled":false,"network_listener":false}
	}`)
	if _, err := validateSKVIResult("0.1.0-dev", "inspect", missingSafetyAssertion); err == nil {
		t.Fatal("inspect result with an omitted safety assertion was accepted")
	}

	inspect := json.RawMessage(`{
		"readiness":"read_check_propose_project",
		"canonical_apply_enabled":true,
		"engine_decides_membership":false,
		"descriptor":{"engine_id":"symphony-skvi","canonical_apply_enabled":false,"session_mutation_enabled":false,"network_listener":false}
	}`)
	if _, err := validateSKVIResult("0.1.0-dev", "inspect", inspect); err == nil {
		t.Fatal("inspect result that enabled apply was accepted")
	}

	proposal := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1",
		"module_id":"skvi-engine",
		"engine_id":"symphony-skvi",
		"vector_id":"skvi",
		"proposal_id":"skvi-proposal:test",
		"proposal_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":true},
		"operations":[{}]
	}`)
	if _, err := validateSKVIResult("0.1.0-dev", "propose", proposal); err == nil {
		t.Fatal("self-ratified proposal was accepted")
	}

	projection := json.RawMessage(`{
		"protocol":"symphony.skvi.projection.v1",
		"module_id":"skvi-engine",
		"engine_id":"symphony-skvi",
		"vector_id":"skvi",
		"entry_count":0,
		"entries":[],
		"projection_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"noncanonical":false,
		"rebuildable":true
	}`)
	if _, err := validateSKVIResult("0.1.0-dev", "project", projection); err == nil {
		t.Fatal("projection claiming canonical status was accepted")
	}
}

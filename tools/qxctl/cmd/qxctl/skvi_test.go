package main

import (
	"encoding/json"
	"testing"
)

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
	if _, err := validateSKVIResult("inspect", missingSafetyAssertion); err == nil {
		t.Fatal("inspect result with an omitted safety assertion was accepted")
	}

	inspect := json.RawMessage(`{
		"readiness":"read_check_propose_project",
		"canonical_apply_enabled":true,
		"engine_decides_membership":false,
		"descriptor":{"engine_id":"symphony-skvi","canonical_apply_enabled":false,"session_mutation_enabled":false,"network_listener":false}
	}`)
	if _, err := validateSKVIResult("inspect", inspect); err == nil {
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
		"authority":{"caller_declared_operation":true,"engine_decided_membership":false,"ratified":true},
		"operations":[{}]
	}`)
	if _, err := validateSKVIResult("propose", proposal); err == nil {
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
	if _, err := validateSKVIResult("project", projection); err == nil {
		t.Fatal("projection claiming canonical status was accepted")
	}
}

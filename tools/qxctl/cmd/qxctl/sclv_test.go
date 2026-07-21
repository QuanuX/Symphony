package main

import (
	"encoding/json"
	"testing"
)

func TestSCLVCheckValidityIsPresentationIndependent(t *testing.T) {
	valid, err := sclvCheckValid(json.RawMessage(`{"summary":{"state":"valid","violation":0}}`))
	if err != nil || !valid {
		t.Fatalf("valid result rejected: valid=%t err=%v", valid, err)
	}
	valid, err = sclvCheckValid(json.RawMessage(`{"summary":{"state":"invalid","violation":1}}`))
	if err != nil || valid {
		t.Fatalf("invalid result accepted: valid=%t err=%v", valid, err)
	}
	if _, err := sclvCheckValid(json.RawMessage(`{"summary":{}}`)); err == nil {
		t.Fatal("incomplete check result accepted")
	}
}

func TestValidateSCLVResultRejectsAuthorityAndLifecycleEscalation(t *testing.T) {
	inspect := json.RawMessage(`{
		"read_only":true,
		"canonical_apply_enabled":false,
		"evidence_adapters":["symphony-sclv-evidence-local-git","symphony-sclv-evidence-airgap"],
		"descriptor":{"engine_id":"symphony-sclv","canonical_apply_enabled":false,"session_mutation_enabled":true,"network_listener":false}
	}`)
	if _, err := validateSCLVResult("inspect", inspect); err == nil {
		t.Fatal("inspect result that enabled session mutation was accepted")
	}

	proposal := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1",
		"module_id":"sclv-engine","engine_id":"symphony-sclv","vector_id":"sclv",
		"proposal_id":"sclv-proposal:test",
		"proposal_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_membership":false,"ratified":true},
		"operations":[{}]
	}`)
	if _, err := validateSCLVResult("propose", proposal); err == nil {
		t.Fatal("self-ratified proposal was accepted")
	}

	wrongTarget := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1",
		"module_id":"sclv-engine","engine_id":"symphony-sclv","vector_id":"sclv",
		"proposal_id":"sclv-proposal:test",
		"proposal_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_membership":false,"ratified":false},
		"write_set":[{"target_path":"README.md"}],
		"operations":[{"type":"append_record_v3","target_path":"README.md"}]
	}`)
	if _, err := validateSCLVResult("propose", wrongTarget); err == nil {
		t.Fatal("proposal targeting a noncanonical surface was accepted")
	}

	recovery := json.RawMessage(`{
		"protocol":"symphony.sclv.recovery-result.v1","action":"resume",
		"journal_mutated":true,"canonical_apply_enabled":false,"delete_recommended":false,
		"proposal":null,"result_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"
	}`)
	if _, err := validateSCLVResult("recover", recovery); err == nil {
		t.Fatal("recovery result claiming journal mutation was accepted")
	}

	deleteOnResume := json.RawMessage(`{
		"protocol":"symphony.sclv.recovery-result.v1","action":"resume",
		"journal_mutated":false,"canonical_apply_enabled":false,"delete_recommended":true,
		"proposal":null,"result_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"
	}`)
	if _, err := validateSCLVResult("recover", deleteOnResume); err == nil {
		t.Fatal("resumable recovery recommending journal deletion was accepted")
	}

	projection := json.RawMessage(`{
		"protocol":"symphony.sclv.projection.v1","module_id":"sclv-engine",
		"engine_id":"symphony-sclv","vector_id":"sclv","record_count":0,"records":[],
		"projection_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"noncanonical":false,"rebuildable":true
	}`)
	if _, err := validateSCLVResult("project", projection); err == nil {
		t.Fatal("projection claiming canonical status was accepted")
	}
}

func TestPrintSCLVResultRejectsInvalidPlainCheck(t *testing.T) {
	result := json.RawMessage(`{
		"records_checked":1,
		"ledger":{"digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		"summary":{"pass":1,"warning":0,"violation":1,"state":"invalid"}
	}`)
	if err := printSCLVResult("check", result); err == nil {
		t.Fatal("plain invalid SCLV check did not return an error")
	}
}

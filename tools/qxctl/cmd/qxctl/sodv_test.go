package main

import (
	"encoding/json"
	"testing"
)

func TestSODVCheckValidity(t *testing.T) {
	valid, err := sodvCheckValid(json.RawMessage(`{"summary":{"state":"valid","violations":0}}`))
	if err != nil || !valid {
		t.Fatalf("valid SODV check rejected: valid=%t err=%v", valid, err)
	}
	valid, err = sodvCheckValid(json.RawMessage(`{"summary":{"state":"invalid","violations":1}}`))
	if err != nil || valid {
		t.Fatalf("invalid SODV check accepted: valid=%t err=%v", valid, err)
	}
	if _, err := sodvCheckValid(json.RawMessage(`{"summary":{}}`)); err == nil {
		t.Fatal("incomplete SODV check accepted")
	}
}

func TestValidateSODVSafetyContracts(t *testing.T) {
	inspect := json.RawMessage(`{
		"engine_decides_release_truth":false,"caller_supplies_external_observations":true,
		"network_access":false,"canonical_apply_enabled":false,
		"descriptor":{"engine_id":"symphony-sodv","language":"C++26","thermal_path":"freezing",
		"provider_input":"caller_supplied","network_access":false,"canonical_apply_enabled":false,"network_listener":false}
	}`)
	if valid, err := validateSODVResult("inspect", inspect); err != nil || !valid {
		t.Fatalf("safe inspect rejected: %v", err)
	}

	verify := json.RawMessage(`{
		"protocol":"symphony.sodv.verify-result.v1","verification_state":"completion_candidate",
		"read_only":true,"noncanonical":true,"engine_declares_completion":false,
		"result_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"
	}`)
	if valid, err := validateSODVResult("verify", verify); err != nil || !valid {
		t.Fatalf("safe verify rejected: %v", err)
	}

	proposal := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1","module_id":"sodv-engine",
		"engine_id":"symphony-sodv","vector_id":"sodv","proposal_id":"sodv-proposal:test",
		"proposal_digest":"sha256:2222222222222222222222222222222222222222222222222222222222222222",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":false},
		"write_set":[{"target_path":"knowledge/sodv/RELEASES.md"}],"operations":[{}]
	}`)
	if valid, err := validateSODVResult("propose", proposal); err != nil || !valid {
		t.Fatalf("safe proposal rejected: %v", err)
	}

	recovery := json.RawMessage(`{
		"protocol":"symphony.sodv.recovery-result.v1","action":"completion_proposal_required",
		"verification":` + string(verify) + `,"journal_mutated":false,"canonical_apply_enabled":false,
		"result_digest":"sha256:3333333333333333333333333333333333333333333333333333333333333333"
	}`)
	if valid, err := validateSODVResult("recover", recovery); err != nil || !valid {
		t.Fatalf("safe recovery rejected: %v", err)
	}

	projection := json.RawMessage(`{
		"protocol":"symphony.sodv.projection.v1","module_id":"sodv-engine",
		"engine_id":"symphony-sodv","vector_id":"sodv","record_count":1,"transaction_count":1,
		"records":[{}],"transactions":[{}],
		"projection_digest":"sha256:4444444444444444444444444444444444444444444444444444444444444444",
		"noncanonical":true,"rebuildable":true
	}`)
	if valid, err := validateSODVResult("project", projection); err != nil || !valid {
		t.Fatalf("safe projection rejected: %v", err)
	}
}

func TestValidateSODVRejectsAuthorityEscalation(t *testing.T) {
	unsafe := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1","module_id":"sodv-engine",
		"engine_id":"symphony-sodv","vector_id":"sodv","proposal_id":"sodv-proposal:test",
		"proposal_digest":"sha256:5555555555555555555555555555555555555555555555555555555555555555",
		"canonical_apply_enabled":true,
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":true,"ratified":true},
		"write_set":[{"target_path":"knowledge/sodv/RELEASES.md"}],"operations":[{}]
	}`)
	if _, err := validateSODVResult("propose", unsafe); err == nil {
		t.Fatal("authority-escalated SODV proposal was accepted")
	}
}

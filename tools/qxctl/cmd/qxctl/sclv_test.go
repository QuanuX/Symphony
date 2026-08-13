package main

import (
	"crypto/sha256"
	"encoding/hex"
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
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":true},
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
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":false},
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

func TestValidateSCLVEvidenceResultBindsAdapterSemanticsAndDigest(t *testing.T) {
	local := sclvEvidenceFixture(t, "local-git")
	result, err := validateSCLVEvidenceResult("local-git", "0.1.0-dev", local)
	if err != nil {
		t.Fatalf("valid local-Git evidence rejected: %v", err)
	}
	if result.ProviderNamespace != "local-git" || result.EvidenceKind != "revision" {
		t.Fatalf("unexpected local-Git evidence: %+v", result)
	}

	airgap := sclvEvidenceFixture(t, "airgap")
	if _, err := validateSCLVEvidenceResult("airgap", "0.1.0-dev", airgap); err != nil {
		t.Fatalf("valid air-gap evidence rejected: %v", err)
	}
	if _, err := validateSCLVEvidenceResult("local-git", "0.1.0-dev", airgap); err == nil {
		t.Fatal("air-gap evidence was accepted as local-Git evidence")
	}

	var tampered map[string]any
	if err := json.Unmarshal(local, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["source_reference"] = "changed after normalization"
	encoded, err := json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateSCLVEvidenceResult("local-git", "0.1.0-dev", encoded); err == nil {
		t.Fatal("provider evidence with a stale digest was accepted")
	}

	tampered = nil
	if err := json.Unmarshal(local, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["canonical"] = true
	encoded, err = json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateSCLVEvidenceResult("local-git", "0.1.0-dev", encoded); err == nil {
		t.Fatal("provider evidence claiming an extra canonical field was accepted")
	}
}

func TestValidateSCLVEvidenceResultRejectsRatificationEscalation(t *testing.T) {
	var value map[string]any
	if err := json.Unmarshal(sclvEvidenceFixture(t, "local-git"), &value); err != nil {
		t.Fatal(err)
	}
	value["ratification"] = map[string]any{
		"state": "asserted", "subject": "architect", "effective_permission": "ratify",
		"method": "fixture", "evidence_reference": "fixture:1",
		"evidence_digest": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"absence_reason":  "not_applicable",
	}
	delete(value, "evidence_digest")
	canonical, err := marshalDigestCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	value["evidence_digest"] = "sha256:" + hex.EncodeToString(digest[:])
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateSCLVEvidenceResult("local-git", "0.1.0-dev", encoded); err == nil {
		t.Fatal("local-Git normalization that asserted ratification was accepted")
	}
}

func sclvEvidenceFixture(t *testing.T, adapter string) json.RawMessage {
	t.Helper()
	changeRequest := map[string]any{
		"state": "not_applicable", "provider": "not_applicable", "id": "not_applicable",
		"reference": "not_applicable", "absence_reason": "no change request in this evidence",
	}
	ratification := map[string]any{
		"state": "not_asserted", "subject": "not_applicable",
		"effective_permission": "not_applicable", "method": "not_applicable",
		"evidence_reference": "not_applicable", "evidence_digest": "not_applicable",
		"absence_reason": "this evidence does not assert ratification",
	}
	adapterID, provider, kind := "symphony-sclv-evidence-local-git", "local-git", "revision"
	if adapter == "airgap" {
		adapterID, provider, kind = "symphony-sclv-evidence-airgap", "airgap", "combined"
		changeRequest = map[string]any{
			"state": "present", "provider": "gitlab", "id": "change-42",
			"reference": "gitlab:change-42", "absence_reason": "not_applicable",
		}
		ratification = map[string]any{
			"state": "asserted", "subject": "architect", "effective_permission": "ratify",
			"method": "airgap-record", "evidence_reference": "airgap:record-1",
			"evidence_digest": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			"absence_reason":  "not_applicable",
		}
	}
	value := map[string]any{
		"protocol": "symphony.knowledge.provider-evidence.v1", "adapter_id": adapterID,
		"adapter_version": "0.1.0-dev", "provider_namespace": provider,
		"evidence_kind": kind, "observed_at": "2026-08-12T12:00:00Z",
		"source_reference": "fixture:<source>&bounded",
		"repository": map[string]any{
			"revision_scheme": "git-sha1", "revision_value": "1111111111111111111111111111111111111111",
			"tree_digest": "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		},
		"change_request": changeRequest, "ratification": ratification,
	}
	canonical, err := marshalDigestCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	value["evidence_digest"] = "sha256:" + hex.EncodeToString(digest[:])
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

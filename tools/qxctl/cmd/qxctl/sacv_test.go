package main

import (
	"encoding/json"
	"testing"
)

func TestSACVCheckValidityIsPresentationIndependent(t *testing.T) {
	valid, err := sacvCheckValid(json.RawMessage(`{"summary":{"state":"valid","violation":0}}`))
	if err != nil || !valid {
		t.Fatalf("valid result rejected: valid=%t err=%v", valid, err)
	}
	valid, err = sacvCheckValid(json.RawMessage(`{"summary":{"state":"invalid","violation":1}}`))
	if err != nil || valid {
		t.Fatalf("invalid result accepted: valid=%t err=%v", valid, err)
	}
	if _, err := sacvCheckValid(json.RawMessage(`{"summary":{}}`)); err == nil {
		t.Fatal("incomplete SACV check result accepted")
	}
}

func TestValidateSACVResultRejectsAuthorityEscalation(t *testing.T) {
	inspect := json.RawMessage(`{
		"readiness":"read_check_diff_propose_project","empty_registry_valid":true,
		"engine_decides_ownership":false,"canonical_apply_enabled":false,
		"parser_formats":{"json":"implemented","yaml":"fail_closed_unavailable"},
		"descriptor":{"engine_id":"symphony-sacv","openapi_target":"3.2.0",
		"canonical_apply_enabled":false,"session_mutation_enabled":false,"network_listener":true}
	}`)
	if _, err := validateSACVResult("inspect", inspect); err == nil {
		t.Fatal("SACV inspect result with listener enabled was accepted")
	}

	proposal := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1","module_id":"sacv-engine",
		"engine_id":"symphony-sacv","vector_id":"sacv","proposal_id":"sacv-proposal:test",
		"proposal_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":true},
		"write_set":[{"target_path":"knowledge/sacv/REGISTRY.md"}],"operations":[{}]
	}`)
	if _, err := validateSACVResult("propose", proposal); err == nil {
		t.Fatal("self-ratified SACV proposal was accepted")
	}

	wrongTarget := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1","module_id":"sacv-engine",
		"engine_id":"symphony-sacv","vector_id":"sacv","proposal_id":"sacv-proposal:test",
		"proposal_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":false},
		"write_set":[{"target_path":"README.md"}],"operations":[{}]
	}`)
	if _, err := validateSACVResult("propose", wrongTarget); err == nil {
		t.Fatal("SACV proposal targeting a noncanonical surface was accepted")
	}

	diff := json.RawMessage(`{
		"protocol":"symphony.sacv.diff-result.v1","state":"breaking","changes":[],
		"read_only":true,"noncanonical":false,
		"result_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"
	}`)
	if _, err := validateSACVResult("diff", diff); err == nil {
		t.Fatal("canonical-claiming SACV diff was accepted")
	}

	projection := json.RawMessage(`{
		"protocol":"symphony.sacv.projection.v1","module_id":"sacv-engine",
		"engine_id":"symphony-sacv","vector_id":"sacv","entry_count":0,"entries":[],
		"projection_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"noncanonical":false,"rebuildable":true
	}`)
	if _, err := validateSACVResult("project", projection); err == nil {
		t.Fatal("canonical-claiming SACV projection was accepted")
	}
}

func TestPrintSACVResultRejectsInvalidPlainCheck(t *testing.T) {
	result := json.RawMessage(`{
		"entries_checked":1,"documents_checked":1,"operations_checked":1,
		"registry":{"digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		"summary":{"pass":1,"warning":0,"violation":1,"state":"invalid"}
	}`)
	if err := printSACVResult("check", result); err == nil {
		t.Fatal("plain invalid SACV check did not return an error")
	}
}

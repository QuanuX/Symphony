package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func withSSFVDigest(t *testing.T, object map[string]any, field string) json.RawMessage {
	t.Helper()
	canonical, err := marshalSSFVCanonical(object)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	object[field] = "sha256:" + hex.EncodeToString(digest[:])
	result, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSSFVCheckPayloadFreshnessContracts(t *testing.T) {
	disabled, err := ssfvCheckPayload(ssfvOptions{freshness: "disabled"})
	if err != nil {
		t.Fatalf("disabled payload rejected: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(disabled, &value); err != nil || value["baseline"] != nil {
		t.Fatalf("disabled payload malformed: %v", err)
	}

	baseline := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(baseline, []byte(`{"protocol":"fixture"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ssfvCheckPayload(ssfvOptions{
		freshness: "disabled", baseline: baseline,
	}); err == nil {
		t.Fatal("disabled freshness accepted a baseline")
	}
	if _, err := ssfvCheckPayload(ssfvOptions{freshness: "require"}); err == nil {
		t.Fatal("required freshness accepted no baseline")
	}
	if _, err := ssfvCheckPayload(ssfvOptions{
		freshness: "report", baseline: baseline,
	}); err != nil {
		t.Fatalf("report freshness rejected baseline: %v", err)
	}
}

func TestValidateSSFVResultRejectsAuthorityEscalation(t *testing.T) {
	inspect := json.RawMessage(`{
		"readiness":"read_check_diff_propose_graph","empty_registry_valid":true,
		"engine_decides_feature_worthiness":false,"engine_decides_semantic_truth":false,
		"canonical_apply_enabled":false,
		"descriptor":{"module_id":"ssfv-engine","engine_id":"symphony-ssfv","vector_id":"ssfv",
		"language":"C++26","thermal_path":"freezing","install_state":"installed_undocked",
		"default_receptor":null,"canonical_apply_enabled":false,
		"session_mutation_enabled":false,"network_listener":true}
	}`)
	if _, err := validateSSFVResult("inspect", inspect); err == nil {
		t.Fatal("SSFV inspect result with listener enabled was accepted")
	}

	proposal := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1","module_id":"ssfv-engine",
		"engine_id":"symphony-ssfv","vector_id":"ssfv","proposal_id":"ssfv-proposal:test",
		"proposal_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":true},
		"write_set":[{"target_path":"knowledge/ssfv/REGISTRY.md"}],
		"operations":[{"target_path":"knowledge/ssfv/REGISTRY.md"}]
	}`)
	if _, err := validateSSFVResult("propose", proposal); err == nil {
		t.Fatal("self-ratified SSFV proposal was accepted")
	}

	unsafe := json.RawMessage(`{
		"protocol":"symphony.knowledge.proposal.v1","module_id":"ssfv-engine",
		"engine_id":"symphony-ssfv","vector_id":"ssfv","proposal_id":"ssfv-proposal:test",
		"proposal_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"canonical_apply_enabled":false,
		"authority":{"caller_declared_operation":true,"engine_decided_domain_truth":false,"ratified":false},
		"write_set":[{"target_path":"README.md"}],
		"operations":[{"target_path":"README.md"}]
	}`)
	if _, err := validateSSFVResult("propose", unsafe); err == nil {
		t.Fatal("SSFV proposal with unsafe target was accepted")
	}

	graph := json.RawMessage(`{
		"protocol":"symphony.ssfv.graph-projection.v1","module_id":"ssfv-engine",
		"engine_id":"symphony-ssfv","vector_id":"ssfv","node_count":0,"edge_count":0,
		"nodes":[],"edges":[],
		"projection_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"noncanonical":false,"rebuildable":true
	}`)
	if _, err := validateSSFVResult("graph", graph); err == nil {
		t.Fatal("canonical-claiming SSFV graph was accepted")
	}
}

func TestSSFVRequiredFreshnessCannotPassStaleResult(t *testing.T) {
	stale := json.RawMessage(`{
		"protocol":"symphony.ssfv.check-result.v2","structural_state":"valid",
		"freshness_mode":"require","semantic_freshness_state":"stale",
		"read_only":true,"canonical_apply_enabled":false,
		"summary":{"state":"valid","violation":0}
	}`)
	valid, err := validateSSFVResult("check", stale)
	if err != nil {
		t.Fatalf("bounded stale result was malformed: %v", err)
	}
	if valid {
		t.Fatal("stale required-freshness result was accepted as successful")
	}
}

func TestSafeSSFVWriteTargets(t *testing.T) {
	for _, path := range []string{
		"knowledge/ssfv/NAMESPACES.md",
		"knowledge/ssfv/REGISTRY.md",
		"knowledge/skvi/INDEX.md",
		"FEATURES.md",
		"modules/example/FEATURES.md",
	} {
		if !safeSSFVWriteTarget(path) {
			t.Fatalf("safe SSFV write target rejected: %q", path)
		}
	}
	for _, path := range []string{
		"README.md", "../FEATURES.md", "/FEATURES.md",
		"modules//FEATURES.md", `modules\example\FEATURES.md`,
	} {
		if safeSSFVWriteTarget(path) {
			t.Fatalf("unsafe SSFV write target accepted: %q", path)
		}
	}
}

func TestValidateSSFVResultBindsEmbeddedDigests(t *testing.T) {
	diff := map[string]any{
		"protocol": "symphony.ssfv.diff-result.v2", "state": "identical",
		"added_feature_ids": []any{}, "changed_feature_ids": []any{},
		"removed_feature_ids": []any{}, "stale_references": []any{},
		"semantic_candidates": []any{}, "read_only": true, "noncanonical": true,
	}
	validDiff := withSSFVDigest(t, diff, "result_digest")
	if _, err := validateSSFVResult("diff", validDiff); err != nil {
		t.Fatalf("valid SSFV diff digest rejected: %v", err)
	}
	diff["state"] = "additive"
	tamperedDiff, _ := json.Marshal(diff)
	if _, err := validateSSFVResult("diff", tamperedDiff); err == nil {
		t.Fatal("tampered SSFV diff digest was accepted")
	}

	proposal := map[string]any{
		"protocol": "symphony.knowledge.proposal.v1", "module_id": "ssfv-engine",
		"engine_id": "symphony-ssfv", "vector_id": "ssfv",
		"proposal_id": "ssfv-proposal:test", "canonical_apply_enabled": false,
		"authority": map[string]any{
			"caller_declared_operation": true, "engine_decided_domain_truth": false,
			"ratified": false,
		},
		"write_set":  []any{map[string]any{"target_path": "knowledge/ssfv/REGISTRY.md"}},
		"operations": []any{map[string]any{"target_path": "knowledge/ssfv/REGISTRY.md"}},
		"created_at": "<trusted>&bounded",
	}
	validProposal := withSSFVDigest(t, proposal, "proposal_digest")
	if _, err := validateSSFVResult("propose", validProposal); err != nil {
		t.Fatalf("valid SSFV proposal digest rejected: %v", err)
	}
	proposal["created_at"] = "tampered-after-digest"
	tamperedProposal, _ := json.Marshal(proposal)
	if _, err := validateSSFVResult("propose", tamperedProposal); err == nil {
		t.Fatal("tampered SSFV proposal digest was accepted")
	}

	graph := map[string]any{
		"protocol": "symphony.ssfv.graph-projection.v1", "module_id": "ssfv-engine",
		"engine_id": "symphony-ssfv", "vector_id": "ssfv",
		"node_count": 0, "edge_count": 0, "nodes": []any{}, "edges": []any{},
		"noncanonical": true, "rebuildable": true,
	}
	validGraph := withSSFVDigest(t, graph, "projection_digest")
	if _, err := validateSSFVResult("graph", validGraph); err != nil {
		t.Fatalf("valid SSFV graph digest rejected: %v", err)
	}
	graph["projection_kind"] = "tampered-after-digest"
	tamperedGraph, _ := json.Marshal(graph)
	if _, err := validateSSFVResult("graph", tamperedGraph); err == nil {
		t.Fatal("tampered SSFV graph digest was accepted")
	}
}

func TestMarshalSSFVCanonicalMatchesUTF8EngineEncoding(t *testing.T) {
	value := map[string]any{
		"literal_escape":                      `\u2028`,
		"line_separator":                      "\u2028",
		"paragraph_separator_after_backslash": "\\\u2029",
	}
	got, err := marshalSSFVCanonical(value)
	if err != nil {
		t.Fatalf("marshal SSFV canonical JSON: %v", err)
	}
	want := []byte("{\"line_separator\":\"\u2028\",\"literal_escape\":\"\\\\u2028\",\"paragraph_separator_after_backslash\":\"\\\\\u2029\"}")
	if !bytes.Equal(got, want) {
		t.Fatalf("canonical UTF-8 JSON mismatch:\n got %q\nwant %q", got, want)
	}
}

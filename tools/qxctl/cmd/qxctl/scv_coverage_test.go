package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
)

func TestSCVProviderCoverageExplicitVersionAndPreservedDefaults(t *testing.T) {
	for name, command := range map[string]func() string{
		"coverage": func() string { return newSCVProviderCoverageCommand().Flags().Lookup("version").DefValue },
		"interpret": func() string {
			return scvInterpretationLeaf("provider", "interpret", "provider_interpret", "invoke").Flags().Lookup("version").DefValue
		},
		"artifact": func() string {
			c, _, _ := newSCVArtifactCommand().Find([]string{"import"})
			return c.Flags().Lookup("version").DefValue
		},
		"workflow": func() string {
			c, _, _ := newSCVWorkflowCommand().Find([]string{"run"})
			return c.Flags().Lookup("version").DefValue
		},
	} {
		want := map[string]string{"coverage": "0.5.0-dev", "interpret": "0.3.0-dev", "artifact": "0.4.0-dev", "workflow": "0.4.0-dev"}[name]
		if command() != want {
			t.Fatalf("%s changed exact default", name)
		}
	}
	for _, flag := range []string{"input", "json", "domain", "prefix", "version", "repo"} {
		if newSCVProviderCoverageCommand().Flags().Lookup(flag) == nil {
			t.Fatalf("coverage lacks %s", flag)
		}
	}
}

func TestInstalledSCVProviderCoverageArtifactsReplayOriginalOwner(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COVERAGE_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.5.0-dev packages")
	}
	s := workflowStore(t)
	r, err := newWorkflowRunner(scvOptions{domain: "schv-aws", prefix: prefix, version: "0.5.0-dev", topsID: ssiagTestTOPSID, repository: s.Root}, true)
	if err != nil {
		t.Fatal(err)
	}
	owner := func(op string, input any) map[string]any {
		t.Helper()
		raw, err := r.owner(r.installation, op, input)
		if err != nil {
			t.Fatal(op, err)
		}
		return workflowValue(t, raw)
	}
	providerInput := map[string]any{"provider_id": "aws", "family_id": "schv", "display_name": "Synthetic fixture", "sources": []any{map[string]any{"source_id": "docs", "provider_id": "aws", "family_id": "schv", "publisher": "Fixture author", "authority_role": "documentation", "scope": "Unacquired fixture declaration", "locators": []any{map[string]any{"locator_id": "main", "uri": "https://example.invalid/docs", "role": "primary", "format": "text", "selector": "fixture-v1"}}, "continuity_evidence": []any{}}}}
	provider := owner("provider_onboard", providerInput)
	corpus := owner("corpus_build", map[string]any{"corpus_id": "empty-fixture", "previous": nil, "snapshot_time": "2026-09-10T12:00:00Z", "attempts": []any{}})
	coverageInput := map[string]any{"provider": provider, "corpus_query": map[string]any{"corpus": corpus, "query_time": "2026-09-10T12:00:00Z", "member_ids": []any{}, "selection": "latest_attempt", "max_age_seconds": nil}, "interpretations": []any{}}
	coverage := owner("provider_coverage", coverageInput)
	refs := []any{}
	for _, item := range []struct {
		operation, kind string
		input, result   map[string]any
	}{{"provider_onboard", "provider", providerInput, provider}, {"provider_coverage", "coverage", coverageInput, coverage}} {
		request := map[string]any{"operation": item.operation, "input": item.input, "result": nil}
		raw, err := r.artifact(s, "import", request)
		if err != nil {
			t.Fatal(err)
		}
		record := workflowValue(t, raw)
		if record["kind"] != item.kind || record["artifact_digest"] != item.result["digest"] || record["digest"] == record["artifact_digest"] {
			t.Fatal("retained provenance conflated with native artifact")
		}
		refs = append(refs, record["digest"])
		wrongOwner := workflowClone(t, record)
		wrongOwner["installation"].(map[string]any)["Version"] = "0.4.0-dev"
		wrongOwnerRaw, err := scvworkflow.Seal(wrongOwner)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := scvworkflow.ReadRecord(wrongOwnerRaw); err == nil {
			t.Fatal("new retained kind accepted resealed .4 provenance")
		}
		request["result"] = item.result
		again, err := r.artifact(s, "import", request)
		if err != nil || !scvworkflow.Same(raw, again) {
			t.Fatal("exact supplied result changed retained identity", err)
		}
		wrong := workflowClone(t, item.result)
		wrong["digest"] = provider["digest"]
		if item.operation == "provider_onboard" {
			wrong["display_name"] = "Another provider"
		}
		request["result"] = wrong
		if _, err := r.artifact(s, "import", request); err == nil {
			t.Fatal("supplied result mismatch retained")
		}
	}
	// A selected unrelated domain must not replace the recorded owner on show.
	observer, err := newWorkflowRunner(scvOptions{domain: "scev-cf", prefix: "/nonexistent/unselected", version: "0.4.0-dev", repository: s.Root}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range refs {
		raw, err := observer.artifact(s, "show", map[string]any{"record_digest": ref})
		if err != nil {
			t.Fatal(err)
		}
		if workflowValue(t, raw)["installation"].(map[string]any)["Role"] != "schv-aws" {
			t.Fatal("original validating owner lost")
		}
	}
	listing, err := observer.artifact(s, "list", map[string]any{"after_digest": nil, "limit": json.Number("128")})
	if err != nil {
		t.Fatal(err)
	}
	list := workflowValue(t, listing)
	if len(list["records"].([]any)) != 2 {
		t.Fatal("retained provider/coverage listing lost entries")
	}
	// Read requires the exact original receipt; listing remains envelope-only.
	inspect := observer.inspect
	observer.inspect = func(role, prefix, version string) (knowledgeengine.Installation, error) {
		inst, err := inspect(role, prefix, version)
		inst.ReceiptDigest = "changed-receipt"
		return inst, err
	}
	if _, err := observer.artifact(s, "show", map[string]any{"record_digest": refs[1]}); err == nil {
		t.Fatal("coverage replay accepted replaced recorded owner")
	}
	if _, err := observer.artifact(s, "list", map[string]any{"after_digest": nil, "limit": json.Number("128")}); err != nil {
		t.Fatal("metadata listing required owner replay", err)
	}
}

func TestSCVProviderArtifactKindsRequireExplicitNewVersion(t *testing.T) {
	r, s, _ := mockWorkflow(t)
	for _, op := range []string{"provider_onboard", "provider_coverage"} {
		if _, err := r.artifact(s, "import", map[string]any{"operation": op, "input": map[string]any{}, "result": nil}); err == nil {
			t.Fatalf("%s new retained kind accepted under .4", op)
		}
	}
}

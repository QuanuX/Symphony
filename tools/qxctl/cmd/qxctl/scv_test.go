package main

import (
	"bytes"
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvgraph"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvstate"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSCVAuthorizationUsesAuthenticatedSSIAGDecision(t *testing.T) {
	// This boundary fixture is a kernel-authenticated Unix endpoint, not a
	// running SSIAG/STAV producer. Production audit durability is owned by SSIAG.
	serveNamedVersionAcceptanceSSIAG(t)
	options := scvOptions{topsID: ssiagTestTOPSID, domain: "schv-aws", sourceID: "aws-models"}
	correlation := "83c9dbb8-5445-49e9-8012-fccceade9bd5"
	decision, err := authorizeSCVSource(options, correlation, "relocate")
	if err != nil {
		t.Fatal(err)
	}
	if decision.CorrelationID != correlation || decision.Target.Operation != "symphony.scv.source.relocate" || decision.Target.Resource != scvSourceResource(options) || decision.Capability == nil {
		t.Fatal("source decision binding mismatch")
	}
	other := options
	other.sourceID = "other"
	if scvSourceResource(options) == scvSourceResource(other) {
		t.Fatal("source authority resource collision")
	}
}

func TestSCVAuthorizationRejectsUnavailableAuthority(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("SYMPHONY_SSIAG_SOCKET", "")
	if _, err := authorizeSCVSource(scvOptions{topsID: ssiagTestTOPSID, domain: "scv", sourceID: "docs"}, "83c9dbb8-5445-49e9-8012-fccceade9bd5", "onboard"); err == nil {
		t.Fatal("unavailable authenticated authority accepted")
	}
}

func TestInstalledSCVSourceDeniedThenAuthorizedReplay(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_ACCEPTANCE_PREFIX")
	if prefix == "" {
		t.Skip("requires exact staged SCV engines")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	options := scvOptions{domain: "scv", prefix: prefix, version: "0.1.0-dev", repository: cwd, topsID: ssiagTestTOPSID, stateRoot: root, sourceID: "aws-docs"}
	desired := map[string]any{"source_id": "aws-docs", "provider_id": "aws", "family_id": "schv", "publisher": "AWS", "authority_role": "documentation", "scope": "selected docs",
		"locators": []any{map[string]any{"locator_id": "docs", "uri": "https://docs.aws.amazon.com/", "role": "primary", "format": "html", "selector": ""}}, "continuity_evidence": []string{}}
	plan, err := invokeSCV(options, "source_plan", map[string]any{"operation_id": "onboard-1", "current": nil, "desired": desired, "reason": "explicit test selection"})
	if err != nil {
		t.Fatal(err)
	}
	options.input = filepath.Join(root, "plan.json")
	if err := os.WriteFile(options.input, plan, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "absent-config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(root, "absent-runtime"))
	t.Setenv("SYMPHONY_SSIAG_SOCKET", "")
	if err := runSCVSource("apply", options); err == nil {
		t.Fatal("unauthorized source selected")
	}
	store, err := scvstate.New(root, options.topsID, options.domain, options.sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WithLock(func(tx *scvstate.Transaction) error {
		if string(tx.Current()) != "null" {
			t.Fatal("denied source apply moved head")
		}
		attempt, ok := tx.Attempt("onboard-1")
		if !ok || attempt.Status != "prepared" {
			t.Fatal("denied apply lost exact intent")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	serveNamedVersionAcceptanceSSIAG(t)
	if err := runSCVSource("apply", options); err != nil {
		t.Fatal(err)
	}
	if err := runSCVSource("apply", options); err != nil {
		t.Fatalf("exact replay failed: %v", err)
	}
	var firstSource json.RawMessage
	if err := store.WithLock(func(tx *scvstate.Transaction) error {
		firstSource = tx.Current()
		attempt, _ := tx.Attempt("onboard-1")
		if attempt.Status != "committed" || len(attempt.Authorization) == 0 {
			t.Fatal("committed authorization/source absent")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Change the desired locator under a fresh plan; stale original predecessor
	// must fail after it has already been consumed, without authorizing a write.
	desired["locators"] = []any{map[string]any{"locator_id": "docs", "uri": "https://docs.aws.amazon.com/new", "role": "primary", "format": "html", "selector": ""}}
	second, err := invokeSCV(options, "source_plan", map[string]any{"operation_id": "relocate-2", "current": firstSource, "desired": desired, "reason": "confirmed location"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.input, second, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runSCVSource("apply", options); err != nil {
		t.Fatal(err)
	}
	var changed map[string]any
	d := json.NewDecoder(bytes.NewReader(second))
	d.UseNumber()
	if err := d.Decode(&changed); err != nil {
		t.Fatal(err)
	}
	changed["operation_id"] = "relocate-stale"
	delete(changed, "plan_digest")
	changed["plan_digest"], _ = knowledgeengine.SCVDigest(changed)
	stale, _ := knowledgeengine.SCVCanonical(changed)
	if err := os.WriteFile(options.input, stale, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runSCVSource("apply", options); err == nil {
		t.Fatal("stale source plan accepted")
	}
}

func TestInstalledSCVGraphAuthorizedSelectionReplay(t *testing.T) {
	options, _ := projectionOptions(t)
	options.topsID = ssiagTestTOPSID
	serveNamedVersionAcceptanceSSIAG(t)
	knowledge, err := invokeSCV(options, "knowledge_interpret", map[string]any{"captures": []any{}, "claims": []any{}, "interpreter_version": "empty-fixture",
		"selection_policy": map[string]any{"policy_id": "fixture", "max_age_seconds": nil, "allowed_statement_kinds": []string{"documented_fact"}, "partial_capture": "exclude"}})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := invokeSCV(options, "graph_build", map[string]any{"knowledge": []json.RawMessage{knowledge}})
	if err != nil {
		t.Fatal(err)
	}
	projectionInput(t, &options, map[string]any{"graph": graph, "expected_graph_digest": nil, "expected_generation": 0})
	if err := runSCVProjection("select", options, "research"); err != nil {
		t.Fatal(err)
	}
	if err := runSCVProjection("select", options, "research"); err != nil {
		t.Fatalf("graph replay failed: %v", err)
	}
	store, err := scvgraph.New(options.stateRoot, options.topsID, options.domain, "research")
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if state.GraphDigest == nil || state.Generation != 1 || state.Operations[options.operationID].Status != "committed" || !scvSameJSON(state.SelectedGraph(), graph) {
		t.Fatal("graph selection/replay did not preserve exact coherent revision")
	}
}

func TestSCVAuthorizationExpiryBeforePublication(t *testing.T) {
	_, decision := validSessionAuthorization(t)
	if err := scvAuthorizationFreshness(decision)(); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().UTC().Add(-time.Second)
	decision.ExpiresAt = &expired
	if err := scvAuthorizationFreshness(decision)(); err == nil {
		t.Fatal("expired decision accepted before publication")
	}
	_, decision = validSessionAuthorization(t)
	decision.Capability.ExpiresAt = expired
	if err := scvAuthorizationFreshness(decision)(); err == nil {
		t.Fatal("expired capability accepted before publication")
	}
}

func TestSCVAuthorizationRejectsInvalidAuditCorrelation(t *testing.T) {
	for _, correlation := range []string{"relocate-1", "", "83c9dbb8-5445-59e9-8012-fccceade9bd5"} {
		if _, err := authorizeSCVRequest(ssiagTestTOPSID, correlation, "symphony.scv.source.relocate", "source-fixture"); err == nil {
			t.Fatalf("non-STAV correlation accepted: %q", correlation)
		}
	}
}

package main

import (
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func cliScopeFixture(t *testing.T) knowledgelifecycle.ExecutionScope {
	t.Helper()
	data, err := os.ReadFile("../../../../modules/knowledge-session-coordinator/tests/fixtures/lifecycle_execution_scope.json")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := knowledgelifecycle.DecodeExecutionScope(data)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestLifecycleApplyRejectsInvalidArgumentsBeforeStateMutation(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(knowledgeLifecycleOptions) error
		want string
	}{
		{"apply missing identity", runKnowledgeLifecycleApply, "--operation-id, --source-journal-digest"},
		{"apply invalid identity", func(options knowledgeLifecycleOptions) error {
			options.operationID = "invalid identity"
			options.sourceJournalDigest = "sha256:" + strings.Repeat("1", 64)
			options.expectedApplyDigest, options.expectedAppliedDigest = "absent", "absent"
			return runKnowledgeLifecycleApply(options)
		}, "mutation identities must be exact tokens"},
		{"apply invalid bound", func(options knowledgeLifecycleOptions) error {
			options.operationID = "reviewed-operation"
			options.sourceJournalDigest = "sha256:" + strings.Repeat("1", 64)
			options.expectedApplyDigest, options.expectedAppliedDigest = "absent", "absent"
			options.maxActions = 0
			return runKnowledgeLifecycleApply(options)
		}, "--max-actions must be between"},
		{"apply unknown native owner", func(options knowledgeLifecycleOptions) error {
			options.operationID = "reviewed-operation"
			options.sourceJournalDigest = "sha256:" + strings.Repeat("1", 64)
			options.expectedApplyDigest, options.expectedAppliedDigest = "absent", "absent"
			options.nativeOwners = []string{"arbitrary-owner=/tmp/owner"}
			return runKnowledgeLifecycleApply(options)
		}, "native owner must be a known component"},
		{"apply native owner without scope", func(options knowledgeLifecycleOptions) error {
			options.operationID = "reviewed-operation"
			options.sourceJournalDigest = "sha256:" + strings.Repeat("1", 64)
			options.expectedApplyDigest, options.expectedAppliedDigest = "absent", "absent"
			options.nativeOwners = []string{"stav-append-authority=/tmp/owner"}
			return runKnowledgeLifecycleApply(options)
		}, "--native-owner requires an exact --execution-scope"},
		{"recover missing operation", func(options knowledgeLifecycleOptions) error {
			return runKnowledgeLifecycleApplyState("lifecycle_apply_recover", options)
		}, "--operation-id is required"},
		{"recover conflicting expectation", func(options knowledgeLifecycleOptions) error {
			options.operationID, options.discover = "reviewed-operation", true
			options.expectedApplyDigest = "sha256:" + strings.Repeat("1", 64)
			return runKnowledgeLifecycleApplyState("lifecycle_apply_recover", options)
		}, "mutually exclusive"},
		{"recover missing expectation", func(options knowledgeLifecycleOptions) error {
			options.operationID = "reviewed-operation"
			return runKnowledgeLifecycleApplyState("lifecycle_apply_recover", options)
		}, "--expected-apply-journal-digest or --discover is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			options := knowledgeLifecycleOptions{
				stateRoot: root, topsID: "11111111-1111-4111-8111-111111111111",
				profileID: "default", scope: "user", ttl: time.Minute, maxActions: 4096,
			}
			if err := test.run(options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("invalid request returned %v; expected %s", err, test.want)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("invalid request created lifecycle state before validation: entries=%v err=%v", entries, err)
			}
		})
	}
}

func TestLifecycleApplyAndRecoverKeepExecutionLease(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	options := knowledgeLifecycleOptions{
		stateRoot: root, topsID: "11111111-1111-4111-8111-111111111111",
		profileID: "default", scope: "user", ttl: time.Minute, maxActions: 4096,
		operationID: "reviewed-operation", sourceJournalDigest: "sha256:" + strings.Repeat("1", 64),
		expectedApplyDigest: "sha256:" + strings.Repeat("2", 64), expectedAppliedDigest: "absent",
	}
	store, err := lifecycleStore(options)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WithExecutionLease(func() error {
		for _, test := range []struct {
			name string
			run  func() error
		}{
			{"apply", func() error { return runKnowledgeLifecycleApply(options) }},
			{"recover", func() error { return runKnowledgeLifecycleApplyState("lifecycle_apply_recover", options) }},
		} {
			t.Run(test.name, func(t *testing.T) {
				if err := test.run(); err == nil || !strings.Contains(err.Error(), "lifecycle execution is busy") {
					t.Fatalf("validated operation bypassed the held execution lease: %v", err)
				}
			})
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func TestScopedApplyFlagsAndAuthorizationBinding(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"apply", "apply-status", "apply-recover"} {
		command, _, err := root.Find([]string{"knowledge", "lifecycle", name})
		if err != nil || command.Flags().Lookup("execution-scope") == nil {
			t.Fatalf("scope flag missing on %s: %v", name, err)
		}
		if (command.Flags().Lookup("native-owner") != nil) != (name == "apply") {
			t.Fatalf("native executable dispatch flag must be limited to apply: %s", name)
		}
	}
	scope := cliScopeFixture(t)
	options := knowledgeLifecycleOptions{executionScope: &scope}
	command := map[string]any{"protocol": "symphony.knowledge.lifecycle-apply-command.v1"}
	bindLifecycleExecutionScope(options, command)
	if command["protocol"] != "symphony.knowledge.lifecycle-apply-command.v2" || command["execution_scope"] != &scope || lifecycleScopeEvidence(options, "status") != "status\nscope="+scope.ScopeDigest {
		t.Fatal("scope protocol or authorization evidence was not bound")
	}
	if _, err := requireLifecycleResultScope(options, validatedLifecycleApplyResult{}, "lifecycle_apply_status"); err == nil {
		t.Fatal("scoped request silently fell back to v1")
	}
	if !strings.Contains(strings.Join(lifecycleApplyClient()["capabilities"].([]string), ","), "scoped-apply-v1") {
		t.Fatal("scoped capability omitted")
	}
}
func TestScopedApplySelectionPreservesWholePlanAndExactEffects(t *testing.T) {
	scope := cliScopeFixture(t)
	effect := scope.Effects[0]
	desired := knowledgelifecycle.DesiredState{Components: []knowledgelifecycle.DesiredComponent{{ComponentID: effect.ComponentID, InstallRoot: effect.InstallRoot}, {ComponentID: "other", InstallRoot: "/opt/other"}}}
	allowed := knowledgelifecycle.PlannedAction{ActionID: "allowed", ComponentID: effect.ComponentID, Kind: effect.Kind, TargetStateDigest: effect.TargetStateDigest, ExpectedArtifactDigests: effect.ExpectedArtifactDigests, Disposition: "ready"}
	outside := allowed
	outside.ActionID = "outside"
	outside.ComponentID = "other"
	downstream := allowed
	downstream.ActionID = "selection"
	downstream.Kind = "select"
	actions := []knowledgelifecycle.PlannedAction{outside, downstream, allowed}
	selected := selectScopedLifecycleAction(actions, effect.ExpectedArtifactDigests, false, &scope, desired)
	if selected == nil || selected.ActionID != "allowed" || len(actions) != 3 {
		t.Fatal("scoped selection mutated or projected the complete safety plan")
	}
	if selectScopedLifecycleAction(actions[:2], effect.ExpectedArtifactDigests, false, &scope, desired) != nil {
		t.Fatal("scope authorized another component or downstream effect")
	}

	observed := knowledgelifecycle.Observation{Components: []knowledgelifecycle.ObservedComponent{{ComponentID: effect.ComponentID, Packages: []knowledgelifecycle.ObservedPackage{{ReceiptDigest: effect.ExpectedArtifactDigests[0], InstallRoot: effect.InstallRoot, Integrity: "valid", EntryPointsValidated: true}}}}}
	if !lifecycleScopeConverged(actions, 0, &scope, desired, observed) || lifecycleScopeConverged(actions, 1, &scope, desired, observed) || lifecyclePlanConverged(actions, 0) {
		t.Fatal("exact install completion confused with downstream/global convergence")
	}

}
func TestScopedResumeRejectsDriftBeforeOwnershipOrExecution(t *testing.T) {
	scope := cliScopeFixture(t)
	scope.Components = append(scope.Components, "module-b")
	scope.ScopeDigest, _ = knowledgelifecycle.ExecutionScopeDigest(scope)
	observation := knowledgelifecycle.Observation{Components: []knowledgelifecycle.ObservedComponent{{ComponentID: "module-a", Activation: "inactive"}, {ComponentID: "module-b", Activation: "inactive"}, {ComponentID: "outside", Activation: "inactive"}}}
	result := validatedLifecycleApplyResult{ScopeDigest: scope.ScopeDigest, ActiveAction: &knowledgelifecycle.PlannedAction{ComponentID: "module-a", Kind: "activate"}}
	result.PreservedComponentsDigest, _ = scope.PreservedComponentsDigest(observation)
	result.ActionPreservedComponentsDigest, _ = knowledgelifecycle.ActionPreservedComponentsDigest(observation, *result.ActiveAction)
	options := knowledgeLifecycleOptions{executionScope: &scope}
	if err := requireScopeExecutionCheckpoint(options, result, observation); err != nil {
		t.Fatal(err)
	}
	observation.Components[0].Activation = "active"
	if err := requireScopeExecutionCheckpoint(options, result, observation); err != nil {
		t.Fatalf("interrupted selected effect cannot resume: %v", err)
	}
	for _, index := range []int{1, 2} {
		observation.Components[index].Activation = "active"
		calls := 0
		err := func() error {
			if err := requireScopeExecutionCheckpoint(options, result, observation); err != nil {
				return err
			}
			calls++
			return nil
		}()
		if err == nil || calls != 0 {
			t.Fatalf("drift admitted ownership/execution for component %d", index)
		}
		observation.Components[index].Activation = "inactive"
	}
	result.ActionPreservedComponentsDigest = ""
	if err := requireScopeExecutionCheckpoint(options, result, observation); err == nil {
		t.Fatal("missing action checkpoint accepted")
	}
}
func scopedStatusResult(t *testing.T, scope knowledgelifecycle.ExecutionScope, complete bool) validatedLifecycleApplyResult {
	t.Helper()
	payload := map[string]any{"scope": scope, "scope_complete": complete, "profile_converged": false, "preserved_components_digest": lifecycleTestDigest("preserved"), "action_preserved_components_digest": nil}
	encoded, err := knowledgelifecycle.CanonicalScopeJSON(payload)
	if err != nil {
		t.Fatal(err)
	}
	extension := map[string]any{"extension_id": "scoped-apply-v1", "extension_version": "1.0.0", "critical": true, "payload": payload, "payload_digest": taggedLifecycleDigest(encoded)}
	return validatedLifecycleApplyResult{Scoped: true, JournalPresent: true, ScopeDigest: scope.ScopeDigest, ScopeComplete: complete, State: "verified", Raw: map[string]any{"journal": map[string]any{"extensions": []any{extension}, "current_stable_inventory_digest": lifecycleTestDigest("stable")}}}
}
func TestScopedResultCompletionRecoveryAndCriticalExtension(t *testing.T) {
	scope := cliScopeFixture(t)
	options := knowledgeLifecycleOptions{executionScope: &scope, profileID: scope.ProfileID, topsID: scope.TOPSID}
	result := scopedStatusResult(t, scope, true)
	validated, err := requireLifecycleResultScope(options, result, "lifecycle_apply_status")
	if err != nil || !validated.ScopeComplete || validated.ProfileConverged {
		t.Fatalf("partial completion invalid: %+v %v", validated, err)
	}
	newer := scope
	newer.ScopeID = "next-scope"
	newer.ScopeDigest, _ = knowledgelifecycle.ExecutionScopeDigest(newer)
	options.executionScope = &newer
	if got, err := requireLifecycleResultScope(options, result, "lifecycle_apply_status"); err != nil || got.ScopeDigest == newer.ScopeDigest {
		t.Fatalf("prior completion confused with new scope: %v", err)
	}
	if _, err := requireLifecycleResultScope(options, result, "lifecycle_apply_recover"); err == nil {
		t.Fatal("recovery adopted another scope")
	}
	result.ScopeComplete = false
	if _, err := requireLifecycleResultScope(options, result, "lifecycle_apply_status"); err == nil {
		t.Fatal("active prior scope was allowed to drift")
	}
	options.executionScope = &scope
	result = scopedStatusResult(t, scope, true)
	extension := result.Raw.(map[string]any)["journal"].(map[string]any)["extensions"].([]any)[0].(map[string]any)
	extension["payload_digest"] = lifecycleTestDigest("wrong")
	if _, err := requireLifecycleResultScope(options, result, "lifecycle_apply_status"); err == nil {
		t.Fatal("tampered scope extension accepted")
	}
}
func TestValidateLifecycleApplyV2AbsentResult(t *testing.T) {
	scope := cliScopeFixture(t)
	value := map[string]any{"protocol": "symphony.knowledge.lifecycle-apply-result.v2", "operation": "lifecycle_apply_status", "compatibility": map[string]any{"mode": "full", "process_protocol": "symphony.knowledge.engine-process.v1", "journal_read_version": 2, "journal_write_version": 2, "missing_capabilities": []any{}, "two_way_procedural_compatibility": true, "reason": "full"}, "journal_present": false, "journal": nil, "journal_digest": nil, "plan": nil, "action": nil, "applied_state": nil, "changed": false, "recovered": false, "repair_actions": []any{}, "read_only": true, "apply_authorized": false, "canonical": false, "scope_digest": scope.ScopeDigest, "scope_complete": false, "profile_converged": false}
	raw, _ := json.Marshal(value)
	result, err := validateLifecycleApplyResult(raw, "lifecycle_apply_status", scope.ProfileID, scope.TOPSID, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	options := knowledgeLifecycleOptions{executionScope: &scope, profileID: scope.ProfileID, topsID: scope.TOPSID}
	if _, err := requireLifecycleResultScope(options, result, "lifecycle_apply_status"); err != nil {
		t.Fatal(err)
	}
	delete(value, "scope_complete")
	raw, _ = json.Marshal(value)
	if _, err := validateLifecycleApplyResult(raw, "lifecycle_apply_status", scope.ProfileID, scope.TOPSID, "", "", ""); err == nil {
		t.Fatal("incomplete v2 result accepted")
	}
}

// This optional fixture is emitted by the native lifecycle test binary, rather
// than reconstructed by Go. Qualification enables it to test the real boundary.
func TestScopedNativeProducedResults(t *testing.T) {
	path := os.Getenv("SYMPHONY_SCOPED_APPLY_FIXTURE")
	if path == "" {
		t.Skip("native-produced qualification fixture not requested")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"activation", "installation"} {
		t.Run(name, func(t *testing.T) {
			var record struct {
				Scope    json.RawMessage                 `json:"scope"`
				Desired  knowledgelifecycle.DesiredState `json:"desired"`
				Before   knowledgelifecycle.Observation  `json:"observation_before"`
				After    knowledgelifecycle.Observation  `json:"observation_after"`
				Prepare  json.RawMessage                 `json:"prepare_result"`
				Finalize json.RawMessage                 `json:"finalize_result"`
				Source   string                          `json:"source_report_journal_digest"`
			}
			if err := json.Unmarshal(fixture[name], &record); err != nil {
				t.Fatal(err)
			}
			scope, err := knowledgelifecycle.DecodeExecutionScope(record.Scope)
			if err != nil {
				t.Fatal(err)
			}
			if err := scope.ValidateDesired(scope.ProfileDigest, record.Desired, record.Source); err != nil {
				t.Fatal(err)
			}
			options := knowledgeLifecycleOptions{executionScope: &scope, profileID: scope.ProfileID, topsID: scope.TOPSID}
			prepared, err := validateLifecycleApplyResult(record.Prepare, "lifecycle_apply_prepare", scope.ProfileID, scope.TOPSID, record.Desired.DesiredStateDigest, record.Before.ObservationDigest, "")
			if err != nil {
				t.Fatal(err)
			}
			prepared, err = requireLifecycleResultScope(options, prepared, "lifecycle_apply_prepare")
			if err != nil {
				t.Fatal(err)
			}
			if err := requireScopeExecutionCheckpoint(options, prepared, record.Before); err != nil {
				t.Fatalf("native preparation checkpoint parity: %v", err)
			}
			if err := requireScopeExecutionCheckpoint(options, prepared, record.After); err != nil {
				t.Fatalf("native authorized effect projection parity: %v", err)
			}
			stable, err := knowledgelifecycle.StableInventoryDigest(record.Before)
			if err != nil {
				t.Fatal(err)
			}
			semantic, err := knowledgelifecycle.InitialScopePlanDigest(prepared.Plan.Raw, stable)
			if err != nil {
				t.Fatal(err)
			}
			if err := scope.ValidateInitial(record.Desired, record.Before, semantic, stable, "absent", "absent"); err != nil {
				t.Fatalf("native initial semantic plan parity: %v", err)
			}
			finalized, err := validateLifecycleApplyResult(record.Finalize, "lifecycle_apply_finalize", scope.ProfileID, scope.TOPSID, record.Desired.DesiredStateDigest, record.After.ObservationDigest, "")
			if err != nil {
				t.Fatal(err)
			}
			finalized, err = requireLifecycleResultScope(options, finalized, "lifecycle_apply_finalize")
			if err != nil {
				t.Fatal(err)
			}
			if scope.EffectsComplete(record.Desired, record.After, finalized.Plan.FatalCount) != finalized.ScopeComplete {
				t.Fatal("native exact effect completion disagrees")
			}
			if lifecyclePlanConverged(finalized.Plan.Actions, finalized.Plan.FatalCount) != finalized.ProfileConverged {
				t.Fatal("native global convergence disagrees")
			}
			if name == "installation" && (!finalized.ScopeComplete || finalized.ProfileConverged) {
				t.Fatal("install-only completion falsely selected or globally converged")
			}
		})
	}
}

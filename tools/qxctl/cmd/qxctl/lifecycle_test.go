package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
)

func TestValidateLifecyclePlanRejectsApplyAndAcceptsDynamicReport(t *testing.T) {
	desired := lifecycleTestDigest("desired")
	observed := lifecycleTestDigest("observed")
	actionID := "lifecycle-action:" + strings.Repeat("a", 64)
	plan := map[string]any{
		"protocol": "symphony.knowledge.lifecycle-plan.v1", "format_version": 1,
		"transaction_id": "lifecycle-transaction:" + strings.Repeat("b", 64),
		"revision":       1, "previous_plan_digest": nil,
		"desired_state_digest": desired, "observation_digest": observed,
		"observation_key": lifecycleTestDigest("observation-key"), "prior_applied_state_digest": nil,
		"compatibility": map[string]any{
			"mode": "full", "coordinator_version": "0.1.0-dev",
			"desired_state_version": 1, "observation_version": 1, "plan_version": 1,
			"applied_state_version": 1, "receipt_versions": []any{1, 2},
			"required_capabilities": []any{"dependency-ready-set-v1"},
			"missing_capabilities":  []any{}, "two_way_procedural_compatibility": true,
			"reason": "all report-only lifecycle capabilities are shared",
		},
		"scheduler": map[string]any{
			"algorithm": "dependency_ready_set_v1", "dynamic_replanning": true,
			"directionality": "forward_and_inverse", "tie_break": "lexicographic_action_id",
			"safety_phase_order": []any{"lock", "observe", "authorize", "compare_and_swap", "act", "verify", "audit"},
			"cycle_policy":       "block_cyclic_component_continue_unrelated",
			"max_actions":        4096, "max_replans_per_transaction": 256, "max_attempts_per_action": 8,
		},
		"actions": []any{map[string]any{
			"action_id": actionID, "component_id": "example", "kind": "activate", "direction": "forward",
			"prerequisite_action_ids": []any{}, "inverse_action_id": "lifecycle-action:" + strings.Repeat("c", 64),
			"expected_before_digest": lifecycleTestDigest("before"), "target_state_digest": lifecycleTestDigest("target"),
			"target_receptor_id": nil, "expected_artifact_digests": []any{},
			"expected_evidence": []any{"authorization_required_at_apply"}, "disposition": "ready", "blockers": []any{},
		}},
		"ready_action_ids": []any{actionID}, "deferred_action_ids": []any{}, "blocked_action_ids": []any{},
		"advisories": []any{}, "fatal_blockers": []any{}, "apply_authorized": false, "canonical": false,
	}
	plan["plan_digest"] = lifecyclePlanDigest(t, plan)
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	validated, err := validateLifecyclePlan(raw, desired, observed, "")
	if err != nil || validated.ActionCount != 1 || validated.ReadyCount != 1 || validated.BlockedCount != 0 {
		t.Fatalf("valid dynamic report was rejected: %+v err=%v", validated, err)
	}

	scheduler := plan["scheduler"].(map[string]any)
	scheduler["unknown"] = true
	plan["plan_digest"] = lifecyclePlanDigest(t, plan)
	raw, _ = json.Marshal(plan)
	if _, err := validateLifecyclePlan(raw, desired, observed, ""); err == nil {
		t.Fatal("unknown scheduler field was accepted")
	}
	delete(scheduler, "unknown")

	action := plan["actions"].([]any)[0].(map[string]any)
	action["expected_evidence"] = []any{"authorization_required_at_apply", "authorization_required_at_apply"}
	plan["plan_digest"] = lifecyclePlanDigest(t, plan)
	raw, _ = json.Marshal(plan)
	if _, err := validateLifecyclePlan(raw, desired, observed, ""); err == nil {
		t.Fatal("duplicate action evidence was accepted")
	}
	action["expected_evidence"] = []any{"authorization_required_at_apply"}
	action["prerequisite_action_ids"] = []any{"lifecycle-action:" + strings.Repeat("d", 64)}
	plan["plan_digest"] = lifecyclePlanDigest(t, plan)
	raw, _ = json.Marshal(plan)
	if _, err := validateLifecyclePlan(raw, desired, observed, ""); err == nil {
		t.Fatal("unknown prerequisite action was accepted")
	}
	action["prerequisite_action_ids"] = []any{}

	plan["apply_authorized"] = true
	plan["plan_digest"] = lifecyclePlanDigest(t, plan)
	raw, _ = json.Marshal(plan)
	if _, err := validateLifecyclePlan(raw, desired, observed, ""); err == nil {
		t.Fatal("engine-declared lifecycle apply authority was accepted")
	}
}

func TestLifecycleCommandGrammarIsRegistered(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := root.Find([]string{"knowledge", "lifecycle", "profile", "set"})
	if err != nil || command == nil || command.Name() != "set" {
		t.Fatalf("lifecycle profile set grammar is absent: command=%v err=%v", command, err)
	}
	command, _, err = root.Find([]string{"knowledge", "lifecycle", "report"})
	if err != nil || command == nil || command.Name() != "report" {
		t.Fatalf("lifecycle report grammar is absent: command=%v err=%v", command, err)
	}
	for _, name := range []string{"boot", "status", "recover", "apply", "apply-status", "apply-recover"} {
		command, _, err = root.Find([]string{"knowledge", "lifecycle", name})
		if err != nil || command == nil || command.Name() != name {
			t.Fatalf("lifecycle %s grammar is absent: command=%v err=%v", name, command, err)
		}
	}
}

func TestValidateLifecycleBootResultEnforcesDurableReportOnlyBoundary(t *testing.T) {
	result := lifecycleBootResultFixture(t, "lifecycle_boot_status")
	raw, _ := json.Marshal(result)
	validated, err := validateLifecycleBootResult(raw, "lifecycle_boot_status", "default", "tops-test", "", "", "", "")
	if err != nil || !validated.JournalPresent || validated.State != "verified" || validated.Generation != 1 {
		t.Fatalf("valid lifecycle journal result was rejected: %+v err=%v", validated, err)
	}
	if _, err := validateLifecycleBootResult(
		raw, "lifecycle_boot_status", "default", "tops-test",
		lifecycleTestDigest("wrong-profile"), "", "", ""); err == nil {
		t.Fatal("lifecycle journal with the wrong expected profile digest was accepted")
	}

	result["apply_authorized"] = true
	raw, _ = json.Marshal(result)
	if _, err := validateLifecycleBootResult(raw, "lifecycle_boot_status", "default", "tops-test", "", "", "", ""); err == nil {
		t.Fatal("apply-authorized lifecycle journal result was accepted")
	}
	result["apply_authorized"] = false

	journal := result["journal"].(map[string]any)
	journal["format_version"] = 2
	journal["journal_digest"] = lifecycleObjectDigest(t, journal, "journal_digest")
	result["journal_digest"] = journal["journal_digest"]
	raw, _ = json.Marshal(result)
	if _, err := validateLifecycleBootResult(raw, "lifecycle_boot_status", "default", "tops-test", "", "", "", ""); err == nil {
		t.Fatal("future lifecycle journal version was accepted")
	}
}

func TestValidateLifecycleBootResultRejectsCriticalOrDigestDrift(t *testing.T) {
	result := lifecycleBootResultFixture(t, "lifecycle_boot_status")
	journal := result["journal"].(map[string]any)
	journal["journal_id"] = "drifted"
	raw, _ := json.Marshal(result)
	if _, err := validateLifecycleBootResult(raw, "lifecycle_boot_status", "default", "tops-test", "", "", "", ""); err == nil {
		t.Fatal("lifecycle journal digest drift was accepted")
	}

	result = lifecycleBootResultFixture(t, "lifecycle_boot_status")
	journal = result["journal"].(map[string]any)
	payload := map[string]any{"future": true}
	payloadJSON, _ := json.Marshal(payload)
	journal["extensions"] = []any{map[string]any{
		"extension_id": "future-extension", "extension_version": "1", "critical": true,
		"payload": payload, "payload_digest": taggedLifecycleDigest(payloadJSON),
	}}
	journal["journal_digest"] = lifecycleObjectDigest(t, journal, "journal_digest")
	result["journal_digest"] = journal["journal_digest"]
	raw, _ = json.Marshal(result)
	if _, err := validateLifecycleBootResult(raw, "lifecycle_boot_status", "default", "tops-test", "", "", "", ""); err == nil {
		t.Fatal("unknown critical lifecycle extension was accepted")
	}
}

func TestValidateLifecycleApplyResultEnforcesMutationAndCompatibilityBoundaries(t *testing.T) {
	fixture := func(operation string) map[string]any {
		return map[string]any{
			"protocol": "symphony.knowledge.lifecycle-apply-result.v1", "operation": operation,
			"compatibility": map[string]any{
				"mode": "full", "process_protocol": "symphony.knowledge.engine-process.v1",
				"journal_read_version": 2, "journal_write_version": 2,
				"missing_capabilities": []any{}, "two_way_procedural_compatibility": true,
				"reason": "client and coordinator share the complete apply-capable v2 contract",
			},
			"journal_present": false, "journal": nil, "journal_digest": nil, "plan": nil,
			"action": nil, "applied_state": nil, "changed": false, "recovered": false,
			"repair_actions": []any{}, "read_only": operation == "lifecycle_apply_status",
			"apply_authorized": operation != "lifecycle_apply_status", "canonical": false,
		}
	}
	validate := func(result map[string]any, operation string) error {
		raw, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		_, err = validateLifecycleApplyResult(raw, operation, "default", "tops-test", "", "", "")
		return err
	}

	status := fixture("lifecycle_apply_status")
	validatedRaw, _ := json.Marshal(status)
	validated, err := validateLifecycleApplyResult(
		validatedRaw, "lifecycle_apply_status", "default", "tops-test", "", "", "",
	)
	if err != nil || validated.JournalPresent || validated.State != "absent" {
		t.Fatalf("valid absent apply status was rejected: %+v err=%v", validated, err)
	}

	status["changed"] = true
	if err := validate(status, "lifecycle_apply_status"); err == nil {
		t.Fatal("apply status carrying a mutation was accepted")
	}
	status = fixture("lifecycle_apply_status")
	status["repair_actions"] = []any{"unexpected repair"}
	if err := validate(status, "lifecycle_apply_status"); err == nil {
		t.Fatal("apply status carrying repair evidence was accepted")
	}
	status = fixture("lifecycle_apply_status")
	status["compatibility"].(map[string]any)["journal_read_version"] = 1
	if err := validate(status, "lifecycle_apply_status"); err == nil {
		t.Fatal("unsupported apply journal version was accepted")
	}

	recover := fixture("lifecycle_apply_recover")
	recover["compatibility"].(map[string]any)["mode"] = "read_only"
	if err := validate(recover, "lifecycle_apply_recover"); err == nil {
		t.Fatal("apply recovery without full compatibility was accepted")
	}
	recover = fixture("lifecycle_apply_recover")
	recover["recovered"] = true
	if err := validate(recover, "lifecycle_apply_recover"); err == nil {
		t.Fatal("recovery evidence without a state change was accepted")
	}
}

func TestSelectExecutableLifecycleActionUsesOnlyExactStagedInstallException(t *testing.T) {
	actionID := "lifecycle-action:" + strings.Repeat("a", 64)
	receiptDigest := lifecycleTestDigest("staged-install")
	blockerActionID := actionID
	install := knowledgelifecycle.PlannedAction{
		ActionID: actionID, ComponentID: "example", Kind: "install", Direction: "forward",
		PrerequisiteActionIDs: []string{}, TargetStateDigest: lifecycleTestDigest("target"),
		ExpectedArtifactDigests: []string{receiptDigest}, ExpectedEvidence: []string{"receipt_integrity"},
		Disposition: "blocked", Blockers: []knowledgelifecycle.Blocker{{
			Class: "dependency_wait", ComponentID: "example", ActionID: &blockerActionID,
			Retryable: true, Detail: "the exact desired package is not present in the observation",
		}},
	}
	selected := selectExecutableLifecycleAction([]knowledgelifecycle.PlannedAction{install}, []string{receiptDigest})
	if selected == nil || selected.ActionID != actionID {
		t.Fatal("exact staged package did not satisfy the isolated package-absence blocker")
	}

	install.PrerequisiteActionIDs = []string{"lifecycle-action:" + strings.Repeat("b", 64)}
	if selected := selectExecutableLifecycleAction([]knowledgelifecycle.PlannedAction{install}, []string{receiptDigest}); selected != nil {
		t.Fatal("staged package bypassed a graph prerequisite")
	}
	install.PrerequisiteActionIDs = []string{}
	install.Blockers = append(install.Blockers, knowledgelifecycle.Blocker{
		Class: "dependency_wait", ComponentID: "example", ActionID: &blockerActionID,
		Retryable: true, Detail: "a separate component dependency is not satisfied",
	})
	if selected := selectExecutableLifecycleAction([]knowledgelifecycle.PlannedAction{install}, []string{receiptDigest}); selected != nil {
		t.Fatal("staged package bypassed an additional dependency blocker")
	}
	install.Blockers = install.Blockers[:1]
	install.Disposition = "waiting"
	if selected := selectExecutableLifecycleAction([]knowledgelifecycle.PlannedAction{install}, []string{receiptDigest}); selected != nil {
		t.Fatal("staged package bypassed planner ordering through a waiting disposition")
	}
}

func lifecycleBootResultFixture(t *testing.T, operation string) map[string]any {
	t.Helper()
	journal := map[string]any{
		"protocol": "symphony.knowledge.lifecycle-boot-journal.v1", "format_version": 1,
		"journal_id": "lifecycle-journal:test", "transaction_id": "lifecycle-transaction:test",
		"operation_id": "operation-test", "generation": 1, "previous_journal_digest": nil,
		"profile_id": "default", "profile_digest": lifecycleTestDigest("profile"),
		"tops_id": "tops-test", "mode": "report", "state": "verified",
		"desired_state_digest":            lifecycleTestDigest("desired"),
		"observation_key":                 lifecycleTestDigest("key"),
		"current_observation_digest":      lifecycleTestDigest("observation"),
		"current_stable_inventory_digest": lifecycleTestDigest("stable"),
		"prior_applied_state_digest":      nil, "current_plan_digest": lifecycleTestDigest("plan"),
		"current_plan_revision": 1, "replan_count": 0, "action_attempts": []any{},
		"blockers": []any{}, "checkpoints": []any{map[string]any{"bounded": true}},
		"compatibility": map[string]any{"bounded": true}, "extensions": []any{},
		"recovery": map[string]any{"state": "clean"}, "started_at": "2026-08-10T00:00:00Z",
		"updated_at": "2026-08-10T00:00:00Z", "closed_at": nil,
		"canonical": false, "apply_authorized": false,
	}
	journal["journal_digest"] = lifecycleObjectDigest(t, journal, "journal_digest")
	return map[string]any{
		"protocol": "symphony.knowledge.lifecycle-boot-result.v1", "operation": operation,
		"compatibility": map[string]any{
			"mode": "full", "process_protocol": "symphony.knowledge.engine-process.v1",
			"journal_read_version": 1, "journal_write_version": 1, "missing_capabilities": []any{},
			"two_way_procedural_compatibility": true, "reason": "full bounded compatibility",
		},
		"journal_present": true, "journal": journal, "journal_digest": journal["journal_digest"],
		"plan": nil, "changed": false, "recovered": false, "repair_actions": []any{},
		"read_only": operation == "lifecycle_boot_status", "apply_authorized": false, "canonical": false,
	}
}

func lifecycleObjectDigest(t *testing.T, object map[string]any, field string) string {
	t.Helper()
	copy := make(map[string]any, len(object)-1)
	for key, value := range object {
		if key != field {
			copy[key] = value
		}
	}
	encoded, err := json.Marshal(copy)
	if err != nil {
		t.Fatal(err)
	}
	return taggedLifecycleDigest(encoded)
}

func lifecyclePlanDigest(t *testing.T, plan map[string]any) string {
	t.Helper()
	copy := make(map[string]any, len(plan))
	for key, value := range plan {
		if key != "plan_digest" {
			copy[key] = value
		}
	}
	encoded, err := json.Marshal(copy)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func lifecycleTestDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

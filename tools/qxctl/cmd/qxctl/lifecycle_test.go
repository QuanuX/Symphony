package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
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

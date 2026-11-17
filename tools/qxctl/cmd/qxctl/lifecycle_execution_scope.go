package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
)

func loadLifecycleExecutionScope(options *knowledgeLifecycleOptions) error {
	if options.executionScopePath == "" {
		return nil
	}
	scope, err := knowledgelifecycle.ReadExecutionScope(options.executionScopePath)
	if err != nil {
		return fmt.Errorf("read execution scope: %w", err)
	}
	if scope.ProfileID != options.profileID || scope.TOPSID != options.topsID {
		return fmt.Errorf("execution scope identifies another profile")
	}
	options.executionScope = &scope
	return nil
}
func lifecycleScopeEvidence(options knowledgeLifecycleOptions, evidence string) string {
	if options.executionScope != nil {
		return evidence + "\nscope=" + options.executionScope.ScopeDigest
	}
	return evidence
}
func bindLifecycleExecutionScope(options knowledgeLifecycleOptions, command map[string]any) {
	if options.executionScope != nil {
		command["protocol"] = "symphony.knowledge.lifecycle-apply-command.v2"
		command["execution_scope"] = options.executionScope
	}
}
func selectScopedLifecycleAction(actions []knowledgelifecycle.PlannedAction, available []string, docking bool, scope *knowledgelifecycle.ExecutionScope, desired knowledgelifecycle.DesiredState) *knowledgelifecycle.PlannedAction {
	if scope == nil {
		return selectExecutableLifecycleActionWithDocking(actions, available, docking)
	}
	eligible := make([]knowledgelifecycle.PlannedAction, 0)
	for _, action := range actions {
		if scope.AllowsAction(action, desired) {
			eligible = append(eligible, action)
		}
	}
	return selectExecutableLifecycleActionWithDocking(eligible, available, docking)
}
func lifecycleScopeConverged(actions []knowledgelifecycle.PlannedAction, fatal int, scope *knowledgelifecycle.ExecutionScope, desired knowledgelifecycle.DesiredState, observation knowledgelifecycle.Observation) bool {
	if scope != nil {
		return scope.EffectsComplete(desired, observation, fatal)
	}
	return lifecyclePlanConverged(actions, fatal)
}

func requireLifecycleResultScope(options knowledgeLifecycleOptions, result validatedLifecycleApplyResult, operation string) (validatedLifecycleApplyResult, error) {
	scope := options.executionScope
	if scope == nil {
		if result.Scoped {
			return result, fmt.Errorf("unscoped request returned scoped state")
		}
		return result, nil
	}
	if !result.Scoped {
		return result, fmt.Errorf("coordinator lacks required scoped apply v2 support")
	}
	if result.ScopeDigest != scope.ScopeDigest && (operation != "lifecycle_apply_status" || !result.ScopeComplete) {
		return result, fmt.Errorf("coordinator result differs from immutable execution scope")
	}
	raw, ok := result.Raw.(map[string]any)
	if !ok {
		return result, fmt.Errorf("invalid scoped result")
	}
	if !result.JournalPresent {
		if result.ScopeComplete || result.ProfileConverged {
			return result, fmt.Errorf("absent scope cannot claim completion")
		}
		return result, nil
	}
	journal, ok := raw["journal"].(map[string]any)
	if !ok {
		return result, fmt.Errorf("invalid scoped journal")
	}
	extensions, ok := journal["extensions"].([]any)
	if !ok {
		return result, fmt.Errorf("scoped journal extensions are invalid")
	}
	found := false
	for _, item := range extensions {
		extension, ok := item.(map[string]any)
		if !ok {
			return result, fmt.Errorf("invalid scoped journal extension")
		}
		if extension["extension_id"] != "scoped-apply-v1" {
			if extension["critical"] == true {
				return result, fmt.Errorf("unknown critical apply extension")
			}
			continue
		}
		if found || !exactLifecycleFields(extension, []string{"extension_id", "extension_version", "critical", "payload", "payload_digest"}) || extension["extension_version"] != "1.0.0" || extension["critical"] != true {
			return result, fmt.Errorf("invalid scoped apply extension identity")
		}
		found = true
		result.HasExecutionScope = true
		stable, stableOK := journal["current_stable_inventory_digest"].(string)
		if !stableOK || !validTaggedDigest(stable) {
			return result, fmt.Errorf("scoped journal lacks stable inventory checkpoint")
		}
		result.StableInventoryDigest = stable
		payload, ok := extension["payload"].(map[string]any)
		if !ok || !exactLifecycleFields(payload, []string{"scope", "scope_complete", "profile_converged", "preserved_components_digest", "action_preserved_components_digest"}) {
			return result, fmt.Errorf("invalid scoped apply extension payload")
		}
		encoded, err := knowledgelifecycle.CanonicalScopeJSON(payload)
		if err != nil || extension["payload_digest"] != taggedLifecycleDigest(encoded) {
			return result, fmt.Errorf("scoped apply extension digest mismatch")
		}
		scopeBytes, err := json.Marshal(payload["scope"])
		if err != nil {
			return result, err
		}
		stored, err := knowledgelifecycle.DecodeExecutionScope(scopeBytes)
		if err != nil {
			return result, err
		}
		actionPreserved, actionPreservedOK := payload["action_preserved_components_digest"].(string)
		if payload["action_preserved_components_digest"] != nil && (!actionPreservedOK || !validTaggedDigest(actionPreserved)) {
			return result, fmt.Errorf("invalid action preservation digest")
		}
		result.ActionPreservedComponentsDigest = actionPreserved
		preserved, ok := payload["preserved_components_digest"].(string)
		result.PreservedComponentsDigest = preserved
		if !ok || !validTaggedDigest(preserved) || stored.ScopeDigest != result.ScopeDigest || payload["scope_complete"] != result.ScopeComplete || payload["profile_converged"] != result.ProfileConverged || stored.ProfileID != options.profileID || stored.TOPSID != options.topsID {
			return result, fmt.Errorf("scope completion differs from durable journal evidence")
		}
		if result.ScopeComplete && (result.State != "verified" || result.ActiveAction != nil) {
			return result, fmt.Errorf("scoped completion requires verified journal without active action")
		}
	}
	if !found && (operation != "lifecycle_apply_status" || result.ScopeComplete || result.State != "closed") {
		return result, fmt.Errorf("scoped apply requires its critical journal extension")
	}
	return result, nil
}

// Check the durable preparation checkpoint before any ownership or package write.
// Only the selected action effect may have changed after interruption; all
// other component fields must still match the durable preparation.
func requireScopeExecutionCheckpoint(options knowledgeLifecycleOptions, result validatedLifecycleApplyResult, observation knowledgelifecycle.Observation) error {
	if options.executionScope == nil {
		return nil
	}
	if result.ActiveAction == nil || result.ScopeDigest != options.executionScope.ScopeDigest || result.PreservedComponentsDigest == "" || result.ActionPreservedComponentsDigest == "" {
		return fmt.Errorf("scoped execution lacks an exact durable preparation checkpoint")
	}
	preserved, err := options.executionScope.PreservedComponentsDigest(observation)
	if err != nil {
		return err
	}
	if preserved != result.PreservedComponentsDigest {
		return fmt.Errorf("outside-scope observation changed since preparation; execution refused")
	}
	actionPreserved, err := knowledgelifecycle.ActionPreservedComponentsDigest(observation, *result.ActiveAction)
	if err != nil {
		return err
	}
	if actionPreserved != result.ActionPreservedComponentsDigest {
		return fmt.Errorf("another component changed since action preparation; execution refused")
	}
	return nil
}

func reconcileScopedCloseOwnership(options knowledgeLifecycleOptions, store *knowledgelifecycle.Store, executor *knowledgelifecycle.Executor, profileDigest string, profile knowledgelifecycle.Profile, observation knowledgelifecycle.Observation, last validatedLifecycleApplyResult, plan validatedLifecyclePlan, stable, journalDigest, appliedDigest string) error {
	scope := options.executionScope
	if err := scope.ValidateDesired(profileDigest, profile.DesiredState, options.sourceJournalDigest); err != nil {
		return err
	}
	if last.HasExecutionScope && last.ScopeDigest == scope.ScopeDigest {
		preserved, err := scope.PreservedComponentsDigest(observation)
		if err != nil {
			return err
		}
		if preserved != last.PreservedComponentsDigest || stable != last.StableInventoryDigest {
			return fmt.Errorf("scoped completion observation changed since durable checkpoint")
		}
	} else {
		semanticPlan, err := knowledgelifecycle.InitialScopePlanDigest(plan.Raw, stable)
		if err != nil {
			return err
		}
		if err := scope.ValidateInitial(profile.DesiredState, observation, semanticPlan, stable, journalDigest, appliedDigest); err != nil {
			return err
		}
	}
	source, err := invokeKnowledgeLifecycleBootState("lifecycle_boot_status", options)
	if err != nil {
		return err
	}
	sourceObject, ok := source.Raw.(map[string]any)
	if !ok {
		return fmt.Errorf("source report evidence is absent")
	}
	sourceJournal, ok := sourceObject["journal"].(map[string]any)
	if !source.JournalPresent || !ok || source.JournalDigest != options.sourceJournalDigest || sourceJournal["mode"] != "apply-compatible" || sourceJournal["profile_digest"] != profileDigest || sourceJournal["desired_state_digest"] != profile.DesiredState.DesiredStateDigest {
		return fmt.Errorf("scoped completion source report evidence does not match")
	}
	if err := authorizeKnowledgeLifecycle(options, "ownership.reconcile", lifecycleResource(options.topsID, options.profileID, lifecycleScopeEvidence(options, profileDigest+"\n"+observation.ObservationDigest))); err != nil {
		return err
	}
	return store.WithProfileSnapshot(options.profileID, func(locked knowledgelifecycle.Profile) error {
		if locked.ProfileDigest != profileDigest || locked.DesiredState.DesiredStateDigest != profile.DesiredState.DesiredStateDigest {
			return fmt.Errorf("profile changed before scoped ownership completion")
		}
		_, err := executor.ReconcileOwnership(locked.DesiredState, observation)
		return err
	})
}

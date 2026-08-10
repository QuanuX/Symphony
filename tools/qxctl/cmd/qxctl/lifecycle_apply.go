package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
	qxversion "github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

type validatedLifecycleApplyResult struct {
	Raw            any
	JournalPresent bool
	JournalDigest  string
	State          string
	ProfileDigest  string
	DesiredDigest  string
	SourceDigest   string
	ActiveAction   *knowledgelifecycle.PlannedAction
	AppliedDigest  string
	Plan           *validatedLifecyclePlan
	Changed        bool
	Recovered      bool
}

func lifecycleApplyClient() map[string]any {
	return map[string]any{
		"client_id": "qxctl", "client_version": strings.ReplaceAll(qxversion.Version, " ", "-"),
		"process_protocols":     []string{"symphony.knowledge.engine-process.v1"},
		"journal_read_versions": []uint64{1, 2}, "journal_write_versions": []uint64{2},
		"capabilities": []string{
			"action-attempt-journal-v2", "applied-state-v1", "discovery-recovery-v2",
			"dynamic-replanning-v1", "expected-state-cas-v1", "external-action-adapter-v1",
			"forward-inverse-v1", "opaque-extension-preservation-v1", "per-action-authorization-v1",
			"recovery-forward-v1", "staged-package-v2", "verified-observation-commit-v1",
		},
	}
}

func runKnowledgeLifecycleApply(options knowledgeLifecycleOptions) error {
	if options.operationID == "" || options.sourceJournalDigest == "" ||
		options.expectedApplyDigest == "" || options.expectedAppliedDigest == "" {
		return fmt.Errorf("--operation-id, --source-journal-digest, --expected-apply-journal-digest, and --expected-applied-state-digest are required")
	}
	if !validSessionToken(options.operationID) || !validTaggedDigest(options.sourceJournalDigest) ||
		options.expectedApplyDigest != "absent" && !validTaggedDigest(options.expectedApplyDigest) ||
		options.expectedAppliedDigest != "absent" && !validTaggedDigest(options.expectedAppliedDigest) {
		return fmt.Errorf("lifecycle apply mutation identities must be exact tokens, absent, or tagged SHA-256 digests")
	}
	if options.maxActions == 0 || options.maxActions > 4096 {
		return fmt.Errorf("--max-actions must be between 1 and 4096")
	}
	observation, profileDigest, profile, err := buildLifecycleObservation(options, false)
	if err != nil {
		return err
	}
	if profile.BootMode != "apply-compatible" {
		return fmt.Errorf("lifecycle profile %q is not apply-compatible", profile.ProfileID)
	}
	store, err := lifecycleStore(options)
	if err != nil {
		return err
	}
	repositoryRoot, err := resolveKnowledgeRepository(options.repository)
	if err != nil {
		return err
	}
	coordinator, err := exactBoundCoordinator(options.stateRoot)
	if err != nil {
		return err
	}
	executor, err := knowledgelifecycle.NewExecutor(store.StateRoot(), options.topsID, options.profileID, options.stagedRoots)
	if err != nil {
		return err
	}
	status, err := invokeLifecycleApplyState(
		options, coordinator.Prefix, coordinator.Version, repositoryRoot, "lifecycle_apply_status", "", "")
	if err != nil {
		return err
	}
	if err := requireApplyCAS(status, options.expectedApplyDigest, options.expectedAppliedDigest); err != nil {
		return err
	}
	journalDigest := options.expectedApplyDigest
	appliedDigest := options.expectedAppliedDigest
	active := status.ActiveAction
	last := status
	var available []string

	for sequence := uint64(0); sequence < options.maxActions; sequence++ {
		observation, profileDigest, profile, err = buildLifecycleObservation(options, false)
		if err != nil {
			return err
		}
		available, err = executor.AvailableArtifactDigests()
		if err != nil {
			return err
		}
		stableDigest, err := knowledgelifecycle.StableInventoryDigest(observation)
		if err != nil {
			return err
		}
		prior := appliedDigest
		if prior == "absent" {
			prior = ""
		}

		if active == nil {
			plan, err := invokeLifecyclePlan(
				options, coordinator.Prefix, coordinator.Version, repositoryRoot,
				profileDigest, profile.DesiredState, observation, prior)
			if err != nil {
				return err
			}
			selected := selectExecutableLifecycleAction(plan.Actions, available)
			if selected == nil {
				if lifecyclePlanConverged(plan.Actions, plan.FatalCount) {
					operationID := applyOperationID(options.operationID, "converged")
					last, err = invokeLifecycleApplyMutation(
						options, coordinator.Prefix, coordinator.Version, repositoryRoot,
						"lifecycle_apply_close", operationID, journalDigest, appliedDigest,
						profileDigest, stableDigest, profile.DesiredState, observation,
						prior, nil, available, nil)
					if err != nil {
						return err
					}
					return printLifecycleApplyResult(options, last, sequence)
				}
				return fmt.Errorf("lifecycle apply has no executable ready action; %d actions remain blocked or waiting", plan.ActionCount)
			}
			active = selected
			operationID := applyOperationID(options.operationID, active.ActionID)
			prepared, err := invokeLifecycleApplyMutation(
				options, coordinator.Prefix, coordinator.Version, repositoryRoot,
				"lifecycle_apply_prepare", operationID, journalDigest, appliedDigest,
				profileDigest, stableDigest, profile.DesiredState, observation,
				prior, active, available, nil)
			if err != nil {
				return err
			}
			if prepared.ActiveAction == nil || prepared.ActiveAction.ActionID != active.ActionID {
				return fmt.Errorf("coordinator prepared a different lifecycle action")
			}
			active = prepared.ActiveAction
			journalDigest = prepared.JournalDigest
			last = prepared
		}
		if last.ProfileDigest != profileDigest ||
			last.DesiredDigest != profile.DesiredState.DesiredStateDigest ||
			last.SourceDigest != options.sourceJournalDigest {
			return fmt.Errorf("prepared lifecycle action belongs to different profile, desired-state, or source-journal evidence; restore that evidence before resuming")
		}

		execution := executor.Execute(*active, profile.DesiredState, observation)
		after, afterProfileDigest, afterProfile, err := buildLifecycleObservation(options, false)
		if err != nil {
			return err
		}
		if afterProfileDigest != profileDigest || afterProfile.DesiredState.DesiredStateDigest != profile.DesiredState.DesiredStateDigest {
			return fmt.Errorf("lifecycle desired profile changed during an action; prepared evidence remains resumable")
		}
		afterStable, err := knowledgelifecycle.StableInventoryDigest(after)
		if err != nil {
			return err
		}
		operationID := applyOperationID(options.operationID, active.ActionID)
		finalized, finalizeErr := invokeLifecycleApplyMutation(
			options, coordinator.Prefix, coordinator.Version, repositoryRoot,
			"lifecycle_apply_finalize", operationID, journalDigest, appliedDigest,
			profileDigest, afterStable, profile.DesiredState, after,
			prior, active, available, &execution)
		if finalizeErr != nil {
			return fmt.Errorf("lifecycle action executed but durable verification did not finalize; retry with the current apply journal: %w", finalizeErr)
		}
		last = finalized
		journalDigest = finalized.JournalDigest
		if finalized.AppliedDigest != "" {
			appliedDigest = finalized.AppliedDigest
		}
		active = finalized.ActiveAction
		if execution.Outcome != "committed" && execution.Outcome != "already_applied" {
			return fmt.Errorf("lifecycle action %s was durably recorded as %s: %s", execution.ActionID, execution.Outcome, execution.Detail)
		}
		if finalized.State == "closed" {
			return printLifecycleApplyResult(options, finalized, sequence+1)
		}
	}
	return fmt.Errorf("lifecycle apply reached its --max-actions bound at journal %s; resume with exact returned state", last.JournalDigest)
}

func invokeLifecyclePlan(
	options knowledgeLifecycleOptions,
	prefix, version, repositoryRoot, profileDigest string,
	desired knowledgelifecycle.DesiredState,
	observation knowledgelifecycle.Observation,
	prior string,
) (validatedLifecyclePlan, error) {
	stable, err := knowledgelifecycle.StableInventoryDigest(observation)
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	if err := authorizeKnowledgeLifecycle(options, "report", lifecycleResource(
		options.topsID, options.profileID, profileDigest+"\n"+stable)); err != nil {
		return validatedLifecyclePlan{}, err
	}
	var priorValue any
	if prior != "" {
		priorValue = prior
	}
	payload, err := json.Marshal(map[string]any{
		"protocol": "symphony.knowledge.lifecycle-plan-command.v1", "operation": "lifecycle_plan",
		"desired_state": desired, "observation": observation,
		"prior_applied_state_digest": priorValue, "client": lifecyclePlannerClient(),
	})
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(), prefix, version, repositoryRoot, "lifecycle_plan", payload)
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	return validateLifecyclePlan(response.Result, desired.DesiredStateDigest, observation.ObservationDigest, prior)
}

func invokeLifecycleApplyMutation(
	options knowledgeLifecycleOptions,
	prefix, version, repositoryRoot, operation, operationID, expectedJournal, expectedApplied,
	profileDigest, stableDigest string,
	desired knowledgelifecycle.DesiredState,
	observation knowledgelifecycle.Observation,
	prior string,
	action *knowledgelifecycle.PlannedAction,
	available []string,
	execution *knowledgelifecycle.ExecutionResult,
) (validatedLifecycleApplyResult, error) {
	stateRoot, err := lifecycleStateRoot(options)
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	actionEvidence := "converged"
	var actionID any
	if action != nil {
		actionEvidence = action.ActionID
		actionID = action.ActionID
	}
	evidence := profileDigest + "\n" + desired.DesiredStateDigest + "\n" + stableDigest + "\n" +
		options.sourceJournalDigest + "\n" + expectedJournal + "\n" + actionEvidence
	permission := "apply." + strings.TrimPrefix(operation, "lifecycle_apply_")
	decision, err := authorizeKnowledgeLifecycleDecision(
		options, permission, lifecycleResource(options.topsID, options.profileID, evidence))
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	var priorValue any
	if prior != "" {
		priorValue = prior
	}
	var outcome, blockerClass, executionDigest any
	if execution != nil {
		outcome = execution.Outcome
		blockerClass = execution.BlockerClass
		executionDigest = execution.EvidenceDigest
	}
	payload, err := json.Marshal(map[string]any{
		"protocol": "symphony.knowledge.lifecycle-apply-command.v1", "operation": operation,
		"state_root": stateRoot, "operation_id": operationID,
		"expected_journal_digest": expectedJournal, "expected_applied_state_digest": expectedApplied,
		"source_report_journal_digest": options.sourceJournalDigest,
		"profile_id":                   options.profileID, "tops_id": options.topsID, "profile_digest": profileDigest,
		"stable_inventory_digest": stableDigest, "desired_state": desired, "observation": observation,
		"prior_applied_state_digest": priorValue, "action_id": actionID,
		"available_artifact_digests": available, "outcome": outcome, "blocker_class": blockerClass,
		"execution_evidence_digest": executionDigest, "authorization_decision": decision,
		"planner_client": lifecyclePlannerClient(), "journal_client": lifecycleApplyClient(),
	})
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(), prefix, version, repositoryRoot, operation, payload)
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	return validateLifecycleApplyResult(
		response.Result, operation, options.profileID, options.topsID,
		desired.DesiredStateDigest, observation.ObservationDigest, prior)
}

func invokeLifecycleApplyState(
	options knowledgeLifecycleOptions,
	prefix, version, repositoryRoot, operation, operationID, expected string,
) (validatedLifecycleApplyResult, error) {
	stateRoot, err := lifecycleStateRoot(options)
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	evidence := "status"
	permission := "apply.status"
	var operationValue, expectedValue any
	if operation == "lifecycle_apply_recover" {
		evidence, permission = expected, "apply.recover"
		operationValue, expectedValue = operationID, expected
	}
	decision, err := authorizeKnowledgeLifecycleDecision(
		options, permission, lifecycleResource(options.topsID, options.profileID, evidence))
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	payload, err := json.Marshal(map[string]any{
		"protocol": "symphony.knowledge.lifecycle-apply-command.v1", "operation": operation,
		"state_root": stateRoot, "operation_id": operationValue,
		"expected_journal_digest": expectedValue, "profile_id": options.profileID, "tops_id": options.topsID,
		"authorization_decision": decision, "journal_client": lifecycleApplyClient(),
	})
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(), prefix, version, repositoryRoot, operation, payload)
	if err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	return validateLifecycleApplyResult(response.Result, operation, options.profileID, options.topsID, "", "", "")
}

func runKnowledgeLifecycleApplyState(operation string, options knowledgeLifecycleOptions) error {
	if operation == "lifecycle_apply_recover" {
		if options.operationID == "" {
			return fmt.Errorf("--operation-id is required")
		}
		if options.discover && options.expectedApplyDigest != "" {
			return fmt.Errorf("--discover and --expected-apply-journal-digest are mutually exclusive")
		}
		if options.discover {
			options.expectedApplyDigest = "discover"
		}
		if options.expectedApplyDigest == "" ||
			options.expectedApplyDigest != "discover" && !validTaggedDigest(options.expectedApplyDigest) {
			return fmt.Errorf("--expected-apply-journal-digest or --discover is required")
		}
	}
	if _, err := lifecycleStore(options); err != nil {
		return err
	}
	repositoryRoot, err := resolveKnowledgeRepository(options.repository)
	if err != nil {
		return err
	}
	coordinator, err := exactBoundCoordinator(options.stateRoot)
	if err != nil {
		return err
	}
	result, err := invokeLifecycleApplyState(
		options, coordinator.Prefix, coordinator.Version, repositoryRoot, operation,
		options.operationID, options.expectedApplyDigest)
	if err != nil {
		return err
	}
	return printLifecycleApplyResult(options, result, 0)
}

func selectExecutableLifecycleAction(actions []knowledgelifecycle.PlannedAction, available []string) *knowledgelifecycle.PlannedAction {
	availableSet := make(map[string]struct{}, len(available))
	for _, digest := range available {
		availableSet[digest] = struct{}{}
	}
	candidates := make([]knowledgelifecycle.PlannedAction, 0)
	for _, action := range actions {
		if action.Kind == "dock" || action.Kind == "undock" || action.Kind == "verify" ||
			action.Kind == "preserve" || action.Kind == "report" {
			continue
		}
		ready := action.Disposition == "ready"
		if action.Kind == "install" && action.Disposition == "blocked" &&
			len(action.ExpectedArtifactDigests) != 0 && len(action.PrerequisiteActionIDs) == 0 && len(action.Blockers) == 1 {
			ready = true
			for _, digest := range action.ExpectedArtifactDigests {
				_, ready = availableSet[digest]
				if !ready {
					break
				}
			}
			blocker := action.Blockers[0]
			if blocker.Class != "dependency_wait" || blocker.ComponentID != action.ComponentID ||
				blocker.ActionID == nil || *blocker.ActionID != action.ActionID {
				ready = false
			}
		}
		if ready {
			candidates = append(candidates, action)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ActionID < candidates[j].ActionID })
	if len(candidates) == 0 {
		return nil
	}
	return &candidates[0]
}

func lifecyclePlanConverged(actions []knowledgelifecycle.PlannedAction, fatal int) bool {
	if fatal != 0 {
		return false
	}
	for _, action := range actions {
		if action.Kind != "preserve" && action.Kind != "report" {
			return false
		}
	}
	return true
}

func applyOperationID(base, action string) string {
	digest := sha256.Sum256([]byte(base + "\n" + action))
	return "lifecycle-apply:" + hex.EncodeToString(digest[:])
}

func lifecycleStateRoot(options knowledgeLifecycleOptions) (string, error) {
	store, err := lifecycleStore(options)
	if err != nil {
		return "", err
	}
	return store.StateRoot(), nil
}

func requireApplyCAS(result validatedLifecycleApplyResult, expectedJournal, expectedApplied string) error {
	actualJournal := "absent"
	if result.JournalPresent {
		actualJournal = result.JournalDigest
	}
	actualApplied := "absent"
	if result.AppliedDigest != "" {
		actualApplied = result.AppliedDigest
	}
	if actualJournal != expectedJournal || actualApplied != expectedApplied {
		return fmt.Errorf("lifecycle apply compare-and-swap mismatch: current journal=%s applied=%s", actualJournal, actualApplied)
	}
	return nil
}

func printLifecycleApplyResult(options knowledgeLifecycleOptions, result validatedLifecycleApplyResult, actions uint64) error {
	if options.jsonOutput {
		return printIndentedJSON(result.Raw)
	}
	fmt.Printf("Knowledge lifecycle apply: profile=%s state=%s actions=%d journal_digest=%s applied_state_digest=%s canonical=false\n",
		options.profileID, result.State, actions, nullableDigest(result.JournalDigest), nullableDigest(result.AppliedDigest))
	return nil
}

func nullableDigest(value string) string {
	if value == "" {
		return "absent"
	}
	return value
}

func validateLifecycleApplyResult(
	raw json.RawMessage, operation, profileID, topsID, desiredDigest, observationDigest, priorDigest string,
) (validatedLifecycleApplyResult, error) {
	if err := knowledgeengine.ValidateJSONObject(raw, 4*1024*1024); err != nil {
		return validatedLifecycleApplyResult{}, fmt.Errorf("invalid lifecycle apply result JSON: %w", err)
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	if !exactLifecycleFields(object, []string{
		"protocol", "operation", "compatibility", "journal_present", "journal", "journal_digest",
		"plan", "action", "applied_state", "changed", "recovered", "repair_actions", "read_only",
		"apply_authorized", "canonical",
	}) || object["protocol"] != "symphony.knowledge.lifecycle-apply-result.v1" || object["operation"] != operation ||
		object["canonical"] != false {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply result identity is invalid")
	}
	readOnly, readOK := object["read_only"].(bool)
	changed, changedOK := object["changed"].(bool)
	recovered, recoveredOK := object["recovered"].(bool)
	present, presentOK := object["journal_present"].(bool)
	if !readOK || !changedOK || !recoveredOK || !presentOK ||
		(operation == "lifecycle_apply_status") != readOnly || object["apply_authorized"] != !readOnly ||
		(operation == "lifecycle_apply_status" && (changed || recovered || object["plan"] != nil)) ||
		(operation == "lifecycle_apply_recover" && (object["plan"] != nil || changed != recovered)) ||
		(operation != "lifecycle_apply_recover" && recovered) {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply mutation boundary is invalid")
	}
	compatibility, ok := object["compatibility"].(map[string]any)
	if !ok || !exactLifecycleFields(compatibility, []string{
		"mode", "process_protocol", "journal_read_version", "journal_write_version", "missing_capabilities",
		"two_way_procedural_compatibility", "reason",
	}) || !oneOfInterface(compatibility["mode"], "full", "read_only", "blocked") ||
		compatibility["two_way_procedural_compatibility"] != true ||
		!validUniqueTokenArray(compatibility["missing_capabilities"], 0, 128) {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply compatibility is invalid")
	}
	if reason, ok := compatibility["reason"].(string); !ok || !validPlanText(reason) {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply compatibility reason is invalid")
	}
	if compatibility["process_protocol"] != nil &&
		compatibility["process_protocol"] != "symphony.knowledge.engine-process.v1" {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply process protocol is unsupported")
	}
	for _, field := range []string{"journal_read_version", "journal_write_version"} {
		if compatibility[field] != nil && compatibility[field] != json.Number("2") {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply compatibility version is unsupported")
		}
	}
	if operation != "lifecycle_apply_status" && compatibility["mode"] != "full" {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply mutation requires full compatibility")
	}
	if compatibility["mode"] == "full" &&
		(compatibility["process_protocol"] != "symphony.knowledge.engine-process.v1" ||
			compatibility["journal_read_version"] != json.Number("2") ||
			compatibility["journal_write_version"] != json.Number("2") ||
			!validUniqueTokenArray(compatibility["missing_capabilities"], 0, 0)) {
		return validatedLifecycleApplyResult{}, fmt.Errorf("full lifecycle apply compatibility is internally inconsistent")
	}
	repairActions, ok := object["repair_actions"].([]any)
	if !ok || len(repairActions) > 64 || (operation != "lifecycle_apply_recover" && len(repairActions) != 0) {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply repair evidence is invalid")
	}
	for _, item := range repairActions {
		value, ok := item.(string)
		if !ok || !validPlanText(value) {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply repair evidence is invalid")
		}
	}
	result := validatedLifecycleApplyResult{
		JournalPresent: present, Changed: changed, Recovered: recovered, State: "absent",
	}
	if present {
		journal, ok := object["journal"].(map[string]any)
		digest, digestOK := object["journal_digest"].(string)
		profileDigest, profileOK := journal["profile_digest"].(string)
		desiredStateDigest, desiredOK := journal["desired_state_digest"].(string)
		sourceDigest, sourceOK := journal["source_report_journal_digest"].(string)
		observationValue, observationOK := journal["current_observation_digest"].(string)
		if !ok || !digestOK || !validTaggedDigest(digest) || journal["journal_digest"] != digest ||
			!profileOK || !validTaggedDigest(profileDigest) ||
			!desiredOK || !validTaggedDigest(desiredStateDigest) ||
			!sourceOK || !validTaggedDigest(sourceDigest) ||
			!observationOK || !validTaggedDigest(observationValue) ||
			journal["protocol"] != "symphony.knowledge.lifecycle-boot-journal.v2" ||
			journal["format_version"] != json.Number("2") || journal["profile_id"] != profileID ||
			journal["tops_id"] != topsID || journal["mode"] != "apply-compatible" ||
			journal["canonical"] != false || journal["apply_authorized"] != true ||
			!oneOfInterface(journal["state"], "open", "acting", "blocked", "verified", "closed") {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply journal identity is invalid")
		}
		copy := make(map[string]any, len(journal)-1)
		for key, value := range journal {
			if key != "journal_digest" {
				copy[key] = value
			}
		}
		encoded, err := json.Marshal(copy)
		if err != nil || taggedLifecycleDigest(encoded) != digest {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply journal digest mismatch")
		}
		result.JournalDigest = digest
		result.State = journal["state"].(string)
		result.ProfileDigest = profileDigest
		result.DesiredDigest = desiredStateDigest
		result.SourceDigest = sourceDigest
		if desiredDigest != "" && desiredDigest != desiredStateDigest ||
			observationDigest != "" && observationDigest != observationValue {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply journal does not bind the requested evidence")
		}
		journalAction := journal["active_action"]
		if result.State == "acting" {
			if journalAction == nil || object["action"] == nil || !lifecycleJSONEqual(journalAction, object["action"]) {
				return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply active action evidence is inconsistent")
			}
		} else if journalAction != nil || object["action"] != nil {
			return validatedLifecycleApplyResult{}, fmt.Errorf("non-acting lifecycle apply result carries an active action")
		}
	} else if object["journal"] != nil || object["journal_digest"] != nil || object["action"] != nil ||
		object["applied_state"] != nil || object["plan"] != nil || changed || recovered {
		return validatedLifecycleApplyResult{}, fmt.Errorf("absent lifecycle apply result carries state")
	}
	if object["action"] != nil {
		action, err := knowledgelifecycle.DecodePlannedAction(object["action"])
		if err != nil {
			return validatedLifecycleApplyResult{}, err
		}
		result.ActiveAction = &action
	}
	if object["applied_state"] != nil {
		applied, ok := object["applied_state"].(map[string]any)
		digest, digestOK := applied["applied_state_digest"].(string)
		if !ok || !digestOK || !validTaggedDigest(digest) ||
			applied["protocol"] != "symphony.knowledge.lifecycle-applied-state.v1" ||
			applied["profile_id"] != profileID || applied["tops_id"] != topsID || applied["canonical"] != false {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle applied state identity is invalid")
		}
		copy := make(map[string]any, len(applied)-1)
		for key, value := range applied {
			if key != "applied_state_digest" {
				copy[key] = value
			}
		}
		encoded, err := json.Marshal(copy)
		if err != nil || taggedLifecycleDigest(encoded) != digest {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle applied state digest mismatch")
		}
		result.AppliedDigest = digest
	}
	if present {
		journal := object["journal"].(map[string]any)
		journalApplied := journal["applied_state_digest"]
		if result.AppliedDigest == "" && journalApplied != nil ||
			result.AppliedDigest != "" && journalApplied != result.AppliedDigest {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle applied state and journal selector differ")
		}
	} else if result.AppliedDigest != "" {
		return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle applied state exists without a journal selector")
	}
	if object["plan"] != nil {
		if desiredDigest == "" || observationDigest == "" {
			return validatedLifecycleApplyResult{}, fmt.Errorf("lifecycle apply state operation returned an unexpected plan")
		}
		encoded, err := json.Marshal(object["plan"])
		if err != nil {
			return validatedLifecycleApplyResult{}, err
		}
		plan, err := validateLifecyclePlan(encoded, desiredDigest, observationDigest, priorDigest)
		if err != nil {
			return validatedLifecycleApplyResult{}, err
		}
		result.Plan = &plan
	}
	if err := json.Unmarshal(raw, &result.Raw); err != nil {
		return validatedLifecycleApplyResult{}, err
	}
	return result, nil
}

func lifecycleJSONEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

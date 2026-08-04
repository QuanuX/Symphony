package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	qxversion "github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

func runKnowledgeLifecycleProfile(operation string, options knowledgeLifecycleOptions) error {
	store, err := lifecycleStore(options)
	if err != nil {
		return err
	}
	switch operation {
	case "list":
		result, err := store.List()
		if err != nil {
			return err
		}
		if err := authorizeKnowledgeLifecycle(options, "profile.list", lifecycleResource(
			options.topsID, "profiles", result.ListDigest)); err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(result)
		}
		fmt.Printf("Knowledge lifecycle profiles: tops_id=%s count=%d digest=%s canonical=false\n",
			result.TOPSID, len(result.Profiles), result.ListDigest)
		for _, profile := range result.Profiles {
			fmt.Printf("Knowledge lifecycle profile: profile=%s generation=%d mode=%s components=%d digest=%s\n",
				profile.ProfileID, profile.Generation, profile.BootMode, profile.ComponentCount, profile.ProfileDigest)
		}
		return nil
	case "show":
		snapshot, err := store.Snapshot(options.profileID)
		if err != nil {
			return err
		}
		if !snapshot.Exists {
			return fmt.Errorf("lifecycle profile %q is absent", options.profileID)
		}
		if err := authorizeKnowledgeLifecycle(options, "profile.show", lifecycleResource(
			options.topsID, options.profileID, snapshot.Profile.ProfileDigest)); err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(snapshot.Profile)
		}
		fmt.Printf("Knowledge lifecycle profile: profile=%s generation=%d mode=%s components=%d roots=%d digest=%s canonical=false\n",
			snapshot.Profile.ProfileID, snapshot.Profile.Generation, snapshot.Profile.BootMode,
			len(snapshot.Profile.DesiredState.Components), len(snapshot.Profile.ConfiguredRoots), snapshot.Profile.ProfileDigest)
		return nil
	case "set":
		if options.input == "" || options.expectedProfileDigest == "" {
			return fmt.Errorf("--input and --expected-profile-digest are required")
		}
		data, err := knowledgeengine.ReadPayload(options.input)
		if err != nil {
			return err
		}
		input, err := knowledgelifecycle.DecodeProfileInput(data)
		if err != nil {
			return err
		}
		if input.ProfileID != options.profileID || input.TOPSID != options.topsID {
			return fmt.Errorf("profile input identity does not match --profile-id and --tops-id")
		}
		inputDigest, err := knowledgelifecycle.ProfileInputDigest(input)
		if err != nil {
			return err
		}
		if err := authorizeKnowledgeLifecycle(options, "profile.set", lifecycleResource(
			options.topsID, options.profileID, options.expectedProfileDigest+"\n"+inputDigest)); err != nil {
			return err
		}
		profile, changed, err := store.Set(input, options.expectedProfileDigest)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"changed": changed, "profile": profile})
		}
		fmt.Printf("Knowledge lifecycle profile: operation=set profile=%s changed=%t generation=%d mode=%s digest=%s canonical=false\n",
			profile.ProfileID, changed, profile.Generation, profile.BootMode, profile.ProfileDigest)
		if profile.BootMode == "apply-compatible" {
			fmt.Println("Knowledge lifecycle profile: apply-compatible requested; runtime apply remains unavailable and reports only")
		}
		return nil
	case "remove":
		if options.expectedProfileDigest == "" {
			return fmt.Errorf("--expected-profile-digest is required")
		}
		if err := authorizeKnowledgeLifecycle(options, "profile.remove", lifecycleResource(
			options.topsID, options.profileID, options.expectedProfileDigest)); err != nil {
			return err
		}
		changed, err := store.Remove(options.profileID, options.expectedProfileDigest)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{
				"schema":     "qxctl.knowledge.lifecycle-profile-removal.v1",
				"profile_id": options.profileID, "tops_id": options.topsID,
				"changed": changed, "canonical": false,
			})
		}
		fmt.Printf("Knowledge lifecycle profile: operation=remove profile=%s changed=%t canonical=false\n",
			options.profileID, changed)
		return nil
	default:
		return fmt.Errorf("unsupported knowledge lifecycle profile operation")
	}
}

func runKnowledgeLifecycleObserve(options knowledgeLifecycleOptions) error {
	observation, profileDigest, _, err := buildLifecycleObservation(options, len(options.configuredRoots) != 0)
	if err != nil {
		return err
	}
	stableDigest, err := knowledgelifecycle.StableInventoryDigest(observation)
	if err != nil {
		return err
	}
	if err := authorizeKnowledgeLifecycle(options, "observe", lifecycleResource(
		options.topsID, options.profileID, profileDigest+"\n"+stableDigest)); err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(observation)
	}
	fmt.Printf("Knowledge lifecycle observation: profile=%s components=%d unknown=%d roots=%d stable_digest=%s document_digest=%s canonical=false\n",
		observation.ProfileID, len(observation.Components), len(observation.UnknownPackages),
		len(observation.ConfiguredRoots), stableDigest, observation.ObservationDigest)
	return nil
}

func runKnowledgeLifecycleReport(options knowledgeLifecycleOptions) error {
	if options.priorAppliedStateDigest != "" && !validTaggedDigest(options.priorAppliedStateDigest) {
		return fmt.Errorf("--prior-applied-state-digest must be an exact tagged SHA-256 digest")
	}
	observation, profileDigest, profile, err := buildLifecycleObservation(options, false)
	if err != nil {
		return err
	}
	stableDigest, err := knowledgelifecycle.StableInventoryDigest(observation)
	if err != nil {
		return err
	}
	if err := authorizeKnowledgeLifecycle(options, "report", lifecycleResource(
		options.topsID, options.profileID, profileDigest+"\n"+stableDigest)); err != nil {
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
	var prior any
	if options.priorAppliedStateDigest != "" {
		prior = options.priorAppliedStateDigest
	}
	payload, err := json.Marshal(map[string]any{
		"protocol":                   "symphony.knowledge.lifecycle-plan-command.v1",
		"operation":                  "lifecycle_plan",
		"desired_state":              profile.DesiredState,
		"observation":                observation,
		"prior_applied_state_digest": prior,
		"client": map[string]any{
			"client_id": "qxctl", "client_version": strings.ReplaceAll(qxversion.Version, " ", "-"),
			"process_protocols":           []string{"symphony.knowledge.engine-process.v1"},
			"desired_state_read_versions": []uint64{1}, "observation_read_versions": []uint64{1},
			"plan_read_versions": []uint64{1}, "applied_state_read_versions": []uint64{1},
			"receipt_read_versions": []uint64{1, 2},
			"capabilities": []string{
				"dependency-ready-set-v1", "deterministic-action-id-v1", "forward-inverse-v1",
				"localized-blocker-isolation-v1", "ordered-safety-phases-v1",
				"receipt-v1-adapter", "receipt-v2", "report-only-v1", "unknown-critical-block-v1",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("encode lifecycle report request: %w", err)
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(), coordinator.Prefix, coordinator.Version, repositoryRoot, "lifecycle_plan", payload)
	if err != nil {
		return err
	}
	plan, err := validateLifecyclePlan(
		response.Result, profile.DesiredState.DesiredStateDigest, observation.ObservationDigest,
		options.priorAppliedStateDigest)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(plan.Raw)
	}
	fmt.Printf("Knowledge lifecycle report: profile=%s transaction=%s actions=%d ready=%d waiting=%d blocked=%d fatal=%d plan_digest=%s apply_authorized=false canonical=false\n",
		options.profileID, plan.TransactionID, plan.ActionCount, plan.ReadyCount,
		plan.DeferredCount, plan.BlockedCount, plan.FatalCount, plan.PlanDigest)
	if profile.BootMode == "apply-compatible" {
		fmt.Println("Knowledge lifecycle report: apply-compatible requested; lifecycle mutation remains unavailable")
	}
	return nil
}

func lifecycleStore(options knowledgeLifecycleOptions) (*knowledgelifecycle.Store, error) {
	if options.topsID == "" {
		return nil, fmt.Errorf("--tops-id is required")
	}
	if options.scope != "user" && options.scope != "system" {
		return nil, fmt.Errorf("--scope must be user or system")
	}
	if options.ttl <= 0 || options.ttl > 24*time.Hour {
		return nil, fmt.Errorf("--ttl must be greater than zero and no more than 24h")
	}
	return knowledgelifecycle.NewStore(options.stateRoot, options.topsID)
}

func buildLifecycleObservation(
	options knowledgeLifecycleOptions,
	explicitRoots bool,
) (knowledgelifecycle.Observation, string, knowledgelifecycle.Profile, error) {
	store, err := lifecycleStore(options)
	if err != nil {
		return knowledgelifecycle.Observation{}, "", knowledgelifecycle.Profile{}, err
	}
	var profile knowledgelifecycle.Profile
	profileDigest := "bootstrap"
	roots := append([]string(nil), options.configuredRoots...)
	var desired *knowledgelifecycle.DesiredState
	if !explicitRoots {
		snapshot, err := store.Snapshot(options.profileID)
		if err != nil {
			return knowledgelifecycle.Observation{}, "", profile, err
		}
		if !snapshot.Exists {
			return knowledgelifecycle.Observation{}, "", profile, fmt.Errorf("lifecycle profile %q is absent", options.profileID)
		}
		profile = snapshot.Profile
		profileDigest = profile.ProfileDigest
		roots = append([]string(nil), profile.ConfiguredRoots...)
		desired = &profile.DesiredState
	} else if len(roots) == 0 {
		return knowledgelifecycle.Observation{}, "", profile, fmt.Errorf("at least one --root is required for bootstrap observation")
	}

	bindingStore, err := knowledgebinding.NewStore(options.stateRoot)
	if err != nil {
		return knowledgelifecycle.Observation{}, "", profile, err
	}
	bindingSnapshot, err := bindingStore.Snapshot()
	if err != nil {
		return knowledgelifecycle.Observation{}, "", profile, err
	}
	selected := make(map[string]string)
	var bindingDigest *string
	var coordinatorIdentity *knowledgelifecycle.Identity
	if bindingSnapshot.Exists {
		bindingDigest = stringAddress(bindingSnapshot.Registry.RegistryDigest)
		for _, binding := range bindingSnapshot.Registry.Bindings {
			installed, inspectErr := knowledgeengine.InspectInstallation(binding.Role, binding.Prefix, binding.Version)
			if inspectErr != nil || installed.ModuleID != binding.ModuleID || installed.EngineID != binding.EngineID ||
				installed.ReceiptDigest != binding.ReceiptDigest || installed.ExecutableDigest != binding.ExecutableDigest {
				continue
			}
			selected[binding.Role] = binding.ReceiptDigest
			if binding.Role == "coordinator" {
				coordinatorIdentity = &knowledgelifecycle.Identity{
					ComponentID: "knowledge-session-coordinator", Version: binding.Version,
					ExecutableDigest: binding.ExecutableDigest,
				}
			}
		}
	}
	qxctlDigest, err := knowledgelifecycle.DigestCurrentExecutable()
	if err != nil {
		return knowledgelifecycle.Observation{}, "", profile, err
	}
	observation, err := knowledgelifecycle.Observe(knowledgelifecycle.ObservationInput{
		ProfileID: options.profileID, TOPSID: options.topsID, ConfiguredRoots: roots,
		DesiredState: desired, BindingRegistryDigest: bindingDigest, SelectedReceipts: selected,
		QxctlIdentity: knowledgelifecycle.Identity{
			ComponentID: "qxctl", Version: strings.ReplaceAll(qxversion.Version, " ", "-"),
			ExecutableDigest: qxctlDigest,
		},
		CoordinatorIdentity: coordinatorIdentity,
		ProviderAvailability: []knowledgelifecycle.ProviderAvailability{
			{ProviderID: "ssiag", Available: true},
			{ProviderID: "knowledge-session-coordinator", Available: coordinatorIdentity != nil},
			{ProviderID: "maestro", Available: false},
		},
		ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	return observation, profileDigest, profile, err
}

func exactBoundCoordinator(stateRoot string) (knowledgebinding.Binding, error) {
	store, err := knowledgebinding.NewStore(stateRoot)
	if err != nil {
		return knowledgebinding.Binding{}, err
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		return knowledgebinding.Binding{}, err
	}
	if !snapshot.Exists {
		return knowledgebinding.Binding{}, fmt.Errorf("knowledge engine binding registry is absent")
	}
	for _, binding := range snapshot.Registry.Bindings {
		if binding.Role != "coordinator" {
			continue
		}
		installed, err := knowledgeengine.InspectInstallation("coordinator", binding.Prefix, binding.Version)
		if err != nil {
			return knowledgebinding.Binding{}, fmt.Errorf("bound coordinator installation is unavailable: %w", err)
		}
		if installed.ModuleID != binding.ModuleID || installed.EngineID != binding.EngineID ||
			installed.ReceiptDigest != binding.ReceiptDigest || installed.ExecutableDigest != binding.ExecutableDigest {
			return knowledgebinding.Binding{}, fmt.Errorf("bound coordinator installation no longer matches its content-addressed identity")
		}
		return binding, nil
	}
	return knowledgebinding.Binding{}, fmt.Errorf("knowledge-session coordinator is not bound")
}

func authorizeKnowledgeLifecycle(options knowledgeLifecycleOptions, operation, resource string) error {
	if _, err := lifecycleStore(options); err != nil {
		return err
	}
	client, err := ssiagclient.NewForTOPS(options.scope, options.topsID, 4*time.Second)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, options.topsID, options.scope); err != nil {
		return err
	}
	requestID, err := randomUUID()
	if err != nil {
		return err
	}
	correlationID, err := randomUUID()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Truncate(time.Second)
	request := ssiagclient.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: requestID,
		CorrelationID: correlationID, Operation: "symphony.knowledge.lifecycle." + operation,
		Resource: resource, Audience: "qxctl", Scope: "tops:" + options.topsID,
		RequestedAt: now, RequestedExpiresAt: now.Add(options.ttl).UTC().Truncate(time.Second),
	}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return err
	}
	if err := validateSessionAuthorization(decision, request, options.topsID); err != nil {
		return fmt.Errorf("SSIAG lifecycle authorization rejected: %w", err)
	}
	return nil
}

func lifecycleResource(topsID, profileID, evidence string) string {
	digest := sha256.Sum256([]byte(topsID + "\n" + profileID + "\n" + evidence))
	return "symphony.knowledge.lifecycle:" + hex.EncodeToString(digest[:])
}

type validatedLifecyclePlan struct {
	Raw           any
	TransactionID string
	PlanDigest    string
	ActionCount   int
	ReadyCount    int
	DeferredCount int
	BlockedCount  int
	FatalCount    int
}

func validateLifecyclePlan(raw json.RawMessage, desiredDigest, observationDigest, priorDigest string) (validatedLifecyclePlan, error) {
	if err := knowledgeengine.ValidateJSONObject(raw, 4*1024*1024); err != nil {
		return validatedLifecyclePlan{}, fmt.Errorf("invalid lifecycle plan JSON: %w", err)
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return validatedLifecyclePlan{}, err
	}
	required := []string{
		"protocol", "format_version", "transaction_id", "revision", "previous_plan_digest",
		"desired_state_digest", "observation_digest", "observation_key", "prior_applied_state_digest",
		"compatibility", "scheduler", "actions", "ready_action_ids", "deferred_action_ids",
		"blocked_action_ids", "advisories", "fatal_blockers", "apply_authorized", "canonical", "plan_digest",
	}
	if len(object) != len(required) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan has an invalid root field set")
	}
	for _, field := range required {
		if _, present := object[field]; !present {
			return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan is missing %s", field)
		}
	}
	transaction, transactionOK := object["transaction_id"].(string)
	planDigest, planDigestOK := object["plan_digest"].(string)
	revision, revisionOK := object["revision"].(json.Number)
	revisionValue := int64(0)
	if revisionOK {
		revisionValue, _ = revision.Int64()
	}
	if object["protocol"] != "symphony.knowledge.lifecycle-plan.v1" || object["format_version"] != json.Number("1") ||
		!transactionOK || !validSessionToken(transaction) || !planDigestOK || !validTaggedDigest(planDigest) ||
		!revisionOK || revisionValue < 1 || revisionValue > 256 ||
		object["desired_state_digest"] != desiredDigest || object["observation_digest"] != observationDigest ||
		!digestOrNil(object["previous_plan_digest"]) || !digestString(object["observation_key"]) ||
		object["apply_authorized"] != false || object["canonical"] != false {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan identity or non-apply boundary is invalid")
	}
	if priorDigest == "" {
		if object["prior_applied_state_digest"] != nil {
			return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan prior applied-state digest mismatch")
		}
	} else if object["prior_applied_state_digest"] != priorDigest {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan prior applied-state digest mismatch")
	}
	compatibility, ok := object["compatibility"].(map[string]any)
	if !ok || len(compatibility) != 11 ||
		!oneOfInterface(compatibility["mode"], "full", "blocked") ||
		!digestlessVersion(compatibility["coordinator_version"]) ||
		compatibility["desired_state_version"] != json.Number("1") ||
		compatibility["observation_version"] != json.Number("1") ||
		compatibility["plan_version"] != json.Number("1") ||
		compatibility["applied_state_version"] != json.Number("1") {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan compatibility envelope is invalid")
	}
	if !validUniqueIntegerArray(compatibility["receipt_versions"], 0, 16, 1, 16) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan receipt compatibility is invalid")
	}
	if !validUniqueTokenArray(compatibility["required_capabilities"], 1, 128) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan capability compatibility is invalid")
	}
	if !validUniqueTokenArray(compatibility["missing_capabilities"], 0, 128) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan missing-capability evidence is invalid")
	}
	if _, ok := compatibility["two_way_procedural_compatibility"].(bool); !ok {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan two-way compatibility evidence is invalid")
	}
	if reason, ok := compatibility["reason"].(string); !ok || !validPlanText(reason) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan compatibility reason is invalid")
	}
	scheduler, ok := object["scheduler"].(map[string]any)
	if !ok || len(scheduler) != 9 || scheduler["algorithm"] != "dependency_ready_set_v1" || scheduler["dynamic_replanning"] != true ||
		scheduler["directionality"] != "forward_and_inverse" || scheduler["tie_break"] != "lexicographic_action_id" ||
		scheduler["cycle_policy"] != "block_cyclic_component_continue_unrelated" ||
		scheduler["max_actions"] != json.Number("4096") || scheduler["max_replans_per_transaction"] != json.Number("256") ||
		scheduler["max_attempts_per_action"] != json.Number("8") {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan scheduler contract is invalid")
	}
	expectedPhases := []string{"lock", "observe", "authorize", "compare_and_swap", "act", "verify", "audit"}
	phaseValues, ok := scheduler["safety_phase_order"].([]any)
	if !ok || len(phaseValues) != len(expectedPhases) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan safety phase order is invalid")
	}
	for index, phase := range expectedPhases {
		if phaseValues[index] != phase {
			return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan safety phase order is invalid")
		}
	}
	actions, ok := object["actions"].([]any)
	if !ok || len(actions) > 4096 {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan action collection is invalid")
	}
	seenActions := make(map[string]string, len(actions))
	actionObjects := make([]map[string]any, 0, len(actions))
	for _, rawAction := range actions {
		action, ok := rawAction.(map[string]any)
		if !ok || len(action) != 13 {
			return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan action field set is invalid")
		}
		for _, field := range []string{
			"action_id", "component_id", "kind", "direction", "prerequisite_action_ids",
			"inverse_action_id", "expected_before_digest", "target_state_digest", "target_receptor_id",
			"expected_artifact_digests", "expected_evidence", "disposition", "blockers",
		} {
			if _, present := action[field]; !present {
				return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan action is missing %s", field)
			}
		}
		id, idOK := action["action_id"].(string)
		disposition, dispositionOK := action["disposition"].(string)
		targetDigest, targetOK := action["target_state_digest"].(string)
		componentID, componentOK := action["component_id"].(string)
		if !idOK || !validSessionToken(id) || !componentOK || !validSessionToken(componentID) ||
			!oneOfInterface(action["kind"], "install", "uninstall", "select", "deselect", "activate", "deactivate", "dock", "undock", "verify", "preserve", "report") ||
			!oneOfInterface(action["direction"], "forward", "inverse", "neutral") || !dispositionOK ||
			!oneOfText(disposition, "ready", "waiting", "blocked", "completed", "skipped", "fatal") ||
			!targetOK || !validTaggedDigest(targetDigest) || !digestOrNil(action["expected_before_digest"]) ||
			!tokenOrNil(action["inverse_action_id"]) || !tokenOrNil(action["target_receptor_id"]) {
			return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan action identity or disposition is invalid")
		}
		if !validUniqueTokenArray(action["prerequisite_action_ids"], 0, 4096) ||
			!validUniqueDigestArray(action["expected_artifact_digests"], 4096) ||
			!validUniqueTokenArray(action["expected_evidence"], 0, 128) ||
			!validBlockerArray(action["blockers"], 64) {
			return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan action collection is invalid")
		}
		if action["kind"] == "dock" {
			if action["target_receptor_id"] == nil {
				return validatedLifecyclePlan{}, fmt.Errorf("lifecycle dock action lacks its exact receptor")
			}
		} else if action["target_receptor_id"] != nil {
			return validatedLifecyclePlan{}, fmt.Errorf("non-dock lifecycle action carries a receptor")
		}
		if _, duplicate := seenActions[id]; duplicate {
			return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan action identity is duplicated")
		}
		seenActions[id] = disposition
		actionObjects = append(actionObjects, action)
	}
	for _, action := range actionObjects {
		id := action["action_id"].(string)
		for _, value := range action["prerequisite_action_ids"].([]any) {
			prerequisite := value.(string)
			if prerequisite == id || seenActions[prerequisite] == "" {
				return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan prerequisite references an unknown or identical action")
			}
		}
	}
	ready, err := validatePlanIDSet(object["ready_action_ids"], seenActions, "ready")
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	deferred, err := validatePlanIDSet(object["deferred_action_ids"], seenActions, "waiting")
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	blocked, err := validatePlanIDSet(object["blocked_action_ids"], seenActions, "blocked", "fatal")
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	fatal, ok := object["fatal_blockers"].([]any)
	if !ok || !validBlockerArray(fatal, 4096) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan fatal blocker collection is invalid")
	}
	if !validAdvisoryArray(object["advisories"], 4096) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan advisory collection is invalid")
	}
	if ready+deferred+blocked != len(actions) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan action-ID sets do not partition report actions")
	}
	digestInput := make(map[string]any, len(object)-1)
	for key, value := range object {
		if key != "plan_digest" {
			digestInput[key] = value
		}
	}
	encoded, err := json.Marshal(digestInput)
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	digest := sha256.Sum256(encoded)
	if planDigest != "sha256:"+hex.EncodeToString(digest[:]) {
		return validatedLifecyclePlan{}, fmt.Errorf("lifecycle plan digest mismatch")
	}
	var display any
	if err := json.Unmarshal(raw, &display); err != nil {
		return validatedLifecyclePlan{}, err
	}
	return validatedLifecyclePlan{
		Raw: display, TransactionID: transaction, PlanDigest: planDigest, ActionCount: len(actions),
		ReadyCount: ready, DeferredCount: deferred, BlockedCount: blocked, FatalCount: len(fatal),
	}, nil
}

func validatePlanIDSet(value any, actions map[string]string, dispositions ...string) (int, error) {
	values, ok := value.([]any)
	if !ok || len(values) > 4096 {
		return 0, fmt.Errorf("lifecycle plan action-ID set is invalid")
	}
	allowed := make(map[string]struct{}, len(dispositions))
	for _, disposition := range dispositions {
		allowed[disposition] = struct{}{}
	}
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		id, ok := item.(string)
		if !ok {
			return 0, fmt.Errorf("lifecycle plan action-ID set contains a non-string")
		}
		disposition, present := actions[id]
		if !present {
			return 0, fmt.Errorf("lifecycle plan action-ID set references an unknown action")
		}
		if _, allowedDisposition := allowed[disposition]; !allowedDisposition {
			return 0, fmt.Errorf("lifecycle plan action-ID set contradicts its action disposition")
		}
		if _, duplicate := seen[id]; duplicate {
			return 0, fmt.Errorf("lifecycle plan action-ID set contains a duplicate")
		}
		seen[id] = struct{}{}
	}
	return len(values), nil
}

func oneOfText(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}

func oneOfInterface(value any, candidates ...string) bool {
	text, ok := value.(string)
	return ok && oneOfText(text, candidates...)
}

func digestString(value any) bool {
	text, ok := value.(string)
	return ok && validTaggedDigest(text)
}

func digestOrNil(value any) bool { return value == nil || digestString(value) }

func tokenOrNil(value any) bool {
	if value == nil {
		return true
	}
	text, ok := value.(string)
	return ok && validSessionToken(text)
}

func digestlessVersion(value any) bool {
	text, ok := value.(string)
	if !ok || text == "" || len(text) > 64 {
		return false
	}
	for _, character := range text {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune(".+-", character) {
			continue
		}
		return false
	}
	return true
}

func validUniqueIntegerArray(value any, minimumItems, maximumItems, minimumValue, maximumValue int64) bool {
	values, ok := value.([]any)
	if !ok || int64(len(values)) < minimumItems || int64(len(values)) > maximumItems {
		return false
	}
	seen := make(map[int64]struct{}, len(values))
	for _, item := range values {
		number, ok := item.(json.Number)
		if !ok {
			return false
		}
		integer, err := number.Int64()
		if err != nil || integer < minimumValue || integer > maximumValue {
			return false
		}
		if _, duplicate := seen[integer]; duplicate {
			return false
		}
		seen[integer] = struct{}{}
	}
	return true
}

func validUniqueTokenArray(value any, minimum, maximum int) bool {
	return validUniqueStringArray(value, minimum, maximum, validSessionToken)
}

func validUniqueDigestArray(value any, maximum int) bool {
	return validUniqueStringArray(value, 0, maximum, validTaggedDigest)
}

func validUniqueStringArray(value any, minimum, maximum int, valid func(string) bool) bool {
	values, ok := value.([]any)
	if !ok || len(values) < minimum || len(values) > maximum {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		text, ok := item.(string)
		if !ok || !valid(text) {
			return false
		}
		if _, duplicate := seen[text]; duplicate {
			return false
		}
		seen[text] = struct{}{}
	}
	return true
}

func validBlockerArray(value any, maximum int) bool {
	values, ok := value.([]any)
	if !ok || len(values) > maximum {
		return false
	}
	for _, item := range values {
		blocker, ok := item.(map[string]any)
		if !ok || len(blocker) != 5 ||
			!oneOfInterface(blocker["class"], "dependency_wait", "observation_retryable", "compatibility_blocked", "authorization_denied", "integrity_fatal", "critical_state_unknown", "cycle_detected") ||
			!tokenString(blocker["component_id"]) || !tokenOrNil(blocker["action_id"]) {
			return false
		}
		retryable, retryableOK := blocker["retryable"].(bool)
		detail, detailOK := blocker["detail"].(string)
		if !retryableOK || !detailOK || !validPlanText(detail) {
			return false
		}
		class := blocker["class"].(string)
		if (class == "dependency_wait" || class == "observation_retryable") != retryable &&
			class != "compatibility_blocked" {
			return false
		}
	}
	return true
}

func validAdvisoryArray(value any, maximum int) bool {
	values, ok := value.([]any)
	if !ok || len(values) > maximum {
		return false
	}
	for _, item := range values {
		advisory, ok := item.(map[string]any)
		if !ok || len(advisory) != 5 || advisory["class"] != "noncritical_dependency_unsatisfied" ||
			!tokenString(advisory["component_id"]) || !tokenString(advisory["target_component_id"]) ||
			!oneOfInterface(advisory["condition"], "present", "absent", "installed", "active", "inactive", "docked", "undocked", "compatible") {
			return false
		}
		detail, ok := advisory["detail"].(string)
		if !ok || !validPlanText(detail) {
			return false
		}
	}
	return true
}

func tokenString(value any) bool {
	text, ok := value.(string)
	return ok && validSessionToken(text)
}

func validPlanText(value string) bool { return value != "" && len(value) <= 4096 }

func stringAddress(value string) *string { return &value }

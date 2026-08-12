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
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/maestroclient"
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
		profile, changed, err := store.SetGuarded(input, options.expectedProfileDigest,
			func(current knowledgelifecycle.Profile, exists bool) error {
				if !exists {
					return nil
				}
				return requireRetainedProfileClaims(store.StateRoot(), options, current, input)
			})
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"changed": changed, "profile": profile})
		}
		fmt.Printf("Knowledge lifecycle profile: operation=set profile=%s changed=%t generation=%d mode=%s digest=%s canonical=false\n",
			profile.ProfileID, changed, profile.Generation, profile.BootMode, profile.ProfileDigest)
		if profile.BootMode == "apply-compatible" {
			fmt.Println("Knowledge lifecycle profile: apply-compatible requested; mutation requires a separate explicit lifecycle apply")
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
		changed, err := store.RemoveGuarded(options.profileID, options.expectedProfileDigest,
			func(current knowledgelifecycle.Profile) error {
				return requireReleasedProfileClaims(store.StateRoot(), options, current)
			})
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

func requireRetainedProfileClaims(
	stateRoot string,
	options knowledgeLifecycleOptions,
	current knowledgelifecycle.Profile,
	input knowledgelifecycle.ProfileInput,
) error {
	for _, root := range current.ConfiguredRoots {
		ownership, err := knowledgelifecycle.NewOwnershipStore(
			root, stateRoot, options.topsID, options.profileID)
		if err != nil {
			return err
		}
		claims, err := ownership.ProfileClaims()
		if err != nil {
			return fmt.Errorf("inspect shared-root ownership before profile update: %w", err)
		}
		for _, claim := range claims {
			if !profileInputRetainsClaim(input, root, claim.ComponentID) {
				return fmt.Errorf("profile still owns or retires component %s under %s; converge desired absence before removing or relocating it", claim.ComponentID, root)
			}
		}
	}
	return nil
}

func requireReleasedProfileClaims(
	stateRoot string,
	options knowledgeLifecycleOptions,
	current knowledgelifecycle.Profile,
) error {
	for _, root := range current.ConfiguredRoots {
		ownership, err := knowledgelifecycle.NewOwnershipStore(
			root, stateRoot, options.topsID, options.profileID)
		if err != nil {
			return err
		}
		claimed, err := ownership.HasProfileClaims()
		if err != nil {
			return fmt.Errorf("inspect shared-root ownership before profile removal: %w", err)
		}
		if claimed {
			return fmt.Errorf("profile still owns or retires packages under %s; converge desired absence before removal", root)
		}
	}
	return nil
}

func profileInputRetainsClaim(input knowledgelifecycle.ProfileInput, root, componentID string) bool {
	configured := false
	for _, candidate := range input.ConfiguredRoots {
		configured = configured || candidate == root
	}
	if !configured {
		return false
	}
	for _, component := range input.Components {
		if component.ComponentID == componentID && component.InstallRoot == root {
			return true
		}
	}
	return false
}

func runKnowledgeLifecycleOwnership(operation string, options knowledgeLifecycleOptions) error {
	if options.ownershipRoot == "" {
		return fmt.Errorf("--root is required")
	}
	store, err := lifecycleStore(options)
	if err != nil {
		return err
	}
	snapshot, err := store.Snapshot(options.profileID)
	if err != nil {
		return err
	}
	if !snapshot.Exists {
		return fmt.Errorf("lifecycle profile %q is absent", options.profileID)
	}
	configured := false
	for _, root := range snapshot.Profile.ConfiguredRoots {
		if root == options.ownershipRoot {
			configured = true
			break
		}
	}
	if !configured {
		return fmt.Errorf("ownership root is not an exact configured root of the selected profile")
	}
	if err := authorizeKnowledgeLifecycle(options, "ownership."+operation, lifecycleResource(
		options.topsID, options.profileID, options.ownershipRoot)); err != nil {
		return err
	}
	var observation knowledgelifecycle.Observation
	expectedProfileDigest := snapshot.Profile.ProfileDigest
	if operation == "reconcile" {
		var observedProfile knowledgelifecycle.Profile
		observation, expectedProfileDigest, observedProfile, err = buildLifecycleObservation(options, false)
		if err != nil {
			return err
		}
		if observedProfile.ProfileDigest != expectedProfileDigest {
			return fmt.Errorf("lifecycle observation returned mismatched profile evidence")
		}
	}
	if (operation == "adopt" || operation == "release") && !validTaggedDigest(options.expectedOwnershipDigest) {
		return fmt.Errorf("--expected-ownership-registry-digest is required and must be an exact tagged SHA-256 digest")
	}
	if operation == "release" && !validTaggedDigest(options.receiptDigest) {
		return fmt.Errorf("--receipt-digest is required and must be an exact tagged SHA-256 digest")
	}
	var result knowledgelifecycle.OwnershipResult
	err = store.WithProfileSnapshot(options.profileID, func(profile knowledgelifecycle.Profile) error {
		if profile.ProfileDigest != expectedProfileDigest {
			return fmt.Errorf("lifecycle profile changed before ownership %s; observe and retry", operation)
		}
		configured := false
		for _, root := range profile.ConfiguredRoots {
			configured = configured || root == options.ownershipRoot
		}
		if !configured {
			return fmt.Errorf("ownership root is no longer an exact configured root of the selected profile")
		}
		ownership, err := knowledgelifecycle.NewOwnershipStore(
			options.ownershipRoot, store.StateRoot(), options.topsID, options.profileID)
		if err != nil {
			return err
		}
		switch operation {
		case "status":
			state, snapshotErr := ownership.Snapshot()
			if snapshotErr != nil {
				return snapshotErr
			}
			result = knowledgelifecycle.OwnershipView("status", state)
			return nil
		case "reconcile":
			var reconcileErr error
			result, reconcileErr = ownership.Reconcile(profile.DesiredState, observation)
			return reconcileErr
		case "adopt":
			var adoptErr error
			result, adoptErr = ownership.Adopt(options.expectedOwnershipDigest)
			return adoptErr
		case "release":
			var releaseErr error
			result, releaseErr = ownership.ReleaseLegacy(options.receiptDigest, options.expectedOwnershipDigest)
			return releaseErr
		default:
			return fmt.Errorf("unsupported knowledge lifecycle ownership operation")
		}
	})
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(result)
	}
	state := "absent"
	generation := uint64(0)
	digest := "absent"
	claims := 0
	if result.Snapshot.Exists && result.Snapshot.Registry != nil {
		state = result.Snapshot.Registry.EnforcementState
		generation = result.Snapshot.Registry.Generation
		digest = result.Snapshot.Registry.OwnershipRegistryDigest
		claims = len(result.Snapshot.Registry.Claims)
	}
	fmt.Printf("Knowledge lifecycle ownership: operation=%s root=%s state=%s changed=%t generation=%d claims=%d digest=%s canonical=false\n",
		operation, options.ownershipRoot, state, result.Changed, generation, claims, digest)
	return nil
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
		"client":                     lifecyclePlannerClient(),
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
		fmt.Println("Knowledge lifecycle report: non-mutating report complete; explicit lifecycle apply remains a separate operation")
	}
	return nil
}

func lifecyclePlannerClient() map[string]any {
	return map[string]any{
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
	}
}

func lifecycleJournalClient() map[string]any {
	return map[string]any{
		"client_id": "qxctl", "client_version": strings.ReplaceAll(qxversion.Version, " ", "-"),
		"process_protocols":      []string{"symphony.knowledge.engine-process.v1"},
		"journal_read_versions":  []uint64{1},
		"journal_write_versions": []uint64{1},
		"capabilities": []string{
			"atomic-head-v1", "dual-slot-journal-v1", "dynamic-replanning-v1",
			"expected-state-cas-v1", "idempotent-operation-v1",
			"opaque-extension-preservation-v1", "recovery-forward-v1", "report-only-v1",
		},
	}
}

func runKnowledgeLifecycleBoot(options knowledgeLifecycleOptions) error {
	if options.operationID == "" || options.expectedJournalDigest == "" {
		return fmt.Errorf("--operation-id and --expected-journal-digest are required")
	}
	if options.expectedJournalDigest != "absent" && !validTaggedDigest(options.expectedJournalDigest) {
		return fmt.Errorf("--expected-journal-digest must be absent or an exact tagged SHA-256 digest")
	}
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
	decision, err := authorizeKnowledgeLifecycleDecision(options, "boot", lifecycleResource(
		options.topsID, options.profileID, profileDigest+"\n"+
			profile.DesiredState.DesiredStateDigest+"\n"+profile.BootMode+"\n"+stableDigest))
	if err != nil {
		return err
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
	var prior any
	if options.priorAppliedStateDigest != "" {
		prior = options.priorAppliedStateDigest
	}
	payload, err := json.Marshal(map[string]any{
		"protocol": "symphony.knowledge.lifecycle-boot-command.v1", "operation": "lifecycle_boot",
		"state_root": store.StateRoot(), "operation_id": options.operationID,
		"expected_journal_digest": options.expectedJournalDigest,
		"profile_id":              options.profileID, "tops_id": options.topsID,
		"profile_digest": profileDigest, "stable_inventory_digest": stableDigest,
		"mode": profile.BootMode, "desired_state": profile.DesiredState, "observation": observation,
		"prior_applied_state_digest": prior, "authorization_decision": decision,
		"planner_client": lifecyclePlannerClient(), "journal_client": lifecycleJournalClient(),
	})
	if err != nil {
		return fmt.Errorf("encode lifecycle boot request: %w", err)
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(), coordinator.Prefix, coordinator.Version, repositoryRoot, "lifecycle_boot", payload)
	if err != nil {
		return err
	}
	result, err := validateLifecycleBootResult(
		response.Result, "lifecycle_boot", options.profileID, options.topsID,
		profileDigest, profile.DesiredState.DesiredStateDigest, observation.ObservationDigest,
		options.priorAppliedStateDigest)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(result.Raw)
	}
	fmt.Printf("Knowledge lifecycle boot: profile=%s changed=%t state=%s generation=%d revision=%d journal_digest=%s apply_authorized=false canonical=false\n",
		options.profileID, result.Changed, result.State, result.Generation,
		result.PlanRevision, result.JournalDigest)
	if profile.BootMode == "apply-compatible" {
		fmt.Println("Knowledge lifecycle boot: durable planning is active; action execution requires a separate explicit lifecycle apply")
	}
	return nil
}

func runKnowledgeLifecycleBootState(operation string, options knowledgeLifecycleOptions) error {
	expected := options.expectedJournalDigest
	if operation == "lifecycle_boot_recover" {
		if options.operationID == "" {
			return fmt.Errorf("--operation-id is required")
		}
		if options.discover && expected != "" {
			return fmt.Errorf("--discover and --expected-journal-digest are mutually exclusive")
		}
		if options.discover {
			expected = "discover"
		}
		if expected == "" {
			return fmt.Errorf("--expected-journal-digest or --discover is required")
		}
		if expected != "discover" && !validTaggedDigest(expected) {
			return fmt.Errorf("--expected-journal-digest must be an exact tagged SHA-256 digest")
		}
	}
	evidence := "status"
	permission := "boot.status"
	if operation == "lifecycle_boot_recover" {
		evidence = expected
		permission = "boot.recover"
	}
	decision, err := authorizeKnowledgeLifecycleDecision(
		options, permission, lifecycleResource(options.topsID, options.profileID, evidence))
	if err != nil {
		return err
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
	var operationID any
	var expectedValue any
	if operation == "lifecycle_boot_recover" {
		operationID = options.operationID
		expectedValue = expected
	}
	payload, err := json.Marshal(map[string]any{
		"protocol": "symphony.knowledge.lifecycle-boot-command.v1", "operation": operation,
		"state_root": store.StateRoot(), "operation_id": operationID,
		"expected_journal_digest": expectedValue, "profile_id": options.profileID,
		"tops_id": options.topsID, "authorization_decision": decision,
		"journal_client": lifecycleJournalClient(),
	})
	if err != nil {
		return fmt.Errorf("encode lifecycle boot state request: %w", err)
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(), coordinator.Prefix, coordinator.Version, repositoryRoot, operation, payload)
	if err != nil {
		return err
	}
	result, err := validateLifecycleBootResult(
		response.Result, operation, options.profileID, options.topsID, "", "", "", "")
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(result.Raw)
	}
	fmt.Printf("Knowledge lifecycle %s: profile=%s present=%t changed=%t recovered=%t state=%s generation=%d revision=%d journal_digest=%s apply_authorized=false canonical=false\n",
		strings.TrimPrefix(operation, "lifecycle_boot_"), options.profileID, result.JournalPresent,
		result.Changed, result.Recovered, result.State, result.Generation,
		result.PlanRevision, result.JournalDigest)
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
	runtimeStore, err := knowledgelifecycle.NewRuntimeStore(options.stateRoot, options.topsID, options.profileID)
	if err != nil {
		return knowledgelifecycle.Observation{}, "", profile, err
	}
	runtimeSnapshot, err := runtimeStore.Snapshot()
	if err != nil {
		return knowledgelifecycle.Observation{}, "", profile, err
	}
	var runtimeState *knowledgelifecycle.RuntimeState
	if runtimeSnapshot.Exists {
		runtimeState = &runtimeSnapshot.State
	}
	qxctlDigest, err := knowledgelifecycle.DigestCurrentExecutable()
	if err != nil {
		return knowledgelifecycle.Observation{}, "", profile, err
	}
	if (options.maestroPrefix == "") != (len(options.maestroReceptorIDs) == 0) {
		return knowledgelifecycle.Observation{}, "", profile,
			fmt.Errorf("--maestro-prefix and --maestro-receptor-id must be supplied together")
	}
	receptors := append([]string(nil), options.maestroReceptorIDs...)
	sort.Strings(receptors)
	for index, receptorID := range receptors {
		if !validSessionToken(receptorID) {
			return knowledgelifecycle.Observation{}, "", profile,
				fmt.Errorf("--maestro-receptor-id has invalid syntax")
		}
		if index > 0 && receptorID == receptors[index-1] {
			return knowledgelifecycle.Observation{}, "", profile,
				fmt.Errorf("--maestro-receptor-id values must be unique")
		}
	}
	if desired != nil && len(receptors) != 0 {
		for _, component := range desired.Components {
			if component.Docking.Disposition == "docked" &&
				(component.Docking.ReceptorID == nil ||
					!containsLifecycleReceptor(receptors, *component.Docking.ReceptorID)) {
				return knowledgelifecycle.Observation{}, "", profile,
					fmt.Errorf("desired component %q targets a Maestro receptor outside the exhaustive configured set", component.ComponentID)
			}
		}
	}
	maestroAvailable := options.maestroPrefix != ""
	var maestroInstallation knowledgeengine.Installation
	if maestroAvailable {
		version := options.maestroVersion
		if version == "" {
			version = "0.1.0-dev"
		}
		maestroInstallation, err = knowledgeengine.InspectMaestroInstallation(options.maestroPrefix, version)
		if err != nil {
			return knowledgelifecycle.Observation{}, "", profile, fmt.Errorf("Maestro installation is unavailable: %w", err)
		}
	}
	observation, err := knowledgelifecycle.Observe(knowledgelifecycle.ObservationInput{
		ProfileID: options.profileID, TOPSID: options.topsID, ConfiguredRoots: roots,
		DesiredState: desired, BindingRegistryDigest: bindingDigest, SelectedReceipts: selected,
		RuntimeState: runtimeState,
		QxctlIdentity: knowledgelifecycle.Identity{
			ComponentID: "qxctl", Version: strings.ReplaceAll(qxversion.Version, " ", "-"),
			ExecutableDigest: qxctlDigest,
		},
		CoordinatorIdentity: coordinatorIdentity,
		ProviderAvailability: []knowledgelifecycle.ProviderAvailability{
			{ProviderID: "ssiag", Available: true},
			{ProviderID: "knowledge-session-coordinator", Available: coordinatorIdentity != nil},
			{ProviderID: "maestro", Available: maestroAvailable},
		},
		ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	if err != nil || !maestroAvailable {
		return observation, profileDigest, profile, err
	}
	repositoryRoot, err := resolveKnowledgeRepository(options.repository)
	if err != nil {
		return knowledgelifecycle.Observation{}, "", profile, err
	}
	effective := make(map[string]knowledgelifecycle.DockingPresence)
	for _, receptorID := range receptors {
		statusResource := maestroclient.Resource(
			options.topsID, receptorID, "status", "all", "none", "status")
		decision, authErr := authorizeMaestro(maestroOptions{
			topsID: options.topsID, receptorID: receptorID, scope: options.scope, ttl: options.ttl,
		}, "status", statusResource)
		if authErr != nil {
			return knowledgelifecycle.Observation{}, "", profile, authErr
		}
		status, statusErr := maestroclient.Status(context.Background(), maestroInstallation.Prefix,
			maestroInstallation.Version, repositoryRoot, store.StateRoot(), options.topsID,
			receptorID, "", decision)
		if statusErr != nil {
			return knowledgelifecycle.Observation{}, "", profile, statusErr
		}
		registry, decodeErr := status.DecodedRegistry()
		if decodeErr != nil {
			return knowledgelifecycle.Observation{}, "", profile, decodeErr
		}
		if registry == nil {
			continue
		}
		for _, item := range registry.Components {
			current, found := effective[item.ComponentID]
			if item.Disposition == "docked" && found && current.Disposition == "docked" {
				return knowledgelifecycle.Observation{}, "", profile,
					fmt.Errorf("component %q is docked at multiple configured receptors", item.ComponentID)
			}
			if !found || item.Disposition == "docked" {
				effective[item.ComponentID] = knowledgelifecycle.DockingPresence{
					ComponentID: item.ComponentID, Disposition: item.Disposition, ReceptorID: item.ReceptorID,
				}
			}
		}
	}
	presence := make([]knowledgelifecycle.DockingPresence, 0, len(effective))
	for _, item := range effective {
		presence = append(presence, item)
	}
	observation, err = knowledgelifecycle.OverlayDockingPresence(observation, presence)
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
	_, err := authorizeKnowledgeLifecycleDecision(options, operation, resource)
	return err
}

func authorizeKnowledgeLifecycleDecision(
	options knowledgeLifecycleOptions,
	operation, resource string,
) (ssiagclient.AuthorizationDecision, error) {
	if _, err := lifecycleStore(options); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	client, err := ssiagclient.NewForTOPS(options.scope, options.topsID, 4*time.Second)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, options.topsID, options.scope); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	requestID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	correlationID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
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
		return ssiagclient.AuthorizationDecision{}, err
	}
	if err := validateSessionAuthorization(decision, request, options.topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, fmt.Errorf("SSIAG lifecycle authorization rejected: %w", err)
	}
	return decision, nil
}

func lifecycleResource(topsID, profileID, _ string) string {
	// SSIAG grants bind the stable TOPS/profile administration boundary. Exact
	// evidence remains independently bound by operation-specific schemas,
	// coordinator validation, receipt digests, and compare-and-swap state. This
	// avoids requiring a new policy grant for every content digest while never
	// widening an operation grant or bypassing evidence validation.
	digest := sha256.Sum256([]byte("profile\n" + topsID + "\n" + profileID))
	return "symphony.knowledge.lifecycle:" + hex.EncodeToString(digest[:])
}

func lifecycleCatalogResource(topsID string) string {
	// Catalog enumeration is a distinct read boundary, not a synthetic profile.
	// Domain separation prevents any valid profile ID from colliding with it.
	digest := sha256.Sum256([]byte("profile-catalog\n" + topsID))
	return "symphony.knowledge.lifecycle-catalog:" + hex.EncodeToString(digest[:])
}

type validatedLifecycleBootResult struct {
	Raw            any
	JournalPresent bool
	Changed        bool
	Recovered      bool
	State          string
	Generation     uint64
	PlanRevision   uint64
	JournalDigest  string
}

func validateLifecycleBootResult(
	raw json.RawMessage,
	operation, profileID, topsID, profileDigest, desiredDigest, observationDigest, priorDigest string,
) (validatedLifecycleBootResult, error) {
	if err := knowledgeengine.ValidateJSONObject(raw, 4*1024*1024); err != nil {
		return validatedLifecycleBootResult{}, fmt.Errorf("invalid lifecycle boot result JSON: %w", err)
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return validatedLifecycleBootResult{}, err
	}
	if !exactLifecycleFields(object, []string{
		"protocol", "operation", "compatibility", "journal_present", "journal", "journal_digest",
		"plan", "changed", "recovered", "repair_actions", "read_only", "apply_authorized", "canonical",
	}) || object["protocol"] != "symphony.knowledge.lifecycle-boot-result.v1" ||
		object["operation"] != operation || object["apply_authorized"] != false || object["canonical"] != false {
		return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot result identity or non-apply boundary is invalid")
	}
	compatibility, ok := object["compatibility"].(map[string]any)
	if !ok || !exactLifecycleFields(compatibility, []string{
		"mode", "process_protocol", "journal_read_version", "journal_write_version",
		"missing_capabilities", "two_way_procedural_compatibility", "reason",
	}) || !oneOfInterface(compatibility["mode"], "full", "read_only", "blocked") ||
		compatibility["two_way_procedural_compatibility"] != true ||
		!validUniqueTokenArray(compatibility["missing_capabilities"], 0, 64) {
		return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot compatibility envelope is invalid")
	}
	if reason, ok := compatibility["reason"].(string); !ok || !validPlanText(reason) {
		return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot compatibility reason is invalid")
	}
	for _, field := range []string{"journal_read_version", "journal_write_version"} {
		if compatibility[field] != nil && compatibility[field] != json.Number("1") {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot compatibility version is unsupported")
		}
	}
	if compatibility["process_protocol"] != nil &&
		compatibility["process_protocol"] != "symphony.knowledge.engine-process.v1" {
		return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot process protocol is unsupported")
	}
	present, presentOK := object["journal_present"].(bool)
	changed, changedOK := object["changed"].(bool)
	recovered, recoveredOK := object["recovered"].(bool)
	readOnly, readOnlyOK := object["read_only"].(bool)
	if !presentOK || !changedOK || !recoveredOK || !readOnlyOK ||
		(operation == "lifecycle_boot_status" && (!readOnly || changed || recovered)) ||
		(operation != "lifecycle_boot_status" && readOnly) ||
		(operation != "lifecycle_boot_recover" && recovered) {
		return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot result mutation boundary is invalid")
	}
	actions, ok := object["repair_actions"].([]any)
	if !ok || len(actions) > 64 {
		return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot repair evidence is invalid")
	}
	for _, item := range actions {
		value, ok := item.(string)
		if !ok || !validPlanText(value) {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot repair evidence is invalid")
		}
	}
	result := validatedLifecycleBootResult{Changed: changed, Recovered: recovered, JournalPresent: present}
	if !present {
		if object["journal"] != nil || object["journal_digest"] != nil || object["plan"] != nil || changed || recovered {
			return validatedLifecycleBootResult{}, fmt.Errorf("absent lifecycle boot result carries state")
		}
	} else {
		journal, ok := object["journal"].(map[string]any)
		journalDigest, digestOK := object["journal_digest"].(string)
		if !ok || !digestOK || !validTaggedDigest(journalDigest) || journal["journal_digest"] != journalDigest {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot journal identity is invalid")
		}
		if !exactLifecycleFields(journal, []string{
			"protocol", "format_version", "journal_id", "transaction_id", "operation_id", "generation",
			"previous_journal_digest", "profile_id", "profile_digest", "tops_id", "mode", "state", "desired_state_digest",
			"observation_key", "current_observation_digest", "current_stable_inventory_digest",
			"prior_applied_state_digest", "current_plan_digest", "current_plan_revision", "replan_count",
			"action_attempts", "blockers", "checkpoints", "compatibility", "extensions", "recovery",
			"started_at", "updated_at", "closed_at", "canonical", "apply_authorized", "journal_digest",
		}) || journal["protocol"] != "symphony.knowledge.lifecycle-boot-journal.v1" ||
			journal["format_version"] != json.Number("1") || journal["profile_id"] != profileID ||
			journal["tops_id"] != topsID || journal["canonical"] != false || journal["apply_authorized"] != false ||
			!tokenString(journal["journal_id"]) || !tokenString(journal["transaction_id"]) ||
			!tokenString(journal["operation_id"]) || !oneOfInterface(journal["mode"], "report", "apply-compatible") ||
			!oneOfInterface(journal["state"], "open", "blocked", "verified", "closed") ||
			!digestString(journal["profile_digest"]) || !digestString(journal["desired_state_digest"]) ||
			!digestString(journal["observation_key"]) ||
			!digestString(journal["current_observation_digest"]) ||
			!digestString(journal["current_stable_inventory_digest"]) ||
			!digestString(journal["current_plan_digest"]) || !digestOrNil(journal["previous_journal_digest"]) ||
			!digestOrNil(journal["prior_applied_state_digest"]) {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot journal contract is invalid")
		}
		if profileDigest != "" && journal["profile_digest"] != profileDigest {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot journal profile digest mismatch")
		}
		generation, generationOK := lifecycleUint(journal["generation"], 1, 9007199254740991)
		revision, revisionOK := lifecycleUint(journal["current_plan_revision"], 1, 256)
		if !generationOK || !revisionOK {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot journal generation is invalid")
		}
		attempts, attemptsOK := journal["action_attempts"].([]any)
		if !attemptsOK || len(attempts) != 0 {
			return validatedLifecycleBootResult{}, fmt.Errorf("report-only lifecycle journal contains action attempts")
		}
		extensions, extensionsOK := journal["extensions"].([]any)
		if !extensionsOK || len(extensions) > 64 {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle journal extensions are invalid")
		}
		for _, item := range extensions {
			extension, ok := item.(map[string]any)
			if !ok || !exactLifecycleFields(extension, []string{"extension_id", "extension_version", "critical", "payload", "payload_digest"}) ||
				!tokenString(extension["extension_id"]) || extension["critical"] != false || !digestString(extension["payload_digest"]) {
				return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle journal contains an unsupported extension")
			}
			payload, err := json.Marshal(extension["payload"])
			if err != nil || extension["payload_digest"] != taggedLifecycleDigest(payload) {
				return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle journal extension digest mismatch")
			}
		}
		digestInput := make(map[string]any, len(journal)-1)
		for key, value := range journal {
			if key != "journal_digest" {
				digestInput[key] = value
			}
		}
		encoded, err := json.Marshal(digestInput)
		if err != nil || journalDigest != taggedLifecycleDigest(encoded) {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot journal digest mismatch")
		}
		result.State = journal["state"].(string)
		result.Generation = generation
		result.PlanRevision = revision
		result.JournalDigest = journalDigest
	}
	if object["plan"] != nil {
		if operation != "lifecycle_boot" || !present || desiredDigest == "" || observationDigest == "" {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot result carries an unexpected plan")
		}
		planRaw, err := json.Marshal(object["plan"])
		if err != nil {
			return validatedLifecycleBootResult{}, err
		}
		plan, err := validateLifecyclePlan(planRaw, desiredDigest, observationDigest, priorDigest)
		if err != nil {
			return validatedLifecycleBootResult{}, err
		}
		journal := object["journal"].(map[string]any)
		if journal["current_plan_digest"] != plan.PlanDigest {
			return validatedLifecycleBootResult{}, fmt.Errorf("lifecycle boot plan and journal digests differ")
		}
	}
	if err := json.Unmarshal(raw, &result.Raw); err != nil {
		return validatedLifecycleBootResult{}, err
	}
	return result, nil
}

func exactLifecycleFields(object map[string]any, fields []string) bool {
	if len(object) != len(fields) {
		return false
	}
	for _, field := range fields {
		if _, present := object[field]; !present {
			return false
		}
	}
	return true
}

func lifecycleUint(value any, minimum, maximum uint64) (uint64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	parsed, err := number.Int64()
	if err != nil || parsed < 0 || uint64(parsed) < minimum || uint64(parsed) > maximum {
		return 0, false
	}
	return uint64(parsed), true
}

func taggedLifecycleDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

type validatedLifecyclePlan struct {
	Raw           any
	Actions       []knowledgelifecycle.PlannedAction
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
	decodedActions, err := decodedLifecycleActions(actionObjects)
	if err != nil {
		return validatedLifecyclePlan{}, err
	}
	return validatedLifecyclePlan{
		Raw: display, Actions: decodedActions, TransactionID: transaction,
		PlanDigest: planDigest, ActionCount: len(actions),
		ReadyCount: ready, DeferredCount: deferred, BlockedCount: blocked, FatalCount: len(fatal),
	}, nil
}

func decodedLifecycleActions(values []map[string]any) ([]knowledgelifecycle.PlannedAction, error) {
	actions := make([]knowledgelifecycle.PlannedAction, 0, len(values))
	for _, value := range values {
		action, err := knowledgelifecycle.DecodePlannedAction(value)
		if err != nil {
			return nil, fmt.Errorf("decode validated lifecycle action: %w", err)
		}
		actions = append(actions, action)
	}
	return actions, nil
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

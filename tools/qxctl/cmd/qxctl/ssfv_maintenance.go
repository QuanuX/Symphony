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
	"github.com/QuanuX/Symphony/tools/qxctl/internal/maestroclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	qxversion "github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

const ssfvMaintenanceResultProtocol = "symphony.knowledge.ssfv-maintenance-result.v1"

var ssfvMaintenanceCapabilities = []string{
	"atomic-head-v1", "content-addressed-ssfv-baseline-v1", "dual-slot-journal-v1",
	"expected-state-cas-v1", "idempotent-operation-v1", "maestro-inventory-lineage-v1",
	"opaque-extension-preservation-v1", "recovery-forward-v1", "ssiag-capability-binding-v1",
}

type ssfvMaintenanceResult struct {
	Protocol              string          `json:"protocol"`
	FormatVersion         uint64          `json:"format_version"`
	Operation             string          `json:"operation"`
	Compatibility         json.RawMessage `json:"compatibility"`
	JournalPresent        *bool           `json:"journal_present"`
	Journal               json.RawMessage `json:"journal"`
	JournalDigest         *string         `json:"journal_digest"`
	EffectiveState        string          `json:"effective_state"`
	ReviewState           string          `json:"review_state"`
	Changed               *bool           `json:"changed"`
	Recovered             *bool           `json:"recovered"`
	RepairActions         []string        `json:"repair_actions"`
	ReadOnly              *bool           `json:"read_only"`
	CanonicalApplyEnabled *bool           `json:"canonical_apply_enabled"`
	Canonical             *bool           `json:"canonical"`
}

type ssfvSemanticSnapshot struct {
	Protocol       string `json:"protocol"`
	ModuleID       string `json:"module_id"`
	EngineID       string `json:"engine_id"`
	EngineVersion  string `json:"engine_version"`
	VectorID       string `json:"vector_id"`
	SnapshotDigest string `json:"snapshot_digest"`
}

func runKnowledgeSSFVMaintenance(operation string, options knowledgeSSFVMaintenanceOptions) error {
	invocation, err := executeKnowledgeSSFVMaintenance(operation, options)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var display any
		if err := json.Unmarshal(invocation.Raw, &display); err != nil {
			return fmt.Errorf("decode SSFV maintenance result: %w", err)
		}
		return printIndentedJSON(display)
	}
	digest := "absent"
	if invocation.Result.JournalDigest != nil {
		digest = *invocation.Result.JournalDigest
	}
	fmt.Printf("SSFV session maintenance: operation=%s state=%s review=%s present=%t changed=%t recovered=%t digest=%s canonical=false\n",
		operation, invocation.Result.EffectiveState, invocation.Result.ReviewState,
		*invocation.Result.JournalPresent, *invocation.Result.Changed,
		*invocation.Result.Recovered, digest)
	return nil
}

type ssfvMaintenanceInvocation struct {
	Result ssfvMaintenanceResult
	Raw    json.RawMessage
}

func executeKnowledgeSSFVMaintenance(operation string, options knowledgeSSFVMaintenanceOptions) (ssfvMaintenanceInvocation, error) {
	if options.topsID == "" {
		return ssfvMaintenanceInvocation{}, fmt.Errorf("--tops-id is required")
	}
	if options.scope != "user" && options.scope != "system" {
		return ssfvMaintenanceInvocation{}, fmt.Errorf("--scope must be user or system")
	}
	if options.ttl <= 0 || options.ttl > 24*time.Hour {
		return ssfvMaintenanceInvocation{}, fmt.Errorf("--ttl must be greater than zero and no more than 24h")
	}
	if operation != "status" && !validSessionToken(options.operationID) {
		return ssfvMaintenanceInvocation{}, fmt.Errorf("--operation-id is required and must be a stable token")
	}
	expected := options.expectedJournalDigest
	switch operation {
	case "status":
		if options.operationID != "" || expected != "" || options.discover {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("status does not accept mutation state")
		}
	case "begin":
		if expected != "absent" && !validTaggedDigest(expected) {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("--expected-journal-digest must be absent or an exact tagged SHA-256 digest")
		}
	case "checkpoint", "close":
		if !validTaggedDigest(expected) {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("--expected-journal-digest must be an exact tagged SHA-256 digest")
		}
	case "recover":
		if options.discover == (expected != "") {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("exactly one of --expected-journal-digest or --discover is required")
		}
		if options.discover {
			expected = "discover"
		} else if !validTaggedDigest(expected) {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("--expected-journal-digest must be an exact tagged SHA-256 digest")
		}
	default:
		return ssfvMaintenanceInvocation{}, fmt.Errorf("unsupported SSFV maintenance operation")
	}

	repositoryRoot, err := resolveKnowledgeRepository(options.repository)
	if err != nil {
		return ssfvMaintenanceInvocation{}, err
	}
	store, err := knowledgebinding.NewStore(options.stateRoot)
	if err != nil {
		return ssfvMaintenanceInvocation{}, err
	}
	bindings, err := store.Snapshot()
	if err != nil {
		return ssfvMaintenanceInvocation{}, err
	}
	if !bindings.Exists || !validTaggedDigest(bindings.Registry.RegistryDigest) {
		return ssfvMaintenanceInvocation{}, fmt.Errorf("knowledge engine binding registry is absent or invalid")
	}
	coordinator, err := exactKnowledgeBinding(bindings, "coordinator")
	if err != nil {
		return ssfvMaintenanceInvocation{}, err
	}
	if _, err := verifyKnowledgeBinding(*coordinator); err != nil {
		return ssfvMaintenanceInvocation{}, err
	}

	engineOperation := "ssfv_maintenance_" + operation
	var operationID any
	var expectedValue any
	var sessionDigest any
	var bindingDigest any
	var engineEvidence any
	var snapshotEvidence any
	var diffEvidence any
	var inventoryEvidence any
	authoritySessionDigest := "none"
	if operation != "status" {
		operationID = options.operationID
		expectedValue = expected
		session, sessionErr := executeKnowledgeSession("status", knowledgeSessionOptions{
			topsID: options.topsID, scope: options.scope, stateRoot: options.stateRoot,
			repository: repositoryRoot, ttl: options.ttl,
		})
		if sessionErr != nil {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("read authenticated knowledge session: %w", sessionErr)
		}
		if session.Result.EffectiveState != "open" || session.Result.JournalDigest == nil {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("SSFV maintenance mutation requires an open authenticated knowledge session")
		}
		authoritySessionDigest = *session.Result.JournalDigest
		sessionDigest = authoritySessionDigest
	}

	if operation == "begin" || operation == "checkpoint" || operation == "close" {
		ssfv, bindingErr := exactKnowledgeBinding(bindings, "ssfv")
		if bindingErr != nil {
			return ssfvMaintenanceInvocation{}, bindingErr
		}
		installation, inspectErr := verifyKnowledgeBinding(*ssfv)
		if inspectErr != nil {
			return ssfvMaintenanceInvocation{}, inspectErr
		}
		engineMap := map[string]any{
			"module_id": installation.ModuleID, "engine_id": installation.EngineID,
			"vector_id": "ssfv", "version": installation.Version,
			"receipt_digest":    installation.ReceiptDigest,
			"executable_digest": installation.ExecutableDigest,
		}
		engineMap["evidence_digest"], err = maintenanceObjectDigest(engineMap, "evidence_digest")
		if err != nil {
			return ssfvMaintenanceInvocation{}, err
		}
		engineEvidence = engineMap

		checkPayload, marshalErr := json.Marshal(map[string]any{
			"expected_namespace_digest": nil, "expected_registry_digest": nil,
			"freshness": "disabled", "baseline": nil,
		})
		if marshalErr != nil {
			return ssfvMaintenanceInvocation{}, marshalErr
		}
		check, invokeErr := knowledgeengine.InvokeSSFV(context.Background(), installation.Prefix,
			installation.Version, repositoryRoot, "check", checkPayload)
		if invokeErr != nil {
			return ssfvMaintenanceInvocation{}, invokeErr
		}
		valid, validateErr := validateSSFVResult("check", check.Result)
		if validateErr != nil || !valid {
			if validateErr != nil {
				return ssfvMaintenanceInvocation{}, validateErr
			}
			return ssfvMaintenanceInvocation{}, fmt.Errorf("SSFV structural check did not produce a valid semantic snapshot")
		}
		var checkResult struct {
			SemanticSnapshot json.RawMessage `json:"semantic_snapshot"`
		}
		if err := json.Unmarshal(check.Result, &checkResult); err != nil || len(checkResult.SemanticSnapshot) == 0 {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("SSFV check result lacks semantic snapshot evidence")
		}
		var snapshot ssfvSemanticSnapshot
		if err := json.Unmarshal(checkResult.SemanticSnapshot, &snapshot); err != nil ||
			snapshot.Protocol != "symphony.ssfv.semantic-snapshot.v1" || snapshot.ModuleID != "ssfv-engine" ||
			snapshot.EngineID != "symphony-ssfv" || snapshot.VectorID != "ssfv" ||
			snapshot.EngineVersion != installation.Version || !validTaggedDigest(snapshot.SnapshotDigest) {
			return ssfvMaintenanceInvocation{}, fmt.Errorf("SSFV semantic snapshot violates its identity contract")
		}
		if err := validateSSFVEmbeddedDigest(checkResult.SemanticSnapshot, "snapshot_digest", snapshot.SnapshotDigest); err != nil {
			return ssfvMaintenanceInvocation{}, err
		}
		snapshotEvidence = json.RawMessage(checkResult.SemanticSnapshot)
		bindingDigest = bindings.Registry.RegistryDigest

		inventoryMap, inventoryErr := collectMaintenanceMaestroEvidence(options, store.StateRoot(), repositoryRoot)
		if inventoryErr != nil {
			return ssfvMaintenanceInvocation{}, inventoryErr
		}
		inventoryEvidence = inventoryMap
		if operation != "begin" {
			status, statusErr := invokeSSFVMaintenanceCoordinator(
				*coordinator, repositoryRoot, store.StateRoot(), options.topsID,
				"ssfv_maintenance_status", nil, nil, nil, nil, nil, nil, nil, nil,
				options.scope, options.ttl)
			if statusErr != nil {
				return ssfvMaintenanceInvocation{}, fmt.Errorf("read SSFV maintenance baseline: %w", statusErr)
			}
			if !*status.Result.JournalPresent || status.Result.EffectiveState != "open" {
				return ssfvMaintenanceInvocation{}, fmt.Errorf("checkpoint or close requires an open SSFV maintenance journal")
			}
			var journal struct {
				BaselineSnapshot json.RawMessage `json:"baseline_snapshot"`
			}
			if err := json.Unmarshal(status.Result.Journal, &journal); err != nil || len(journal.BaselineSnapshot) == 0 {
				return ssfvMaintenanceInvocation{}, fmt.Errorf("SSFV maintenance journal lacks a readable baseline")
			}
			diffPayload, encodeErr := json.Marshal(map[string]any{
				"baseline":                         json.RawMessage(journal.BaselineSnapshot),
				"expected_current_snapshot_digest": snapshot.SnapshotDigest,
				"scope_feature_ids":                []string{}, "include_semantic_candidates": true,
			})
			if encodeErr != nil {
				return ssfvMaintenanceInvocation{}, encodeErr
			}
			diff, invokeErr := knowledgeengine.InvokeSSFV(context.Background(), installation.Prefix,
				installation.Version, repositoryRoot, "diff", diffPayload)
			if invokeErr != nil {
				return ssfvMaintenanceInvocation{}, invokeErr
			}
			if valid, validateErr := validateSSFVResult("diff", diff.Result); validateErr != nil || !valid {
				if validateErr != nil {
					return ssfvMaintenanceInvocation{}, validateErr
				}
				return ssfvMaintenanceInvocation{}, fmt.Errorf("SSFV diff evidence is invalid")
			}
			diffEvidence = json.RawMessage(diff.Result)
		}
	}

	return invokeSSFVMaintenanceCoordinator(
		*coordinator, repositoryRoot, store.StateRoot(), options.topsID, engineOperation,
		operationID, expectedValue, sessionDigest, bindingDigest, engineEvidence,
		snapshotEvidence, diffEvidence, inventoryEvidence, options.scope, options.ttl)
}

func exactKnowledgeBinding(snapshot knowledgebinding.Snapshot, role string) (*knowledgebinding.Binding, error) {
	for index := range snapshot.Registry.Bindings {
		if snapshot.Registry.Bindings[index].Role == role {
			return &snapshot.Registry.Bindings[index], nil
		}
	}
	return nil, fmt.Errorf("%s engine is not bound", role)
}

func verifyKnowledgeBinding(binding knowledgebinding.Binding) (knowledgeengine.Installation, error) {
	installed, err := knowledgeengine.InspectInstallation(binding.Role, binding.Prefix, binding.Version)
	if err != nil {
		return knowledgeengine.Installation{}, fmt.Errorf("bound %s installation is unavailable: %w", binding.Role, err)
	}
	if installed.ModuleID != binding.ModuleID || installed.EngineID != binding.EngineID ||
		installed.ReceiptDigest != binding.ReceiptDigest || installed.ExecutableDigest != binding.ExecutableDigest {
		return knowledgeengine.Installation{}, fmt.Errorf("bound %s installation no longer matches its content-addressed identity", binding.Role)
	}
	return installed, nil
}

func collectMaintenanceMaestroEvidence(options knowledgeSSFVMaintenanceOptions, stateRoot, repositoryRoot string) (map[string]any, error) {
	var evidence map[string]any
	if options.maestroPrefix == "" {
		evidence = map[string]any{
			"availability": "not_configured",
			"reason":       "Maestro inventory observation was not configured for this maintenance checkpoint",
			"observation":  nil,
		}
	} else {
		installation, err := knowledgeengine.InspectMaestroInstallation(options.maestroPrefix, options.maestroVersion)
		if err != nil {
			return nil, err
		}
		decision, err := authorizeMaestroInventory(maestroOptions{
			prefix: installation.Prefix, version: installation.Version, repository: repositoryRoot,
			stateRoot: stateRoot, topsID: options.topsID, scope: options.scope, ttl: options.ttl,
		})
		if err != nil {
			return nil, err
		}
		inventory, err := maestroclient.Inventory(context.Background(), installation.Prefix,
			installation.Version, repositoryRoot, stateRoot, options.topsID, decision)
		if err != nil {
			return nil, err
		}
		evidence = map[string]any{
			"availability": "observed", "reason": "derived from the exact authenticated Maestro installation",
			"observation": inventory,
		}
	}
	digest, err := maintenanceObjectDigest(evidence, "evidence_digest")
	if err != nil {
		return nil, err
	}
	evidence["evidence_digest"] = digest
	return evidence, nil
}

func invokeSSFVMaintenanceCoordinator(
	coordinator knowledgebinding.Binding,
	repositoryRoot, stateRoot, topsID, operation string,
	operationID, expected, sessionDigest, bindingDigest, engineEvidence,
	snapshotEvidence, diffEvidence, inventoryEvidence any,
	scope string, ttl time.Duration,
) (ssfvMaintenanceInvocation, error) {
	snapshotDigest := "none"
	if snapshotEvidence != nil {
		var snapshot ssfvSemanticSnapshot
		if err := json.Unmarshal(snapshotEvidence.(json.RawMessage), &snapshot); err != nil {
			return ssfvMaintenanceInvocation{}, err
		}
		snapshotDigest = snapshot.SnapshotDigest
	}
	inventoryDigest := "none"
	if inventoryEvidence != nil {
		inventoryDigest = inventoryEvidence.(map[string]any)["evidence_digest"].(string)
	}
	expectedResource := "status"
	sessionResource := "none"
	if expected != nil {
		expectedResource = expected.(string)
	}
	if sessionDigest != nil {
		sessionResource = sessionDigest.(string)
	}
	resource := ssfvMaintenanceResource(topsID, repositoryRoot, operation,
		expectedResource, sessionResource, snapshotDigest, inventoryDigest)
	decision, err := authorizeSSFVMaintenance(topsID, scope, ttl, operation, resource)
	if err != nil {
		return ssfvMaintenanceInvocation{}, err
	}
	payload, err := json.Marshal(map[string]any{
		"protocol": "symphony.knowledge.ssfv-maintenance-command.v1", "operation": operation,
		"state_root": stateRoot, "tops_id": topsID, "operation_id": operationID,
		"expected_journal_digest": expected, "repository_root": repositoryRoot,
		"session_journal_digest": sessionDigest, "binding_registry_digest": bindingDigest,
		"engine": engineEvidence, "semantic_snapshot": snapshotEvidence, "diff_result": diffEvidence,
		"maestro_inventory": inventoryEvidence, "authorization_decision": decision,
		"client": map[string]any{
			"client_id": "qxctl", "client_version": strings.ReplaceAll(qxversion.Version, " ", "-"),
			"process_protocols":     []string{"symphony.knowledge.engine-process.v1"},
			"journal_read_versions": []uint64{1}, "journal_write_versions": []uint64{1},
			"capabilities": ssfvMaintenanceCapabilities,
		},
	})
	if err != nil {
		return ssfvMaintenanceInvocation{}, fmt.Errorf("encode SSFV maintenance command: %w", err)
	}
	response, err := knowledgeengine.InvokeCoordinator(context.Background(), coordinator.Prefix,
		coordinator.Version, repositoryRoot, operation, payload)
	if err != nil {
		return ssfvMaintenanceInvocation{}, err
	}
	result, err := validateSSFVMaintenanceResult(response.Result, operation)
	if err != nil {
		return ssfvMaintenanceInvocation{}, err
	}
	return ssfvMaintenanceInvocation{Result: result, Raw: append(json.RawMessage(nil), response.Result...)}, nil
}

func authorizeSSFVMaintenance(topsID, scope string, ttl time.Duration, operation, resource string) (ssiagclient.AuthorizationDecision, error) {
	client, err := ssiagclient.NewForTOPS(scope, topsID, 4*time.Second)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, topsID, scope); err != nil {
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
		CorrelationID: correlationID, Operation: "symphony.knowledge." + operation,
		Resource: resource, Audience: "qxctl", Scope: "tops:" + topsID,
		RequestedAt: now, RequestedExpiresAt: now.Add(ttl).UTC().Truncate(time.Second),
	}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	if err := validateSessionAuthorization(decision, request, topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, fmt.Errorf("SSIAG SSFV maintenance authorization rejected: %w", err)
	}
	return decision, nil
}

func ssfvMaintenanceResource(topsID, repositoryRoot, operation, expected, sessionDigest, snapshotDigest, inventoryDigest string) string {
	joined := strings.Join([]string{topsID, repositoryRoot, operation, expected, sessionDigest, snapshotDigest, inventoryDigest}, "\n")
	digest := sha256.Sum256([]byte(joined))
	return "symphony.knowledge.ssfv-maintenance:" + hex.EncodeToString(digest[:])
}

func maintenanceObjectDigest(value any, digestField string) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var canonical any
	if err := decoder.Decode(&canonical); err != nil {
		return "", err
	}
	object, ok := canonical.(map[string]any)
	if !ok {
		return "", fmt.Errorf("maintenance digest input must be an object")
	}
	delete(object, digestField)
	encoded, err = marshalSSFVCanonical(object)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validateSSFVMaintenanceResult(raw json.RawMessage, operation string) (ssfvMaintenanceResult, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return ssfvMaintenanceResult{}, fmt.Errorf("decode SSFV maintenance result: %w", err)
	}
	required := []string{"protocol", "format_version", "operation", "compatibility", "journal_present", "journal",
		"journal_digest", "effective_state", "review_state", "changed", "recovered", "repair_actions",
		"read_only", "canonical_apply_enabled", "canonical"}
	if len(fields) != len(required) {
		return ssfvMaintenanceResult{}, fmt.Errorf("SSFV maintenance result has an invalid field set")
	}
	for _, field := range required {
		if _, ok := fields[field]; !ok {
			return ssfvMaintenanceResult{}, fmt.Errorf("SSFV maintenance result is incomplete")
		}
	}
	var result ssfvMaintenanceResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return ssfvMaintenanceResult{}, err
	}
	states := map[string]bool{"absent": true, "open": true, "closed": true}
	reviews := map[string]bool{"not_evaluated": true, "current": true, "review_required": true}
	if result.Protocol != ssfvMaintenanceResultProtocol || result.FormatVersion != 1 || result.Operation != operation ||
		result.JournalPresent == nil || result.Changed == nil || result.Recovered == nil || result.ReadOnly == nil ||
		!explicitFalse(result.CanonicalApplyEnabled) || !explicitFalse(result.Canonical) ||
		result.RepairActions == nil || !states[result.EffectiveState] || !reviews[result.ReviewState] {
		return ssfvMaintenanceResult{}, fmt.Errorf("SSFV maintenance result violates its v1 identity contract")
	}
	journalNull := bytes.Equal(bytes.TrimSpace(result.Journal), []byte("null"))
	if !*result.JournalPresent {
		if !journalNull || result.JournalDigest != nil || result.EffectiveState != "absent" || result.ReviewState != "not_evaluated" {
			return ssfvMaintenanceResult{}, fmt.Errorf("absent SSFV maintenance result is inconsistent")
		}
	} else {
		if journalNull || result.JournalDigest == nil || !validTaggedDigest(*result.JournalDigest) {
			return ssfvMaintenanceResult{}, fmt.Errorf("present SSFV maintenance result lacks a journal digest")
		}
		var journal struct {
			Protocol      string `json:"protocol"`
			State         string `json:"state"`
			ReviewState   string `json:"review_state"`
			JournalDigest string `json:"journal_digest"`
			Canonical     *bool  `json:"canonical"`
		}
		if err := json.Unmarshal(result.Journal, &journal); err != nil ||
			journal.Protocol != "symphony.knowledge.ssfv-maintenance-journal.v1" ||
			journal.State != result.EffectiveState || journal.ReviewState != result.ReviewState ||
			journal.JournalDigest != *result.JournalDigest || !explicitFalse(journal.Canonical) {
			return ssfvMaintenanceResult{}, fmt.Errorf("SSFV maintenance journal identity is inconsistent")
		}
		if err := validateSSFVEmbeddedDigest(result.Journal, "journal_digest", journal.JournalDigest); err != nil {
			return ssfvMaintenanceResult{}, err
		}
	}
	if operation == "ssfv_maintenance_status" && (!*result.ReadOnly || *result.Changed || *result.Recovered) {
		return ssfvMaintenanceResult{}, fmt.Errorf("SSFV maintenance status is not read-only")
	}
	return result, nil
}

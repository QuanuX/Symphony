package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	qxversion "github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

const namedVersionResultProtocol = "symphony.knowledge.named-version-result.v1"

var namedVersionCapabilities = []string{
	"atomic-head-v1", "content-addressed-named-version-v1", "dual-slot-registry-v1",
	"expected-state-cas-v1", "idempotent-operation-v1", "immutable-object-v1",
	"opaque-extension-preservation-v1", "recovery-forward-v1", "ssiag-capability-binding-v1",
}

type namedVersionOptions struct {
	topsID                 string
	scope                  string
	stateRoot              string
	repository             string
	input                  string
	operationID            string
	expectedRegistryDigest string
	preparedOperationID    string
	proposalDigest         string
	alias                  string
	digest                 string
	id                     string
	discover               bool
	ttl                    time.Duration
	jsonOutput             bool
}

type namedVersionResult struct {
	Protocol              string          `json:"protocol"`
	FormatVersion         uint64          `json:"format_version"`
	Operation             string          `json:"operation"`
	Compatibility         json.RawMessage `json:"compatibility"`
	RegistryPresent       *bool           `json:"registry_present"`
	RegistryDigest        *string         `json:"registry_digest"`
	VersionCount          uint64          `json:"version_count"`
	AliasCount            uint64          `json:"alias_count"`
	ProposalDigest        *string         `json:"proposal_digest"`
	Artifact              json.RawMessage `json:"artifact"`
	SelectedAlias         *string         `json:"selected_alias"`
	Changed               *bool           `json:"changed"`
	Recovered             *bool           `json:"recovered"`
	RepairActions         []string        `json:"repair_actions"`
	ReadOnly              *bool           `json:"read_only"`
	CanonicalApplyEnabled *bool           `json:"canonical_apply_enabled"`
	Canonical             *bool           `json:"canonical"`
	STAVAppendEnabled     *bool           `json:"stav_append_enabled"`
	ResultDigest          string          `json:"result_digest"`
	AuditDisposition      string          `json:"-"`
	AuditCandidateDigest  string          `json:"-"`
}

func runNamedVersion(operation string, options namedVersionOptions) error {
	result, raw, err := executeNamedVersion(operation, options)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var display any
		if err := json.Unmarshal(raw, &display); err != nil {
			return fmt.Errorf("decode Named Version result: %w", err)
		}
		if result.AuditDisposition != "" {
			display = map[string]any{"audit": map[string]any{"candidate_digest": result.AuditCandidateDigest, "disposition": result.AuditDisposition}, "result": display}
		}
		return printIndentedJSON(display)
	}
	registry := "absent"
	if result.RegistryDigest != nil {
		registry = *result.RegistryDigest
	}
	proposal := "none"
	if result.ProposalDigest != nil {
		proposal = *result.ProposalDigest
	}
	alias := "none"
	if result.SelectedAlias != nil {
		alias = *result.SelectedAlias
	}
	audit := result.AuditDisposition
	if audit == "" {
		audit = "not_configured"
	}
	fmt.Printf("SAV Named Version: operation=%s versions=%d aliases=%d registry=%s proposal=%s alias=%s changed=%t recovered=%t canonical=false stav=%s\n",
		operation, result.VersionCount, result.AliasCount, registry, proposal, alias,
		*result.Changed, *result.Recovered, audit)
	return nil
}

func executeNamedVersion(operation string, options namedVersionOptions) (namedVersionResult, json.RawMessage, error) {
	if options.topsID == "" {
		return namedVersionResult{}, nil, fmt.Errorf("--tops-id is required")
	}
	if options.scope != "user" && options.scope != "system" {
		return namedVersionResult{}, nil, fmt.Errorf("--scope must be user or system")
	}
	if options.ttl <= 0 || options.ttl > 24*time.Hour {
		return namedVersionResult{}, nil, fmt.Errorf("--ttl must be greater than zero and no more than 24h")
	}
	mutating := operation == "named_version_prepare" || operation == "named_version_seal" ||
		operation == "named_version_alias" || operation == "named_version_recover"
	if mutating && !validSessionToken(options.operationID) {
		return namedVersionResult{}, nil, fmt.Errorf("--operation-id is required and must be a stable token")
	}
	if !mutating && options.operationID != "" {
		return namedVersionResult{}, nil, fmt.Errorf("read-only Named Version operations do not accept --operation-id")
	}
	expected := options.expectedRegistryDigest
	switch operation {
	case "named_version_status", "named_version_lookup":
		if expected != "" || options.discover {
			return namedVersionResult{}, nil, fmt.Errorf("read-only Named Version operations do not accept mutation state")
		}
	case "named_version_prepare", "named_version_seal", "named_version_alias":
		if expected != "absent" && !validTaggedDigest(expected) {
			return namedVersionResult{}, nil, fmt.Errorf("--expected-registry-digest must be absent or an exact tagged SHA-256 digest")
		}
	case "named_version_recover":
		if options.discover == (expected != "") {
			return namedVersionResult{}, nil, fmt.Errorf("exactly one of --expected-registry-digest or --discover is required")
		}
		if options.discover {
			expected = "discover"
		} else if !validTaggedDigest(expected) {
			return namedVersionResult{}, nil, fmt.Errorf("--expected-registry-digest must be an exact tagged SHA-256 digest")
		}
	default:
		return namedVersionResult{}, nil, fmt.Errorf("unsupported Named Version operation")
	}

	repositoryRoot, err := resolveKnowledgeRepository(options.repository)
	if err != nil {
		return namedVersionResult{}, nil, err
	}
	store, err := knowledgebinding.NewStore(options.stateRoot)
	if err != nil {
		return namedVersionResult{}, nil, err
	}
	bindings, err := store.Snapshot()
	if err != nil {
		return namedVersionResult{}, nil, err
	}
	if !bindings.Exists || !validTaggedDigest(bindings.Registry.RegistryDigest) {
		return namedVersionResult{}, nil, fmt.Errorf("knowledge engine binding registry is absent or invalid")
	}
	coordinator, err := exactKnowledgeBinding(bindings, "coordinator")
	if err != nil {
		return namedVersionResult{}, nil, err
	}
	coordinatorInstallation, err := verifyKnowledgeBinding(*coordinator)
	if err != nil {
		return namedVersionResult{}, nil, err
	}

	var versionEvidence any
	var validationEvidence any
	var engineEvidence any
	if operation == "named_version_prepare" {
		savBinding, bindingErr := exactKnowledgeBinding(bindings, "sav")
		if bindingErr != nil {
			return namedVersionResult{}, nil, bindingErr
		}
		savInstallation, inspectErr := verifyKnowledgeBinding(*savBinding)
		if inspectErr != nil {
			return namedVersionResult{}, nil, inspectErr
		}
		input, readErr := knowledgeengine.ReadPayload(options.input)
		if readErr != nil {
			return namedVersionResult{}, nil, readErr
		}
		var wrapper map[string]json.RawMessage
		if err := json.Unmarshal(input, &wrapper); err != nil || len(wrapper) != 1 || len(wrapper["named_version"]) == 0 {
			return namedVersionResult{}, nil, fmt.Errorf("--input must contain the exact SAV named-version validation payload")
		}
		response, invokeErr := knowledgeengine.InvokeSAV(context.Background(), savInstallation.Prefix,
			savInstallation.Version, repositoryRoot, "named_version_validate", input)
		if invokeErr != nil {
			return namedVersionResult{}, nil, invokeErr
		}
		if _, _, err := validateAccordareResult("sav", "named_version_validate", response.Result); err != nil {
			return namedVersionResult{}, nil, err
		}
		var candidate any
		if err := decodeStrictJSON(wrapper["named_version"], &candidate); err != nil {
			return namedVersionResult{}, nil, fmt.Errorf("decode Named Version artifact: %w", err)
		}
		var validation any
		if err := decodeStrictJSON(response.Result, &validation); err != nil {
			return namedVersionResult{}, nil, fmt.Errorf("decode SAV validation result: %w", err)
		}
		engineMap := map[string]any{
			"module_id": savInstallation.ModuleID, "engine_id": savInstallation.EngineID, "vector_id": "sav",
			"version": savInstallation.Version, "receipt_digest": savInstallation.ReceiptDigest,
			"executable_digest": savInstallation.ExecutableDigest,
		}
		engineMap["evidence_digest"], err = maintenanceObjectDigest(engineMap, "evidence_digest")
		if err != nil {
			return namedVersionResult{}, nil, err
		}
		versionEvidence, validationEvidence, engineEvidence = candidate, validation, engineMap
	} else if options.input != "" {
		return namedVersionResult{}, nil, fmt.Errorf("--input is accepted only by named-version propose")
	}

	var selector any
	if operation == "named_version_lookup" {
		count := 0
		if options.digest != "" {
			count++
			selector = map[string]any{"kind": "digest", "value": options.digest}
		}
		if options.id != "" {
			count++
			selector = map[string]any{"kind": "id", "value": options.id}
		}
		if options.alias != "" {
			count++
			selector = map[string]any{"kind": "alias", "value": options.alias}
		}
		if count != 1 {
			return namedVersionResult{}, nil, fmt.Errorf("exactly one of --digest, --id, or --alias is required")
		}
	} else if operation == "named_version_alias" {
		if options.alias == "" || !validTaggedDigest(options.digest) {
			return namedVersionResult{}, nil, fmt.Errorf("alias selection requires --alias and an exact --digest")
		}
		selector = map[string]any{"kind": "digest", "value": options.digest}
	}
	if operation == "named_version_seal" {
		if !validSessionToken(options.preparedOperationID) || !validTaggedDigest(options.proposalDigest) {
			return namedVersionResult{}, nil, fmt.Errorf("seal requires --prepared-operation-id and --proposal-digest")
		}
	}

	payload := map[string]any{
		"protocol": "symphony.knowledge.named-version-command.v1", "operation": operation,
		"state_root": store.StateRoot(), "tops_id": options.topsID,
		"operation_id": nil, "expected_registry_digest": nil,
		"named_version": versionEvidence, "validation_result": validationEvidence, "sav_engine": engineEvidence,
		"prepared_operation_id": nil, "proposal_digest": nil, "alias": nil, "selector": selector,
		"authorization_decision": nil,
		"client": map[string]any{
			"client_id": "qxctl", "client_version": strings.ReplaceAll(qxversion.Version, " ", "-"),
			"process_protocols":      []string{"symphony.knowledge.engine-process.v1"},
			"registry_read_versions": []uint64{1}, "registry_write_versions": []uint64{1},
			"capabilities": namedVersionCapabilities,
		},
	}
	if mutating {
		payload["operation_id"] = options.operationID
		payload["expected_registry_digest"] = expected
	}
	if operation == "named_version_seal" {
		payload["prepared_operation_id"] = options.preparedOperationID
		payload["proposal_digest"] = options.proposalDigest
	}
	if operation == "named_version_alias" {
		payload["alias"] = options.alias
	}
	decision, err := authorizeNamedVersion(options.topsID, options.scope, options.ttl, operation, payload)
	if err != nil {
		return namedVersionResult{}, nil, err
	}
	payload["authorization_decision"] = decision
	encoded, err := json.Marshal(payload)
	if err != nil {
		return namedVersionResult{}, nil, fmt.Errorf("encode Named Version command: %w", err)
	}
	var auditSession namedVersionAuditSession
	if mutating {
		auditSession, err = prepareNamedVersionAudit(options, operation, encoded, coordinatorInstallation)
		if err != nil {
			return namedVersionResult{}, nil, err
		}
	}
	response, err := knowledgeengine.InvokeCoordinator(context.Background(), coordinator.Prefix,
		coordinator.Version, repositoryRoot, operation, encoded)
	if err != nil {
		if auditSession.configured {
			return namedVersionResult{}, nil, fmt.Errorf("Named Version coordinator did not return terminal evidence; durable audit intent %s requires exact command retry: %w", auditSession.intentID, err)
		}
		return namedVersionResult{}, nil, err
	}
	result, err := validateNamedVersionResult(response.Result, operation)
	if err != nil {
		return namedVersionResult{}, nil, err
	}
	if len(bytes.TrimSpace(result.Artifact)) > 0 && !bytes.Equal(bytes.TrimSpace(result.Artifact), []byte("null")) {
		if err := revalidateStoredNamedVersion(repositoryRoot, bindings, result.Artifact); err != nil {
			return namedVersionResult{}, nil, err
		}
	}
	if mutating {
		audit, configured, auditErr := completeNamedVersionAudit(auditSession, response.Result)
		if auditErr != nil {
			return namedVersionResult{}, nil, fmt.Errorf("Named Version mutation completed but durable audit submission failed: %w", auditErr)
		}
		if configured {
			result.AuditDisposition = audit.Disposition
			result.AuditCandidateDigest = audit.CandidateDigest
		}
	}
	return result, append(json.RawMessage(nil), response.Result...), nil
}

type namedVersionAuditSession struct {
	client     *accordareclient.Client
	configured bool
	intentID   string
	request    accordareclient.Submission
}

func prepareNamedVersionAudit(options namedVersionOptions, operation string, command []byte, coordinator knowledgeengine.Installation) (namedVersionAuditSession, error) {
	path, err := accordareclient.ConfigPath(options.scope, options.topsID)
	if err != nil {
		return namedVersionAuditSession{}, err
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return namedVersionAuditSession{}, nil
	} else if err != nil {
		return namedVersionAuditSession{configured: true}, err
	}
	client, err := accordareclient.NewFromConfig(path)
	if err != nil {
		return namedVersionAuditSession{configured: true}, err
	}
	request := accordareclient.Submission{Command: command, Coordinator: accordareclient.InstallationEvidence{
		ComponentID: "knowledge-session-coordinator", ExecutableDigest: coordinator.ExecutableDigest,
		ReceiptDigest: coordinator.ReceiptDigest, Version: coordinator.Version,
	}, Operation: operation, TOPSID: options.topsID}
	requestID, err := randomUUID()
	if err != nil {
		return namedVersionAuditSession{configured: true}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	audit, err := client.Prepare(ctx, requestID, request)
	if err != nil {
		return namedVersionAuditSession{configured: true}, err
	}
	if audit.Disposition != "prepared" || audit.IntentID == "" {
		return namedVersionAuditSession{configured: true}, fmt.Errorf("producer rejected pre-mutation audit intent: %s", audit.ReasonCode)
	}
	return namedVersionAuditSession{client: client, configured: true, intentID: audit.IntentID, request: request}, nil
}

func completeNamedVersionAudit(session namedVersionAuditSession, result []byte) (accordareclient.AuditResult, bool, error) {
	if !session.configured {
		return accordareclient.AuditResult{}, false, nil
	}
	requestID, err := randomUUID()
	if err != nil {
		return accordareclient.AuditResult{}, true, err
	}
	session.request.Result = result
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	audit, err := session.client.Complete(ctx, requestID, session.request)
	if err != nil {
		return accordareclient.AuditResult{}, true, err
	}
	if audit.IntentID != session.intentID || (audit.Disposition != "committed" && audit.Disposition != "pending") {
		return accordareclient.AuditResult{}, true, fmt.Errorf("producer rejected exact audit completion: %s", audit.ReasonCode)
	}
	return audit, true, nil
}

func authorizeNamedVersion(topsID, scope string, ttl time.Duration, operation string,
	payload map[string]any) (ssiagclient.AuthorizationDecision, error) {
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
		Resource: namedVersionResource(payload), Audience: "qxctl", Scope: "tops:" + topsID,
		RequestedAt: now, RequestedExpiresAt: now.Add(ttl).UTC().Truncate(time.Second),
	}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	if err := validateSessionAuthorization(decision, request, topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, fmt.Errorf("SSIAG Named Version authorization rejected: %w", err)
	}
	return decision, nil
}

func namedVersionResource(payload map[string]any) string {
	var versionDigest any
	if version, ok := payload["named_version"].(map[string]any); ok {
		versionDigest = version["named_version_digest"]
	}
	normalized := map[string]any{
		"tops_id": payload["tops_id"], "operation": payload["operation"],
		"expected_registry_digest": payload["expected_registry_digest"],
		"named_version_digest":     versionDigest, "prepared_operation_id": payload["prepared_operation_id"],
		"proposal_digest": payload["proposal_digest"], "alias": payload["alias"], "selector": payload["selector"],
	}
	encoded, _ := json.Marshal(normalized)
	digest := sha256.Sum256(encoded)
	return "symphony.knowledge.named-version:" + hex.EncodeToString(digest[:])
}

func validateNamedVersionResult(raw json.RawMessage, operation string) (namedVersionResult, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return namedVersionResult{}, fmt.Errorf("decode Named Version result: %w", err)
	}
	required := []string{"protocol", "format_version", "operation", "compatibility", "registry_present",
		"registry_digest", "version_count", "alias_count", "proposal_digest", "artifact", "selected_alias",
		"changed", "recovered", "repair_actions", "read_only", "canonical_apply_enabled", "canonical",
		"stav_append_enabled", "result_digest"}
	if len(fields) != len(required) {
		return namedVersionResult{}, fmt.Errorf("Named Version result has an invalid field set")
	}
	for _, field := range required {
		if _, present := fields[field]; !present {
			return namedVersionResult{}, fmt.Errorf("Named Version result is incomplete")
		}
	}
	var result namedVersionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return namedVersionResult{}, err
	}
	if result.Protocol != namedVersionResultProtocol || result.FormatVersion != 1 || result.Operation != operation ||
		result.RegistryPresent == nil || result.Changed == nil || result.Recovered == nil || result.ReadOnly == nil ||
		!explicitFalse(result.CanonicalApplyEnabled) || !explicitFalse(result.Canonical) ||
		!explicitFalse(result.STAVAppendEnabled) || result.RepairActions == nil || !validTaggedDigest(result.ResultDigest) ||
		result.VersionCount > 4096 || result.AliasCount > 4096 {
		return namedVersionResult{}, fmt.Errorf("Named Version result violates its v1 identity contract")
	}
	if *result.RegistryPresent != (result.RegistryDigest != nil) ||
		(result.RegistryDigest != nil && !validTaggedDigest(*result.RegistryDigest)) ||
		(result.ProposalDigest != nil && !validTaggedDigest(*result.ProposalDigest)) {
		return namedVersionResult{}, fmt.Errorf("Named Version result state is inconsistent")
	}
	computed, err := maintenanceObjectDigest(fields, "result_digest")
	if err != nil || computed != result.ResultDigest {
		return namedVersionResult{}, fmt.Errorf("Named Version result digest mismatch")
	}
	if (operation == "named_version_status" || operation == "named_version_lookup") &&
		(!*result.ReadOnly || *result.Changed || *result.Recovered) {
		return namedVersionResult{}, fmt.Errorf("Named Version read-only result reports mutation")
	}
	return result, nil
}

func revalidateStoredNamedVersion(repositoryRoot string, bindings knowledgebinding.Snapshot,
	artifact json.RawMessage) error {
	savBinding, err := exactKnowledgeBinding(bindings, "sav")
	if err != nil {
		return err
	}
	sav, err := verifyKnowledgeBinding(*savBinding)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"named_version": json.RawMessage(artifact)})
	if err != nil {
		return err
	}
	response, err := knowledgeengine.InvokeSAV(context.Background(), sav.Prefix, sav.Version,
		repositoryRoot, "named_version_validate", payload)
	if err != nil {
		return fmt.Errorf("revalidate stored Named Version: %w", err)
	}
	if _, _, err := validateAccordareResult("sav", "named_version_validate", response.Result); err != nil {
		return fmt.Errorf("stored Named Version failed SAV revalidation: %w", err)
	}
	return nil
}

func decodeStrictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON values are not allowed")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

package foundationlifecycle

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const (
	CommandProtocol     = "symphony.foundation.lifecycle-command.v1"
	AdapterProtocol     = "symphony.foundation.lifecycle-adapter.v1"
	ResultProtocol      = "symphony.foundation.lifecycle-result.v1"
	ObservationProtocol = "symphony.foundation.lifecycle-observation.v1"
	PlanProtocol        = "symphony.foundation.lifecycle-plan.v1"
	maxDescriptorBytes  = 256 * 1024
	maxDiagnosticBytes  = 64 * 1024
	maxRequestBytes     = 1024 * 1024
	maxResponseBytes    = 4 * 1024 * 1024
	maxJSONValues       = 65536
	operationTimeout    = 60 * time.Second
)

var componentBinaries = map[string]string{
	"ssiag": "symphony-ssiag", "stav": "symphony-stav-append-authority",
}

type Options struct {
	Component           string
	Prefix              string
	Version             string
	Surface             string
	Operation           string
	Scope               string
	TOPSID              string
	OperationID         *string
	ExpectedStateDigest *string
	ExpectedAttempt     *string
	Intent              *Intent
	Plan                *Plan
	Discover            bool
	Now                 time.Time
}

type Intent struct {
	DesiredState string  `json:"desired_state"`
	TOPSName     *string `json:"tops_name"`
	ServiceUID   *uint32 `json:"service_uid"`
	ServiceGID   *uint32 `json:"service_gid"`
	AuthorityUID *uint32 `json:"authority_uid"`
	AuthorityGID *uint32 `json:"authority_gid"`
	AuditMode    string  `json:"audit_mode"`
	TTLSeconds   uint64  `json:"ttl_seconds"`
}

type Plan struct {
	Protocol            string  `json:"protocol"`
	FormatVersion       uint64  `json:"format_version"`
	Component           string  `json:"component"`
	Surface             string  `json:"surface"`
	Scope               string  `json:"scope"`
	TOPSID              string  `json:"tops_id"`
	OperationID         string  `json:"operation_id"`
	RequestID           string  `json:"request_id"`
	CorrelationID       string  `json:"correlation_id"`
	ExpectedStateDigest string  `json:"expected_state_digest"`
	DesiredState        string  `json:"desired_state"`
	TOPSName            *string `json:"tops_name"`
	ServiceUID          *uint32 `json:"service_uid"`
	ServiceGID          *uint32 `json:"service_gid"`
	AuthorityUID        *uint32 `json:"authority_uid"`
	AuthorityGID        *uint32 `json:"authority_gid"`
	AuditMode           string  `json:"audit_mode"`
	CreatedAt           string  `json:"created_at"`
	ExpiresAt           string  `json:"expires_at"`
	PlanDigest          string  `json:"plan_digest"`
}

type Installation struct {
	State                 string  `json:"state"`
	BinaryPath            *string `json:"binary_path"`
	BinaryDigest          *string `json:"binary_digest"`
	InstallEvidenceDigest *string `json:"install_evidence_digest"`
	ReceiptDigest         *string `json:"receipt_digest"`
	Legacy                bool    `json:"legacy"`
}

type Enrollment struct {
	State         string  `json:"state"`
	RecordPath    *string `json:"record_path"`
	RecordDigest  *string `json:"record_digest"`
	ConfigPath    *string `json:"config_path"`
	ConfigDigest  *string `json:"config_digest"`
	UID           *uint32 `json:"uid"`
	GID           *uint32 `json:"gid"`
	DataPreserved bool    `json:"data_preserved"`
}

type Supervisor struct {
	Manager              *string `json:"manager"`
	ManagerState         string  `json:"manager_state"`
	DescriptorState      string  `json:"descriptor_state"`
	DescriptorPath       *string `json:"descriptor_path"`
	DescriptorDigest     *string `json:"descriptor_digest"`
	Enablement           string  `json:"enablement"`
	ProcessState         string  `json:"process_state"`
	EndpointState        string  `json:"endpoint_state"`
	ActivationGeneration *string `json:"activation_generation"`
	PackageReceiptDigest *string `json:"package_receipt_digest"`
}

type Observation struct {
	Protocol            string       `json:"protocol"`
	FormatVersion       uint64       `json:"format_version"`
	Component           string       `json:"component"`
	Surface             string       `json:"surface"`
	Scope               string       `json:"scope"`
	TOPSID              string       `json:"tops_id"`
	Installation        Installation `json:"installation"`
	Enrollment          Enrollment   `json:"enrollment"`
	Supervisor          Supervisor   `json:"supervisor"`
	RecoveryRequired    bool         `json:"recovery_required"`
	ActiveAttemptDigest *string      `json:"active_attempt_digest"`
	ObservedAt          string       `json:"observed_at"`
	StableStateDigest   string       `json:"stable_state_digest"`
	ObservationDigest   string       `json:"observation_digest"`
}

type ResultError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	Protocol               string       `json:"protocol"`
	FormatVersion          uint64       `json:"format_version"`
	Operation              string       `json:"operation"`
	Component              string       `json:"component"`
	Surface                string       `json:"surface"`
	Scope                  string       `json:"scope"`
	TOPSID                 string       `json:"tops_id"`
	OperationID            *string      `json:"operation_id"`
	Disposition            string       `json:"disposition"`
	DesiredState           *string      `json:"desired_state"`
	Observation            Observation  `json:"observation"`
	Plan                   *Plan        `json:"plan"`
	Changed                bool         `json:"changed"`
	Replayed               bool         `json:"replayed"`
	Recovered              bool         `json:"recovered"`
	RecoveryRequired       bool         `json:"recovery_required"`
	ReconciliationRequired bool         `json:"reconciliation_required"`
	AttemptDigest          *string      `json:"attempt_digest"`
	AuditState             string       `json:"audit_state"`
	AuditReceiptDigest     *string      `json:"audit_receipt_digest"`
	StartedAt              *string      `json:"started_at"`
	CompletedAt            string       `json:"completed_at"`
	Error                  *ResultError `json:"error"`
	ReadOnly               bool         `json:"read_only"`
	Canonical              bool         `json:"canonical"`
	ResultDigest           string       `json:"result_digest"`
}

type Adapter struct {
	Protocol              string        `json:"protocol"`
	FormatVersion         uint64        `json:"format_version"`
	Component             string        `json:"component"`
	AdapterVersion        string        `json:"adapter_version"`
	BinaryPath            string        `json:"binary_path"`
	BinaryDigest          string        `json:"binary_digest"`
	InstallEvidenceDigest string        `json:"install_evidence_digest"`
	Operations            []string      `json:"operations"`
	SupportedScopes       []string      `json:"supported_scopes"`
	SupportedManagers     []string      `json:"supported_managers"`
	Compatibility         Compatibility `json:"compatibility"`
	Limits                Limits        `json:"limits"`
	CanonicalApplyEnabled bool          `json:"canonical_apply_enabled"`
	NetworkListener       bool          `json:"network_listener"`
	DescriptorDigest      string        `json:"descriptor_digest"`
}

type Compatibility struct {
	ConfigReadMajors  []uint64 `json:"config_read_majors"`
	ConfigWriteMajor  uint64   `json:"config_write_major"`
	RuntimeReadMajors []uint64 `json:"runtime_read_majors"`
	RuntimeWriteMajor uint64   `json:"runtime_write_major"`
	StateReadMajors   []uint64 `json:"state_read_majors"`
	StateWriteMajor   uint64   `json:"state_write_major"`
	RollbackReadable  bool     `json:"rollback_readable"`
}

type Limits struct {
	RequestBytes  uint64 `json:"request_bytes"`
	ResponseBytes uint64 `json:"response_bytes"`
	DeadlineMS    uint64 `json:"deadline_ms"`
	JSONDepth     uint64 `json:"json_depth"`
	JSONValues    uint64 `json:"json_values"`
}

type command struct {
	Protocol              string  `json:"protocol"`
	FormatVersion         uint64  `json:"format_version"`
	Operation             string  `json:"operation"`
	Component             string  `json:"component"`
	Surface               string  `json:"surface"`
	Scope                 string  `json:"scope"`
	TOPSID                string  `json:"tops_id"`
	OperationID           *string `json:"operation_id"`
	RequestID             *string `json:"request_id"`
	CorrelationID         *string `json:"correlation_id"`
	ExpectedStateDigest   *string `json:"expected_state_digest"`
	ExpectedAttemptDigest *string `json:"expected_attempt_digest"`
	Intent                *Intent `json:"intent"`
	Plan                  *Plan   `json:"plan"`
	Discover              bool    `json:"discover"`
	RequestedAt           string  `json:"requested_at"`
	DeadlineAt            string  `json:"deadline_at"`
}

type installEvidence struct {
	digest       string
	binary       string
	binaryDigest string
	version      string
}

func Invoke(ctx context.Context, options Options) (Result, error) {
	if err := validateOptions(options); err != nil {
		return Result{}, err
	}
	evidence, err := verifiedInstallation(options.Component, options.Prefix, options.Version)
	if err != nil {
		return Result{}, err
	}
	adapter, err := describe(ctx, evidence, options.Component, options.Scope)
	if err != nil {
		return Result{}, err
	}
	operationID := "engop:symphony:" + options.Component + "." + options.Surface + "." + strings.ReplaceAll(options.Operation, "_", "-")
	if !contains(adapter.Operations, operationID) || !contains(adapter.SupportedScopes, options.Scope) {
		return Result{}, fmt.Errorf("installed %s lifecycle adapter does not support %s in %s scope", options.Component, operationID, options.Scope)
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC().Truncate(time.Second)
	} else {
		now = now.UTC().Truncate(time.Second)
	}
	requestID, err := randomUUID()
	if err != nil {
		return Result{}, fmt.Errorf("generate lifecycle request identity: %w", err)
	}
	deadline := now.Add(time.Duration(adapter.Limits.DeadlineMS) * time.Millisecond).UTC().Truncate(time.Second)
	request := command{
		Protocol: CommandProtocol, FormatVersion: 1, Operation: options.Operation,
		Component: options.Component, Surface: options.Surface, Scope: options.Scope, TOPSID: options.TOPSID,
		OperationID: options.OperationID, RequestID: &requestID, CorrelationID: &requestID,
		ExpectedStateDigest: options.ExpectedStateDigest, ExpectedAttemptDigest: options.ExpectedAttempt,
		Intent: options.Intent, Plan: options.Plan, Discover: options.Discover,
		RequestedAt: timestamp(now), DeadlineAt: timestamp(deadline),
	}
	encoded, err := json.Marshal(request)
	if err != nil || len(encoded) > int(adapter.Limits.RequestBytes) || len(encoded) > maxRequestBytes {
		return Result{}, fmt.Errorf("encoded foundational lifecycle command exceeds the negotiated bound")
	}
	processDeadline := time.Duration(adapter.Limits.DeadlineMS)*time.Millisecond + time.Second
	if processDeadline > operationTimeout+time.Second {
		processDeadline = operationTimeout + time.Second
	}
	child, cancel := context.WithTimeout(ctx, processDeadline)
	defer cancel()
	stdout, stderr := &boundedBuffer{limit: min(int(adapter.Limits.ResponseBytes), maxResponseBytes)}, &boundedBuffer{limit: maxDiagnosticBytes}
	process := exec.CommandContext(child, evidence.binary, "foundation-lifecycle")
	process.Env = controlledEnvironment(options.Scope)
	process.Dir = filepath.Dir(evidence.binary)
	process.Stdin, process.Stdout, process.Stderr = bytes.NewReader(encoded), stdout, stderr
	runErr := process.Run()
	if child.Err() != nil {
		return Result{}, fmt.Errorf("%s lifecycle adapter exceeded its hard process deadline", options.Component)
	}
	if stdout.exceeded || stderr.exceeded {
		return Result{}, fmt.Errorf("%s lifecycle adapter exceeded a process output bound", options.Component)
	}
	result, resultErr := validateResult(stdout.Bytes(), request, adapter)
	if resultErr != nil {
		return Result{}, resultErr
	}
	if runErr != nil && result.Error == nil {
		return Result{}, fmt.Errorf("%s lifecycle adapter failed without a structured error result", options.Component)
	}
	if result.Error == nil && len(stderr.Bytes()) != 0 {
		return Result{}, fmt.Errorf("%s lifecycle adapter emitted diagnostics during a successful operation", options.Component)
	}
	if result.Error != nil {
		return result, fmt.Errorf("%s lifecycle adapter %s: %s", options.Component, result.Error.Code, result.Error.Message)
	}
	return result, nil
}

func ReadPlan(path string) (Plan, error) {
	var plan Plan
	data, err := knowledgeengine.ReadPayload(path)
	if err != nil {
		return plan, fmt.Errorf("read foundational lifecycle plan: %w", err)
	}
	var header struct {
		Protocol string `json:"protocol"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return plan, fmt.Errorf("decode foundational lifecycle plan identity: %w", err)
	}
	if header.Protocol == ResultProtocol {
		var result Result
		if err := decodeStrict(data, &result, maxResponseBytes); err != nil {
			return plan, fmt.Errorf("decode foundational lifecycle plan result: %w", err)
		}
		if result.Protocol != ResultProtocol || result.FormatVersion != 1 || result.Operation != "plan" || result.Disposition != "planned" ||
			result.Plan == nil || result.Error != nil || !result.ReadOnly || result.Changed || result.Replayed || result.Recovered ||
			result.RecoveryRequired || result.ReconciliationRequired || result.AttemptDigest != nil || result.AuditState != "not_applicable" ||
			result.AuditReceiptDigest != nil || result.StartedAt != nil || result.Canonical || !validTimestamp(result.CompletedAt) {
			return plan, fmt.Errorf("foundational lifecycle plan result violates the required contract")
		}
		if result.OperationID == nil || *result.OperationID != result.Plan.OperationID || result.DesiredState == nil || *result.DesiredState != result.Plan.DesiredState {
			return plan, fmt.Errorf("foundational lifecycle plan result identity mismatch")
		}
		if err := validateObservation(result.Observation, result.Component, result.Surface, result.Scope, result.TOPSID); err != nil {
			return plan, err
		}
		if err := verifyDigest(result, "result_digest"); err != nil {
			return plan, err
		}
		plan = *result.Plan
	} else {
		if err := decodeStrict(data, &plan, maxRequestBytes); err != nil {
			return plan, fmt.Errorf("decode foundational lifecycle plan: %w", err)
		}
	}
	if err := validatePlan(plan, plan.Component, plan.Surface, plan.Scope, plan.TOPSID); err != nil {
		return plan, err
	}
	return plan, nil
}

func verifiedInstallation(component, prefix, version string) (installEvidence, error) {
	name, ok := componentBinaries[component]
	if !ok {
		return installEvidence{}, fmt.Errorf("unsupported foundational component %q", component)
	}
	moduleID := map[string]string{"ssiag": "secure-identity-access-governance", "stav": "stav-append-authority"}[component]
	entryID := component + ".foundation-lifecycle"
	versions := []string{version}
	if version == "" {
		var err error
		versions, err = knowledgeengine.DiscoverReceiptV2Versions(prefix, moduleID)
		if err != nil {
			return installEvidence{}, err
		}
	}
	if len(versions) == 0 {
		return installEvidence{}, fmt.Errorf("no receipt-v2 %s lifecycle adapter is installed under --prefix", component)
	}
	matches := make([]knowledgeengine.ReceiptV2EntryPoint, 0, len(versions))
	for _, candidate := range versions {
		relative := filepath.ToSlash(filepath.Join("libexec", "symphony", moduleID, candidate, name))
		match, err := knowledgeengine.InspectReceiptV2EntryPoint(prefix, candidate, knowledgeengine.ReceiptV2EntryPointSpec{
			Label: component + ".foundation-lifecycle", ComponentID: moduleID, ComponentKind: "service",
			ModuleID: moduleID, PackageID: moduleID, EntryPointID: entryID, EntryPointKind: "adapter",
			EntryPointRelativePath: relative, RequiredProtocols: []string{CommandProtocol},
			RequiredCapabilities: []string{AdapterProtocol},
		})
		if err != nil {
			return installEvidence{}, fmt.Errorf("validate %s lifecycle adapter version %s: %w", component, candidate, err)
		}
		matches = append(matches, match)
	}
	if len(matches) != 1 {
		return installEvidence{}, fmt.Errorf("%s lifecycle adapter selection is ambiguous; specify --version", component)
	}
	match := matches[0]
	return installEvidence{digest: match.ReceiptDigest, binary: match.ExecutablePath, binaryDigest: match.ExecutableDigest, version: match.Version}, nil
}

func describe(ctx context.Context, evidence installEvidence, component, scope string) (Adapter, error) {
	child, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stdout, stderr := &boundedBuffer{limit: maxDescriptorBytes}, &boundedBuffer{limit: maxDiagnosticBytes}
	process := exec.CommandContext(child, evidence.binary, "foundation-lifecycle", "describe", "--json")
	process.Env, process.Dir, process.Stdout, process.Stderr = controlledEnvironment(scope), filepath.Dir(evidence.binary), stdout, stderr
	if err := process.Run(); err != nil || child.Err() != nil {
		return Adapter{}, fmt.Errorf("installed %s binary lacks the required foundational lifecycle adapter descriptor", component)
	}
	if stdout.exceeded || stderr.exceeded || len(stderr.Bytes()) != 0 {
		return Adapter{}, fmt.Errorf("installed %s adapter descriptor violated process output bounds", component)
	}
	var adapter Adapter
	if err := decodeStrict(stdout.Bytes(), &adapter, maxDescriptorBytes); err != nil {
		return Adapter{}, fmt.Errorf("invalid %s lifecycle adapter descriptor: %w", component, err)
	}
	if err := validateAdapter(adapter, evidence, component); err != nil {
		return Adapter{}, err
	}
	return adapter, nil
}

func validateAdapter(value Adapter, evidence installEvidence, component string) error {
	wantOps := make([]string, 0, 10)
	for _, surface := range []string{"enrollment", "supervisor"} {
		for _, operation := range []string{"observe", "plan", "apply", "apply-status", "recover"} {
			wantOps = append(wantOps, "engop:symphony:"+component+"."+surface+"."+operation)
		}
	}
	sort.Strings(wantOps)
	gotOps := append([]string(nil), value.Operations...)
	sort.Strings(gotOps)
	if value.Protocol != AdapterProtocol || value.FormatVersion != 1 || value.Component != component ||
		value.BinaryPath != evidence.binary || value.BinaryDigest != evidence.binaryDigest ||
		value.InstallEvidenceDigest != evidence.digest || value.AdapterVersion != evidence.version || !validVersion(value.AdapterVersion) || !validDigest(value.DescriptorDigest) ||
		strings.Join(gotOps, "\n") != strings.Join(wantOps, "\n") || !validEnumSet(value.SupportedScopes, 1, 2, "user", "system") ||
		!validEnumSet(value.SupportedManagers, 0, 3, "launchd", "systemd", "external") ||
		!validMajors(value.Compatibility.ConfigReadMajors) || !validMajors(value.Compatibility.RuntimeReadMajors) || !validMajors(value.Compatibility.StateReadMajors) ||
		value.Compatibility.ConfigWriteMajor < 1 || value.Compatibility.RuntimeWriteMajor < 1 || value.Compatibility.StateWriteMajor < 1 ||
		value.Limits.RequestBytes < 1 || value.Limits.RequestBytes > maxRequestBytes || value.Limits.ResponseBytes < 1 || value.Limits.ResponseBytes > maxResponseBytes ||
		value.Limits.DeadlineMS < 1 || value.Limits.DeadlineMS > 60000 || value.Limits.JSONDepth < 1 || value.Limits.JSONDepth > 64 || value.Limits.JSONValues < 1 || value.Limits.JSONValues > maxJSONValues ||
		value.CanonicalApplyEnabled || value.NetworkListener {
		return fmt.Errorf("installed %s lifecycle adapter descriptor violates the required capability contract", component)
	}
	if err := verifyDigest(value, "descriptor_digest"); err != nil {
		return fmt.Errorf("installed %s lifecycle adapter descriptor: %w", component, err)
	}
	return nil
}

func validateResult(value []byte, request command, adapter Adapter) (Result, error) {
	var result Result
	if err := decodeStrict(value, &result, min(int(adapter.Limits.ResponseBytes), maxResponseBytes)); err != nil {
		return result, fmt.Errorf("invalid %s lifecycle adapter result: %w", request.Component, err)
	}
	if result.Protocol != ResultProtocol || result.FormatVersion != 1 || result.Operation != request.Operation ||
		result.Component != request.Component || result.Surface != request.Surface || result.Scope != request.Scope || result.TOPSID != request.TOPSID ||
		!resultOperationIDMatches(result.OperationID, request) ||
		!oneOf(result.Disposition, "observed", "planned", "applied", "already_applied", "blocked", "failed", "recovered") ||
		(result.DesiredState != nil && !validDesiredState(*result.DesiredState, result.Surface)) || !validTimestamp(result.CompletedAt) ||
		(result.StartedAt != nil && !validTimestamp(*result.StartedAt)) || result.Canonical || !validDigest(result.ResultDigest) ||
		!oneOf(result.AuditState, "not_applicable", "committed", "audit_deferred", "reconciled") ||
		(result.AttemptDigest != nil && !validDigest(*result.AttemptDigest)) || (result.AuditReceiptDigest != nil && !validDigest(*result.AuditReceiptDigest)) {
		return result, fmt.Errorf("%s lifecycle adapter result identity or bounds are invalid", request.Component)
	}
	if (request.Operation == "observe" || request.Operation == "plan" || request.Operation == "apply_status") != result.ReadOnly ||
		(request.Operation != "recover" && result.Recovered) || result.RecoveryRequired != result.Observation.RecoveryRequired ||
		(result.AuditState == "audit_deferred" && !result.ReconciliationRequired) ||
		(result.AuditState == "reconciled" && (result.ReconciliationRequired || result.AuditReceiptDigest == nil)) ||
		(result.AuditState == "committed" && result.AuditReceiptDigest == nil) ||
		(request.Operation == "observe" && result.AuditState != "not_applicable") ||
		(result.Error == nil) != (result.Disposition != "blocked" && result.Disposition != "failed") ||
		(result.Error != nil && (!validToken(result.Error.Code) || result.Error.Message == "" || len(result.Error.Message) > 4096)) {
		return result, fmt.Errorf("%s lifecycle adapter result disposition is inconsistent", request.Component)
	}
	if err := validateResultOperationShape(result, request); err != nil {
		return result, err
	}
	if err := validateObservation(result.Observation, request.Component, request.Surface, request.Scope, request.TOPSID); err != nil {
		return result, err
	}
	if result.Observation.Installation.State != "installed" || result.Observation.Installation.Legacy ||
		result.Observation.Installation.BinaryPath == nil || *result.Observation.Installation.BinaryPath != adapter.BinaryPath ||
		result.Observation.Installation.BinaryDigest == nil || *result.Observation.Installation.BinaryDigest != adapter.BinaryDigest ||
		result.Observation.Installation.InstallEvidenceDigest == nil || *result.Observation.Installation.InstallEvidenceDigest != adapter.InstallEvidenceDigest ||
		result.Observation.Installation.ReceiptDigest == nil || *result.Observation.Installation.ReceiptDigest != adapter.InstallEvidenceDigest {
		return result, fmt.Errorf("%s lifecycle observation does not bind the invoked installation", request.Component)
	}
	if request.Operation == "plan" && result.Error == nil {
		if result.Plan == nil || result.Disposition != "planned" {
			return result, fmt.Errorf("%s lifecycle plan result omitted its plan", request.Component)
		}
		if err := validatePlan(*result.Plan, request.Component, request.Surface, request.Scope, request.TOPSID); err != nil {
			return result, err
		}
		if err := validatePlanRequestBinding(*result.Plan, request); err != nil {
			return result, err
		}
	}
	if result.Plan != nil {
		if err := validatePlan(*result.Plan, request.Component, request.Surface, request.Scope, request.TOPSID); err != nil {
			return result, err
		}
	}
	if err := verifyDigest(result, "result_digest"); err != nil {
		return result, err
	}
	return result, nil
}

func validateResultOperationShape(result Result, request command) error {
	successDispositions := map[string][]string{
		"observe":      {"observed"},
		"plan":         {"planned"},
		"apply":        {"applied", "already_applied"},
		"apply_status": {"observed"},
		"recover":      {"recovered", "already_applied"},
	}
	if result.Error == nil && !oneOf(result.Disposition, successDispositions[request.Operation]...) {
		return fmt.Errorf("%s lifecycle adapter returned an invalid %s disposition", request.Component, request.Operation)
	}
	if (result.Changed || result.Replayed || result.Recovered) && request.Operation != "apply" && request.Operation != "recover" ||
		result.Changed && result.Replayed || result.Replayed != (result.Disposition == "already_applied") || result.Recovered != (result.Disposition == "recovered") {
		return fmt.Errorf("%s lifecycle adapter returned inconsistent transition flags", request.Component)
	}
	if result.ReconciliationRequired != (result.AuditState == "audit_deferred") ||
		result.AuditState == "audit_deferred" && result.AuditReceiptDigest != nil ||
		result.AuditState == "not_applicable" && result.AuditReceiptDigest != nil {
		return fmt.Errorf("%s lifecycle adapter returned inconsistent audit evidence", request.Component)
	}
	if result.RecoveryRequired && (result.Observation.ActiveAttemptDigest == nil || result.AttemptDigest == nil || *result.AttemptDigest != *result.Observation.ActiveAttemptDigest) {
		return fmt.Errorf("%s lifecycle adapter omitted exact active recovery evidence", request.Component)
	}
	if request.Operation == "observe" {
		if result.DesiredState != nil || result.Plan != nil || result.AttemptDigest != nil || result.StartedAt != nil || result.AuditState != "not_applicable" || result.ReconciliationRequired {
			return fmt.Errorf("%s lifecycle observation contains attempt-only evidence", request.Component)
		}
	}
	if request.Operation == "plan" && result.Error == nil && (result.Plan == nil || result.DesiredState == nil) {
		return fmt.Errorf("%s lifecycle plan result omitted planned evidence", request.Component)
	}
	if request.Operation == "apply_status" && result.Plan != nil {
		return fmt.Errorf("%s lifecycle apply-status unexpectedly returned a mutation plan", request.Component)
	}
	return nil
}

func validatePlanRequestBinding(plan Plan, request command) error {
	if request.OperationID == nil || request.RequestID == nil || request.CorrelationID == nil || request.ExpectedStateDigest == nil || request.Intent == nil ||
		plan.OperationID != *request.OperationID || plan.RequestID != *request.RequestID || plan.CorrelationID != *request.CorrelationID ||
		plan.ExpectedStateDigest != *request.ExpectedStateDigest || plan.DesiredState != request.Intent.DesiredState ||
		!equalOptional(plan.TOPSName, request.Intent.TOPSName) || plan.AuditMode != request.Intent.AuditMode ||
		request.Scope == "system" && (!equalOptionalUint32(plan.ServiceUID, request.Intent.ServiceUID) ||
			!equalOptionalUint32(plan.ServiceGID, request.Intent.ServiceGID) || !equalOptionalUint32(plan.AuthorityUID, request.Intent.AuthorityUID) ||
			!equalOptionalUint32(plan.AuthorityGID, request.Intent.AuthorityGID)) {
		return fmt.Errorf("%s lifecycle plan does not bind the exact request intent", request.Component)
	}
	created, _ := time.Parse("2006-01-02T15:04:05Z", plan.CreatedAt)
	expires, _ := time.Parse("2006-01-02T15:04:05Z", plan.ExpiresAt)
	if expires.Sub(created) != time.Duration(request.Intent.TTLSeconds)*time.Second {
		return fmt.Errorf("%s lifecycle plan changed the requested validity interval", request.Component)
	}
	return nil
}

func resultOperationIDMatches(resultID *string, request command) bool {
	if request.Operation == "apply_status" && request.OperationID == nil {
		return resultID == nil || validToken(*resultID)
	}
	return equalOptional(resultID, request.OperationID)
}

func validateObservation(value Observation, component, surface, scope, topsID string) error {
	if value.Protocol != ObservationProtocol || value.FormatVersion != 1 || value.Component != component || value.Surface != surface || value.Scope != scope || value.TOPSID != topsID ||
		!oneOf(value.Component, "ssiag", "stav") || !oneOf(value.Surface, "enrollment", "supervisor") || !oneOf(value.Scope, "user", "system") || !validUUID(value.TOPSID) ||
		!oneOf(value.Installation.State, "absent", "legacy", "installed", "drifted", "unsupported") ||
		!oneOf(value.Enrollment.State, "unenrolled", "enrolled", "drifted", "unsafe", "unknown") ||
		!oneOf(value.Supervisor.ManagerState, "available", "manager_unavailable", "externally_managed", "unknown") ||
		!oneOf(value.Supervisor.DescriptorState, "absent", "installed", "drifted", "unsafe", "unknown") ||
		!oneOf(value.Supervisor.Enablement, "enabled", "disabled", "not_applicable", "unknown") ||
		!oneOf(value.Supervisor.ProcessState, "running", "stopped", "starting", "stopping", "failed", "unknown") ||
		!oneOf(value.Supervisor.EndpointState, "ready", "absent", "stale", "foreign", "indeterminate", "unknown") ||
		!validTimestamp(value.ObservedAt) || !validDigest(value.StableStateDigest) || !validDigest(value.ObservationDigest) ||
		(value.ActiveAttemptDigest != nil && !validDigest(*value.ActiveAttemptDigest)) || !validNullablePaths(value) || !validNullableDigests(value) {
		return fmt.Errorf("%s lifecycle observation violates the required contract", component)
	}
	if value.Supervisor.Manager != nil && !oneOf(*value.Supervisor.Manager, "launchd", "systemd", "external") {
		return fmt.Errorf("%s lifecycle observation has an invalid supervisor manager", component)
	}
	if value.Supervisor.ActivationGeneration != nil && !validToken(*value.Supervisor.ActivationGeneration) {
		return fmt.Errorf("%s lifecycle observation has an invalid activation generation", component)
	}
	if err := verifyDigest(value, "observation_digest"); err != nil {
		return err
	}
	computed, err := digestWithoutFields(value, "observed_at", "stable_state_digest", "observation_digest")
	if err != nil {
		return err
	}
	if value.StableStateDigest != computed {
		return fmt.Errorf("%s lifecycle stable state digest mismatch", component)
	}
	return nil
}

func validatePlan(value Plan, component, surface, scope, topsID string) error {
	if value.Protocol != PlanProtocol || value.FormatVersion != 1 || value.Component != component || value.Surface != surface || value.Scope != scope || value.TOPSID != topsID ||
		!oneOf(value.Component, "ssiag", "stav") || !oneOf(value.Surface, "enrollment", "supervisor") || !oneOf(value.Scope, "user", "system") || !validUUID(value.TOPSID) ||
		!validToken(value.OperationID) || !validUUID(value.RequestID) || !validUUID(value.CorrelationID) ||
		(value.ExpectedStateDigest != "absent" && !validDigest(value.ExpectedStateDigest)) || !validDesiredState(value.DesiredState, surface) ||
		!oneOf(value.AuditMode, "ordinary", "audit_deferred") || !validTimestamp(value.CreatedAt) || !validTimestamp(value.ExpiresAt) || !validDigest(value.PlanDigest) ||
		(value.TOPSName != nil && (*value.TOPSName == "" || len(*value.TOPSName) > 256)) {
		return fmt.Errorf("%s lifecycle plan violates the required contract", component)
	}
	if surface == "supervisor" && (value.TOPSName != nil || value.ServiceUID != nil || value.ServiceGID != nil || value.AuthorityUID != nil || value.AuthorityGID != nil) {
		return fmt.Errorf("supervisor lifecycle plan contains enrollment-only identity fields")
	}
	if component == "ssiag" && (value.AuthorityUID != nil || value.AuthorityGID != nil) || component == "stav" && (value.TOPSName != nil || value.ServiceUID != nil || value.ServiceGID != nil) {
		return fmt.Errorf("%s lifecycle plan contains another component's identity fields", component)
	}
	if (value.ServiceUID == nil) != (value.ServiceGID == nil) || (value.AuthorityUID == nil) != (value.AuthorityGID == nil) {
		return fmt.Errorf("%s lifecycle plan contains an incomplete service identity", component)
	}
	if surface == "enrollment" && value.DesiredState == "unenrolled_preserved" &&
		(value.TOPSName != nil || value.ServiceUID != nil || value.ServiceGID != nil || value.AuthorityUID != nil || value.AuthorityGID != nil) {
		return fmt.Errorf("unenrollment plan contains enrollment-only identity intent")
	}
	if component == "ssiag" && surface == "enrollment" && value.DesiredState == "enrolled" && value.TOPSName == nil {
		return fmt.Errorf("SSIAG enrollment plan omits the required TOPS name")
	}
	if scope == "user" {
		uid, gid := effectiveIdentity()
		if value.ServiceUID != nil && (*value.ServiceUID != uid || *value.ServiceGID != gid) ||
			value.AuthorityUID != nil && (*value.AuthorityUID != uid || *value.AuthorityGID != gid) {
			return fmt.Errorf("user-scope lifecycle plan contains a non-kernel service identity")
		}
	}
	if surface == "enrollment" && scope == "system" && value.DesiredState == "enrolled" &&
		(component == "ssiag" && value.ServiceUID == nil || component == "stav" && value.AuthorityUID == nil) {
		return fmt.Errorf("system-scope lifecycle plan omits its explicit service identity")
	}
	created, createdErr := time.Parse("2006-01-02T15:04:05Z", value.CreatedAt)
	expires, expiresErr := time.Parse("2006-01-02T15:04:05Z", value.ExpiresAt)
	if createdErr != nil || expiresErr != nil || !expires.After(created) || expires.Sub(created) > 10*time.Minute {
		return fmt.Errorf("%s lifecycle plan has an invalid validity interval", component)
	}
	if err := verifyDigest(value, "plan_digest"); err != nil {
		return err
	}
	return nil
}

func validateOptions(value Options) error {
	if !oneOf(value.Component, "ssiag", "stav") || value.Prefix == "" || !oneOf(value.Surface, "enrollment", "supervisor") || !oneOf(value.Operation, "observe", "plan", "apply", "apply_status", "recover") || !oneOf(value.Scope, "user", "system") || !validUUID(value.TOPSID) {
		return fmt.Errorf("invalid foundational lifecycle target or operation")
	}
	if !foundationPlatformSupported() {
		return fmt.Errorf("foundational lifecycle administration is supported only on Darwin and Linux")
	}
	if value.Scope == "system" && !hasSystemAuthority() {
		return fmt.Errorf("system-scope foundational lifecycle administration requires effective UID zero")
	}
	if value.OperationID != nil && !validToken(*value.OperationID) || value.ExpectedStateDigest != nil && *value.ExpectedStateDigest != "absent" && !validDigest(*value.ExpectedStateDigest) || value.ExpectedAttempt != nil && *value.ExpectedAttempt != "absent" && !validDigest(*value.ExpectedAttempt) {
		return fmt.Errorf("foundational lifecycle operation identity or digest is invalid")
	}
	switch value.Operation {
	case "observe":
		if value.OperationID != nil || value.ExpectedStateDigest != nil || value.ExpectedAttempt != nil || value.Intent != nil || value.Plan != nil || value.Discover {
			return fmt.Errorf("observe accepts no mutation or recovery evidence")
		}
	case "plan":
		if value.Intent == nil || value.OperationID == nil || value.ExpectedStateDigest == nil || value.ExpectedAttempt != nil || value.Plan != nil || value.Discover {
			return fmt.Errorf("plan requires exact operation, state, and intent evidence only")
		}
		if err := validateIntent(*value.Intent, value.Component, value.Surface, value.Scope); err != nil {
			return err
		}
	case "apply":
		if value.Plan == nil || value.OperationID == nil || value.ExpectedStateDigest == nil || value.ExpectedAttempt == nil || value.Intent != nil || value.Discover {
			return fmt.Errorf("apply requires exact plan, operation, state, and attempt evidence only")
		}
		if err := validatePlan(*value.Plan, value.Component, value.Surface, value.Scope, value.TOPSID); err != nil {
			return err
		}
		if *value.OperationID != value.Plan.OperationID || *value.ExpectedStateDigest != value.Plan.ExpectedStateDigest {
			return fmt.Errorf("apply evidence does not match its immutable plan")
		}
	case "apply_status":
		if value.ExpectedStateDigest != nil || value.ExpectedAttempt != nil || value.Intent != nil || value.Plan != nil || value.Discover {
			return fmt.Errorf("apply-status accepts only an optional operation identity")
		}
	case "recover":
		if value.OperationID == nil || value.ExpectedStateDigest != nil || value.Intent != nil || value.Plan != nil || value.Discover == (value.ExpectedAttempt != nil) {
			return fmt.Errorf("recover requires --operation-id and exactly one of --expected-attempt-digest or --discover")
		}
		if value.ExpectedAttempt != nil && !validDigest(*value.ExpectedAttempt) {
			return fmt.Errorf("recovery requires an exact protected attempt digest")
		}
	}
	return nil
}

func validateIntent(value Intent, component, surface, scope string) error {
	if !validDesiredState(value.DesiredState, surface) || !oneOf(value.AuditMode, "ordinary", "audit_deferred") || value.TTLSeconds < 1 || value.TTLSeconds > 600 ||
		(value.TOPSName != nil && (*value.TOPSName == "" || len(*value.TOPSName) > 256)) ||
		(value.ServiceUID == nil) != (value.ServiceGID == nil) || (value.AuthorityUID == nil) != (value.AuthorityGID == nil) {
		return fmt.Errorf("%s lifecycle intent violates the required contract", component)
	}
	if surface == "supervisor" && (value.TOPSName != nil || value.ServiceUID != nil || value.ServiceGID != nil || value.AuthorityUID != nil || value.AuthorityGID != nil) {
		return fmt.Errorf("supervisor lifecycle intent contains enrollment-only identity fields")
	}
	if component == "ssiag" && (value.AuthorityUID != nil || value.AuthorityGID != nil) || component == "stav" && (value.TOPSName != nil || value.ServiceUID != nil || value.ServiceGID != nil) {
		return fmt.Errorf("%s lifecycle intent contains another component's identity fields", component)
	}
	if surface == "enrollment" && value.DesiredState == "unenrolled_preserved" &&
		(value.TOPSName != nil || value.ServiceUID != nil || value.ServiceGID != nil || value.AuthorityUID != nil || value.AuthorityGID != nil) {
		return fmt.Errorf("unenrollment intent contains enrollment-only identity fields")
	}
	if component == "ssiag" && surface == "enrollment" && value.DesiredState == "enrolled" && value.TOPSName == nil {
		return fmt.Errorf("SSIAG enrollment intent omits the required TOPS name")
	}
	if scope == "user" && (value.ServiceUID != nil || value.ServiceGID != nil || value.AuthorityUID != nil || value.AuthorityGID != nil) {
		return fmt.Errorf("user-scope lifecycle intent must derive its service identity")
	}
	if surface == "enrollment" && scope == "system" && value.DesiredState == "enrolled" &&
		(component == "ssiag" && value.ServiceUID == nil || component == "stav" && value.AuthorityUID == nil) {
		return fmt.Errorf("system-scope lifecycle intent omits its explicit service identity")
	}
	return nil
}

func validNullablePaths(value Observation) bool {
	paths := []*string{value.Installation.BinaryPath, value.Enrollment.RecordPath, value.Enrollment.ConfigPath, value.Supervisor.DescriptorPath}
	for _, path := range paths {
		if path != nil && (!filepath.IsAbs(*path) || len(*path) > 4096 || filepath.Clean(*path) != *path) {
			return false
		}
	}
	return true
}

func validNullableDigests(value Observation) bool {
	digests := []*string{value.Installation.BinaryDigest, value.Installation.InstallEvidenceDigest, value.Installation.ReceiptDigest, value.Enrollment.RecordDigest, value.Enrollment.ConfigDigest, value.Supervisor.DescriptorDigest, value.Supervisor.PackageReceiptDigest}
	for _, digest := range digests {
		if digest != nil && !validDigest(*digest) {
			return false
		}
	}
	return true
}

func controlledEnvironment(scope string) []string {
	env := []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"}
	if scope == "user" {
		for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR"} {
			if value, ok := os.LookupEnv(key); ok && filepath.IsAbs(value) {
				env = append(env, key+"="+filepath.Clean(value))
			}
		}
	}
	return env
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if b.Len()+len(data) > b.limit {
		b.exceeded = true
		return 0, fmt.Errorf("output exceeds %d bytes", b.limit)
	}
	return b.Buffer.Write(data)
}

func decodeStrict(data []byte, target any, limit int) error {
	if err := knowledgeengine.ValidateJSONObjectWithValueLimit(data, int64(limit), maxJSONValues); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing JSON value")
	}
	return nil
}

func verifyDigest(value any, field string) error {
	provided, err := stringField(value, field)
	if err != nil || !validDigest(provided) {
		return fmt.Errorf("%s is invalid", field)
	}
	computed, err := digestWithoutFields(value, field)
	if err != nil {
		return err
	}
	if provided != computed {
		return fmt.Errorf("%s mismatch", field)
	}
	return nil
}

func stringField(value any, field string) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return "", err
	}
	provided, ok := object[field].(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", field)
	}
	return provided, nil
}

func digestWithoutFields(value any, fields ...string) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return "", err
	}
	for _, field := range fields {
		delete(object, field)
	}
	canonical, err := json.Marshal(object)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func randomUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = raw[6]&0x0f | 0x40
	raw[8] = raw[8]&0x3f | 0x80
	hexValue := hex.EncodeToString(raw[:])
	return hexValue[:8] + "-" + hexValue[8:12] + "-" + hexValue[12:16] + "-" + hexValue[16:20] + "-" + hexValue[20:], nil
}

func validUUID(value string) bool {
	if len(value) != 36 || strings.ToLower(value) != value || value == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return value[14] >= '1' && value[14] <= '8' && strings.Contains("89ab", value[19:20])
}
func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}
func validToken(value string) bool {
	if value == "" || len(value) > 256 {
		return false
	}
	for _, r := range value {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || strings.ContainsRune("._:-", r)) {
			return false
		}
	}
	return true
}
func validVersion(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || strings.ContainsRune(".+-", r)) {
			return false
		}
	}
	return true
}
func validTimestamp(value string) bool {
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	return err == nil && timestamp(parsed) == value
}
func timestamp(value time.Time) string {
	return value.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}
func validDesiredState(value, surface string) bool {
	if surface == "enrollment" {
		return oneOf(value, "enrolled", "unenrolled_preserved")
	}
	return oneOf(value, "native_running", "native_installed_stopped", "absent_stopped")
}
func validMajors(values []uint64) bool {
	if len(values) < 1 || len(values) > 16 {
		return false
	}
	seen := map[uint64]bool{}
	for _, value := range values {
		if value < 1 || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
func validEnumSet(values []string, minCount, maxCount int, allowed ...string) bool {
	if len(values) < minCount || len(values) > maxCount {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] || !oneOf(value, allowed...) {
			return false
		}
		seen[value] = true
	}
	return true
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func equalOptional(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
func equalOptionalUint32(left, right *uint32) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

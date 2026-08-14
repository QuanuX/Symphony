package ssiagclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	ProviderInstallationInventoryProtocol  = "symphony.ssiag.provider-installation-inventory.v1"
	ProviderBindingStatusProtocol          = "symphony.ssiag.provider-binding-status.v1"
	ProviderBindingPlanRequestProtocol     = "symphony.ssiag.provider-binding-plan-request.v1"
	ProviderBindingPlanProtocol            = "symphony.ssiag.provider-binding-plan.v1"
	ProviderBindingApplyRequestProtocol    = "symphony.ssiag.provider-binding-apply-request.v1"
	ProviderBindingRecoveryRequestProtocol = "symphony.ssiag.provider-binding-recovery-request.v1"
	ProviderBindingResultProtocol          = "symphony.ssiag.provider-binding-result.v1"
)

// ProviderInstallation is SSIAG-owned safe inventory evidence. The opaque
// installation ID is the only selector qxctl accepts; qxctl never resolves the
// receipt or executable digests to filesystem paths.
type ProviderInstallation struct {
	InstallationID             string `json:"installation_id"`
	AdapterIdentifier          string `json:"adapter_identifier"`
	AdapterVersion             string `json:"adapter_version"`
	ProviderProtocol           string `json:"provider_protocol"`
	CommandProtocol            string `json:"command_protocol"`
	ReceiptDigest              string `json:"receipt_digest"`
	ExecutableDigest           string `json:"executable_digest"`
	FoundationVersion          string `json:"foundation_version"`
	FoundationReceiptDigest    string `json:"foundation_receipt_digest"`
	FoundationExecutableDigest string `json:"foundation_executable_digest"`
	CompatibilityState         string `json:"compatibility_state"`
	ReasonCode                 string `json:"reason_code"`
}

func (value *ProviderInstallation) UnmarshalJSON(data []byte) error {
	type plain ProviderInstallation
	if err := requireExactFields(data, []string{
		"installation_id", "adapter_identifier", "adapter_version", "provider_protocol",
		"command_protocol", "receipt_digest", "executable_digest", "foundation_version",
		"foundation_receipt_digest", "foundation_executable_digest", "compatibility_state", "reason_code",
	}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

type ProviderInstallationInventory struct {
	Protocol                  string                 `json:"protocol"`
	TOPSID                    string                 `json:"tops_id"`
	Scope                     string                 `json:"scope"`
	ProviderName              string                 `json:"provider_name"`
	ProviderKind              string                 `json:"provider_kind"`
	Installations             []ProviderInstallation `json:"installations"`
	ObservedAt                string                 `json:"observed_at"`
	OperationalAccessEnabled  bool                   `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool                   `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool                   `json:"secret_channel_enabled"`
	ReadOnly                  bool                   `json:"read_only"`
	Canonical                 bool                   `json:"canonical"`
	InventoryDigest           string                 `json:"inventory_digest"`
}

func (value *ProviderInstallationInventory) UnmarshalJSON(data []byte) error {
	type plain ProviderInstallationInventory
	if err := requireExactFields(data, []string{
		"protocol", "tops_id", "scope", "provider_name", "provider_kind", "installations", "observed_at",
		"operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "read_only",
		"canonical", "inventory_digest",
	}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

type ProviderBindingStatus struct {
	Protocol                  string `json:"protocol"`
	TOPSID                    string `json:"tops_id"`
	Scope                     string `json:"scope"`
	ProviderName              string `json:"provider_name"`
	ProviderKind              string `json:"provider_kind"`
	BindingState              string `json:"binding_state"`
	Generation                uint64 `json:"generation"`
	InstallationID            string `json:"installation_id"`
	PreviousInstallationID    string `json:"previous_installation_id"`
	StateDigest               string `json:"state_digest"`
	AttemptState              string `json:"attempt_state"`
	AttemptOperationID        string `json:"attempt_operation_id"`
	AttemptDigest             string `json:"attempt_digest"`
	RecoveryRequired          bool   `json:"recovery_required"`
	ReasonCode                string `json:"reason_code"`
	ObservedAt                string `json:"observed_at"`
	OperationalAccessEnabled  bool   `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool   `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool   `json:"secret_channel_enabled"`
	ReadOnly                  bool   `json:"read_only"`
	CallerClassUsed           bool   `json:"caller_class_used"`
	Canonical                 bool   `json:"canonical"`
	ResultDigest              string `json:"result_digest"`
}

func (value *ProviderBindingStatus) UnmarshalJSON(data []byte) error {
	type plain ProviderBindingStatus
	if err := requireExactFields(data, []string{
		"protocol", "tops_id", "scope", "provider_name", "provider_kind", "binding_state", "generation",
		"installation_id", "previous_installation_id", "state_digest", "attempt_state", "attempt_operation_id",
		"attempt_digest", "recovery_required", "reason_code", "observed_at", "operational_access_enabled",
		"provider_operations_enabled", "secret_channel_enabled", "read_only", "caller_class_used", "canonical",
		"result_digest",
	}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

// ProviderBindingPlanRequest is intentionally envelope-free. Protocol truth
// lives in knowledge/ssiag; the route and exact three-member body select it.
type ProviderBindingPlanRequest struct {
	InstallationID      string `json:"installation_id"`
	ExpectedStateDigest string `json:"expected_state_digest"`
	Reason              string `json:"reason"`
}

type ProviderBindingAction struct {
	ActionID  string   `json:"action_id"`
	Kind      string   `json:"kind"`
	Direction string   `json:"direction"`
	DependsOn []string `json:"depends_on"`
}

func (value *ProviderBindingAction) UnmarshalJSON(data []byte) error {
	type plain ProviderBindingAction
	if err := requireExactFields(data, []string{"action_id", "kind", "direction", "depends_on"}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

type ProviderBindingPlan struct {
	Protocol                  string                  `json:"protocol"`
	PlanID                    string                  `json:"plan_id"`
	TOPSID                    string                  `json:"tops_id"`
	Scope                     string                  `json:"scope"`
	ProviderName              string                  `json:"provider_name"`
	ProviderKind              string                  `json:"provider_kind"`
	DesiredState              string                  `json:"desired_state"`
	InstallationID            string                  `json:"installation_id"`
	ExpectedStateDigest       string                  `json:"expected_state_digest"`
	CurrentStateDigest        string                  `json:"current_state_digest"`
	InventoryDigest           string                  `json:"inventory_digest"`
	Actions                   []ProviderBindingAction `json:"actions"`
	Applicable                bool                    `json:"applicable"`
	Changed                   bool                    `json:"changed"`
	RecoveryRequired          bool                    `json:"recovery_required"`
	Reason                    string                  `json:"reason"`
	ExpiresAt                 string                  `json:"expires_at"`
	OperationalAccessEnabled  bool                    `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool                    `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool                    `json:"secret_channel_enabled"`
	CallerClassUsed           bool                    `json:"caller_class_used"`
	Canonical                 bool                    `json:"canonical"`
	PlanDigest                string                  `json:"plan_digest"`
}

func (value *ProviderBindingPlan) UnmarshalJSON(data []byte) error {
	type plain ProviderBindingPlan
	if err := requireExactFields(data, []string{
		"protocol", "plan_id", "tops_id", "scope", "provider_name", "provider_kind", "desired_state",
		"installation_id", "expected_state_digest", "current_state_digest", "inventory_digest", "actions",
		"applicable", "changed", "recovery_required", "reason", "expires_at", "operational_access_enabled",
		"provider_operations_enabled", "secret_channel_enabled", "caller_class_used", "canonical", "plan_digest",
	}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

type ProviderBindingApplyRequest struct {
	PlanDigest          string `json:"plan_digest"`
	ExpectedStateDigest string `json:"expected_state_digest"`
}

type ProviderBindingRecoveryRequest struct {
	ExpectedStateDigest string `json:"expected_state_digest"`
	Reason              string `json:"reason"`
}

type ProviderBindingResult struct {
	Protocol                  string `json:"protocol"`
	Operation                 string `json:"operation"`
	OperationID               string `json:"operation_id"`
	TOPSID                    string `json:"tops_id"`
	Scope                     string `json:"scope"`
	ProviderName              string `json:"provider_name"`
	ProviderKind              string `json:"provider_kind"`
	BindingState              string `json:"binding_state"`
	Generation                uint64 `json:"generation"`
	InstallationID            string `json:"installation_id"`
	PreviousInstallationID    string `json:"previous_installation_id"`
	StateDigest               string `json:"state_digest"`
	AttemptState              string `json:"attempt_state"`
	AttemptDigest             string `json:"attempt_digest"`
	ReceiptDigest             string `json:"receipt_digest"`
	Changed                   bool   `json:"changed"`
	Recovered                 bool   `json:"recovered"`
	RecoveryRequired          bool   `json:"recovery_required"`
	ReasonCode                string `json:"reason_code"`
	ObservedAt                string `json:"observed_at"`
	OperationalAccessEnabled  bool   `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool   `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool   `json:"secret_channel_enabled"`
	CallerClassUsed           bool   `json:"caller_class_used"`
	Canonical                 bool   `json:"canonical"`
	ResultDigest              string `json:"result_digest"`
}

func (value *ProviderBindingResult) UnmarshalJSON(data []byte) error {
	type plain ProviderBindingResult
	if err := requireExactFields(data, []string{
		"protocol", "operation", "operation_id", "tops_id", "scope", "provider_name", "provider_kind",
		"binding_state", "generation", "installation_id", "previous_installation_id", "state_digest",
		"attempt_state", "attempt_digest", "receipt_digest", "changed", "recovered", "recovery_required",
		"reason_code", "observed_at", "operational_access_enabled", "provider_operations_enabled",
		"secret_channel_enabled", "caller_class_used", "canonical", "result_digest",
	}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

func (c *Client) ProviderInstallations(ctx context.Context, providerName string) (ProviderInstallationInventory, error) {
	var result ProviderInstallationInventory
	if err := ValidateProviderName(providerName); err != nil {
		return result, err
	}
	if err := c.get(ctx, "/v1/provider-installations/"+url.PathEscape(providerName), &result); err != nil {
		return result, err
	}
	if err := validateProviderInstallationInventory(result, providerName); err != nil {
		return ProviderInstallationInventory{}, err
	}
	return result, nil
}

func (c *Client) ProviderBindingStatus(ctx context.Context, providerName string) (ProviderBindingStatus, error) {
	var result ProviderBindingStatus
	if err := ValidateProviderName(providerName); err != nil {
		return result, err
	}
	if err := c.get(ctx, "/v1/provider-bindings/"+url.PathEscape(providerName), &result); err != nil {
		return result, err
	}
	if err := validateProviderBindingStatus(result, providerName); err != nil {
		return ProviderBindingStatus{}, err
	}
	return result, nil
}

func (c *Client) PlanProviderBinding(ctx context.Context, providerName string, request ProviderBindingPlanRequest) (ProviderBindingPlan, error) {
	var result ProviderBindingPlan
	if err := ValidateProviderName(providerName); err != nil {
		return result, err
	}
	if !validDigestOrNotApplicable(request.InstallationID) || !validStateDigest(request.ExpectedStateDigest) || !validReason(request.Reason) {
		return result, fmt.Errorf("SSIAG provider binding plan request is invalid")
	}
	path := "/v1/provider-bindings/" + url.PathEscape(providerName) + "/plans"
	if err := c.post(ctx, path, request, &result); err != nil {
		return result, err
	}
	if err := validateProviderBindingPlan(result, providerName, request); err != nil {
		return ProviderBindingPlan{}, err
	}
	return result, nil
}

func (c *Client) ApplyProviderBinding(ctx context.Context, providerName string, request ProviderBindingApplyRequest) (ProviderBindingResult, error) {
	return c.providerBindingMutation(ctx, providerName, "/apply", "apply", request)
}

func (c *Client) ProviderBindingApplyStatus(ctx context.Context, providerName, operationID string) (ProviderBindingResult, error) {
	var result ProviderBindingResult
	if err := ValidateProviderName(providerName); err != nil {
		return result, err
	}
	if !validUUID(operationID) {
		return result, fmt.Errorf("SSIAG provider binding operation ID is invalid")
	}
	path := "/v1/provider-bindings/" + url.PathEscape(providerName) + "/attempts/" + url.PathEscape(operationID)
	if err := c.get(ctx, path, &result); err != nil {
		return result, err
	}
	if err := validateProviderBindingResult(result, "apply-status", providerName, operationID); err != nil {
		return ProviderBindingResult{}, err
	}
	return result, nil
}

func (c *Client) RecoverProviderBinding(ctx context.Context, providerName string, request ProviderBindingRecoveryRequest) (ProviderBindingResult, error) {
	return c.providerBindingMutation(ctx, providerName, "/recover", "recover", request)
}

func (c *Client) providerBindingMutation(ctx context.Context, providerName, suffix, operation string, request any) (ProviderBindingResult, error) {
	var result ProviderBindingResult
	if err := ValidateProviderName(providerName); err != nil {
		return result, err
	}
	switch value := request.(type) {
	case ProviderBindingApplyRequest:
		if !validTaggedDigest(value.PlanDigest) || !validStateDigest(value.ExpectedStateDigest) {
			return result, fmt.Errorf("SSIAG provider binding apply request is invalid")
		}
	case ProviderBindingRecoveryRequest:
		if !validStateDigest(value.ExpectedStateDigest) || !validReason(value.Reason) {
			return result, fmt.Errorf("SSIAG provider binding recovery request is invalid")
		}
	default:
		return result, fmt.Errorf("SSIAG provider binding request type is invalid")
	}
	path := "/v1/provider-bindings/" + url.PathEscape(providerName) + suffix
	if err := c.post(ctx, path, request, &result); err != nil {
		return result, err
	}
	if err := validateProviderBindingResult(result, operation, providerName, ""); err != nil {
		return ProviderBindingResult{}, err
	}
	return result, nil
}

func validateProviderInstallationInventory(result ProviderInstallationInventory, providerName string) error {
	if result.Protocol != ProviderInstallationInventoryProtocol || validateTOPSID(result.TOPSID) != nil ||
		!oneOf(result.Scope, "user", "system") || result.ProviderName != providerName || !validProviderName(result.ProviderName) ||
		!validProviderToken(result.ProviderKind) || result.Installations == nil || len(result.Installations) > 128 ||
		!strictUTCTimestamp(result.ObservedAt) || result.OperationalAccessEnabled || result.ProviderOperationsEnabled ||
		result.SecretChannelEnabled || !result.ReadOnly || result.Canonical || !validTaggedDigest(result.InventoryDigest) {
		return fmt.Errorf("SSIAG provider installation inventory violates the safety contract")
	}
	prior := ""
	for _, installation := range result.Installations {
		if !validTaggedDigest(installation.InstallationID) || installation.InstallationID <= prior ||
			!validProviderToken(installation.AdapterIdentifier) || !validProviderToken(installation.AdapterVersion) ||
			installation.ProviderProtocol != "symphony.ssiag.provider.v1" ||
			installation.CommandProtocol != "symphony.ssiag.provider.control.v1" ||
			!validTaggedDigest(installation.ReceiptDigest) || !validTaggedDigest(installation.ExecutableDigest) ||
			!validProviderToken(installation.FoundationVersion) || !validDigestOrNotApplicable(installation.FoundationReceiptDigest) ||
			!validTaggedDigest(installation.FoundationExecutableDigest) ||
			!oneOf(installation.CompatibilityState, "exact", "incompatible", "unsupported") ||
			!validProviderReasonCode(installation.ReasonCode) {
			return fmt.Errorf("SSIAG provider installation inventory contains invalid evidence")
		}
		prior = installation.InstallationID
		if installation.CompatibilityState == "exact" && !validTaggedDigest(installation.FoundationReceiptDigest) {
			return fmt.Errorf("SSIAG exact provider installation lacks a foundation receipt digest")
		}
	}
	return verifyObjectDigest(result, "inventory_digest", result.InventoryDigest)
}

func validateProviderBindingStatus(result ProviderBindingStatus, providerName string) error {
	if result.Protocol != ProviderBindingStatusProtocol || validateTOPSID(result.TOPSID) != nil ||
		!oneOf(result.Scope, "user", "system") || result.ProviderName != providerName || !validProviderToken(result.ProviderKind) ||
		!validBindingState(result.BindingState) || !validDigestOrNotApplicable(result.InstallationID) ||
		!validDigestOrNotApplicable(result.PreviousInstallationID) || !validStateDigest(result.StateDigest) ||
		!validAttemptState(result.AttemptState) || !validUUIDOrNotApplicable(result.AttemptOperationID) ||
		!validDigestOrNotApplicable(result.AttemptDigest) || !validReasonCode(result.ReasonCode) ||
		!strictUTCTimestamp(result.ObservedAt) || result.OperationalAccessEnabled || result.ProviderOperationsEnabled ||
		result.SecretChannelEnabled || !result.ReadOnly || result.CallerClassUsed || result.Canonical ||
		!validTaggedDigest(result.ResultDigest) {
		return fmt.Errorf("SSIAG provider binding status violates the safety contract")
	}
	if result.AttemptState == "none" && (result.AttemptOperationID != "not_applicable" || result.AttemptDigest != "not_applicable") {
		return fmt.Errorf("SSIAG provider binding status absent attempt is contradictory")
	}
	if result.AttemptState != "none" && !result.RecoveryRequired {
		return fmt.Errorf("SSIAG provider binding status recovery evidence is contradictory")
	}
	if result.BindingState == "unbound" && result.InstallationID != "not_applicable" {
		return fmt.Errorf("SSIAG provider binding unbound status is contradictory")
	}
	if result.BindingState == "unbound" && ((result.Generation == 0 && result.StateDigest != "absent") ||
		(result.Generation > 0 && !validTaggedDigest(result.StateDigest))) {
		return fmt.Errorf("SSIAG provider binding unbound status generation is contradictory")
	}
	if result.BindingState == "bound" && (!validTaggedDigest(result.InstallationID) || !validTaggedDigest(result.StateDigest)) {
		return fmt.Errorf("SSIAG provider binding bound status lacks exact evidence")
	}
	return verifyObjectDigest(result, "result_digest", result.ResultDigest)
}

func validateProviderBindingPlan(result ProviderBindingPlan, providerName string, request ProviderBindingPlanRequest) error {
	if result.Protocol != ProviderBindingPlanProtocol || !validUUID(result.PlanID) || validateTOPSID(result.TOPSID) != nil ||
		!oneOf(result.Scope, "user", "system") || result.ProviderName != providerName || !validProviderToken(result.ProviderKind) ||
		!oneOf(result.DesiredState, "bound", "unbound_preserved") || result.InstallationID != request.InstallationID ||
		result.ExpectedStateDigest != request.ExpectedStateDigest || !validStateDigest(result.CurrentStateDigest) ||
		!validTaggedDigest(result.InventoryDigest) || result.Actions == nil || len(result.Actions) > 32 ||
		!validReason(result.Reason) || result.Reason != request.Reason || !strictUTCTimestamp(result.ExpiresAt) ||
		result.OperationalAccessEnabled || result.ProviderOperationsEnabled || result.SecretChannelEnabled ||
		result.CallerClassUsed || result.Canonical || !validTaggedDigest(result.PlanDigest) {
		return fmt.Errorf("SSIAG provider binding plan violates the safety contract")
	}
	if !validProviderBindingActions(result.Actions) {
		return fmt.Errorf("SSIAG provider binding plan action graph is invalid")
	}
	if !result.Applicable && result.Changed {
		return fmt.Errorf("SSIAG provider binding plan applicability is contradictory")
	}
	if result.DesiredState == "bound" && !validTaggedDigest(result.InstallationID) {
		return fmt.Errorf("SSIAG bound provider plan lacks an exact installation")
	}
	if result.DesiredState == "unbound_preserved" && result.InstallationID != "not_applicable" {
		return fmt.Errorf("SSIAG unbind plan carries an installation")
	}
	if !result.Changed && len(result.Actions) > 1 {
		return fmt.Errorf("SSIAG unchanged provider plan contains mutation actions")
	}
	if !result.Changed && len(result.Actions) == 1 && result.Actions[0].Kind != "retain" {
		return fmt.Errorf("SSIAG unchanged provider plan contains a non-retain action")
	}
	return verifyObjectDigest(result, "plan_digest", result.PlanDigest)
}

func validateProviderBindingResult(result ProviderBindingResult, operation, providerName, operationID string) error {
	if result.Protocol != ProviderBindingResultProtocol || result.Operation != operation || !validUUID(result.OperationID) ||
		(operationID != "" && result.OperationID != operationID) || validateTOPSID(result.TOPSID) != nil ||
		!oneOf(result.Scope, "user", "system") || result.ProviderName != providerName || !validProviderToken(result.ProviderKind) ||
		!validBindingState(result.BindingState) || !validDigestOrNotApplicable(result.InstallationID) ||
		!validDigestOrNotApplicable(result.PreviousInstallationID) || !validStateDigest(result.StateDigest) ||
		!validAttemptState(result.AttemptState) || !validDigestOrNotApplicable(result.AttemptDigest) ||
		!validDigestOrNotApplicable(result.ReceiptDigest) || !validReasonCode(result.ReasonCode) ||
		!strictUTCTimestamp(result.ObservedAt) || result.OperationalAccessEnabled || result.ProviderOperationsEnabled ||
		result.SecretChannelEnabled || result.CallerClassUsed || result.Canonical || !validTaggedDigest(result.ResultDigest) {
		return fmt.Errorf("SSIAG provider binding result violates the safety contract")
	}
	if operation != "recover" && result.Recovered {
		return fmt.Errorf("SSIAG provider binding result has an invalid recovered flag")
	}
	return verifyObjectDigest(result, "result_digest", result.ResultDigest)
}

func verifyObjectDigest(value any, field, provided string) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode SSIAG provider binding evidence: %w", err)
	}
	var object map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return fmt.Errorf("decode SSIAG provider binding evidence: %w", err)
	}
	delete(object, field)
	canonical, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("canonicalize SSIAG provider binding evidence: %w", err)
	}
	digest := sha256Tagged(canonical)
	if provided != digest {
		return fmt.Errorf("SSIAG provider binding evidence digest mismatch")
	}
	return nil
}

func sha256Tagged(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validBindingState(value string) bool {
	return oneOf(value, "unbound", "bound", "recovery_required")
}
func validAttemptState(value string) bool {
	return oneOf(value, "none", "prepared", "candidate_verified", "audited", "committed")
}
func validReasonCode(value string) bool {
	return len(value) <= 256 && strings.HasPrefix(value, "symphony.ssiag.provider.binding.") && validProviderToken(value)
}
func validProviderReasonCode(value string) bool {
	return len(value) <= 256 && strings.HasPrefix(value, "symphony.ssiag.provider.") && validProviderToken(value)
}
func validReason(value string) bool {
	count := utf8.RuneCountInString(value)
	return utf8.ValidString(value) && count >= 1 && count <= 1024 && !strings.ContainsAny(value, "\x00\r\n")
}

func validStateDigest(value string) bool { return value == "absent" || validTaggedDigest(value) }

func validUUIDOrNotApplicable(value string) bool {
	return value == "not_applicable" || validUUID(value)
}

func uniqueProviderTokens(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validProviderToken(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validProviderBindingActions(actions []ProviderBindingAction) bool {
	byID := make(map[string]ProviderBindingAction, len(actions))
	for _, action := range actions {
		if !validProviderToken(action.ActionID) ||
			!oneOf(action.Kind, "retain", "bind", "unbind_preserved") ||
			!oneOf(action.Direction, "none", "forward", "reverse") || action.DependsOn == nil ||
			len(action.DependsOn) > 32 || !uniqueProviderTokens(action.DependsOn) {
			return false
		}
		if _, duplicate := byID[action.ActionID]; duplicate {
			return false
		}
		byID[action.ActionID] = action
	}
	for id, action := range byID {
		for _, dependency := range action.DependsOn {
			if dependency == id {
				return false
			}
			if _, present := byID[dependency]; !present {
				return false
			}
		}
	}
	visiting := make(map[string]bool, len(actions))
	visited := make(map[string]bool, len(actions))
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return false
		}
		if visited[id] {
			return true
		}
		visiting[id] = true
		for _, dependency := range byID[id].DependsOn {
			if !visit(dependency) {
				return false
			}
		}
		visiting[id] = false
		visited[id] = true
		return true
	}
	for id := range byID {
		if !visit(id) {
			return false
		}
	}
	return true
}

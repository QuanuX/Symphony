package provider

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

const (
	ProviderReadinessResultProtocol = "symphony.ssiag.provider-readiness-result.v1"
	ProviderReadinessOperationID    = "engop:symphony:ssiag.provider.readiness.observe"
	AdapterReadinessProtocol        = "symphony.ssiag.provider-readiness-observation.v1"
)

type ReadinessLayer struct {
	State      string `json:"state"`
	Evaluated  bool   `json:"evaluated"`
	ReasonCode string `json:"reason_code"`
}

type AdapterReadinessObservation struct {
	Protocol                     string         `json:"protocol"`
	MetadataOnly                 bool           `json:"metadata_only"`
	StructuralValidation         ReadinessLayer `json:"structural_validation"`
	PolicyMatch                  ReadinessLayer `json:"policy_match"`
	OperationalEligibility       ReadinessLayer `json:"operational_eligibility"`
	AppLikeBundleObserved        bool           `json:"app_like_bundle_observed"`
	ProvisioningProfileFileState string         `json:"provisioning_profile_file_state"`
	StaticSignatureState         string         `json:"static_signature_state"`
	DynamicSignatureState        string         `json:"dynamic_signature_state"`
	SigningIdentifier            string         `json:"signing_identifier"`
	DesignatedRequirementDigest  string         `json:"designated_requirement_digest"`
	PolicyRequirementDigest      string         `json:"policy_requirement_digest"`
	SecuritySessionObserved      bool           `json:"security_session_observed"`
	SecuritySessionRoot          bool           `json:"security_session_root"`
	SecuritySessionGraphical     bool           `json:"security_session_graphical"`
	SecuritySessionTTY           bool           `json:"security_session_tty"`
	SecuritySessionRemote        bool           `json:"security_session_remote"`
	AuthorizationDecisionMade    bool           `json:"authorization_decision_made"`
	OperationalAccessEnabled     bool           `json:"operational_access_enabled"`
	ProviderOperationsEnabled    bool           `json:"provider_operations_enabled"`
	SecretChannelEnabled         bool           `json:"secret_channel_enabled"`
	ReasonCodes                  []string       `json:"reason_codes"`
}

type ReadinessResult struct {
	Protocol                  string                      `json:"protocol"`
	Operation                 string                      `json:"operation"`
	TOPSID                    string                      `json:"tops_id"`
	Scope                     string                      `json:"scope"`
	ProviderName              string                      `json:"provider_name"`
	ProviderKind              string                      `json:"provider_kind"`
	AdapterIdentifier         string                      `json:"adapter_identifier"`
	AdapterVersion            string                      `json:"adapter_version"`
	InstallationDigest        string                      `json:"installation_digest"`
	ExecutableDigest          string                      `json:"executable_digest"`
	ReadinessState            string                      `json:"readiness_state"`
	Observation               AdapterReadinessObservation `json:"observation"`
	OperationalAccessEnabled  bool                        `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool                        `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool                        `json:"secret_channel_enabled"`
	ObservedAt                string                      `json:"observed_at"`
	ReadOnly                  bool                        `json:"read_only"`
	CallerClassUsed           bool                        `json:"caller_class_used"`
	Canonical                 bool                        `json:"canonical"`
	ResultDigest              string                      `json:"result_digest"`
}

type readinessLauncher interface {
	ObserveReadiness(context.Context, ExecutableTrust) (AdapterReadinessObservation, error)
}

func (m *TrustManager) ObserveReadiness(ctx context.Context, providerName string) (ReadinessResult, bool) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ReadinessResult{}, false
	}
	result := ReadinessResult{
		Protocol: ProviderReadinessResultProtocol, Operation: ProviderReadinessOperationID,
		TOPSID: m.layout.TOPSID, Scope: string(m.scope), ProviderName: item.Name, ProviderKind: item.Kind,
		AdapterIdentifier: "not_applicable", AdapterVersion: "not_applicable",
		InstallationDigest: "not_applicable", ExecutableDigest: "not_applicable",
		ReadinessState: "unavailable", Observation: unavailableReadinessObservation("symphony.ssiag.provider.readiness.adapter_unavailable"),
		ObservedAt: timestamp(m.now().UTC().Truncate(time.Second)), ReadOnly: true,
	}
	finish := func() ReadinessResult {
		result.ResultDigest = objectDigest(result, "result_digest")
		return result
	}
	if !item.Enabled {
		result.ReadinessState = "disabled"
		result.Observation = unavailableReadinessObservation("symphony.ssiag.provider.readiness.provider_disabled")
		return finish(), true
	}
	release, admitted := m.acquireVerification(providerName)
	if !admitted {
		result.Observation = unavailableReadinessObservation("symphony.ssiag.provider.readiness.busy")
		return finish(), true
	}
	defer release()
	declaration, state, _ := m.inspect(item)
	if state != "unverified" {
		result.Observation = unavailableReadinessObservation("symphony.ssiag.provider.readiness.installation_mismatch")
		return finish(), true
	}
	result.AdapterIdentifier = declaration.AdapterIdentifier
	result.AdapterVersion = declaration.AdapterVersion
	result.InstallationDigest = declaration.InstallationDigest
	result.ExecutableDigest = declaration.ExecutableDigest
	launcher, supported := m.launcher.(readinessLauncher)
	if !supported {
		result.Observation = unavailableReadinessObservation("symphony.ssiag.provider.readiness.launcher_unsupported")
		return finish(), true
	}
	deadline := m.now().Add(defaultProviderTimeout)
	operationContext, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	observation, err := launcher.ObserveReadiness(operationContext, declaration)
	if err != nil {
		reason := "symphony.ssiag.provider.readiness.observation_failed"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(operationContext.Err(), context.DeadlineExceeded) {
			reason = "symphony.ssiag.provider.readiness.timeout"
		}
		result.Observation = unavailableReadinessObservation(reason)
		return finish(), true
	}
	if err := validateReadinessObservation(observation); err != nil {
		result.Observation = unavailableReadinessObservation("symphony.ssiag.provider.readiness.response_invalid")
		return finish(), true
	}
	result.Observation = observation
	if observation.StructuralValidation.State == "valid" && observation.PolicyMatch.State == "matched" {
		result.ReadinessState = "readiness_proven_operations_disabled"
	} else {
		result.ReadinessState = "not_ready"
	}
	return finish(), true
}

func unavailableReadinessObservation(reason string) AdapterReadinessObservation {
	reasonCodes := []string{reason, "symphony.ssiag.provider.readiness.phase_10b_operational_gate"}
	sort.Strings(reasonCodes)
	return AdapterReadinessObservation{
		Protocol: AdapterReadinessProtocol, MetadataOnly: true,
		StructuralValidation:         ReadinessLayer{State: "unavailable", ReasonCode: reason},
		PolicyMatch:                  ReadinessLayer{State: "not_evaluated", ReasonCode: reason},
		OperationalEligibility:       ReadinessLayer{State: "disabled", ReasonCode: "symphony.ssiag.provider.readiness.phase_10b_operational_gate"},
		ProvisioningProfileFileState: "unavailable", StaticSignatureState: "unavailable", DynamicSignatureState: "unavailable",
		SigningIdentifier: "not_applicable", DesignatedRequirementDigest: "not_applicable", PolicyRequirementDigest: "not_applicable",
		ReasonCodes: reasonCodes,
	}
}

func validateReadinessObservation(value AdapterReadinessObservation) error {
	if value.Protocol != AdapterReadinessProtocol || !value.MetadataOnly || value.AuthorizationDecisionMade ||
		value.OperationalAccessEnabled || value.ProviderOperationsEnabled || value.SecretChannelEnabled ||
		value.OperationalEligibility.State != "disabled" || value.OperationalEligibility.Evaluated {
		return errors.New("provider readiness crossed the metadata-only boundary")
	}
	if !oneOf(value.StructuralValidation.State, "valid", "invalid", "unavailable") ||
		!oneOf(value.PolicyMatch.State, "matched", "mismatch", "not_configured", "invalid", "unavailable") ||
		!oneOf(value.ProvisioningProfileFileState, "absent", "regular_safe", "unsafe", "unavailable") ||
		!oneOf(value.StaticSignatureState, "valid", "invalid", "unavailable") ||
		!oneOf(value.DynamicSignatureState, "valid", "invalid", "unavailable", "not_evaluated") ||
		!validReadinessEvidence(value.SigningIdentifier) || !validReadinessDigest(value.DesignatedRequirementDigest) ||
		!validReadinessDigest(value.PolicyRequirementDigest) || !sortedUnique(value.ReasonCodes) {
		return errors.New("provider readiness observation is invalid")
	}
	if (value.StructuralValidation.State == "valid" || value.StructuralValidation.State == "invalid") && !value.StructuralValidation.Evaluated ||
		(value.PolicyMatch.State == "matched" || value.PolicyMatch.State == "mismatch") && !value.PolicyMatch.Evaluated ||
		(value.PolicyMatch.State == "not_configured" || value.PolicyMatch.State == "invalid") && value.PolicyMatch.Evaluated ||
		(value.StructuralValidation.State != value.StaticSignatureState) {
		return errors.New("provider readiness layer evidence is contradictory")
	}
	for _, layer := range []ReadinessLayer{value.StructuralValidation, value.PolicyMatch, value.OperationalEligibility} {
		if !validReason(layer.ReasonCode) {
			return errors.New("provider readiness layer reason is invalid")
		}
	}
	return nil
}

func validReadinessEvidence(value string) bool {
	return value == "not_applicable" || value == "not_observed" || validToken(value)
}

func validReadinessDigest(value string) bool { return value == "not_applicable" || validDigest(value) }

func validateReadinessShape(payload []byte) error {
	object, err := requireJSONObjectFields(payload, []string{
		"protocol", "metadata_only", "structural_validation", "policy_match", "operational_eligibility",
		"app_like_bundle_observed", "provisioning_profile_file_state", "static_signature_state", "dynamic_signature_state",
		"signing_identifier", "designated_requirement_digest", "policy_requirement_digest", "security_session_observed",
		"security_session_root", "security_session_graphical", "security_session_tty", "security_session_remote",
		"authorization_decision_made", "operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "reason_codes",
	})
	if err != nil {
		return err
	}
	for _, name := range []string{"structural_validation", "policy_match", "operational_eligibility"} {
		if _, err := requireJSONObjectFields(object[name], []string{"state", "evaluated", "reason_code"}); err != nil {
			return err
		}
	}
	var reasons []string
	if err := json.Unmarshal(object["reason_codes"], &reasons); err != nil {
		return err
	}
	return nil
}

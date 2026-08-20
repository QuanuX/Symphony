package ssiagclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	ProviderReadinessObservationRequestProtocol = "symphony.ssiag.provider-readiness-observation-request.v1"
	ProviderReadinessResultProtocol             = "symphony.ssiag.provider-readiness-result.v1"
)

type ProviderReadinessObservationRequest struct {
	Protocol       string `json:"protocol"`
	RequestID      string `json:"request_id"`
	CorrelationID  string `json:"correlation_id"`
	AuthorityBasis string `json:"authority_basis"`
}

type ProviderReadinessLayer struct {
	State      string `json:"state"`
	Evaluated  bool   `json:"evaluated"`
	ReasonCode string `json:"reason_code"`
}

func (value *ProviderReadinessLayer) UnmarshalJSON(data []byte) error {
	type plain ProviderReadinessLayer
	if err := requireExactFields(data, []string{"state", "evaluated", "reason_code"}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

type ProviderReadinessObservation struct {
	Protocol                     string                 `json:"protocol"`
	MetadataOnly                 bool                   `json:"metadata_only"`
	StructuralValidation         ProviderReadinessLayer `json:"structural_validation"`
	PolicyMatch                  ProviderReadinessLayer `json:"policy_match"`
	OperationalEligibility       ProviderReadinessLayer `json:"operational_eligibility"`
	AppLikeBundleObserved        bool                   `json:"app_like_bundle_observed"`
	ProvisioningProfileFileState string                 `json:"provisioning_profile_file_state"`
	StaticSignatureState         string                 `json:"static_signature_state"`
	DynamicSignatureState        string                 `json:"dynamic_signature_state"`
	SigningIdentifier            string                 `json:"signing_identifier"`
	DesignatedRequirementDigest  string                 `json:"designated_requirement_digest"`
	PolicyRequirementDigest      string                 `json:"policy_requirement_digest"`
	SecuritySessionObserved      bool                   `json:"security_session_observed"`
	SecuritySessionRoot          bool                   `json:"security_session_root"`
	SecuritySessionGraphical     bool                   `json:"security_session_graphical"`
	SecuritySessionTTY           bool                   `json:"security_session_tty"`
	SecuritySessionRemote        bool                   `json:"security_session_remote"`
	AuthorizationDecisionMade    bool                   `json:"authorization_decision_made"`
	OperationalAccessEnabled     bool                   `json:"operational_access_enabled"`
	ProviderOperationsEnabled    bool                   `json:"provider_operations_enabled"`
	SecretChannelEnabled         bool                   `json:"secret_channel_enabled"`
	ReasonCodes                  []string               `json:"reason_codes"`
}

func (value *ProviderReadinessObservation) UnmarshalJSON(data []byte) error {
	type plain ProviderReadinessObservation
	if err := requireExactFields(data, []string{
		"protocol", "metadata_only", "structural_validation", "policy_match", "operational_eligibility",
		"app_like_bundle_observed", "provisioning_profile_file_state", "static_signature_state", "dynamic_signature_state",
		"signing_identifier", "designated_requirement_digest", "policy_requirement_digest", "security_session_observed",
		"security_session_root", "security_session_graphical", "security_session_tty", "security_session_remote",
		"authorization_decision_made", "operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "reason_codes",
	}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

type ProviderReadinessResult struct {
	Protocol                  string                       `json:"protocol"`
	Operation                 string                       `json:"operation"`
	TOPSID                    string                       `json:"tops_id"`
	Scope                     string                       `json:"scope"`
	ProviderName              string                       `json:"provider_name"`
	ProviderKind              string                       `json:"provider_kind"`
	AdapterIdentifier         string                       `json:"adapter_identifier"`
	AdapterVersion            string                       `json:"adapter_version"`
	InstallationDigest        string                       `json:"installation_digest"`
	ExecutableDigest          string                       `json:"executable_digest"`
	ReadinessState            string                       `json:"readiness_state"`
	Observation               ProviderReadinessObservation `json:"observation"`
	OperationalAccessEnabled  bool                         `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool                         `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool                         `json:"secret_channel_enabled"`
	ObservedAt                string                       `json:"observed_at"`
	ReadOnly                  bool                         `json:"read_only"`
	CallerClassUsed           bool                         `json:"caller_class_used"`
	Canonical                 bool                         `json:"canonical"`
	ResultDigest              string                       `json:"result_digest"`
}

func (value *ProviderReadinessResult) UnmarshalJSON(data []byte) error {
	type plain ProviderReadinessResult
	if err := requireExactFields(data, []string{
		"protocol", "operation", "tops_id", "scope", "provider_name", "provider_kind", "adapter_identifier",
		"adapter_version", "installation_digest", "executable_digest", "readiness_state", "observation",
		"operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "observed_at",
		"read_only", "caller_class_used", "canonical", "result_digest",
	}); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(value))
}

func (c *Client) ObserveProviderReadiness(ctx context.Context, providerName string, request ProviderReadinessObservationRequest) (ProviderReadinessResult, error) {
	var result ProviderReadinessResult
	if err := ValidateProviderName(providerName); err != nil {
		return result, err
	}
	if request.Protocol != ProviderReadinessObservationRequestProtocol || !validUUID(request.RequestID) ||
		!validUUID(request.CorrelationID) || !oneOf(request.AuthorityBasis, "host_owner", "granted_permission") {
		return result, fmt.Errorf("SSIAG provider readiness observation request is invalid")
	}
	path := "/v1/provider-readiness/" + url.PathEscape(providerName) + "/observations"
	if err := c.post(ctx, path, request, &result); err != nil {
		return result, err
	}
	if err := validateProviderReadinessResult(result, providerName); err != nil {
		return ProviderReadinessResult{}, err
	}
	return result, nil
}

func validateProviderReadinessResult(result ProviderReadinessResult, providerName string) error {
	value := result.Observation
	if result.Protocol != ProviderReadinessResultProtocol || result.Operation != "engop:symphony:ssiag.provider.readiness.observe" ||
		result.ProviderName != providerName || validateTOPSID(result.TOPSID) != nil || !oneOf(result.Scope, "user", "system") ||
		!validProviderToken(result.ProviderKind) || !validProviderValue(result.AdapterIdentifier) || !validProviderValue(result.AdapterVersion) ||
		!validDigestOrNotApplicable(result.InstallationDigest) || !validDigestOrNotApplicable(result.ExecutableDigest) ||
		!oneOf(result.ReadinessState, "disabled", "unavailable", "not_ready", "readiness_proven_operations_disabled") ||
		result.OperationalAccessEnabled || result.ProviderOperationsEnabled || result.SecretChannelEnabled ||
		!strictUTCTimestamp(result.ObservedAt) || !result.ReadOnly || result.CallerClassUsed || result.Canonical ||
		!validTaggedDigest(result.ResultDigest) || value.Protocol != "symphony.ssiag.provider-readiness-observation.v1" ||
		!value.MetadataOnly || value.AuthorizationDecisionMade || value.OperationalAccessEnabled || value.ProviderOperationsEnabled || value.SecretChannelEnabled ||
		!oneOf(value.StructuralValidation.State, "valid", "invalid", "unavailable") ||
		!oneOf(value.PolicyMatch.State, "matched", "mismatch", "not_configured", "invalid", "unavailable") ||
		value.OperationalEligibility.State != "disabled" || value.OperationalEligibility.Evaluated ||
		len(value.ReasonCodes) < 2 || len(value.ReasonCodes) > 32 ||
		!sort.StringsAreSorted(value.ReasonCodes) || !validReadinessReasons(value.ReasonCodes) {
		return fmt.Errorf("SSIAG provider readiness result identity or safety boundary is invalid")
	}
	proven := value.StructuralValidation.State == "valid" && value.PolicyMatch.State == "matched"
	if (result.ReadinessState == "readiness_proven_operations_disabled") != proven ||
		(value.StructuralValidation.State == "valid" || value.StructuralValidation.State == "invalid") && !value.StructuralValidation.Evaluated ||
		(value.PolicyMatch.State == "matched" || value.PolicyMatch.State == "mismatch") && !value.PolicyMatch.Evaluated ||
		(value.PolicyMatch.State == "not_configured" || value.PolicyMatch.State == "invalid") && value.PolicyMatch.Evaluated ||
		value.StructuralValidation.State != value.StaticSignatureState {
		return fmt.Errorf("SSIAG provider readiness result contains contradictory layer evidence")
	}
	for _, layer := range []ProviderReadinessLayer{value.StructuralValidation, value.PolicyMatch, value.OperationalEligibility} {
		if !validProviderToken(layer.State) || !validProviderToken(layer.ReasonCode) || !strings.HasPrefix(layer.ReasonCode, "symphony.ssiag.provider.readiness.") {
			return fmt.Errorf("SSIAG provider readiness layer is invalid")
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return err
	}
	delete(object, "result_digest")
	canonical, _ := json.Marshal(object)
	digest := sha256.Sum256(canonical)
	if result.ResultDigest != "sha256:"+hex.EncodeToString(digest[:]) {
		return fmt.Errorf("SSIAG provider readiness result digest mismatch")
	}
	return nil
}

func validReadinessReasons(values []string) bool {
	prior := ""
	for _, value := range values {
		if value == prior || !strings.HasPrefix(value, "symphony.ssiag.provider.readiness.") || !validProviderToken(value) {
			return false
		}
		prior = value
	}
	return true
}

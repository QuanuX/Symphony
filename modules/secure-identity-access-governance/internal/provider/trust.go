package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const (
	ProviderTrustResultProtocol            = "symphony.ssiag.provider-trust-result.v1"
	ControlRequestProtocol                 = "symphony.ssiag.provider-control-request.v1"
	ControlResponseProtocol                = "symphony.ssiag.provider-control-response.v1"
	ProviderProtocol                       = "symphony.ssiag.provider.v1"
	ProviderControlProtocol                = "symphony.ssiag.provider.control.v1"
	MacOSKeychainAdapterID                 = "adapter:symphony:ssiag.macos-keychain-provider.v1"
	ProviderTrustShowOperationID           = "engop:symphony:ssiag.provider.trust.show"
	ProviderTrustVerifyOperationID         = "engop:symphony:ssiag.provider.trust.verify"
	defaultProviderTimeout                 = 5 * time.Second
	maximumControlBytes                    = 64 << 10
	maximumProviderExecutableBytes         = 512 << 20
	maximumConcurrentProviderVerifications = 4
)

type TrustCheck struct {
	CheckID    string `json:"check_id"`
	Outcome    string `json:"outcome"`
	ReasonCode string `json:"reason_code"`
}

type MutualTrust struct {
	FoundationVerifiedAdapter bool `json:"foundation_verified_adapter"`
	AdapterVerifiedFoundation bool `json:"adapter_verified_foundation"`
}

type TrustResult struct {
	Protocol                  string       `json:"protocol"`
	Operation                 string       `json:"operation"`
	TOPSID                    string       `json:"tops_id"`
	ProviderName              string       `json:"provider_name"`
	ProviderKind              string       `json:"provider_kind"`
	DeclarationState          string       `json:"declaration_state"`
	TrustState                string       `json:"trust_state"`
	VerificationMode          string       `json:"verification_mode"`
	AdapterIdentifier         string       `json:"adapter_identifier"`
	AdapterVersion            string       `json:"adapter_version"`
	ProviderProtocol          string       `json:"provider_protocol"`
	Capabilities              []string     `json:"capabilities"`
	Exportable                bool         `json:"exportable"`
	Interactive               bool         `json:"interactive"`
	InstallationDigest        string       `json:"installation_digest"`
	ExecutableDigest          string       `json:"executable_digest"`
	Checks                    []TrustCheck `json:"checks"`
	MutualTrust               MutualTrust  `json:"mutual_trust"`
	OperationalAccessEnabled  bool         `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool         `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool         `json:"secret_channel_enabled"`
	ObservedAt                string       `json:"observed_at"`
	ReadOnly                  bool         `json:"read_only"`
	CallerClassUsed           bool         `json:"caller_class_used"`
	Canonical                 bool         `json:"canonical"`
	ResultDigest              string       `json:"result_digest"`
}

type ExecutableTrust struct {
	Protocol                     string `json:"protocol"`
	TOPSID                       string `json:"tops_id"`
	Scope                        string `json:"scope"`
	ProviderName                 string `json:"provider_name"`
	ProviderKind                 string `json:"provider_kind"`
	AdapterIdentifier            string `json:"adapter_identifier"`
	AdapterVersion               string `json:"adapter_version"`
	ProviderProtocol             string `json:"provider_protocol"`
	ExecutablePath               string `json:"executable_path"`
	InstallationDigest           string `json:"installation_digest"`
	ExecutableDigest             string `json:"executable_digest"`
	OwnerUID                     uint32 `json:"owner_uid"`
	OwnerGID                     uint32 `json:"owner_gid"`
	FileMode                     string `json:"file_mode"`
	AdapterSigningIdentity       string `json:"adapter_signing_identity"`
	FoundationExecutablePath     string `json:"foundation_executable_path"`
	FoundationInstallationDigest string `json:"foundation_installation_digest"`
	FoundationExecutableDigest   string `json:"foundation_executable_digest"`
	FoundationOwnerUID           uint32 `json:"foundation_owner_uid"`
	FoundationOwnerGID           uint32 `json:"foundation_owner_gid"`
	FoundationSigningIdentity    string `json:"foundation_signing_identity"`
	OperationalAccessEnabled     bool   `json:"operational_access_enabled"`
	ProviderOperationsEnabled    bool   `json:"provider_operations_enabled"`
	SecretChannelEnabled         bool   `json:"secret_channel_enabled"`
	DeclarationDigest            string `json:"declaration_digest"`
}

type ControlRequest struct {
	Protocol                     string `json:"protocol"`
	RequestID                    string `json:"request_id"`
	CorrelationID                string `json:"correlation_id"`
	TOPSID                       string `json:"tops_id"`
	ProviderName                 string `json:"provider_name"`
	AdapterIdentifier            string `json:"adapter_identifier"`
	Operation                    string `json:"operation"`
	RequestedAt                  string `json:"requested_at"`
	DeadlineAt                   string `json:"deadline_at"`
	TimeoutMilliseconds          int64  `json:"timeout_milliseconds"`
	FoundationExecutablePath     string `json:"foundation_executable_path"`
	FoundationInstallationDigest string `json:"foundation_installation_digest"`
	FoundationExecutableDigest   string `json:"foundation_executable_digest"`
	FoundationSigningIdentity    string `json:"foundation_signing_identity"`
	OperationalAccessRequested   bool   `json:"operational_access_requested"`
	ProviderOperationRequested   bool   `json:"provider_operation_requested"`
	SecretChannelRequested       bool   `json:"secret_channel_requested"`
	RequestDigest                string `json:"request_digest"`
}

type FoundationTrust struct {
	Verified           bool   `json:"verified"`
	ExecutablePath     string `json:"executable_path"`
	InstallationDigest string `json:"installation_digest"`
	ExecutableDigest   string `json:"executable_digest"`
	SigningIdentity    string `json:"signing_identity"`
	ReasonCode         string `json:"reason_code"`
}

type HandshakeLimits struct {
	MaximumControlRequestBytes  int `json:"maximum_control_request_bytes"`
	MaximumControlResponseBytes int `json:"maximum_control_response_bytes"`
	DefaultTimeoutMilliseconds  int `json:"default_timeout_milliseconds"`
	MaximumTimeoutMilliseconds  int `json:"maximum_timeout_milliseconds"`
	MaximumCapabilities         int `json:"maximum_capabilities"`
	MaximumChecks               int `json:"maximum_checks"`
	RequestsPerProcess          int `json:"requests_per_process"`
	ResponsesPerProcess         int `json:"responses_per_process"`
}

type Handshake struct {
	Protocol                  string          `json:"protocol"`
	ProviderProtocol          string          `json:"provider_protocol"`
	ProviderName              string          `json:"provider_name"`
	ProviderKind              string          `json:"provider_kind"`
	AdapterIdentifier         string          `json:"adapter_identifier"`
	AdapterVersion            string          `json:"adapter_version"`
	Platform                  string          `json:"platform"`
	Architecture              string          `json:"architecture"`
	Transport                 string          `json:"transport"`
	ControlRequestProtocol    string          `json:"control_request_protocol"`
	ControlResponseProtocol   string          `json:"control_response_protocol"`
	OneShotChannelProtocol    string          `json:"one_shot_channel_protocol"`
	Status                    string          `json:"status"`
	ReasonCode                string          `json:"reason_code"`
	Capabilities              []string        `json:"capabilities"`
	Exportable                bool            `json:"exportable"`
	Interactive               bool            `json:"interactive"`
	SafeOperations            []string        `json:"safe_operations"`
	Limits                    HandshakeLimits `json:"limits"`
	FoundationTrust           FoundationTrust `json:"foundation_trust"`
	OperationalAccessEnabled  bool            `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool            `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool            `json:"secret_channel_enabled"`
	HandshakeDigest           string          `json:"handshake_digest"`
}

type SafeError struct {
	Code                   string `json:"code"`
	Category               string `json:"category"`
	Retryable              bool   `json:"retryable"`
	NativeDetailIncluded   bool   `json:"native_detail_included"`
	SecretMaterialIncluded bool   `json:"secret_material_included"`
}

type ControlResponse struct {
	Protocol                  string     `json:"protocol"`
	RequestID                 string     `json:"request_id"`
	CorrelationID             string     `json:"correlation_id"`
	TOPSID                    string     `json:"tops_id"`
	ProviderName              string     `json:"provider_name"`
	AdapterIdentifier         string     `json:"adapter_identifier"`
	Operation                 string     `json:"operation"`
	DeadlineAt                string     `json:"deadline_at"`
	Outcome                   string     `json:"outcome"`
	Status                    string     `json:"status"`
	ReasonCode                string     `json:"reason_code"`
	Handshake                 *Handshake `json:"handshake"`
	Capabilities              []string   `json:"capabilities"`
	Error                     *SafeError `json:"error"`
	OperationalAccessEnabled  bool       `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool       `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool       `json:"secret_channel_enabled"`
	CompletedAt               string     `json:"completed_at"`
	ResponseDigest            string     `json:"response_digest"`
}

type Launcher interface {
	Exchange(context.Context, ExecutableTrust, ControlRequest) (ControlResponse, error)
}

type TrustManager struct {
	scope             ssiagpaths.Scope
	layout            ssiagpaths.InstanceLayout
	registry          *Registry
	launcher          Launcher
	now               func() time.Time
	foundation        foundationEvidence
	verificationSlots chan struct{}
	verificationMu    sync.Mutex
	providerSlots     map[string]chan struct{}
	bindings          *BindingManager
}

type foundationEvidence struct {
	ExecutablePath, InstallationDigest, ExecutableDigest, SigningIdentity string
	OwnerUID, OwnerGID                                                    uint32
}

func NewTrustManager(scope ssiagpaths.Scope, layout ssiagpaths.InstanceLayout, registry *Registry) (*TrustManager, error) {
	evidence, err := observeFoundation()
	if err != nil {
		return nil, fmt.Errorf("observe SSIAG foundation executable: %w", err)
	}
	return newTrustManager(scope, layout, registry, processLauncher{}, time.Now, evidence)
}

func newTrustManager(scope ssiagpaths.Scope, layout ssiagpaths.InstanceLayout, registry *Registry, launcher Launcher, now func() time.Time, foundation foundationEvidence) (*TrustManager, error) {
	if registry == nil || launcher == nil || now == nil {
		return nil, fmt.Errorf("provider trust manager dependencies are required")
	}
	if layout.Scope != scope || layout.ProviderTrustDir == "" {
		return nil, fmt.Errorf("provider trust layout does not match scope")
	}
	return &TrustManager{
		scope: scope, layout: layout, registry: registry, launcher: launcher, now: now, foundation: foundation,
		verificationSlots: make(chan struct{}, maximumConcurrentProviderVerifications),
		providerSlots:     make(map[string]chan struct{}),
	}, nil
}

func (m *TrustManager) Show(providerName string) (TrustResult, bool) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return TrustResult{}, false
	}
	result := m.baseResult(ProviderTrustShowOperationID, "snapshot", item)
	if !item.Enabled {
		result.TrustState = "disabled"
		result.Checks = []TrustCheck{{"declaration.enabled", "not_applicable", "symphony.ssiag.provider.disabled"}}
		return finishResult(result), true
	}
	declaration, state, reason := m.inspect(item)
	applyDeclaration(&result, declaration)
	result.TrustState = state
	result.Checks = staticChecks(state, reason)
	return finishResult(result), true
}

func (m *TrustManager) acquireVerification(providerName string) (func(), bool) {
	select {
	case m.verificationSlots <- struct{}{}:
	default:
		return func() {}, false
	}
	m.verificationMu.Lock()
	slot := m.providerSlots[providerName]
	if slot == nil {
		slot = make(chan struct{}, 1)
		m.providerSlots[providerName] = slot
	}
	m.verificationMu.Unlock()
	select {
	case slot <- struct{}{}:
		return func() {
			<-slot
			<-m.verificationSlots
		}, true
	default:
		<-m.verificationSlots
		return func() {}, false
	}
}

func (m *TrustManager) Verify(ctx context.Context, providerName, requestID, correlationID string) (TrustResult, bool) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return TrustResult{}, false
	}
	result := m.baseResult(ProviderTrustVerifyOperationID, "fresh", item)
	if !item.Enabled {
		result.TrustState = "disabled"
		result.Checks = []TrustCheck{{"declaration.enabled", "not_applicable", "symphony.ssiag.provider.disabled"}}
		return finishResult(result), true
	}
	release, admitted := m.acquireVerification(providerName)
	if !admitted {
		result.TrustState = "unavailable"
		result.Checks = []TrustCheck{{"handshake.admission", "failed", "symphony.ssiag.provider.busy"}}
		return finishResult(result), true
	}
	defer release()
	declaration, state, reason := m.inspect(item)
	applyDeclaration(&result, declaration)
	if state != "unverified" {
		result.TrustState = state
		result.Checks = staticChecks(state, reason)
		return finishResult(result), true
	}
	return m.verifyPrepared(ctx, item, declaration, requestID, correlationID, result), true
}

func (m *TrustManager) verifyDeclaration(ctx context.Context, item config.ProviderConfig, declaration ExecutableTrust, requestID, correlationID string) TrustResult {
	result := m.baseResult(ProviderTrustVerifyOperationID, "fresh", item)
	applyDeclaration(&result, declaration)
	if err := validateExecutableTrust(declaration, item, m.layout, m.foundation); err != nil {
		result.TrustState = "mismatch"
		result.Checks = staticChecks("mismatch", "symphony.ssiag.provider.declaration_mismatch")
		return finishResult(result)
	}
	if err := validateAdapterReceipt(declaration, m.scope); err != nil {
		result.TrustState = "mismatch"
		result.Checks = staticChecks("mismatch", "symphony.ssiag.provider.installation_mismatch")
		return finishResult(result)
	}
	release, admitted := m.acquireVerification(item.Name)
	if !admitted {
		result.TrustState = "unavailable"
		result.Checks = []TrustCheck{{"handshake.admission", "failed", "symphony.ssiag.provider.busy"}}
		return finishResult(result)
	}
	defer release()
	return m.verifyPrepared(ctx, item, declaration, requestID, correlationID, result)
}

func (m *TrustManager) verifyPrepared(ctx context.Context, item config.ProviderConfig, declaration ExecutableTrust, requestID, correlationID string, result TrustResult) TrustResult {
	now := m.now().UTC().Truncate(time.Second)
	deadline := now.Add(defaultProviderTimeout)
	request := ControlRequest{
		Protocol: ControlRequestProtocol, RequestID: requestID, CorrelationID: correlationID,
		TOPSID: m.layout.TOPSID, ProviderName: item.Name, AdapterIdentifier: declaration.AdapterIdentifier,
		Operation: "handshake", RequestedAt: timestamp(now), DeadlineAt: timestamp(deadline),
		TimeoutMilliseconds:          defaultProviderTimeout.Milliseconds(),
		FoundationExecutablePath:     m.foundation.ExecutablePath,
		FoundationInstallationDigest: m.foundation.InstallationDigest,
		FoundationExecutableDigest:   m.foundation.ExecutableDigest,
		FoundationSigningIdentity:    m.foundation.SigningIdentity,
	}
	request.RequestDigest = objectDigest(request, "request_digest")
	exchangeContext, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	response, err := m.launcher.Exchange(exchangeContext, declaration, request)
	if err != nil {
		result.TrustState = "unavailable"
		reason := "symphony.ssiag.provider.handshake_failed"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(exchangeContext.Err(), context.DeadlineExceeded) {
			reason = "symphony.ssiag.provider.timeout"
		}
		result.Checks = append(staticChecks("unverified", "symphony.ssiag.provider.static_trust_verified"), TrustCheck{"handshake.response", "failed", reason})
		return finishResult(result)
	}
	if err := validateResponse(response, request, item, declaration, m.foundation); err != nil {
		result.TrustState = "mismatch"
		result.Checks = append(staticChecks("unverified", "symphony.ssiag.provider.static_trust_verified"), TrustCheck{"handshake.response", "failed", "symphony.ssiag.provider.handshake_mismatch"})
		return finishResult(result)
	}
	result.TrustState = "verified"
	result.MutualTrust = MutualTrust{true, true}
	result.Capabilities = append([]string(nil), response.Handshake.Capabilities...)
	result.Checks = []TrustCheck{
		{"adapter.executable", "passed", "symphony.ssiag.provider.executable_verified"},
		{"adapter.receipt", "passed", "symphony.ssiag.provider.installation_verified"},
		{"foundation.executable", "passed", "symphony.ssiag.provider.foundation_verified"},
		{"handshake.response", "passed", "symphony.ssiag.provider.handshake_verified"},
	}
	sort.Slice(result.Checks, func(i, j int) bool { return result.Checks[i].CheckID < result.Checks[j].CheckID })
	return finishResult(result)
}

func (m *TrustManager) baseResult(operationID, mode string, item config.ProviderConfig) TrustResult {
	operation := ""
	switch operationID {
	case ProviderTrustShowOperationID:
		operation = "show"
	case ProviderTrustVerifyOperationID:
		operation = "verify"
	default:
		panic("unsupported provider trust operation identity")
	}
	declarationState := "declared"
	if !item.Enabled {
		declarationState = "disabled"
	}
	capabilities := append([]string(nil), item.Capabilities...)
	sort.Strings(capabilities)
	return TrustResult{
		Protocol: ProviderTrustResultProtocol, Operation: operation, TOPSID: m.layout.TOPSID,
		ProviderName: item.Name, ProviderKind: item.Kind, DeclarationState: declarationState,
		TrustState: "unbound", VerificationMode: mode, AdapterIdentifier: "not_applicable",
		AdapterVersion: "not_applicable", ProviderProtocol: "not_applicable", Capabilities: capabilities,
		Exportable: item.Exportable, Interactive: item.Interactive, InstallationDigest: "not_applicable",
		ExecutableDigest: "not_applicable", Checks: []TrustCheck{}, ObservedAt: timestamp(m.now()), ReadOnly: true,
	}
}

func (m *TrustManager) inspect(item config.ProviderConfig) (ExecutableTrust, string, string) {
	if m.bindings != nil {
		declaration, managed, err := m.bindings.ActiveDeclaration(item.Name)
		if err != nil {
			return declaration, "mismatch", "symphony.ssiag.provider.binding_mismatch"
		}
		if managed {
			if declaration.AdapterIdentifier == "" {
				return ExecutableTrust{}, "unbound", "symphony.ssiag.provider.binding_unbound"
			}
			if err := validateExecutableTrust(declaration, item, m.layout, m.foundation); err != nil {
				return declaration, "mismatch", "symphony.ssiag.provider.declaration_mismatch"
			}
			if err := validateAdapterReceipt(declaration, m.scope); err != nil {
				return declaration, "mismatch", "symphony.ssiag.provider.installation_mismatch"
			}
			return declaration, "unverified", "symphony.ssiag.provider.static_trust_verified"
		}
	}
	path := filepath.Join(m.layout.ProviderTrustDir, item.Name+".json")
	declaration, exists, err := loadExecutableTrust(path, m.scope)
	if err != nil {
		return ExecutableTrust{}, "mismatch", "symphony.ssiag.provider.declaration_invalid"
	}
	if !exists {
		return ExecutableTrust{}, "unbound", "symphony.ssiag.provider.declaration_absent"
	}
	if err := validateExecutableTrust(declaration, item, m.layout, m.foundation); err != nil {
		return declaration, "mismatch", "symphony.ssiag.provider.declaration_mismatch"
	}
	if err := validateAdapterReceipt(declaration, m.scope); err != nil {
		return declaration, "mismatch", "symphony.ssiag.provider.installation_mismatch"
	}
	return declaration, "unverified", "symphony.ssiag.provider.static_trust_verified"
}

func applyDeclaration(result *TrustResult, declaration ExecutableTrust) {
	if declaration.AdapterIdentifier == "" {
		return
	}
	result.AdapterIdentifier = declaration.AdapterIdentifier
	result.AdapterVersion = declaration.AdapterVersion
	result.ProviderProtocol = declaration.ProviderProtocol
	result.InstallationDigest = declaration.InstallationDigest
	result.ExecutableDigest = declaration.ExecutableDigest
}

func staticChecks(state, reason string) []TrustCheck {
	outcome := "failed"
	if state == "unverified" {
		outcome = "passed"
	}
	if state == "unbound" {
		outcome = "not_applicable"
	}
	return []TrustCheck{{"adapter.static_trust", outcome, reason}}
}

func finishResult(result TrustResult) TrustResult {
	if result.ObservedAt == "" {
		result.ObservedAt = timestamp(time.Now())
	}
	result.ResultDigest = objectDigest(result, "result_digest")
	return result
}

func timestamp(value time.Time) string {
	return value.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}

func objectDigest(value any, omit string) string {
	encoded, _ := json.Marshal(value)
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	_ = decoder.Decode(&object)
	delete(object, omit)
	canonical, _ := json.Marshal(object)
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func loadExecutableTrust(path string, scope ssiagpaths.Scope) (ExecutableTrust, bool, error) {
	file, err := openProviderFile(path)
	if os.IsNotExist(err) {
		return ExecutableTrust{}, false, nil
	}
	if err != nil {
		return ExecutableTrust{}, false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumControlBytes {
		return ExecutableTrust{}, false, fmt.Errorf("unsafe provider trust declaration")
	}
	uid, _, err := fileOwner(info)
	if err != nil {
		return ExecutableTrust{}, false, err
	}
	if scope == ssiagpaths.ScopeUser && (uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o077 != 0) {
		return ExecutableTrust{}, false, fmt.Errorf("unsafe user provider trust declaration")
	}
	if scope == ssiagpaths.ScopeSystem && (uid != 0 || info.Mode().Perm()&0o022 != 0) {
		return ExecutableTrust{}, false, fmt.Errorf("unsafe system provider trust declaration")
	}
	payload, err := io.ReadAll(io.LimitReader(file, maximumControlBytes+1))
	if err != nil || len(payload) > maximumControlBytes || validateJSONMembers(payload) != nil {
		return ExecutableTrust{}, false, fmt.Errorf("invalid provider trust declaration")
	}
	if _, err := requireJSONObjectFields(payload, []string{
		"protocol", "tops_id", "scope", "provider_name", "provider_kind", "adapter_identifier", "adapter_version",
		"provider_protocol", "executable_path", "installation_digest", "executable_digest", "owner_uid", "owner_gid",
		"file_mode", "adapter_signing_identity", "foundation_executable_path", "foundation_installation_digest",
		"foundation_executable_digest", "foundation_owner_uid", "foundation_owner_gid", "foundation_signing_identity",
		"operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "declaration_digest",
	}); err != nil {
		return ExecutableTrust{}, false, fmt.Errorf("invalid provider trust declaration")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var declaration ExecutableTrust
	if err := decoder.Decode(&declaration); err != nil {
		return ExecutableTrust{}, false, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ExecutableTrust{}, false, fmt.Errorf("multiple declaration values")
	}
	if declaration.DeclarationDigest != objectDigest(declaration, "declaration_digest") {
		return ExecutableTrust{}, false, fmt.Errorf("provider trust declaration digest mismatch")
	}
	return declaration, true, nil
}

func validateExecutableTrust(value ExecutableTrust, item config.ProviderConfig, layout ssiagpaths.InstanceLayout, foundation foundationEvidence) error {
	if value.Protocol != "symphony.ssiag.provider-executable-trust.v1" || value.TOPSID != layout.TOPSID || value.Scope != string(layout.Scope) || value.ProviderName != item.Name ||
		value.ProviderKind != item.Kind || value.AdapterIdentifier != MacOSKeychainAdapterID || !validToken(value.AdapterVersion) ||
		value.ProviderProtocol != ProviderProtocol || !filepath.IsAbs(value.ExecutablePath) || filepath.Clean(value.ExecutablePath) != value.ExecutablePath ||
		!validDigest(value.InstallationDigest) || !validDigest(value.ExecutableDigest) ||
		value.FoundationExecutablePath != foundation.ExecutablePath || value.FoundationInstallationDigest != foundation.InstallationDigest ||
		value.FoundationExecutableDigest != foundation.ExecutableDigest || value.FoundationOwnerUID != foundation.OwnerUID ||
		value.FoundationOwnerGID != foundation.OwnerGID || value.FoundationSigningIdentity != foundation.SigningIdentity ||
		value.OperationalAccessEnabled || value.ProviderOperationsEnabled || value.SecretChannelEnabled {
		return fmt.Errorf("provider executable trust identity mismatch")
	}
	// Phase 9 binds exact receipt and executable digests. A signing identity is
	// never accepted merely because a declaration names it; the later
	// platform-signing verifier must independently prove it first.
	if value.AdapterSigningIdentity != "not_applicable" {
		return fmt.Errorf("adapter signing identity cannot be proven by this verifier")
	}
	return nil
}

func observeFoundation() (foundationEvidence, error) {
	executable, err := os.Executable()
	if err != nil {
		return foundationEvidence{}, err
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return foundationEvidence{}, err
	}
	info, err := os.Lstat(executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return foundationEvidence{}, fmt.Errorf("foundation executable is unsafe")
	}
	uid, gid, err := fileOwner(info)
	if err != nil {
		return foundationEvidence{}, err
	}
	digest, err := digestPath(executable)
	if err != nil {
		return foundationEvidence{}, err
	}
	installation := "not_applicable"
	if evidence, installed, inspectErr := packageinstall.InspectExecutable(executable); inspectErr != nil {
		return foundationEvidence{}, inspectErr
	} else if installed {
		installation = evidence.ReceiptDigest
		if digest != evidence.BinaryDigest {
			return foundationEvidence{}, fmt.Errorf("foundation receipt digest mismatch")
		}
	}
	return foundationEvidence{ExecutablePath: executable, InstallationDigest: installation, ExecutableDigest: digest, SigningIdentity: "not_applicable", OwnerUID: uid, OwnerGID: gid}, nil
}

func validateResponse(response ControlResponse, request ControlRequest, item config.ProviderConfig, declaration ExecutableTrust, foundation foundationEvidence) error {
	if response.Protocol != ControlResponseProtocol || response.RequestID != request.RequestID || response.CorrelationID != request.CorrelationID ||
		response.TOPSID != request.TOPSID || response.ProviderName != item.Name || response.AdapterIdentifier != declaration.AdapterIdentifier ||
		response.Operation != "handshake" || response.DeadlineAt != request.DeadlineAt || response.Outcome != "succeeded" ||
		!oneOf(response.Status, "declared", "ready", "degraded", "locked", "unavailable", "disabled") || !validReason(response.ReasonCode) || response.Error != nil ||
		response.Handshake == nil || response.OperationalAccessEnabled || response.ProviderOperationsEnabled || response.SecretChannelEnabled ||
		response.ResponseDigest != objectDigest(response, "response_digest") {
		return fmt.Errorf("provider control response binding mismatch")
	}
	requestedAt, requestedOK := strictTimestamp(request.RequestedAt)
	completedAt, completedOK := strictTimestamp(response.CompletedAt)
	deadlineAt, deadlineOK := strictTimestamp(request.DeadlineAt)
	if !requestedOK || !completedOK || !deadlineOK || completedAt.Before(requestedAt) || completedAt.After(deadlineAt) {
		return fmt.Errorf("provider control response time binding mismatch")
	}
	h := *response.Handshake
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	if h.Protocol != "symphony.ssiag.provider-handshake.v1" || h.ProviderProtocol != ProviderProtocol || h.ProviderName != item.Name ||
		h.ProviderKind != item.Kind || h.AdapterIdentifier != declaration.AdapterIdentifier || h.AdapterVersion != declaration.AdapterVersion ||
		h.Platform != platform || h.Architecture != runtime.GOARCH || !validToken(h.Architecture) || h.Transport != "stdio-one-shot-json" ||
		h.ControlRequestProtocol != ControlRequestProtocol || h.ControlResponseProtocol != ControlResponseProtocol ||
		h.OneShotChannelProtocol != "symphony.ssiag.provider-one-shot-channel.v1" || !oneOf(h.Status, "declared", "ready", "degraded", "locked", "unavailable", "disabled") ||
		h.Status != response.Status || h.ReasonCode != response.ReasonCode || !validReason(h.ReasonCode) || !validReason(h.FoundationTrust.ReasonCode) || h.OperationalAccessEnabled ||
		h.ProviderOperationsEnabled || h.SecretChannelEnabled || h.HandshakeDigest != objectDigest(h, "handshake_digest") ||
		!h.FoundationTrust.Verified || h.FoundationTrust.ExecutablePath != foundation.ExecutablePath ||
		h.FoundationTrust.InstallationDigest != foundation.InstallationDigest || h.FoundationTrust.ExecutableDigest != foundation.ExecutableDigest ||
		h.FoundationTrust.SigningIdentity != foundation.SigningIdentity || h.Exportable != item.Exportable || h.Interactive != item.Interactive ||
		!sortedUnique(h.Capabilities) || !sortedUnique(response.Capabilities) || !sortedUnique(h.SafeOperations) ||
		!sameStrings(h.Capabilities, item.Capabilities) || !sameStrings(response.Capabilities, h.Capabilities) ||
		!sameStrings(h.SafeOperations, []string{"capabilities", "handshake", "status"}) ||
		h.Limits.MaximumControlRequestBytes != maximumControlBytes || h.Limits.MaximumControlResponseBytes != maximumControlBytes ||
		h.Limits.DefaultTimeoutMilliseconds != 5000 || h.Limits.MaximumTimeoutMilliseconds != 30000 ||
		h.Limits.MaximumCapabilities != 128 || h.Limits.MaximumChecks != 128 || h.Limits.RequestsPerProcess != 1 || h.Limits.ResponsesPerProcess != 1 {
		return fmt.Errorf("provider handshake mismatch")
	}
	return nil
}

func validateControlResponseShape(payload []byte) error {
	object, err := requireJSONObjectFields(payload, []string{
		"protocol", "request_id", "correlation_id", "tops_id", "provider_name", "adapter_identifier", "operation",
		"deadline_at", "outcome", "status", "reason_code", "handshake", "capabilities", "error",
		"operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "completed_at", "response_digest",
	})
	if err != nil {
		return err
	}
	if bytes.Equal(bytes.TrimSpace(object["handshake"]), []byte("null")) {
		return nil
	}
	handshake, err := requireJSONObjectFields(object["handshake"], []string{
		"protocol", "provider_protocol", "provider_name", "provider_kind", "adapter_identifier", "adapter_version", "platform",
		"architecture", "transport", "control_request_protocol", "control_response_protocol", "one_shot_channel_protocol", "status",
		"reason_code", "foundation_trust", "capabilities", "exportable", "interactive", "safe_operations", "limits",
		"operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "handshake_digest",
	})
	if err != nil {
		return err
	}
	if _, err := requireJSONObjectFields(handshake["foundation_trust"], []string{
		"verified", "executable_path", "installation_digest", "executable_digest", "signing_identity", "reason_code",
	}); err != nil {
		return err
	}
	_, err = requireJSONObjectFields(handshake["limits"], []string{
		"maximum_control_request_bytes", "maximum_control_response_bytes", "default_timeout_milliseconds",
		"maximum_timeout_milliseconds", "maximum_capabilities", "maximum_checks", "requests_per_process", "responses_per_process",
	})
	return err
}

func requireJSONObjectFields(payload []byte, required []string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&object); err != nil || object == nil {
		return nil, fmt.Errorf("JSON value is not an object")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("JSON object has trailing data")
	}
	if len(object) != len(required) {
		return nil, fmt.Errorf("JSON object member count mismatch")
	}
	for _, member := range required {
		if _, present := object[member]; !present {
			return nil, fmt.Errorf("JSON object is missing required member %q", member)
		}
	}
	return object, nil
}

// ValidateStrictJSONObject exposes the foundation's duplicate-aware exact
// member-set validator to its local HTTP boundary without exporting decoded
// provider values or weakening schema ownership.
func ValidateStrictJSONObject(payload []byte, required []string) error {
	if err := validateJSONMembers(payload); err != nil {
		return err
	}
	_, err := requireJSONObjectFields(payload, required)
	return err
}

func strictTimestamp(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	return parsed, err == nil && timestamp(parsed) == value
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func validReason(value string) bool {
	return strings.HasPrefix(value, "symphony.ssiag.provider.") && validToken(value)
}

func sameStrings(left, right []string) bool {
	a, b := append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(a)
	sort.Strings(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] || (i > 0 && a[i] == a[i-1]) {
			return false
		}
	}
	return true
}

func validToken(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && !strings.ContainsRune("._:-", c) {
			return false
		}
	}
	return true
}
func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil && value == strings.ToLower(value)
}

func validateJSONMembers(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("JSON nesting exceeds limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		if delim == '{' {
			seen := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("invalid JSON key")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON member")
				}
				seen[key] = struct{}{}
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		} else if delim == '[' {
			for decoder.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("unexpected delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

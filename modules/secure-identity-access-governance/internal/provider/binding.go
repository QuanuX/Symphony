package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

const (
	ProviderInstallationInventoryProtocol = "symphony.ssiag.provider-installation-inventory.v1"
	ProviderBindingStatusProtocol         = "symphony.ssiag.provider-binding-status.v1"
	ProviderBindingPlanProtocol           = "symphony.ssiag.provider-binding-plan.v1"
	ProviderBindingResultProtocol         = "symphony.ssiag.provider-binding-result.v1"
	providerBindingStateProtocol          = "symphony.ssiag.provider-binding-state.v1"
	providerBindingAttemptProtocol        = "symphony.ssiag.provider-binding-attempt.v1"
	providerBindingPlanLifetime           = 10 * time.Minute
)

var (
	ErrBindingConflict         = errors.New("SSIAG provider binding compare-and-swap conflict")
	ErrBindingRecoveryRequired = errors.New("SSIAG provider binding recovery is required")
	ErrBindingRecoveryAbsent   = errors.New("SSIAG provider binding recovery evidence is absent")
	ErrBindingInstallation     = errors.New("SSIAG provider installation is not an exact compatible candidate")
)

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

type ProviderBindingApplyRequest struct {
	PlanDigest          string `json:"plan_digest"`
	ExpectedStateDigest string `json:"expected_state_digest"`
}

type ProviderBindingRecoveryRequest struct {
	ExpectedStateDigest string `json:"expected_state_digest"`
	Reason              string `json:"reason"`
}

type ProviderBindingAuditIdentity struct {
	ActorID              string `json:"actor_id"`
	ActorKind            string `json:"actor_kind"`
	AuthenticationMethod string `json:"authentication_method"`
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

type ProviderBindingState struct {
	Protocol                  string `json:"protocol"`
	TOPSID                    string `json:"tops_id"`
	Scope                     string `json:"scope"`
	ProviderName              string `json:"provider_name"`
	ProviderKind              string `json:"provider_kind"`
	Generation                uint64 `json:"generation"`
	BindingState              string `json:"binding_state"`
	InstallationID            string `json:"installation_id"`
	PreviousInstallationID    string `json:"previous_installation_id"`
	PreviousStateDigest       string `json:"previous_state_digest"`
	UpdatedAt                 string `json:"updated_at"`
	OperationalAccessEnabled  bool   `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool   `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool   `json:"secret_channel_enabled"`
	StateDigest               string `json:"state_digest"`
}

type ProviderBindingAttempt struct {
	Protocol                  string                       `json:"protocol"`
	OperationID               string                       `json:"operation_id"`
	TOPSID                    string                       `json:"tops_id"`
	Scope                     string                       `json:"scope"`
	ProviderName              string                       `json:"provider_name"`
	ProviderKind              string                       `json:"provider_kind"`
	Plan                      ProviderBindingPlan          `json:"plan"`
	AuditIdentity             ProviderBindingAuditIdentity `json:"audit_identity"`
	Stage                     string                       `json:"stage"`
	PreparedAt                string                       `json:"prepared_at"`
	CandidateVerifiedAt       *string                      `json:"candidate_verified_at"`
	AuditedAt                 *string                      `json:"audited_at"`
	CommittedAt               *string                      `json:"committed_at"`
	ReceiptDigest             string                       `json:"receipt_digest"`
	TargetState               ProviderBindingState         `json:"target_state"`
	OperationalAccessEnabled  bool                         `json:"operational_access_enabled"`
	ProviderOperationsEnabled bool                         `json:"provider_operations_enabled"`
	SecretChannelEnabled      bool                         `json:"secret_channel_enabled"`
	AttemptDigest             string                       `json:"attempt_digest"`
}

type BindingManager struct {
	scope    ssiagpaths.Scope
	layout   ssiagpaths.InstanceLayout
	registry *Registry
	trust    *TrustManager
	now      func() time.Time
	roots    []string
}

type providerBindingAbsence struct {
	Protocol     string `json:"protocol"`
	TOPSID       string `json:"tops_id"`
	Scope        string `json:"scope"`
	ProviderName string `json:"provider_name"`
	ProviderKind string `json:"provider_kind"`
}

func NewBindingManager(scope ssiagpaths.Scope, layout ssiagpaths.InstanceLayout, registry *Registry, trust *TrustManager) (*BindingManager, error) {
	return newBindingManager(scope, layout, registry, trust, time.Now, nil)
}

func newBindingManager(scope ssiagpaths.Scope, layout ssiagpaths.InstanceLayout, registry *Registry, trust *TrustManager, now func() time.Time, roots []string) (*BindingManager, error) {
	if layout.Scope != scope || layout.ProviderBindingDir == "" || registry == nil || trust == nil || now == nil {
		return nil, fmt.Errorf("provider binding manager dependencies are invalid")
	}
	if len(roots) == 0 {
		root := "/usr/local"
		if scope == ssiagpaths.ScopeUser {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			root = filepath.Join(home, ".local")
		}
		roots = []string{root}
	}
	cleanRoots := make([]string, 0, len(roots))
	seen := map[string]bool{}
	for _, root := range roots {
		root = filepath.Clean(root)
		if !filepath.IsAbs(root) || root == string(filepath.Separator) || seen[root] {
			return nil, fmt.Errorf("provider installation root is unsafe or duplicated")
		}
		seen[root] = true
		cleanRoots = append(cleanRoots, root)
	}
	sort.Strings(cleanRoots)
	m := &BindingManager{scope: scope, layout: layout, registry: registry, trust: trust, now: now, roots: cleanRoots}
	trust.bindings = m
	return m, nil
}

func (m *BindingManager) Inventory(providerName string) (ProviderInstallationInventory, bool, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ProviderInstallationInventory{}, false, nil
	}
	installations, _, err := m.discover(item)
	if err != nil {
		return ProviderInstallationInventory{}, true, err
	}
	result := ProviderInstallationInventory{
		Protocol: ProviderInstallationInventoryProtocol, TOPSID: m.layout.TOPSID, Scope: string(m.scope),
		ProviderName: item.Name, ProviderKind: item.Kind, Installations: installations,
		ObservedAt: timestamp(m.now()), ReadOnly: true,
	}
	result.InventoryDigest = objectDigest(result, "inventory_digest")
	return result, true, nil
}

func (m *BindingManager) Status(providerName string) (ProviderBindingStatus, bool, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ProviderBindingStatus{}, false, nil
	}
	var state ProviderBindingState
	var attempt *ProviderBindingAttempt
	err := withBindingStore(m.layout.ProviderBindingDir, providerName, false, func(store *bindingStore) error {
		var err error
		state, attempt, err = m.readSnapshot(store, item)
		return err
	})
	if err != nil {
		return ProviderBindingStatus{}, true, err
	}
	result := m.statusFrom(item, state, attempt)
	return result, true, nil
}

func (m *BindingManager) Plan(providerName string, request ProviderBindingPlanRequest) (ProviderBindingPlan, bool, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ProviderBindingPlan{}, false, nil
	}
	if !validBindingStateDigest(request.ExpectedStateDigest) || !validBindingInstallationID(request.InstallationID) || !validBindingReason(request.Reason) {
		return ProviderBindingPlan{}, true, fmt.Errorf("invalid provider binding plan request")
	}
	installations, declarations, err := m.discover(item)
	if err != nil {
		return ProviderBindingPlan{}, true, err
	}
	inventory := ProviderInstallationInventory{Protocol: ProviderInstallationInventoryProtocol, TOPSID: m.layout.TOPSID, Scope: string(m.scope), ProviderName: item.Name, ProviderKind: item.Kind, Installations: installations, ObservedAt: timestamp(m.now()), ReadOnly: true}
	inventory.InventoryDigest = objectDigest(inventory, "inventory_digest")
	var plan ProviderBindingPlan
	err = withBindingStore(m.layout.ProviderBindingDir, providerName, true, func(store *bindingStore) error {
		current, pending, err := m.readSnapshot(store, item)
		if err != nil {
			return err
		}
		if pending != nil {
			return ErrBindingRecoveryRequired
		}
		currentDigest := exposedStateDigest(current)
		if request.ExpectedStateDigest != currentDigest {
			return ErrBindingConflict
		}
		desiredState := "bound"
		kind := "bind"
		if request.InstallationID == "not_applicable" {
			desiredState, kind = "unbound_preserved", "unbind_preserved"
		} else if _, ok := declarations[request.InstallationID]; !ok {
			return ErrBindingInstallation
		}
		changed := request.InstallationID != current.InstallationID
		direction := "forward"
		if !changed {
			kind, direction = "retain", "none"
		} else if request.InstallationID == current.PreviousInstallationID {
			direction = "reverse"
		}
		planID, err := newBindingUUID()
		if err != nil {
			return err
		}
		plan = ProviderBindingPlan{
			Protocol: ProviderBindingPlanProtocol, PlanID: planID, TOPSID: m.layout.TOPSID, Scope: string(m.scope),
			ProviderName: item.Name, ProviderKind: item.Kind, DesiredState: desiredState, InstallationID: request.InstallationID,
			ExpectedStateDigest: request.ExpectedStateDigest, CurrentStateDigest: currentDigest, InventoryDigest: inventory.InventoryDigest,
			Actions:    []ProviderBindingAction{{ActionID: "provider-binding-change", Kind: kind, Direction: direction, DependsOn: []string{}}},
			Applicable: true, Changed: changed, Reason: request.Reason,
			ExpiresAt: timestamp(m.now().UTC().Add(providerBindingPlanLifetime)),
		}
		plan.PlanDigest = objectDigest(plan, "plan_digest")
		return store.write("plan.json", plan)
	})
	return plan, true, err
}

func (m *BindingManager) Prepare(providerName string, request ProviderBindingApplyRequest, auditIdentity ProviderBindingAuditIdentity) (ProviderBindingAttempt, bool, bool, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ProviderBindingAttempt{}, false, false, nil
	}
	if !validDigest(request.PlanDigest) || !validBindingStateDigest(request.ExpectedStateDigest) || !validBindingAuditIdentity(auditIdentity) {
		return ProviderBindingAttempt{}, true, false, fmt.Errorf("invalid provider binding apply request")
	}
	var attempt ProviderBindingAttempt
	var alreadyApplied bool
	err := withBindingStore(m.layout.ProviderBindingDir, providerName, true, func(store *bindingStore) error {
		current, pending, err := m.readSnapshot(store, item)
		if err != nil {
			return err
		}
		if request.ExpectedStateDigest != exposedStateDigest(current) {
			return ErrBindingConflict
		}
		if pending != nil {
			if pending.Plan.PlanDigest == request.PlanDigest {
				attempt = *pending
				return nil
			}
			return ErrBindingRecoveryRequired
		}
		var plan ProviderBindingPlan
		exists, err := store.read("plan.json", &plan)
		if err != nil || !exists || plan.PlanDigest != request.PlanDigest {
			return fmt.Errorf("provider binding plan was not issued by this service")
		}
		if err := m.validatePlan(plan, item); err != nil || plan.ExpectedStateDigest != request.ExpectedStateDigest {
			return fmt.Errorf("provider binding plan evidence mismatch")
		}
		if !plan.Applicable || plan.CurrentStateDigest != plan.ExpectedStateDigest {
			return fmt.Errorf("provider binding plan is not applicable to the exact current state")
		}
		expires, err := time.Parse("2006-01-02T15:04:05Z", plan.ExpiresAt)
		if err != nil || m.now().UTC().After(expires) {
			return fmt.Errorf("provider binding plan expired")
		}
		if !plan.Changed {
			alreadyApplied = true
			attempt = ProviderBindingAttempt{
				Protocol: providerBindingAttemptProtocol, OperationID: plan.PlanID, TOPSID: m.layout.TOPSID, Scope: string(m.scope),
				ProviderName: item.Name, ProviderKind: item.Kind, Plan: plan, AuditIdentity: auditIdentity, Stage: "committed", PreparedAt: timestamp(m.now()),
				ReceiptDigest: "not_applicable", TargetState: current,
			}
			now := timestamp(m.now())
			attempt.CandidateVerifiedAt, attempt.AuditedAt, attempt.CommittedAt = &now, &now, &now
			attempt.AttemptDigest = objectDigest(attempt, "attempt_digest")
			return store.remove("plan.json")
		}
		target := ProviderBindingState{
			Protocol: providerBindingStateProtocol, TOPSID: m.layout.TOPSID, Scope: string(m.scope), ProviderName: item.Name, ProviderKind: item.Kind,
			Generation: current.Generation + 1, BindingState: "bound", InstallationID: plan.InstallationID,
			PreviousInstallationID: current.InstallationID, PreviousStateDigest: exposedStateDigest(current), UpdatedAt: timestamp(m.now()),
		}
		if plan.InstallationID == "not_applicable" {
			target.BindingState = "unbound"
		}
		target.StateDigest = objectDigest(target, "state_digest")
		attempt = ProviderBindingAttempt{
			Protocol: providerBindingAttemptProtocol, OperationID: plan.PlanID, TOPSID: m.layout.TOPSID, Scope: string(m.scope),
			ProviderName: item.Name, ProviderKind: item.Kind, Plan: plan, AuditIdentity: auditIdentity, Stage: "prepared", PreparedAt: timestamp(m.now()),
			ReceiptDigest: "not_applicable", TargetState: target,
		}
		attempt.AttemptDigest = objectDigest(attempt, "attempt_digest")
		if err := store.write("attempt.json", attempt); err != nil {
			return err
		}
		return store.remove("plan.json")
	})
	return attempt, true, alreadyApplied, err
}

func (m *BindingManager) NoChangeResult(providerName string, attempt ProviderBindingAttempt) (ProviderBindingResult, error) {
	item, found := m.registry.Configuration(providerName)
	if !found || attempt.ProviderName != providerName || attempt.Stage != "committed" || attempt.Plan.Changed {
		return ProviderBindingResult{}, ErrBindingRecoveryAbsent
	}
	result := m.result("apply", item, attempt.TargetState, &attempt, false, false)
	if err := withBindingStore(m.layout.ProviderBindingDir, providerName, true, func(store *bindingStore) error {
		return store.write("result.json", result)
	}); err != nil {
		return ProviderBindingResult{}, err
	}
	return result, nil
}

func (m *BindingManager) VerifyCandidate(ctx context.Context, attempt ProviderBindingAttempt) (ProviderBindingAttempt, error) {
	item, found := m.registry.Configuration(attempt.ProviderName)
	if !found {
		return ProviderBindingAttempt{}, ErrBindingInstallation
	}
	if attempt.TargetState.InstallationID != "not_applicable" {
		_, declarations, err := m.discover(item)
		if err != nil {
			return ProviderBindingAttempt{}, err
		}
		declaration, ok := declarations[attempt.TargetState.InstallationID]
		if !ok {
			return ProviderBindingAttempt{}, ErrBindingInstallation
		}
		result := m.trust.verifyDeclaration(ctx, item, declaration, attempt.OperationID, attempt.OperationID)
		if result.TrustState != "verified" {
			return ProviderBindingAttempt{}, fmt.Errorf("provider binding candidate verification failed: %s", result.TrustState)
		}
	}
	if attempt.Stage == "prepared" {
		return m.advance(attempt.ProviderName, attempt.OperationID, "prepared", "candidate_verified", "not_applicable")
	}
	return attempt, nil
}

func (m *BindingManager) MarkAudited(providerName, operationID, expectedCandidateDigest string, receipt stavprotocol.Receipt) (ProviderBindingAttempt, error) {
	if receipt.Validate() != nil || receipt.Disposition != "committed" || receipt.RequestID != operationID || receipt.TOPSID != m.layout.TOPSID ||
		!validDigest(expectedCandidateDigest) || receipt.CandidateDigest != expectedCandidateDigest {
		return ProviderBindingAttempt{}, fmt.Errorf("STAV receipt is not committed and bound to the provider binding attempt")
	}
	return m.advance(providerName, operationID, "candidate_verified", "audited", objectDigest(receipt, ""))
}

func (m *BindingManager) Commit(providerName, operationID string, recovered bool) (ProviderBindingResult, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ProviderBindingResult{}, ErrBindingRecoveryAbsent
	}
	var result ProviderBindingResult
	err := withBindingStore(m.layout.ProviderBindingDir, providerName, true, func(store *bindingStore) error {
		current, pending, err := m.readSnapshot(store, item)
		if err != nil {
			return err
		}
		if pending == nil || pending.OperationID != operationID || (pending.Stage != "audited" && pending.Stage != "committed") {
			return ErrBindingRecoveryAbsent
		}
		if pending.TargetState.InstallationID != "not_applicable" {
			_, declarations, err := m.discover(item)
			if err != nil {
				return err
			}
			if _, exact := declarations[pending.TargetState.InstallationID]; !exact {
				return ErrBindingInstallation
			}
		}
		if current.StateDigest != pending.TargetState.StateDigest {
			if exposedStateDigest(current) != pending.TargetState.PreviousStateDigest {
				return ErrBindingConflict
			}
			if err := store.write("state.json", pending.TargetState); err != nil {
				return err
			}
		}
		if pending.Stage == "audited" {
			now := timestamp(m.now())
			pending.Stage, pending.CommittedAt = "committed", &now
			pending.AttemptDigest = objectDigest(*pending, "attempt_digest")
			if err := store.write("attempt.json", pending); err != nil {
				return err
			}
		}
		result = m.result("apply", item, pending.TargetState, pending, true, recovered)
		if recovered {
			result.Operation = "recover"
			result.ResultDigest = objectDigest(result, "result_digest")
		}
		if err := store.write("result.json", result); err != nil {
			return err
		}
		return store.remove("attempt.json")
	})
	return result, err
}

func (m *BindingManager) Pending(providerName string, request ProviderBindingRecoveryRequest) (ProviderBindingAttempt, bool, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ProviderBindingAttempt{}, false, nil
	}
	if !validBindingStateDigest(request.ExpectedStateDigest) || !validBindingReason(request.Reason) {
		return ProviderBindingAttempt{}, true, fmt.Errorf("invalid provider binding recovery request")
	}
	var result ProviderBindingAttempt
	err := withBindingStore(m.layout.ProviderBindingDir, providerName, false, func(store *bindingStore) error {
		current, pending, err := m.readSnapshot(store, item)
		if err != nil {
			return err
		}
		if pending == nil {
			return ErrBindingRecoveryAbsent
		}
		if request.ExpectedStateDigest != exposedStateDigest(current) {
			return ErrBindingConflict
		}
		result = *pending
		return nil
	})
	return result, true, err
}

func (m *BindingManager) AttemptStatus(providerName, operationID string) (ProviderBindingResult, bool, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ProviderBindingResult{}, false, nil
	}
	var result ProviderBindingResult
	err := withBindingStore(m.layout.ProviderBindingDir, providerName, false, func(store *bindingStore) error {
		current, pending, err := m.readSnapshot(store, item)
		if err != nil {
			return err
		}
		if pending == nil {
			var completed ProviderBindingResult
			exists, err := store.read("result.json", &completed)
			if err != nil {
				return err
			}
			if !exists || completed.OperationID != operationID || completed.ResultDigest != objectDigest(completed, "result_digest") {
				return ErrBindingRecoveryAbsent
			}
			completed.Operation = "apply-status"
			completed.ResultDigest = objectDigest(completed, "result_digest")
			result = completed
			return nil
		}
		if pending.OperationID != operationID {
			return ErrBindingRecoveryAbsent
		}
		changed := pending.Stage == "committed" && current.StateDigest == pending.TargetState.StateDigest
		result = m.result("apply-status", item, current, pending, changed, false)
		result.RecoveryRequired = true
		result.ReasonCode = "symphony.ssiag.provider.binding.recovery_required"
		result.ResultDigest = objectDigest(result, "result_digest")
		return nil
	})
	return result, true, err
}

func (m *BindingManager) AuditDigests(attempt ProviderBindingAttempt) (string, string, error) {
	item, found := m.registry.Configuration(attempt.ProviderName)
	if !found || attempt.TOPSID != m.layout.TOPSID || attempt.Scope != string(m.scope) || attempt.ProviderKind != item.Kind ||
		!validDigest(attempt.TargetState.StateDigest) {
		return "", "", fmt.Errorf("provider binding audit transition is invalid")
	}
	previous := attempt.TargetState.PreviousStateDigest
	if previous == "absent" {
		previous = objectDigest(providerBindingAbsence{
			Protocol: "symphony.ssiag.provider-binding-absence.v1", TOPSID: m.layout.TOPSID, Scope: string(m.scope),
			ProviderName: item.Name, ProviderKind: item.Kind,
		}, "")
	}
	if !validDigest(previous) {
		return "", "", fmt.Errorf("provider binding audit previous state is invalid")
	}
	return previous, attempt.TargetState.StateDigest, nil
}

func (m *BindingManager) ActiveDeclaration(providerName string) (ExecutableTrust, bool, error) {
	item, found := m.registry.Configuration(providerName)
	if !found {
		return ExecutableTrust{}, false, nil
	}
	var current ProviderBindingState
	err := withBindingStore(m.layout.ProviderBindingDir, providerName, false, func(store *bindingStore) error {
		var pending *ProviderBindingAttempt
		var err error
		current, pending, err = m.readSnapshot(store, item)
		_ = pending
		return err
	})
	if err != nil || current.Generation == 0 {
		return ExecutableTrust{}, false, err
	}
	if current.InstallationID == "not_applicable" {
		return ExecutableTrust{}, true, nil
	}
	_, declarations, err := m.discover(item)
	if err != nil {
		return ExecutableTrust{}, true, err
	}
	declaration, ok := declarations[current.InstallationID]
	if !ok {
		return ExecutableTrust{}, true, ErrBindingInstallation
	}
	return declaration, true, nil
}

func (m *BindingManager) advance(providerName, operationID, from, to, receiptDigest string) (ProviderBindingAttempt, error) {
	var result ProviderBindingAttempt
	err := withBindingStore(m.layout.ProviderBindingDir, providerName, true, func(store *bindingStore) error {
		item, found := m.registry.Configuration(providerName)
		if !found {
			return ErrBindingRecoveryAbsent
		}
		_, pending, err := m.readSnapshot(store, item)
		if err != nil {
			return err
		}
		if pending == nil || pending.OperationID != operationID {
			return ErrBindingRecoveryAbsent
		}
		if pending.Stage == to {
			result = *pending
			return nil
		}
		if pending.Stage != from {
			return ErrBindingRecoveryRequired
		}
		now := timestamp(m.now())
		pending.Stage = to
		switch to {
		case "candidate_verified":
			pending.CandidateVerifiedAt = &now
		case "audited":
			pending.AuditedAt = &now
			pending.ReceiptDigest = receiptDigest
		}
		pending.AttemptDigest = objectDigest(*pending, "attempt_digest")
		if err := store.write("attempt.json", pending); err != nil {
			return err
		}
		result = *pending
		return nil
	})
	return result, err
}

func (m *BindingManager) readSnapshot(store *bindingStore, item config.ProviderConfig) (ProviderBindingState, *ProviderBindingAttempt, error) {
	state := ProviderBindingState{
		Protocol: providerBindingStateProtocol, TOPSID: m.layout.TOPSID, Scope: string(m.scope), ProviderName: item.Name,
		ProviderKind: item.Kind, BindingState: "unbound", InstallationID: "not_applicable", PreviousInstallationID: "not_applicable",
		PreviousStateDigest: "absent", UpdatedAt: timestamp(m.now()),
	}
	if exists, err := store.read("state.json", &state); err != nil {
		return ProviderBindingState{}, nil, err
	} else if exists {
		if err := m.validateState(state, item); err != nil {
			return ProviderBindingState{}, nil, err
		}
	}
	var attempt ProviderBindingAttempt
	exists, err := store.read("attempt.json", &attempt)
	if err != nil {
		return ProviderBindingState{}, nil, err
	}
	if !exists {
		return state, nil, nil
	}
	if err := m.validateAttempt(attempt, item); err != nil {
		return ProviderBindingState{}, nil, fmt.Errorf("provider binding attempt evidence is invalid")
	}
	if state.StateDigest == attempt.TargetState.StateDigest {
		if attempt.Stage != "audited" && attempt.Stage != "committed" {
			return ProviderBindingState{}, nil, fmt.Errorf("provider binding state advanced before its audit committed")
		}
	} else if attempt.TargetState.PreviousStateDigest != exposedStateDigest(state) ||
		attempt.TargetState.Generation != state.Generation+1 ||
		attempt.TargetState.PreviousInstallationID != state.InstallationID {
		return ProviderBindingState{}, nil, fmt.Errorf("provider binding attempt is not linked to the exact current state")
	}
	return state, &attempt, nil
}

func (m *BindingManager) validatePlan(plan ProviderBindingPlan, item config.ProviderConfig) error {
	if plan.Protocol != ProviderBindingPlanProtocol || !validBindingUUID(plan.PlanID) || plan.TOPSID != m.layout.TOPSID ||
		plan.Scope != string(m.scope) || plan.ProviderName != item.Name || plan.ProviderKind != item.Kind ||
		!oneOf(plan.DesiredState, "bound", "unbound_preserved") || !validBindingInstallationID(plan.InstallationID) ||
		!validBindingStateDigest(plan.ExpectedStateDigest) || !validBindingStateDigest(plan.CurrentStateDigest) || !validDigest(plan.InventoryDigest) ||
		len(plan.Actions) == 0 || len(plan.Actions) > 32 || !validBindingReason(plan.Reason) || plan.RecoveryRequired ||
		plan.OperationalAccessEnabled || plan.ProviderOperationsEnabled || plan.SecretChannelEnabled || plan.CallerClassUsed || plan.Canonical ||
		plan.PlanDigest != objectDigest(plan, "plan_digest") || (plan.DesiredState == "bound") != (plan.InstallationID != "not_applicable") {
		return fmt.Errorf("provider binding plan is invalid")
	}
	if _, ok := strictTimestamp(plan.ExpiresAt); !ok {
		return fmt.Errorf("provider binding plan expiry is invalid")
	}
	seen := make(map[string]bool, len(plan.Actions))
	for _, action := range plan.Actions {
		if !validToken(action.ActionID) || seen[action.ActionID] || !oneOf(action.Kind, "retain", "bind", "unbind_preserved") ||
			!oneOf(action.Direction, "none", "forward", "reverse") || len(action.DependsOn) > 32 || !sort.StringsAreSorted(action.DependsOn) {
			return fmt.Errorf("provider binding action graph is invalid")
		}
		seen[action.ActionID] = true
		dependencySeen := map[string]bool{}
		for _, dependency := range action.DependsOn {
			if !validToken(dependency) || dependencySeen[dependency] {
				return fmt.Errorf("provider binding action dependencies are invalid")
			}
			dependencySeen[dependency] = true
		}
	}
	visiting := make(map[string]bool, len(plan.Actions))
	visited := make(map[string]bool, len(plan.Actions))
	var visit func(string) bool
	visit = func(actionID string) bool {
		if visiting[actionID] {
			return false
		}
		if visited[actionID] {
			return true
		}
		visiting[actionID] = true
		for _, dependency := range plan.Actions[indexOfBindingAction(plan.Actions, actionID)].DependsOn {
			if !seen[dependency] || !visit(dependency) {
				return false
			}
		}
		visiting[actionID] = false
		visited[actionID] = true
		return true
	}
	for actionID := range seen {
		if !visit(actionID) {
			return fmt.Errorf("provider binding action graph contains a missing dependency or cycle")
		}
	}
	return nil
}

func indexOfBindingAction(actions []ProviderBindingAction, actionID string) int {
	for index := range actions {
		if actions[index].ActionID == actionID {
			return index
		}
	}
	return -1
}

func (m *BindingManager) validateAttempt(attempt ProviderBindingAttempt, item config.ProviderConfig) error {
	if attempt.Protocol != providerBindingAttemptProtocol || !validBindingUUID(attempt.OperationID) || attempt.TOPSID != m.layout.TOPSID ||
		attempt.Scope != string(m.scope) || attempt.ProviderName != item.Name || attempt.ProviderKind != item.Kind ||
		!oneOf(attempt.Stage, "prepared", "candidate_verified", "audited", "committed") ||
		attempt.OperationalAccessEnabled || attempt.ProviderOperationsEnabled || attempt.SecretChannelEnabled ||
		attempt.AttemptDigest != objectDigest(attempt, "attempt_digest") || attempt.OperationID != attempt.Plan.PlanID ||
		!validBindingAuditIdentity(attempt.AuditIdentity) {
		return fmt.Errorf("provider binding attempt identity is invalid")
	}
	if err := m.validatePlan(attempt.Plan, item); err != nil || m.validateState(attempt.TargetState, item) != nil ||
		!attempt.Plan.Changed || attempt.TargetState.PreviousStateDigest != attempt.Plan.ExpectedStateDigest ||
		attempt.TargetState.InstallationID != attempt.Plan.InstallationID ||
		(attempt.Plan.DesiredState == "bound") != (attempt.TargetState.BindingState == "bound") {
		return fmt.Errorf("provider binding attempt contains invalid plan or target state")
	}
	preparedAt, ok := strictTimestamp(attempt.PreparedAt)
	if !ok {
		return fmt.Errorf("provider binding attempt preparation time is invalid")
	}
	parsedTime := func(value *string) (time.Time, bool) {
		if value == nil {
			return time.Time{}, false
		}
		return strictTimestamp(*value)
	}
	switch attempt.Stage {
	case "prepared":
		if attempt.CandidateVerifiedAt != nil || attempt.AuditedAt != nil || attempt.CommittedAt != nil || attempt.ReceiptDigest != "not_applicable" {
			return fmt.Errorf("prepared provider binding attempt has later-stage evidence")
		}
	case "candidate_verified":
		candidateAt, valid := parsedTime(attempt.CandidateVerifiedAt)
		if !valid || candidateAt.Before(preparedAt) || attempt.AuditedAt != nil || attempt.CommittedAt != nil || attempt.ReceiptDigest != "not_applicable" {
			return fmt.Errorf("candidate-verified provider binding attempt is inconsistent")
		}
	case "audited":
		candidateAt, candidateValid := parsedTime(attempt.CandidateVerifiedAt)
		auditedAt, auditedValid := parsedTime(attempt.AuditedAt)
		if !candidateValid || !auditedValid || candidateAt.Before(preparedAt) || auditedAt.Before(candidateAt) || attempt.CommittedAt != nil || !validDigest(attempt.ReceiptDigest) {
			return fmt.Errorf("audited provider binding attempt is inconsistent")
		}
	case "committed":
		candidateAt, candidateValid := parsedTime(attempt.CandidateVerifiedAt)
		auditedAt, auditedValid := parsedTime(attempt.AuditedAt)
		committedAt, committedValid := parsedTime(attempt.CommittedAt)
		if !candidateValid || !auditedValid || !committedValid || candidateAt.Before(preparedAt) || auditedAt.Before(candidateAt) || committedAt.Before(auditedAt) || !validDigest(attempt.ReceiptDigest) {
			return fmt.Errorf("committed provider binding attempt is inconsistent")
		}
	}
	return nil
}

func (m *BindingManager) validateState(state ProviderBindingState, item config.ProviderConfig) error {
	if state.Protocol != providerBindingStateProtocol || state.TOPSID != m.layout.TOPSID || state.Scope != string(m.scope) || state.ProviderName != item.Name ||
		state.ProviderKind != item.Kind || state.Generation == 0 || !oneOf(state.BindingState, "bound", "unbound") ||
		!validBindingInstallationID(state.InstallationID) || !validBindingInstallationID(state.PreviousInstallationID) ||
		!validBindingStateDigest(state.PreviousStateDigest) || state.StateDigest != objectDigest(state, "state_digest") ||
		state.OperationalAccessEnabled || state.ProviderOperationsEnabled || state.SecretChannelEnabled ||
		(state.BindingState == "unbound") != (state.InstallationID == "not_applicable") ||
		(state.Generation == 1) != (state.PreviousStateDigest == "absent") {
		return fmt.Errorf("provider binding state evidence is invalid")
	}
	if _, ok := strictTimestamp(state.UpdatedAt); !ok {
		return fmt.Errorf("provider binding state update time is invalid")
	}
	return nil
}

func (m *BindingManager) statusFrom(item config.ProviderConfig, state ProviderBindingState, attempt *ProviderBindingAttempt) ProviderBindingStatus {
	result := ProviderBindingStatus{
		Protocol: ProviderBindingStatusProtocol, TOPSID: m.layout.TOPSID, Scope: string(m.scope), ProviderName: item.Name,
		ProviderKind: item.Kind, BindingState: state.BindingState, Generation: state.Generation, InstallationID: state.InstallationID,
		PreviousInstallationID: state.PreviousInstallationID, StateDigest: exposedStateDigest(state), AttemptState: "none",
		AttemptOperationID: "not_applicable", AttemptDigest: "not_applicable", ReasonCode: "symphony.ssiag.provider.binding.observed",
		ObservedAt: timestamp(m.now()), ReadOnly: true,
	}
	if state.Generation == 0 {
		result.BindingState, result.ReasonCode = "unbound", "symphony.ssiag.provider.binding.absent"
	}
	if attempt != nil {
		result.BindingState = "recovery_required"
		result.AttemptState, result.AttemptOperationID, result.AttemptDigest = attempt.Stage, attempt.OperationID, attempt.AttemptDigest
		result.RecoveryRequired, result.ReasonCode = true, "symphony.ssiag.provider.binding.recovery_required"
	}
	result.ResultDigest = objectDigest(result, "result_digest")
	return result
}

func (m *BindingManager) result(operation string, item config.ProviderConfig, state ProviderBindingState, attempt *ProviderBindingAttempt, changed, recovered bool) ProviderBindingResult {
	result := ProviderBindingResult{
		Protocol: ProviderBindingResultProtocol, Operation: operation, OperationID: attempt.OperationID, TOPSID: m.layout.TOPSID,
		Scope: string(m.scope), ProviderName: item.Name, ProviderKind: item.Kind, BindingState: state.BindingState,
		Generation: state.Generation, InstallationID: state.InstallationID, PreviousInstallationID: state.PreviousInstallationID,
		StateDigest: exposedStateDigest(state), AttemptState: attempt.Stage, AttemptDigest: attempt.AttemptDigest,
		ReceiptDigest: attempt.ReceiptDigest, Changed: changed, Recovered: recovered, ObservedAt: timestamp(m.now()),
		ReasonCode: "symphony.ssiag.provider.binding.succeeded",
	}
	if attempt.Stage != "committed" {
		result.RecoveryRequired = true
		result.ReasonCode = "symphony.ssiag.provider.binding.recovery_required"
	}
	result.ResultDigest = objectDigest(result, "result_digest")
	return result
}

func (m *BindingManager) discover(item config.ProviderConfig) ([]ProviderInstallation, map[string]ExecutableTrust, error) {
	roots := append([]string(nil), m.roots...)
	legacyPath := filepath.Join(m.layout.ProviderTrustDir, item.Name+".json")
	if legacy, exists, err := loadExecutableTrust(legacyPath, m.scope); err == nil && exists {
		if root, ok := providerPrefix(legacy.ExecutablePath); ok && !containsString(roots, root) {
			roots = append(roots, root)
		}
	}
	sort.Strings(roots)
	installations := make([]ProviderInstallation, 0)
	declarations := make(map[string]ExecutableTrust)
	for _, root := range roots {
		receiptRoot := filepath.Join(root, "share", "symphony", "receipts", macOSKeychainPackageID)
		versions, err := os.ReadDir(receiptRoot)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("provider installation inventory root is unreadable")
		}
		if len(versions) > 128 {
			return nil, nil, fmt.Errorf("provider installation inventory exceeds its 128-entry bound")
		}
		if len(installations)+len(versions) > 128 {
			return nil, nil, fmt.Errorf("provider installation inventory exceeds its cumulative 128-entry bound")
		}
		for _, entry := range versions {
			if !entry.IsDir() {
				return nil, nil, fmt.Errorf("provider installation inventory contains an unsafe non-directory entry")
			}
			if !validToken(entry.Name()) {
				return nil, nil, fmt.Errorf("provider installation inventory contains an unsafe version directory")
			}
			declaration, err := m.declaration(root, entry.Name(), item)
			if err != nil {
				return nil, nil, fmt.Errorf("provider installation inventory contains unsafe or incomplete evidence")
			}
			state, reason := "exact", "symphony.ssiag.provider.binding.compatible"
			if err := validateExecutableTrust(declaration, item, m.layout, m.trust.foundation); err != nil {
				state, reason = "incompatible", "symphony.ssiag.provider.binding.identity_mismatch"
			} else if err := validateAdapterReceipt(declaration, m.scope); err != nil {
				state, reason = "incompatible", "symphony.ssiag.provider.binding.receipt_mismatch"
			} else if !validDigest(declaration.FoundationInstallationDigest) {
				state, reason = "incompatible", "symphony.ssiag.provider.binding.foundation_unreceipted"
			}
			installation := ProviderInstallation{
				InstallationID: declaration.DeclarationDigest, AdapterIdentifier: declaration.AdapterIdentifier, AdapterVersion: declaration.AdapterVersion,
				ProviderProtocol: declaration.ProviderProtocol, CommandProtocol: ProviderControlProtocol, ReceiptDigest: declaration.InstallationDigest,
				ExecutableDigest: declaration.ExecutableDigest, FoundationVersion: version.Version,
				FoundationReceiptDigest: declaration.FoundationInstallationDigest, FoundationExecutableDigest: declaration.FoundationExecutableDigest,
				CompatibilityState: state, ReasonCode: reason,
			}
			installations = append(installations, installation)
			if state == "exact" {
				declarations[installation.InstallationID] = declaration
			}
		}
	}
	sort.Slice(installations, func(i, j int) bool { return installations[i].InstallationID < installations[j].InstallationID })
	for index := 1; index < len(installations); index++ {
		if installations[index-1].InstallationID == installations[index].InstallationID {
			return nil, nil, fmt.Errorf("provider installation inventory contains a duplicate opaque identity")
		}
	}
	return installations, declarations, nil
}

func (m *BindingManager) declaration(root, adapterVersion string, item config.ProviderConfig) (ExecutableTrust, error) {
	executable := filepath.Join(root, "libexec", "symphony", macOSKeychainPackageID, adapterVersion, macOSKeychainExecutableName)
	receiptPath := filepath.Join(root, "share", "symphony", "receipts", macOSKeychainPackageID, adapterVersion, "install-receipt.json")
	receipt, _, _, err := readAdapterReceipt(receiptPath, m.scope)
	if err != nil {
		return ExecutableTrust{}, err
	}
	info, err := os.Lstat(executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return ExecutableTrust{}, fmt.Errorf("provider executable is unsafe")
	}
	uid, gid, err := fileOwner(info)
	if err != nil {
		return ExecutableTrust{}, err
	}
	digest, err := digestPath(executable)
	if err != nil {
		return ExecutableTrust{}, err
	}
	value := ExecutableTrust{
		Protocol: "symphony.ssiag.provider-executable-trust.v1", TOPSID: m.layout.TOPSID, Scope: string(m.scope), ProviderName: item.Name, ProviderKind: item.Kind,
		AdapterIdentifier: MacOSKeychainAdapterID, AdapterVersion: adapterVersion, ProviderProtocol: ProviderProtocol, ExecutablePath: executable,
		InstallationDigest: receipt.ReceiptDigest, ExecutableDigest: digest, OwnerUID: uid, OwnerGID: gid, FileMode: fmt.Sprintf("0%03o", info.Mode().Perm()),
		AdapterSigningIdentity: "not_applicable", FoundationExecutablePath: m.trust.foundation.ExecutablePath,
		FoundationInstallationDigest: m.trust.foundation.InstallationDigest, FoundationExecutableDigest: m.trust.foundation.ExecutableDigest,
		FoundationOwnerUID: m.trust.foundation.OwnerUID, FoundationOwnerGID: m.trust.foundation.OwnerGID, FoundationSigningIdentity: m.trust.foundation.SigningIdentity,
	}
	value.DeclarationDigest = objectDigest(value, "declaration_digest")
	return value, nil
}

func providerPrefix(executable string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(executable)), "/")
	for i := 1; i+4 < len(parts); i++ {
		if parts[i] == "libexec" && parts[i+1] == "symphony" && parts[i+2] == macOSKeychainPackageID {
			return "/" + filepath.Join(parts[1:i]...), true
		}
	}
	return "", false
}

func exposedStateDigest(state ProviderBindingState) string {
	if state.Generation == 0 {
		return "absent"
	}
	return state.StateDigest
}

func validBindingStateDigest(value string) bool { return value == "absent" || validDigest(value) }
func validBindingInstallationID(value string) bool {
	return value == "not_applicable" || validDigest(value)
}
func validBindingReason(value string) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > 1024 {
		return false
	}
	for _, character := range value {
		if character == 0 || character == '\r' || character == '\n' {
			return false
		}
	}
	return true
}

func validBindingAuditIdentity(value ProviderBindingAuditIdentity) bool {
	return validToken(value.ActorID) && validToken(value.ActorKind) && validToken(value.AuthenticationMethod) &&
		value.ActorID != "not_applicable" && value.ActorKind != "not_applicable" && value.AuthenticationMethod != "not_applicable"
}

func validBindingUUID(value string) bool {
	if len(value) != 36 || strings.ToLower(value) != value || value == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for index, character := range value {
		switch index {
		case 8, 13, 18, 23:
			if character != '-' {
				return false
			}
		default:
			if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
				return false
			}
		}
	}
	return value[14] >= '1' && value[14] <= '8' && strings.Contains("89ab", value[19:20])
}
func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func newBindingUUID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}

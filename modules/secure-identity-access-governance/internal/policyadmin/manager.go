package policyadmin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
)

const (
	ProposalRequestProtocol = "symphony.ssiag.policy-proposal-request.v1"
	ProposalProtocol        = "symphony.ssiag.policy-proposal.v1"
	ApplyRequestProtocol    = "symphony.ssiag.policy-apply-request.v1"
	RecoveryRequestProtocol = "symphony.ssiag.policy-recovery-request.v1"
	ResultProtocol          = "symphony.ssiag.policy-result.v1"
	stateProtocol           = "symphony.ssiag.policy-state.v1"
	attemptProtocol         = "symphony.ssiag.policy-attempt.v1"
	maxProposalLifetime     = 10 * time.Minute
)

var (
	ErrConflict         = errors.New("SSIAG policy compare-and-swap conflict")
	ErrRecoveryRequired = errors.New("SSIAG policy recovery is required")
	ErrNoRecovery       = errors.New("SSIAG policy recovery evidence is absent")
)

type ProposalRequest struct {
	Protocol             string                      `json:"protocol"`
	OperationID          string                      `json:"operation_id"`
	RequestID            string                      `json:"request_id"`
	CorrelationID        string                      `json:"correlation_id"`
	AuthorityBasis       string                      `json:"authority_basis"`
	ExpectedPolicyDigest string                      `json:"expected_policy_digest"`
	Change               string                      `json:"change"`
	DesiredPolicy        *config.AuthorizationConfig `json:"desired_policy"`
	RequestedAt          time.Time                   `json:"requested_at"`
	ExpiresAt            time.Time                   `json:"expires_at"`
}

type Proposal struct {
	Protocol             string                      `json:"protocol"`
	OperationID          string                      `json:"operation_id"`
	RequestID            string                      `json:"request_id"`
	CorrelationID        string                      `json:"correlation_id"`
	TOPSID               string                      `json:"tops_id"`
	Subject              identity.Subject            `json:"subject"`
	AuthorityBasis       string                      `json:"authority_basis"`
	ExpectedPolicyDigest string                      `json:"expected_policy_digest"`
	Change               string                      `json:"change"`
	DesiredPolicy        *config.AuthorizationConfig `json:"desired_policy"`
	DesiredPolicyDigest  string                      `json:"desired_policy_digest"`
	ConfigDigest         string                      `json:"config_digest"`
	CreatedAt            time.Time                   `json:"created_at"`
	ExpiresAt            time.Time                   `json:"expires_at"`
	CallerClassUsed      bool                        `json:"caller_class_used"`
	Canonical            bool                        `json:"canonical"`
	Applied              bool                        `json:"applied"`
	ProposalDigest       string                      `json:"proposal_digest"`
}

type ApplyRequest struct {
	Protocol string   `json:"protocol"`
	Proposal Proposal `json:"proposal"`
}

type RecoveryRequest struct {
	Protocol              string `json:"protocol"`
	OperationID           string `json:"operation_id"`
	ExpectedAttemptDigest string `json:"expected_attempt_digest,omitempty"`
	Discover              bool   `json:"discover"`
}

type Result struct {
	Protocol         string    `json:"protocol"`
	Operation        string    `json:"operation"`
	TOPSID           string    `json:"tops_id"`
	Source           string    `json:"source"`
	Generation       uint64    `json:"generation"`
	PolicyDigest     string    `json:"policy_digest"`
	StateDigest      string    `json:"state_digest"`
	RecoveryRequired bool      `json:"recovery_required"`
	AttemptDigest    string    `json:"attempt_digest,omitempty"`
	Changed          bool      `json:"changed"`
	Recovered        bool      `json:"recovered"`
	ObservedAt       time.Time `json:"observed_at"`
	ReadOnly         bool      `json:"read_only"`
	CallerClassUsed  bool      `json:"caller_class_used"`
	Canonical        bool      `json:"canonical"`
}

type state struct {
	Protocol             string                      `json:"protocol"`
	TOPSID               string                      `json:"tops_id"`
	Generation           uint64                      `json:"generation"`
	Source               string                      `json:"source"`
	PreviousPolicyDigest string                      `json:"previous_policy_digest"`
	Policy               *config.AuthorizationConfig `json:"policy"`
	PolicyDigest         string                      `json:"policy_digest"`
	ConfigDigest         string                      `json:"config_digest"`
	UpdatedAt            time.Time                   `json:"updated_at"`
	StateDigest          string                      `json:"state_digest"`
}

type Attempt struct {
	Protocol      string     `json:"protocol"`
	TOPSID        string     `json:"tops_id"`
	Proposal      Proposal   `json:"proposal"`
	Stage         string     `json:"stage"`
	PreparedAt    time.Time  `json:"prepared_at"`
	AuditedAt     *time.Time `json:"audited_at"`
	ReceiptDigest string     `json:"receipt_digest,omitempty"`
	AttemptDigest string     `json:"attempt_digest"`
}

type Manager struct {
	stateDir     string
	cfg          config.Config
	configDigest string
	now          func() time.Time
}

func New(stateDir string, cfg config.Config, now func() time.Time) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	configDigest, err := digest(cfg)
	if err != nil {
		return nil, err
	}
	manager := &Manager{stateDir: stateDir, cfg: cfg, configDigest: configDigest, now: now}
	if _, _, err := manager.snapshot(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *Manager) Effective() (*config.AuthorizationConfig, string, error) {
	current, _, err := m.snapshot()
	if err != nil {
		return nil, "", err
	}
	policyValue := current.Policy
	if current.Source == "config" {
		policyValue = m.configPolicy()
	}
	copyValue := clonePolicy(policyValue)
	return copyValue, current.PolicyDigest, nil
}

func (m *Manager) Status() (Result, error) {
	current, pending, err := m.snapshot()
	if err != nil {
		return Result{}, err
	}
	return m.result("status", current, pending, false, false, true), nil
}

func (m *Manager) Propose(subject identity.Subject, request ProposalRequest) (Proposal, error) {
	now := m.now().UTC().Truncate(time.Second)
	if err := m.validateRequest(request, now); err != nil {
		return Proposal{}, err
	}
	var proposal Proposal
	err := withStore(m.stateDir, true, func(store *store) error {
		current, pending, err := m.readSnapshot(store)
		if err != nil {
			return err
		}
		if pending != nil {
			return ErrRecoveryRequired
		}
		if request.ExpectedPolicyDigest != current.PolicyDigest {
			return ErrConflict
		}
		desired := clonePolicy(request.DesiredPolicy)
		if request.Change == "reset" {
			desired = m.configPolicy()
		}
		desiredDigest, err := digest(desired)
		if err != nil {
			return err
		}
		proposal = Proposal{
			Protocol: ProposalProtocol, OperationID: request.OperationID,
			RequestID: request.RequestID, CorrelationID: request.CorrelationID,
			TOPSID: m.cfg.TOPS.ID, Subject: subject, AuthorityBasis: request.AuthorityBasis,
			ExpectedPolicyDigest: request.ExpectedPolicyDigest, Change: request.Change,
			DesiredPolicy: clonePolicy(request.DesiredPolicy), DesiredPolicyDigest: desiredDigest,
			ConfigDigest: m.configDigest, CreatedAt: now, ExpiresAt: request.ExpiresAt.UTC().Truncate(time.Second),
			CallerClassUsed: false, Canonical: false, Applied: false,
		}
		proposal.ProposalDigest, err = digestWithout(proposal, "proposal_digest")
		if err != nil {
			return err
		}
		return store.write("proposal.json", proposal)
	})
	return proposal, err
}

func (m *Manager) Prepare(proposal Proposal) (Attempt, bool, error) {
	var result Attempt
	var alreadyApplied bool
	err := withStore(m.stateDir, true, func(store *store) error {
		current, pending, err := m.readSnapshot(store)
		if err != nil {
			return err
		}
		if err := m.validateProposal(proposal, m.now().UTC().Truncate(time.Second)); err != nil {
			return err
		}
		if pending != nil {
			if pending.Proposal.ProposalDigest == proposal.ProposalDigest {
				result = *pending
				return nil
			}
			return ErrRecoveryRequired
		}
		if current.PolicyDigest != proposal.ExpectedPolicyDigest {
			if current.PolicyDigest == proposal.DesiredPolicyDigest {
				alreadyApplied = true
				return store.remove("proposal.json")
			}
			return ErrConflict
		}
		var issued Proposal
		exists, err := store.read("proposal.json", &issued)
		if err != nil {
			return err
		}
		if !exists || issued.ProposalDigest != proposal.ProposalDigest {
			return fmt.Errorf("SSIAG policy proposal was not issued by this service")
		}
		issuedDigest, err := digest(issued)
		if err != nil {
			return err
		}
		suppliedDigest, err := digest(proposal)
		if err != nil || issuedDigest != suppliedDigest {
			return fmt.Errorf("SSIAG policy proposal does not match issued evidence")
		}
		result = Attempt{
			Protocol: attemptProtocol, TOPSID: m.cfg.TOPS.ID, Proposal: proposal,
			Stage: "prepared", PreparedAt: m.now().UTC().Truncate(time.Second),
		}
		result.AttemptDigest, err = digestWithout(result, "attempt_digest")
		if err != nil {
			return err
		}
		if err := store.write("attempt.json", result); err != nil {
			return err
		}
		return store.remove("proposal.json")
	})
	return result, alreadyApplied, err
}

func (m *Manager) MarkAudited(proposalDigest string, receipt stavprotocol.Receipt) (Attempt, error) {
	var result Attempt
	err := withStore(m.stateDir, true, func(store *store) error {
		_, pending, err := m.readSnapshot(store)
		if err != nil {
			return err
		}
		if pending == nil || pending.Proposal.ProposalDigest != proposalDigest {
			return ErrNoRecovery
		}
		if pending.Stage == "audited" {
			result = *pending
			return nil
		}
		if receipt.Disposition != "committed" || receipt.RequestID != pending.Proposal.RequestID || receipt.TOPSID != m.cfg.TOPS.ID {
			return fmt.Errorf("STAV receipt is not committed and bound to the SSIAG policy attempt")
		}
		receiptDigest, err := digest(receipt)
		if err != nil {
			return err
		}
		now := m.now().UTC().Truncate(time.Second)
		pending.Stage = "audited"
		pending.AuditedAt = &now
		pending.ReceiptDigest = receiptDigest
		pending.AttemptDigest, err = digestWithout(*pending, "attempt_digest")
		if err != nil {
			return err
		}
		if err := store.write("attempt.json", pending); err != nil {
			return err
		}
		result = *pending
		return nil
	})
	return result, err
}

func (m *Manager) Commit(proposalDigest string, recovered bool) (Result, *config.AuthorizationConfig, error) {
	var result Result
	var effective *config.AuthorizationConfig
	err := withStore(m.stateDir, true, func(store *store) error {
		current, pending, err := m.readSnapshot(store)
		if err != nil {
			return err
		}
		if pending == nil || pending.Proposal.ProposalDigest != proposalDigest || pending.Stage != "audited" {
			return ErrNoRecovery
		}
		desired := clonePolicy(pending.Proposal.DesiredPolicy)
		source := "overlay"
		storedPolicy := desired
		if pending.Proposal.Change == "reset" {
			source = "config"
			storedPolicy = nil
			desired = m.configPolicy()
		}
		next := state{
			Protocol: stateProtocol, TOPSID: m.cfg.TOPS.ID, Generation: current.Generation + 1,
			Source: source, PreviousPolicyDigest: current.PolicyDigest, Policy: storedPolicy,
			PolicyDigest: pending.Proposal.DesiredPolicyDigest, ConfigDigest: m.configDigest,
			UpdatedAt: m.now().UTC().Truncate(time.Second),
		}
		next.StateDigest, err = digestWithout(next, "state_digest")
		if err != nil {
			return err
		}
		if err := store.write("state.json", next); err != nil {
			return err
		}
		if err := store.remove("attempt.json"); err != nil {
			return err
		}
		effective = desired
		result = m.result("apply", next, nil, true, recovered, false)
		return nil
	})
	return result, effective, err
}

func (m *Manager) Pending(request RecoveryRequest) (Attempt, error) {
	if request.Protocol != RecoveryRequestProtocol || !safeToken(request.OperationID) || request.Discover == (request.ExpectedAttemptDigest != "") {
		return Attempt{}, fmt.Errorf("invalid SSIAG policy recovery request")
	}
	_, pending, err := m.snapshot()
	if err != nil {
		return Attempt{}, err
	}
	if pending == nil {
		return Attempt{}, ErrNoRecovery
	}
	if pending.Proposal.OperationID != request.OperationID {
		return Attempt{}, ErrConflict
	}
	if !request.Discover && pending.AttemptDigest != request.ExpectedAttemptDigest {
		return Attempt{}, ErrConflict
	}
	return *pending, nil
}

func (m *Manager) validateRequest(request ProposalRequest, now time.Time) error {
	if request.Protocol != ProposalRequestProtocol || !safeToken(request.OperationID) ||
		!safeToken(request.RequestID) || !safeToken(request.CorrelationID) ||
		!validDigest(request.ExpectedPolicyDigest) {
		return fmt.Errorf("invalid SSIAG policy proposal identity")
	}
	if request.AuthorityBasis != "host_owner" && request.AuthorityBasis != "granted_permission" {
		return fmt.Errorf("invalid SSIAG policy proposal authority basis")
	}
	if request.Change != "replace" && request.Change != "reset" {
		return fmt.Errorf("invalid SSIAG policy proposal change")
	}
	if request.Change == "replace" {
		if request.DesiredPolicy == nil || config.ValidateAuthorization(request.DesiredPolicy, m.cfg.Authentication) != nil {
			return fmt.Errorf("invalid SSIAG replacement authorization policy")
		}
	} else if request.DesiredPolicy != nil {
		return fmt.Errorf("reset proposal must not carry a desired policy")
	}
	if !exactUTCSecond(request.RequestedAt) || !exactUTCSecond(request.ExpiresAt) ||
		request.RequestedAt.Before(now.Add(-30*time.Second)) || request.RequestedAt.After(now.Add(30*time.Second)) ||
		!request.ExpiresAt.After(now) || !request.ExpiresAt.After(request.RequestedAt) ||
		request.ExpiresAt.Sub(request.RequestedAt) > maxProposalLifetime {
		return fmt.Errorf("SSIAG policy proposal timestamps are invalid")
	}
	return nil
}

func (m *Manager) validateProposal(proposal Proposal, now time.Time) error {
	if proposal.Protocol != ProposalProtocol || proposal.TOPSID != m.cfg.TOPS.ID ||
		proposal.ConfigDigest != m.configDigest || proposal.CallerClassUsed || proposal.Canonical || proposal.Applied ||
		!validDigest(proposal.ProposalDigest) || !validDigest(proposal.ExpectedPolicyDigest) ||
		!validDigest(proposal.DesiredPolicyDigest) || !safeToken(proposal.OperationID) ||
		!safeToken(proposal.RequestID) || !safeToken(proposal.CorrelationID) ||
		!safeToken(proposal.Subject.ID) || !safeToken(proposal.Subject.Kind) || !safeToken(proposal.Subject.Authority) ||
		(proposal.AuthorityBasis != "host_owner" && proposal.AuthorityBasis != "granted_permission") ||
		!exactUTCSecond(proposal.CreatedAt) || !exactUTCSecond(proposal.ExpiresAt) ||
		!proposal.ExpiresAt.After(now) || !proposal.ExpiresAt.After(proposal.CreatedAt) ||
		proposal.ExpiresAt.Sub(proposal.CreatedAt) > maxProposalLifetime {
		return fmt.Errorf("invalid or expired SSIAG policy proposal")
	}
	want, err := digestWithout(proposal, "proposal_digest")
	if err != nil || want != proposal.ProposalDigest {
		return fmt.Errorf("SSIAG policy proposal digest mismatch")
	}
	desired := proposal.DesiredPolicy
	if proposal.Change == "reset" {
		if desired != nil {
			return fmt.Errorf("reset proposal carries a policy")
		}
		desired = m.configPolicy()
	} else if proposal.Change != "replace" || desired == nil || config.ValidateAuthorization(desired, m.cfg.Authentication) != nil {
		return fmt.Errorf("invalid SSIAG policy proposal change")
	}
	wantPolicy, err := digest(desired)
	if err != nil || wantPolicy != proposal.DesiredPolicyDigest {
		return fmt.Errorf("SSIAG desired policy digest mismatch")
	}
	return nil
}

func (m *Manager) snapshot() (state, *Attempt, error) {
	var current state
	var pending *Attempt
	err := withStore(m.stateDir, false, func(store *store) error {
		var err error
		current, pending, err = m.readSnapshot(store)
		return err
	})
	return current, pending, err
}

func (m *Manager) readSnapshot(store *store) (state, *Attempt, error) {
	current := state{Protocol: stateProtocol, TOPSID: m.cfg.TOPS.ID, Source: "config", ConfigDigest: m.configDigest}
	current.PolicyDigest, _ = digest(m.configPolicy())
	var persisted state
	if exists, err := store.read("state.json", &persisted); err != nil {
		return state{}, nil, err
	} else if exists {
		if err := m.validateState(persisted); err != nil {
			return state{}, nil, err
		}
		current = persisted
	}
	var pending Attempt
	exists, err := store.read("attempt.json", &pending)
	if err != nil {
		return state{}, nil, err
	}
	if !exists {
		return current, nil, nil
	}
	if err := m.validateAttempt(pending); err != nil {
		return state{}, nil, err
	}
	return current, &pending, nil
}

func (m *Manager) validateState(value state) error {
	if value.Protocol != stateProtocol || value.TOPSID != m.cfg.TOPS.ID || value.Generation == 0 ||
		value.ConfigDigest != m.configDigest || !validDigest(value.PreviousPolicyDigest) ||
		!validDigest(value.PolicyDigest) || !validDigest(value.StateDigest) ||
		!exactUTCSecond(value.UpdatedAt) || (value.Source != "config" && value.Source != "overlay") {
		return fmt.Errorf("invalid SSIAG policy state")
	}
	policyValue := value.Policy
	if value.Source == "config" {
		if policyValue != nil {
			return fmt.Errorf("config-backed SSIAG policy state contains an overlay")
		}
		policyValue = m.configPolicy()
	} else if policyValue == nil || config.ValidateAuthorization(policyValue, m.cfg.Authentication) != nil {
		return fmt.Errorf("invalid SSIAG policy overlay")
	}
	wantPolicy, _ := digest(policyValue)
	wantState, _ := digestWithout(value, "state_digest")
	if wantPolicy != value.PolicyDigest || wantState != value.StateDigest {
		return fmt.Errorf("SSIAG policy state digest mismatch")
	}
	return nil
}

func (m *Manager) validateAttempt(value Attempt) error {
	if value.Protocol != attemptProtocol || value.TOPSID != m.cfg.TOPS.ID ||
		(value.Stage != "prepared" && value.Stage != "audited") || !validDigest(value.AttemptDigest) ||
		!exactUTCSecond(value.PreparedAt) {
		return fmt.Errorf("invalid SSIAG policy attempt")
	}
	if value.Stage == "prepared" && (value.AuditedAt != nil || value.ReceiptDigest != "") {
		return fmt.Errorf("prepared SSIAG policy attempt contains audit completion")
	}
	if value.Stage == "audited" && (value.AuditedAt == nil || !exactUTCSecond(*value.AuditedAt) || !validDigest(value.ReceiptDigest)) {
		return fmt.Errorf("audited SSIAG policy attempt lacks receipt evidence")
	}
	want, _ := digestWithout(value, "attempt_digest")
	if want != value.AttemptDigest {
		return fmt.Errorf("SSIAG policy attempt digest mismatch")
	}
	// Recovery may occur after proposal expiry, so structural validation uses
	// the instant at which the attempt was durably prepared.
	return m.validateProposal(value.Proposal, value.PreparedAt)
}

func (m *Manager) result(operation string, current state, pending *Attempt, changed, recovered, readOnly bool) Result {
	result := Result{
		Protocol: ResultProtocol, Operation: operation, TOPSID: m.cfg.TOPS.ID,
		Source: current.Source, Generation: current.Generation, PolicyDigest: current.PolicyDigest,
		StateDigest: current.StateDigest, Changed: changed, Recovered: recovered,
		ObservedAt: m.now().UTC().Truncate(time.Second), ReadOnly: readOnly,
		CallerClassUsed: false, Canonical: false,
	}
	if result.StateDigest == "" {
		result.StateDigest = "absent"
	}
	if pending != nil {
		result.RecoveryRequired = true
		result.AttemptDigest = pending.AttemptDigest
	}
	return result
}

func (m *Manager) configPolicy() *config.AuthorizationConfig {
	if m.cfg.Authorization == nil {
		return &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 900, Grants: []config.AuthorizationGrant{}}
	}
	return clonePolicy(m.cfg.Authorization)
}

func clonePolicy(value *config.AuthorizationConfig) *config.AuthorizationConfig {
	if value == nil {
		return nil
	}
	copyValue := *value
	copyValue.Grants = make([]config.AuthorizationGrant, len(value.Grants))
	copy(copyValue.Grants, value.Grants)
	sort.Slice(copyValue.Grants, func(i, j int) bool { return copyValue.Grants[i].ID < copyValue.Grants[j].ID })
	return &copyValue
}

func digest(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode SSIAG policy digest: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func digestWithout(value any, field string) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		return "", err
	}
	delete(object, field)
	return digest(object)
}

func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func safeToken(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune("._:-", character) {
			continue
		}
		return false
	}
	return true
}

func exactUTCSecond(value time.Time) bool {
	return !value.IsZero() && value.Location() == time.UTC && value.Nanosecond() == 0
}

package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
)

const (
	requestSchema    = "symphony.ssiag.authorization-request.v1"
	decisionSchema   = "symphony.ssiag.authorization-decision.v1"
	capabilitySchema = "symphony.ssiag.capability.v1"
	requestSkew      = 30 * time.Second
)

type Evaluator interface {
	Evaluate(context.Context, identity.Subject, model.AuthorizationRequest) model.AuthorizationDecision
}

type Engine struct {
	topsID       string
	policy       *config.AuthorizationConfig
	policyDigest string
	configDigest string
	now          func() time.Time
}

func New(cfg config.Config, now func() time.Time) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	configBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode configuration digest: %w", err)
	}
	policyValue := cfg.Authorization
	if policyValue == nil {
		policyValue = &config.AuthorizationConfig{
			DefaultEffect: "deny", MaxCapabilitySeconds: 900,
			Grants: []config.AuthorizationGrant{},
		}
	}
	policyBytes, err := json.Marshal(policyValue)
	if err != nil {
		return nil, fmt.Errorf("encode policy digest: %w", err)
	}
	return &Engine{
		topsID: cfg.TOPS.ID, policy: policyValue,
		policyDigest: taggedDigest(policyBytes),
		configDigest: taggedDigest(configBytes), now: now,
	}, nil
}

func ValidateRequest(request model.AuthorizationRequest, now time.Time) error {
	if request.Schema != requestSchema {
		return fmt.Errorf("unsupported authorization request schema %q", request.Schema)
	}
	for name, value := range map[string]string{
		"request_id": request.RequestID, "correlation_id": request.CorrelationID,
		"operation": request.Operation, "resource": request.Resource,
		"audience": request.Audience, "scope": request.Scope,
	} {
		if !safeToken(value) {
			return fmt.Errorf("authorization %s is invalid", name)
		}
	}
	now = now.UTC()
	if request.RequestedAt.IsZero() || request.RequestedExpiresAt.IsZero() ||
		request.RequestedAt.Location() != time.UTC || request.RequestedExpiresAt.Location() != time.UTC {
		return fmt.Errorf("authorization timestamps must be explicit UTC values")
	}
	if request.RequestedAt.Before(now.Add(-requestSkew)) || request.RequestedAt.After(now.Add(requestSkew)) {
		return fmt.Errorf("authorization request is outside the freshness window")
	}
	if !request.RequestedExpiresAt.After(now) {
		return fmt.Errorf("authorization request is already expired")
	}
	if !request.RequestedExpiresAt.After(request.RequestedAt) {
		return fmt.Errorf("authorization expiry must follow the request timestamp")
	}
	return nil
}

func (e *Engine) Evaluate(_ context.Context, subject identity.Subject, request model.AuthorizationRequest) model.AuthorizationDecision {
	now := e.now().UTC().Truncate(time.Second)
	target := model.DecisionTarget{
		Operation: request.Operation, Resource: request.Resource,
		Audience: request.Audience, Scope: request.Scope,
	}
	decisionSubject := model.DecisionSubject{ID: subject.ID, Kind: subject.Kind, Authority: subject.Authority}
	decision := model.AuthorizationDecision{
		Schema: decisionSchema, RequestID: request.RequestID, CorrelationID: request.CorrelationID,
		TOPSID: e.topsID, Subject: decisionSubject, Target: target,
		Effect: "deny", ReasonCode: "symphony.ssiag.policy.no-matching-grant",
		PolicyDigest: e.policyDigest, ConfigDigest: e.configDigest,
		DecidedAt: now, CallerClassUsed: false, CanonicalApply: false,
	}

	grants := append([]config.AuthorizationGrant(nil), e.policy.Grants...)
	sort.Slice(grants, func(i, j int) bool { return grants[i].ID < grants[j].ID })
	for _, grant := range grants {
		if grant.SubjectID != subject.ID || grant.Operation != request.Operation ||
			grant.Resource != request.Resource || grant.Audience != request.Audience ||
			grant.Scope != request.Scope {
			continue
		}
		expires := now.Add(time.Duration(e.policy.MaxCapabilitySeconds) * time.Second)
		if request.RequestedExpiresAt.Before(expires) {
			expires = request.RequestedExpiresAt.UTC().Truncate(time.Second)
		}
		basis := grant.AuthorityBasis
		capability := model.Capability{
			Protocol: capabilitySchema, Subject: decisionSubject, TOPSID: e.topsID,
			Target: target, AuthorityBasis: basis, GrantID: grant.ID,
			RequestID: request.RequestID, CorrelationID: request.CorrelationID,
			IssuedAt: now, ExpiresAt: expires, PolicyDigest: e.policyDigest,
			ConfigDigest: e.configDigest, Transferable: false, CanonicalApply: false,
		}
		capability.BindingDigest = taggedDigest([]byte(capabilityBinding(capability)))
		capability.CapabilityID = "ssiag-capability:" + strings.TrimPrefix(capability.BindingDigest, "sha256:")
		decision.Effect = "allow"
		decision.ReasonCode = "symphony.ssiag.policy.exact-grant"
		decision.AuthorityBasis = &basis
		decision.Capability = &capability
		decision.ExpiresAt = &expires
		break
	}
	decisionBytes, _ := json.Marshal(struct {
		RequestID     string                `json:"request_id"`
		CorrelationID string                `json:"correlation_id"`
		TOPSID        string                `json:"tops_id"`
		Subject       model.DecisionSubject `json:"subject"`
		Target        model.DecisionTarget  `json:"target"`
		Effect        string                `json:"effect"`
		ReasonCode    string                `json:"reason_code"`
		PolicyDigest  string                `json:"policy_digest"`
		ConfigDigest  string                `json:"config_digest"`
		DecidedAt     time.Time             `json:"decided_at"`
	}{decision.RequestID, decision.CorrelationID, decision.TOPSID, decision.Subject,
		decision.Target, decision.Effect, decision.ReasonCode, decision.PolicyDigest,
		decision.ConfigDigest, decision.DecidedAt})
	decision.DecisionID = "ssiag-decision:" + strings.TrimPrefix(taggedDigest(decisionBytes), "sha256:")
	return decision
}

func capabilityBinding(capability model.Capability) string {
	return strings.Join([]string{
		capability.Protocol, capability.Subject.ID, capability.Subject.Kind,
		capability.Subject.Authority, capability.TOPSID,
		capability.Target.Operation, capability.Target.Resource,
		capability.Target.Audience, capability.Target.Scope,
		capability.AuthorityBasis, capability.GrantID, capability.RequestID,
		capability.CorrelationID, capability.IssuedAt.Format(time.RFC3339),
		capability.ExpiresAt.Format(time.RFC3339), capability.PolicyDigest,
		capability.ConfigDigest, "transferable=false", "canonical_apply=false",
	}, "\n")
}

func taggedDigest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
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

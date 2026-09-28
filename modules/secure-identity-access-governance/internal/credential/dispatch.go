package credential

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync/atomic"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/policy"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

// DispatchIntent has no caller-supplied subject, digest, receipt or lease.
// Preparing an attempt grants no authority and performs no provider operation.
type DispatchIntent struct {
	Authorization model.AuthorizationRequest
	Reference     Reference
	MaximumBytes  uint32
}

type DispatchStatus string

const (
	DispatchInvalid         DispatchStatus = "invalid"
	DispatchSpent           DispatchStatus = "already_attempted"
	DispatchUnauthenticated DispatchStatus = "unauthenticated"
	DispatchDenied          DispatchStatus = "denied"
	DispatchAuditFailed     DispatchStatus = "audit_failed"
	DispatchUnavailable     DispatchStatus = "unavailable"
	DispatchCancelled       DispatchStatus = "cancelled"
	DispatchDelivered       DispatchStatus = "delivered"
	DispatchRefused         DispatchStatus = "refused"
	DispatchIndeterminate   DispatchStatus = "indeterminate"
)

// ProviderAdmission is trusted owner code, not an adapter response or a caller
// assertion. Pin must verify executable/connection identity and the current
// exact resource-to-reference mapping, provider binding and lease generation.
// It must hold that state stable until Release and must not access key material.
// No operational implementation is installed by this package.
type ProviderAdmission interface {
	Pin(context.Context, peerauth.Peer, DispatchIntent) (PinnedUse, error)
}

type PinnedEvidence struct {
	TOPSID                string
	Subject               model.DecisionSubject
	Peer                  peerauth.Credentials
	Target                model.DecisionTarget
	Reference             Reference
	ConsumerDigest        string
	ProviderBindingDigest string
	LeaseID               string
	MaximumBytes          uint32
	ExpiresAt             time.Time
}

// PinnedUse owns the authenticated recipient/channel lifetime, not just its PID.
// Execute must enforce the complete binding, byte ceiling and context timeout,
// and must return indeterminate after ambiguous delivery. Release closes its
// handles on every path. Neither method may reenter the policy engine. These
// methods carry no credential bytes through the dispatcher's Go interfaces.
type PinnedUse interface {
	Evidence() PinnedEvidence
	Execute(context.Context, UseBinding) DispatchStatus
	Release()
}

type Dispatcher struct {
	policy   *policy.Engine
	audit    *stavproducer.Producer
	provider ProviderAdmission
	now      func() time.Time
}

var errDispatchUnavailable = errors.New("credential dispatch unavailable")

func NewDispatcher(p *policy.Engine, audit *stavproducer.Producer, provider ProviderAdmission) (*Dispatcher, error) {
	if p == nil || audit == nil || provider == nil {
		return nil, errDispatchUnavailable
	}
	return &Dispatcher{policy: p, audit: audit, provider: provider, now: time.Now}, nil
}

// DispatchAttempt is private process-local state. Do not copy it. A future
// durable request journal must control creation/recreation across requests and
// restarts; this type supplies no global request-ID deduplication or lease issuer.
type DispatchAttempt struct {
	owner     *Dispatcher
	intent    DispatchIntent
	attempted atomic.Bool
}

func (d *Dispatcher) Prepare(intent DispatchIntent) (*DispatchAttempt, error) {
	if d == nil || d.now == nil || d.policy == nil || d.audit == nil || d.provider == nil {
		return nil, errDispatchUnavailable
	}
	now := d.now().UTC()
	if policy.ValidateRequest(intent.Authorization, now) != nil ||
		intent.Authorization.RequestedExpiresAt.Sub(now) > MaxUseLifetime ||
		intent.MaximumBytes == 0 || intent.MaximumBytes > MaxUseBytes {
		return nil, errDispatchUnavailable
	}
	for _, v := range [...]string{intent.Reference.Provider, intent.Reference.Name, intent.Reference.Version, intent.Reference.Type} {
		if !useToken(v) {
			return nil, errDispatchUnavailable
		}
	}
	return &DispatchAttempt{owner: d, intent: intent}, nil
}

func (a *DispatchAttempt) Run(ctx context.Context) DispatchStatus {
	if a == nil {
		return DispatchInvalid
	}
	if !a.attempted.CompareAndSwap(false, true) {
		return DispatchSpent
	}
	if a.owner == nil || ctx == nil {
		return DispatchInvalid
	}
	if ctx.Err() != nil {
		return DispatchCancelled
	}
	peer, err := peerauth.PeerFromContext(ctx)
	if err != nil || !peer.Mapped {
		return DispatchUnauthenticated
	}
	d := a.owner
	now := d.now().UTC()
	remaining := a.intent.Authorization.RequestedExpiresAt.Sub(now)
	if remaining <= 0 || remaining > MaxUseLifetime {
		return DispatchInvalid
	}
	// Go arms a duration timer for this absolute deadline. Keep the caller's
	// exact deadline; recomputing now+remaining here could extend it slightly.
	bounded, cancel := context.WithDeadline(ctx, a.intent.Authorization.RequestedExpiresAt)
	defer cancel()
	result := DispatchUnavailable
	err = d.policy.WithDecision(bounded, peer.Subject, a.intent.Authorization, func(decision model.AuthorizationDecision) error {
		result = d.authorized(bounded, peer, a.intent, decision)
		return nil
	})
	if err != nil {
		if bounded.Err() != nil {
			return DispatchCancelled
		}
		return DispatchUnavailable
	}
	return result
}

func (d *Dispatcher) authorized(ctx context.Context, peer peerauth.Peer, intent DispatchIntent, decision model.AuthorizationDecision) DispatchStatus {
	outcome := "denied"
	if decision.Effect == "allow" {
		outcome = "allowed"
	}
	receipt, err := d.audit.Submit(ctx, stavproducer.Record{
		Kind: stavproducer.PolicyDecision, RequestID: decision.RequestID, CorrelationID: decision.CorrelationID,
		Actor:          stav.SafeReference{ID: decision.Subject.ID, Kind: decision.Subject.Kind},
		Authentication: stav.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"},
		Target:         stav.SafeReference{ID: decision.Target.Resource, Kind: "symphony.ssiag.resource"}, Outcome: outcome,
		Configuration: stav.Configuration{PreviousDigest: decision.ConfigDigest, NewDigest: decision.ConfigDigest, State: "digests"},
		TROG:          stav.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}, Classification: "administrative_metadata",
	})
	if err != nil || receipt.TOPSID != decision.TOPSID || receipt.RequestID != decision.RequestID {
		return DispatchAuditFailed
	}
	if decision.Effect != "allow" || decision.Capability == nil {
		return DispatchDenied
	}
	if ctx.Err() != nil {
		return DispatchCancelled
	}
	pin, err := d.provider.Pin(ctx, peer, intent)
	// Even a failed Pin may own cleanup resources; never silently leak them.
	if pin != nil {
		defer pin.Release()
	}
	if err != nil || pin == nil {
		return DispatchUnavailable
	}
	if ctx.Err() != nil {
		return DispatchCancelled
	}
	evidence := pin.Evidence()
	if evidence.TOPSID != decision.TOPSID || evidence.Subject != decision.Subject || evidence.Peer != peer.Credentials ||
		evidence.Target != decision.Target || evidence.Reference != intent.Reference || evidence.MaximumBytes < intent.MaximumBytes ||
		evidence.ExpiresAt.IsZero() || evidence.ExpiresAt.Location() != time.UTC {
		return DispatchRefused
	}
	issued := d.now().UTC()
	deadline := decision.Capability.ExpiresAt
	if evidence.ExpiresAt.Before(deadline) {
		deadline = evidence.ExpiresAt
	}
	if callerDeadline, ok := ctx.Deadline(); ok && callerDeadline.Before(deadline) {
		deadline = callerDeadline.UTC()
	}
	encoded, err := stav.EncodeReceipt(receipt)
	if err != nil {
		return DispatchAuditFailed
	}
	sum := sha256.Sum256(encoded) // committed nonsecret receipt, never credential material
	binding := UseBinding{
		TOPSID: decision.TOPSID, SubjectID: decision.Subject.ID, SubjectKind: decision.Subject.Kind, SubjectAuthority: decision.Subject.Authority,
		ConsumerDigest: evidence.ConsumerDigest, ProviderBindingDigest: evidence.ProviderBindingDigest,
		PolicyDigest: decision.PolicyDigest, ConfigDigest: decision.ConfigDigest, AuthorizationDigest: decision.Capability.BindingDigest,
		AuditReceiptDigest: "sha256:" + hex.EncodeToString(sum[:]), LeaseID: evidence.LeaseID,
		RequestID: decision.RequestID, CorrelationID: decision.CorrelationID, Reference: evidence.Reference,
		Operation: decision.Target.Operation, Resource: decision.Target.Resource, Audience: decision.Target.Audience, Scope: decision.Target.Scope,
		MaximumBytes: intent.MaximumBytes, IssuedAt: issued, Deadline: deadline,
	}
	use, status := NewOneShotUse(binding, issued)
	if status != UseOK {
		return DispatchRefused
	}
	deliveryCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if use.Consume(deliveryCtx, binding, d.now().UTC()) != UseOK {
		return DispatchRefused
	}
	result := pin.Execute(deliveryCtx, binding)
	// Once Execute starts, cancellation/error cannot prove that no bytes escaped.
	if deliveryCtx.Err() != nil {
		return DispatchIndeterminate
	}
	switch result {
	case DispatchDelivered, DispatchRefused:
		return result
	default:
		return DispatchIndeterminate
	}
}

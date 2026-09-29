package credential

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
)

// AdmissionGrant is nonsecret, owner-verified active state, not a bearer grant
// or a lease issuance request. Only trusted SSIAG owner code may publish it.
// Subject/target select one current exact reference; no generation fallback.
type AdmissionGrant struct {
	Subject                                        model.DecisionSubject
	UID, GID                                       uint32
	Target                                         model.DecisionTarget
	Reference                                      Reference
	ConsumerDigest, ProviderBindingDigest, LeaseID string
	MaximumBytes                                   uint32
	NotBefore, ExpiresAt                           time.Time
}

type admissionKey struct {
	Subject model.DecisionSubject
	Target  model.DecisionTarget
}

var errAdmission = errors.New("credential admission unavailable")

type AdmissionUpdate string

const (
	AdmissionUpdated   AdmissionUpdate = "updated"
	AdmissionInvalid   AdmissionUpdate = "invalid"
	AdmissionBusy      AdmissionUpdate = "busy"
	AdmissionConflict  AdmissionUpdate = "revision_conflict"
	AdmissionCancelled AdmissionUpdate = "cancelled"
)

// AdmissionRegistry holds a process-local snapshot pin across native admission,
// execution and native cleanup. It must not be copied. Native remains responsible
// for actual signed process/connection identity and protected channel enforcement.
// No public route, lease issuer or persistent authority loader is provided here.
type AdmissionRegistry struct {
	mu       sync.RWMutex
	topsID   string
	revision uint64
	grants   map[admissionKey]AdmissionGrant
	native   ProviderAdmission
	now      func() time.Time
}

func NewAdmissionRegistry(topsID string, native ProviderAdmission) (*AdmissionRegistry, error) {
	if paths.ValidateTOPSID(topsID) != nil || native == nil {
		return nil, errAdmission
	}
	return &AdmissionRegistry{topsID: topsID, revision: 1, grants: make(map[admissionKey]AdmissionGrant), native: native, now: time.Now}, nil
}

// Replace is an internal compare-and-swap of a complete active snapshot. Empty
// removes all admission. Busy means NOTHING changed: an admitted use still owns
// a pin. The owner must not report a rotation/revocation committed on this result.
// Revision starts at 1, advances even for identical state and never wraps. This
// counter is only meaningful within this registry's lifetime, not across restart.
func (r *AdmissionRegistry) Replace(ctx context.Context, expected uint64, grants []AdmissionGrant) (uint64, AdmissionUpdate) {
	if r == nil || r.now == nil || ctx == nil || expected == 0 {
		return 0, AdmissionInvalid
	}
	if ctx.Err() != nil {
		return 0, AdmissionCancelled
	}
	next := make(map[admissionKey]AdmissionGrant, len(grants))
	leases := make(map[string]struct{}, len(grants))
	now := r.now().UTC()
	for _, g := range grants {
		if ctx.Err() != nil {
			return 0, AdmissionCancelled
		}
		if !validAdmissionGrant(g) || !now.Before(g.ExpiresAt) {
			return 0, AdmissionInvalid
		}
		key := admissionKey{g.Subject, g.Target}
		if _, ok := next[key]; ok {
			return 0, AdmissionInvalid
		}
		if _, ok := leases[g.LeaseID]; ok {
			return 0, AdmissionInvalid
		}
		g.NotBefore = g.NotBefore.Round(0)
		g.ExpiresAt = g.ExpiresAt.Round(0)
		next[key] = g
		leases[g.LeaseID] = struct{}{}
	}
	if !r.mu.TryLock() {
		return 0, AdmissionBusy
	}
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		return r.revision, AdmissionCancelled
	}
	if expected != r.revision {
		return r.revision, AdmissionConflict
	}
	if r.revision == math.MaxUint64 {
		return r.revision, AdmissionInvalid
	}
	// Validation may have taken time; do not publish already expired leases.
	now = r.now().UTC()
	for _, g := range next {
		if !now.Before(g.ExpiresAt) {
			return r.revision, AdmissionInvalid
		}
	}
	r.grants = next
	r.revision++
	return r.revision, AdmissionUpdated
}

func validAdmissionGrant(g AdmissionGrant) bool {
	if g.Subject.Authority != peerauth.Mechanism || !useDigest(g.ConsumerDigest) || !useDigest(g.ProviderBindingDigest) ||
		g.MaximumBytes == 0 || g.MaximumBytes > MaxUseBytes || g.NotBefore.IsZero() || g.ExpiresAt.IsZero() ||
		g.NotBefore.Location() != time.UTC || g.ExpiresAt.Location() != time.UTC || !g.ExpiresAt.After(g.NotBefore) {
		return false
	}
	for _, v := range []string{g.Subject.ID, g.Subject.Kind, g.Target.Operation, g.Target.Resource, g.Target.Audience, g.Target.Scope,
		g.Reference.Provider, g.Reference.Name, g.Reference.Version, g.Reference.Type, g.LeaseID} {
		if !useToken(v) {
			return false
		}
	}
	return true
}

func (r *AdmissionRegistry) Pin(ctx context.Context, peer peerauth.Peer, in DispatchIntent) (PinnedUse, error) {
	if r == nil || r.native == nil || r.now == nil || ctx == nil || ctx.Err() != nil || !intentValid(in) {
		return nil, errAdmission
	}
	actual, err := peerauth.PeerFromContext(ctx)
	if err != nil || !actual.Mapped || !peer.Mapped || actual.Credentials != peer.Credentials ||
		actual.Subject.ID != peer.Subject.ID || actual.Subject.Kind != peer.Subject.Kind || actual.Subject.Authority != peer.Subject.Authority ||
		len(actual.Subject.Attributes) != 0 || len(peer.Subject.Attributes) != 0 {
		return nil, errAdmission
	}
	if !r.mu.TryRLock() {
		return nil, errAdmission
	}
	retained := false
	defer func() {
		if !retained {
			r.mu.RUnlock()
		}
	}()
	subject := model.DecisionSubject{ID: peer.Subject.ID, Kind: peer.Subject.Kind, Authority: peer.Subject.Authority}
	a := in.Authorization
	target := model.DecisionTarget{Operation: a.Operation, Resource: a.Resource, Audience: a.Audience, Scope: a.Scope}
	g, ok := r.grants[admissionKey{subject, target}]
	now := r.now().UTC()
	if !ok || peer.Credentials.UID != g.UID || peer.Credentials.GID != g.GID || in.Reference != g.Reference ||
		in.MaximumBytes > g.MaximumBytes || now.Before(g.NotBefore) || !now.Before(g.ExpiresAt) ||
		now.Before(a.RequestedAt) || !now.Before(a.RequestedExpiresAt) {
		return nil, errAdmission
	}
	deadline := a.RequestedExpiresAt
	if g.ExpiresAt.Before(deadline) {
		deadline = g.ExpiresAt
	}
	bounded, cancel := context.WithDeadline(ctx, deadline)
	pin, err := r.native.Pin(bounded, peer, in)
	accepted := false
	defer func() {
		if !accepted {
			cancel()
			if pin != nil {
				pin.Release()
			}
		}
	}()
	if err != nil || pin == nil || bounded.Err() != nil {
		return nil, errAdmission
	}
	e := pin.Evidence()
	now = r.now().UTC()
	if e.TOPSID != r.topsID || e.Subject != g.Subject || e.Peer != peer.Credentials || e.Target != g.Target ||
		e.Reference != g.Reference || e.ConsumerDigest != g.ConsumerDigest || e.ProviderBindingDigest != g.ProviderBindingDigest ||
		e.LeaseID != g.LeaseID || e.MaximumBytes < in.MaximumBytes || e.MaximumBytes > g.MaximumBytes ||
		e.ExpiresAt.Location() != time.UTC || !now.Before(e.ExpiresAt) || e.ExpiresAt.After(g.ExpiresAt) || now.Before(g.NotBefore) {
		return nil, errAdmission
	}
	accepted = true
	retained = true
	return &admissionPin{registry: r, native: pin, evidence: e, grant: g, intent: in, ctx: bounded, cancel: cancel}, nil
}

type admissionPin struct {
	registry  *AdmissionRegistry
	native    PinnedUse
	evidence  PinnedEvidence
	grant     AdmissionGrant
	intent    DispatchIntent
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	attempted atomic.Bool
	released  bool
}

func (p *admissionPin) Evidence() PinnedEvidence { return p.evidence }
func (p *admissionPin) Release() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.released {
		return
	}
	p.released = true
	defer p.registry.mu.RUnlock()
	defer p.cancel()
	p.native.Release()
}
func (p *admissionPin) Execute(ctx context.Context, b UseBinding) DispatchStatus {
	if !p.attempted.CompareAndSwap(false, true) {
		return DispatchSpent
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.released || ctx == nil || ctx.Err() != nil || p.ctx.Err() != nil {
		return DispatchRefused
	}
	now := p.registry.now().UTC()
	e, a := p.evidence, p.intent.Authorization
	if !validUse(b, now) || !now.Before(b.Deadline) || b.IssuedAt.Before(p.grant.NotBefore) ||
		b.TOPSID != e.TOPSID || b.SubjectID != e.Subject.ID || b.SubjectKind != e.Subject.Kind || b.SubjectAuthority != e.Subject.Authority ||
		b.ConsumerDigest != e.ConsumerDigest || b.ProviderBindingDigest != e.ProviderBindingDigest || b.LeaseID != e.LeaseID ||
		b.Reference != e.Reference || b.Operation != e.Target.Operation || b.Resource != e.Target.Resource || b.Audience != e.Target.Audience || b.Scope != e.Target.Scope ||
		b.RequestID != a.RequestID || b.CorrelationID != a.CorrelationID || b.MaximumBytes != p.intent.MaximumBytes ||
		b.Deadline.After(e.ExpiresAt) || b.Deadline.After(a.RequestedExpiresAt) || p.native.Evidence() != e {
		return DispatchRefused
	}
	bounded, cancel := context.WithDeadline(ctx, b.Deadline)
	stop := context.AfterFunc(p.ctx, cancel)
	defer func() { stop(); cancel() }()
	if bounded.Err() != nil || p.ctx.Err() != nil {
		return DispatchRefused
	}
	result := p.native.Execute(bounded, b)
	if bounded.Err() != nil || p.ctx.Err() != nil {
		return DispatchIndeterminate
	}
	switch result {
	case DispatchDelivered, DispatchRefused:
		return result
	default:
		return DispatchIndeterminate
	}
}

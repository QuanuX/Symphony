//go:build darwin || linux

package credential

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
)

func admissionFixture(t *testing.T) (*AdmissionRegistry, *Dispatcher, DispatchIntent, *dispatchProvider, context.Context, AdmissionGrant) {
	t.Helper()
	d, in, n, _, _ := dispatchFixture(t)
	ctx := dispatchPeer(t, true)
	peer, err := peerauth.PeerFromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	g := AdmissionGrant{Subject: model.DecisionSubject{ID: peer.Subject.ID, Kind: peer.Subject.Kind, Authority: peer.Subject.Authority},
		UID: peer.Credentials.UID, GID: peer.Credentials.GID, Target: model.DecisionTarget{Operation: in.Authorization.Operation, Resource: in.Authorization.Resource, Audience: in.Authorization.Audience, Scope: in.Authorization.Scope},
		Reference: in.Reference, ConsumerDigest: fixtureDigest, ProviderBindingDigest: fixtureDigest, LeaseID: "fixture-lease", MaximumBytes: 32,
		NotBefore: in.Authorization.RequestedAt, ExpiresAt: in.Authorization.RequestedExpiresAt}
	r, err := NewAdmissionRegistry(dispatchTOPS, n)
	if err != nil {
		t.Fatal(err)
	}
	if rev, status := r.Replace(ctx, 1, []AdmissionGrant{g}); rev != 2 || status != AdmissionUpdated {
		t.Fatal(rev, status)
	}
	d.provider = r
	return r, d, in, n, ctx, g
}
func admissionAcquire(t *testing.T, r *AdmissionRegistry, ctx context.Context, in DispatchIntent) PinnedUse {
	t.Helper()
	peer, err := peerauth.PeerFromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := r.Pin(ctx, peer, in)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pin.Release)
	return pin
}
func admissionBinding(in DispatchIntent, g AdmissionGrant) UseBinding {
	return UseBinding{TOPSID: dispatchTOPS, SubjectID: g.Subject.ID, SubjectKind: g.Subject.Kind, SubjectAuthority: g.Subject.Authority,
		ConsumerDigest: g.ConsumerDigest, ProviderBindingDigest: g.ProviderBindingDigest, PolicyDigest: fixtureDigest, ConfigDigest: fixtureDigest,
		AuthorizationDigest: fixtureDigest, AuditReceiptDigest: fixtureDigest, LeaseID: g.LeaseID, RequestID: in.Authorization.RequestID, CorrelationID: in.Authorization.CorrelationID,
		Reference: g.Reference, Operation: g.Target.Operation, Resource: g.Target.Resource, Audience: g.Target.Audience, Scope: g.Target.Scope, MaximumBytes: in.MaximumBytes,
		IssuedAt: time.Now().UTC(), Deadline: in.Authorization.RequestedExpiresAt}
}

func TestAdmissionDispatchAndCleanupPin(t *testing.T) {
	r, d, in, n, ctx, _ := admissionFixture(t)
	n.execute = func(_ context.Context, b UseBinding) DispatchStatus {
		if b.LeaseID != "fixture-lease" {
			t.Error("wrong lease")
		}
		if _, s := r.Replace(ctx, 2, nil); s != AdmissionBusy {
			t.Error("mutation crossed executing pin", s)
		}
		return DispatchDelivered
	}
	a, err := d.Prepare(in)
	if err != nil {
		t.Fatal(err)
	}
	if s := a.Run(ctx); s != DispatchDelivered {
		t.Fatal(s)
	}
	if n.executions.Load() != 1 || n.releases.Load() != 1 {
		t.Fatal("native lifecycle")
	}
	if rev, s := r.Replace(ctx, 2, nil); rev != 3 || s != AdmissionUpdated {
		t.Fatal(rev, s)
	}
}
func TestAdmissionSnapshotCASAndOwnedValues(t *testing.T) {
	r, _, in, n, ctx, g := admissionFixture(t)
	grants := []AdmissionGrant{g}
	if rev, s := r.Replace(ctx, 2, grants); rev != 3 || s != AdmissionUpdated {
		t.Fatal(rev, s)
	}
	grants[0].Reference.Version = "mutated"
	pin := admissionAcquire(t, r, ctx, in)
	pin.Release()
	pin.Release()
	if n.releases.Load() != 1 {
		t.Fatal("double release")
	}
	if _, s := r.Replace(ctx, 2, nil); s != AdmissionConflict {
		t.Fatal(s)
	}
	if rev, s := r.Replace(ctx, 3, nil); rev != 4 || s != AdmissionUpdated {
		t.Fatal(rev, s)
	}
	peer, _ := peerauth.PeerFromContext(ctx)
	if p, e := r.Pin(ctx, peer, in); e == nil || p != nil {
		t.Fatal("revoked admission")
	}
	if n.pins.Load() != 1 {
		t.Fatal("revocation reached native")
	}
	// Restoring identical metadata still advances revision, defeating local ABA.
	if rev, s := r.Replace(ctx, 4, []AdmissionGrant{g}); rev != 5 || s != AdmissionUpdated {
		t.Fatal(rev, s)
	}
	if _, s := r.Replace(ctx, 3, nil); s != AdmissionConflict {
		t.Fatal(s)
	}
	r.revision = math.MaxUint64
	if _, s := r.Replace(ctx, math.MaxUint64, nil); s != AdmissionInvalid {
		t.Fatal("wrapped revision")
	}
}
func TestAdmissionRotationExactGeneration(t *testing.T) {
	r, _, in, n, ctx, g := admissionFixture(t)
	pin := admissionAcquire(t, r, ctx, in)
	g.Reference.Version = "generation-2"
	g.LeaseID = "lease-2"
	if _, s := r.Replace(ctx, 2, []AdmissionGrant{g}); s != AdmissionBusy {
		t.Fatal(s)
	}
	pin.Release()
	if _, s := r.Replace(ctx, 2, []AdmissionGrant{g}); s != AdmissionUpdated {
		t.Fatal(s)
	}
	peer, _ := peerauth.PeerFromContext(ctx)
	if p, e := r.Pin(ctx, peer, in); e == nil || p != nil {
		t.Fatal("old generation admitted")
	}
	in.Reference.Version = "generation-2"
	// An unchanged native lease cannot satisfy the newly activated mapping.
	if p, e := r.Pin(ctx, peer, in); e == nil || p != nil {
		t.Fatal("old lease admitted")
	}
	n.edit = func(e *PinnedEvidence) { e.LeaseID = "lease-2" }
	p := admissionAcquire(t, r, ctx, in)
	p.Release()
	if n.pins.Load() != 3 || n.releases.Load() != 3 {
		t.Fatal("pin leak", n.pins.Load(), n.releases.Load())
	}
}
func TestAdmissionRejectsInvalidSnapshot(t *testing.T) {
	edits := map[string]func(*AdmissionGrant){
		"subject": func(g *AdmissionGrant) { g.Subject.ID = "" }, "authority": func(g *AdmissionGrant) { g.Subject.Authority = "caller" },
		"target": func(g *AdmissionGrant) { g.Target.Scope = "*" }, "generation": func(g *AdmissionGrant) { g.Reference.Version = "" },
		"consumer": func(g *AdmissionGrant) { g.ConsumerDigest = "unverified" }, "provider": func(g *AdmissionGrant) { g.ProviderBindingDigest = "unverified" },
		"lease": func(g *AdmissionGrant) { g.LeaseID = "" }, "zero": func(g *AdmissionGrant) { g.MaximumBytes = 0 }, "oversize": func(g *AdmissionGrant) { g.MaximumBytes = MaxUseBytes + 1 },
		"inverted": func(g *AdmissionGrant) { g.ExpiresAt = g.NotBefore }, "expired": func(g *AdmissionGrant) {
			g.ExpiresAt = time.Now().UTC().Add(-time.Hour)
			g.NotBefore = g.ExpiresAt.Add(-time.Hour)
		},
		"zone": func(g *AdmissionGrant) { g.NotBefore = g.NotBefore.In(time.FixedZone("not UTC", 0)) },
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			r, _, _, _, ctx, g := admissionFixture(t)
			edit(&g)
			if _, s := r.Replace(ctx, 2, []AdmissionGrant{g}); s != AdmissionInvalid {
				t.Fatal(s)
			}
			if r.revision != 2 {
				t.Fatal("changed state")
			}
		})
	}
	for _, sameLease := range []bool{false, true} {
		r, _, _, _, ctx, g := admissionFixture(t)
		other := g
		if sameLease {
			other.Target.Resource = "other"
		}
		if _, s := r.Replace(ctx, 2, []AdmissionGrant{g, other}); s != AdmissionInvalid {
			t.Fatal("ambiguous snapshot")
		}
	}
}
func TestAdmissionRefusesBeforeNative(t *testing.T) {
	for _, name := range []string{"unmapped", "no-context", "forged-peer", "uid", "gid", "subject", "target", "reference", "bytes", "not-yet-valid", "expired", "cancelled", "writer"} {
		t.Run(name, func(t *testing.T) {
			r, _, in, n, ctx, g := admissionFixture(t)
			peer, _ := peerauth.PeerFromContext(ctx)
			switch name {
			case "unmapped":
				peer.Mapped = false
			case "no-context":
				ctx = context.Background()
			case "forged-peer":
				peer.Credentials.PID++
			case "uid":
				g.UID++
			case "gid":
				g.GID++
			case "subject":
				g.Subject.Kind = "different-kind"
			case "target":
				g.Target.Audience = "different-audience"
			case "reference":
				in.Reference.Version = "absent"
			case "bytes":
				in.MaximumBytes++
			case "not-yet-valid":
				g.NotBefore = time.Now().UTC().Add(time.Second)
			case "expired":
				r.now = func() time.Time { return g.ExpiresAt }
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "writer":
				r.mu.Lock()
				defer r.mu.Unlock()
			}
			if name == "uid" || name == "gid" || name == "subject" || name == "target" || name == "not-yet-valid" {
				if _, s := r.Replace(ctx, 2, []AdmissionGrant{g}); s != AdmissionUpdated {
					t.Fatal(s)
				}
			}
			if p, e := r.Pin(ctx, peer, in); e == nil || p != nil {
				t.Fatal("admitted", name)
			}
			if n.pins.Load() != 0 {
				t.Fatal("native reached")
			}
		})
	}
}
func TestAdmissionNativeEvidenceAndErrorCleanup(t *testing.T) {
	edits := map[string]func(*PinnedEvidence){
		"tops": func(e *PinnedEvidence) { e.TOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120003" }, "subject": func(e *PinnedEvidence) { e.Subject.ID = "other" },
		"peer": func(e *PinnedEvidence) { e.Peer.PID++ }, "target": func(e *PinnedEvidence) { e.Target.Operation = "other" }, "reference": func(e *PinnedEvidence) { e.Reference.Version = "other" },
		"consumer": func(e *PinnedEvidence) {
			e.ConsumerDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		},
		"provider": func(e *PinnedEvidence) {
			e.ProviderBindingDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		},
		"lease": func(e *PinnedEvidence) { e.LeaseID = "other" }, "too-small": func(e *PinnedEvidence) { e.MaximumBytes-- }, "too-large": func(e *PinnedEvidence) { e.MaximumBytes++ },
		"extended": func(e *PinnedEvidence) { e.ExpiresAt = e.ExpiresAt.Add(time.Second) }, "expired": func(e *PinnedEvidence) { e.ExpiresAt = time.Now().UTC().Add(-time.Second) },
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			r, _, in, n, ctx, _ := admissionFixture(t)
			n.edit = edit
			peer, _ := peerauth.PeerFromContext(ctx)
			if p, e := r.Pin(ctx, peer, in); e == nil || p != nil {
				t.Fatal("admitted drift")
			}
			if n.releases.Load() != 1 {
				t.Fatal("leaked native")
			}
			if _, s := r.Replace(ctx, 2, nil); s != AdmissionUpdated {
				t.Fatal("leaked registry")
			}
		})
	}
	r, _, in, n, ctx, _ := admissionFixture(t)
	n.err = true
	peer, _ := peerauth.PeerFromContext(ctx)
	if p, e := r.Pin(ctx, peer, in); e != errAdmission || p != nil {
		t.Fatal("raw error or pin escaped")
	}
	if n.releases.Load() != 1 {
		t.Fatal("error pin leaked")
	}
}
func TestAdmissionExecuteRechecksAndSpends(t *testing.T) {
	for _, name := range []string{"request", "correlation", "bytes", "deadline", "lease", "released", "cancelled", "parent-cancelled", "evidence-drift", "expired"} {
		t.Run(name, func(t *testing.T) {
			r, _, in, n, ctx, g := admissionFixture(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			pin := admissionAcquire(t, r, ctx, in)
			b := admissionBinding(in, g)
			executeCtx := context.Background()
			switch name {
			case "request":
				b.RequestID = "different"
			case "correlation":
				b.CorrelationID = "different"
			case "bytes":
				b.MaximumBytes--
			case "deadline":
				b.Deadline = b.Deadline.Add(time.Second)
			case "lease":
				b.LeaseID = "other"
			case "released":
				pin.Release()
			case "cancelled":
				var stop context.CancelFunc
				executeCtx, stop = context.WithCancel(executeCtx)
				stop()
			case "parent-cancelled":
				cancel()
			case "evidence-drift":
				pin.(*admissionPin).native.(*dispatchPin).e.ConsumerDigest = "changed"
			case "expired":
				r.now = func() time.Time { return b.Deadline }
			}
			if s := pin.Execute(executeCtx, b); s != DispatchRefused {
				t.Fatal(s)
			}
			if s := pin.Execute(context.Background(), admissionBinding(in, g)); s != DispatchSpent {
				t.Fatal(s)
			}
			if n.executions.Load() != 0 {
				t.Fatal("native execution reached")
			}
		})
	}
}
func TestAdmissionConcurrentExecuteReleaseAndReplacement(t *testing.T) {
	r, _, in, n, ctx, g := admissionFixture(t)
	pin := admissionAcquire(t, r, ctx, in)
	b := admissionBinding(in, g)
	entered, finish := make(chan struct{}), make(chan struct{})
	n.execute = func(context.Context, UseBinding) DispatchStatus { close(entered); <-finish; return DispatchDelivered }
	done := make(chan DispatchStatus, 1)
	go func() { done <- pin.Execute(ctx, b) }()
	<-entered
	released := make(chan struct{})
	go func() { pin.Release(); close(released) }()
	var wg sync.WaitGroup
	var spent atomic.Int32
	for i := 0; i < 32; i++ {
		wg.Go(func() {
			if pin.Execute(ctx, b) == DispatchSpent {
				spent.Add(1)
			}
		})
	}
	wg.Wait()
	if spent.Load() != 32 {
		t.Fatal("duplicate execute")
	}
	if _, s := r.Replace(ctx, 2, nil); s != AdmissionBusy {
		t.Fatal("released while executing")
	}
	close(finish)
	if s := <-done; s != DispatchDelivered {
		t.Fatal(s)
	}
	<-released
	if n.executions.Load() != 1 || n.releases.Load() != 1 {
		t.Fatal("lifecycle")
	}
	if _, s := r.Replace(ctx, 2, nil); s != AdmissionUpdated {
		t.Fatal(s)
	}
}
func TestAdmissionCancellationDuringExecutionAndUnknownOutcome(t *testing.T) {
	for _, cancelDuring := range []bool{false, true} {
		r, _, in, n, ctx, g := admissionFixture(t)
		ctx, cancel := context.WithCancel(ctx)
		pin := admissionAcquire(t, r, ctx, in)
		n.execute = func(c context.Context, _ UseBinding) DispatchStatus {
			if cancelDuring {
				cancel()
				<-c.Done()
				return DispatchDelivered
			}
			return DispatchUnavailable
		}
		if s := pin.Execute(context.Background(), admissionBinding(in, g)); s != DispatchIndeterminate {
			t.Fatal(s)
		}
		pin.Release()
		cancel()
	}
}
func TestAdmissionSnapshotConcurrentCAS(t *testing.T) {
	r, _, _, _, ctx, g := admissionFixture(t)
	var wg sync.WaitGroup
	var updated atomic.Int32
	for i := 0; i < 32; i++ {
		wg.Go(func() {
			_, s := r.Replace(ctx, 2, []AdmissionGrant{g})
			switch s {
			case AdmissionUpdated:
				updated.Add(1)
			case AdmissionBusy, AdmissionConflict:
			default:
				t.Error(s)
			}
		})
	}
	wg.Wait()
	if updated.Load() != 1 || r.revision != 3 {
		t.Fatal("CAS did not serialize")
	}
}

type admissionNativeFunc func(context.Context, peerauth.Peer, DispatchIntent) (PinnedUse, error)

func (f admissionNativeFunc) Pin(c context.Context, p peerauth.Peer, i DispatchIntent) (PinnedUse, error) {
	return f(c, p, i)
}

type admissionReleaseHook struct {
	PinnedUse
	before func()
}

func (p *admissionReleaseHook) Release() { p.before(); p.PinnedUse.Release() }

func TestAdmissionHoldsSnapshotThroughNativeCleanup(t *testing.T) {
	r, _, in, n, ctx, _ := admissionFixture(t)
	r.native = admissionNativeFunc(func(c context.Context, p peerauth.Peer, i DispatchIntent) (PinnedUse, error) {
		pin, e := n.Pin(c, p, i)
		return &admissionReleaseHook{PinnedUse: pin, before: func() {
			if _, s := r.Replace(ctx, 2, nil); s != AdmissionBusy {
				t.Error("cleanup lost snapshot pin", s)
			}
		}}, e
	})
	pin := admissionAcquire(t, r, ctx, in)
	pin.Release()
	if _, s := r.Replace(ctx, 2, nil); s != AdmissionUpdated {
		t.Fatal(s)
	}
}
func TestAdmissionBoundsNativeLifetimeAndCancellation(t *testing.T) {
	r, d, in, n, ctx, g := admissionFixture(t)
	g.ExpiresAt = in.Authorization.RequestedAt.Add(10 * time.Second)
	if _, s := r.Replace(ctx, 2, []AdmissionGrant{g}); s != AdmissionUpdated {
		t.Fatal(s)
	}
	n.edit = func(e *PinnedEvidence) { e.ExpiresAt = g.ExpiresAt }
	n.execute = func(c context.Context, b UseBinding) DispatchStatus {
		deadline, ok := c.Deadline()
		if !ok || !deadline.Equal(g.ExpiresAt) || !b.Deadline.Equal(g.ExpiresAt) {
			t.Error("extended deadline")
		}
		return DispatchDelivered
	}
	var nativeCtx context.Context
	r.native = admissionNativeFunc(func(c context.Context, p peerauth.Peer, i DispatchIntent) (PinnedUse, error) {
		nativeCtx = c
		deadline, ok := c.Deadline()
		if !ok || !deadline.Equal(g.ExpiresAt) {
			t.Error("unbounded native pin")
		}
		return n.Pin(c, p, i)
	})
	attempt, err := d.Prepare(in)
	if err != nil {
		t.Fatal(err)
	}
	if s := attempt.Run(ctx); s != DispatchDelivered {
		t.Fatal(s)
	}
	if nativeCtx.Err() == nil {
		t.Fatal("native context retained after release")
	}
}
func TestAdmissionCancelledPinCleansBothOwners(t *testing.T) {
	r, _, in, n, ctx, _ := admissionFixture(t)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.native = admissionNativeFunc(func(c context.Context, p peerauth.Peer, i DispatchIntent) (PinnedUse, error) {
		pin, e := n.Pin(c, p, i)
		cancel()
		return pin, e
	})
	peer, _ := peerauth.PeerFromContext(ctx)
	if p, e := r.Pin(ctx, peer, in); e == nil || p != nil {
		t.Fatal("cancelled admission")
	}
	if n.releases.Load() != 1 {
		t.Fatal("native leaked")
	}
	if _, s := r.Replace(context.Background(), 2, nil); s != AdmissionUpdated {
		t.Fatal("registry leaked")
	}
}
func TestAdmissionEmptyAndCancelledUpdate(t *testing.T) {
	r, _, in, n, ctx, _ := admissionFixture(t)
	empty, e := NewAdmissionRegistry(dispatchTOPS, n)
	if e != nil {
		t.Fatal(e)
	}
	peer, _ := peerauth.PeerFromContext(ctx)
	if pin, e := empty.Pin(ctx, peer, in); e == nil || pin != nil {
		t.Fatal("empty registry allowed")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, s := r.Replace(cancelled, 2, nil); s != AdmissionCancelled {
		t.Fatal(s)
	}
	if r.revision != 2 {
		t.Fatal("cancelled mutation")
	}
	if _, e := NewAdmissionRegistry("bad", n); e == nil {
		t.Fatal("bad TOPS")
	}
	if _, e := NewAdmissionRegistry(dispatchTOPS, nil); e == nil {
		t.Fatal("missing native backend")
	}
}

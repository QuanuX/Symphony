//go:build darwin || linux

package credential

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/policy"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

const dispatchTOPS = "018f0c3a-7b2d-7e11-8c12-0242ac120002"
const fixtureDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type dispatchAudit struct {
	calls  atomic.Int32
	mutate func(*stav.LocalResponse)
	after  func()
}

func (a *dispatchAudit) Do(_ context.Context, r stav.LocalRequest) (stav.LocalResponse, error) {
	a.calls.Add(1)
	digest, _ := stav.CandidateDigest(*r.Candidate)
	receipt := &stav.Receipt{Schema: stav.SchemaReceipt, TOPSID: r.TOPSID, RequestID: r.RequestID, CandidateDigest: digest,
		Disposition: "committed", ReasonCode: stav.ReasonReceiptCommitted,
		Commit: stav.CommitResult{State: "committed", Sequence: 1, EventDigest: fixtureDigest, EventID: "b90e1205-1b3b-4e47-9b91-1cd624cd87cd", Timestamp: "2026-09-27T23:00:00.000000001Z"}}
	response := stav.LocalResponse{Schema: stav.SchemaLocalResponse, TOPSID: r.TOPSID, RequestID: r.RequestID, Operation: r.Operation, Disposition: stav.LocalDispositionSucceeded, ReasonCode: stav.ReasonResponseSucceeded, Receipt: receipt}
	if a.mutate != nil {
		a.mutate(&response)
	}
	if a.after != nil {
		a.after()
	}
	return response, nil
}

type dispatchProvider struct {
	pins, executions, releases atomic.Int32
	edit                       func(*PinnedEvidence)
	execute                    func(context.Context, UseBinding) DispatchStatus
	err                        bool
	binding                    UseBinding
}

func (p *dispatchProvider) Pin(_ context.Context, peer peerauth.Peer, in DispatchIntent) (PinnedUse, error) {
	p.pins.Add(1)
	e := PinnedEvidence{TOPSID: dispatchTOPS, Subject: model.DecisionSubject{ID: peer.Subject.ID, Kind: peer.Subject.Kind, Authority: peer.Subject.Authority}, Peer: peer.Credentials,
		Target: model.DecisionTarget{Operation: in.Authorization.Operation, Resource: in.Authorization.Resource, Audience: in.Authorization.Audience, Scope: in.Authorization.Scope}, Reference: in.Reference,
		ConsumerDigest: fixtureDigest, ProviderBindingDigest: fixtureDigest, LeaseID: "fixture-lease", MaximumBytes: in.MaximumBytes, ExpiresAt: in.Authorization.RequestedExpiresAt}
	if p.edit != nil {
		p.edit(&e)
	}
	pin := &dispatchPin{p: p, e: e}
	if p.err {
		return pin, errors.New("private provider detail must not be returned")
	}
	return pin, nil
}

type dispatchPin struct {
	p *dispatchProvider
	e PinnedEvidence
}

func (p *dispatchPin) Evidence() PinnedEvidence { return p.e }
func (p *dispatchPin) Release()                 { p.p.releases.Add(1) }
func (p *dispatchPin) Execute(ctx context.Context, b UseBinding) DispatchStatus {
	p.p.executions.Add(1)
	p.p.binding = b
	if p.p.execute != nil {
		return p.p.execute(ctx, b)
	}
	return DispatchDelivered
}

func dispatchPeer(t *testing.T, mapped bool) context.Context {
	t.Helper()
	// Short socket path is required on Darwin. Failure is a test failure, not a skip.
	dir, err := os.MkdirTemp("", "sqv15-peer-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	ln, err := net.Listen("unix", filepath.Join(dir, "s"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	client, err := net.Dial("unix", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	server, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	mappings := []peerauth.Mapping{}
	if mapped {
		mappings = append(mappings, peerauth.Mapping{SubjectID: "fixture-consumer", SubjectKind: "symphony.identity.service", UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())})
	}
	resolver, err := peerauth.NewResolver(mappings)
	if err != nil {
		t.Fatal(err)
	}
	return peerauth.ContextWithConnection(context.Background(), server, resolver)
}
func dispatchFixture(t *testing.T) (*Dispatcher, DispatchIntent, *dispatchProvider, *dispatchAudit, *config.AuthorizationConfig) {
	t.Helper()
	now := time.Now().UTC()
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	auth := &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 30, Grants: []config.AuthorizationGrant{{ID: "fixture-grant", SubjectID: "fixture-consumer", AuthorityBasis: "granted_permission", Operation: "fixture-use", Resource: "fixture-resource", Audience: "fixture-audience", Scope: "fixture-scope"}}}
	cfg := config.Config{Schema: "symphony.ssiag.config.v1", Mode: "development", TOPS: config.TOPSConfig{ID: dispatchTOPS, Name: "Fixture"}, Listen: config.ListenConfig{Network: "unix", Address: "/tmp/fixture-not-opened.sock"}, Authentication: &config.AuthenticationConfig{Mechanism: peerauth.Mechanism, Subjects: []config.SubjectConfig{{ID: "fixture-consumer", Kind: "symphony.identity.service", UID: &uid, GID: &gid}}}, Authorization: auth, Providers: []config.ProviderConfig{}}
	engine, err := policy.New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	audit := &dispatchAudit{}
	producer, err := stavproducer.New(dispatchTOPS, audit)
	if err != nil {
		t.Fatal(err)
	}
	provider := &dispatchProvider{}
	d, err := NewDispatcher(engine, producer, provider)
	if err != nil {
		t.Fatal(err)
	}
	in := DispatchIntent{Authorization: model.AuthorizationRequest{Schema: "symphony.ssiag.authorization-request.v1", RequestID: "684921d8-a8b5-49da-872b-568eb6a6dc03", CorrelationID: "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287", Operation: "fixture-use", Resource: "fixture-resource", Audience: "fixture-audience", Scope: "fixture-scope", RequestedAt: now, RequestedExpiresAt: now.Add(20 * time.Second)}, Reference: Reference{Provider: "fixture-provider", Name: "fixture-reference", Version: "generation-1", Type: "api-key"}, MaximumBytes: 32}
	return d, in, provider, audit, auth
}
func TestDispatchKernelIdentityAuditAndSingleAttempt(t *testing.T) {
	d, in, p, a, auth := dispatchFixture(t)
	ctx := dispatchPeer(t, true)
	// A later mutation of the caller's config must not mutate the active snapshot.
	auth.Grants[0].Operation = "changed"
	auth.DefaultEffect = "allow"
	attempt, err := d.Prepare(in)
	if err != nil {
		t.Fatal(err)
	}
	in.Reference.Version = "generation-2"
	var wg sync.WaitGroup
	var delivered, spent atomic.Int32
	for i := 0; i < 64; i++ {
		wg.Go(func() {
			switch attempt.Run(ctx) {
			case DispatchDelivered:
				delivered.Add(1)
			case DispatchSpent:
				spent.Add(1)
			default:
				t.Error("unexpected dispatch status")
			}
		})
	}
	wg.Wait()
	if delivered.Load() != 1 || spent.Load() != 63 || a.calls.Load() != 1 || p.executions.Load() != 1 || p.releases.Load() != 1 {
		t.Fatal("single-attempt ordering failed")
	}
	if p.binding.SubjectID != "fixture-consumer" || p.binding.SubjectAuthority != peerauth.Mechanism || p.binding.Reference.Version != "generation-1" || p.binding.AuditReceiptDigest == "" {
		t.Fatal("derived binding mismatch")
	}
	raw, _ := json.Marshal(attempt)
	if string(raw) != "{}" {
		t.Fatal("attempt serialized")
	}
}
func TestDispatchRefusesBeforeProvider(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Dispatcher, *DispatchIntent, *dispatchAudit)
		ctx    func(*testing.T) context.Context
		want   DispatchStatus
	}{
		{"no-peer", nil, func(*testing.T) context.Context { return context.Background() }, DispatchUnauthenticated},
		{"unmapped", nil, func(t *testing.T) context.Context { return dispatchPeer(t, false) }, DispatchUnauthenticated},
		{"denied", func(_ *Dispatcher, in *DispatchIntent, _ *dispatchAudit) {
			in.Authorization.Resource = "other-resource"
		}, nil, DispatchDenied},
		{"receipt-candidate", func(_ *Dispatcher, _ *DispatchIntent, a *dispatchAudit) {
			a.mutate = func(r *stav.LocalResponse) { r.Receipt.CandidateDigest = fixtureDigest }
		}, nil, DispatchAuditFailed},
		{"receipt-tops", func(_ *Dispatcher, _ *DispatchIntent, a *dispatchAudit) {
			a.mutate = func(r *stav.LocalResponse) {
				r.TOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120003"
				r.Receipt.TOPSID = r.TOPSID
			}
		}, nil, DispatchAuditFailed},
		{"receipt-request", func(_ *Dispatcher, _ *DispatchIntent, a *dispatchAudit) {
			a.mutate = func(r *stav.LocalResponse) {
				r.RequestID = "684921d8-a8b5-49da-872b-568eb6a6dc04"
				r.Receipt.RequestID = r.RequestID
			}
		}, nil, DispatchAuditFailed},
		{"cancel-after-audit", nil, nil, DispatchCancelled},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, in, p, a, _ := dispatchFixture(t)
			if c.change != nil {
				c.change(d, &in, a)
			}
			ctx := context.Context(nil)
			if c.ctx != nil {
				ctx = c.ctx(t)
			} else {
				ctx = dispatchPeer(t, true)
			}
			if c.name == "cancel-after-audit" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				a.after = cancel
				defer cancel()
			}
			attempt, err := d.Prepare(in)
			if err != nil {
				t.Fatal(err)
			}
			if got := attempt.Run(ctx); got != c.want {
				t.Fatalf("got %s want %s", got, c.want)
			}
			if p.pins.Load() != 0 || attempt.Run(ctx) != DispatchSpent {
				t.Fatal("provider reached or attempt replayed")
			}
		})
	}
}
func TestDispatchPinnedEvidenceAndCleanup(t *testing.T) {
	cases := []struct {
		name string
		edit func(*PinnedEvidence)
	}{
		{"tops", func(e *PinnedEvidence) { e.TOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120003" }},
		{"subject", func(e *PinnedEvidence) { e.Subject.ID = "other" }},
		{"peer", func(e *PinnedEvidence) { e.Peer.PID++ }},
		{"resource", func(e *PinnedEvidence) { e.Target.Resource = "other" }},
		{"generation", func(e *PinnedEvidence) { e.Reference.Version = "generation-2" }},
		{"consumer", func(e *PinnedEvidence) { e.ConsumerDigest = "invalid" }},
		{"provider", func(e *PinnedEvidence) { e.ProviderBindingDigest = "invalid" }},
		{"lease", func(e *PinnedEvidence) { e.LeaseID = "" }},
		{"capacity", func(e *PinnedEvidence) { e.MaximumBytes = 1 }},
		{"expiry", func(e *PinnedEvidence) { e.ExpiresAt = time.Now().UTC().Add(-time.Second) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, in, p, _, _ := dispatchFixture(t)
			p.edit = c.edit
			attempt, _ := d.Prepare(in)
			if attempt.Run(dispatchPeer(t, true)) != DispatchRefused || p.executions.Load() != 0 || p.releases.Load() != 1 {
				t.Fatal("unsafe pinned evidence or leaked pin")
			}
		})
	}
	t.Run("pin-error-with-resources", func(t *testing.T) {
		d, in, p, _, _ := dispatchFixture(t)
		p.err = true
		attempt, _ := d.Prepare(in)
		if got := attempt.Run(dispatchPeer(t, true)); got != DispatchUnavailable || strings.Contains(string(got), "private") || p.releases.Load() != 1 || p.executions.Load() != 0 {
			t.Fatal("unsafe provider error")
		}
	})
}
func TestDispatchDeliveryUncertaintyAndDeadline(t *testing.T) {
	for _, test := range []struct {
		name   string
		out    DispatchStatus
		cancel bool
		want   DispatchStatus
	}{{"delivered", DispatchDelivered, false, DispatchDelivered}, {"refused", DispatchRefused, false, DispatchRefused}, {"unknown", DispatchStatus("private error"), false, DispatchIndeterminate}, {"cancelled", DispatchDelivered, true, DispatchIndeterminate}} {
		t.Run(test.name, func(t *testing.T) {
			d, in, p, a, _ := dispatchFixture(t)
			ctx, cancel := context.WithCancel(dispatchPeer(t, true))
			defer cancel()
			p.execute = func(c context.Context, b UseBinding) DispatchStatus {
				if a.calls.Load() != 1 {
					t.Fatal("provider called before audit")
				}
				deadline, ok := c.Deadline()
				if !ok || deadline.After(in.Authorization.RequestedExpiresAt) || !b.Deadline.After(b.IssuedAt) {
					t.Fatal("missing bounded delivery")
				}
				if test.cancel {
					cancel()
				}
				return test.out
			}
			attempt, _ := d.Prepare(in)
			if attempt.Run(ctx) != test.want || p.releases.Load() != 1 || attempt.Run(ctx) != DispatchSpent {
				t.Fatal("unsafe delivery completion")
			}
		})
	}
}

func TestDispatchPreparationAndSpentFailures(t *testing.T) {
	for _, edit := range []func(*DispatchIntent){
		func(in *DispatchIntent) { in.Reference.Version = "" },
		func(in *DispatchIntent) { in.MaximumBytes = 0 },
		func(in *DispatchIntent) { in.MaximumBytes = MaxUseBytes + 1 },
		func(in *DispatchIntent) { in.Authorization.RequestedExpiresAt = time.Now().UTC().Add(time.Hour) },
		func(in *DispatchIntent) { in.Authorization.RequestedExpiresAt = time.Now().UTC().Add(-time.Second) },
	} {
		d, in, _, _, _ := dispatchFixture(t)
		edit(&in)
		if a, err := d.Prepare(in); err == nil || a != nil {
			t.Fatal("invalid intent admitted")
		}
	}
	d, in, p, a, _ := dispatchFixture(t)
	attempt, _ := d.Prepare(in)
	if attempt.Run(nil) != DispatchInvalid || attempt.Run(context.Background()) != DispatchSpent || p.pins.Load() != 0 || a.calls.Load() != 0 {
		t.Fatal("nil-context attempt was reusable")
	}
	attempt, _ = d.Prepare(in)
	d.now = func() time.Time { return in.Authorization.RequestedExpiresAt }
	if attempt.Run(dispatchPeer(t, true)) != DispatchInvalid || p.pins.Load() != 0 {
		t.Fatal("expired prepared attempt used provider")
	}
}
func TestDispatchDeadlineCancelsInFlightAndReleases(t *testing.T) {
	d, in, p, _, _ := dispatchFixture(t)
	ctx, cancel := context.WithTimeout(dispatchPeer(t, true), 50*time.Millisecond)
	defer cancel()
	p.execute = func(ctx context.Context, b UseBinding) DispatchStatus {
		deadline, ok := ctx.Deadline()
		if !ok || !b.Deadline.Equal(deadline) {
			t.Fatal("caller deadline missing from recipient binding")
		}
		<-ctx.Done()
		return DispatchDelivered
	}
	attempt, _ := d.Prepare(in)
	if attempt.Run(ctx) != DispatchIndeterminate || p.releases.Load() != 1 {
		t.Fatal("cancelled delivery not classified or released")
	}
}

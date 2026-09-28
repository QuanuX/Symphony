package policy

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"testing"
	"time"
)

func TestPolicyAdmissionPinsSnapshotAndNeverQueues(t *testing.T) {
	now := time.Now().UTC()
	cfg := policyConfig()
	e, err := New(cfg, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	subject := identity.Subject{ID: "owner.primary", Kind: "owner", Authority: "unix_peer_credentials"}
	request := policyRequest(now)
	e.mu.Lock()
	if e.WithDecision(context.Background(), subject, request, func(model.AuthorizationDecision) error { t.Fatal("callback while writer owns policy"); return nil }) != ErrAdmissionUnavailable {
		t.Fatal("admission queued or accepted while busy")
	}
	e.mu.Unlock()
	replacement := *cfg.Authorization
	replacement.Grants = nil
	encoded, _ := json.Marshal(replacement)
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- e.WithDecision(context.Background(), subject, request, func(d model.AuthorizationDecision) error {
			if d.Effect != "allow" {
				return errors.New("original policy not admitted")
			}
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	if e.mu.TryLock() {
		e.mu.Unlock()
		t.Fatal("policy not pinned")
	}
	replaced := make(chan error, 1)
	go func() { replaced <- e.Replace(&replacement, taggedDigest(encoded)) }()
	// Release is the linearization boundary before a replacement can commit.
	select {
	case <-replaced:
		t.Fatal("replacement committed during pin")
	default:
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if err := <-replaced; err != nil {
		t.Fatal(err)
	}
	if e.Evaluate(context.Background(), subject, request).Effect != "deny" {
		t.Fatal("replacement failed after pin released")
	}
}
func TestPolicyAdmissionInvalidContextAndRequest(t *testing.T) {
	now := time.Now().UTC()
	e, _ := New(policyConfig(), func() time.Time { return now })
	subject := identity.Subject{ID: "owner.primary"}
	req := policyRequest(now)
	called := false
	fn := func(model.AuthorizationDecision) error { called = true; return nil }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e.WithDecision(ctx, subject, req, fn) == nil || e.WithDecision(nil, subject, req, fn) == nil {
		t.Fatal("invalid context accepted")
	}
	req.RequestedExpiresAt = now
	if e.WithDecision(context.Background(), subject, req, fn) == nil || called {
		t.Fatal("expired admission called callback")
	}
	var absent *Engine
	if absent.WithDecision(ctx, subject, req, fn) == nil || new(Engine).WithDecision(ctx, subject, req, fn) == nil {
		t.Fatal("invalid engine accepted")
	}
}
func TestInitialPolicyDoesNotAliasCaller(t *testing.T) {
	now := time.Now().UTC()
	cfg := policyConfig()
	e, _ := New(cfg, func() time.Time { return now })
	before := e.PolicyDigest()
	cfg.Authorization.Grants[0].Operation = "different"
	cfg.Authorization.MaxCapabilitySeconds = 1
	d := e.Evaluate(context.Background(), identity.Subject{ID: "owner.primary", Kind: "owner", Authority: "unix_peer_credentials"}, policyRequest(now))
	if d.Effect != "allow" || d.PolicyDigest != before || d.Capability.ExpiresAt.Sub(d.Capability.IssuedAt) != 900*time.Second {
		t.Fatal("caller mutation changed policy without a digest transition")
	}
}

func TestInitialEmptyPolicyRetainsCanonicalDigest(t *testing.T) {
	cfg := policyConfig()
	cfg.Authorization.Grants = cfg.Authorization.Grants[:0]
	e, err := New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(e.policy)
	if err != nil || taggedDigest(encoded) != e.PolicyDigest() {
		t.Fatal("owned empty policy changed its canonical representation")
	}
}

package policy

import (
	"context"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
)

const policyTestTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func policyConfig() config.Config {
	uid, gid := uint32(501), uint32(20)
	return config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "development",
		TOPS:   config.TOPSConfig{ID: policyTestTOPSID, Name: "Test TOPS"},
		Listen: config.ListenConfig{Network: "unix", Address: "/tmp/ssiag-policy-test.sock"},
		Authentication: &config.AuthenticationConfig{
			Mechanism: "unix_peer_credentials",
			Subjects:  []config.SubjectConfig{{ID: "owner.primary", Kind: "owner", UID: &uid, GID: &gid}},
		},
		Authorization: &config.AuthorizationConfig{
			DefaultEffect: "deny", MaxCapabilitySeconds: 900,
			Grants: []config.AuthorizationGrant{{
				ID: "session-begin", SubjectID: "owner.primary", AuthorityBasis: "host_owner",
				Operation: "symphony.knowledge.session.begin", Resource: "symphony.knowledge.repository:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Audience: "qxctl", Scope: "tops:" + policyTestTOPSID,
			}},
		},
		Providers: []config.ProviderConfig{},
	}
}

func policyRequest(now time.Time) model.AuthorizationRequest {
	return model.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: "request-1",
		CorrelationID: "correlation-1", Operation: "symphony.knowledge.session.begin",
		Resource: "symphony.knowledge.repository:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Audience: "qxctl", Scope: "tops:" + policyTestTOPSID,
		RequestedAt: now, RequestedExpiresAt: now.Add(30 * time.Minute),
	}
}

func TestExactGrantIsCallerClassNeutralAndContentBound(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	engine, err := New(policyConfig(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"human", "ai-agent", "service", "future-owner-kind"} {
		decision := engine.Evaluate(context.Background(), identity.Subject{
			ID: "owner.primary", Kind: kind, Authority: "unix_peer_credentials",
		}, policyRequest(now))
		if decision.Effect != "allow" || decision.Capability == nil || decision.CallerClassUsed {
			t.Fatalf("kind %q changed authority: %+v", kind, decision)
		}
		if decision.Capability.Transferable || decision.Capability.CanonicalApply ||
			decision.Capability.BindingDigest == "" || decision.PolicyDigest == decision.ConfigDigest {
			t.Fatalf("unsafe or incomplete capability: %+v", decision.Capability)
		}
	}
}

func TestUnknownTupleAndLegacyPolicyFailClosed(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cfg := policyConfig()
	engine, _ := New(cfg, func() time.Time { return now })
	request := policyRequest(now)
	request.Operation = "symphony.knowledge.session.checkpoint"
	decision := engine.Evaluate(context.Background(), identity.Subject{
		ID: "owner.primary", Kind: "owner", Authority: "unix_peer_credentials",
	}, request)
	if decision.Effect != "deny" || decision.Capability != nil || decision.AuthorityBasis != nil {
		t.Fatalf("unknown tuple did not deny: %+v", decision)
	}
	cfg.Authorization = nil
	legacy, err := New(cfg, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if got := legacy.Evaluate(context.Background(), identity.Subject{ID: "owner.primary"}, policyRequest(now)); got.Effect != "deny" {
		t.Fatalf("legacy config gained authority: %+v", got)
	}
}

func TestRequestFreshnessFailsClosed(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	request := policyRequest(now)
	request.RequestedAt = now.Add(-time.Minute)
	if err := ValidateRequest(request, now); err == nil {
		t.Fatal("stale authorization request was accepted")
	}
	request = policyRequest(now)
	request.RequestedExpiresAt = now
	if err := ValidateRequest(request, now); err == nil {
		t.Fatal("expired authorization request was accepted")
	}
	request = policyRequest(now)
	request.RequestedAt = now.Add(20 * time.Second)
	request.RequestedExpiresAt = now.Add(10 * time.Second)
	if err := ValidateRequest(request, now); err == nil {
		t.Fatal("authorization expiry preceding the request timestamp was accepted")
	}
}

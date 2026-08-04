package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
)

func validSessionAuthorization(t *testing.T) (ssiagclient.AuthorizationRequest, ssiagclient.AuthorizationDecision) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	topsID := ssiagTestTOPSID
	request := ssiagclient.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: "request-1", CorrelationID: "correlation-1",
		Operation: "symphony.knowledge.session.begin", Resource: sessionRepositoryResource("/tmp/symphony-repository"),
		Audience: "qxctl", Scope: "tops:" + topsID, RequestedAt: now, RequestedExpiresAt: now.Add(15 * time.Minute),
	}
	subject := ssiagclient.DecisionSubject{ID: "owner.primary", Kind: "owner", Authority: "unix_peer_credentials"}
	target := ssiagclient.DecisionTarget{Operation: request.Operation, Resource: request.Resource, Audience: request.Audience, Scope: request.Scope}
	basis := "host_owner"
	expires := now.Add(10 * time.Minute)
	capability := &ssiagclient.Capability{
		Protocol: "symphony.ssiag.capability.v1",
		Subject:  subject, TOPSID: topsID, Target: target, AuthorityBasis: basis, GrantID: "session-begin",
		RequestID: request.RequestID, CorrelationID: request.CorrelationID, IssuedAt: now, ExpiresAt: expires,
		PolicyDigest: "sha256:" + strings.Repeat("a", 64), ConfigDigest: "sha256:" + strings.Repeat("b", 64),
		Transferable: false, CanonicalApply: false,
	}
	capability.BindingDigest = sessionCapabilityBinding(*capability)
	capability.CapabilityID = "ssiag-capability:" + strings.TrimPrefix(capability.BindingDigest, "sha256:")
	decision := ssiagclient.AuthorizationDecision{
		Schema: "symphony.ssiag.authorization-decision.v1", DecisionID: "ssiag-decision:one",
		RequestID: request.RequestID, CorrelationID: request.CorrelationID, TOPSID: topsID,
		Subject: subject, Target: target, Effect: "allow", ReasonCode: "symphony.ssiag.policy.exact-grant",
		AuthorityBasis: &basis, Capability: capability, PolicyDigest: capability.PolicyDigest,
		ConfigDigest: capability.ConfigDigest, DecidedAt: now, ExpiresAt: &expires,
		CallerClassUsed: false, CanonicalApply: false,
	}
	return request, decision
}

func TestValidateSessionAuthorizationRejectsEscalationAndDrift(t *testing.T) {
	request, decision := validSessionAuthorization(t)
	if err := validateSessionAuthorization(decision, request, ssiagTestTOPSID); err != nil {
		t.Fatal(err)
	}
	decision.CallerClassUsed = true
	if err := validateSessionAuthorization(decision, request, ssiagTestTOPSID); err == nil {
		t.Fatal("caller-class-dependent decision was accepted")
	}
	_, decision = validSessionAuthorization(t)
	decision.Capability.CanonicalApply = true
	if err := validateSessionAuthorization(decision, request, ssiagTestTOPSID); err == nil {
		t.Fatal("canonical-apply capability was accepted")
	}
	_, decision = validSessionAuthorization(t)
	decision.Target.Operation = "symphony.knowledge.session.close"
	if err := validateSessionAuthorization(decision, request, ssiagTestTOPSID); err == nil {
		t.Fatal("target-drifted decision was accepted")
	}
	_, decision = validSessionAuthorization(t)
	decision.Capability.BindingDigest = "sha256:" + strings.Repeat("c", 64)
	if err := validateSessionAuthorization(decision, request, ssiagTestTOPSID); err == nil {
		t.Fatal("capability with a forged binding digest was accepted")
	}
}

func validSessionResult(t *testing.T, operation string, present bool) json.RawMessage {
	t.Helper()
	var journal any
	var digest any
	state := "absent"
	if present {
		digest = "sha256:" + strings.Repeat("1", 64)
		journal = map[string]any{"protocol": "symphony.knowledge.session-journal.v1", "journal_digest": digest, "canonical": false}
		state = "open"
	}
	value := map[string]any{
		"protocol": "symphony.knowledge.session-result.v1", "operation": operation,
		"compatibility":   map[string]any{"mode": "full"},
		"journal_present": present, "journal": journal, "journal_digest": digest, "effective_state": state,
		"changed": false, "recovered": false, "repair_actions": []string{},
		"read_only": operation == "session_status", "canonical_apply_enabled": false, "canonical": false,
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestValidateSessionResultClosesAuthorityBoundary(t *testing.T) {
	for _, operation := range []string{"session_begin", "session_status", "session_checkpoint", "session_close", "session_recover"} {
		if _, err := validateSessionResult(validSessionResult(t, operation, true), operation); err != nil {
			t.Fatalf("%s valid result rejected: %v", operation, err)
		}
	}
	var value map[string]any
	if err := json.Unmarshal(validSessionResult(t, "session_status", true), &value); err != nil {
		t.Fatal(err)
	}
	value["canonical_apply_enabled"] = true
	data, _ := json.Marshal(value)
	if _, err := validateSessionResult(data, "session_status"); err == nil {
		t.Fatal("canonical-apply session result was accepted")
	}
}

func TestRandomUUIDIsCanonicalV4(t *testing.T) {
	value, err := randomUUID()
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 36 || value[14] != '4' || !strings.Contains("89ab", value[19:20]) {
		t.Fatalf("noncanonical random UUID %q", value)
	}
}

func TestSessionRepositoryResourceIsStableAndPathBound(t *testing.T) {
	first := sessionRepositoryResource("/tmp/symphony-repository")
	if first != sessionRepositoryResource("/tmp/symphony-repository") {
		t.Fatal("repository resource is not deterministic")
	}
	if first == sessionRepositoryResource("/tmp/other-repository") ||
		!strings.HasPrefix(first, "symphony.knowledge.repository:") || len(first) != 94 {
		t.Fatalf("repository resource is not a path-bound opaque digest: %q", first)
	}
}

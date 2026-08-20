package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/policyadmin"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

type recordingAudit struct {
	record stavproducer.Record
}

func (audit *recordingAudit) Submit(_ context.Context, record stavproducer.Record) (stavprotocol.Receipt, error) {
	audit.record = record
	candidateDigest, err := stavproducer.CandidateDigest(testTOPSID, record)
	if err != nil {
		return stavprotocol.Receipt{}, err
	}
	return stavprotocol.Receipt{
		CandidateDigest: candidateDigest,
		Commit: stavprotocol.CommitResult{
			EventDigest: "sha256:4236aee922a67725aa5b90e22e88bfcf0aa510875f03777b82e326a1ffa5eef2",
			EventID:     "b90e1205-1b3b-4e47-9b91-1cd624cd87cd", Sequence: 1, State: "committed", Timestamp: "2026-08-14T12:00:00.000000000Z",
		},
		Disposition: "committed", ReasonCode: stavprotocol.ReasonReceiptCommitted, RequestID: record.RequestID,
		Schema: stavprotocol.SchemaReceipt, TOPSID: testTOPSID,
	}, nil
}

func TestProviderBindingAuditProjectionExcludesAdministrativeReason(t *testing.T) {
	const marker = "token=forbidden-provider-binding-secret-marker"
	attempt := provider.ProviderBindingAttempt{
		OperationID: "684921d8-a8b5-49da-872b-568eb6a6dc03", TOPSID: testTOPSID, ProviderName: "native",
		Plan: provider.ProviderBindingPlan{Reason: marker},
		AuditIdentity: provider.ProviderBindingAuditIdentity{
			ActorID: "symphony.host.owner.uid.501", ActorKind: "symphony.identity.host-owner", AuthenticationMethod: "symphony.ssiag.local-peer",
		},
	}
	record := ProviderBindingAuditRecord(attempt,
		"sha256:4236aee922a67725aa5b90e22e88bfcf0aa510875f03777b82e326a1ffa5eef2",
		"sha256:5236aee922a67725aa5b90e22e88bfcf0aa510875f03777b82e326a1ffa5eef2")
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), marker) {
		t.Fatalf("administrative reason propagated into STAV record: %s", encoded)
	}
	if _, err := stavproducer.CandidateDigest(testTOPSID, record); err != nil {
		t.Fatalf("safe provider-binding audit projection is invalid: %v", err)
	}
}

func TestAuthorizationOverKernelAuthenticatedSocketIsAudited(t *testing.T) {
	socket := shortSocketPath(t)
	probe, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	authentication := serviceAuthentication(uid, gid)
	authentication.Subjects = []config.SubjectConfig{{ID: "owner.primary", Kind: "owner", UID: &uid, GID: &gid}}
	cfg := config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: authentication,
		Authorization: &config.AuthorizationConfig{
			DefaultEffect: "deny", MaxCapabilitySeconds: 900,
			Grants: []config.AuthorizationGrant{{
				ID: "session-begin", SubjectID: "owner.primary", AuthorityBasis: "host_owner",
				Operation: "symphony.knowledge.session.begin", Resource: "symphony.knowledge.repository:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Audience: "qxctl", Scope: "tops:" + testTOPSID,
			}},
		},
		Providers: []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	audit := &recordingAudit{}
	instance, err := NewWithAudit(cfg, registry, audit)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- instance.Run(ctx) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if info, statErr := os.Stat(socket); statErr == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		select {
		case runErr := <-done:
			t.Fatalf("authorization server stopped before creating its socket: %v", runErr)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("socket %s was not created", socket)
		}
		time.Sleep(10 * time.Millisecond)
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	now := time.Now().UTC().Truncate(time.Second)
	payload, _ := json.Marshal(model.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: "request-1", CorrelationID: "correlation-1",
		Operation: "symphony.knowledge.session.begin", Resource: "symphony.knowledge.repository:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Audience: "qxctl", Scope: "tops:" + testTOPSID, RequestedAt: now, RequestedExpiresAt: now.Add(10 * time.Minute),
	})
	response, err := client.Post("http://unix/v1/authorization/decisions", "application/json", bytes.NewReader(payload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	defer response.Body.Close()
	var decision model.AuthorizationDecision
	if err := json.NewDecoder(response.Body).Decode(&decision); err != nil {
		cancel()
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || decision.Effect != "allow" || decision.Capability == nil {
		cancel()
		t.Fatalf("authorization response was not allowed: status=%d decision=%+v", response.StatusCode, decision)
	}
	if audit.record.Kind != stavproducer.PolicyDecision || audit.record.Outcome != "allowed" ||
		audit.record.Actor.ID != "owner.primary" || audit.record.Configuration.State != "digests" {
		cancel()
		t.Fatalf("safe STAV policy evidence was not submitted: %+v", audit.record)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestHostOwnerCanProposeApplyAndActivateLocalPolicy(t *testing.T) {
	socket := shortSocketPath(t)
	probe, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	authentication := serviceAuthentication(uid, gid)
	authentication.Subjects = []config.SubjectConfig{{ID: "owner.primary", Kind: "owner", UID: &uid, GID: &gid}}
	cfg := config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: authentication,
		Authorization:  &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 900, Grants: []config.AuthorizationGrant{}},
		Providers:      []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	audit := &recordingAudit{}
	manager, err := policyadmin.New(t.TempDir(), cfg, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := NewWithPolicyAdministration(cfg, registry, audit, manager)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- instance.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	select {
	case runErr := <-done:
		t.Fatalf("policy administration server stopped before creating its socket: %v", runErr)
	default:
	}
	waitForSocket(t, socket)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}

	statusResponse, err := client.Get("http://unix/v1/policy/status")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var status policyadmin.Result
	if err := json.NewDecoder(statusResponse.Body).Decode(&status); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = statusResponse.Body.Close()
	now := time.Now().UTC().Truncate(time.Second)
	proposalRequest := policyadmin.ProposalRequest{
		Protocol: policyadmin.ProposalRequestProtocol, OperationID: "policy-operation-1",
		RequestID: "policy-request-1", CorrelationID: "policy-correlation-1", AuthorityBasis: "host_owner",
		ExpectedPolicyDigest: status.PolicyDigest, Change: "replace",
		DesiredPolicy: &config.AuthorizationConfig{
			DefaultEffect: "deny", MaxCapabilitySeconds: 300,
			Grants: []config.AuthorizationGrant{{
				ID: "allow-test", SubjectID: "owner.primary", AuthorityBasis: "host_owner",
				Operation: "symphony.test.read", Resource: "symphony.test:one", Audience: "qxctl", Scope: "tops:" + testTOPSID,
			}},
		},
		RequestedAt: now, ExpiresAt: now.Add(5 * time.Minute),
	}
	proposalPayload, _ := json.Marshal(proposalRequest)
	proposalResponse, err := client.Post("http://unix/v1/policy/proposals", "application/json", bytes.NewReader(proposalPayload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var proposal policyadmin.Proposal
	if err := json.NewDecoder(proposalResponse.Body).Decode(&proposal); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = proposalResponse.Body.Close()
	if proposalResponse.StatusCode != http.StatusOK || proposal.Subject.ID != fmt.Sprintf("symphony.host.owner.uid.%d", uid) || proposal.CallerClassUsed {
		cancel()
		t.Fatalf("unexpected policy proposal: status=%d proposal=%+v", proposalResponse.StatusCode, proposal)
	}
	applyPayload, _ := json.Marshal(policyadmin.ApplyRequest{Protocol: policyadmin.ApplyRequestProtocol, Proposal: proposal})
	applyResponse, err := client.Post("http://unix/v1/policy/apply", "application/json", bytes.NewReader(applyPayload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var result policyadmin.Result
	if err := json.NewDecoder(applyResponse.Body).Decode(&result); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = applyResponse.Body.Close()
	if applyResponse.StatusCode != http.StatusOK || !result.Changed || result.Generation != 1 || result.RecoveryRequired {
		cancel()
		t.Fatalf("unexpected policy apply: status=%d result=%+v", applyResponse.StatusCode, result)
	}
	if audit.record.Target.ID != "symphony.ssiag.policy:"+testTOPSID || audit.record.Configuration.NewDigest != result.PolicyDigest {
		cancel()
		t.Fatalf("policy mutation was not safely audited: %+v", audit.record)
	}
	authorizationPayload, _ := json.Marshal(model.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: "authorization-request-1", CorrelationID: "authorization-correlation-1",
		Operation: "symphony.test.read", Resource: "symphony.test:one", Audience: "qxctl", Scope: "tops:" + testTOPSID,
		RequestedAt: now, RequestedExpiresAt: now.Add(time.Minute),
	})
	authorizationResponse, err := client.Post("http://unix/v1/authorization/decisions", "application/json", bytes.NewReader(authorizationPayload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var decision model.AuthorizationDecision
	if err := json.NewDecoder(authorizationResponse.Body).Decode(&decision); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = authorizationResponse.Body.Close()
	if decision.Effect != "allow" || decision.PolicyDigest != result.PolicyDigest {
		cancel()
		t.Fatalf("committed policy was not activated: %+v", decision)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestProviderTrustEndpointsRemainSSIAGOwnedAndFailClosedWhenUnbound(t *testing.T) {
	socket := shortSocketPath(t)
	probe, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	cfg := config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Provider Trust TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(uid, gid),
		Authorization:  &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 900, Grants: []config.AuthorizationGrant{}},
		Providers:      []config.ProviderConfig{{Name: "native", Kind: "macos-keychain", Enabled: true, Capabilities: []string{"capability-discovery", "metadata"}}},
	}
	registry, err := provider.New(cfg.Providers)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	layout := ssiagpaths.InstanceLayout{Scope: ssiagpaths.ScopeUser, TOPSID: testTOPSID, ProviderTrustDir: filepath.Join(root, "provider-trust")}
	if err := os.MkdirAll(layout.ProviderTrustDir, 0o700); err != nil {
		t.Fatal(err)
	}
	providerTrust, err := provider.NewTrustManager(ssiagpaths.ScopeUser, layout, registry)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := policyadmin.New(filepath.Join(root, "policy"), cfg, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := NewWithPolicyAdministrationAndProviderTrust(cfg, registry, nil, manager, providerTrust)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- instance.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	select {
	case runErr := <-done:
		t.Fatalf("provider trust server stopped before creating its socket: %v", runErr)
	default:
	}
	waitForSocket(t, socket)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	response, err := client.Get("http://unix/v1/provider-trust/native")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var shown provider.TrustResult
	if err := json.NewDecoder(response.Body).Decode(&shown); err != nil {
		_ = response.Body.Close()
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || shown.Operation != "show" || shown.TrustState != "unbound" || !shown.ReadOnly || shown.Canonical {
		cancel()
		t.Fatalf("unexpected provider trust snapshot: status=%d result=%+v", response.StatusCode, shown)
	}
	payload := []byte(`{"protocol":"symphony.ssiag.provider-trust-verification-request.v1","request_id":"018f0c3a-7b2d-7e11-8c12-0242ac120003","correlation_id":"018f0c3a-7b2d-7e11-8c12-0242ac120004","authority_basis":"host_owner"}`)
	response, err = client.Post("http://unix/v1/provider-trust/native/verifications", "application/json", bytes.NewReader(payload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var verified provider.TrustResult
	if err := json.NewDecoder(response.Body).Decode(&verified); err != nil {
		_ = response.Body.Close()
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || verified.Operation != "verify" || verified.VerificationMode != "fresh" || verified.TrustState != "unbound" || verified.MutualTrust.FoundationVerifiedAdapter || verified.MutualTrust.AdapterVerifiedFoundation {
		cancel()
		t.Fatalf("unbound provider verification did not fail closed: status=%d result=%+v", response.StatusCode, verified)
	}
	duplicate := []byte(`{"protocol":"symphony.ssiag.provider-trust-verification-request.v1","request_id":"018f0c3a-7b2d-7e11-8c12-0242ac120003","correlation_id":"018f0c3a-7b2d-7e11-8c12-0242ac120004","authority_basis":"host_owner","authority_basis":"granted_permission"}`)
	response, err = client.Post("http://unix/v1/provider-trust/native/verifications", "application/json", bytes.NewReader(duplicate))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		cancel()
		t.Fatalf("duplicate provider verification member was accepted: status=%d", response.StatusCode)
	}
	readinessPayload := []byte(`{"protocol":"symphony.ssiag.provider-readiness-observation-request.v1","request_id":"018f0c3a-7b2d-7e11-8c12-0242ac120003","correlation_id":"018f0c3a-7b2d-7e11-8c12-0242ac120004","authority_basis":"host_owner"}`)
	response, err = client.Post("http://unix/v1/provider-readiness/native/observations", "application/json", bytes.NewReader(readinessPayload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var readiness provider.ReadinessResult
	if err := json.NewDecoder(response.Body).Decode(&readiness); err != nil {
		_ = response.Body.Close()
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || readiness.Protocol != provider.ProviderReadinessResultProtocol ||
		readiness.ReadinessState != "unavailable" || !readiness.ReadOnly || readiness.Canonical ||
		readiness.OperationalAccessEnabled || readiness.ProviderOperationsEnabled || readiness.SecretChannelEnabled {
		cancel()
		t.Fatalf("unbound provider readiness did not fail closed: status=%d result=%+v", response.StatusCode, readiness)
	}
	duplicateReadiness := []byte(`{"protocol":"symphony.ssiag.provider-readiness-observation-request.v1","request_id":"018f0c3a-7b2d-7e11-8c12-0242ac120003","correlation_id":"018f0c3a-7b2d-7e11-8c12-0242ac120004","authority_basis":"host_owner","authority_basis":"granted_permission"}`)
	response, err = client.Post("http://unix/v1/provider-readiness/native/observations", "application/json", bytes.NewReader(duplicateReadiness))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		cancel()
		t.Fatalf("duplicate provider readiness member was accepted: status=%d", response.StatusCode)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestProviderBindingRoutesAreHeadlessStrictAndIdempotent(t *testing.T) {
	socket := shortSocketPath(t)
	probe, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	cfg := config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Provider Binding TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(uid, gid),
		Authorization:  &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 900, Grants: []config.AuthorizationGrant{}},
		Providers:      []config.ProviderConfig{{Name: "native", Kind: "macos-keychain", Enabled: true, Capabilities: []string{"capability-discovery", "metadata"}}},
	}
	registry, err := provider.New(cfg.Providers)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	layout := ssiagpaths.InstanceLayout{
		Scope: ssiagpaths.ScopeUser, TOPSID: testTOPSID, ProviderTrustDir: filepath.Join(root, "provider-trust"),
		StateDir: filepath.Join(root, "state"), ProviderBindingDir: filepath.Join(root, "state", "provider-bindings"),
	}
	if err := os.MkdirAll(layout.ProviderTrustDir, 0o700); err != nil {
		t.Fatal(err)
	}
	providerTrust, err := provider.NewTrustManager(ssiagpaths.ScopeUser, layout, registry)
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := provider.NewBindingManager(ssiagpaths.ScopeUser, layout, registry, providerTrust)
	if err != nil {
		t.Fatal(err)
	}
	policyManager, err := policyadmin.New(filepath.Join(root, "policy"), cfg, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	audit := &recordingAudit{}
	instance, err := NewWithPolicyAdministrationProviderTrustAndBindings(cfg, registry, audit, policyManager, providerTrust, bindings)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- instance.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	select {
	case runErr := <-done:
		t.Fatalf("provider binding server stopped before creating its socket: %v", runErr)
	default:
	}
	waitForSocket(t, socket)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}

	response, err := client.Get("http://unix/v1/provider-installations/native")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var inventory provider.ProviderInstallationInventory
	if err := json.NewDecoder(response.Body).Decode(&inventory); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || len(inventory.Installations) != 0 || !inventory.ReadOnly || inventory.InventoryDigest == "" {
		cancel()
		t.Fatalf("unexpected empty installation inventory: status=%d result=%+v", response.StatusCode, inventory)
	}
	response, err = client.Get("http://unix/v1/provider-bindings/native")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var status provider.ProviderBindingStatus
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || status.StateDigest != "absent" || status.BindingState != "unbound" {
		cancel()
		t.Fatalf("unexpected initial binding status: status=%d result=%+v", response.StatusCode, status)
	}
	planPayload := []byte(`{"installation_id":"not_applicable","expected_state_digest":"absent","reason":"retain explicit unbound state"}`)
	response, err = client.Post("http://unix/v1/provider-bindings/native/plans", "application/json", bytes.NewReader(planPayload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var plan provider.ProviderBindingPlan
	if err := json.NewDecoder(response.Body).Decode(&plan); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || plan.Changed || plan.PlanDigest == "" {
		cancel()
		t.Fatalf("unexpected no-op plan: status=%d result=%+v", response.StatusCode, plan)
	}
	applyPayload, _ := json.Marshal(provider.ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"})
	response, err = client.Post("http://unix/v1/provider-bindings/native/apply", "application/json", bytes.NewReader(applyPayload))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var result provider.ProviderBindingResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || result.Changed || result.StateDigest != "absent" || result.RecoveryRequired || audit.record.Kind != "" {
		cancel()
		t.Fatalf("no-op binding apply was not idempotent and audit-free: status=%d result=%+v audit=%+v", response.StatusCode, result, audit.record)
	}
	duplicate := []byte(`{"installation_id":"not_applicable","installation_id":"not_applicable","expected_state_digest":"absent","reason":"duplicate"}`)
	response, err = client.Post("http://unix/v1/provider-bindings/native/plans", "application/json", bytes.NewReader(duplicate))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		cancel()
		t.Fatalf("duplicate provider binding member was accepted: %d", response.StatusCode)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestStatusOverUnixSocket(t *testing.T) {
	socket := shortSocketPath(t)
	probe, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers: []config.ProviderConfig{{
			Name: "native",
			Kind: "native-keyring",
		}},
	}
	registry, err := provider.New(cfg.Providers)
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()
	waitForSocket(t, socket)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	response, err := client.Get("http://unix/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var status model.Status
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.ProviderCount != 1 || status.TOPSID != testTOPSID {
		t.Fatalf("unexpected status: %+v", status)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRefusesRegularFileAtSocketPath(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "ssiag.sock")
	if err := os.WriteFile(socket, []byte("do not replace"), 0600); err != nil {
		t.Fatal(err)
	}
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers:      []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Run(context.Background()); err == nil {
		t.Fatal("expected non-socket collision error")
	}
}

func TestRefusesActiveSocketPath(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "ssiag.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Skipf("unix sockets are unavailable in this test environment: %v", err)
	}
	defer listener.Close()

	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers:      []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "active SSIAG socket") {
		t.Fatalf("expected active-socket collision error, got %v", err)
	}
	if info, err := os.Lstat(socket); err != nil || info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("active socket was removed: info=%v error=%v", info, err)
	}
}

func TestHandlerRejectsRequestWithoutKernelPeerContext(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "ssiag.sock")
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(currentUID, currentGID),
		Providers:      []config.ProviderConfig{},
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	var failure model.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != "request.peer_authentication_failed" {
		t.Fatalf("unexpected error: %+v", failure)
	}
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeSocket != 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("socket %s was not created", path)
}

func shortSocketPath(t *testing.T) string {
	t.Helper()
	file, err := os.CreateTemp("/tmp", "symphony-ssiag-*.sock")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

func TestRunRejectsIdentityMismatch(t *testing.T) {
	socket := shortSocketPath(t)
	wrongUID := uint32(os.Geteuid() + 1)
	wrongGID := uint32(os.Getegid() + 1)
	cfg := config.Config{
		Schema:         "symphony.ssiag.config.v1",
		Mode:           "development",
		TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen:         config.ListenConfig{Network: "unix", Address: socket},
		Authentication: serviceAuthentication(wrongUID, wrongGID),
	}
	registry, _ := provider.New(nil)
	server, err := New(cfg, registry)
	if err != nil {
		t.Fatal(err)
	}
	err = server.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "process identity mismatch") {
		t.Fatalf("expected identity mismatch error, got %v", err)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("identity-mismatched process changed the socket path: %v", err)
	}
}

func serviceAuthentication(uid, gid uint32) *config.AuthenticationConfig {
	return &config.AuthenticationConfig{
		Mechanism: "unix_peer_credentials",
		Service: &config.SubjectConfig{
			ID: config.ServiceSubjectID, Kind: config.ServiceSubjectKind, UID: &uid, GID: &gid,
		},
		Subjects: []config.SubjectConfig{},
	}
}

package policyadmin

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestPolicyApplyPersistsAndResetRestoresConfig(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	cfg := testConfig(t)
	manager, err := New(t.TempDir(), cfg, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Source != "config" || status.Generation != 0 || status.StateDigest != "absent" {
		t.Fatalf("unexpected initial status: %+v", status)
	}
	desired := &config.AuthorizationConfig{
		DefaultEffect: "deny", MaxCapabilitySeconds: 60,
		Grants: []config.AuthorizationGrant{{
			ID: "policy-admin", SubjectID: "operator.primary", AuthorityBasis: "granted_permission",
			Operation: "symphony.ssiag.policy.apply", Resource: "symphony.ssiag.policy:" + testTOPSID,
			Audience: "ssiag", Scope: "tops:" + testTOPSID,
		}},
	}
	proposal := propose(t, manager, now, status.PolicyDigest, "replace", desired, "operation-1")
	attempt, applied, err := manager.Prepare(proposal)
	if err != nil || applied || attempt.Stage != "prepared" {
		t.Fatalf("prepare: attempt=%+v applied=%t error=%v", attempt, applied, err)
	}
	status, _ = manager.Status()
	if !status.RecoveryRequired || status.AttemptDigest != attempt.AttemptDigest {
		t.Fatalf("prepared attempt was not visible: %+v", status)
	}
	receipt := stavprotocol.Receipt{Schema: stavprotocol.SchemaReceipt, RequestID: proposal.RequestID, TOPSID: testTOPSID, Disposition: "committed"}
	attempt, err = manager.MarkAudited(proposal.ProposalDigest, receipt)
	if err != nil || attempt.Stage != "audited" {
		t.Fatalf("mark audited: %+v %v", attempt, err)
	}
	result, effective, err := manager.Commit(proposal.ProposalDigest, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Generation != 1 || result.Source != "overlay" || effective.MaxCapabilitySeconds != 60 {
		t.Fatalf("unexpected apply result: %+v effective=%+v", result, effective)
	}
	reloaded, err := New(manager.stateDir, cfg, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	loaded, digest, err := reloaded.Effective()
	if err != nil || digest != result.PolicyDigest || loaded.MaxCapabilitySeconds != 60 {
		t.Fatalf("overlay did not reload: policy=%+v digest=%s error=%v", loaded, digest, err)
	}
	reset := propose(t, reloaded, now.Add(time.Minute), digest, "reset", nil, "operation-2")
	resetAttempt, _, err := reloaded.Prepare(reset)
	if err != nil {
		t.Fatal(err)
	}
	receipt.RequestID = reset.RequestID
	if _, err := reloaded.MarkAudited(reset.ProposalDigest, receipt); err != nil {
		t.Fatal(err)
	}
	resetResult, resetPolicy, err := reloaded.Commit(resetAttempt.Proposal.ProposalDigest, false)
	if err != nil || resetResult.Source != "config" || resetResult.Generation != 2 || resetPolicy.MaxCapabilitySeconds != 900 {
		t.Fatalf("unexpected reset: result=%+v policy=%+v error=%v", resetResult, resetPolicy, err)
	}
}

func TestPolicyCASAndRecoveryEvidenceFailClosed(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	manager, err := New(t.TempDir(), testConfig(t), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	status, _ := manager.Status()
	proposal := propose(t, manager, now, status.PolicyDigest, "replace", &config.AuthorizationConfig{
		DefaultEffect: "deny", MaxCapabilitySeconds: 120, Grants: []config.AuthorizationGrant{},
	}, "operation-1")
	forged := proposal
	forged.OperationID = "forged-operation"
	forged.ProposalDigest, _ = digestWithout(forged, "proposal_digest")
	if _, _, err := manager.Prepare(forged); err == nil {
		t.Fatal("apply accepted a proposal not issued by the service")
	}
	attempt, _, err := manager.Prepare(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Propose(identity.Subject{ID: "owner", Kind: "owner", Authority: "test"}, ProposalRequest{
		Protocol: ProposalRequestProtocol, OperationID: "operation-2", RequestID: "request-2", CorrelationID: "correlation-2",
		AuthorityBasis: "host_owner", ExpectedPolicyDigest: status.PolicyDigest, Change: "reset",
		RequestedAt: now, ExpiresAt: now.Add(time.Minute),
	}); err != ErrRecoveryRequired {
		t.Fatalf("expected recovery-required conflict, got %v", err)
	}
	if _, err := manager.Pending(RecoveryRequest{
		Protocol: RecoveryRequestProtocol, OperationID: "operation-1", ExpectedAttemptDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}); err != ErrConflict {
		t.Fatalf("expected attempt digest conflict, got %v", err)
	}
	if pending, err := manager.Pending(RecoveryRequest{
		Protocol: RecoveryRequestProtocol, OperationID: "operation-1", Discover: true,
	}); err != nil || pending.AttemptDigest != attempt.AttemptDigest {
		t.Fatalf("discover did not return exact attempt: %+v %v", pending, err)
	}
}

func TestPolicyStateRejectsSymlinkAndTampering(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "policy")); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root, testConfig(t), func() time.Time { return now }); err == nil {
		t.Fatal("symlinked policy state directory was accepted")
	}

	root = t.TempDir()
	manager, err := New(root, testConfig(t), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	status, _ := manager.Status()
	proposal := propose(t, manager, now, status.PolicyDigest, "replace", &config.AuthorizationConfig{
		DefaultEffect: "deny", MaxCapabilitySeconds: 120, Grants: []config.AuthorizationGrant{},
	}, "operation-1")
	if _, _, err := manager.Prepare(proposal); err != nil {
		t.Fatal(err)
	}
	attemptPath := filepath.Join(root, "policy", "attempt.json")
	data, err := os.ReadFile(attemptPath)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)/2] ^= 1
	if err := os.WriteFile(attemptPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root, testConfig(t), func() time.Time { return now }); err == nil {
		t.Fatal("tampered policy attempt was accepted")
	}
}

func propose(t *testing.T, manager *Manager, now time.Time, expected, change string, desired *config.AuthorizationConfig, operation string) Proposal {
	t.Helper()
	proposal, err := manager.Propose(identity.Subject{ID: "owner", Kind: "owner", Authority: "test"}, ProposalRequest{
		Protocol: ProposalRequestProtocol, OperationID: operation, RequestID: "request-" + operation,
		CorrelationID: "correlation-" + operation, AuthorityBasis: "host_owner",
		ExpectedPolicyDigest: expected, Change: change, DesiredPolicy: desired,
		RequestedAt: now, ExpiresAt: now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return proposal
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	return config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "development",
		TOPS:   config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen: config.ListenConfig{Network: "unix", Address: filepath.Join(t.TempDir(), "ssiag.sock")},
		Authentication: &config.AuthenticationConfig{
			Mechanism: "unix_peer_credentials",
			Service:   &config.SubjectConfig{ID: config.ServiceSubjectID, Kind: config.ServiceSubjectKind, UID: &uid, GID: &gid},
			Subjects:  []config.SubjectConfig{{ID: "operator.primary", Kind: "operator", UID: &uid, GID: &gid}},
		},
		Authorization: &config.AuthorizationConfig{DefaultEffect: "deny", MaxCapabilitySeconds: 900, Grants: []config.AuthorizationGrant{}},
		Providers:     []config.ProviderConfig{},
	}
}

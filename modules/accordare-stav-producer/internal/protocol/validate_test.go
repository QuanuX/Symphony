package protocol

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const testTOPS = "11111111-1111-4111-8111-111111111111"

func TestVerifySubmissionClosesCallerControlledVocabulary(t *testing.T) {
	submission, subject := validSubmission(t)
	verified, err := VerifySubmission(submission, subject, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if verified.Candidate.Operation.EventClass != "symphony.sav.named-version.lifecycle" ||
		verified.Candidate.Operation.OperationID != "symphony.sav.named-version.prepare" ||
		verified.Candidate.Result.ReasonCode != "symphony.sav.named-version.prepare.succeeded" {
		t.Fatalf("producer did not derive the closed vocabulary: %+v", verified.Candidate)
	}
	if strings.Contains(string(mustJSON(t, verified.Candidate)), "authorization_decision") {
		t.Fatal("authorization proof leaked into the safe candidate")
	}
}

func TestVerifySubmissionRejectsResultAndAuthorityDrift(t *testing.T) {
	submission, subject := validSubmission(t)
	var result map[string]any
	if err := json.Unmarshal(submission.Result, &result); err != nil {
		t.Fatal(err)
	}
	result["changed"] = false
	submission.Result = mustJSON(t, result)
	if _, err := VerifySubmission(submission, subject, time.Now().UTC()); err == nil {
		t.Fatal("result mutation under the original digest was accepted")
	}

	submission, subject = validSubmission(t)
	var command map[string]any
	if err := json.Unmarshal(submission.Command, &command); err != nil {
		t.Fatal(err)
	}
	decision := command["authorization_decision"].(map[string]any)
	decision["caller_class_used"] = true
	submission.Command = mustJSON(t, command)
	if _, err := VerifySubmission(submission, subject, time.Now().UTC()); err == nil {
		t.Fatal("caller-class-dependent authority was accepted")
	}
}

func TestStableCandidateIdentityMakesRetryIdempotent(t *testing.T) {
	submission, subject := validSubmission(t)
	first, err := VerifySubmission(submission, subject, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	second, err := VerifySubmission(submission, subject, time.Now().UTC().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if first.Candidate.Correlation != second.Candidate.Correlation || first.CandidateDigest != second.CandidateDigest {
		t.Fatal("an exact retry produced different STAV identity")
	}
}

func TestIntentPrecedesAndBindsCompletion(t *testing.T) {
	completed, subject := validSubmission(t)
	prepared := completed
	prepared.Result = nil
	prepared.Outcome = nil
	prepared.ReasonCode = nil
	when := time.Now().UTC().Truncate(time.Second)
	intent, err := VerifyIntent(prepared, subject, when)
	if err != nil {
		t.Fatal(err)
	}
	verified, completedID, err := VerifyCompletion(intent.Submission, completed, subject, when, when)
	if err != nil {
		t.Fatal(err)
	}
	if completedID != intent.IntentID || verified.CandidateDigest == "" {
		t.Fatal("completion was not bound to its durable intent")
	}
	completed.Coordinator.Version = "0.2.0"
	if _, _, err := VerifyCompletion(intent.Submission, completed, subject, when, when); err == nil {
		t.Fatal("coordinator drift was accepted")
	}
}

func TestTypedUnavailableCompletionContainsNoRawError(t *testing.T) {
	completed, subject := validSubmission(t)
	completed.Result = nil
	outcome, reason := "unavailable", "symphony.sav.named-version.prepare.unavailable"
	completed.Outcome, completed.ReasonCode = &outcome, &reason
	verified, err := VerifySubmission(completed, subject, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if verified.Candidate.Result.Outcome != outcome || verified.Candidate.Result.ReasonCode != reason || verified.Candidate.Configuration.PreviousDigest != verified.Candidate.Configuration.NewDigest {
		t.Fatalf("typed unavailable candidate is unsafe: %+v", verified.Candidate)
	}
	if strings.Contains(string(mustJSON(t, verified.Candidate)), "error") {
		t.Fatal("terminal candidate leaked raw error material")
	}
}

func TestIntentSemanticIdentityAllowsFreshAuthorizationOnly(t *testing.T) {
	left, _ := validSubmission(t)
	var command map[string]any
	if err := json.Unmarshal(left.Command, &command); err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), left.Command...)
	command["authorization_decision"] = map[string]any{"fresh": "replacement-proof"}
	if !CommandsMatchIntent(original, mustJSON(t, command)) {
		t.Fatal("fresh authorization changed stable intent semantics")
	}
	command["expected_registry_digest"] = "sha256:" + strings.Repeat("9", 64)
	if CommandsMatchIntent(original, mustJSON(t, command)) {
		t.Fatal("mutation semantic drift was accepted")
	}
}

func validSubmission(t *testing.T) (Submission, stavprotocol.SafeReference) {
	t.Helper()
	subject := stavprotocol.SafeReference{ID: "owner.test", Kind: "owner"}
	operationID := "operation-test-1"
	versionDigest := "sha256:" + strings.Repeat("1", 64)
	proposalDigest := "sha256:" + strings.Repeat("2", 64)
	command := map[string]any{
		"alias": nil, "authorization_decision": nil,
		"client": map[string]any{"client_id": "qxctl"}, "expected_registry_digest": "absent",
		"named_version": map[string]any{"named_version_digest": versionDigest}, "operation": "named_version_prepare",
		"operation_id": operationID, "prepared_operation_id": nil, "proposal_digest": nil,
		"protocol": "symphony.knowledge.named-version-command.v1", "sav_engine": map[string]any{"engine_id": "sav"},
		"selector": nil, "state_root": "/tmp/state", "tops_id": testTOPS,
		"validation_result": map[string]any{"valid": true},
	}
	resource := namedVersionResource(mustJSON(t, command))
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(5 * time.Minute)
	target := decisionTarget{Operation: "symphony.knowledge.named_version_prepare", Resource: resource, Audience: "qxctl", Scope: "tops:" + testTOPS}
	decisionSubject := decisionSubject{ID: subject.ID, Kind: subject.Kind, Authority: "unix_peer_credentials"}
	capability := capability{
		Protocol: "symphony.ssiag.capability.v1", Subject: decisionSubject, TOPSID: testTOPS, Target: target,
		AuthorityBasis: "host_owner", GrantID: "accordare-test", RequestID: "ssiag-request-test",
		CorrelationID: "ssiag-correlation-test", IssuedAt: now, ExpiresAt: expires,
		PolicyDigest: "sha256:" + strings.Repeat("a", 64), ConfigDigest: "sha256:" + strings.Repeat("b", 64),
	}
	capability.BindingDigest = capabilityBinding(capability)
	capability.CapabilityID = "ssiag-capability:" + strings.TrimPrefix(capability.BindingDigest, "sha256:")
	basis := "host_owner"
	command["authorization_decision"] = authorizationDecision{
		Schema: "symphony.ssiag.authorization-decision.v1", DecisionID: "ssiag-decision:test",
		RequestID: capability.RequestID, CorrelationID: capability.CorrelationID, TOPSID: testTOPS,
		Subject: decisionSubject, Target: target, Effect: "allow", ReasonCode: "symphony.ssiag.policy.exact-grant",
		AuthorityBasis: &basis, Capability: &capability, PolicyDigest: capability.PolicyDigest,
		ConfigDigest: capability.ConfigDigest, DecidedAt: now, ExpiresAt: &expires,
	}
	result := map[string]any{
		"alias_count": 0, "artifact": nil, "canonical": false, "canonical_apply_enabled": false,
		"changed": true, "compatibility": map[string]any{"mode": "full"}, "format_version": 1,
		"operation": "named_version_prepare", "proposal_digest": proposalDigest, "read_only": false,
		"recovered": false, "registry_digest": nil, "registry_present": false, "repair_actions": []any{},
		"result_digest": "", "selected_alias": nil, "stav_append_enabled": false, "protocol": resultProtocol,
		"version_count": 0,
	}
	resultBytes := mustJSON(t, result)
	digest, err := digestWithout(resultBytes, "result_digest")
	if err != nil {
		t.Fatal(err)
	}
	result["result_digest"] = digest
	outcome, reason := "succeeded", "symphony.sav.named-version.prepare.succeeded"
	return Submission{Command: mustJSON(t, command), Coordinator: InstallationEvidence{ComponentID: "knowledge-session-coordinator", ExecutableDigest: "sha256:" + strings.Repeat("c", 64), ReceiptDigest: "sha256:" + strings.Repeat("d", 64), Version: "0.1.0"}, Operation: "named_version_prepare", Outcome: &outcome, ReasonCode: &reason, Result: mustJSON(t, result), Schema: SchemaSubmission, TOPSID: testTOPS}, subject
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

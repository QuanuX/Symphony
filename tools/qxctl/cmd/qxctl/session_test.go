package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
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

func transitionInvocation(t *testing.T, state, digest string, operations ...string) sessionInvocation {
	t.Helper()
	present := state != "absent"
	changed := state != "absent" && state != "open-status"
	recovered := false
	readOnly := false
	apply := false
	canonical := false
	effective := state
	if state == "open-status" {
		effective = "open"
		changed = false
		readOnly = true
	}
	var journalDigest *string
	journal := json.RawMessage("null")
	if present {
		value := digest
		journalDigest = &value
		checkpoints := make([]map[string]any, 0, len(operations))
		for _, operation := range operations {
			checkpoints = append(checkpoints, map[string]any{"operation_id": operation})
		}
		encoded, err := json.Marshal(map[string]any{
			"protocol":       "symphony.knowledge.session-journal.v1",
			"journal_digest": digest, "canonical": false, "checkpoints": checkpoints,
		})
		if err != nil {
			t.Fatal(err)
		}
		journal = encoded
	}
	return sessionInvocation{Result: sessionResult{
		Protocol: "symphony.knowledge.session-result.v1", Operation: "session_status",
		JournalPresent: &present, Journal: journal, JournalDigest: journalDigest,
		EffectiveState: effective, Changed: &changed, Recovered: &recovered,
		RepairActions: []string{}, ReadOnly: &readOnly, ApplyEnabled: &apply, Canonical: &canonical,
	}}
}

func TestSessionTransitionLoginClosesStaleEpochAndBeginsLinkedEpoch(t *testing.T) {
	const first = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	const closed = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	const begun = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	calls := make([]struct {
		operation string
		options   knowledgeSessionOptions
	}, 0, 3)
	results := []sessionInvocation{
		transitionInvocation(t, "open-status", first, "prior:begin"),
		transitionInvocation(t, "closed", closed, "prior:begin", "login-2:close-prior"),
		transitionInvocation(t, "open", begun, "login-2:begin"),
	}
	invoke := func(operation string, options knowledgeSessionOptions) (sessionInvocation, error) {
		calls = append(calls, struct {
			operation string
			options   knowledgeSessionOptions
		}{operation, options})
		result := results[len(calls)-1]
		return result, nil
	}
	report, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		topsID: ssiagTestTOPSID, scope: "user", event: "login", eventID: "login-2",
		contextRefs: []string{"reconcile:one"}, ttl: 15 * time.Minute,
	}, invoke)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || calls[0].operation != "status" || calls[1].operation != "close" ||
		calls[1].options.expectedJournalDigest != first || calls[2].operation != "begin" ||
		calls[2].options.expectedJournalDigest != closed || len(calls[2].options.contextRefs) != 1 {
		t.Fatalf("unexpected transition calls: %#v", calls)
	}
	if report.Disposition != "begun" || report.InitialState != "open" || report.FinalState != "open" ||
		report.FinalJournalDigest == nil || *report.FinalJournalDigest != begun || !validTaggedDigest(report.ResultDigest) {
		t.Fatalf("unexpected transition report: %#v", report)
	}
}

func TestSessionTransitionRetryIsNoOp(t *testing.T) {
	const digest = "sha256:4444444444444444444444444444444444444444444444444444444444444444"
	calls := 0
	invoke := func(operation string, options knowledgeSessionOptions) (sessionInvocation, error) {
		calls++
		if operation != "status" || options.operationID != "" {
			t.Fatalf("unexpected retry invocation: %s %#v", operation, options)
		}
		return transitionInvocation(t, "open-status", digest, "stable-login:begin"), nil
	}
	report, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		topsID: ssiagTestTOPSID, scope: "user", event: "login", eventID: "stable-login", ttl: time.Minute,
	}, invoke)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || report.Disposition != "already_applied" || len(report.Steps) != 1 {
		t.Fatalf("retry was not idempotent: calls=%d report=%#v", calls, report)
	}
}

func TestSessionTransitionRefreshRotatesWhenReauthenticationIsRequired(t *testing.T) {
	const first = "sha256:5555555555555555555555555555555555555555555555555555555555555555"
	const closed = "sha256:6666666666666666666666666666666666666666666666666666666666666666"
	const begun = "sha256:7777777777777777777777777777777777777777777777777777777777777777"
	calls := 0
	invoke := func(operation string, options knowledgeSessionOptions) (sessionInvocation, error) {
		calls++
		switch calls {
		case 1:
			return transitionInvocation(t, "open-status", first, "prior:begin"), nil
		case 2:
			return sessionInvocation{}, &knowledgeengine.ProcessError{
				Code: "session.reauthentication_required", Message: "configuration changed",
			}
		case 3:
			if operation != "close" || options.expectedJournalDigest != first {
				t.Fatalf("unexpected close: %s %#v", operation, options)
			}
			return transitionInvocation(t, "closed", closed, "refresh-1:close"), nil
		case 4:
			if operation != "begin" || options.expectedJournalDigest != closed {
				t.Fatalf("unexpected begin: %s %#v", operation, options)
			}
			return transitionInvocation(t, "open", begun, "refresh-1:begin"), nil
		default:
			return sessionInvocation{}, errors.New("unexpected invocation")
		}
	}
	report, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		topsID: ssiagTestTOPSID, scope: "user", event: "refresh", eventID: "refresh-1", ttl: time.Minute,
	}, invoke)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 || report.Disposition != "reauthenticated" || report.FinalState != "open" {
		t.Fatalf("refresh did not rotate: calls=%d report=%#v", calls, report)
	}
}

func TestSessionTransitionRefreshResumesAfterCommittedClose(t *testing.T) {
	const closed = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const begun = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	calls := 0
	invoke := func(operation string, options knowledgeSessionOptions) (sessionInvocation, error) {
		calls++
		switch calls {
		case 1:
			return transitionInvocation(t, "closed", closed, "refresh-resume:close"), nil
		case 2:
			if operation != "begin" || options.operationID != "refresh-resume:begin" ||
				options.expectedJournalDigest != closed {
				t.Fatalf("unexpected resumed begin: %s %#v", operation, options)
			}
			return transitionInvocation(t, "open", begun, "refresh-resume:begin"), nil
		default:
			return sessionInvocation{}, errors.New("unexpected invocation")
		}
	}
	report, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		topsID: ssiagTestTOPSID, scope: "user", event: "refresh", eventID: "refresh-resume",
		contextRefs: []string{"context:resume"}, ttl: time.Minute,
	}, invoke)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || report.Disposition != "reauthenticated" || report.FinalState != "open" ||
		report.FinalJournalDigest == nil || *report.FinalJournalDigest != begun {
		t.Fatalf("refresh did not resume its committed close: calls=%d report=%#v", calls, report)
	}
}

func TestSessionTransitionLogoutAbsentIsStableNoOp(t *testing.T) {
	calls := 0
	invoke := func(string, knowledgeSessionOptions) (sessionInvocation, error) {
		calls++
		return transitionInvocation(t, "absent", ""), nil
	}
	report, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		topsID: ssiagTestTOPSID, scope: "user", event: "logout", eventID: "logout-1", ttl: time.Minute,
	}, invoke)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || report.Disposition != "no_change" || report.FinalState != "absent" {
		t.Fatalf("absent logout was not stable: calls=%d report=%#v", calls, report)
	}
}

func TestSessionTransitionRecoveryIsExplicitAndBounded(t *testing.T) {
	const recoveredDigest = "sha256:8888888888888888888888888888888888888888888888888888888888888888"
	const begunDigest = "sha256:9999999999999999999999999999999999999999999999999999999999999999"
	calls := 0
	invoke := func(operation string, options knowledgeSessionOptions) (sessionInvocation, error) {
		calls++
		switch calls {
		case 1:
			return sessionInvocation{}, &knowledgeengine.ProcessError{Code: "session.head_invalid", Message: "damaged head"}
		case 2:
			if operation != "recover" || !options.discover || options.operationID != "login-recover:recover" {
				t.Fatalf("unexpected recovery invocation: %s %#v", operation, options)
			}
			value := transitionInvocation(t, "closed", recoveredDigest, "prior:recover")
			*value.Result.Recovered = true
			return value, nil
		case 3:
			return transitionInvocation(t, "closed", recoveredDigest, "prior:recover"), nil
		case 4:
			return transitionInvocation(t, "open", begunDigest, "login-recover:begin"), nil
		default:
			return sessionInvocation{}, errors.New("unexpected invocation")
		}
	}
	report, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		topsID: ssiagTestTOPSID, scope: "user", event: "login", eventID: "login-recover",
		recoverTransition: true, ttl: time.Minute,
	}, invoke)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 || !report.Recovered || report.InitialState != "closed" || report.Disposition != "begun" {
		t.Fatalf("recovery transition mismatch: calls=%d report=%#v", calls, report)
	}
}

func TestSessionTransitionRejectsUnstableEventIdentity(t *testing.T) {
	for _, test := range []struct {
		name    string
		event   string
		eventID string
		message string
	}{
		{name: "unknown event", event: "boot", eventID: "event-1", message: "--event must"},
		{name: "unsafe id", event: "login", eventID: "not valid", message: "--event-id"},
		{name: "oversize id", event: "login", eventID: strings.Repeat("a", 225), message: "--event-id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
				event: test.event, eventID: test.eventID, ttl: time.Minute,
			}, func(string, knowledgeSessionOptions) (sessionInvocation, error) {
				return sessionInvocation{}, nil
			})
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("unstable event identity was accepted: %v", err)
			}
		})
	}
}

func TestSessionTransitionDoesNotRecoverAmbiguousState(t *testing.T) {
	calls := 0
	_, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		event: "login", eventID: "ambiguous-login", recoverTransition: true, ttl: time.Minute,
	}, func(string, knowledgeSessionOptions) (sessionInvocation, error) {
		calls++
		return sessionInvocation{}, &knowledgeengine.ProcessError{
			Code: "session.recovery_ambiguous", Message: "two successors",
		}
	})
	if err == nil || calls != 1 {
		t.Fatalf("ambiguous state was converted into recovery: calls=%d err=%v", calls, err)
	}
}

func TestSessionTransitionRefreshDoesNotAdoptUnrelatedClosedEpoch(t *testing.T) {
	const digest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	calls := 0
	_, err := performKnowledgeSessionTransition(knowledgeSessionOptions{
		event: "refresh", eventID: "new-refresh", ttl: time.Minute,
	}, func(string, knowledgeSessionOptions) (sessionInvocation, error) {
		calls++
		return transitionInvocation(t, "closed", digest, "older-refresh:close"), nil
	})
	if err == nil || !strings.Contains(err.Error(), "submit a login transition") || calls != 1 {
		t.Fatalf("unrelated closed epoch was adopted: calls=%d err=%v", calls, err)
	}
}

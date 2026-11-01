package foundation

import (
	"context"
	"errors"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/foundationaudit"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"os"
	"strings"
	"testing"
	"time"
)

func auditReceipt(t *testing.T, request foundationaudit.Request) foundationaudit.Response {
	t.Helper()
	candidate, err := foundationaudit.Candidate(request, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	candidateDigest, err := stav.CandidateDigest(candidate)
	if err != nil {
		t.Fatal(err)
	}
	return foundationaudit.Response{Schema: foundationaudit.Schema, Phase: request.Phase, RequestDigest: foundationaudit.Hash(request), Receipt: stav.Receipt{Schema: stav.SchemaReceipt, TOPSID: request.TOPSID, RequestID: request.RequestID, CandidateDigest: candidateDigest, Disposition: "committed", ReasonCode: stav.ReasonReceiptCommitted, Commit: stav.CommitResult{State: "committed", Sequence: 1, EventID: "123e4567-e89b-42d3-a456-426614174003", EventDigest: "sha256:" + strings.Repeat("c", 64), Timestamp: "2026-11-01T12:00:00.000000000Z"}}}
}

func ordinaryCommand(t *testing.T) Command {
	t.Helper()
	observed, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	command := commandAt("plan", "enrollment", time.Now().UTC().Truncate(time.Second))
	command.ExpectedStateDigest = &observed.StableStateDigest
	command.Intent = &Intent{DesiredState: "enrolled", AuditMode: "ordinary", TTLSeconds: 60}
	planned, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	command.Operation = "apply"
	command.Intent = nil
	command.Plan = planned.Plan
	absent := "absent"
	command.ExpectedAttemptDigest = &absent
	return command
}
func TestOrdinarySTAVLifecycleOutcomePendingAndRecover(t *testing.T) {
	setupInstalled(t)
	command := ordinaryCommand(t)
	previous := submitFoundationAudit
	t.Cleanup(func() { submitFoundationAudit = previous })
	available := false
	calls := []string{}
	submitFoundationAudit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		calls = append(calls, r.Phase)
		if r.Phase == "outcome" && !available {
			return foundationaudit.Response{}, errors.New("audit stopped")
		}
		return auditReceipt(t, r), nil
	}
	applied, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "intent" || calls[1] != "outcome" || applied.AuditState != "audit_deferred" || applied.AuditReceiptDigest != nil || !applied.ReconciliationRequired {
		t.Fatalf("intent falsely certified mutation outcome: %+v calls=%v", applied, calls)
	}
	replay, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || len(calls) != 2 {
		t.Fatal("repeat re-executed mutation/audit")
	}
	command.Operation = "recover"
	command.Plan = nil
	command.Discover = true
	pending, err := Execute(command)
	if err == nil || pending.Recovered || !pending.ReconciliationRequired {
		t.Fatalf("unavailable reconciliation accepted: %+v %v", pending, err)
	}
	available = true
	recovered, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Recovered || recovered.Replayed || recovered.AuditState != "reconciled" || recovered.AuditReceiptDigest == nil || recovered.ReconciliationRequired {
		t.Fatalf("outcome reconciliation not committed: %+v", recovered)
	}
	count := len(calls)
	if _, err = Execute(command); err != nil {
		t.Fatal(err)
	}
	if len(calls) != count {
		t.Fatal("reconciliation appended twice")
	}
}
func TestOrdinarySTAVLostIntentReplyRecoverDoesNotBypassGate(t *testing.T) {
	setupInstalled(t)
	command := ordinaryCommand(t)
	previous := submitFoundationAudit
	t.Cleanup(func() { submitFoundationAudit = previous })
	calls := 0
	var identity string
	submitFoundationAudit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		calls++
		if r.Phase == "intent" {
			if identity != "" && identity != foundationaudit.Hash(r) {
				t.Fatal("retry changed candidate")
			}
			identity = foundationaudit.Hash(r)
			if calls == 1 {
				return foundationaudit.Response{}, errors.New("reply lost")
			}
		}
		return auditReceipt(t, r), nil
	}
	result, err := Execute(command)
	if err == nil || !result.RecoveryRequired {
		t.Fatalf("lost intent reply not retained: %+v %v", result, err)
	}
	observed, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	if observed.Enrollment.State == "enrolled" {
		t.Fatal("mutation crossed uncertain intent gate")
	}
	command.Operation = "recover"
	command.Plan = nil
	command.Discover = true
	recovered, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Recovered || recovered.AuditState != "reconciled" || calls != 3 {
		t.Fatalf("recovery failed exact intent/outcome path: %+v calls=%d", recovered, calls)
	}
}

func TestOutcomeRecoveryCASUsesLatestNativeAttemptAfterInterruptedJournalBinding(t *testing.T) {
	setupInstalled(t)
	command := ordinaryCommand(t)
	previous := submitFoundationAudit
	t.Cleanup(func() { submitFoundationAudit = previous })
	submitFoundationAudit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		return auditReceipt(t, r), nil
	}
	if _, err := Execute(command); err != nil {
		t.Fatal(err)
	}
	layout, _ := stavpaths.ResolveInstance(stavpaths.ScopeUser, foundationTOPSID)
	latest, _, err := readAttempt(layout.LastAttempt)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate last native write succeeding immediately before journal rebinding.
	latest.UpdatedAt = formatTimestamp(time.Now().UTC().Add(time.Second))
	setAttemptDigest(latest)
	if err = writeAttempt(layout.LastAttempt, *latest); err != nil {
		t.Fatal(err)
	}
	command.Operation = "recover"
	command.Plan = nil
	command.Discover = false
	command.ExpectedAttemptDigest = &latest.AttemptDigest
	result, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Recovered || result.ReconciliationRequired {
		t.Fatalf("exact latest native CAS was rejected: %+v", result)
	}
}

func TestSTAVIntentWaitCannotCrossCommandDeadline(t *testing.T) {
	setupInstalled(t)
	command := ordinaryCommand(t)
	previous := submitFoundationAudit
	t.Cleanup(func() { submitFoundationAudit = previous })
	deadline := time.Now().UTC().Truncate(time.Second).Add(time.Second)
	command.DeadlineAt = formatTimestamp(deadline)
	submitFoundationAudit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		time.Sleep(time.Until(deadline.Add(10 * time.Millisecond)))
		return auditReceipt(t, r), nil
	}
	result, err := Execute(command)
	if err == nil || result.Error == nil || result.Error.Code != "deadline_elapsed" || !result.RecoveryRequired {
		t.Fatalf("expired intent crossed mutation gate: %+v %v", result, err)
	}
	observed, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	if observed.Enrollment.State == "enrolled" {
		t.Fatal("expired command mutated")
	}
}

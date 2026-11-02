package foundationlifecycle

import (
	"context"
	"errors"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationaudit"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
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

func TestOrdinaryLifecycleReceiptsAndPendingOutcomeRecoverAfterRestart(t *testing.T) {
	engine, _ := setupEngine(t)
	command, _ := planEnrollment(t, engine, "enrolled")
	command.Plan.AuditMode = "ordinary"
	command.Plan.PlanDigest, _ = digestWithout(*command.Plan, "plan_digest")
	calls := []string{}
	outcomeAvailable := false
	submit := func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		calls = append(calls, r.Phase)
		if r.Phase == "outcome" && !outcomeAvailable {
			return foundationaudit.Response{}, errors.New("audit stopped")
		}
		return auditReceipt(t, r), nil
	}
	engine.audit = submit
	originalMutate := engine.mutate
	mutations := 0
	engine.mutate = func(p Plan) error {
		mutations++
		if len(calls) != 1 || calls[0] != "intent" {
			t.Fatal("native mutation preceded committed intent")
		}
		return originalMutate(p)
	}
	applied, err := engine.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if mutations != 1 || applied.AuditState != "audit_deferred" || !applied.ReconciliationRequired || applied.AuditReceiptDigest != nil {
		t.Fatalf("intent masqueraded as completion: %+v", applied)
	}
	replay, err := engine.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || mutations != 1 {
		t.Fatal("repeat mutated again")
	}
	// A fresh engine uses durable outcome, never repeats native mutation.
	restarted := New()
	restarted.now = engine.now
	restarted.audit = submit
	restarted.mutate = func(Plan) error { t.Fatal("reconciliation reran native mutation"); return nil }
	command.Operation = "recover"
	command.Plan = nil
	command.Discover = true
	pending, err := restarted.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Disposition != "blocked" || pending.AuditState != "audit_deferred" || pending.Recovered {
		t.Fatalf("unavailable reconciliation claimed completion: %+v", pending)
	}
	outcomeAvailable = true
	recovered, err := restarted.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.AuditState != "reconciled" || recovered.AuditReceiptDigest == nil || recovered.ReconciliationRequired || !recovered.Recovered {
		t.Fatalf("missing real outcome receipt: %+v", recovered)
	}
	count := len(calls)
	again, err := restarted.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if again.AuditReceiptDigest == nil || *again.AuditReceiptDigest != *recovered.AuditReceiptDigest || len(calls) != count {
		t.Fatal("reconciliation was not idempotent")
	}
}

func TestOrdinaryLifecycleMismatchedReceiptCannotMutate(t *testing.T) {
	engine, _ := setupEngine(t)
	command, _ := planEnrollment(t, engine, "enrolled")
	command.Plan.AuditMode = "ordinary"
	command.Plan.PlanDigest, _ = digestWithout(*command.Plan, "plan_digest")
	engine.audit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		response := auditReceipt(t, r)
		response.Receipt.CandidateDigest = "sha256:" + strings.Repeat("d", 64)
		return response, nil
	}
	engine.mutate = func(Plan) error { t.Fatal("native mutation admitted mismatched receipt"); return nil }
	result, err := engine.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Error == nil || result.Error.Code != "audit.unavailable" || !result.RecoveryRequired || result.AuditReceiptDigest != nil {
		t.Fatalf("mismatch was not recoverably refused: %+v", result)
	}
}

func TestObservedOutcomeCrashBeforeNativeClosureFinishesWithoutMutation(t *testing.T) {
	engine, _ := setupEngine(t)
	command, _ := planEnrollment(t, engine, "enrolled")
	engine.audit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		return auditReceipt(t, r), nil
	}
	if _, err := engine.Execute(command); err != nil {
		t.Fatal(err)
	}
	store, err := openAttemptStore(mustInstallLayout(parseScope("user")).LifecycleDir, testTOPSID, "enrollment", true)
	if err != nil {
		t.Fatal(err)
	}
	attempt, present, err := store.read()
	if err != nil || !present {
		t.Fatal(err)
	}
	// Exact crash boundary: observed outcome persisted, native close not yet
	// written. The current attempt differs from the journal's earlier CAS value.
	attempt.Phase = "observing"
	attempt.CompletedAt = nil
	attempt.AuditState = "audit_deferred"
	attempt.AuditReceiptDigest = nil
	if err = store.write(&attempt); err != nil {
		t.Fatal(err)
	}
	store.close()
	engine.mutate = func(Plan) error { t.Fatal("durable observed execution was replayed"); return nil }
	command.Operation = "recover"
	command.Plan = nil
	command.Discover = false
	command.ExpectedAttemptDigest = &attempt.AttemptDigest
	result, err := engine.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Recovered || result.RecoveryRequired || result.ReconciliationRequired {
		t.Fatalf("native close remained active: %+v", result)
	}
	store, err = openAttemptStore(mustInstallLayout(parseScope("user")).LifecycleDir, testTOPSID, "enrollment", false)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	closed, _, err := store.read()
	if err != nil || closed.Phase != "closed" {
		t.Fatalf("recovery did not close exact attempt: %+v %v", closed, err)
	}
}

func TestExplicitRestartPreservesAndReconcilesOriginalStopOutcome(t *testing.T) {
	engine, _ := setupEngine(t)
	running := true
	nativeMutations := 0
	observe := engine.observeFn
	engine.observeFn = func(surface string, scope ssiagpaths.Scope, tops string, attempt *Attempt) (Observation, error) {
		value, err := observe(surface, scope, tops, attempt)
		if err != nil {
			return value, err
		}
		value.Supervisor.ManagerState = "available"
		value.Supervisor.DescriptorState = "installed"
		value.Supervisor.ProcessState = "stopped"
		value.Supervisor.EndpointState = "absent"
		if running {
			value.Supervisor.ProcessState = "running"
			value.Supervisor.EndpointState = "ready"
		}
		value.StableStateDigest, _ = digestWithoutFields(value, "observed_at", "stable_state_digest", "observation_digest")
		value.ObservationDigest, _ = digestWithout(value, "observation_digest")
		return value, nil
	}
	engine.mutate = func(plan Plan) error { nativeMutations++; running = plan.DesiredState == "native_running"; return nil }
	var stoppedEvidence string
	engine.audit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		if r.Phase == "outcome" && r.DesiredState == "native_installed_stopped" {
			if stoppedEvidence != "" && stoppedEvidence != foundationaudit.Hash(r) {
				t.Fatal("restart rewrote original stop observation")
			}
			stoppedEvidence = foundationaudit.Hash(r)
		}
		if !running {
			return foundationaudit.Response{}, errors.New("audit service stopped")
		}
		return auditReceipt(t, r), nil
	}
	run := func(op, request, desired, mode string) Result {
		t.Helper()
		observation, err := engine.observeFn("supervisor", parseScope("user"), testTOPSID, nil)
		if err != nil {
			t.Fatal(err)
		}
		correlation := "23b79938-b558-4e52-9a16-2fe36b32c6a1"
		command := Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "plan", Component: "ssiag", Surface: "supervisor", Scope: "user", TOPSID: testTOPSID, OperationID: &op, RequestID: &request, CorrelationID: &correlation, ExpectedStateDigest: &observation.StableStateDigest, Intent: &Intent{DesiredState: desired, AuditMode: mode, TTLSeconds: 300}, RequestedAt: "2026-08-13T12:00:00Z", DeadlineAt: "2026-08-13T12:05:00Z"}
		planned, err := engine.Execute(command)
		if err != nil || planned.Plan == nil {
			t.Fatalf("plan failed: %+v %v", planned, err)
		}
		command.Operation = "apply"
		command.Intent = nil
		command.Plan = planned.Plan
		absent := "absent"
		command.ExpectedAttemptDigest = &absent
		result, err := engine.Execute(command)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	stopped := run("test:stop", "123e4567-e89b-42d3-a456-426614174001", "native_installed_stopped", "ordinary")
	if stopped.AuditState != "audit_deferred" || stopped.AuditReceiptDigest != nil || running {
		t.Fatalf("stop result false: %+v", stopped)
	}
	started := run("test:restart", "123e4567-e89b-42d3-a456-426614174002", "native_running", "audit_deferred")
	if started.AuditState != "reconciled" || !running || nativeMutations != 2 {
		t.Fatalf("explicit dependency restoration failed: %+v", started)
	}
	op := "test:stop"
	recovered, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "recover", Component: "ssiag", Surface: "supervisor", Scope: "user", TOPSID: testTOPSID, OperationID: &op, ExpectedAttemptDigest: stopped.AttemptDigest, RequestedAt: "2026-08-13T12:00:00Z", DeadlineAt: "2026-08-13T12:05:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if recovered.AuditState != "reconciled" || nativeMutations != 2 || !running || recovered.Observation.Supervisor.ProcessState != "running" {
		t.Fatalf("historical stop reconciliation rewrote current state: %+v", recovered)
	}
}

func TestIntentCommitCannotAuthorizeMutationPastCommandDeadline(t *testing.T) {
	engine, _ := setupEngine(t)
	command, _ := planEnrollment(t, engine, "enrolled")
	command.Plan.AuditMode = "ordinary"
	command.Plan.PlanDigest, _ = digestWithout(*command.Plan, "plan_digest")
	original := engine.now
	engine.audit = func(_ context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
		engine.now = func() time.Time { return original().Add(6 * time.Minute) }
		return auditReceipt(t, r), nil
	}
	engine.mutate = func(Plan) error { t.Fatal("expired command mutated after waiting for receipt"); return nil }
	result, err := engine.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Error == nil || result.Error.Code != "deadline.elapsed" || !result.RecoveryRequired {
		t.Fatalf("expired committed intent not retained for explicit recovery: %+v", result)
	}
}

package foundationlifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/lifecycle"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const testTOPSID = "018f0c3a-7b2d-4e11-8c12-0242ac120002"

func setupEngine(t *testing.T) (*Engine, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	source := filepath.Join(home, "source-ssiag")
	if err := os.WriteFile(source, []byte("ssiag-test-executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.Install(source, ssiagpaths.ScopeUser, false); err != nil {
		t.Fatal(err)
	}
	engine := New()
	engine.now = func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) }
	return engine, home
}

func planEnrollment(t *testing.T, engine *Engine, desired string) (Command, Result) {
	t.Helper()
	now, deadline := "2026-08-13T12:00:00Z", "2026-08-13T12:05:00Z"
	observed, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "observe", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	op := "test:enrollment:" + desired
	request := "0a4d72cf-0dc5-4b4a-8bfa-6e6cf27a67c1"
	correlation := "23b79938-b558-4e52-9a16-2fe36b32c6a1"
	name := "Test TOPS"
	intent := &Intent{DesiredState: desired, AuditMode: "audit_deferred", TTLSeconds: 300}
	if desired == "enrolled" {
		intent.TOPSName = &name
	}
	expected := observed.Observation.StableStateDigest
	planned, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "plan", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &op, RequestID: &request, CorrelationID: &correlation, ExpectedStateDigest: &expected, Intent: intent, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	absent := "absent"
	return Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "apply", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &op, ExpectedAttemptDigest: &absent, Plan: planned.Plan, RequestedAt: now, DeadlineAt: deadline}, planned
}

func TestEnrollmentApplyReplayAndStableDigests(t *testing.T) {
	engine, _ := setupEngine(t)
	apply, planned := planEnrollment(t, engine, "enrolled")
	if planned.Plan == nil || !validDigest(planned.Plan.PlanDigest) || !validDigest(planned.ResultDigest) {
		t.Fatal("plan did not carry stable digests")
	}
	result, err := engine.Execute(apply)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "applied" || !result.Changed || result.RecoveryRequired || !result.ReconciliationRequired || result.AuditState != "audit_deferred" || !validDigest(result.ResultDigest) || !validSTSC(result.CompletedAt) {
		t.Fatalf("unexpected apply result: %+v", result)
	}
	replay, err := engine.Execute(apply)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Disposition != "already_applied" || !replay.Replayed || replay.Changed {
		t.Fatalf("unexpected replay: %+v", replay)
	}
}

func TestCrashAfterMutationRecoversFromExactAttempt(t *testing.T) {
	engine, _ := setupEngine(t)
	apply, _ := planEnrollment(t, engine, "enrolled")
	plan := *apply.Plan
	before, err := engine.observe("enrollment", ssiagpaths.ScopeUser, testTOPSID, nil)
	if err != nil {
		t.Fatal(err)
	}
	install := mustInstallLayout(ssiagpaths.ScopeUser)
	store, err := openAttemptStore(install.LifecycleDir, testTOPSID, "enrollment", true)
	if err != nil {
		t.Fatal(err)
	}
	attempt := &Attempt{Protocol: AttemptProtocol, FormatVersion: 1, Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: plan.OperationID, RequestID: plan.RequestID, CorrelationID: plan.CorrelationID, Phase: "mutating", PlanDigest: plan.PlanDigest, PriorStateDigest: before.StableStateDigest, DesiredState: plan.DesiredState, BinaryDigest: *before.Installation.BinaryDigest, InstallEvidenceDigest: *before.Installation.InstallEvidenceDigest, AuditState: "audit_deferred", StartedAt: "2026-08-13T12:00:00Z", UpdatedAt: "2026-08-13T12:00:00Z"}
	if err := store.writePlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := store.write(attempt); err != nil {
		t.Fatal(err)
	}
	if err := engine.mutatePlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	discover := true
	recoverCommand := Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "recover", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &plan.OperationID, Discover: discover, RequestedAt: "2026-08-13T12:00:00Z", DeadlineAt: "2026-08-13T12:05:00Z"}
	result, err := engine.Execute(recoverCommand)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "recovered" || !result.Recovered || result.Changed || result.RecoveryRequired || !result.ReconciliationRequired || result.AuditState != "audit_deferred" {
		t.Fatalf("unexpected recovery: %+v", result)
	}
}

func TestApplyRefusesObservationDrift(t *testing.T) {
	engine, _ := setupEngine(t)
	apply, _ := planEnrollment(t, engine, "enrolled")
	install := mustInstallLayout(ssiagpaths.ScopeUser)
	if err := os.WriteFile(install.Binary, []byte("drifted"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := engine.Execute(apply)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "blocked" || result.Error == nil || result.Error.Code != "state.compare-and-swap" {
		t.Fatalf("drift was not refused: %+v", result)
	}
}

func TestSupervisorApplyReportsManagerUnavailableWithoutAttempt(t *testing.T) {
	engine, _ := setupEngine(t)
	if _, err := lifecycle.Enroll(ssiagpaths.ScopeUser, testTOPSID, "Test TOPS", nil, nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	now, deadline := "2026-08-13T12:00:00Z", "2026-08-13T12:05:00Z"
	observed, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "observe", Component: "ssiag", Surface: "supervisor", Scope: "user", TOPSID: testTOPSID, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	if observed.Observation.Supervisor.ManagerState != "manager_unavailable" {
		t.Fatalf("manager state = %s", observed.Observation.Supervisor.ManagerState)
	}
	op, request, correlation := "test:supervisor:start", "0a4d72cf-0dc5-4b4a-8bfa-6e6cf27a67c1", "23b79938-b558-4e52-9a16-2fe36b32c6a1"
	expected := observed.Observation.StableStateDigest
	planned, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "plan", Component: "ssiag", Surface: "supervisor", Scope: "user", TOPSID: testTOPSID, OperationID: &op, RequestID: &request, CorrelationID: &correlation, ExpectedStateDigest: &expected, Intent: &Intent{DesiredState: "native_running", AuditMode: "audit_deferred", TTLSeconds: 300}, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	absent := "absent"
	result, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "apply", Component: "ssiag", Surface: "supervisor", Scope: "user", TOPSID: testTOPSID, OperationID: &op, ExpectedAttemptDigest: &absent, Plan: planned.Plan, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "blocked" || result.Error == nil || result.Error.Code != "manager.unavailable" || result.AttemptDigest != nil {
		t.Fatalf("unexpected unavailable result: %+v", result)
	}
}

func TestStableDigestIncludesRecoveryAndIgnoresJSONFieldOrder(t *testing.T) {
	observation := Observation{Protocol: ObservationProtocol, FormatVersion: 1, Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, Installation: InstallationObservation{State: "absent", Legacy: true}, Enrollment: EnrollmentObservation{State: "unenrolled"}, Supervisor: SupervisorObservation{ManagerState: "unknown", DescriptorState: "absent", Enablement: "unknown", ProcessState: "unknown", EndpointState: "unknown"}, ObservedAt: "2026-08-13T12:00:00Z"}
	base, err := digestWithoutFields(observation, "observed_at", "stable_state_digest", "observation_digest")
	if err != nil {
		t.Fatal(err)
	}
	observation.RecoveryRequired = true
	observation.ActiveAttemptDigest = stringPointer("sha256:" + strings.Repeat("0", 64))
	changed, err := digestWithoutFields(observation, "observed_at", "stable_state_digest", "observation_digest")
	if err != nil {
		t.Fatal(err)
	}
	if base == changed {
		t.Fatal("recovery metadata did not invalidate stable state")
	}
	var left, right map[string]any
	if err := json.Unmarshal([]byte(`{"b":2,"a":1}`), &left); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"a":1,"b":2}`), &right); err != nil {
		t.Fatal(err)
	}
	leftDigest, _ := digestValue(left)
	rightDigest, _ := digestValue(right)
	if leftDigest != rightDigest {
		t.Fatal("JSON field order changed canonical digest")
	}
}

func TestInterruptedMutationRequiresRecovery(t *testing.T) {
	engine, _ := setupEngine(t)
	apply, _ := planEnrollment(t, engine, "enrolled")
	engine.mutate = func(Plan) error { return errors.New("simulated interruption") }
	result, err := engine.Execute(apply)
	if err != nil {
		t.Fatal(err)
	}
	if !result.RecoveryRequired || result.AttemptDigest == nil || result.Disposition != "failed" {
		t.Fatalf("interruption was not preserved: %+v", result)
	}
}

func TestOrdinaryAuditFailsBeforeAttemptOrMutation(t *testing.T) {
	engine, _ := setupEngine(t)
	mutated := false
	engine.mutate = func(Plan) error { mutated = true; return nil }
	now, deadline := "2026-08-13T12:00:00Z", "2026-08-13T12:05:00Z"
	observed, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "observe", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	op, request, correlation := "test:ordinary", "0a4d72cf-0dc5-4b4a-8bfa-6e6cf27a67c1", "23b79938-b558-4e52-9a16-2fe36b32c6a1"
	name, expected := "Test TOPS", observed.Observation.StableStateDigest
	planned, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "plan", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &op, RequestID: &request, CorrelationID: &correlation, ExpectedStateDigest: &expected, Intent: &Intent{DesiredState: "enrolled", TOPSName: &name, AuditMode: "ordinary", TTLSeconds: 300}, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	absent := "absent"
	result, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "apply", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &op, ExpectedAttemptDigest: &absent, Plan: planned.Plan, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "blocked" || result.Error == nil || result.Error.Code != "audit.unavailable" || mutated {
		t.Fatalf("ordinary audit did not fail before mutation: %+v mutated=%t", result, mutated)
	}
	store, err := openAttemptStore(mustInstallLayout(ssiagpaths.ScopeUser).LifecycleDir, testTOPSID, "enrollment", false)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	if _, present, err := store.read(); err != nil || present {
		t.Fatalf("ordinary audit created an attempt: present=%t err=%v", present, err)
	}
}

func TestExecuteRejectsExpiredDeadlineBeforeStateAccess(t *testing.T) {
	engine, _ := setupEngine(t)
	_, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "observe", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, RequestedAt: "2026-08-13T11:58:00Z", DeadlineAt: "2026-08-13T11:59:59Z"})
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("expired deadline accepted: %v", err)
	}
}

func TestReceiptV2EvidenceWorksWithoutLegacyInstallation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	installed, err := packageinstall.Install(source, filepath.Join(home, "prefix"), "dev")
	if err != nil {
		t.Fatal(err)
	}
	engine := New()
	engine.now = func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) }
	binaryDigest, err := digestFile(installed.Binary)
	if err != nil {
		t.Fatal(err)
	}
	receiptDigest := installed.ReceiptDigest
	engine.evidence = func(ssiagpaths.Scope) (invocationEvidence, error) {
		return invocationEvidence{Binary: installed.Binary, BinaryDigest: binaryDigest, InstallEvidenceDigest: receiptDigest, ReceiptDigest: &receiptDigest, Version: "dev", State: "installed", Verified: true}, nil
	}
	apply, _ := planEnrollment(t, engine, "enrolled")
	result, err := engine.Execute(apply)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "applied" || result.Observation.Installation.State != "installed" || result.Observation.Installation.Legacy || result.Observation.Installation.BinaryPath == nil || *result.Observation.Installation.BinaryPath != installed.Binary {
		t.Fatalf("receipt-v2 apply used wrong installation identity: %+v", result)
	}
	legacy := mustInstallLayout(ssiagpaths.ScopeUser)
	if _, err := os.Lstat(legacy.Binary); !os.IsNotExist(err) {
		t.Fatalf("receipt-v2 apply unexpectedly required legacy binary: %v", err)
	}
}

func TestBlockedRecoverCarriesActiveRecoveryEvidence(t *testing.T) {
	engine, _ := setupEngine(t)
	apply, _ := planEnrollment(t, engine, "enrolled")
	engine.mutate = func(Plan) error { return errors.New("simulated interruption") }
	failed, err := engine.Execute(apply)
	if err != nil {
		t.Fatal(err)
	}
	if !failed.RecoveryRequired || failed.AttemptDigest == nil {
		t.Fatalf("failed apply did not establish recovery evidence: %+v", failed)
	}
	wrongOperation := "test:different-recovery-owner"
	blocked, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "recover", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &wrongOperation, Discover: true, RequestedAt: "2026-08-13T12:00:00Z", DeadlineAt: "2026-08-13T12:05:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Disposition != "blocked" || blocked.Error == nil || blocked.Error.Code != "operation.binding-mismatch" || !blocked.RecoveryRequired || !blocked.Observation.RecoveryRequired || blocked.AttemptDigest == nil || *blocked.AttemptDigest != *failed.AttemptDigest || blocked.Observation.ActiveAttemptDigest == nil || *blocked.Observation.ActiveAttemptDigest != *failed.AttemptDigest {
		t.Fatalf("blocked recover lost active recovery evidence: %+v", blocked)
	}
}

func TestDeferredAuditAttemptCannotBeOverwrittenBeforeReconciliation(t *testing.T) {
	engine, _ := setupEngine(t)
	firstApply, _ := planEnrollment(t, engine, "enrolled")
	first, err := engine.Execute(firstApply)
	if err != nil {
		t.Fatal(err)
	}
	if first.AttemptDigest == nil || !first.ReconciliationRequired {
		t.Fatalf("first operation lacks deferred evidence: %+v", first)
	}
	secondApply, _ := planEnrollment(t, engine, "unenrolled_preserved")
	blocked, err := engine.Execute(secondApply)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Disposition != "blocked" || blocked.Error == nil || blocked.Error.Code != "audit.reconciliation-required" || !blocked.ReconciliationRequired || blocked.AuditState != "audit_deferred" || blocked.AttemptDigest == nil || *blocked.AttemptDigest != *first.AttemptDigest {
		t.Fatalf("unreconciled audit attempt was not preserved: %+v", blocked)
	}
	statusOperation := "test:status"
	status, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "apply_status", Component: "ssiag", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &statusOperation, RequestedAt: "2026-08-13T12:00:00Z", DeadlineAt: "2026-08-13T12:05:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if status.AttemptDigest == nil || *status.AttemptDigest != *first.AttemptDigest || !status.ReconciliationRequired {
		t.Fatalf("deferred attempt was overwritten: %+v", status)
	}
}

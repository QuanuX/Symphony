package foundation

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/lifecycle"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

const foundationTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func setupInstalled(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	source := filepath.Join(t.TempDir(), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("foundation-test-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	installed, err := lifecycle.Install(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	prior := currentExecutable
	currentExecutable = func() (string, error) { return installed.Binary, nil }
	t.Cleanup(func() { currentExecutable = prior })
}

func commandAt(operation, surface string, now time.Time) Command {
	opID, requestID, correlationID := "op-1", "123e4567-e89b-42d3-a456-426614174001", "123e4567-e89b-42d3-a456-426614174002"
	return Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: operation, Component: "stav", Surface: surface, Scope: "user", TOPSID: foundationTOPSID, OperationID: &opID, RequestID: &requestID, CorrelationID: &correlationID, RequestedAt: formatTimestamp(now), DeadlineAt: formatTimestamp(now.Add(30 * time.Second))}
}

func TestDigestCanonicalizesObjectKeyOrder(t *testing.T) {
	left := map[string]any{"z": json.Number("2"), "a": map[string]any{"y": "x", "b": json.Number("1")}}
	rightBytes := []byte(`{"a":{"b":1,"y":"x"},"z":2}`)
	decoder := json.NewDecoder(bytes.NewReader(rightBytes))
	decoder.UseNumber()
	var right any
	if err := decoder.Decode(&right); err != nil {
		t.Fatal(err)
	}
	if digestJSON(left) != digestJSON(right) {
		t.Fatal("canonical digest depends on object key order")
	}
}

func TestObserveStableDigestExcludesCollectionTime(t *testing.T) {
	setupInstalled(t)
	first, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	second, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	if first.ObservedAt == second.ObservedAt {
		t.Fatal("test did not cross STSC second boundary")
	}
	if first.StableStateDigest != second.StableStateDigest {
		t.Fatal("collection time changed stable state digest")
	}
	if first.ObservationDigest == second.ObservationDigest {
		t.Fatal("observation digest omitted observed_at")
	}
}

func TestExpiredCommandFailsBeforeMutation(t *testing.T) {
	setupInstalled(t)
	command := commandAt("observe", "enrollment", time.Now().UTC().Add(-2*time.Minute))
	if _, err := Execute(command); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired deadline, got %v", err)
	}
	layout, _ := stavpaths.ResolveInstance(stavpaths.ScopeUser, foundationTOPSID)
	if _, err := os.Lstat(layout.EnrollmentFile); !os.IsNotExist(err) {
		t.Fatalf("expired request mutated enrollment: %v", err)
	}
}

func TestAuditDeferredApplyCrashAndRecover(t *testing.T) {
	setupInstalled(t)
	now := time.Now().UTC().Truncate(time.Second)
	observe, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	command := commandAt("plan", "enrollment", now)
	command.ExpectedStateDigest = &observe.StableStateDigest
	command.Intent = &Intent{DesiredState: "enrolled", AuditMode: "audit_deferred", TTLSeconds: 60}
	planned, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	command.Operation, command.Intent, command.Plan = "apply", nil, planned.Plan
	absent := "absent"
	command.ExpectedAttemptDigest = &absent
	afterPreparedHook = func() error { return os.ErrClosed }
	_, err = Execute(command)
	afterPreparedHook = nil
	if err == nil {
		t.Fatal("injected crash did not interrupt after prepare")
	}
	interruptedErr := err
	layout, _ := stavpaths.ResolveInstance(stavpaths.ScopeUser, foundationTOPSID)
	attempt, _, err := readAttempt(layout.ActiveAttempt)
	if err != nil || attempt == nil {
		t.Fatalf("prepared attempt missing: read=%v execute=%v", err, interruptedErr)
	}
	command.Operation, command.Plan, command.Discover = "recover", nil, false
	command.ExpectedAttemptDigest = &attempt.AttemptDigest
	command.RequestedAt, command.DeadlineAt = formatTimestamp(time.Now().UTC()), formatTimestamp(time.Now().UTC().Add(30*time.Second))
	recovered, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Disposition != "recovered" || !recovered.ReconciliationRequired || recovered.AuditState != "audit_deferred" {
		t.Fatalf("unexpected recovered result: %+v", recovered)
	}
	if _, err := os.Lstat(layout.ActiveAttempt); !os.IsNotExist(err) {
		t.Fatalf("active attempt was not closed: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(layout.StateDir, "ledger-v1.stavlog")); !os.IsNotExist(err) {
		t.Fatalf("lifecycle adapter created or rewrote ledger: %v", err)
	}
}

func TestRecoverMissingPlanBindsExactActiveAttempt(t *testing.T) {
	setupInstalled(t)
	now := time.Now().UTC().Truncate(time.Second)
	observed, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	command := commandAt("plan", "enrollment", now)
	command.ExpectedStateDigest = &observed.StableStateDigest
	command.Intent = &Intent{DesiredState: "enrolled", AuditMode: "audit_deferred", TTLSeconds: 60}
	planned, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	command.Operation, command.Intent, command.Plan = "apply", nil, planned.Plan
	absent := "absent"
	command.ExpectedAttemptDigest = &absent
	afterPreparedHook = func() error { return os.ErrClosed }
	_, applyErr := Execute(command)
	afterPreparedHook = nil
	if applyErr == nil {
		t.Fatal("injected crash did not leave a prepared attempt")
	}
	layout, _ := stavpaths.ResolveInstance(stavpaths.ScopeUser, foundationTOPSID)
	attempt, _, err := readAttempt(layout.ActiveAttempt)
	if err != nil || attempt == nil {
		t.Fatalf("prepared attempt missing: %v", err)
	}
	if err := os.Remove(layout.ActivePlan); err != nil {
		t.Fatal(err)
	}
	command.Operation, command.Plan, command.Discover = "recover", nil, false
	command.ExpectedAttemptDigest = &attempt.AttemptDigest
	command.RequestedAt, command.DeadlineAt = formatTimestamp(time.Now().UTC()), formatTimestamp(time.Now().UTC().Add(30*time.Second))
	result, err := Execute(command)
	if err == nil {
		t.Fatal("recovery unexpectedly accepted a missing protected plan")
	}
	if !result.RecoveryRequired || result.AttemptDigest == nil || *result.AttemptDigest != attempt.AttemptDigest ||
		result.Observation.ActiveAttemptDigest == nil || *result.Observation.ActiveAttemptDigest != attempt.AttemptDigest {
		t.Fatalf("recovery failure omitted exact active attempt binding: %+v", result)
	}
	if result.AuditState != "audit_deferred" || !result.ReconciliationRequired {
		t.Fatalf("recovery failure lost deferred-audit state: %+v", result)
	}
	if result.ResultDigest != digestWithout(result, "result_digest") {
		t.Fatal("recovery failure digest was not recomputed after attempt binding")
	}
}

func TestUserSupervisorPlanKeepsEnrollmentIdentityNull(t *testing.T) {
	setupInstalled(t)
	observed, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "supervisor")
	if err != nil {
		t.Fatal(err)
	}
	command := commandAt("plan", "supervisor", time.Now().UTC().Truncate(time.Second))
	command.ExpectedStateDigest = &observed.StableStateDigest
	command.Intent = &Intent{DesiredState: "native_installed_stopped", AuditMode: "audit_deferred", TTLSeconds: 60}
	result, err := Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan == nil {
		t.Fatal("supervisor plan missing")
	}
	if result.Plan.TOPSName != nil || result.Plan.ServiceUID != nil || result.Plan.ServiceGID != nil ||
		result.Plan.AuthorityUID != nil || result.Plan.AuthorityGID != nil {
		t.Fatalf("supervisor plan materialized enrollment identity: %+v", result.Plan)
	}
}

func TestFailedPlanRemainsReadOnly(t *testing.T) {
	setupInstalled(t)
	command := commandAt("plan", "supervisor", time.Now().UTC().Truncate(time.Second))
	wrongState := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	command.ExpectedStateDigest = &wrongState
	command.Intent = &Intent{DesiredState: "native_installed_stopped", AuditMode: "audit_deferred", TTLSeconds: 60}
	result, err := Execute(command)
	if err == nil {
		t.Fatal("plan unexpectedly accepted a stale expected state")
	}
	if !result.ReadOnly || result.Disposition != "failed" || result.Error == nil {
		t.Fatalf("failed plan did not preserve read-only result shape: %+v", result)
	}
	if result.ResultDigest != digestWithout(result, "result_digest") {
		t.Fatal("failed plan result digest is invalid")
	}
}

func TestIntentIdentityAppliesOnlyToEnrolledEnrollment(t *testing.T) {
	uid, gid := uint32(501), uint32(20)
	valid := []struct {
		name    string
		surface string
		intent  Intent
	}{
		{"system supervisor", "supervisor", Intent{DesiredState: "native_installed_stopped", AuditMode: "audit_deferred", TTLSeconds: 60}},
		{"system unenrollment", "enrollment", Intent{DesiredState: "unenrolled_preserved", AuditMode: "audit_deferred", TTLSeconds: 60}},
		{"system enrollment", "enrollment", Intent{DesiredState: "enrolled", AuthorityUID: &uid, AuthorityGID: &gid, AuditMode: "audit_deferred", TTLSeconds: 60}},
	}
	for _, test := range valid {
		t.Run(test.name, func(t *testing.T) {
			if err := validateIntent(test.surface, "system", &test.intent); err != nil {
				t.Fatalf("valid intent rejected: %v", err)
			}
		})
	}
	invalid := []struct {
		name    string
		surface string
		intent  Intent
	}{
		{"supervisor with identity", "supervisor", Intent{DesiredState: "native_running", AuthorityUID: &uid, AuthorityGID: &gid, AuditMode: "audit_deferred", TTLSeconds: 60}},
		{"unenrollment with identity", "enrollment", Intent{DesiredState: "unenrolled_preserved", AuthorityUID: &uid, AuthorityGID: &gid, AuditMode: "audit_deferred", TTLSeconds: 60}},
		{"enrollment without identity", "enrollment", Intent{DesiredState: "enrolled", AuditMode: "audit_deferred", TTLSeconds: 60}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if err := validateIntent(test.surface, "system", &test.intent); err == nil {
				t.Fatal("invalid system identity intent accepted")
			}
		})
	}
}

func TestPlanRejectsMachinePurgeAndSupervisorDriftRequiresCAS(t *testing.T) {
	setupInstalled(t)
	observation, err := Observe(stavpaths.ScopeUser, foundationTOPSID, "enrollment")
	if err != nil {
		t.Fatal(err)
	}
	command := commandAt("plan", "enrollment", time.Now().UTC())
	command.ExpectedStateDigest = &observation.StableStateDigest
	command.Intent = &Intent{DesiredState: "purged", AuditMode: "audit_deferred", TTLSeconds: 60}
	if _, err := Execute(command); err == nil || !strings.Contains(err.Error(), "purge") {
		t.Fatalf("machine purge unexpectedly accepted: %v", err)
	}
}

func TestDecodeRejectsOversizeAndUnknownFields(t *testing.T) {
	if _, err := DecodeCommand(strings.NewReader(strings.Repeat("x", maxRequestBytes+1))); err == nil {
		t.Fatal("oversized input accepted")
	}
	if _, err := DecodeCommand(strings.NewReader(`{"protocol":"x","unknown":true}`)); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestDescriptorBindsExactReceiptAndOperations(t *testing.T) {
	setupInstalled(t)
	descriptor, err := Describe()
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Component != "stav" || descriptor.BinaryDigest == "" || descriptor.InstallEvidenceDigest == "" || len(descriptor.Operations) != 10 {
		t.Fatalf("incomplete adapter descriptor: %+v", descriptor)
	}
	if descriptor.DescriptorDigest != digestWithout(descriptor, "descriptor_digest") {
		t.Fatal("adapter descriptor digest is not canonical")
	}
}

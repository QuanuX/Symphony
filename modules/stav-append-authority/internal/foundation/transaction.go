package foundation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/lifecycle"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/supervision"
)

func apply(command Command, base Result) (Result, error) {
	plan := command.Plan
	if err := validatePlan(command, plan); err != nil {
		result := failedResult(base, "invalid_plan", err, true)
		return result, err
	}
	layout, err := stavpaths.ResolveInstance(stavpaths.Scope(command.Scope), command.TOPSID)
	if err != nil {
		return base, err
	}
	if err := ensureTransactionDirectory(layout); err != nil {
		return base, err
	}
	lease, err := acquireLifecycleLease(layout.LifecycleLock)
	if err != nil {
		result := failedResult(base, "operation_busy", err, true)
		return result, err
	}
	defer lease.Close()

	current, err := Observe(stavpaths.Scope(command.Scope), command.TOPSID, command.Surface)
	if err != nil {
		return base, err
	}
	base.Observation, base.Plan, base.DesiredState = current, plan, &plan.DesiredState
	if active, _, activeErr := readAttempt(layout.ActiveAttempt); activeErr != nil {
		result := failedResult(base, "recovery_required", activeErr, true)
		result.RecoveryRequired = true
		return result, activeErr
	} else if active != nil {
		if command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != active.AttemptDigest {
			err := fmt.Errorf("apply expected attempt digest does not match active attempt")
			result := failedResult(base, "state_conflict", err, true)
			return result, err
		}
		if active.OperationID != plan.OperationID || active.PlanDigest != plan.PlanDigest {
			err := fmt.Errorf("another unresolved lifecycle attempt requires recovery")
			result := failedResult(base, "recovery_required", err, true)
			result.RecoveryRequired, result.AttemptDigest = true, &active.AttemptDigest
			return result, err
		}
		if desiredSatisfied(current, active.DesiredState) {
			return closeAttempt(layout, *active, *plan, base, current, true, false)
		}
		return resumePrepared(layout, *active, *plan, base, current, true, false)
	}
	if command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != "absent" {
		err := fmt.Errorf("apply requires expected_attempt_digest absent when no attempt is active")
		result := failedResult(base, "state_conflict", err, true)
		return result, err
	}
	if plan.ExpectedStateDigest != current.StableStateDigest {
		err := fmt.Errorf("apply expected state digest does not match current STAV state")
		result := failedResult(base, "state_conflict", err, true)
		return result, err
	}
	_, installRecord, evidenceDigest, err := currentInstall(stavpaths.Scope(command.Scope))
	if err != nil {
		result := failedResult(base, "installation_unverified", err, true)
		return result, err
	}
	attempt := Attempt{Protocol: AttemptProtocol, FormatVersion: 1, Component: "stav", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: plan.OperationID, RequestID: plan.RequestID, CorrelationID: plan.CorrelationID, Phase: "prepared", PlanDigest: plan.PlanDigest, PriorStateDigest: current.StableStateDigest, DesiredState: plan.DesiredState, BinaryDigest: "sha256:" + installRecord.BinarySHA256, InstallEvidenceDigest: evidenceDigest, AuditState: plan.AuditMode, StartedAt: canonicalNow(), UpdatedAt: canonicalNow()}
	if plan.AuditMode == "ordinary" {
		err := fmt.Errorf("ordinary foundational mutation requires a committed closed SSIAG/STAV receipt")
		result := failedResult(base, "audit_unavailable", err, true)
		return result, err
	}
	setAttemptDigest(&attempt)
	if err := writeAttempt(layout.ActiveAttempt, attempt); err != nil {
		return base, err
	}
	if err := writeProtectedJSON(layout.ActivePlan, plan); err != nil {
		return base, err
	}
	if afterPreparedHook != nil {
		if err := afterPreparedHook(); err != nil {
			return base, err
		}
	}
	return resumePrepared(layout, attempt, *plan, base, current, false, false)
}

func validatePlan(command Command, plan *Plan) error {
	if plan == nil || plan.Protocol != PlanProtocol || plan.FormatVersion != 1 || plan.Component != "stav" || plan.Surface != command.Surface || plan.Scope != command.Scope || plan.TOPSID != command.TOPSID || command.OperationID == nil || plan.OperationID != *command.OperationID {
		return fmt.Errorf("plan identity does not match apply command")
	}
	if plan.PlanDigest != digestWithout(*plan, "plan_digest") {
		return fmt.Errorf("plan digest is invalid")
	}
	created, createdErr := parseTimestamp(plan.CreatedAt)
	expires, expiresErr := parseTimestamp(plan.ExpiresAt)
	if createdErr != nil || expiresErr != nil || !expires.After(created) || expires.Sub(created) > 10*time.Minute || !expires.After(time.Now().UTC()) {
		return fmt.Errorf("plan is expired")
	}
	if !tokenPattern.MatchString(plan.OperationID) || !uuidPattern.MatchString(plan.RequestID) || !uuidPattern.MatchString(plan.CorrelationID) || !digestPattern.MatchString(plan.ExpectedStateDigest) {
		return fmt.Errorf("plan identities or expected digest are invalid")
	}
	intent := &Intent{DesiredState: plan.DesiredState, TOPSName: plan.TOPSName, ServiceUID: plan.ServiceUID, ServiceGID: plan.ServiceGID, AuthorityUID: plan.AuthorityUID, AuthorityGID: plan.AuthorityGID, AuditMode: plan.AuditMode, TTLSeconds: 1}
	if plan.Surface == "enrollment" && plan.DesiredState == "enrolled" && plan.Scope == "user" {
		if plan.AuthorityUID == nil || plan.AuthorityGID == nil || *plan.AuthorityUID != uint32(os.Geteuid()) || *plan.AuthorityGID != uint32(os.Getegid()) {
			return fmt.Errorf("user plan authority does not match kernel credentials")
		}
		intent.AuthorityUID, intent.AuthorityGID = nil, nil
	}
	return validateIntent(plan.Surface, plan.Scope, intent)
}

func resumePrepared(layout stavpaths.InstanceLayout, attempt Attempt, plan Plan, base Result, prior Observation, replayed, recovered bool) (Result, error) {
	attempt.Phase, attempt.UpdatedAt = "mutating", canonicalNow()
	setAttemptDigest(&attempt)
	if err := writeAttempt(layout.ActiveAttempt, attempt); err != nil {
		return base, err
	}
	preMutation, err := Observe(stavpaths.Scope(plan.Scope), plan.TOPSID, plan.Surface)
	if err != nil {
		return base, err
	}
	if stateBeforeActiveAttempt(preMutation) != attempt.PriorStateDigest {
		err := fmt.Errorf("observed STAV state changed after attempt preparation")
		attempt.Phase, attempt.UpdatedAt = "recovery_required", canonicalNow()
		setAttemptDigest(&attempt)
		_ = writeAttempt(layout.ActiveAttempt, attempt)
		result := failedResult(base, "state_conflict", err, false)
		result.Observation, result.RecoveryRequired, result.AttemptDigest, result.AuditState = preMutation, true, &attempt.AttemptDigest, attempt.AuditState
		result.ReconciliationRequired = attempt.AuditState == "audit_deferred"
		finalizeResult(&result)
		return result, err
	}
	changed, err := mutate(plan, prior)
	if err != nil {
		attempt.Phase, attempt.UpdatedAt = "recovery_required", canonicalNow()
		setAttemptDigest(&attempt)
		_ = writeAttempt(layout.ActiveAttempt, attempt)
		result := failedResult(base, sanitizeCode(err), err, false)
		result.RecoveryRequired, result.AttemptDigest, result.AuditState = true, &attempt.AttemptDigest, attempt.AuditState
		result.ReconciliationRequired = attempt.AuditState == "audit_deferred"
		finalizeResult(&result)
		return result, err
	}
	attempt.Phase, attempt.UpdatedAt = "observing", canonicalNow()
	setAttemptDigest(&attempt)
	if err := writeAttempt(layout.ActiveAttempt, attempt); err != nil {
		return base, err
	}
	after, err := Observe(stavpaths.Scope(plan.Scope), plan.TOPSID, plan.Surface)
	if err != nil {
		return base, err
	}
	if !desiredSatisfied(after, plan.DesiredState) {
		err := fmt.Errorf("post-mutation observation does not prove desired STAV state")
		attempt.Phase, attempt.UpdatedAt = "recovery_required", canonicalNow()
		setAttemptDigest(&attempt)
		_ = writeAttempt(layout.ActiveAttempt, attempt)
		result := failedResult(base, "recovery_required", err, false)
		result.Observation, result.RecoveryRequired, result.AttemptDigest, result.AuditState = after, true, &attempt.AttemptDigest, attempt.AuditState
		result.ReconciliationRequired = attempt.AuditState == "audit_deferred"
		finalizeResult(&result)
		return result, err
	}
	base.Changed = changed
	return closeAttempt(layout, attempt, plan, base, after, replayed, recovered)
}

func mutate(plan Plan, prior Observation) (bool, error) {
	scope := stavpaths.Scope(plan.Scope)
	switch plan.Surface {
	case "enrollment":
		if plan.DesiredState == "enrolled" {
			if prior.Enrollment.State == "enrolled" {
				return false, nil
			}
			if plan.AuthorityUID == nil || plan.AuthorityGID == nil {
				return false, fmt.Errorf("enrollment authority identity is missing")
			}
			install, _, _, verifyErr := currentInstall(scope)
			if verifyErr != nil {
				return false, verifyErr
			}
			_, err := lifecycle.EnrollWithExecutable(scope, plan.TOPSID, uint64(*plan.AuthorityUID), uint64(*plan.AuthorityGID), install.Binary)
			return err == nil, err
		}
		if prior.Enrollment.State == "unenrolled" {
			return false, nil
		}
		_, err := lifecycle.Unenroll(scope, plan.TOPSID, false)
		return err == nil, err
	case "supervisor":
		layout, err := stavpaths.ResolveInstance(scope, plan.TOPSID)
		if err != nil {
			return false, err
		}
		cfg, err := config.Load(layout.ConfigFile)
		if err != nil {
			return false, err
		}
		if err := config.ValidateLayout(cfg, layout); err != nil {
			return false, err
		}
		install, _, _, err := currentInstall(scope)
		if err != nil {
			return false, err
		}
		spec, err := supervision.SpecFromConfig(scope, plan.TOPSID, install.Binary, cfg)
		if err != nil {
			return false, err
		}
		expected := "absent"
		if prior.Supervisor.DescriptorDigest != nil {
			expected = *prior.Supervisor.DescriptorDigest
		}
		if plan.DesiredState == "absent_stopped" {
			record, err := supervision.UninstallCAS(spec, expected, true)
			return record.Changed, err
		}
		record, err := supervision.InstallCAS(spec, expected)
		if err != nil {
			return false, err
		}
		if plan.DesiredState == "native_running" {
			if err := supervision.Start(record); err != nil {
				return record.Changed, err
			}
		} else if prior.Supervisor.ProcessState == "running" {
			if err := supervision.Stop(record); err != nil {
				return record.Changed, err
			}
		}
		return record.Changed || prior.Supervisor.ProcessState != desiredProcess(plan.DesiredState), nil
	}
	return false, fmt.Errorf("unsupported mutation surface")
}

func desiredProcess(state string) string {
	if state == "native_running" {
		return "running"
	}
	return "stopped"
}

func closeAttempt(layout stavpaths.InstanceLayout, attempt Attempt, plan Plan, base Result, observation Observation, replayed, recovered bool) (Result, error) {
	completed := canonicalNow()
	attempt.Phase, attempt.UpdatedAt, attempt.CompletedAt = "mutation_closed", completed, &completed
	if attempt.AuditState == "ordinary" {
		attempt.AuditState = "pending"
	}
	base.Plan, base.DesiredState = &plan, &plan.DesiredState
	base.Disposition, base.Replayed, base.Recovered = "applied", replayed, recovered
	if replayed {
		base.Disposition = "already_applied"
	}
	if recovered {
		base.Disposition = "recovered"
	}
	base.AuditState = attempt.AuditState
	base.ReconciliationRequired = attempt.AuditState == "audit_deferred"
	base.StartedAt, base.CompletedAt = &attempt.StartedAt, completed
	setAttemptDigest(&attempt)
	mutationDigest := attempt.AttemptDigest
	if err := writeAttempt(layout.LastAttempt, attempt); err != nil {
		return base, err
	}
	if err := os.Remove(layout.ActiveAttempt); err != nil && !os.IsNotExist(err) {
		return base, err
	}
	if err := os.Remove(layout.ActivePlan); err != nil && !os.IsNotExist(err) {
		return base, err
	}
	if err := syncDirectory(layout.LifecycleDir); err != nil {
		return base, err
	}
	closedObservation, err := Observe(stavpaths.Scope(plan.Scope), plan.TOPSID, plan.Surface)
	if err != nil {
		return base, err
	}
	base.Observation = closedObservation
	base.AttemptDigest = &mutationDigest
	finalizeResult(&base)
	attempt.PredecessorDigest = &mutationDigest
	attempt.ResultDigest = &base.ResultDigest
	setAttemptDigest(&attempt)
	if err := writeAttempt(layout.LastAttempt, attempt); err != nil {
		return base, err
	}
	archive := filepath.Join(layout.AttemptDir, attempt.AttemptDigest[len("sha256:"):]+".json")
	if err := writeAttempt(archive, attempt); err != nil {
		return base, err
	}
	return base, nil
}

func applyStatus(command Command, base Result) (Result, error) {
	base.ReadOnly = true
	layout, err := stavpaths.ResolveInstance(stavpaths.Scope(command.Scope), command.TOPSID)
	if err != nil {
		return base, err
	}
	if err := ensureTransactionDirectory(layout); err != nil {
		return base, err
	}
	lease, err := acquireLifecycleLease(layout.LifecycleLock)
	if err != nil {
		result := failedResult(base, "operation_busy", err, true)
		return result, err
	}
	defer lease.Close()
	current, err := Observe(stavpaths.Scope(command.Scope), command.TOPSID, command.Surface)
	if err != nil {
		return base, err
	}
	base.Observation = current
	attempt, _, err := readAttempt(layout.ActiveAttempt)
	if err != nil {
		result := failedResult(base, "recovery_required", err, false)
		return result, err
	}
	if attempt == nil {
		attempt, _, err = readAttempt(layout.LastAttempt)
	}
	if err != nil || attempt == nil {
		err = fmt.Errorf("no lifecycle attempt is available")
		result := failedResult(base, "attempt_absent", err, true)
		return result, err
	}
	base.OperationID, base.DesiredState, base.AttemptDigest, base.StartedAt = &attempt.OperationID, &attempt.DesiredState, &attempt.AttemptDigest, &attempt.StartedAt
	base.AuditState, base.ReconciliationRequired = attempt.AuditState, attempt.AuditState == "audit_deferred"
	base.RecoveryRequired = attempt.Phase != "mutation_closed" && attempt.Phase != "closed"
	base.Disposition, base.ReadOnly = "observed", true
	finalizeResult(&base)
	return base, nil
}

func recoverAttempt(command Command, base Result) (Result, error) {
	layout, err := stavpaths.ResolveInstance(stavpaths.Scope(command.Scope), command.TOPSID)
	if err != nil {
		return base, err
	}
	if err := ensureTransactionDirectory(layout); err != nil {
		return base, err
	}
	lease, err := acquireLifecycleLease(layout.LifecycleLock)
	if err != nil {
		return base, err
	}
	defer lease.Close()
	current, err := Observe(stavpaths.Scope(command.Scope), command.TOPSID, command.Surface)
	if err != nil {
		return base, err
	}
	base.Observation = current
	attempt, _, err := readAttempt(layout.ActiveAttempt)
	if err != nil || attempt == nil {
		if err == nil {
			err = fmt.Errorf("no active lifecycle attempt requires recovery")
		}
		result := failedResult(base, "attempt_absent", err, true)
		return result, err
	}
	if !command.Discover {
		if command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != attempt.AttemptDigest {
			err := fmt.Errorf("expected attempt digest does not match active attempt")
			result := failedResult(base, "state_conflict", err, true)
			result.AuditState, result.ReconciliationRequired = attempt.AuditState, attempt.AuditState == "audit_deferred"
			finalizeResult(&result)
			return result, err
		}
	}
	if command.OperationID == nil || *command.OperationID != attempt.OperationID || command.Surface != attempt.Surface {
		err := fmt.Errorf("recovery command does not match active attempt identity")
		result := failedResult(base, "state_conflict", err, true)
		result.AuditState, result.ReconciliationRequired = attempt.AuditState, attempt.AuditState == "audit_deferred"
		finalizeResult(&result)
		return result, err
	}
	plan, err := readPlan(layout.ActivePlan)
	if err != nil || plan.PlanDigest != attempt.PlanDigest {
		if err == nil {
			err = fmt.Errorf("active attempt plan digest is inconsistent")
		}
		result := failedResult(base, "recovery_required", err, false)
		result.RecoveryRequired, result.AttemptDigest, result.AuditState = true, &attempt.AttemptDigest, attempt.AuditState
		result.ReconciliationRequired = attempt.AuditState == "audit_deferred"
		finalizeResult(&result)
		return result, err
	}
	current, err = Observe(stavpaths.Scope(command.Scope), command.TOPSID, command.Surface)
	if err != nil {
		return base, err
	}
	base.Observation, base.Plan, base.DesiredState = current, &plan, &plan.DesiredState
	if desiredSatisfied(current, plan.DesiredState) {
		return closeAttempt(layout, *attempt, plan, base, current, true, true)
	}
	if stateBeforeActiveAttempt(current) != attempt.PriorStateDigest {
		err := fmt.Errorf("active attempt has divergent or partial observed state")
		attempt.Phase, attempt.UpdatedAt = "recovery_required", canonicalNow()
		setAttemptDigest(attempt)
		_ = writeAttempt(layout.ActiveAttempt, *attempt)
		result := failedResult(base, "recovery_required", err, false)
		result.RecoveryRequired, result.AttemptDigest, result.AuditState = true, &attempt.AttemptDigest, attempt.AuditState
		result.ReconciliationRequired = attempt.AuditState == "audit_deferred"
		finalizeResult(&result)
		return result, err
	}
	return resumePrepared(layout, *attempt, plan, base, current, true, true)
}

func stateBeforeActiveAttempt(observation Observation) string {
	observation.RecoveryRequired = false
	observation.ActiveAttemptDigest = nil
	observation.StableStateDigest = digestWithout(observation, "observed_at", "stable_state_digest", "observation_digest")
	return observation.StableStateDigest
}

// BindAuditReceipt is the typed hook used by the future closed SSIAG producer
// reconciliation path. It never appends to STAV and never marks reconciliation
// complete without an exact receipt digest supplied by that closed producer.
func BindAuditReceipt(scope stavpaths.Scope, topsID, attemptDigest, receiptDigest string) (Attempt, error) {
	if !digestPattern.MatchString(receiptDigest) || !digestPattern.MatchString(attemptDigest) {
		return Attempt{}, fmt.Errorf("invalid audit receipt or attempt digest")
	}
	layout, err := stavpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return Attempt{}, err
	}
	if err := ensureTransactionDirectory(layout); err != nil {
		return Attempt{}, err
	}
	lease, err := acquireLifecycleLease(layout.LifecycleLock)
	if err != nil {
		return Attempt{}, err
	}
	defer lease.Close()
	attempt, _, err := readAttempt(layout.LastAttempt)
	if err != nil || attempt == nil || attempt.AttemptDigest != attemptDigest || attempt.AuditState != "audit_deferred" {
		return Attempt{}, fmt.Errorf("exact audit-deferred attempt is unavailable for reconciliation")
	}
	attempt.PredecessorDigest = &attempt.AttemptDigest
	attempt.AuditReceiptDigest, attempt.AuditState, attempt.UpdatedAt, attempt.Phase = &receiptDigest, "reconciled", canonicalNow(), "closed"
	setAttemptDigest(attempt)
	if err := writeAttempt(layout.LastAttempt, *attempt); err != nil {
		return Attempt{}, err
	}
	return *attempt, nil
}

// PreflightNativePurge enforces the stronger module-native purge boundary.
// Machine/qxctl v1 deliberately has no purge desired state.
func PreflightNativePurge(scope stavpaths.Scope, topsID string) error {
	layout, err := stavpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return err
	}
	if err := ensureTransactionDirectory(layout); err != nil {
		return err
	}
	lease, err := acquireLifecycleLease(layout.LifecycleLock)
	if err != nil {
		return err
	}
	defer lease.Close()
	return preflightNativePurgeLocked(scope, topsID, layout)
}

func NativePurge(scope stavpaths.Scope, topsID string) (lifecycle.EnrollmentRecord, error) {
	layout, err := stavpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return lifecycle.EnrollmentRecord{}, err
	}
	if err := ensureTransactionDirectory(layout); err != nil {
		return lifecycle.EnrollmentRecord{}, err
	}
	lease, err := acquireLifecycleLease(layout.LifecycleLock)
	if err != nil {
		return lifecycle.EnrollmentRecord{}, err
	}
	defer lease.Close()
	if err := preflightNativePurgeLocked(scope, topsID, layout); err != nil {
		return lifecycle.EnrollmentRecord{}, err
	}
	return lifecycle.Unenroll(scope, topsID, true)
}

func preflightNativePurgeLocked(scope stavpaths.Scope, topsID string, layout stavpaths.InstanceLayout) error {
	if attempt, _, attemptErr := readAttempt(layout.ActiveAttempt); attemptErr != nil {
		return fmt.Errorf("native purge requires recovery of unsafe attempt evidence: %w", attemptErr)
	} else if attempt != nil {
		return fmt.Errorf("native purge is blocked by unresolved lifecycle attempt %s", attempt.AttemptDigest)
	}
	if attempt, _, attemptErr := readAttempt(layout.LastAttempt); attemptErr != nil {
		return fmt.Errorf("native purge requires valid prior attempt evidence: %w", attemptErr)
	} else if attempt != nil && attempt.AuditState == "audit_deferred" {
		return fmt.Errorf("native purge is blocked until deferred lifecycle attempt %s is reconciled", attempt.AttemptDigest)
	}
	observation, err := Observe(scope, topsID, "supervisor")
	if err != nil {
		return err
	}
	if observation.Supervisor.ProcessState == "running" || (observation.Supervisor.DescriptorState != "absent" && observation.Supervisor.DescriptorState != "unknown") {
		return fmt.Errorf("native purge requires the STAV supervisor absent and process stopped")
	}
	return nil
}

func ensureTransactionDirectory(layout stavpaths.InstanceLayout) error {
	if layout.Scope == stavpaths.ScopeSystem && os.Geteuid() != 0 {
		return fmt.Errorf("system foundational lifecycle requires administrator privileges")
	}
	for _, path := range []string{layout.LifecycleDir, layout.AttemptDir} {
		if err := ensurePrivateDirectory(path); err != nil {
			return err
		}
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("unsafe lifecycle directory")
	}
	parent := filepath.Dir(path)
	if parent != path {
		if _, err := os.Lstat(parent); os.IsNotExist(err) {
			if err := ensurePrivateDirectory(parent); err != nil {
				return err
			}
		}
	}
	info, err := os.Lstat(path)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe lifecycle directory")
		}
		return os.Chmod(path, 0o700)
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(path, 0o700)
}

func readAttempt(path string) (*Attempt, []byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, data, err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return nil, data, fmt.Errorf("protected lifecycle attempt is unsafe")
	}
	var attempt Attempt
	if err := json.Unmarshal(data, &attempt); err != nil {
		return nil, data, err
	}
	if attempt.Protocol != AttemptProtocol || attempt.FormatVersion != 1 || attempt.AttemptDigest != digestWithout(attempt, "attempt_digest") {
		return nil, data, fmt.Errorf("protected lifecycle attempt digest is invalid")
	}
	return &attempt, data, nil
}

func readPlan(path string) (Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return Plan{}, fmt.Errorf("protected lifecycle plan is unsafe")
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return Plan{}, err
	}
	if plan.PlanDigest != digestWithout(plan, "plan_digest") {
		return Plan{}, fmt.Errorf("protected lifecycle plan digest is invalid")
	}
	return plan, nil
}

func setAttemptDigest(attempt *Attempt) {
	attempt.AttemptDigest = digestWithout(*attempt, "attempt_digest")
}
func writeAttempt(path string, attempt Attempt) error { return writeProtectedJSON(path, attempt) }

func writeProtectedJSON(path string, value any) error {
	if err := ensurePrivateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("refusing unsafe lifecycle evidence")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".stav-foundation-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(append(data, '\n')); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

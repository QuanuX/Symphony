package foundation

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/foundationaudit"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

var submitFoundationAudit foundationaudit.Submit = foundationaudit.SubmitToOwner

func auditRequest(p Plan) foundationaudit.Request {
	return foundationaudit.Request{Schema: foundationaudit.Schema, Component: p.Component, Surface: p.Surface, Scope: p.Scope, TOPSID: p.TOPSID, OperationID: p.OperationID, RequestID: p.RequestID, CorrelationID: p.CorrelationID, PlanDigest: p.PlanDigest, PriorStateDigest: p.ExpectedStateDigest, DesiredState: p.DesiredState, AuditMode: p.AuditMode, Phase: "intent"}
}

func prepareAudit(layout stavpaths.InstanceLayout, p Plan, attempt Attempt) error {
	j, err := foundationaudit.Open(filepath.Join(layout.LifecycleDir, "audit"), auditRequest(p))
	if err != nil {
		return err
	}
	if err = j.BindAttempt(attempt.AttemptDigest); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return j.CommitIntent(ctx, submitFoundationAudit)
}

func observeAudit(layout stavpaths.InstanceLayout, p Plan, attempt *Attempt, observation Observation, recovered bool) error {
	j, err := foundationaudit.Open(filepath.Join(layout.LifecycleDir, "audit"), auditRequest(p))
	if err != nil {
		return err
	}
	if err = j.Observe(stateBeforeActiveAttempt(observation), observation.ObservedAt); err != nil {
		return err
	}
	attempt.AuditState = "audit_deferred"
	attempt.AuditReceiptDigest = nil
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = j.Reconcile(ctx, submitFoundationAudit); err == nil {
		attempt.AuditState = "committed"
		if recovered || p.AuditMode == "audit_deferred" {
			attempt.AuditState = "reconciled"
		}
		value := foundationaudit.Hash(j.Completion.Receipt)
		attempt.AuditReceiptDigest = &value
	}
	return nil
}

func recoverAudit(layout stavpaths.InstanceLayout, command Command, base Result) (Result, bool, error) {
	if command.OperationID == nil {
		return base, false, nil
	}
	j, err := foundationaudit.Load(filepath.Join(layout.LifecycleDir, "audit"), *command.OperationID)
	if err != nil || j.Outcome == nil {
		return base, false, nil
	}
	if j.Base.Component != "stav" || j.Base.Scope != command.Scope || j.Base.TOPSID != command.TOPSID || j.Base.Surface != command.Surface {
		return base, true, fmt.Errorf("stored audit target mismatch")
	}
	last, _, readErr := readAttempt(layout.LastAttempt)
	if readErr != nil {
		return base, true, readErr
	}
	expected := j.AttemptDigest
	if last != nil && last.OperationID == j.Base.OperationID {
		if last.PlanDigest != j.Base.PlanDigest || last.Scope != j.Base.Scope || last.TOPSID != j.Base.TOPSID || last.Surface != j.Base.Surface {
			return base, true, fmt.Errorf("native attempt and audit journal disagree")
		}
		expected = last.AttemptDigest
	}
	if !command.Discover && (command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != expected) {
		err := fmt.Errorf("exact observed attempt is required")
		return failedResult(base, "state_conflict", err, true), true, err
	}
	base.DesiredState = &j.Base.DesiredState
	base.AttemptDigest = &expected
	base.StartedAt = nil
	if last != nil && last.OperationID == j.Base.OperationID {
		base.StartedAt = &last.StartedAt
	}
	base.AuditState = "audit_deferred"
	base.ReconciliationRequired = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = j.Reconcile(ctx, submitFoundationAudit); err != nil {
		err = fmt.Errorf("observed lifecycle outcome remains pending exact STAV receipt")
		result := failedResult(base, "audit_unavailable", err, true)
		result.AuditState = "audit_deferred"
		result.ReconciliationRequired = true
		finalizeResult(&result)
		return result, true, err
	}
	base.AuditState = "reconciled"
	base.ReconciliationRequired = false
	value := foundationaudit.Hash(j.Completion.Receipt)
	base.AuditReceiptDigest = &value
	base.Disposition = "recovered"
	base.Recovered = true
	base.Replayed = false
	base.RecoveryRequired = false
	last, _, err = readAttempt(layout.LastAttempt)
	if err != nil {
		return base, true, err
	}
	if last != nil && last.OperationID == j.Base.OperationID {
		old := last.AttemptDigest
		last.PredecessorDigest = &old
		last.AuditState = "reconciled"
		last.AuditReceiptDigest = &value
		last.UpdatedAt = canonicalNow()
		setAttemptDigest(last)
		if err = writeAttempt(layout.LastAttempt, *last); err != nil {
			return base, true, err
		}
		base.AttemptDigest = &last.AttemptDigest
		if err = j.BindAttempt(last.AttemptDigest); err != nil {
			return base, true, err
		}
	}
	finalizeResult(&base)
	return base, true, nil
}

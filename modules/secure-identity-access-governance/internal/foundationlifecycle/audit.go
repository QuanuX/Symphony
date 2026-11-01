package foundationlifecycle

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/client"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationaudit"
)

func submitFoundationAudit(ctx context.Context, r foundationaudit.Request) (foundationaudit.Response, error) {
	c, err := client.NewForTOPS(parseScope(r.Scope), r.TOPSID, 5*time.Second)
	if err != nil {
		return foundationaudit.Response{}, err
	}
	return c.FoundationAudit(ctx, r)
}

func auditRequest(p Plan) foundationaudit.Request {
	return foundationaudit.Request{Schema: foundationaudit.Schema, Component: p.Component, Surface: p.Surface, Scope: p.Scope, TOPSID: p.TOPSID, OperationID: p.OperationID, RequestID: p.RequestID, CorrelationID: p.CorrelationID, PlanDigest: p.PlanDigest, PriorStateDigest: p.ExpectedStateDigest, DesiredState: p.DesiredState, AuditMode: p.AuditMode, Phase: "intent"}
}

func (engine *Engine) prepareAudit(store *attemptStore, p Plan, attempt *Attempt) error {
	j, err := foundationaudit.Open(filepath.Join(store.directory, "audit"), auditRequest(p))
	if err != nil {
		return err
	}
	if err = j.BindAttempt(attempt.AttemptDigest); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return j.CommitIntent(ctx, engine.audit)
}

func (engine *Engine) observeAudit(store *attemptStore, p Plan, attempt *Attempt, observed Observation, recovered bool) error {
	j, err := foundationaudit.Open(filepath.Join(store.directory, "audit"), auditRequest(p))
	if err != nil {
		return err
	}
	if err = j.Observe(observed.StableStateDigest, observed.ObservedAt); err != nil {
		return err
	}
	attempt.AuditState = "audit_deferred"
	attempt.AuditReceiptDigest = nil
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = j.Reconcile(ctx, engine.audit); err == nil {
		attempt.AuditState = "committed"
		if recovered || p.AuditMode == "audit_deferred" {
			attempt.AuditState = "reconciled"
		}
		attempt.AuditReceiptDigest = stringPointer(foundationaudit.Hash(j.Completion.Receipt))
	}
	return nil
}

// Historical reconciliation never re-executes the old plan or substitutes the
// present state for a previously persisted observation (notably stop/restart).
func (engine *Engine) recoverAudit(command Command, observation Observation, store *attemptStore) (Result, bool, error) {
	if command.OperationID == nil {
		return Result{}, false, nil
	}
	j, err := foundationaudit.Load(filepath.Join(store.directory, "audit"), *command.OperationID)
	if err != nil || j.Outcome == nil {
		return Result{}, false, nil
	}
	if j.Base.Component != "ssiag" || j.Base.Surface != command.Surface || j.Base.Scope != command.Scope || j.Base.TOPSID != command.TOPSID {
		return Result{}, true, fmt.Errorf("stored audit target mismatch")
	}
	current, present, readErr := store.read()
	if readErr != nil {
		return Result{}, true, readErr
	}
	expected := j.AttemptDigest
	if present && current.OperationID == j.Base.OperationID {
		if current.PlanDigest != j.Base.PlanDigest || current.Scope != j.Base.Scope || current.TOPSID != j.Base.TOPSID || current.Surface != j.Base.Surface {
			return Result{}, true, fmt.Errorf("native attempt and audit journal disagree")
		}
		expected = current.AttemptDigest
	}
	if !command.Discover && (command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != expected) {
		r, e := engine.blocked(command, observation, nil, "attempt.compare-and-swap", "exact observed attempt is required")
		return r, true, e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := Result{Operation: "recover", Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: command.OperationID, DesiredState: &j.Base.DesiredState, Observation: observation, AttemptDigest: &expected, AuditState: "audit_deferred", ReconciliationRequired: true, Disposition: "blocked"}
	if present && current.OperationID == j.Base.OperationID {
		result.StartedAt = &current.StartedAt
	}
	if err = j.Reconcile(ctx, engine.audit); err != nil {
		result.Error = &ErrorResult{Code: "audit.unavailable", Message: "observed lifecycle outcome remains pending exact STAV receipt"}
	} else {
		result.AuditState = "reconciled"
		result.AuditReceiptDigest = stringPointer(foundationaudit.Hash(j.Completion.Receipt))
		result.ReconciliationRequired = false
		result.Disposition = "recovered"
		result.Recovered = true
		// Keep latest native status aligned when this is its operation. Older
		// operations remain independently recoverable in the durable audit journal.
		current, present, e := store.read()
		if e != nil {
			return Result{}, true, e
		}
		if present && current.OperationID == j.Base.OperationID && current.Phase == "closed" {
			current.PredecessorDigest = stringPointer(current.AttemptDigest)
			current.AuditState = "reconciled"
			current.AuditReceiptDigest = result.AuditReceiptDigest
			current.UpdatedAt = engine.stscNow()
			if e = store.write(&current); e != nil {
				return Result{}, true, e
			}
			result.AttemptDigest = stringPointer(current.AttemptDigest)
			if e = j.BindAttempt(current.AttemptDigest); e != nil {
				return Result{}, true, e
			}
		}
	}
	r, e := engine.finish(result)
	return r, true, e
}

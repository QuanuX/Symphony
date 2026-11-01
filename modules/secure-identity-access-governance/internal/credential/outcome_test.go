//go:build darwin || linux

package credential

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

type outcomeTransport struct {
	mu       sync.Mutex
	requests []stav.LocalRequest
	fail     bool
	corrupt  string
	base     dispatchAudit
}

func (a *outcomeTransport) Do(ctx context.Context, request stav.LocalRequest) (stav.LocalResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requests = append(a.requests, request)
	response, err := a.base.Do(ctx, request)
	if request.Candidate.Operation.EventClass == "symphony.ssiag.provider.operation" {
		if a.fail {
			return stav.LocalResponse{}, errors.New("fixture audit unavailable after execution")
		}
		switch a.corrupt {
		case "wrong-candidate":
			response.Receipt.CandidateDigest = fixtureDigest
		case "wrong-request":
			response.RequestID = "b90e1205-1b3b-4e47-9b91-1cd624cd87cd"
			response.Receipt.RequestID = response.RequestID
		case "wrong-tops":
			response.TOPSID = "123e4567-e89b-42d3-a456-426614174000"
			response.Receipt.TOPSID = response.TOPSID
		}
	}
	return response, err
}
func (a *outcomeTransport) outcomes() []stav.LocalRequest {
	var results []stav.LocalRequest
	for _, r := range a.requests {
		if r.Candidate.Operation.EventClass == "symphony.ssiag.provider.operation" {
			results = append(results, r)
		}
	}
	return results
}
func installOutcomeTransport(t *testing.T, d *Dispatcher) *outcomeTransport {
	t.Helper()
	a := &outcomeTransport{}
	var err error
	d.audit, err = stavproducer.New(dispatchTOPS, a)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func readOutcomeRecord(t *testing.T, d *Dispatcher, in DispatchIntent) journalRecord {
	t.Helper()
	data, err := os.ReadFile(journalPath(d.journal, in.Authorization.RequestID))
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeJournal(data)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestOutcomeAuditReconcilesWithoutRepeatingDelivery(t *testing.T) {
	for _, fault := range []string{"unavailable", "wrong-candidate", "wrong-request", "wrong-tops"} {
		t.Run(fault, func(t *testing.T) {
			d, in, p, _, _ := dispatchFixture(t)
			a := installOutcomeTransport(t, d)
			a.fail, a.corrupt = fault == "unavailable", fault
			ctx := dispatchPeer(t, true)
			attempt, _ := d.Prepare(in)
			if got := attempt.Run(ctx); got != DispatchRecoveryRequired {
				t.Fatalf("audit failure became %s", got)
			}
			r := readOutcomeRecord(t, d, in)
			if r.Phase != "closed" || r.Outcome != DispatchDelivered || r.Audit.Receipt != nil || p.executions.Load() != 1 || p.releases.Load() != 1 {
				t.Fatal("lost observed outcome or cleanup")
			}
			if retry, _ := d.Prepare(in); retry.Run(ctx) != DispatchRecoveryRequired {
				t.Fatal("pending audit authorized replay")
			}
			originalID := r.Audit.RequestID
			a.fail, a.corrupt = false, ""
			result, err := d.Recover(ctx, in, recoveryAuthorization(in))
			if err != nil || result.Outcome != DispatchDelivered || result.AuditState != "committed" {
				t.Fatal(result, err)
			}
			if p.executions.Load() != 1 || p.pins.Load() != 1 {
				t.Fatal("recovery touched provider")
			}
			outcomes := a.outcomes()
			if len(outcomes) != 2 || outcomes[0].RequestID != originalID || outcomes[1].RequestID != originalID || originalID == in.Authorization.RequestID {
				t.Fatal("outcome audit identity was not retained independently")
			}
			first, _ := stav.CandidateDigest(*outcomes[0].Candidate)
			second, _ := stav.CandidateDigest(*outcomes[1].Candidate)
			if first != second {
				t.Fatal("reconciliation changed candidate")
			}
			if _, err := d.Recover(ctx, in, recoveryAuthorization(in)); err != nil || len(a.outcomes()) != 2 {
				t.Fatal("completed audit replayed", err)
			}
		})
	}
}

func TestOutcomeAuditReceiptPersistenceFailureKeepsOriginalIdentity(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-write", true: "after-write"}[after], func(t *testing.T) {
			d, in, p, _, _ := dispatchFixture(t)
			a := installOutcomeTransport(t, d)
			d.journal.open = func(j *Journal, id string, create bool) (journalFile, error) {
				f, err := openJournalFile(j, id, create)
				if err != nil {
					return nil, err
				}
				return &failJournalFile{journalFile: f, at: 4, after: after}, nil
			}
			ctx := dispatchPeer(t, true)
			attempt, _ := d.Prepare(in)
			if got := attempt.Run(ctx); got != DispatchRecoveryRequired {
				t.Fatalf("receipt persistence failed: %s", got)
			}
			first := a.outcomes()[0]
			d.journal.open = openJournalFile
			result, err := d.Recover(ctx, in, recoveryAuthorization(in))
			if err != nil || result.AuditState != "committed" || p.executions.Load() != 1 {
				t.Fatal(result, err)
			}
			for _, request := range a.outcomes() {
				if request.RequestID != first.RequestID {
					t.Fatal("audit identity changed after acknowledged receipt")
				}
			}
		})
	}
}

func TestOutcomeLegacyMigrationPreservesExecutionEvidence(t *testing.T) {
	d, in, p, _, _ := dispatchFixture(t)
	ctx := dispatchPeer(t, true)
	attempt, _ := d.Prepare(in)
	if attempt.Run(ctx) != DispatchDelivered {
		t.Fatal("initial delivery")
	}
	r := readOutcomeRecord(t, d, in)
	r.Format, r.Audit = legacyJournalFormat, nil
	legacy, err := encodeJournal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journalPath(d.journal, in.Authorization.RequestID), legacy, 0600); err != nil {
		t.Fatal(err)
	}
	a := installOutcomeTransport(t, d)
	if retry, _ := d.Prepare(in); retry.Run(ctx) != DispatchRecoveryRequired {
		t.Fatal("legacy evidence reissued")
	}
	result, err := d.Recover(ctx, in, recoveryAuthorization(in))
	if err != nil || result.Outcome != DispatchDelivered || result.AuditState != "committed" || p.executions.Load() != 1 {
		t.Fatal(result, err)
	}
	current := readOutcomeRecord(t, d, in)
	if current.Format != journalFormat || current.Identity != r.Identity || *current.Binding != *r.Binding || current.Outcome != r.Outcome || len(a.outcomes()) != 1 {
		t.Fatal("migration altered original evidence")
	}
}

func TestOutcomeUncertaintyNeverBecomesSuccess(t *testing.T) {
	d, in, p, _, _ := dispatchFixture(t)
	a := installOutcomeTransport(t, d)
	p.execute = func(context.Context, UseBinding) DispatchStatus { return DispatchIndeterminate }
	ctx := dispatchPeer(t, true)
	attempt, _ := d.Prepare(in)
	if attempt.Run(ctx) != DispatchIndeterminate {
		t.Fatal("lost uncertain delivery")
	}
	if got := a.outcomes(); len(got) != 1 || got[0].Candidate.Result.Outcome != "unavailable" {
		t.Fatal("uncertainty became confirmed completion")
	}
	result, err := d.Recover(ctx, in, recoveryAuthorization(in))
	if err != nil || result.Outcome != DispatchIndeterminate || result.AuditState != "committed" || p.executions.Load() != 1 {
		t.Fatal(result, err)
	}
}

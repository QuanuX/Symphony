package credential

import (
	"context"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

func outcomeRecord(r journalRecord) stavproducer.Record {
	b := r.Binding
	outcome := "unavailable"
	if r.Outcome == DispatchDelivered {
		outcome = "succeeded"
	} else if r.Outcome == DispatchRefused {
		outcome = "failed"
	}
	// The closed v1 vocabulary has no indeterminate outcome. Unavailable means
	// confirmed completion evidence is unavailable, never that delivery failed.
	// The request journal preserves the exact indeterminate execution status.
	return stavproducer.Record{
		Kind: stavproducer.ProviderOperation, RequestID: r.Audit.RequestID, CorrelationID: b.CorrelationID,
		Actor:          stav.SafeReference{ID: b.SubjectID, Kind: b.SubjectKind},
		Authentication: stav.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"},
		Target:         stav.SafeReference{ID: b.Resource, Kind: "symphony.ssiag.resource"}, Outcome: outcome,
		Configuration: stav.Configuration{PreviousDigest: b.ConfigDigest, NewDigest: b.ConfigDigest, State: "digests"},
		TROG:          stav.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}, Classification: "administrative_metadata",
	}
}

func validOutcomeAudit(r journalRecord) bool {
	if r.Format == legacyJournalFormat {
		return r.Audit == nil
	}
	if r.Binding == nil {
		return r.Audit == nil
	}
	if r.Audit == nil || stav.ValidateRequestUUID(r.Audit.RequestID) != nil ||
		r.Audit.RequestID == r.Identity.RequestID || r.Audit.RequestID == r.Binding.CorrelationID {
		return false
	}
	if r.Audit.Receipt == nil {
		return true
	}
	if r.Phase != "closed" {
		return false
	}
	receipt := r.Audit.Receipt
	want, err := stavproducer.CandidateDigest(r.Identity.TOPSID, outcomeRecord(r))
	return err == nil && receipt.Validate() == nil && receipt.Disposition == "committed" &&
		receipt.TOPSID == r.Identity.TOPSID && receipt.RequestID == r.Audit.RequestID && receipt.CandidateDigest == want
}

// commitOutcome runs while the original request's kernel lock remains held.
// It can replay only the stable safe audit candidate, never provider execution.
func (d *Dispatcher) commitOutcome(ctx context.Context, a *journalAttempt) error {
	if d == nil || d.audit == nil || a == nil || ctx == nil || ctx.Err() != nil ||
		a.record.Phase != "closed" || a.record.Binding == nil || !validJournalRecord(a.record) {
		return errJournal
	}
	r := a.record
	if r.Audit == nil {
		// Legacy records remain duplicate-suppressing evidence. Migrate only under
		// authenticated recovery, preserving identity, binding and exact outcome.
		id, err := stav.GenerateUUIDv4()
		if err != nil {
			return errJournal
		}
		r.Format = journalFormat
		r.Audit = &outcomeAudit{RequestID: id}
		data, err := encodeJournal(r)
		if err != nil || a.file.write(data) != nil {
			return errJournal
		}
		a.record = r
	}
	if r.Audit.Receipt != nil {
		return nil
	}
	if ctx.Err() != nil {
		return errJournal
	}
	receipt, err := d.audit.Submit(ctx, outcomeRecord(r))
	if err != nil {
		return errJournal
	}
	r.Audit = &outcomeAudit{RequestID: r.Audit.RequestID, Receipt: &receipt}
	data, err := encodeJournal(r)
	if err != nil || a.file.write(data) != nil {
		return errJournal
	}
	a.record = r
	return nil
}

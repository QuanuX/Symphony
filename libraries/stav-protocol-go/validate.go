package stavprotocol

import (
	"fmt"
	"time"
)

var allowedOutcomes = map[string]struct{}{
	"allowed": {}, "denied": {}, "failed": {}, "succeeded": {}, "unavailable": {},
}

func validateSafeInteger(n uint64) error {
	if n > MaxSafeInteger {
		return fmt.Errorf("stav: integer exceeds I-JSON safe range")
	}
	return nil
}

func (r SafeReference) validate() error {
	if err := validateOpaqueReference(r.ID); err != nil {
		return err
	}
	return validateRegisteredIdentifier(r.Kind)
}

func (t TROG) validate() error {
	switch t.State {
	case "identified":
		if t.ReasonCode != "" {
			return fmt.Errorf("stav: identified TROG cannot contain reason_code")
		}
		return validateOpaqueReference(t.ID)
	case "not_applicable":
		if t.ID != "" {
			return fmt.Errorf("stav: not-applicable TROG cannot contain id")
		}
		return validateRegisteredIdentifier(t.ReasonCode)
	default:
		return fmt.Errorf("stav: invalid TROG state")
	}
}

func (a Authentication) validate() error {
	switch a.State {
	case "identified":
		if a.ReasonCode != "" {
			return fmt.Errorf("stav: identified authentication cannot contain reason_code")
		}
		return validateRegisteredIdentifier(a.MethodID)
	case "not_applicable":
		if a.MethodID != "" {
			return fmt.Errorf("stav: not-applicable authentication cannot contain method_id")
		}
		return validateRegisteredIdentifier(a.ReasonCode)
	default:
		return fmt.Errorf("stav: invalid authentication state")
	}
}

func (c Configuration) validate() error {
	switch c.State {
	case "digests":
		if c.ReasonCode != "" {
			return fmt.Errorf("stav: digest configuration cannot contain reason_code")
		}
		if err := validateDigest(c.PreviousDigest); err != nil {
			return err
		}
		return validateDigest(c.NewDigest)
	case "not_applicable":
		if c.PreviousDigest != "" || c.NewDigest != "" {
			return fmt.Errorf("stav: not-applicable configuration cannot contain digests")
		}
		return validateRegisteredIdentifier(c.ReasonCode)
	default:
		return fmt.Errorf("stav: invalid configuration state")
	}
}

func (t Topology) validate() error {
	if err := ValidateTOPSID(t.TOPSID); err != nil {
		return err
	}
	return t.TROG.validate()
}

func (c Correlation) validate() error {
	if err := ValidateRequestUUID(c.CorrelationID); err != nil {
		return err
	}
	return ValidateRequestUUID(c.RequestID)
}

func (o Operation) validate() error {
	if err := validateRegisteredIdentifier(o.EventClass); err != nil {
		return err
	}
	if err := validateRegisteredIdentifier(o.OperationID); err != nil {
		return err
	}
	return o.Target.validate()
}

func (r Result) validate() error {
	if err := validateRegisteredIdentifier(r.IntentID); err != nil {
		return err
	}
	if _, ok := allowedOutcomes[r.Outcome]; !ok {
		return fmt.Errorf("stav: invalid outcome")
	}
	return validateRegisteredIdentifier(r.ReasonCode)
}

func (r Redaction) validate() error {
	if r.Classification != "administrative_metadata" && r.Classification != "restricted_metadata" {
		return fmt.Errorf("stav: invalid redaction classification")
	}
	return nil
}

// Validate applies the canonical candidate schema and closed-value rules.
func (c Candidate) Validate() error {
	if c.Schema != SchemaCandidate {
		return fmt.Errorf("stav: invalid candidate schema")
	}
	if err := c.Actor.Authentication.validate(); err != nil {
		return err
	}
	if err := c.Actor.Principal.validate(); err != nil {
		return err
	}
	if err := c.Configuration.validate(); err != nil {
		return err
	}
	if err := c.Correlation.validate(); err != nil {
		return err
	}
	if err := c.Operation.validate(); err != nil {
		return err
	}
	if err := c.Redaction.validate(); err != nil {
		return err
	}
	if err := c.Result.validate(); err != nil {
		return err
	}
	return c.Topology.validate()
}

// Validate applies the canonical ten-group event schema. It does not assert
// that trusted fields were actually assigned by an append authority.
func (e Event) Validate() error {
	if e.Identity.Schema != SchemaEvent {
		return fmt.Errorf("stav: invalid event schema")
	}
	if err := ValidateEventUUID(e.Identity.EventID); err != nil {
		return err
	}
	if err := e.Actor.Authentication.validate(); err != nil {
		return err
	}
	if err := e.Actor.Principal.validate(); err != nil {
		return err
	}
	if err := e.Actor.Producer.validate(); err != nil {
		return err
	}
	if err := e.Configuration.validate(); err != nil {
		return err
	}
	if err := e.Correlation.validate(); err != nil {
		return err
	}
	if err := validateDigest(e.Integrity.PrecedingEventDigest); err != nil {
		return err
	}
	if err := e.Operation.validate(); err != nil {
		return err
	}
	if e.Ordering.Sequence == 0 {
		return fmt.Errorf("stav: event sequence must start at one")
	}
	if err := validateSafeInteger(e.Ordering.Sequence); err != nil {
		return err
	}
	if err := validateTimestamp(e.Ordering.Timestamp); err != nil {
		return err
	}
	if err := e.Redaction.validate(); err != nil {
		return err
	}
	if err := e.Result.validate(); err != nil {
		return err
	}
	return e.Topology.validate()
}

// Validate applies receipt structure. A valid committed representation does
// not authorize operational code to emit it before the durability gate.
func (r Receipt) Validate() error {
	if r.Schema != SchemaReceipt {
		return fmt.Errorf("stav: invalid receipt schema")
	}
	if err := ValidateTOPSID(r.TOPSID); err != nil {
		return err
	}
	if err := ValidateRequestUUID(r.RequestID); err != nil {
		return err
	}
	if err := validateDigest(r.CandidateDigest); err != nil {
		return err
	}
	if err := validateRegisteredIdentifier(r.ReasonCode); err != nil {
		return err
	}
	switch r.Disposition {
	case "committed":
		if r.Commit.State != "committed" || r.Commit.ReasonCode != "" {
			return fmt.Errorf("stav: invalid committed receipt state")
		}
		if err := ValidateEventUUID(r.Commit.EventID); err != nil {
			return err
		}
		if r.Commit.Sequence == 0 {
			return fmt.Errorf("stav: committed sequence must start at one")
		}
		if err := validateSafeInteger(r.Commit.Sequence); err != nil {
			return err
		}
		if err := validateTimestamp(r.Commit.Timestamp); err != nil {
			return err
		}
		return validateDigest(r.Commit.EventDigest)
	case "rejected":
		if r.Commit.State != "not_committed" || r.Commit.EventID != "" || r.Commit.EventDigest != "" || r.Commit.Sequence != 0 || r.Commit.Timestamp != "" {
			return fmt.Errorf("stav: invalid rejected receipt state")
		}
		return validateRegisteredIdentifier(r.Commit.ReasonCode)
	default:
		return fmt.Errorf("stav: invalid receipt disposition")
	}
}

// Validate applies bounded, forward-only query rules.
func (q Query) Validate() error {
	if q.Schema != SchemaQuery {
		return fmt.Errorf("stav: invalid query schema")
	}
	if err := ValidateTOPSID(q.TOPSID); err != nil {
		return err
	}
	if err := validateSafeInteger(q.AfterSequence); err != nil {
		return err
	}
	if q.ThroughSequence != nil {
		if err := validateSafeInteger(*q.ThroughSequence); err != nil {
			return err
		}
		if *q.ThroughSequence <= q.AfterSequence {
			return fmt.Errorf("stav: through_sequence must follow after_sequence")
		}
	}
	if q.Limit == 0 || q.Limit > MaxQueryLimit {
		return fmt.Errorf("stav: query limit out of range")
	}
	if q.EventClasses == nil || len(q.EventClasses) > 16 {
		return fmt.Errorf("stav: invalid event_classes")
	}
	if q.Outcomes == nil || len(q.Outcomes) > 5 {
		return fmt.Errorf("stav: invalid outcomes")
	}
	if err := validateUniqueRegistered(q.EventClasses); err != nil {
		return err
	}
	seenOutcomes := make(map[string]struct{}, len(q.Outcomes))
	for _, outcome := range q.Outcomes {
		if _, ok := allowedOutcomes[outcome]; !ok {
			return fmt.Errorf("stav: invalid outcome filter")
		}
		if _, exists := seenOutcomes[outcome]; exists {
			return fmt.Errorf("stav: duplicate outcome filter")
		}
		seenOutcomes[outcome] = struct{}{}
	}
	if q.CorrelationID != "" {
		if err := ValidateRequestUUID(q.CorrelationID); err != nil {
			return err
		}
	}
	if q.RequestID != "" {
		if err := ValidateRequestUUID(q.RequestID); err != nil {
			return err
		}
	}
	if q.FromTime != "" {
		if err := validateTimestamp(q.FromTime); err != nil {
			return err
		}
	}
	if q.ThroughTime != "" {
		if err := validateTimestamp(q.ThroughTime); err != nil {
			return err
		}
	}
	if q.FromTime != "" && q.ThroughTime != "" {
		from, _ := time.Parse(timestampLayout, q.FromTime)
		through, _ := time.Parse(timestampLayout, q.ThroughTime)
		if through.Before(from) {
			return fmt.Errorf("stav: through_time precedes from_time")
		}
	}
	return nil
}

func validateUniqueRegistered(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateRegisteredIdentifier(value); err != nil {
			return err
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("stav: duplicate registered identifier")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (p QueryProjection) validate() error {
	if err := p.Actor.validate(); err != nil {
		return err
	}
	if err := ValidateRequestUUID(p.CorrelationID); err != nil {
		return err
	}
	if err := validateRegisteredIdentifier(p.EventClass); err != nil {
		return err
	}
	if err := validateRegisteredIdentifier(p.OperationID); err != nil {
		return err
	}
	if _, ok := allowedOutcomes[p.Outcome]; !ok {
		return fmt.Errorf("stav: invalid projected outcome")
	}
	if err := validateRegisteredIdentifier(p.ReasonCode); err != nil {
		return err
	}
	if err := ValidateRequestUUID(p.RequestID); err != nil {
		return err
	}
	if err := p.Target.validate(); err != nil {
		return err
	}
	return validateTimestamp(p.Timestamp)
}

// Validate applies ascending, source-qualified projection rules.
func (p QueryPage) Validate() error {
	if p.Schema != SchemaQueryPage {
		return fmt.Errorf("stav: invalid query-page schema")
	}
	if err := ValidateTOPSID(p.TOPSID); err != nil {
		return err
	}
	if err := validateSafeInteger(p.AfterSequence); err != nil {
		return err
	}
	if p.Entries == nil || len(p.Entries) > MaxQueryLimit {
		return fmt.Errorf("stav: invalid query-page entries")
	}
	last := p.AfterSequence
	for _, entry := range p.Entries {
		if entry.Sequence <= last || entry.Sequence > MaxSafeInteger {
			return fmt.Errorf("stav: query-page sequence is not ascending")
		}
		last = entry.Sequence
		if err := ValidateEventUUID(entry.EventID); err != nil {
			return err
		}
		if err := validateDigest(entry.EventDigest); err != nil {
			return err
		}
		if entry.RedactionState != "allowlisted" && entry.RedactionState != "restricted" {
			return fmt.Errorf("stav: invalid redaction state")
		}
		if entry.VerificationState != "verified" && entry.VerificationState != "unverified" {
			return fmt.Errorf("stav: invalid verification state")
		}
		if err := entry.Projection.validate(); err != nil {
			return err
		}
	}
	switch p.Next.State {
	case "complete":
		if p.Next.AfterSequence != 0 {
			return fmt.Errorf("stav: complete page cannot contain next cursor")
		}
	case "available":
		if p.Next.AfterSequence != last || p.Next.AfterSequence == 0 {
			return fmt.Errorf("stav: invalid next cursor")
		}
	default:
		return fmt.Errorf("stav: invalid page continuation state")
	}
	return nil
}

// Validate applies bounded chain-verification result rules.
func (v Verification) Validate() error {
	if v.Schema != SchemaVerification {
		return fmt.Errorf("stav: invalid verification schema")
	}
	if err := ValidateTOPSID(v.TOPSID); err != nil {
		return err
	}
	if err := validateSafeInteger(v.AfterSequence); err != nil {
		return err
	}
	if err := validateSafeInteger(v.ThroughSequence); err != nil {
		return err
	}
	if err := validateSafeInteger(v.EventsChecked); err != nil {
		return err
	}
	if v.ThroughSequence < v.AfterSequence || v.EventsChecked > v.ThroughSequence-v.AfterSequence {
		return fmt.Errorf("stav: invalid verification range")
	}
	switch v.Result.State {
	case "verified":
		if v.Result.AtSequence != 0 || v.Result.ReasonCode != "" {
			return fmt.Errorf("stav: verified result contains failure fields")
		}
	case "failed":
		if v.Result.AtSequence <= v.AfterSequence || v.Result.AtSequence > v.ThroughSequence {
			return fmt.Errorf("stav: failure sequence outside verification range")
		}
		if err := validateRegisteredIdentifier(v.Result.ReasonCode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("stav: invalid verification state")
	}
	return nil
}

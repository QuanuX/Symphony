package stavprotocol

import (
	"fmt"
	"strings"
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

func (p PeerPermission) validate() error {
	if err := validateRegisteredIdentifier(p.EventClass); err != nil {
		return err
	}
	return validateRegisteredIdentifier(p.OperationID)
}

func validatePeer(uid, gid uint64) error {
	if uid > 4294967295 || gid > 4294967295 {
		return fmt.Errorf("stav: peer credential outside uint32 range")
	}
	return nil
}

func (c AppendAuthorityConfig) Validate() error {
	if c.Schema != SchemaAppendAuthorityConfig {
		return fmt.Errorf("stav: invalid append-authority config schema")
	}
	if err := ValidateTOPSID(c.TOPSID); err != nil {
		return err
	}
	if c.Mode != "user" && c.Mode != "system" {
		return fmt.Errorf("stav: invalid append-authority mode")
	}
	if c.Listen.Network != "unix" || !strings.HasPrefix(c.Listen.Address, "/") || len(c.Listen.Address) > 4096 {
		return fmt.Errorf("stav: invalid append-authority listener")
	}
	if c.Ledger.Durability != "fsync-before-receipt" || c.Ledger.Recovery != "preserve-incomplete-tail" || c.Ledger.Retention != "preserve_all" || c.Ledger.Rotation != "disabled" {
		return fmt.Errorf("stav: invalid append-authority ledger policy")
	}
	if !strings.HasPrefix(c.Ledger.Path, "/") || len(c.Ledger.Path) > 4096 || c.Ledger.MaxBytes < 1_048_576 || c.Ledger.MaxBytes > MaxSafeInteger {
		return fmt.Errorf("stav: invalid append-authority ledger")
	}
	if c.Authentication.Mechanism != "kernel-peer-credentials" || c.Authentication.Producers == nil || c.Authentication.Readers == nil || len(c.Authentication.Producers) > 128 || len(c.Authentication.Readers) > 128 {
		return fmt.Errorf("stav: invalid append-authority authentication")
	}
	if err := validatePeer(c.Authentication.Authority.UID, c.Authentication.Authority.GID); err != nil {
		return err
	}
	if err := c.Authentication.Authority.Subject.validate(); err != nil {
		return err
	}
	producerPeers := make(map[[2]uint64]struct{}, len(c.Authentication.Producers))
	for _, grant := range c.Authentication.Producers {
		if err := validatePeer(grant.UID, grant.GID); err != nil {
			return err
		}
		peer := [2]uint64{grant.UID, grant.GID}
		if _, exists := producerPeers[peer]; exists {
			return fmt.Errorf("stav: ambiguous producer peer grant")
		}
		producerPeers[peer] = struct{}{}
		if err := grant.Subject.validate(); err != nil {
			return err
		}
		if err := grant.Producer.validate(); err != nil {
			return err
		}
		if len(grant.Permissions) == 0 || len(grant.Permissions) > 128 {
			return fmt.Errorf("stav: invalid producer permissions")
		}
		seen := make(map[[2]string]struct{}, len(grant.Permissions))
		for _, permission := range grant.Permissions {
			if err := permission.validate(); err != nil {
				return err
			}
			key := [2]string{permission.EventClass, permission.OperationID}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("stav: duplicate producer permission")
			}
			seen[key] = struct{}{}
		}
	}
	readerPeers := make(map[[2]uint64]struct{}, len(c.Authentication.Readers))
	for _, grant := range c.Authentication.Readers {
		if err := validatePeer(grant.UID, grant.GID); err != nil {
			return err
		}
		peer := [2]uint64{grant.UID, grant.GID}
		if _, exists := readerPeers[peer]; exists {
			return fmt.Errorf("stav: ambiguous reader peer grant")
		}
		readerPeers[peer] = struct{}{}
		if err := grant.Subject.validate(); err != nil {
			return err
		}
		if len(grant.Classifications) == 0 || len(grant.Classifications) > 2 {
			return fmt.Errorf("stav: invalid reader classifications")
		}
		seen := make(map[string]struct{}, len(grant.Classifications))
		for _, classification := range grant.Classifications {
			if err := (Redaction{Classification: classification}).validate(); err != nil {
				return err
			}
			if _, exists := seen[classification]; exists {
				return fmt.Errorf("stav: duplicate reader classification")
			}
			seen[classification] = struct{}{}
		}
	}
	return nil
}

func (s AppendAuthorityStatus) Validate() error {
	if s.Schema != SchemaAppendAuthorityStatus {
		return fmt.Errorf("stav: invalid append-authority status schema")
	}
	if err := ValidateTOPSID(s.TOPSID); err != nil {
		return err
	}
	if s.Mode != "user" && s.Mode != "system" {
		return fmt.Errorf("stav: invalid append-authority status mode")
	}
	if s.Events != s.LastSequence || s.Events > MaxSafeInteger || s.LedgerBytes > MaxSafeInteger || s.MaxLedgerBytes < 1_048_576 || s.MaxLedgerBytes > MaxSafeInteger || s.LedgerBytes > s.MaxLedgerBytes {
		return fmt.Errorf("stav: inconsistent append-authority status")
	}
	if err := validateDigest(s.GenesisDigest); err != nil {
		return err
	}
	if s.LastSequence == 0 {
		if s.LastEventDigest != "" {
			return fmt.Errorf("stav: empty ledger cannot have last event digest")
		}
	} else if err := validateDigest(s.LastEventDigest); err != nil {
		return err
	}
	switch s.StorageState {
	case "clean":
		if s.RecoveredTail {
			return fmt.Errorf("stav: clean storage cannot report recovered tail")
		}
	case "recovered_incomplete_tail":
		if !s.RecoveredTail {
			return fmt.Errorf("stav: recovered storage must report recovered tail")
		}
	default:
		return fmt.Errorf("stav: invalid storage state")
	}
	return nil
}

func (v VerifyRequest) validate() error {
	if err := validateSafeInteger(v.AfterSequence); err != nil {
		return err
	}
	if v.ThroughSequence != nil {
		if err := validateSafeInteger(*v.ThroughSequence); err != nil {
			return err
		}
		if *v.ThroughSequence <= v.AfterSequence {
			return fmt.Errorf("stav: verification ceiling must follow cursor")
		}
	}
	return nil
}

func (r LocalRequest) Validate() error {
	if r.Schema != SchemaLocalRequest {
		return fmt.Errorf("stav: invalid local request schema")
	}
	if err := ValidateTOPSID(r.TOPSID); err != nil {
		return err
	}
	if err := ValidateRequestUUID(r.RequestID); err != nil {
		return err
	}
	switch r.Operation {
	case LocalOperationAppend:
		if r.Candidate == nil || r.Query != nil || r.Verify != nil {
			return fmt.Errorf("stav: invalid append request payload")
		}
		if err := r.Candidate.Validate(); err != nil {
			return err
		}
		if r.Candidate.Topology.TOPSID != r.TOPSID || r.Candidate.Correlation.RequestID != r.RequestID {
			return fmt.Errorf("stav: append request binding mismatch")
		}
	case LocalOperationStatus:
		if r.Candidate != nil || r.Query != nil || r.Verify != nil {
			return fmt.Errorf("stav: invalid status request payload")
		}
	case LocalOperationQuery:
		if r.Candidate != nil || r.Query == nil || r.Verify != nil {
			return fmt.Errorf("stav: invalid query request payload")
		}
		if err := r.Query.Validate(); err != nil {
			return err
		}
		if r.Query.TOPSID != r.TOPSID {
			return fmt.Errorf("stav: query request binding mismatch")
		}
	case LocalOperationVerify:
		if r.Candidate != nil || r.Query != nil || r.Verify == nil {
			return fmt.Errorf("stav: invalid verify request payload")
		}
		return r.Verify.validate()
	default:
		return fmt.Errorf("stav: invalid local operation")
	}
	return nil
}

func (r LocalResponse) Validate() error {
	if r.Schema != SchemaLocalResponse {
		return fmt.Errorf("stav: invalid local response schema")
	}
	if err := ValidateTOPSID(r.TOPSID); err != nil {
		return err
	}
	if err := ValidateRequestUUID(r.RequestID); err != nil {
		return err
	}
	if err := validateRegisteredIdentifier(r.ReasonCode); err != nil {
		return err
	}
	if r.Disposition == LocalDispositionRejected {
		if r.Receipt != nil || r.Page != nil || r.Status != nil || r.Verification != nil {
			return fmt.Errorf("stav: rejected response cannot contain payload")
		}
		switch r.Operation {
		case LocalOperationAppend, LocalOperationQuery, LocalOperationStatus, LocalOperationVerify:
			return nil
		default:
			return fmt.Errorf("stav: invalid rejected response operation")
		}
	}
	if r.Disposition != LocalDispositionSucceeded || r.ReasonCode != ReasonResponseSucceeded {
		return fmt.Errorf("stav: invalid successful response disposition")
	}
	switch r.Operation {
	case LocalOperationAppend:
		if r.Receipt == nil || r.Page != nil || r.Status != nil || r.Verification != nil {
			return fmt.Errorf("stav: invalid append response payload")
		}
		if err := r.Receipt.Validate(); err != nil {
			return err
		}
		if r.Receipt.RequestID != r.RequestID || r.Receipt.TOPSID != r.TOPSID {
			return fmt.Errorf("stav: append response binding mismatch")
		}
	case LocalOperationQuery:
		if r.Receipt != nil || r.Page == nil || r.Status != nil || r.Verification != nil {
			return fmt.Errorf("stav: invalid query response payload")
		}
		if err := r.Page.Validate(); err != nil {
			return err
		}
		if r.Page.TOPSID != r.TOPSID {
			return fmt.Errorf("stav: query response binding mismatch")
		}
	case LocalOperationStatus:
		if r.Receipt != nil || r.Page != nil || r.Status == nil || r.Verification != nil {
			return fmt.Errorf("stav: invalid status response payload")
		}
		if err := r.Status.Validate(); err != nil {
			return err
		}
		if r.Status.TOPSID != r.TOPSID {
			return fmt.Errorf("stav: status response binding mismatch")
		}
	case LocalOperationVerify:
		if r.Receipt != nil || r.Page != nil || r.Status != nil || r.Verification == nil {
			return fmt.Errorf("stav: invalid verification response payload")
		}
		if err := r.Verification.Validate(); err != nil {
			return err
		}
		if r.Verification.TOPSID != r.TOPSID {
			return fmt.Errorf("stav: verification response binding mismatch")
		}
	default:
		return fmt.Errorf("stav: invalid successful response operation")
	}
	return nil
}

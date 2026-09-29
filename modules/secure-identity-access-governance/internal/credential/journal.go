package credential

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/policy"
)

const maxJournalRecordBytes = 8192
const journalFormat = "ssiag-internal-credential-request-1"

var errJournal = errors.New("credential journal unavailable")

// Journal holds no open descriptors. Its root is an existing private SSIAG state
// directory, supplied by the trusted owner, never a request. No retention purge
// or request-ID reuse exists. It is a local internal format, not an IPC protocol.
type Journal struct {
	root, topsID string
	open         func(*Journal, string, bool) (journalFile, error)
}

func NewJournal(stateDir, topsID string) (*Journal, error) {
	if !filepath.IsAbs(stateDir) || filepath.Clean(stateDir) != stateDir || stateDir == "/" || paths.ValidateTOPSID(topsID) != nil {
		return nil, errJournal
	}
	return &Journal{root: stateDir, topsID: topsID, open: openJournalFile}, nil
}

type journalIdentity struct {
	TOPSID       string
	RequestID    string
	Subject      model.DecisionSubject
	UID          uint32
	GID          uint32
	IntentDigest string
}
type journalRecord struct {
	Format   string
	Identity journalIdentity
	Phase    string
	Binding  *UseBinding
	Outcome  DispatchStatus
	Digest   string
}

// File handles stay pinned from initial claim through result commitment.
type journalFile interface {
	read() ([]byte, bool, error)
	write([]byte) error
	close()
}
type journalAttempt struct {
	file    journalFile
	record  journalRecord
	started bool
}

func intentValid(in DispatchIntent) bool {
	a := in.Authorization
	if policy.ValidateRequest(a, a.RequestedAt) != nil || a.RequestedExpiresAt.Sub(a.RequestedAt) > MaxUseLifetime ||
		stav.ValidateRequestUUID(a.RequestID) != nil || stav.ValidateRequestUUID(a.CorrelationID) != nil || in.MaximumBytes == 0 || in.MaximumBytes > MaxUseBytes {
		return false
	}
	for _, v := range [...]string{in.Reference.Provider, in.Reference.Name, in.Reference.Version, in.Reference.Type} {
		if !useToken(v) {
			return false
		}
	}
	return true
}
func journalKey(tops string, peer peerauth.Peer, in DispatchIntent) journalIdentity {
	data, _ := json.Marshal(in)
	return journalIdentity{TOPSID: tops, RequestID: in.Authorization.RequestID, Subject: model.DecisionSubject{ID: peer.Subject.ID, Kind: peer.Subject.Kind, Authority: peer.Subject.Authority}, UID: peer.Credentials.UID, GID: peer.Credentials.GID, IntentDigest: journalDigest("intent", data)}
}
func journalDigest(domain string, data []byte) string {
	sum := sha256.Sum256(append([]byte(journalFormat+":"+domain+"\x00"), data...))
	return "sha256:" + hex.EncodeToString(sum[:])
}
func validJournalRecord(r journalRecord) bool {
	k := r.Identity
	if r.Format != journalFormat || paths.ValidateTOPSID(k.TOPSID) != nil || stav.ValidateRequestUUID(k.RequestID) != nil || !useDigest(k.IntentDigest) || !useToken(k.Subject.ID) || !useToken(k.Subject.Kind) || k.Subject.Authority != peerauth.Mechanism {
		return false
	}
	if r.Binding != nil {
		b := *r.Binding
		if !validUse(b, b.IssuedAt) || b.TOPSID != k.TOPSID || b.RequestID != k.RequestID || b.SubjectID != k.Subject.ID || b.SubjectKind != k.Subject.Kind || b.SubjectAuthority != k.Subject.Authority {
			return false
		}
	}
	switch r.Phase {
	case "intent":
		return r.Binding == nil && r.Outcome == ""
	case "armed":
		return r.Binding != nil && r.Outcome == ""
	case "closed":
		if r.Binding != nil {
			return r.Outcome == DispatchDelivered || r.Outcome == DispatchRefused || r.Outcome == DispatchIndeterminate
		}
		switch r.Outcome {
		case DispatchInvalid, DispatchDenied, DispatchAuditFailed, DispatchUnavailable, DispatchCancelled, DispatchRefused, DispatchNotDispatched:
			return true
		}
	}
	return false
}
func encodeJournal(r journalRecord) ([]byte, error) {
	if !validJournalRecord(r) {
		return nil, errJournal
	}
	r.Digest = ""
	body, err := json.Marshal(r)
	if err != nil {
		return nil, errJournal
	}
	r.Digest = journalDigest("record", body)
	data, err := json.Marshal(r)
	data = append(data, '\n')
	if err != nil || len(data) > maxJournalRecordBytes {
		return nil, errJournal
	}
	return data, nil
}
func decodeJournal(data []byte) (journalRecord, error) {
	var r journalRecord
	if len(data) > maxJournalRecordBytes || json.Unmarshal(data, &r) != nil {
		return r, errJournal
	}
	canonical, err := encodeJournal(r)
	// Exact canonical re-encoding also rejects duplicate, unknown, omitted and
	// trailing fields, as well as altered digest or noncanonical timestamps.
	if err != nil || !bytes.Equal(canonical, data) {
		return journalRecord{}, errJournal
	}
	return r, nil
}
func (j *Journal) begin(ctx context.Context, k journalIdentity) (*journalAttempt, DispatchStatus) {
	if j == nil || j.open == nil || ctx == nil || ctx.Err() != nil || k.TOPSID != j.topsID {
		return nil, DispatchUnavailable
	}
	f, err := j.open(j, k.RequestID, true)
	if err != nil {
		return nil, DispatchUnavailable
	}
	data, exists, err := f.read()
	if err != nil {
		f.close()
		return nil, DispatchUnavailable
	}
	if exists {
		r, err := decodeJournal(data)
		f.close()
		if err != nil {
			return nil, DispatchUnavailable
		}
		if r.Identity != k {
			return nil, DispatchConflict
		}
		if r.Phase == "closed" {
			return nil, DispatchRecorded
		}
		return nil, DispatchRecoveryRequired
	}
	r := journalRecord{Format: journalFormat, Identity: k, Phase: "intent"}
	payload, err := encodeJournal(r)
	if err != nil || ctx.Err() != nil {
		f.close()
		return nil, DispatchUnavailable
	}
	if f.write(payload) != nil {
		f.close()
		return nil, DispatchUnavailable
	}
	return &journalAttempt{file: f, record: r}, ""
}
func (a *journalAttempt) arm(b UseBinding) error {
	if a == nil || a.record.Phase != "intent" {
		return errJournal
	}
	r := a.record
	r.Phase = "armed"
	r.Binding = &b
	data, err := encodeJournal(r)
	if err != nil {
		return errJournal
	}
	if a.file.write(data) != nil {
		return errJournal
	}
	a.record = r
	a.started = true
	return nil
}
func (a *journalAttempt) finish(outcome DispatchStatus) error {
	r := a.record
	r.Phase = "closed"
	r.Outcome = outcome
	data, err := encodeJournal(r)
	if err != nil {
		return errJournal
	}
	if a.file.write(data) != nil {
		return errJournal
	}
	a.record = r
	return nil
}

// RecoveryResult reports durable metadata; Delivered here describes the earlier
// attempt, not a delivery performed by recovery. No retry authority is returned.
type RecoveryResult struct {
	State   string
	Outcome DispatchStatus
}

func (j *Journal) recover(ctx context.Context, k journalIdentity) (RecoveryResult, error) {
	if j == nil || j.open == nil || ctx == nil || ctx.Err() != nil || k.TOPSID != j.topsID {
		return RecoveryResult{}, errJournal
	}
	f, err := j.open(j, k.RequestID, false)
	if err != nil {
		return RecoveryResult{}, errJournal
	}
	defer f.close()
	data, exists, err := f.read()
	if err != nil || !exists {
		return RecoveryResult{}, errJournal
	}
	r, err := decodeJournal(data)
	if err != nil || r.Identity != k {
		return RecoveryResult{}, errJournal
	}
	if r.Phase != "closed" {
		r.Outcome = DispatchNotDispatched
		if r.Phase == "armed" {
			r.Outcome = DispatchIndeterminate
		}
		r.Phase = "closed"
		data, err = encodeJournal(r)
		if err != nil || ctx.Err() != nil {
			return RecoveryResult{}, errJournal
		}
		if f.write(data) != nil {
			return RecoveryResult{}, errJournal
		}
	}
	return RecoveryResult{State: r.Phase, Outcome: r.Outcome}, nil
}

// Recover requires a new, fresh authorization request for the original exact
// tuple and the same kernel-derived identity. It audits that permission check
// and seals abandoned state without invoking ProviderAdmission or Execute.
func (d *Dispatcher) Recover(ctx context.Context, original DispatchIntent, authorization model.AuthorizationRequest) (RecoveryResult, error) {
	if d == nil || d.journal == nil || ctx == nil || ctx.Err() != nil || !intentValid(original) || authorization.RequestID == original.Authorization.RequestID {
		return RecoveryResult{}, errJournal
	}
	peer, err := peerauth.PeerFromContext(ctx)
	if err != nil || !peer.Mapped {
		return RecoveryResult{}, errJournal
	}
	a := original.Authorization
	if authorization.Operation != a.Operation || authorization.Resource != a.Resource || authorization.Audience != a.Audience || authorization.Scope != a.Scope || policy.ValidateRequest(authorization, d.now().UTC()) != nil {
		return RecoveryResult{}, errJournal
	}
	deadline := d.now().Add(MaxUseLifetime)
	if authorization.RequestedExpiresAt.Before(deadline) {
		deadline = authorization.RequestedExpiresAt
	}
	bounded, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	var result RecoveryResult
	err = d.policy.WithDecision(bounded, peer.Subject, authorization, func(decision model.AuthorizationDecision) error {
		if decision.TOPSID != d.journal.topsID {
			return errJournal
		}
		if _, err := d.auditDecision(bounded, decision); err != nil || decision.Effect != "allow" {
			return errJournal
		}
		var err error
		result, err = d.journal.recover(bounded, journalKey(decision.TOPSID, peer, original))
		return err
	})
	if err != nil {
		return RecoveryResult{}, errJournal
	}
	return result, nil
}

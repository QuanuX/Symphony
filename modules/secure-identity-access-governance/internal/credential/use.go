package credential

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

// This is an internal lifecycle guard, not an authorization token, provider
// operation or wire protocol. Only a future owner-authenticated dispatcher may
// supply the current binding after independently verifying authority and audit.
const (
	MaxUseLifetime        = 30 * time.Second
	MaxUseBytes    uint32 = 4096
)

type UseStatus string

const (
	UseOK        UseStatus = "ok"
	UseInvalid   UseStatus = "invalid_binding"
	UseExpired   UseStatus = "expired"
	UseCancelled UseStatus = "cancelled"
	UseStale     UseStatus = "binding_changed"
	UseSpent     UseStatus = "already_consumed"
)

// UseBinding contains only safe identifiers and bounded lifecycle metadata.
// Reference.Version is mandatory here: an omitted/latest generation can never
// select replacement credential material implicitly. Secret values do not belong
// in any field. Digests bind nonsecret authority/executable evidence only.
// Strings and value fields make ownership independent of mutable caller slices.
type UseBinding struct {
	TOPSID                string
	SubjectID             string
	SubjectKind           string
	SubjectAuthority      string
	ConsumerDigest        string
	ProviderBindingDigest string
	PolicyDigest          string
	ConfigDigest          string
	AuthorizationDigest   string
	AuditReceiptDigest    string
	LeaseID               string
	RequestID             string
	CorrelationID         string
	Reference             Reference
	Operation             string
	Resource              string
	Audience              string
	Scope                 string
	MaximumBytes          uint32
	IssuedAt              time.Time
	Deadline              time.Time
}

// OneShotUse is process-local and exposes no serializable state for restoration.
// It must not be copied after construction. No reference to secret bytes is held.
// A caller must obtain a NEW independently authorized use after any attempt;
// this guard never retries or interprets failed delivery as safe to replay.
type OneShotUse struct {
	binding  UseBinding
	consumed atomic.Bool
}

func NewOneShotUse(binding UseBinding, now time.Time) (*OneShotUse, UseStatus) {
	if !validUse(binding, now) {
		return nil, UseInvalid
	}
	if !now.Before(binding.Deadline) {
		return nil, UseExpired
	}
	return &OneShotUse{binding: canonicalUse(binding)}, UseOK
}

// Consume burns the use before validating the current snapshot. Exactly one
// concurrent attempt can proceed; cancellation, expiry and drift also burn it.
// An OK result proves only this local guard's checks. The dispatcher must hold
// its authority/provider snapshot stable through dispatch and bind the same
// tuple to the authenticated recipient and protected one-shot channel. It must
// recheck cancellation during I/O and treat ambiguous delivery as indeterminate.
func (u *OneShotUse) Consume(ctx context.Context, current UseBinding, now time.Time) UseStatus {
	if u == nil {
		return UseInvalid
	}
	if !u.consumed.CompareAndSwap(false, true) {
		return UseSpent
	}
	if ctx == nil {
		return UseInvalid
	}
	if ctx.Err() != nil {
		return UseCancelled
	}
	if now.IsZero() || now.Location() != time.UTC {
		return UseInvalid
	}
	if now.Before(u.binding.IssuedAt) || !now.Before(u.binding.Deadline) {
		return UseExpired
	}
	if !validUse(current, now) || canonicalUse(current) != u.binding {
		return UseStale
	}
	if ctx.Err() != nil {
		return UseCancelled
	}
	return UseOK
}

func canonicalUse(b UseBinding) UseBinding {
	b.IssuedAt = b.IssuedAt.Round(0)
	b.Deadline = b.Deadline.Round(0)
	return b
}

func validUse(b UseBinding, now time.Time) bool {
	if paths.ValidateTOPSID(b.TOPSID) != nil || now.IsZero() || now.Location() != time.UTC {
		return false
	}
	for _, v := range [...]string{b.SubjectID, b.SubjectKind, b.SubjectAuthority, b.LeaseID, b.RequestID, b.CorrelationID,
		b.Reference.Provider, b.Reference.Name, b.Reference.Version, b.Reference.Type, b.Operation, b.Resource, b.Audience, b.Scope} {
		if !useToken(v) {
			return false
		}
	}
	for _, v := range [...]string{b.ConsumerDigest, b.ProviderBindingDigest, b.PolicyDigest, b.ConfigDigest, b.AuthorizationDigest, b.AuditReceiptDigest} {
		if !useDigest(v) {
			return false
		}
	}
	if b.MaximumBytes == 0 || b.MaximumBytes > MaxUseBytes {
		return false
	}
	if b.IssuedAt.IsZero() || b.Deadline.IsZero() || b.IssuedAt.Location() != time.UTC || b.Deadline.Location() != time.UTC {
		return false
	}
	if b.IssuedAt.After(now) || !b.Deadline.After(b.IssuedAt) || b.Deadline.Sub(b.IssuedAt) > MaxUseLifetime {
		return false
	}
	return true
}

func useToken(s string) bool {
	if s == "" || len(s) > 256 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("._:-", c)) {
			return false
		}
	}
	return true
}

func useDigest(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	for _, c := range s[7:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

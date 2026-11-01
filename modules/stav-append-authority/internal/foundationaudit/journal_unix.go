//go:build darwin || linux

package foundationaudit

import (
	"context"
	"errors"
	"fmt"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// Journal is safe metadata owned by the native lifecycle transaction, not a
// second append writer. The caller holds its existing lifecycle lock throughout
// these operations. Only SSIAG's closed producer submits the candidate.
type Journal struct {
	Base          Request   `json:"base"`
	OwnerUID      uint32    `json:"owner_uid"`
	Intent        *Response `json:"intent,omitempty"`
	Outcome       *Request  `json:"outcome,omitempty"`
	Completion    *Response `json:"completion,omitempty"`
	AttemptDigest string    `json:"attempt_digest"`
	Digest        string    `json:"digest"`
	path          string
}

func journalPath(directory, operationID string) string {
	return filepath.Join(directory, Hash(operationID)[7:]+".json")
}

func privateDirectory(directory string) error {
	if err := os.Mkdir(directory, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	st, err := os.Lstat(directory)
	if err != nil {
		return err
	}
	raw, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 || raw.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("unsafe foundational audit directory")
	}
	return nil
}

func Open(directory string, base Request) (*Journal, error) {
	// Deferred bootstrap does not request an authorization receipt, but its base
	// uses the same intent shape so later outcome binding cannot change identity.
	check := base
	check.AuditMode = "ordinary"
	if check.Phase != "intent" || check.Validate() != nil || (base.AuditMode != "ordinary" && base.AuditMode != "audit_deferred") {
		return nil, fmt.Errorf("invalid foundational audit journal intent")
	}
	if err := privateDirectory(directory); err != nil {
		return nil, err
	}
	j, err := Load(directory, base.OperationID)
	if err == nil {
		if Hash(j.Base) != Hash(base) {
			return nil, fmt.Errorf("foundational audit operation reuse conflict")
		}
		return j, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	j = &Journal{Base: base, OwnerUID: uint32(os.Geteuid()), path: journalPath(directory, base.OperationID)}
	return j, j.save()
}

func Load(directory, operationID string) (*Journal, error) {
	if !operation.MatchString(operationID) {
		return nil, fmt.Errorf("invalid foundational audit operation")
	}
	path := journalPath(directory, operationID)
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	raw, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || raw.Uid != uint32(os.Geteuid()) || st.Size() > 65536 {
		return nil, fmt.Errorf("unsafe foundational audit journal")
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return nil, err
	}
	var j Journal
	if err := Decode(data, &j); err != nil {
		return nil, err
	}
	want := j.Digest
	j.Digest = ""
	if Hash(j) != want || j.OwnerUID != uint32(os.Geteuid()) || j.Base.OperationID != operationID {
		return nil, fmt.Errorf("foundational audit journal binding mismatch")
	}
	j.Digest = want
	j.path = path
	check := j.Base
	check.AuditMode = "ordinary"
	if check.Phase != "intent" || check.Validate() != nil || (j.Base.AuditMode != "ordinary" && j.Base.AuditMode != "audit_deferred") {
		return nil, fmt.Errorf("invalid stored foundational intent")
	}
	if j.Intent != nil && j.Intent.Validate(j.Base, j.OwnerUID) != nil {
		return nil, fmt.Errorf("invalid stored intent receipt")
	}
	if j.Outcome != nil {
		want := j.Base
		want.Phase = "outcome"
		want.RequestID = j.Outcome.RequestID
		want.ObservedAt = j.Outcome.ObservedAt
		want.ObservedStateDigest = j.Outcome.ObservedStateDigest
		if want.RequestID == j.Base.RequestID || want.Validate() != nil || Hash(want) != Hash(*j.Outcome) {
			return nil, fmt.Errorf("stored outcome changed intent identity")
		}
	}
	if j.Completion != nil && (j.Outcome == nil || j.Completion.Validate(*j.Outcome, j.OwnerUID) != nil) {
		return nil, fmt.Errorf("invalid stored outcome receipt")
	}
	return &j, nil
}

func (j *Journal) save() error {
	j.Digest = ""
	j.Digest = Hash(j)
	data, err := jsonBytes(j)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(j.path), ".audit-*")
	if err != nil {
		return err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(temporary, j.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(j.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (j *Journal) BindAttempt(value string) error {
	if !digest.MatchString(value) {
		return fmt.Errorf("invalid lifecycle attempt binding")
	}
	j.AttemptDigest = value
	return j.save()
}

func (j *Journal) CommitIntent(ctx context.Context, submit Submit) error {
	if j.Base.AuditMode != "ordinary" {
		return nil
	}
	if j.Intent != nil {
		return j.Intent.Validate(j.Base, j.OwnerUID)
	}
	if submit == nil {
		return fmt.Errorf("SSIAG audit bridge unavailable")
	}
	response, err := submit(ctx, j.Base)
	if err != nil {
		return err
	}
	if err = response.Validate(j.Base, j.OwnerUID); err != nil {
		return err
	}
	j.Intent = &response
	return j.save()
}

// Observe persists the proven native outcome before any network attempt. Later
// reconciliation reuses these exact bytes even if present state has changed.
func (j *Journal) Observe(stateDigest, observedAt string) error {
	if j.Outcome != nil {
		return nil
	}
	r := j.Base
	r.Phase = "outcome"
	var err error
	r.RequestID, err = stav.GenerateUUIDv4()
	if err != nil {
		return err
	}
	r.ObservedStateDigest = stateDigest
	r.ObservedAt = observedAt
	if err := r.Validate(); err != nil {
		return err
	}
	j.Outcome = &r
	return j.save()
}

func (j *Journal) Reconcile(ctx context.Context, submit Submit) error {
	if j.Outcome == nil {
		return fmt.Errorf("no observed native outcome to reconcile")
	}
	if j.Completion != nil {
		return j.Completion.Validate(*j.Outcome, j.OwnerUID)
	}
	if submit == nil {
		return fmt.Errorf("SSIAG audit bridge unavailable")
	}
	response, err := submit(ctx, *j.Outcome)
	if err != nil {
		return err
	}
	if err = response.Validate(*j.Outcome, j.OwnerUID); err != nil {
		return err
	}
	j.Completion = &response
	return j.save()
}

func jsonBytes(v any) ([]byte, error) { return stav.MarshalCanonical(v) }

package credential

import (
	"bytes"
	"encoding/json"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"sort"
)

const MaxAdmissionGrants = 64
const maxAdmissionRecordBytes = 1 << 20
const admissionFormat = "ssiag-internal-credential-admission-1"

type admissionRecord struct {
	Format          string
	TOPSID          string
	AuthorityDigest string
	Revision        uint64
	Grants          []AdmissionGrant
	Digest          string
}

// OpenAdmissionRegistry restores nonsecret owner-selected admission state under
// one lifetime filesystem lock. authorityDigest binds the owner's current
// protected authority configuration, never an argument received from a caller.
// Reopening with different authority refuses; it never erases/reissues leases.
// This is an internal persistence primitive, not an issuance or audit route.
func OpenAdmissionRegistry(stateDir, topsID, authorityDigest string, native ProviderAdmission) (*AdmissionRegistry, error) {
	if !useDigest(authorityDigest) {
		return nil, errAdmission
	}
	r, err := NewAdmissionRegistry(topsID, native)
	if err != nil {
		return nil, errAdmission
	}
	j, err := NewJournal(stateDir, topsID)
	if err != nil {
		return nil, errAdmission
	}
	f, err := openAdmissionFile(j)
	if err != nil {
		return nil, errAdmission
	}
	keep := false
	defer func() {
		if !keep {
			f.close()
		}
	}()
	data, exists, err := f.read()
	if err != nil {
		return nil, errAdmission
	}
	if exists {
		record, err := decodeAdmission(data)
		if err != nil || record.TOPSID != topsID || record.AuthorityDigest != authorityDigest {
			return nil, errAdmission
		}
		r.revision = record.Revision
		// Expired entries remain retained history, but Pin always checks current time.
		// Current policy and native identity are independently required by dispatch.
		for _, grant := range record.Grants {
			r.grants[admissionKey{grant.Subject, grant.Target}] = grant
		}
	}
	r.persist = func(revision uint64, grants map[admissionKey]AdmissionGrant) error {
		record := admissionRecord{Format: admissionFormat, TOPSID: topsID, AuthorityDigest: authorityDigest, Revision: revision, Grants: make([]AdmissionGrant, 0, len(grants))}
		for _, grant := range grants {
			record.Grants = append(record.Grants, grant)
		}
		sort.Slice(record.Grants, func(i, j int) bool { return record.Grants[i].LeaseID < record.Grants[j].LeaseID })
		data, err := encodeAdmission(record)
		if err != nil {
			return errAdmission
		}
		return f.write(data)
	}
	if !exists && r.persist(r.revision, r.grants) != nil {
		return nil, errAdmission
	}
	r.release = f.close
	keep = true
	return r, nil
}

// Revision returns zero when closed or persistence is uncertain. It confers no
// authority and cannot be used to infer that a lease is presently usable.
func (r *AdmissionRegistry) Revision() uint64 {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.unavailable {
		return 0
	}
	return r.revision
}

// Close never crosses an admitted native pin. Busy requires release/cleanup and
// an explicit retry. It neither revokes persisted grants nor deletes history.
func (r *AdmissionRegistry) Close() AdmissionUpdate {
	if r == nil {
		return AdmissionInvalid
	}
	if !r.mu.TryLock() {
		return AdmissionBusy
	}
	defer r.mu.Unlock()
	r.unavailable = true
	if r.release != nil {
		r.release()
		r.release = nil
	}
	return AdmissionUpdated
}

func encodeAdmission(record admissionRecord) ([]byte, error) {
	if record.Format != admissionFormat || !useDigest(record.AuthorityDigest) || record.Revision == 0 || record.Grants == nil || len(record.Grants) > MaxAdmissionGrants {
		return nil, errAdmission
	}
	// Reuse the constructor's TOPS grammar without opening the filesystem.
	if paths.ValidateTOPSID(record.TOPSID) != nil {
		return nil, errAdmission
	}
	seen := make(map[admissionKey]bool)
	previous := ""
	for _, grant := range record.Grants {
		key := admissionKey{grant.Subject, grant.Target}
		if !validAdmissionGrant(grant) || seen[key] || grant.LeaseID <= previous {
			return nil, errAdmission
		}
		seen[key] = true
		previous = grant.LeaseID
	}
	record.Digest = ""
	body, err := json.Marshal(record)
	if err != nil {
		return nil, errAdmission
	}
	record.Digest = journalVersionDigest(admissionFormat, "snapshot", body)
	data, err := json.Marshal(record)
	data = append(data, '\n')
	if err != nil || len(data) > maxAdmissionRecordBytes {
		return nil, errAdmission
	}
	return data, nil
}

func decodeAdmission(data []byte) (admissionRecord, error) {
	var record admissionRecord
	if len(data) > maxAdmissionRecordBytes || json.Unmarshal(data, &record) != nil {
		return record, errAdmission
	}
	exact, err := encodeAdmission(record)
	if err != nil || !bytes.Equal(exact, data) {
		return admissionRecord{}, errAdmission
	}
	return record, nil
}

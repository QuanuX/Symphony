package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const recordOverhead = 4 + sha256.Size

var (
	ErrLedgerFull          = errors.New("stav storage: ledger full")
	ErrLedgerUnavailable   = errors.New("stav storage: ledger unavailable")
	ErrIdempotencyConflict = errors.New("stav storage: idempotency conflict")
)

type entry struct {
	event  stavprotocol.Event
	digest string
}

type idempotencyRecord struct {
	candidateDigest string
	receipt         stavprotocol.Receipt
}

// Ledger owns one locked, append-only STAV serialization domain.
type Ledger struct {
	mu            sync.RWMutex
	file          *os.File
	path          string
	recoveryDir   string
	topsID        string
	maxBytes      uint64
	genesisDigest string
	entries       []entry
	idempotency   map[string]idempotencyRecord
	size          uint64
	recoveredTail bool
	poisoned      error
}

// Open exclusively locks, scans, and verifies one per-TOPS ledger. It recovers
// only an incomplete final frame and never repairs complete corruption.
func Open(path, recoveryDir, topsID string, maxBytes uint64) (*Ledger, error) {
	if err := stavprotocol.ValidateTOPSID(topsID); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) || !filepath.IsAbs(recoveryDir) || maxBytes < 1_048_576 || maxBytes > stavprotocol.MaxSafeInteger {
		return nil, fmt.Errorf("stav storage: invalid open parameters")
	}
	if err := ensurePrivateDirectory(filepath.Dir(path)); err != nil {
		return nil, err
	}
	if err := ensurePrivateDirectory(recoveryDir); err != nil {
		return nil, err
	}
	file, created, err := openRegularNoFollow(path, 0o600)
	if err != nil {
		return nil, fmt.Errorf("stav storage: open ledger: %w", err)
	}
	if created {
		if err := syncDirectory(filepath.Dir(path)); err != nil {
			_ = file.Close()
			return nil, err
		}
	}
	if err := lockFile(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("stav storage: acquire exclusive ledger lock: %w", err)
	}
	genesis, err := stavprotocol.GenesisDigest(topsID)
	if err != nil {
		_ = unlockFile(file)
		_ = file.Close()
		return nil, err
	}
	ledger := &Ledger{
		file:          file,
		path:          filepath.Clean(path),
		recoveryDir:   filepath.Clean(recoveryDir),
		topsID:        topsID,
		maxBytes:      maxBytes,
		genesisDigest: genesis,
		idempotency:   make(map[string]idempotencyRecord),
	}
	if err := ledger.scan(); err != nil {
		_ = unlockFile(file)
		_ = file.Close()
		return nil, err
	}
	if ledger.size > maxBytes {
		_ = unlockFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("stav storage: existing ledger exceeds configured maximum")
	}
	if _, err := file.Seek(int64(ledger.size), io.SeekStart); err != nil {
		_ = unlockFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("stav storage: seek append position: %w", err)
	}
	return ledger, nil
}

func (l *Ledger) scan() error {
	info, err := l.file.Stat()
	if err != nil {
		return fmt.Errorf("stav storage: stat ledger: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 {
		return fmt.Errorf("stav storage: ledger is not a regular file")
	}
	fileSize := uint64(info.Size())
	var offset uint64
	preceding := l.genesisDigest
	for offset < fileSize {
		remaining := fileSize - offset
		if remaining < 4 {
			return l.recoverIncompleteTail(offset, fileSize)
		}
		var header [4]byte
		if _, err := l.file.ReadAt(header[:], int64(offset)); err != nil {
			return fmt.Errorf("stav storage: read frame header at %d: %w", offset, err)
		}
		length := uint64(binary.BigEndian.Uint32(header[:]))
		if length == 0 || length > stavprotocol.MaxEventBytes {
			return fmt.Errorf("stav storage: corrupt frame length at offset %d", offset)
		}
		recordLength := uint64(recordOverhead) + length
		if remaining < recordLength {
			return l.recoverIncompleteTail(offset, fileSize)
		}
		payload := make([]byte, length)
		if _, err := l.file.ReadAt(payload, int64(offset+4)); err != nil {
			return fmt.Errorf("stav storage: read event at offset %d: %w", offset, err)
		}
		var checksum [sha256.Size]byte
		if _, err := l.file.ReadAt(checksum[:], int64(offset+4+length)); err != nil {
			return fmt.Errorf("stav storage: read checksum at offset %d: %w", offset, err)
		}
		gotChecksum := sha256.Sum256(payload)
		if !bytes.Equal(gotChecksum[:], checksum[:]) {
			return fmt.Errorf("stav storage: corrupt frame checksum at offset %d", offset)
		}
		event, err := stavprotocol.DecodeEvent(payload)
		if err != nil {
			return fmt.Errorf("stav storage: corrupt event at offset %d: %w", offset, err)
		}
		wantSequence := uint64(len(l.entries)) + 1
		if event.Topology.TOPSID != l.topsID || event.Ordering.Sequence != wantSequence || event.Integrity.PrecedingEventDigest != preceding {
			return fmt.Errorf("stav storage: chain mismatch at sequence %d", wantSequence)
		}
		eventDigest, err := stavprotocol.EventDigest(event)
		if err != nil {
			return fmt.Errorf("stav storage: digest event at sequence %d: %w", wantSequence, err)
		}
		candidate, err := stavprotocol.CandidateFromEvent(event)
		if err != nil {
			return fmt.Errorf("stav storage: reconstruct candidate at sequence %d: %w", wantSequence, err)
		}
		candidateDigest, err := stavprotocol.CandidateDigest(candidate)
		if err != nil {
			return fmt.Errorf("stav storage: digest candidate at sequence %d: %w", wantSequence, err)
		}
		receipt := committedReceipt(event, eventDigest, candidateDigest)
		if prior, exists := l.idempotency[event.Correlation.RequestID]; exists {
			if prior.candidateDigest != candidateDigest {
				return fmt.Errorf("stav storage: conflicting historical request ID at sequence %d", wantSequence)
			}
		} else {
			l.idempotency[event.Correlation.RequestID] = idempotencyRecord{candidateDigest: candidateDigest, receipt: receipt}
		}
		l.entries = append(l.entries, entry{event: event, digest: eventDigest})
		preceding = eventDigest
		offset += recordLength
	}
	l.size = offset
	return nil
}

func (l *Ledger) recoverIncompleteTail(offset, fileSize uint64) error {
	tail := make([]byte, fileSize-offset)
	if _, err := l.file.ReadAt(tail, int64(offset)); err != nil {
		return fmt.Errorf("stav storage: read incomplete tail: %w", err)
	}
	id, err := stavprotocol.GenerateUUIDv4()
	if err != nil {
		return err
	}
	evidencePath := filepath.Join(l.recoveryDir, "incomplete-tail-"+id+".bin")
	evidence, err := createExclusiveRegularNoFollow(evidencePath, 0o600)
	if err != nil {
		return fmt.Errorf("stav storage: create recovery evidence: %w", err)
	}
	if err := writeFull(evidence, tail); err != nil {
		_ = evidence.Close()
		return fmt.Errorf("stav storage: write recovery evidence: %w", err)
	}
	if err := evidence.Sync(); err != nil {
		_ = evidence.Close()
		return fmt.Errorf("stav storage: sync recovery evidence: %w", err)
	}
	if err := evidence.Close(); err != nil {
		return fmt.Errorf("stav storage: close recovery evidence: %w", err)
	}
	if err := syncDirectory(l.recoveryDir); err != nil {
		return err
	}
	if err := l.file.Truncate(int64(offset)); err != nil {
		return fmt.Errorf("stav storage: truncate incomplete tail: %w", err)
	}
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("stav storage: sync recovered ledger: %w", err)
	}
	l.size = offset
	l.recoveredTail = true
	return nil
}

// Append serializes one valid candidate and returns only after fsync succeeds.
func (l *Ledger) Append(candidate stavprotocol.Candidate, producer stavprotocol.SafeReference, now time.Time) (stavprotocol.Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.poisoned != nil {
		return stavprotocol.Receipt{}, fmt.Errorf("%w: %v", ErrLedgerUnavailable, l.poisoned)
	}
	if err := candidate.Validate(); err != nil {
		return stavprotocol.Receipt{}, err
	}
	if candidate.Topology.TOPSID != l.topsID {
		return stavprotocol.Receipt{}, fmt.Errorf("stav storage: candidate TOPS mismatch")
	}
	candidateDigest, err := stavprotocol.CandidateDigest(candidate)
	if err != nil {
		return stavprotocol.Receipt{}, err
	}
	if prior, exists := l.idempotency[candidate.Correlation.RequestID]; exists {
		if prior.candidateDigest != candidateDigest {
			return stavprotocol.Receipt{}, ErrIdempotencyConflict
		}
		return prior.receipt, nil
	}
	sequence := uint64(len(l.entries)) + 1
	if sequence > stavprotocol.MaxSafeInteger {
		return stavprotocol.Receipt{}, ErrLedgerFull
	}
	preceding := l.genesisDigest
	if len(l.entries) != 0 {
		preceding = l.entries[len(l.entries)-1].digest
	}
	eventID, err := stavprotocol.GenerateUUIDv4()
	if err != nil {
		return stavprotocol.Receipt{}, err
	}
	event := stavprotocol.Event{
		Actor: stavprotocol.EventActor{
			Authentication: candidate.Actor.Authentication,
			Principal:      candidate.Actor.Principal,
			Producer:       producer,
		},
		Configuration: candidate.Configuration,
		Correlation:   candidate.Correlation,
		Identity: stavprotocol.EventIdentity{
			EventID: eventID,
			Schema:  stavprotocol.SchemaEvent,
		},
		Integrity: stavprotocol.EventIntegrity{PrecedingEventDigest: preceding},
		Operation: candidate.Operation,
		Ordering: stavprotocol.EventOrdering{
			Sequence:  sequence,
			Timestamp: stavprotocol.FormatTimestamp(now),
		},
		Redaction: candidate.Redaction,
		Result:    candidate.Result,
		Topology:  candidate.Topology,
	}
	payload, err := stavprotocol.EncodeEvent(event)
	if err != nil {
		return stavprotocol.Receipt{}, err
	}
	eventDigest, err := stavprotocol.EventDigest(event)
	if err != nil {
		return stavprotocol.Receipt{}, err
	}
	record := make([]byte, recordOverhead+len(payload))
	binary.BigEndian.PutUint32(record[:4], uint32(len(payload)))
	copy(record[4:], payload)
	checksum := sha256.Sum256(payload)
	copy(record[4+len(payload):], checksum[:])
	if uint64(len(record)) > l.maxBytes-l.size {
		return stavprotocol.Receipt{}, ErrLedgerFull
	}
	if err := writeFull(l.file, record); err != nil {
		l.poisoned = err
		return stavprotocol.Receipt{}, fmt.Errorf("%w: append frame: %v", ErrLedgerUnavailable, err)
	}
	if err := l.file.Sync(); err != nil {
		l.poisoned = err
		return stavprotocol.Receipt{}, fmt.Errorf("%w: sync frame: %v", ErrLedgerUnavailable, err)
	}
	l.size += uint64(len(record))
	l.entries = append(l.entries, entry{event: event, digest: eventDigest})
	receipt := committedReceipt(event, eventDigest, candidateDigest)
	l.idempotency[candidate.Correlation.RequestID] = idempotencyRecord{candidateDigest: candidateDigest, receipt: receipt}
	return receipt, nil
}

func committedReceipt(event stavprotocol.Event, eventDigest, candidateDigest string) stavprotocol.Receipt {
	return stavprotocol.Receipt{
		CandidateDigest: candidateDigest,
		Commit: stavprotocol.CommitResult{
			EventDigest: eventDigest,
			EventID:     event.Identity.EventID,
			Sequence:    event.Ordering.Sequence,
			State:       "committed",
			Timestamp:   event.Ordering.Timestamp,
		},
		Disposition: "committed",
		ReasonCode:  stavprotocol.ReasonReceiptCommitted,
		RequestID:   event.Correlation.RequestID,
		Schema:      stavprotocol.SchemaReceipt,
		TOPSID:      event.Topology.TOPSID,
	}
}

// Query returns only classifications explicitly granted to the reader.
func (l *Ledger) Query(query stavprotocol.Query, classifications map[string]bool) (stavprotocol.QueryPage, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if err := query.Validate(); err != nil {
		return stavprotocol.QueryPage{}, err
	}
	if query.TOPSID != l.topsID {
		return stavprotocol.QueryPage{}, fmt.Errorf("stav storage: query TOPS mismatch")
	}
	eventClasses := stringSet(query.EventClasses)
	outcomes := stringSet(query.Outcomes)
	matches := make([]stavprotocol.QueryEntry, 0, query.Limit+1)
	for _, source := range l.entries {
		event := source.event
		if event.Ordering.Sequence <= query.AfterSequence || query.ThroughSequence != nil && event.Ordering.Sequence > *query.ThroughSequence {
			continue
		}
		if !classifications[event.Redaction.Classification] || query.FromTime != "" && event.Ordering.Timestamp < query.FromTime || query.ThroughTime != "" && event.Ordering.Timestamp > query.ThroughTime {
			continue
		}
		if len(eventClasses) != 0 && !eventClasses[event.Operation.EventClass] || len(outcomes) != 0 && !outcomes[event.Result.Outcome] {
			continue
		}
		if query.CorrelationID != "" && query.CorrelationID != event.Correlation.CorrelationID || query.RequestID != "" && query.RequestID != event.Correlation.RequestID {
			continue
		}
		redactionState := "allowlisted"
		if event.Redaction.Classification == "restricted_metadata" {
			redactionState = "restricted"
		}
		matches = append(matches, stavprotocol.QueryEntry{
			EventDigest: source.digest,
			EventID:     event.Identity.EventID,
			Projection: stavprotocol.QueryProjection{
				Actor:         event.Actor.Principal,
				CorrelationID: event.Correlation.CorrelationID,
				EventClass:    event.Operation.EventClass,
				OperationID:   event.Operation.OperationID,
				Outcome:       event.Result.Outcome,
				ReasonCode:    event.Result.ReasonCode,
				RequestID:     event.Correlation.RequestID,
				Target:        event.Operation.Target,
				Timestamp:     event.Ordering.Timestamp,
			},
			RedactionState:    redactionState,
			Sequence:          event.Ordering.Sequence,
			VerificationState: "verified",
		})
		if uint64(len(matches)) > query.Limit {
			break
		}
	}
	pageEntries := matches
	next := stavprotocol.QueryNext{State: "complete"}
	if uint64(len(matches)) > query.Limit {
		pageEntries = matches[:query.Limit]
		next = stavprotocol.QueryNext{AfterSequence: pageEntries[len(pageEntries)-1].Sequence, State: "available"}
	}
	page := stavprotocol.QueryPage{
		AfterSequence: query.AfterSequence,
		Entries:       pageEntries,
		Next:          next,
		Schema:        stavprotocol.SchemaQueryPage,
		TOPSID:        l.topsID,
	}
	if _, err := stavprotocol.EncodeQueryPage(page); err != nil {
		return stavprotocol.QueryPage{}, err
	}
	return page, nil
}

// Verify rechecks the requested in-memory canonical chain range.
func (l *Ledger) Verify(after uint64, through *uint64) stavprotocol.Verification {
	l.mu.RLock()
	defer l.mu.RUnlock()
	last := uint64(len(l.entries))
	ceiling := last
	if through != nil && *through < ceiling {
		ceiling = *through
	}
	if after > ceiling {
		ceiling = after
	}
	result := stavprotocol.VerificationResult{State: "verified"}
	checked := uint64(0)
	preceding := l.genesisDigest
	if after > 0 && after <= last {
		preceding = l.entries[after-1].digest
	}
	for sequence := after + 1; sequence <= ceiling && sequence <= last; sequence++ {
		source := l.entries[sequence-1]
		digest, err := stavprotocol.EventDigest(source.event)
		if err != nil || digest != source.digest || source.event.Integrity.PrecedingEventDigest != preceding || source.event.Ordering.Sequence != sequence || source.event.Topology.TOPSID != l.topsID {
			result = stavprotocol.VerificationResult{AtSequence: sequence, ReasonCode: "symphony.stav.verification.digest-mismatch", State: "failed"}
			break
		}
		checked++
		preceding = digest
	}
	return stavprotocol.Verification{
		AfterSequence:   after,
		EventsChecked:   checked,
		Result:          result,
		Schema:          stavprotocol.SchemaVerification,
		ThroughSequence: ceiling,
		TOPSID:          l.topsID,
	}
}

func (l *Ledger) Status(mode string, ready bool) stavprotocol.AppendAuthorityStatus {
	l.mu.RLock()
	defer l.mu.RUnlock()
	state := "clean"
	if l.recoveredTail {
		state = "recovered_incomplete_tail"
	}
	status := stavprotocol.AppendAuthorityStatus{
		Events:         uint64(len(l.entries)),
		GenesisDigest:  l.genesisDigest,
		LedgerBytes:    l.size,
		LastSequence:   uint64(len(l.entries)),
		MaxLedgerBytes: l.maxBytes,
		Mode:           mode,
		Ready:          ready && l.poisoned == nil,
		RecoveredTail:  l.recoveredTail,
		Schema:         stavprotocol.SchemaAppendAuthorityStatus,
		StorageState:   state,
		TOPSID:         l.topsID,
	}
	if len(l.entries) != 0 {
		status.LastEventDigest = l.entries[len(l.entries)-1].digest
	}
	return status
}

func (l *Ledger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	unlockErr := unlockFile(l.file)
	closeErr := l.file.Close()
	l.file = nil
	if unlockErr != nil {
		return fmt.Errorf("stav storage: unlock ledger: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("stav storage: close ledger: %w", closeErr)
	}
	return nil
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func writeFull(w io.Writer, data []byte) error {
	for len(data) != 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || path == string(filepath.Separator) {
		return fmt.Errorf("stav storage: unsafe directory path")
	}
	parent := filepath.Dir(path)
	if parent != path && parent != string(filepath.Separator) {
		if err := ensurePrivateDirectory(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 && permittedSystemAlias(path) {
			return nil
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("stav storage: unsafe directory")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stav storage: inspect directory: %w", err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return fmt.Errorf("stav storage: create directory: %w", err)
	}
	info, err = os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("stav storage: unsafe directory")
	}
	return nil
}

func permittedSystemAlias(path string) bool {
	expected := map[string]string{"/etc": "/private/etc", "/tmp": "/private/tmp", "/var": "/private/var"}
	want, ok := expected[path]
	if !ok {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == want
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("stav storage: open directory for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("stav storage: sync directory: %w", err)
	}
	return nil
}

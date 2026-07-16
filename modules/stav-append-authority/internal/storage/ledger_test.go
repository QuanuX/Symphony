package storage

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const testTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func testCandidate(requestID, classification string) stavprotocol.Candidate {
	return stavprotocol.Candidate{
		Actor: stavprotocol.CandidateActor{
			Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.fixture", State: "identified"},
			Principal:      stavprotocol.SafeReference{ID: "operator-one", Kind: "symphony.identity.operator"},
		},
		Configuration: stavprotocol.Configuration{ReasonCode: "symphony.stav.configuration.not-applicable", State: "not_applicable"},
		Correlation: stavprotocol.Correlation{
			CorrelationID: "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287",
			RequestID:     requestID,
		},
		Operation: stavprotocol.Operation{
			EventClass:  "symphony.ssiag.authentication.decision",
			OperationID: "symphony.ssiag.authenticate",
			Target:      stavprotocol.SafeReference{ID: "provider-one", Kind: "symphony.ssiag.provider"},
		},
		Redaction: stavprotocol.Redaction{Classification: classification},
		Result: stavprotocol.Result{
			IntentID:   "symphony.ssiag.authentication.requested",
			Outcome:    "allowed",
			ReasonCode: "symphony.ssiag.authentication.allowed",
		},
		Schema:   stavprotocol.SchemaCandidate,
		Topology: stavprotocol.Topology{TOPSID: testTOPSID, TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}},
	}
}

func openTestLedger(t *testing.T) (*Ledger, string, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "state", "ledger-v1.stavlog")
	recovery := filepath.Join(root, "state", "recovery")
	ledger, err := Open(path, recovery, testTOPSID, 1_048_576)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ledger.Close() })
	return ledger, path, recovery
}

func appendCandidate(t *testing.T, ledger *Ledger, candidate stavprotocol.Candidate) stavprotocol.Receipt {
	t.Helper()
	receipt, err := ledger.Append(candidate, stavprotocol.SafeReference{ID: "ssiag", Kind: "symphony.stav.producer"}, time.Date(2026, 7, 16, 12, 0, 0, 1, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := receipt.Validate(); err != nil {
		t.Fatal(err)
	}
	return receipt
}

func TestAppendFsyncReopenAndIdempotency(t *testing.T) {
	ledger, path, recovery := openTestLedger(t)
	candidate := testCandidate("b90e1205-1b3b-4e47-9b91-1cd624cd87cd", "administrative_metadata")
	first := appendCandidate(t, ledger, candidate)
	second := appendCandidate(t, ledger, candidate)
	if first != second {
		t.Fatalf("idempotent receipt changed: %#v != %#v", first, second)
	}
	conflict := candidate
	conflict.Result.Outcome = "denied"
	if _, err := ledger.Append(conflict, stavprotocol.SafeReference{ID: "ssiag", Kind: "symphony.stav.producer"}, time.Now()); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("got %v, want idempotency conflict", err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path, recovery, testTOPSID, 1_048_576)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	replayed, err := reopened.Append(candidate, stavprotocol.SafeReference{ID: "ssiag", Kind: "symphony.stav.producer"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if replayed != first {
		t.Fatal("reconstructed idempotency receipt changed")
	}
	status := reopened.Status("user", true)
	if status.Events != 1 || status.LastSequence != 1 || !status.Ready || status.LastEventDigest != first.Commit.EventDigest {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestExclusiveLedgerOwnership(t *testing.T) {
	ledger, path, recovery := openTestLedger(t)
	if _, err := Open(path, recovery, testTOPSID, 1_048_576); err == nil {
		t.Fatal("second authority unexpectedly acquired the ledger")
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	other, err := Open(path, recovery, testTOPSID, 1_048_576)
	if err != nil {
		t.Fatal(err)
	}
	_ = other.Close()
}

func TestLedgerRejectsPermissionsGrantedToAnotherIdentity(t *testing.T) {
	ledger, path, recovery := openTestLedger(t)
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o660); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, recovery, testTOPSID, 1_048_576); err == nil {
		t.Fatal("ledger accessible to another identity unexpectedly opened")
	}
}

func TestIncompleteTailIsPreservedAndTruncated(t *testing.T) {
	ledger, path, recovery := openTestLedger(t)
	appendCandidate(t, ledger, testCandidate("6dc2310c-f407-4e6a-9506-e237fd622d01", "administrative_metadata"))
	before := ledger.Status("user", true).LedgerBytes
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte{0, 1}); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path, recovery, testTOPSID, 1_048_576)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	status := reopened.Status("user", true)
	if !status.RecoveredTail || status.StorageState != "recovered_incomplete_tail" || status.LedgerBytes != before || status.Events != 1 {
		t.Fatalf("unexpected recovered status: %#v", status)
	}
	files, err := os.ReadDir(recovery)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("recovery files = %d, want 1", len(files))
	}
	evidence, err := os.ReadFile(filepath.Join(recovery, files[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if string(evidence) != string([]byte{0, 1}) {
		t.Fatalf("recovery evidence changed: %v", evidence)
	}
}

func TestInterruptedFrameBoundariesPreserveExactTail(t *testing.T) {
	ledger, path, _ := openTestLedger(t)
	appendCandidate(t, ledger, testCandidate("6dc2310c-f407-4e6a-9506-e237fd622d01", "administrative_metadata"))
	firstSize := ledger.Status("user", true).LedgerBytes
	appendCandidate(t, ledger, testCandidate("8a26caef-2e3f-4d4e-9e44-f1485364f59c", "administrative_metadata"))
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	complete, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	secondFrame := complete[firstSize:]
	cuts := map[string]int{
		"header":   2,
		"payload":  4 + 10,
		"checksum": len(secondFrame) - 1,
	}
	for name, cut := range cuts {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			ledgerPath := filepath.Join(root, "state", "ledger-v1.stavlog")
			recovery := filepath.Join(root, "state", "recovery")
			if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o700); err != nil {
				t.Fatal(err)
			}
			interrupted := append(append([]byte{}, complete[:firstSize]...), secondFrame[:cut]...)
			if err := os.WriteFile(ledgerPath, interrupted, 0o600); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(ledgerPath, recovery, testTOPSID, 1_048_576)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			status := reopened.Status("user", true)
			if status.Events != 1 || !status.RecoveredTail || status.LedgerBytes != firstSize {
				t.Fatalf("unexpected recovery status: %#v", status)
			}
			files, err := os.ReadDir(recovery)
			if err != nil || len(files) != 1 {
				t.Fatalf("recovery evidence count = %d, err=%v", len(files), err)
			}
			evidence, err := os.ReadFile(filepath.Join(recovery, files[0].Name()))
			if err != nil {
				t.Fatal(err)
			}
			if string(evidence) != string(secondFrame[:cut]) {
				t.Fatal("recovery evidence differs from the interrupted bytes")
			}
		})
	}
}

func TestCompleteFrameCorruptionFailsClosed(t *testing.T) {
	ledger, path, recovery := openTestLedger(t)
	appendCandidate(t, ledger, testCandidate("05d45b70-38c0-4459-a584-19afc38ce3c2", "administrative_metadata"))
	status := ledger.Status("user", true)
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte{0xff}, int64(status.LedgerBytes-1)); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, recovery, testTOPSID, 1_048_576); err == nil {
		t.Fatal("complete corruption was silently repaired")
	}
}

func TestCompleteFrameDeletionInsertionAndReorderingFailClosed(t *testing.T) {
	ledger, path, _ := openTestLedger(t)
	appendCandidate(t, ledger, testCandidate("05d45b70-38c0-4459-a584-19afc38ce3c2", "administrative_metadata"))
	firstSize := ledger.Status("user", true).LedgerBytes
	appendCandidate(t, ledger, testCandidate("c1ac95b4-64a8-40ed-bc90-1cc1780baac1", "administrative_metadata"))
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	complete, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	first := complete[:firstSize]
	second := complete[firstSize:]
	mutations := map[string][]byte{
		"deletion":   append([]byte{}, second...),
		"insertion":  append(append(append([]byte{}, first...), first...), second...),
		"reordering": append(append([]byte{}, second...), first...),
	}
	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			ledgerPath := filepath.Join(root, "state", "ledger-v1.stavlog")
			recovery := filepath.Join(root, "state", "recovery")
			if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(ledgerPath, mutation, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(ledgerPath, recovery, testTOPSID, 1_048_576); err == nil {
				t.Fatal("complete structural corruption unexpectedly opened")
			}
		})
	}
}

func TestQueryEnforcesClassificationGrant(t *testing.T) {
	ledger, _, _ := openTestLedger(t)
	appendCandidate(t, ledger, testCandidate("02c25f56-3f3d-4385-9086-f4ea0019009b", "administrative_metadata"))
	appendCandidate(t, ledger, testCandidate("946654b8-83fe-44dc-848c-acbbd8fe47df", "restricted_metadata"))
	query := stavprotocol.Query{
		AfterSequence: 0,
		EventClasses:  []string{},
		Limit:         100,
		Outcomes:      []string{},
		Schema:        stavprotocol.SchemaQuery,
		TOPSID:        testTOPSID,
	}
	page, err := ledger.Query(query, map[string]bool{"administrative_metadata": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 1 || page.Entries[0].Sequence != 1 {
		t.Fatalf("unexpected administrative page: %#v", page)
	}
	page, err = ledger.Query(query, map[string]bool{"administrative_metadata": true, "restricted_metadata": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 2 || page.Entries[1].RedactionState != "restricted" {
		t.Fatalf("unexpected privileged page: %#v", page)
	}
	verification := ledger.Verify(0, nil)
	if err := verification.Validate(); err != nil || verification.EventsChecked != 2 {
		t.Fatalf("unexpected verification: %#v err=%v", verification, err)
	}
}

func TestConcurrentSubmissionsSerializeExactlyOnce(t *testing.T) {
	ledger, _, _ := openTestLedger(t)
	const submissions = 32
	sequences := make(chan uint64, submissions)
	errorsSeen := make(chan error, submissions)
	var workers sync.WaitGroup
	for i := 0; i < submissions; i++ {
		requestID, err := stavprotocol.GenerateUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		candidate := testCandidate(requestID, "administrative_metadata")
		workers.Add(1)
		go func() {
			defer workers.Done()
			receipt, err := ledger.Append(candidate, stavprotocol.SafeReference{ID: "ssiag", Kind: "symphony.stav.producer"}, time.Now())
			if err != nil {
				errorsSeen <- err
				return
			}
			sequences <- receipt.Commit.Sequence
		}()
	}
	workers.Wait()
	close(sequences)
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
	seen := make(map[uint64]bool, submissions)
	for sequence := range sequences {
		if seen[sequence] {
			t.Fatalf("duplicate sequence %d", sequence)
		}
		seen[sequence] = true
	}
	if len(seen) != submissions {
		t.Fatalf("sequences = %d, want %d", len(seen), submissions)
	}
	verification := ledger.Verify(0, nil)
	if verification.Result.State != "verified" || verification.EventsChecked != submissions {
		t.Fatalf("concurrent chain verification failed: %#v", verification)
	}
}

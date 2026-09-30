package snvstate

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func requireBoundary(t *testing.T, err error, code string) {
	t.Helper()
	var refusal *BoundaryError
	if !errors.As(err, &refusal) || refusal.Code != code {
		t.Fatalf("want typed %s, got %v", code, err)
	}
}
func TestSNVTypedLocalRefusalPreservesAcknowledgedState(t *testing.T) {
	evidence, err := NewEvidence(testRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	a := evidenceFixture(t, "acknowledged")
	if err := evidence.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(a); e != nil {
			return e
		}
		return tx.CommitEvidence(a.OperationID, func() error { return nil })
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(evidence.Root, "symphony", "qxctl", "snv", "evidence-v1", "state.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := a
	changed.Installation.ExecutableDigest = "sha256:" + strings.Repeat("d", 64)
	changed.IntentDigest, err = attemptDigest(changed)
	if err != nil {
		t.Fatal(err)
	}
	err = evidence.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(changed); return e })
	requireBoundary(t, err, "snv.intent_conflict")
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("refused immutable intent altered acknowledged journal")
	}
	view, err := NewView(testRoot(t), "00000000-0000-4000-8000-000000000001", "test-view")
	if err != nil {
		t.Fatal(err)
	}
	first := selectionFixture(t, "selected", json.RawMessage("null"), false)
	if err := view.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(first); e != nil {
			return e
		}
		p, _ := tx.Attempt(first.OperationID)
		return tx.CommitSelection(first.OperationID, decisionFixture(t, view, p, false), func() error { return nil })
	}); err != nil {
		t.Fatal(err)
	}
	var prior Document
	if err := view.WithRead(func(tx *Transaction) error { prior = tx.Snapshot(); return nil }); err != nil {
		t.Fatal(err)
	}
	stale := selectionFixture(t, "stale", json.RawMessage("null"), false)
	err = view.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(stale); return e })
	requireBoundary(t, err, "snv.state_conflict")
	if err := view.WithRead(func(tx *Transaction) error {
		if !same(prior, tx.Snapshot()) {
			t.Fatal("refused stale head altered acknowledged state")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func TestSNVTypedRecoverableGuard(t *testing.T) {
	store, err := NewEvidence(testRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	a := evidenceFixture(t, "retained")
	if err := store.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(a); return e }); err != nil {
		t.Fatal(err)
	}
	var prior Document
	if err := store.WithRead(func(tx *Transaction) error { prior = tx.Snapshot(); return nil }); err != nil {
		t.Fatal(err)
	}
	err = store.WithLock(func(tx *Transaction) error {
		return tx.CommitEvidence(a.OperationID, func() error { return errors.New("private-marker") })
	})
	requireBoundary(t, err, "snv.recovery_required")
	if err := store.WithRead(func(tx *Transaction) error {
		if !same(prior, tx.Snapshot()) {
			t.Fatal("opaque failed guard changed acknowledged candidate/journal")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	expired := decisionFixture(t, Store{TOPSID: "00000000-0000-4000-8000-000000000001", ViewID: "test-view"}, Attempt{CorrelationID: "00000000-0000-4000-8000-000000000002"}, true)
	requireBoundary(t, authorizationFresh(expired), "snv.authority_expired")
}

// Controlled, structurally valid history fixture for the public status output
// bound. These are not authenticated grants or actual policy-audit events.
func TestSNVAuthorizationHistoryFixture(t *testing.T) {
	root := os.Getenv("SYMPHONY_SNV_AUTH_HISTORY_FIXTURE")
	if root == "" {
		t.Skip("set explicit disposable history fixture root")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := NewView(root, "00000000-0000-4000-8000-000000000001", "test-view")
	if err != nil {
		t.Fatal(err)
	}
	a := selectionFixture(t, "history", json.RawMessage("null"), false)
	if err := store.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(a); e != nil {
			return e
		}
		next := tx.Snapshot()
		prepared, _ := tx.Attempt(a.OperationID)
		for i := 0; i < 2000; i++ {
			prepared.PriorAuthorizations = append(prepared.PriorAuthorizations, decisionFixture(t, store, prepared, false))
		}
		prepared.Authorization = decisionFixture(t, store, prepared, false)
		next.Operations[a.OperationID] = prepared
		return tx.save(next, nil)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.WithRead(func(tx *Transaction) error {
		a, ok := tx.Attempt("history")
		if !ok || len(a.PriorAuthorizations) != 2000 || a.Status != "prepared" || string(tx.Current()) != "null" {
			t.Fatal("history fixture was not acknowledged intact")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

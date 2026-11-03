package shvstate

import (
	"github.com/QuanuX/Symphony/tools/qxctl/internal/retainedtest"
	"testing"
)

func TestSHVSourceQuiescentRestorePreservesHistoryAndPendingIntent(t *testing.T) {
	s, first := fixture(t)
	commitFixture(t, s, first)
	var pending Intent
	if err := s.WithLock(func(tx *Transaction) error {
		pending = intentFor(t, "interrupted-before-authority", tx.Current(), "Vendor B")
		_, err := tx.Prepare(pending)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	original := retainedtest.Inventory(t, s.Root)
	root := retainedtest.Clone(t, s.Root)
	restored, err := New(root, s.TOPSID, s.SourceID)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.WithLock(func(tx *Transaction) error {
		if len(tx.History()) != 1 {
			t.Fatal("restored committed history missing")
		}
		old, ok := tx.Attempt(first.OperationID)
		if !ok || old.Status != "committed" {
			t.Fatal("original commit missing")
		}
		attempt, ok := tx.Attempt(pending.OperationID)
		if !ok || attempt.Status != "prepared" {
			t.Fatal("pending intent missing")
		}
		// Restore is not permission. The original prepared operation requires a fresh guard.
		if err := tx.Commit(pending.OperationID, nil, nil); err == nil {
			t.Fatal("restore fabricated authority")
		}
		return tx.Commit(pending.OperationID, decisionFixture(t, restored, attempt), func() error { return nil })
	}); err != nil {
		t.Fatal(err)
	}
	if err := restored.WithLock(func(tx *Transaction) error {
		if len(tx.History()) != 2 {
			t.Fatal("restored recovery overwrote historical revision")
		}
		return tx.Commit(first.OperationID, nil, nil)
	}); err != nil {
		t.Fatal(err)
	}
	retainedtest.Equal(t, original, retainedtest.Inventory(t, s.Root))
}

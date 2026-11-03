package snvstate

import (
	"github.com/QuanuX/Symphony/tools/qxctl/internal/retainedtest"
	"testing"
)

func TestSNVQuiescentRestorePreservesEvidenceAndPendingIntent(t *testing.T) {
	s, err := NewEvidence(testRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	first, pending := evidenceFixture(t, "acknowledged"), evidenceFixture(t, "interrupted")
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(first); err != nil {
			return err
		}
		if err := tx.CommitEvidence(first.OperationID, func() error { return nil }); err != nil {
			return err
		}
		_, err := tx.Prepare(pending)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	original := retainedtest.Inventory(t, s.Root)
	root := retainedtest.Clone(t, s.Root)
	restored, err := NewEvidence(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.WithRead(func(tx *Transaction) error {
		if len(tx.Snapshot().Bundles) != 1 || len(tx.Snapshot().Operations) != 2 || string(tx.Current()) != "null" {
			t.Fatal("restore changed evidence or selected a view")
		}
		a, ok := tx.Attempt(pending.OperationID)
		if !ok || a.Status != "prepared" || !same(a.Input, pending.Input) {
			t.Fatal("original pending input missing")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	retainedtest.Equal(t, original, retainedtest.Inventory(t, root))
	if err := restored.WithLock(func(tx *Transaction) error {
		return tx.CommitEvidence(pending.OperationID, func() error { return nil })
	}); err != nil {
		t.Fatal(err)
	}
	if err := restored.WithRead(func(tx *Transaction) error {
		a, ok := tx.Attempt(first.OperationID)
		if !ok || a.Status != "committed" || !same(a.Input, first.Input) {
			t.Fatal("recovery rewrote acknowledged input")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	retainedtest.Equal(t, original, retainedtest.Inventory(t, s.Root))
}

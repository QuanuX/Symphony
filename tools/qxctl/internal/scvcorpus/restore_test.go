package scvcorpus

import (
	"bytes"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/retainedtest"
	"testing"
)

func TestCorpusQuiescentRestorePreservesExactArtifacts(t *testing.T) {
	s := testStore(t)
	raw := testArtifact(t, "synthetic documentary evidence")
	digest, err := objectDigest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.With("", func(tx *Session, _ *Job) error { _, err := tx.Put("captures", raw); return err }); err != nil {
		t.Fatal(err)
	}
	original := retainedtest.Inventory(t, s.Root)
	root := retainedtest.Clone(t, s.Root)
	restored, err := New(root, s.TOPSID, s.Domain)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.With("", func(tx *Session, _ *Job) error {
		actual, err := tx.Get("captures", digest)
		if err != nil {
			return err
		}
		if !bytes.Equal(actual, raw) {
			t.Fatal("restored artifact bytes differ")
		}
		_, err = tx.Put("captures", raw)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	retainedtest.Equal(t, original, retainedtest.Inventory(t, root))
	retainedtest.Equal(t, original, retainedtest.Inventory(t, s.Root))
}

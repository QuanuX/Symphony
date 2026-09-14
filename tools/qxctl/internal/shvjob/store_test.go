package shvjob

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"testing"
)

func sample(n int) json.RawMessage {
	m := map[string]any{"n": n}
	d, _ := knowledgeengine.SCVDigest(m)
	m["digest"] = d
	r, _ := knowledgeengine.SCVCanonical(m)
	return r
}
func TestSHVJobCheckpointAtomicity(t *testing.T) {
	s, e := New(resolvedTemp(t), "job")
	if e != nil {
		t.Fatal(e)
	}
	e = s.WithLock(func(tx *Transaction) error {
		if tx.Read() != nil {
			t.Fatal("unexpected state")
		}
		if e := tx.Save(sample(1), nil); e != nil {
			return e
		}
		if e := tx.Save(sample(2), func() error { return fmt.Errorf("stop before rename") }); e == nil {
			t.Fatal("guard ignored")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	e = s.WithLock(func(tx *Transaction) error {
		if string(tx.Read()) != string(sample(1)) {
			t.Fatal("failed publication changed checkpoint")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestSHVJobFilesystemBoundaries(t *testing.T) {
	root := resolvedTemp(t)
	s, _ := New(root, "job")
	if e := s.WithLock(func(tx *Transaction) error { return tx.Save(sample(1), nil) }); e != nil {
		t.Fatal(e)
	}
	id, _ := knowledgeengineDigestIdentity("job")
	state := filepath.Join(root, "symphony/qxctl/shv/materializations-v1", id, "state.json")
	if e := os.Link(state, state+".link"); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("hardlink accepted")
	}
	_ = os.Remove(state + ".link")
	original, _ := os.ReadFile(state)
	_ = os.Remove(state)
	target := filepath.Join(root, "target")
	_ = os.WriteFile(target, original, 0600)
	_ = os.Symlink(target, state)
	if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("symlink accepted")
	}
}
func TestSHVJobLockAndIdentity(t *testing.T) {
	if _, e := New("relative", "job"); e == nil {
		t.Fatal("relative root")
	}
	if _, e := New(resolvedTemp(t), "../job"); e == nil {
		t.Fatal("unsafe ID")
	}
	s, _ := New(resolvedTemp(t), "job")
	e := s.WithLock(func(*Transaction) error {
		if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
			t.Fatal("competing lock")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}

func resolvedTemp(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

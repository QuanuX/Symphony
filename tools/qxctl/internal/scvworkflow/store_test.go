package scvworkflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const tops = "11111111-2222-3333-4444-555555555555"

func fixture(t *testing.T) Store {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(root, tops)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func artifact(t *testing.T, label string) []byte {
	t.Helper()
	native, err := Seal(map[string]any{"protocol": "symphony.scv.knowledge.v1", "fixture": label})
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRecord("knowledge_interpret", knowledgeengine.Installation{Role: "scv", Version: "0.3.0-dev"}, json.RawMessage(`{}`), native)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Canonical(r)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func objectPath(s Store, kind, digest string) string {
	return filepath.Join(s.Root, "symphony", "qxctl", "scv", "workflows-v1", s.TOPSID, kind, strings.TrimPrefix(digest, "sha256:")+".json")
}
func TestWorkflowImmutablePublicationPaginationAndConcurrentReuse(t *testing.T) {
	s := fixture(t)
	raw := artifact(t, "one")
	digest, _ := Digest(raw)
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			failures <- s.With("", true, func(session *Session, _ *Run) error {
				actual, err := session.Put("records", raw)
				if err != nil {
					return err
				}
				if actual != digest {
					return fmt.Errorf("changed identity")
				}
				return nil
			})
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	before, _ := os.Stat(objectPath(s, "records", digest))
	err := s.With("", true, func(session *Session, _ *Run) error {
		_, err := session.Put("records", raw)
		if err != nil {
			return err
		}
		for _, label := range []string{"two", "three"} {
			if _, err = session.Put("records", artifact(t, label)); err != nil {
				return err
			}
		}
		page, next, err := session.List("", 2)
		if err != nil {
			return err
		}
		if len(page) != 2 || next == nil || page[0].Digest >= page[1].Digest || page[0].Artifact != nil {
			t.Fatal("invalid bounded metadata page")
		}
		rest, tail, err := session.List(*next, 2)
		if err != nil {
			return err
		}
		if len(rest) != 1 || tail != nil || rest[0].Digest <= *next {
			t.Fatal("invalid continuation")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(objectPath(s, "records", digest))
	if !os.SameFile(before, after) {
		t.Fatal("idempotent publication replaced inode")
	}
	if err = os.WriteFile(objectPath(s, "records", digest), artifact(t, "tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = s.With("", true, func(session *Session, _ *Run) error { _, err := session.Put("records", raw); return err }); err == nil {
		t.Fatal("tampered immutable record repaired silently")
	}
}
func TestWorkflowStorageRejectsAliasesCorruptionAndReadCreation(t *testing.T) {
	s := fixture(t)
	missing := s
	missing.Root = filepath.Join(s.Root, "missing")
	if err := missing.With("", false, func(*Session, *Run) error { return nil }); err == nil {
		t.Fatal("missing store accepted")
	}
	if _, err := os.Stat(missing.Root); !os.IsNotExist(err) {
		t.Fatal("read created namespace")
	}
	raw := artifact(t, "one")
	digest, _ := Digest(raw)
	if err := s.With("", true, func(session *Session, _ *Run) error { _, err := session.Put("records", raw); return err }); err != nil {
		t.Fatal(err)
	}
	path := objectPath(s, "records", digest)
	target := filepath.Join(s.Root, "outside")
	if err := os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := s.With("", false, func(session *Session, _ *Run) error { _, err := session.Get("records", digest); return err }); err == nil {
		t.Fatal("symlink record accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(target, path); err != nil {
		t.Fatal(err)
	}
	if err := s.With("", false, func(session *Session, _ *Run) error { _, err := session.Get("records", digest); return err }); err == nil {
		t.Fatal("hardlinked record accepted")
	}
	if _, err := ReadRecord(bytes.Replace(raw, []byte(`"kind":"knowledge"`), []byte(`"kind":"knowledge","kind":"knowledge"`), 1)); err == nil {
		t.Fatal("duplicate JSON field accepted")
	}
	alias := filepath.Join(s.Root, "alias")
	if err := os.Symlink(s.Root, alias); err != nil {
		t.Fatal(err)
	}
	s.Root = alias
	if err := s.With("", false, func(*Session, *Run) error { return nil }); err == nil {
		t.Fatal("aliased root accepted")
	}
}
func TestWorkflowJournalPinsPendingInputsAndCommittedResults(t *testing.T) {
	s := fixture(t)
	request := json.RawMessage(`{"operation_id":"one","prior_evaluation_ref":null}`)
	run, err := NewRun("one", knowledgeengine.Installation{Role: "scv", Version: "0.3.0-dev"}, nil, "", request)
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	other := "sha256:" + strings.Repeat("b", 64)
	err = s.With("one", true, func(session *Session, _ *Run) error {
		saved, err := session.Save(run)
		if err != nil {
			return err
		}
		saved.Stages["evaluate"] = Checkpoint{InputRef: digest}
		saved, err = session.Save(saved)
		if err != nil {
			return err
		}
		changed := saved
		changed.Stages = map[string]Checkpoint{"evaluate": {InputRef: other}}
		if _, err = session.Save(changed); err == nil {
			t.Fatal("pending input replaced")
		}
		saved.Stages["evaluate"] = Checkpoint{InputRef: digest, ResultRef: digest}
		saved.Complete = true
		saved, err = session.Save(saved)
		if err != nil {
			return err
		}
		saved.Stages["evaluate"] = Checkpoint{InputRef: digest, ResultRef: other}
		if _, err = session.Save(saved); err == nil {
			t.Fatal("completed result replaced")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.With("one", false, func(_ *Session, current *Run) error {
		if current == nil || !current.Complete || current.Stages["evaluate"].ResultRef != digest {
			t.Fatal("checkpoint not durable")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func TestWorkflowRunLockIsBoundedAndRecoversAfterOwnerExit(t *testing.T) {
	s := fixture(t)
	busy := errors.New("not checked")
	err := s.With("one", true, func(*Session, *Run) error {
		busy = s.With("one", true, func(*Session, *Run) error { t.Fatal("concurrent writer acquired lock"); return nil })
		return errors.New("simulated owner exit")
	})
	if err == nil || busy == nil || !strings.Contains(busy.Error(), "busy") {
		t.Fatal("lock was not bounded")
	}
	if err = s.With("one", true, func(*Session, *Run) error { return nil }); err != nil {
		t.Fatal("lock survived exited owner", err)
	}
}

func TestWorkflowJournalRequiresRequestedStageOrdering(t *testing.T) {
	inst := knowledgeengine.Installation{Role: "scv", Version: "0.4.0-dev"}
	run, err := NewRun("one", inst, &inst, "/private/tmp/retained-corpus", json.RawMessage(`{"operation_id":"one","corpus":{},"prior_evaluation_ref":null}`))
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	run.Stages["evaluate"] = Checkpoint{InputRef: digest, ResultRef: digest}
	run.Complete = true
	raw, err := Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReadRun(raw, "one"); err == nil {
		t.Fatal("completed corpus workflow lacks interpretation")
	}
	run.Stages["interpret"] = Checkpoint{InputRef: digest, ResultRef: digest}
	raw, err = Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReadRun(raw, "one"); err != nil {
		t.Fatal("valid ordered workflow rejected", err)
	}
	run.Stages["reassess"] = Checkpoint{InputRef: digest, ResultRef: digest}
	raw, err = Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReadRun(raw, "one"); err == nil {
		t.Fatal("unrequested reassessment accepted")
	}
}

package scvcorpus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const testTOPS = "11111111-2222-3333-4444-555555555555"

func testStore(t *testing.T) Store {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(root, testTOPS, "scv")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func testArtifact(t *testing.T, text string) []byte {
	t.Helper()
	value := map[string]any{"protocol": "symphony.scv.capture.v1", "body": text}
	value["digest"], _ = knowledgeengine.SCVDigest(value)
	raw, _ := knowledgeengine.SCVCanonical(value)
	return raw
}
func testJob(t *testing.T) Job {
	t.Helper()
	raw := json.RawMessage(`{"operation_id":"job","corpus_id":"docs","previous_snapshot_digest":null,"members":[{"member_id":"one","capture":{}}]}`)
	job, err := NewJob("job", "import", knowledgeengine.Installation{Role: "scv", Version: "0.2.0-dev"}, raw)
	if err != nil {
		t.Fatal(err)
	}
	return job
}
func objectPath(s Store, kind, digest string) string {
	return filepath.Join(s.Root, "symphony", "qxctl", "scv", "corpora-v1", s.TOPSID, s.Domain, kind, strings.TrimPrefix(digest, "sha256:")+".json")
}
func TestCorpusImmutablePublicationAndConcurrentReuse(t *testing.T) {
	s := testStore(t)
	raw := testArtifact(t, "retained exact bytes")
	digest, _ := objectDigest(raw)
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			failures <- s.With("", func(session *Session, _ *Job) error {
				actual, err := session.Put("captures", raw)
				if err != nil {
					return err
				}
				if actual != digest {
					return fmt.Errorf("wrong digest")
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
	file := objectPath(s, "captures", digest)
	before, _ := os.Stat(file)
	if err := s.With("", func(session *Session, _ *Job) error { _, err := session.Put("captures", raw); return err }); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(file)
	if !os.SameFile(before, after) {
		t.Fatal("duplicate write replaced immutable object")
	}
	if err := os.WriteFile(file, testArtifact(t, "tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.With("", func(session *Session, _ *Job) error { _, err := session.Put("captures", raw); return err }); err == nil {
		t.Fatal("tampered object silently replaced")
	}
}
func TestCorpusRejectsUnsafePathsAndStrictJournals(t *testing.T) {
	s := testStore(t)
	raw := testArtifact(t, "evidence")
	digest, _ := objectDigest(raw)
	if err := s.With("", func(session *Session, _ *Job) error { _, err := session.Put("captures", raw); return err }); err != nil {
		t.Fatal(err)
	}
	path := objectPath(s, "captures", digest)
	target := filepath.Join(s.Root, "outside")
	if err := os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := s.With("", func(session *Session, _ *Job) error { _, err := session.Get("captures", digest); return err }); err == nil {
		t.Fatal("symlink blob accepted")
	}
	job := testJob(t)
	_, sealed, err := sealJob(job)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{append(append([]byte{}, sealed...), []byte(` {"extra":true}`)...), bytes.Replace(sealed, []byte(`"mode":"import"`), []byte(`"mode":"import","mode":"import"`), 1)} {
		if _, err := validateJob(bad, "job"); err == nil {
			t.Fatal("ambiguous journal accepted")
		}
	}
	rootLink := filepath.Join(s.Root, "alias")
	if err = os.Symlink(s.Root, rootLink); err != nil {
		t.Fatal(err)
	}
	s.Root = rootLink
	if err = s.With("", func(*Session, *Job) error { return nil }); err == nil {
		t.Fatal("symlink root accepted")
	}
}
func TestCorpusJournalPreservesIntentAndCompletedCheckpoints(t *testing.T) {
	s := testStore(t)
	job := testJob(t)
	digest := "sha256:" + strings.Repeat("a", 64)
	err := s.With("job", func(session *Session, previous *Job) error {
		if previous != nil {
			t.Fatal("unexpected existing job")
		}
		saved, err := session.Save(job)
		if err != nil {
			return err
		}
		saved.Completed = map[string]Checkpoint{"one": {CaptureDigest: digest, IndexDigest: digest}}
		saved, err = session.Save(saved)
		if err != nil {
			return err
		}
		changed := saved
		changed.Completed = map[string]Checkpoint{}
		if _, err = session.Save(changed); err == nil {
			t.Fatal("completed checkpoint removed")
		}
		other, err := NewJob("job", "acquire", saved.Installation, saved.Input)
		if err != nil {
			return err
		}
		if _, err = session.Save(other); err == nil {
			t.Fatal("pinned intent replaced")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.With("job", func(_ *Session, job *Job) error {
		if job == nil || len(job.Completed) != 1 {
			t.Fatal("checkpoint was not retained")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusReadDoesNotCreateMissingStore(t *testing.T) {
	s := testStore(t)
	s.Root = filepath.Join(s.Root, "absent")
	if err := s.WithRead(func(*Session, *Job) error { return nil }); err == nil {
		t.Fatal("missing evidence store treated as present")
	}
	if _, err := os.Stat(s.Root); !os.IsNotExist(err) {
		t.Fatal("read created corpus root")
	}
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvtransport"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func corpusFixture(t *testing.T) (*corpusRunner, scvcorpus.Store) {
	t.Helper()
	prefix := os.Getenv("SYMPHONY_SCV_CORPUS_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.2.0-dev SCV engine")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newCorpusRunner(scvOptions{domain: "scv", prefix: prefix, version: "0.2.0-dev", topsID: ssiagTestTOPSID, repository: root})
	if err != nil {
		t.Fatal(err)
	}
	runner.now = func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }
	store, err := scvcorpus.New(root, ssiagTestTOPSID, "scv")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "absent-config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(root, "absent-runtime"))
	t.Setenv("SYMPHONY_SSIAG_SOCKET", "")
	return runner, store
}
func corpusSource(t *testing.T, r *corpusRunner, id string) map[string]any {
	t.Helper()
	desired := map[string]any{"source_id": id, "provider_id": "aws", "family_id": "schv", "publisher": "AWS", "authority_role": "documentation", "scope": "explicit fixture docs", "locators": []any{map[string]any{"locator_id": "docs", "uri": "https://docs.aws.amazon.com/" + id, "role": "primary", "format": "html", "selector": ""}}, "continuity_evidence": []string{}}
	raw, err := r.owner("source_plan", map[string]any{"operation_id": "plan-" + id, "current": nil, "desired": desired, "reason": "fixture user selection"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := scvcorpus.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	return plan["source"].(map[string]any)
}
func fixtureAcquisition(source json.RawMessage, body, status string) scvtransport.CaptureInput {
	value, _ := scvcorpus.Decode(source)
	locator := value["locators"].([]any)[0].(map[string]any)
	issues := []string{}
	if status != "complete" {
		issues = []string{"fixture_failed_attempt"}
	}
	return scvtransport.CaptureInput{Source: source, LocatorID: "docs", ResolvedURI: locator["uri"].(string), Redirects: []string{}, ObservedAt: "2026-09-10T11:00:00Z", UpstreamRevision: nil, MediaType: "text/plain", Body: body, Completeness: status, Issues: issues}
}
func corpusCapture(t *testing.T, r *corpusRunner, source map[string]any, body, status string) map[string]any {
	t.Helper()
	raw, err := r.owner("capture_import", fixtureAcquisition(mustCorpusRaw(source), body, status))
	if err != nil {
		t.Fatal(err)
	}
	value, err := scvcorpus.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func corpusRequest(operation, corpus string, previous any, members ...any) map[string]any {
	if members == nil {
		members = []any{}
	}
	return map[string]any{"operation_id": operation, "corpus_id": corpus, "previous_snapshot_digest": previous, "members": members}
}
func executeCorpus(t *testing.T, r *corpusRunner, s scvcorpus.Store, op string, input map[string]any) map[string]any {
	t.Helper()
	raw, err := r.execute(s, op, input)
	if err != nil {
		t.Fatal(err)
	}
	value, err := scvcorpus.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestInstalledSCVCorpusRecoveryPinsSourcesAndRetainsFailedAttempts(t *testing.T) {
	r, s := corpusFixture(t)
	a := corpusSource(t, r, "a")
	b := corpusSource(t, r, "b")
	request := corpusRequest("initial", "docs", nil, map[string]any{"member_id": "a", "source": a, "locator_id": "docs"}, map[string]any{"member_id": "b", "source": b, "locator_id": "docs"})
	calls := map[string]int{}
	interrupted := false
	r.acquire = func(_ context.Context, in scvtransport.Input) (scvtransport.CaptureInput, error) {
		source, _ := scvcorpus.Decode(in.Source)
		id := source["source_id"].(string)
		calls[id]++
		if id == "b" && !interrupted {
			interrupted = true
			return scvtransport.CaptureInput{}, errors.New("injected interruption before completed member")
		}
		return fixtureAcquisition(in.Source, "fixture body "+id, "complete"), nil
	}
	if _, err := r.execute(s, "acquire", request); err == nil {
		t.Fatal("injected interruption ignored")
	}
	a["locators"].([]any)[0].(map[string]any)["uri"] = "https://docs.aws.amazon.com/relocated"
	first := executeCorpus(t, r, s, "recover", map[string]any{"operation_id": "initial"})
	if calls["a"] != 1 || calls["b"] != 2 {
		t.Fatalf("recovery re-fetched completed member: %v", calls)
	}
	snapshot := first["snapshot"].(map[string]any)
	members := snapshot["members"].([]any)
	index := members[0].(map[string]any)["latest_attempt"].(map[string]any)
	if index["requested_uri"] != "https://docs.aws.amazon.com/a" {
		t.Fatal("recovery changed pinned source")
	}
	replay := executeCorpus(t, r, s, "recover", map[string]any{"operation_id": "initial"})
	if replay["snapshot_digest"] != first["snapshot_digest"] || calls["a"] != 1 || calls["b"] != 2 {
		t.Fatal("completed replay fetched or changed result")
	}
	a = corpusSource(t, r, "a")
	r.acquire = func(_ context.Context, in scvtransport.Input) (scvtransport.CaptureInput, error) {
		return fixtureAcquisition(in.Source, "", "failed"), nil
	}
	second := executeCorpus(t, r, s, "acquire", corpusRequest("failed-refresh", "docs", first["snapshot_digest"], map[string]any{"member_id": "a", "source": a, "locator_id": "docs"}))
	latest := second["snapshot"].(map[string]any)["members"].([]any)[0].(map[string]any)
	if latest["latest_attempt"].(map[string]any)["completeness"] != "failed" || !scvcorpus.Same(latest["last_complete"], index) {
		t.Fatal("failed refresh replaced prior dated complete evidence")
	}
	diff := executeCorpus(t, r, s, "diff", map[string]any{"before_snapshot_digest": first["snapshot_digest"], "after_snapshot_digest": second["snapshot_digest"]})
	if !scvcorpus.Same(diff["removed_member_ids"], []string{"b"}) {
		t.Fatal("explicit membership removal not represented")
	}
	export := executeCorpus(t, r, s, "export", map[string]any{"snapshot_digest": second["snapshot_digest"], "member_ids": []any{}, "selection": "last_complete", "query_time": "2026-09-10T12:00:00Z", "max_age_seconds": nil})
	if len(export["captures"].([]any)) != 1 {
		t.Fatal("retained complete bytes missing")
	}
	for _, namespace := range []string{"sources-v1", "graphs-v1"} {
		if _, err := os.Stat(filepath.Join(s.Root, "symphony", "qxctl", "scv", namespace)); !os.IsNotExist(err) {
			t.Fatalf("corpus wrote protected %s namespace", namespace)
		}
	}
}

func corpusJobPath(s scvcorpus.Store, operation string) string {
	digest, _ := knowledgeengine.SCVDigest(map[string]any{"operation_id": operation})
	return filepath.Join(s.Root, "symphony", "qxctl", "scv", "corpora-v1", s.TOPSID, s.Domain, "jobs", strings.TrimPrefix(digest, "sha256:"), "job.json")
}
func rewriteCorpusJournal(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	value, err := scvcorpus.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	mutate(value)
	delete(value, "digest")
	value["digest"], _ = knowledgeengine.SCVDigest(value)
	if err = os.WriteFile(path, mustCorpusRaw(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
func TestInstalledSCVCorpusRejectsTamperedCaptureAndWrongCompletedSnapshot(t *testing.T) {
	r, s := corpusFixture(t)
	source := corpusSource(t, r, "docs")
	capture := corpusCapture(t, r, source, "original", "complete")
	first := executeCorpus(t, r, s, "import", corpusRequest("one", "one", nil, map[string]any{"member_id": "docs", "capture": capture}))
	second := executeCorpus(t, r, s, "import", corpusRequest("two", "two", nil, map[string]any{"member_id": "docs", "capture": capture}))
	path := corpusJobPath(s, "one")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rewriteCorpusJournal(t, path, func(value map[string]any) { value["snapshot_digest"] = second["snapshot_digest"] })
	if _, err = r.execute(s, "recover", map[string]any{"operation_id": "one"}); err == nil {
		t.Fatal("completed job accepted unrelated valid snapshot")
	}
	if err = os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	rewriteCorpusJournal(t, path, func(value map[string]any) { value["snapshot_time"] = "2026-09-10T12:00:01Z" })
	if _, err = r.execute(s, "recover", map[string]any{"operation_id": "one"}); err == nil {
		t.Fatal("completed job accepted altered snapshot time")
	}
	capturePath := filepath.Join(s.Root, "symphony", "qxctl", "scv", "corpora-v1", s.TOPSID, s.Domain, "captures", strings.TrimPrefix(capture["digest"].(string), "sha256:")+".json")
	if err = os.WriteFile(capturePath, []byte(`{"tampered":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = r.execute(s, "inspect", map[string]any{"snapshot_digest": first["snapshot_digest"]}); err == nil || !strings.Contains(err.Error(), capture["digest"].(string)) {
		t.Fatalf("missing exact tampered-capture rejection: %v", err)
	}
}

func TestInstalledSCVCorpusClockRollbackRemainsRecoverable(t *testing.T) {
	r, s := corpusFixture(t)
	source := corpusSource(t, r, "clock")
	request := corpusRequest("clock", "docs", nil, map[string]any{"member_id": "clock", "source": source, "locator_id": "docs"})
	calls := 0
	r.acquire = func(_ context.Context, in scvtransport.Input) (scvtransport.CaptureInput, error) {
		calls++
		r.now = func() time.Time { return time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC) }
		return fixtureAcquisition(in.Source, "body", "complete"), nil
	}
	if _, err := r.execute(s, "acquire", request); err == nil {
		t.Fatal("clock rollback finalized snapshot")
	}
	if err := s.With("clock", func(_ *scvcorpus.Session, job *scvcorpus.Job) error {
		if len(job.Completed) != 1 || job.SnapshotTime != "" || job.SnapshotDigest != "" {
			t.Fatal("clock rollback lost checkpoint or froze impossible time")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	r.now = func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }
	executeCorpus(t, r, s, "recover", map[string]any{"operation_id": "clock"})
	if calls != 1 {
		t.Fatal("clock recovery re-fetched completed capture")
	}
	capture := corpusCapture(t, r, source, "body", "complete")
	r.now = func() time.Time { return time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC) }
	if _, err := r.execute(s, "import", corpusRequest("future", "docs", nil, map[string]any{"member_id": "clock", "capture": capture})); err == nil {
		t.Fatal("future import accepted")
	}
	if err := s.With("future", func(_ *scvcorpus.Session, job *scvcorpus.Job) error {
		if job != nil {
			t.Fatal("future import persisted intent")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledSCVCorpusExportBounds(t *testing.T) {
	r, s := corpusFixture(t)
	members := []any{}
	for i := 0; i < 17; i++ {
		id := fmt.Sprintf("m%02d", i)
		members = append(members, map[string]any{"member_id": id, "source": corpusSource(t, r, id), "locator_id": "docs"})
	}
	r.acquire = func(_ context.Context, in scvtransport.Input) (scvtransport.CaptureInput, error) {
		return fixtureAcquisition(in.Source, strings.Repeat("x", 65536), "complete"), nil
	}
	result := executeCorpus(t, r, s, "acquire", corpusRequest("big", "docs", nil, members...))
	request := map[string]any{"snapshot_digest": result["snapshot_digest"], "member_ids": []any{}, "selection": "latest_attempt", "query_time": "2026-09-10T12:00:00Z", "max_age_seconds": nil}
	if _, err := r.execute(s, "export", request); err == nil || !strings.Contains(err.Error(), "16 captures") {
		t.Fatalf("17 capture bound not enforced: %v", err)
	}
	ids := []any{}
	for i := 0; i < 16; i++ {
		ids = append(ids, fmt.Sprintf("m%02d", i))
	}
	request["member_ids"] = ids
	if _, err := r.execute(s, "export", request); err == nil || !strings.Contains(err.Error(), "process bounds") {
		t.Fatalf("serialized16-capture input bound not enforced: %v", err)
	}
	request["member_ids"] = []any{"m00"}
	export := executeCorpus(t, r, s, "export", request)
	if len(export["captures"].([]any)) != 1 {
		t.Fatal("bounded explicit export failed")
	}
}

func TestInstalledSCVCorpusRejectsUnverifiableSuccessorBeforePublication(t *testing.T) {
	r, s := corpusFixture(t)
	var previous json.RawMessage
	var digest string
	for i := 0; i < corpusMaxAncestry; i++ {
		var ancestor any
		if previous != nil {
			ancestor = previous
		}
		raw, err := r.owner("corpus_build", map[string]any{"corpus_id": "depth", "previous": ancestor, "snapshot_time": "2026-09-10T11:00:00Z", "attempts": []any{}})
		if err != nil {
			t.Fatal(err)
		}
		err = s.With("", func(session *scvcorpus.Session, _ *scvcorpus.Job) error {
			var e error
			digest, e = session.Put("snapshots", raw)
			return e
		})
		if err != nil {
			t.Fatal(err)
		}
		previous = raw
	}
	if _, err := r.execute(s, "import", corpusRequest("too-deep", "depth", digest)); err == nil || !strings.Contains(err.Error(), "ancestry") {
		t.Fatalf("unverifiable successor accepted: %v", err)
	}
	verifier := r.verifier(nil)
	verifier.snapshots[digest] = previous
	if _, err := verifier.snapshot(digest, 1); err == nil {
		t.Fatal("cached ancestor bypassed successor depth budget")
	}
	if err := s.With("too-deep", func(_ *scvcorpus.Session, job *scvcorpus.Job) error {
		if job != nil {
			t.Fatal("impossible successor prepared")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(s.Root, "symphony", "qxctl", "scv", "corpora-v1", s.TOPSID, s.Domain, "snapshots")
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 128 {
		t.Fatalf("unverifiable snapshot published: %d %v", len(entries), err)
	}
}

func TestInstalledSCVCorpusDeadlineRetainsFailureCheckpoint(t *testing.T) {
	r, s := corpusFixture(t)
	a, b := corpusSource(t, r, "a"), corpusSource(t, r, "b")
	r.budget = time.Millisecond
	calls := 0
	r.acquire = func(ctx context.Context, in scvtransport.Input) (scvtransport.CaptureInput, error) {
		calls++
		<-ctx.Done()
		failed := fixtureAcquisition(in.Source, "", "failed")
		failed.Issues = []string{"https_retrieval_deadline"}
		return failed, nil
	}
	request := corpusRequest("deadline", "docs", nil, map[string]any{"member_id": "a", "source": a, "locator_id": "docs"}, map[string]any{"member_id": "b", "source": b, "locator_id": "docs"})
	if _, err := r.execute(s, "acquire", request); err == nil || !strings.Contains(err.Error(), "budget exhausted") {
		t.Fatalf("missing recoverable scheduling budget result: %v", err)
	}
	if err := s.With("deadline", func(_ *scvcorpus.Session, job *scvcorpus.Job) error {
		if len(job.Completed) != 1 || job.Completed["a"].CaptureDigest == "" || job.SnapshotDigest != "" {
			t.Fatal("deadline failed attempt was not checkpointed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	r.budget = time.Second * 10
	r.acquire = func(_ context.Context, in scvtransport.Input) (scvtransport.CaptureInput, error) {
		calls++
		return fixtureAcquisition(in.Source, "complete", "complete"), nil
	}
	result := executeCorpus(t, r, s, "recover", map[string]any{"operation_id": "deadline"})
	if calls != 2 {
		t.Fatal("deadline recovery re-fetched completed failed attempt")
	}
	coverage := result["snapshot"].(map[string]any)["coverage"].(map[string]any)
	if coverage["failed_attempts"] != json.Number("1") || coverage["complete_attempts"] != json.Number("1") {
		t.Fatal("deadline outcome omitted from snapshot")
	}
}

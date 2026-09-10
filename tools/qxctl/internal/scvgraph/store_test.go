package scvgraph

import (
	"encoding/json"
	"errors"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const tops = "00000000-0000-4000-8000-000000000001"

func fixture(t *testing.T) (Store, knowledgeengine.Installation) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(root, tops, "scv", "research")
	if err != nil {
		t.Fatal(err)
	}
	return store, knowledgeengine.Installation{Role: "scv", ModuleID: "scv-engine", EngineID: "symphony-scv", Version: "0.1.0-dev"}
}
func graph(t *testing.T, key string) json.RawMessage {
	t.Helper()
	value := map[string]any{"protocol": "symphony.scv.graph.v1", "domain": "scv", "fixture": key}
	digest, err := selfDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	value["digest"] = digest
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func intent(t *testing.T, id string, expected *string, generation int, body json.RawMessage, i knowledgeengine.Installation) Intent {
	t.Helper()
	v, e := NewIntent(id, expected, generation, body, i)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func authority(Intent) (Authorization, error) {
	return Authorization{Evidence: json.RawMessage(`{"test_authorizer":"storage-unit-fixture-not-SSIAG-evidence"}`), ValidUntil: time.Now().Add(time.Minute)}, nil
}

func TestGraphPublicationCASAndLostResponse(t *testing.T) {
	s, i := fixture(t)
	first := intent(t, "one", nil, 0, graph(t, "one"), i)
	d, err := s.Select(first, authority)
	if err != nil {
		t.Fatal(err)
	}
	if d.Generation != 1 || *d.GraphDigest != first.GraphDigest {
		t.Fatal("wrong initial head")
	}
	denied := func(Intent) (Authorization, error) {
		t.Fatal("committed retry must not request new authority")
		return Authorization{}, errors.New("unexpected")
	}
	again, err := s.Select(first, denied)
	if err != nil || again.Generation != 1 {
		t.Fatal("lost response duplicated graph selection", err)
	}
	stale := intent(t, "stale", nil, 0, graph(t, "two"), i)
	if _, err = s.Select(stale, authority); err == nil {
		t.Fatal("stale graph head accepted")
	}
	staleABA := intent(t, "pre-rollback", d.GraphDigest, d.Generation, graph(t, "three"), i)
	second := intent(t, "two", d.GraphDigest, d.Generation, graph(t, "two"), i)
	d, err = s.Select(second, authority)
	if err != nil || d.Generation != 2 {
		t.Fatal(err)
	}
	rollback := intent(t, "rollback", d.GraphDigest, d.Generation, first.Graph, i)
	d, err = s.Select(rollback, authority)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Select(staleABA, authority); err == nil {
		t.Fatal("pre-rollback intent accepted after A to B to A")
	}
	if d.Generation != 3 || *d.GraphDigest != first.GraphDigest || len(d.Operations) != 3 {
		t.Fatal("rollback erased evidence")
	}
}
func TestInterruptedGraphPublicationKeepsOldCoherentHead(t *testing.T) {
	s, i := fixture(t)
	first := intent(t, "one", nil, 0, graph(t, "one"), i)
	initial, err := s.Select(first, authority)
	if err != nil {
		t.Fatal(err)
	}
	next := intent(t, "two", initial.GraphDigest, initial.Generation, graph(t, "two"), i)
	failure := func(Intent) (Authorization, error) {
		return Authorization{}, errors.New("fixture authority unavailable")
	}
	if _, err = s.Select(next, failure); err == nil {
		t.Fatal("authorization failure passed")
	}
	d, err := s.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if *d.GraphDigest != first.GraphDigest || d.Operations["two"].Status != "prepared" {
		t.Fatal("interruption changed visible graph")
	}
	d, err = s.Select(next, authority)
	if err != nil || *d.GraphDigest != next.GraphDigest {
		t.Fatal("recovery failed", err)
	}
	collision := intent(t, "two", d.GraphDigest, d.Generation, graph(t, "three"), i)
	if _, err = s.Select(collision, authority); err == nil {
		t.Fatal("operation ID rebound")
	}
}
func TestGraphStoreRejectsSymlinkAndTamper(t *testing.T) {
	s, i := fixture(t)
	first := intent(t, "one", nil, 0, graph(t, "one"), i)
	if _, err := s.Select(first, authority); err != nil {
		t.Fatal(err)
	}
	var statePath string
	err := filepath.Walk(s.Root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "state.json" {
			statePath = p
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte(`{"protocol":"forged"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Inspect(); err == nil {
		t.Fatal("tampered graph store accepted")
	}
	if err := os.Remove(statePath); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(s.Root, "unrelated")
	if err := os.WriteFile(target, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, statePath); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Inspect(); err == nil {
		t.Fatal("symlink graph state accepted")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "untouched" {
		t.Fatal("symlink target changed")
	}
}
func TestGraphIntentRequiresAuthorizationEvidence(t *testing.T) {
	s, i := fixture(t)
	first := intent(t, "one", nil, 0, graph(t, "one"), i)
	if _, err := s.Select(first, nil); err == nil {
		t.Fatal("missing authorizer accepted")
	}
	d, err := s.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if d.GraphDigest != nil || d.Operations["one"].Status != "prepared" {
		t.Fatal("failed authorization changed graph")
	}
	bad := first
	bad.GraphDigest = "sha256:forged"
	if _, err := s.Select(bad, authority); err == nil {
		t.Fatal("tampered intent accepted")
	}
}

func TestGraphAuthorityExpiryKeepsRecoverableIntent(t *testing.T) {
	s, i := fixture(t)
	first := intent(t, "expires", nil, 0, graph(t, "one"), i)
	expired := func(v Intent) (Authorization, error) {
		a, e := authority(v)
		a.ValidUntil = time.Now().Add(-time.Second)
		return a, e
	}
	if _, err := s.Select(first, expired); err == nil {
		t.Fatal("expired authority published graph")
	}
	state, err := s.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if state.GraphDigest != nil || state.Operations["expires"].Status != "authorized" {
		t.Fatal("expiry did not retain recoverable exact intent")
	}
	if _, err := s.Select(first, authority); err != nil {
		t.Fatal(err)
	}
}
func TestGraphHistoryRejectsResealedHeadAndFork(t *testing.T) {
	s, i := fixture(t)
	first := intent(t, "one", nil, 0, graph(t, "one"), i)
	d, err := s.Select(first, authority)
	if err != nil {
		t.Fatal(err)
	}
	second := intent(t, "two", d.GraphDigest, d.Generation, graph(t, "two"), i)
	d, err = s.Select(second, authority)
	if err != nil {
		t.Fatal(err)
	}
	bad := copyDocument(d)
	bad.HeadOperationID = &first.OperationID
	bad.GraphDigest = &first.GraphDigest
	bad.Digest, _ = selfDigest(bad)
	if s.validate(bad) == nil {
		t.Fatal("resealed old head accepted as terminal revision")
	}
	bad = copyDocument(d)
	fork := bad.Operations["two"]
	fork.Intent.ExpectedGeneration = 0
	fork.Intent.ExpectedGraphDigest = nil
	fork.Intent.Digest, _ = selfDigest(fork.Intent)
	bad.Operations["two"] = fork
	bad.Digest, _ = selfDigest(bad)
	if s.validate(bad) == nil {
		t.Fatal("resealed publication fork accepted")
	}
}

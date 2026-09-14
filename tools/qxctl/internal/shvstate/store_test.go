package shvstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
)

func raw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, e := knowledgeengine.SCVCanonical(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func sealed(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	delete(m, "digest")
	d, e := knowledgeengine.SCVDigest(m)
	if e != nil {
		t.Fatal(e)
	}
	m["digest"] = d
	return raw(t, m)
}
func object(t *testing.T, r json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if e := json.Unmarshal(r, &m); e != nil {
		t.Fatal(e)
	}
	return m
}
func fixture(t *testing.T) (Store, Intent) {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	s, e := New(root, "00000000-0000-4000-8000-000000000001", "family")
	if e != nil {
		t.Fatal(e)
	}
	return s, intentFor(t, "first", nil, "Vendor")
}
func intentFor(t *testing.T, id string, current json.RawMessage, publisher string) Intent {
	t.Helper()
	generation := int64(1)
	var previous any
	if len(current) > 0 && string(current) != "null" {
		var s retainedSource
		if e := json.Unmarshal(current, &s); e != nil {
			t.Fatal(e)
		}
		generation = s.Generation + 1
		previous = s.Digest
	}
	def := map[string]any{"source_id": "family", "publisher": publisher, "authority_role": "declared", "subject_ids": []any{"cpu-a", "cpu-b"}, "locators": []any{map[string]any{"id": "primary", "uri": "https://vendor.example/family", "format": "opaque"}}}
	source := sealed(t, map[string]any{"protocol": "symphony.shv.source-revision.v1", "definition": def, "generation": generation, "previous_digest": previous})
	kind := "onboard"
	if previous != nil {
		kind = "authority_change"
	}
	plan := sealed(t, map[string]any{"protocol": "symphony.shv.source-plan.v1", "operation_id": id, "expected_state_digest": previous, "change_kind": kind, "reason": "Explicit candidate source change", "source": source})
	transition := sealed(t, map[string]any{"protocol": "symphony.shv.source-transition.v1", "operation_id": id, "expected_state_digest": previous, "source": source})
	prefix := "/retained-historical-installation"
	hash := "sha256:" + strings.Repeat("a", 64)
	inst := knowledgeengine.Installation{Role: "shv-source-engine", ModuleID: "shv-source-engine", EngineID: "symphony-shv-source", Version: "0.1.0-dev", Prefix: prefix, ReceiptPath: prefix + "/share/symphony/receipts/shv-source-engine/0.1.0-dev/install-receipt.json", ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: hash, ExecutablePath: prefix + "/libexec/symphony/shv-source-engine/0.1.0-dev/symphony-shv-source", ExecutableDigest: hash}
	i, e := NewIntent(id, plan, transition, inst)
	if e != nil {
		t.Fatal(e)
	}
	return i
}

// Structural test evidence only. This does not exercise authenticated SSIAG or
// establish real permission; command-level authenticated process tests own that boundary.
func decisionFixture(t *testing.T, s Store, a Attempt) json.RawMessage {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(time.Minute)
	id, e := stavprotocol.GenerateUUIDv4()
	if e != nil {
		t.Fatal(e)
	}
	basis := "host_owner"
	hash := "sha256:" + strings.Repeat("b", 64)
	var p retainedPlan
	_ = json.Unmarshal(a.Intent.Plan, &p)
	d := ssiagclient.AuthorizationDecision{Schema: "symphony.ssiag.authorization-decision.v1", DecisionID: "test-decision", RequestID: id, CorrelationID: a.CorrelationID, TOPSID: s.TOPSID, Subject: ssiagclient.DecisionSubject{ID: "test-owner", Kind: "host", Authority: "unix_peer_credentials"}, Target: ssiagclient.DecisionTarget{Operation: map[string]string{"onboard": "symphony.shv.source.onboard", "authority_change": "symphony.shv.source.authority-change", "relocation": "symphony.shv.source.relocation"}[p.ChangeKind], Resource: resource(s.TOPSID, s.SourceID), Audience: "qxctl", Scope: "tops:" + s.TOPSID}, Effect: "allow", ReasonCode: "symphony.ssiag.policy.exact-grant", AuthorityBasis: &basis, PolicyDigest: hash, ConfigDigest: hash, DecidedAt: now, ExpiresAt: &expires}
	c := ssiagclient.Capability{Protocol: "symphony.ssiag.capability.v1", Subject: d.Subject, TOPSID: d.TOPSID, Target: d.Target, AuthorityBasis: basis, GrantID: "test-grant", RequestID: d.RequestID, CorrelationID: d.CorrelationID, IssuedAt: now, ExpiresAt: expires, PolicyDigest: hash, ConfigDigest: hash}
	c.BindingDigest = capabilityBinding(c)
	c.CapabilityID = "ssiag-capability:" + strings.TrimPrefix(c.BindingDigest, "sha256:")
	d.Capability = &c
	return raw(t, d)
}
func commitFixture(t *testing.T, s Store, i Intent) {
	t.Helper()
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(i); e != nil {
			return e
		}
		a, _ := tx.Attempt(i.OperationID)
		return tx.Commit(i.OperationID, decisionFixture(t, s, a), func() error { return nil })
	}); e != nil {
		t.Fatal(e)
	}
}
func TestSHVStateCASReplayAndHistory(t *testing.T) {
	s, i := fixture(t)
	denied := errors.New("test boundary did not grant permission")
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(i); e != nil {
			return e
		}
		return denied
	}); !errors.Is(e, denied) {
		t.Fatal(e)
	}
	var correlation string
	if e := s.WithLock(func(tx *Transaction) error {
		if string(tx.Current()) != "null" || len(tx.History()) != 0 {
			t.Fatal("prepared branch became head")
		}
		a, _ := tx.Attempt(i.OperationID)
		correlation = a.CorrelationID
		if len(correlation) != 36 || correlation[14] != '4' {
			t.Fatal("not random UUIDv4")
		}
		a.Intent.Plan[0] = '!'
		again, _ := tx.Attempt(i.OperationID)
		if again.Intent.Plan[0] == '!' {
			t.Fatal("attempt leaked mutable journal")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	commitFixture(t, s, i)
	var second Intent
	if e := s.WithLock(func(tx *Transaction) error { second = intentFor(t, "second", tx.Current(), "Vendor B"); return nil }); e != nil {
		t.Fatal(e)
	}
	commitFixture(t, s, second)
	if e := s.WithLock(func(tx *Transaction) error {
		head := tx.Current()
		replay, e := tx.Prepare(i)
		if e != nil || !replay {
			t.Fatal(replay, e)
		}
		if e = tx.Commit(i.OperationID, nil, nil); e != nil {
			t.Fatal(e)
		}
		if !sameJSON(head, tx.Current()) || len(tx.History()) != 2 {
			t.Fatal("historical replay moved head")
		}
		a, _ := tx.Attempt(i.OperationID)
		if a.CorrelationID != correlation {
			t.Fatal("correlation changed")
		}
		changed := i
		changed.Plan = raw(t, map[string]any{"forged": true})
		if _, e = tx.Prepare(changed); e == nil {
			t.Fatal("same digest different exact intent replay")
		}
		stale := intentFor(t, "stale", nil, "Other")
		if _, e = tx.Prepare(stale); e == nil {
			t.Fatal("stale CAS")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}
func TestSHVStateInterruptedPublicationAndExpiry(t *testing.T) {
	s, i := fixture(t)
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(i); e != nil {
			return e
		}
		a, _ := tx.Attempt(i.OperationID)
		auth := decisionFixture(t, s, a)
		failure := errors.New("test publication interruption")
		if e := tx.Commit(i.OperationID, auth, func() error { return failure }); !errors.Is(e, failure) {
			t.Fatal(e)
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(tx *Transaction) error {
		a, _ := tx.Attempt(i.OperationID)
		if a.Status != "authorized" || string(tx.Current()) != "null" {
			t.Fatal("interrupted publication changed head")
		}
		auth := decisionFixture(t, s, a)
		var d ssiagclient.AuthorizationDecision
		_ = json.Unmarshal(auth, &d)
		d.DecidedAt = time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Second)
		expiry := d.DecidedAt.Add(time.Minute)
		d.ExpiresAt = &expiry
		d.Capability.IssuedAt = d.DecidedAt
		d.Capability.ExpiresAt = expiry
		d.Capability.BindingDigest = capabilityBinding(*d.Capability)
		d.Capability.CapabilityID = "ssiag-capability:" + strings.TrimPrefix(d.Capability.BindingDigest, "sha256:")
		if e := tx.Commit(i.OperationID, raw(t, d), func() error { return nil }); e == nil {
			t.Fatal("expired evidence published")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	commitFixture(t, s, i)
	if e := s.WithLock(func(tx *Transaction) error {
		a, _ := tx.Attempt(i.OperationID)
		if a.Status != "committed" || len(a.PriorAuthorizations) != 2 {
			t.Fatal("recovery lost authorization evidence")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}
func TestSHVStateForgedNonHeadAndPendingLineage(t *testing.T) {
	s, i := fixture(t)
	commitFixture(t, s, i)
	var snapshot Document
	if e := s.WithLock(func(tx *Transaction) error { snapshot = tx.Snapshot(); return nil }); e != nil {
		t.Fatal(e)
	}
	// A correctly resealed plan/transition with a fabricated generation must fail
	// independent replay even when it is a non-head prepared branch.
	var head retainedTransition
	_ = json.Unmarshal(i.Transition, &head)
	pending := intentFor(t, "pending", head.Source, "Next")
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(pending); e != nil {
			return e
		}
		snapshot = tx.Snapshot()
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	bad := cloneDocument(snapshot)
	a := bad.Operations["pending"]
	p := object(t, a.Intent.Plan)
	r := object(t, a.Intent.Transition)
	source := object(t, head.Source)
	source["generation"] = 3
	source["previous_digest"] = object(t, head.Source)["digest"]
	definition := source["definition"].(map[string]any)
	definition["publisher"] = "Next"
	src := sealed(t, source)
	p["source"] = src
	r["source"] = src
	a.Intent.Plan = sealed(t, p)
	a.Intent.Transition = sealed(t, r)
	a.Intent.Digest, _ = seal(a.Intent)
	bad.Operations["pending"] = a
	bad.Digest, _ = seal(bad)
	if validateDocument(bad, s) == nil {
		t.Fatal("resealed non-head false generation accepted")
	}
	var pr retainedTransition
	_ = json.Unmarshal(pending.Transition, &pr)
	child := intentFor(t, "pending-child", pr.Source, "Third")
	b := cloneDocument(snapshot)
	correlation, _ := stavprotocol.GenerateUUIDv4()
	b.Operations[child.OperationID] = Attempt{Intent: child, Status: "prepared", CorrelationID: correlation, Authorization: json.RawMessage("null"), PriorAuthorizations: []json.RawMessage{}}
	b.Digest, _ = seal(b)
	if validateDocument(b, s) == nil {
		t.Fatal("uncommitted predecessor accepted")
	}
}
func TestSHVStateAuthorizationBinding(t *testing.T) {
	s, i := fixture(t)
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(i); e != nil {
			return e
		}
		a, _ := tx.Attempt(i.OperationID)
		for _, field := range []string{"correlation_id", "tops_id", "effect", "canonical_apply", "target", "capability"} {
			t.Run(field, func(t *testing.T) {
				bad := object(t, decisionFixture(t, s, a))
				switch field {
				case "canonical_apply":
					bad[field] = true
				case "target":
					bad[field].(map[string]any)["resource"] = "wrong"
				case "capability":
					bad[field].(map[string]any)["binding_digest"] = "sha256:" + strings.Repeat("c", 64)
				default:
					bad[field] = "wrong"
				}
				if e := tx.Commit(i.OperationID, raw(t, bad), func() error { return nil }); e == nil {
					t.Fatal("unbound decision accepted")
				}
			})
		}
		if e := tx.Commit(i.OperationID, json.RawMessage(`{"approved":true}`), func() error { return nil }); e == nil {
			t.Fatal("fake approval flag accepted")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}
func statePath(s Store) string {
	hash, _ := knowledgeengineDigestIdentity(s.TOPSID, s.SourceID)
	return filepath.Join(s.Root, "symphony/qxctl/shv/sources-v1", s.TOPSID, hash, "state.json")
}
func TestSHVStatePathAndHistoryBounds(t *testing.T) {
	for _, name := range []string{"symlink", "hardlink", "insecure"} {
		t.Run(name, func(t *testing.T) {
			s, i := fixture(t)
			if e := s.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(i); return e }); e != nil {
				t.Fatal(e)
			}
			path := statePath(s)
			switch name {
			case "symlink":
				if e := os.Rename(path, path+".prior"); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(path+".prior", path); e != nil {
					t.Fatal(e)
				}
			case "hardlink":
				if e := os.Link(path, path+".link"); e != nil {
					t.Fatal(e)
				}
			case "insecure":
				if e := os.Chmod(path, 0644); e != nil {
					t.Fatal(e)
				}
			}
			if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
				t.Fatal("unsafe state path accepted")
			}
		})
	}
	s, i := fixture(t)
	if e := s.WithLock(func(tx *Transaction) error {
		_, e := tx.Prepare(i)
		if e != nil {
			return e
		}
		d := tx.Snapshot()
		for n := 1; n < maxOperations; n++ {
			other := intentFor(t, fmt.Sprintf("pending-%d", n), nil, "Vendor")
			corr, _ := stavprotocol.GenerateUUIDv4()
			d.Operations[other.OperationID] = Attempt{Intent: other, Status: "prepared", CorrelationID: corr, Authorization: json.RawMessage("null"), PriorAuthorizations: []json.RawMessage{}}
		}
		if e = tx.save(d); e != nil {
			return e
		}
		if _, e = tx.Prepare(intentFor(t, "over-limit", nil, "Vendor")); e == nil {
			t.Fatal("attempt bound ignored")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}

func TestSHVStateHistoryEnvelopeBudget(t *testing.T) {
	s, i := fixture(t)
	d := Document{Protocol: protocol, TOPSID: s.TOPSID, SourceID: s.SourceID, Source: json.RawMessage("null"), Operations: map[string]Attempt{}}
	var current json.RawMessage
	var rejected bool
	for n := 1; n <= 20; n++ {
		candidate := intentFor(t, fmt.Sprintf("large-%d", n), current, fmt.Sprintf("Vendor %d", n))
		p := object(t, candidate.Plan)
		tr := object(t, candidate.Transition)
		src := object(t, raw(t, p["source"]))
		def := src["definition"].(map[string]any)
		locators := []any{}
		for k := 0; k < 16; k++ {
			locators = append(locators, map[string]any{"id": fmt.Sprintf("url-%d", k), "uri": fmt.Sprintf("https://example.test/%d/", k) + strings.Repeat("a", 4000), "format": "opaque"})
		}
		def["locators"] = locators
		source := sealed(t, src)
		p["source"] = source
		tr["source"] = source
		candidate.Plan = sealed(t, p)
		candidate.Transition = sealed(t, tr)
		candidate.Digest, _ = seal(candidate)
		tx := &Transaction{store: s, document: d}
		saved := false
		tx.save = func(Document) error { saved = true; return nil }
		_, err := tx.Prepare(candidate)
		prospective := append(tx.History(), source)
		payload := raw(t, map[string]any{"history": prospective})
		if len(payload) > maxHistoryPayloadBytes {
			if err == nil || saved {
				t.Fatal("oversize prospective native history was prepared")
			}
			rejected = true
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		corr, _ := stavprotocol.GenerateUUIDv4()
		a := Attempt{Intent: candidate, Status: "committed", CorrelationID: corr, PriorAuthorizations: []json.RawMessage{}}
		a.Authorization = decisionFixture(t, s, a)
		d.Operations[candidate.OperationID] = a
		current = source
		var parsed retainedSource
		_ = json.Unmarshal(source, &parsed)
		d.Source = source
		d.StateDigest = &parsed.Digest
		head := candidate.OperationID
		d.HeadOperationID = &head
		d.Digest, _ = seal(d)
	}
	_ = i
	if !rejected {
		t.Fatal("large source fixture did not reach bounded history limit")
	}
	if e := validateDocument(d, s); e != nil {
		t.Fatal("last fitting committed history invalid", e)
	}
}
func TestSHVStateStrictDecodeAndLimits(t *testing.T) {
	s, i := fixture(t)
	if decode(append(append([]byte{}, i.Plan...), []byte(` {}`)...), &retainedPlan{}) == nil {
		t.Fatal("trailing JSON accepted")
	}
	bad := i
	bad.Installation.Version = "0.2.0-dev"
	bad.Digest, _ = seal(bad)
	if validateIntentShape(bad) == nil {
		t.Fatal("historical version silently upgraded")
	}
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(i); e != nil {
			return e
		}
		a, _ := tx.Attempt(i.OperationID)
		decision := decisionFixture(t, s, a)
		if _, e := canonicalAuthorization(append(append([]byte{}, decision...), []byte(` {}`)...), s, a); e == nil {
			t.Fatal("trailing authorization JSON accepted")
		}
		a.Status = "authorized"
		a.Authorization = decision
		a.PriorAuthorizations = []json.RawMessage{}
		for n := 0; n < 64; n++ {
			a.PriorAuthorizations = append(a.PriorAuthorizations, decision)
		}
		d := tx.Snapshot()
		d.Operations[i.OperationID] = a
		if e := tx.save(d); e != nil {
			return e
		}
		if e := tx.Commit(i.OperationID, decision, func() error { return nil }); e == nil {
			t.Fatal("prior authorization bound ignored")
		}
		if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
			t.Fatal("competing lock accepted")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(statePath(s), []byte(strings.Repeat(" ", maxStoreBytes+1)), 0600); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("oversized journal accepted")
	}
}

func TestSHVStateRootAndEmptySnapshot(t *testing.T) {
	s, _ := fixture(t)
	if e := s.WithLock(func(tx *Transaction) error {
		if e := validateDocument(tx.Snapshot(), s); e != nil {
			return e
		}
		if len(tx.History()) != 0 {
			return errors.New("empty history is not empty")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	for _, root := range []string{"/", s.Root + "/", s.Root + "/../elsewhere", "relative", s.Root + "\n"} {
		if _, e := New(root, s.TOPSID, s.SourceID); e == nil {
			t.Fatal("invalid root accepted")
		}
	}
	parent, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	link := filepath.Join(parent, "linked")
	if e = os.Symlink(s.Root, link); e != nil {
		t.Fatal(e)
	}
	via, e := New(link, s.TOPSID, s.SourceID)
	if e != nil {
		t.Fatal(e)
	}
	if via.WithLock(func(*Transaction) error { return nil }) == nil {
		t.Fatal("symlink root accepted")
	}
	if e = os.Chmod(s.Root, 0777); e != nil {
		t.Fatal(e)
	}
	if s.WithLock(func(*Transaction) error { return nil }) == nil {
		t.Fatal("untrusted writable root accepted")
	}
}

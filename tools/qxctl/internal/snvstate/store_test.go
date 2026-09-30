package snvstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
)

func testRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, e := knowledgeengine.SCVCanonical(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func testSealed(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	d, e := knowledgeengine.SCVDigest(m)
	if e != nil {
		t.Fatal(e)
	}
	m["digest"] = d
	return testRaw(t, m)
}
func testRoot(t *testing.T) string {
	t.Helper()
	p, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func testInstallation() knowledgeengine.Installation {
	return knowledgeengine.Installation{Role: "snv", ModuleID: "snv-engine", EngineID: "symphony-snv", Version: "0.1.0-dev", Prefix: "/historical-test-install", ReceiptDigest: "sha256:" + strings.Repeat("a", 64), ExecutableDigest: "sha256:" + strings.Repeat("b", 64)}
}
func evidenceFixture(t *testing.T, id string) Attempt {
	t.Helper()
	bundle := map[string]any{"protocol": "symphony.snv.bundle.v1", "bundle_id": "test-bundle", "artifacts": []any{}, "relations": []any{}}
	input := testRaw(t, map[string]any{"protocol": "symphony.snv.evidence-plan-input.v1", "operation_id": id, "mode": "bundle", "bundle": bundle, "export_records": nil})
	hash, _ := knowledgeengine.SCVDigest(bundle)
	source := DigestBytes(input)
	plan := testSealed(t, map[string]any{"protocol": "symphony.snv.evidence-plan.v1", "operation_id": id, "bundle": bundle, "bundle_digest": hash, "replay": map[string]any{"bundle_digest": hash}, "source_digest": source})
	a, e := NewAttempt("evidence", input, plan, json.RawMessage("null"), testInstallation())
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func selectionFixture(t *testing.T, id string, prior json.RawMessage, unselect bool) Attempt {
	t.Helper()
	tops := "00000000-0000-4000-8000-000000000001"
	var expected any
	generation := int64(1)
	if string(prior) != "null" {
		p, e := rawObject(prior)
		if e != nil {
			t.Fatal(e)
		}
		expected = p["digest"]
		var n int64
		_ = json.Unmarshal(testRaw(t, p["generation"]), &n)
		generation = n + 1
	}
	view := map[string]any{"tops_id": tops, "view_id": "test-view"}
	kind := "select"
	var bundle any = map[string]any{"protocol": "symphony.snv.bundle.v1", "bundle_id": "test", "artifacts": []any{}, "relations": []any{}}
	var bundleDigest any = "sha256:" + strings.Repeat("c", 64)
	if unselect {
		kind = "unselect"
		bundle = nil
		bundleDigest = nil
	}
	input := testRaw(t, map[string]any{"protocol": "symphony.snv.state-plan-input.v1", "view": view, "operation_id": id, "expected_state_digest": expected, "prior_head": prior, "change_kind": kind, "bundle": bundle, "reason": "test supplied native proposal", "migration": nil})
	head := testSealed(t, map[string]any{"protocol": "symphony.snv.head.v1", "view": view, "generation": generation, "previous_digest": expected, "bundle_digest": bundleDigest, "tombstone": unselect, "operation_id": id, "owner_version": "0.1.0-dev"})
	plan := testSealed(t, map[string]any{"protocol": "symphony.snv.state-plan.v1", "operation_id": id, "expected_state_digest": expected, "change_kind": kind, "reason": "test supplied native proposal", "migration": nil, "head": head, "input_digest": DigestBytes(input)})
	var p map[string]any
	_ = json.Unmarshal(plan, &p)
	transition := testSealed(t, map[string]any{"protocol": "symphony.snv.state-transition.v1", "operation_id": id, "expected_state_digest": expected, "head": head, "plan_digest": p["digest"], "effect": "proposed_only"})
	a, e := NewAttempt("selection", input, plan, transition, testInstallation())
	if e != nil {
		t.Fatal(e)
	}
	return a
}

// Structural fixture only; real authenticated SSIAG process acceptance belongs
// to command integration. This package never authenticates a supplied decision.
func decisionFixture(t *testing.T, s Store, a Attempt, expired bool) json.RawMessage {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	if expired {
		now = now.Add(-2 * time.Minute)
	}
	expires := now.Add(time.Minute)
	id, e := stavprotocol.GenerateUUIDv4()
	if e != nil {
		t.Fatal(e)
	}
	basis := "host_owner"
	hash := "sha256:" + strings.Repeat("b", 64)
	d := ssiagclient.AuthorizationDecision{Schema: "symphony.ssiag.authorization-decision.v1", DecisionID: "structural-fixture", RequestID: id, CorrelationID: a.CorrelationID, TOPSID: s.TOPSID, Subject: ssiagclient.DecisionSubject{ID: "test-owner", Kind: "host", Authority: "unix_peer_credentials"}, Target: ssiagclient.DecisionTarget{Operation: "symphony.snv.view.select", Resource: Resource(s.TOPSID, s.ViewID), Audience: "qxctl", Scope: "tops:" + s.TOPSID}, Effect: "allow", ReasonCode: "symphony.ssiag.policy.exact-grant", AuthorityBasis: &basis, PolicyDigest: hash, ConfigDigest: hash, DecidedAt: now, ExpiresAt: &expires}
	c := ssiagclient.Capability{Protocol: "symphony.ssiag.capability.v1", Subject: d.Subject, TOPSID: d.TOPSID, Target: d.Target, AuthorityBasis: basis, GrantID: "structural-test-grant", RequestID: d.RequestID, CorrelationID: d.CorrelationID, IssuedAt: now, ExpiresAt: expires, PolicyDigest: hash, ConfigDigest: hash}
	c.BindingDigest = capabilityBinding(c)
	c.CapabilityID = "ssiag-capability:" + strings.TrimPrefix(c.BindingDigest, "sha256:")
	d.Capability = &c
	return testRaw(t, d)
}
func TestSNVEvidenceRetention(t *testing.T) {
	s, e := NewEvidence(testRoot(t))
	if e != nil {
		t.Fatal(e)
	}
	a := evidenceFixture(t, "capture-1")
	e = s.WithLock(func(tx *Transaction) error {
		committed, e := tx.Prepare(a)
		if e != nil || committed {
			t.Fatalf("prepare: %v %v", committed, e)
		}
		if string(tx.Current()) != "null" || len(tx.Snapshot().Bundles) != 0 {
			t.Fatal("prepare selected or retained bundle")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	e = s.WithLock(func(tx *Transaction) error {
		old, ok := tx.Attempt(a.OperationID)
		if !ok || !same(old.Input, a.Input) || old.Status != "prepared" {
			t.Fatal("prepared original input missing")
		}
		return tx.CommitEvidence(a.OperationID, func() error { return errors.New("original engine drift") })
	})
	if e == nil {
		t.Fatal("failed guard committed")
	}
	e = s.WithLock(func(tx *Transaction) error {
		if len(tx.Snapshot().Bundles) != 0 {
			t.Fatal("failed guard retained bundle")
		}
		return tx.CommitEvidence(a.OperationID, func() error { return nil })
	})
	if e != nil {
		t.Fatal(e)
	}
	e = s.WithLock(func(tx *Transaction) error {
		committed, e := tx.Prepare(a)
		if e != nil || !committed {
			t.Fatal("repeat prepare did not reconcile")
		}
		if len(tx.Snapshot().Bundles) != 1 {
			t.Fatal("immutable bundle absent")
		}
		if e := tx.CommitEvidence(a.OperationID, nil); e != nil {
			t.Fatal(e)
		}
		bundle, _ := rawField(a.Input, "bundle")
		reordered := json.RawMessage(fmt.Sprintf(`{"protocol":"symphony.snv.evidence-plan-input.v1","operation_id":"capture-1","mode":"bundle","export_records":null,"bundle":%s}`, bundle))
		variant, e := NewAttempt("evidence", reordered, a.Plan, a.Transition, a.Installation)
		if e != nil || variant.IntentDigest != a.IntentDigest {
			t.Fatalf("JSON field reordering changed immutable intent: %v", e)
		}
		if committed, e := tx.Prepare(variant); e != nil || !committed {
			t.Fatal("reordered same intent did not reconcile")
		}
		original, _ := tx.Attempt(a.OperationID)
		if string(original.Input) != string(a.Input) {
			t.Fatal("idempotent retry replaced original retained input")
		}
		changed := a
		changed.Installation.ExecutableDigest = "sha256:" + strings.Repeat("d", 64)
		changed.IntentDigest, _ = attemptDigest(changed)
		if _, e := tx.Prepare(changed); e == nil {
			t.Fatal("operation changed installation")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestSNVSelectionGuardAndRecovery(t *testing.T) {
	s, e := NewView(testRoot(t), "00000000-0000-4000-8000-000000000001", "test-view")
	if e != nil {
		t.Fatal(e)
	}
	a := selectionFixture(t, "select-1", json.RawMessage("null"), false)
	e = s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(a); e != nil {
			return e
		}
		prepared, ok := tx.Attempt(a.OperationID)
		if !ok || stavprotocol.ValidateRequestUUID(prepared.CorrelationID) != nil {
			t.Fatal("stable correlation absent")
		}
		decision := decisionFixture(t, s, prepared, false)
		return tx.CommitSelection(a.OperationID, decision, func() error { return errors.New("revoked before effect") })
	})
	if e == nil {
		t.Fatal("revoked guard selected view")
	}
	e = s.WithLock(func(tx *Transaction) error {
		prepared, ok := tx.Attempt(a.OperationID)
		if !ok || prepared.Status != "prepared" || string(tx.Current()) != "null" || string(prepared.Authorization) == "null" {
			t.Fatal("uncertain authorization/result distinction lost")
		}
		if e := tx.CommitSelection(a.OperationID, decisionFixture(t, s, prepared, true), func() error { return nil }); e == nil {
			t.Fatal("expired authority selected")
		}
		return tx.CommitSelection(a.OperationID, decisionFixture(t, s, prepared, false), func() error { return nil })
	})
	if e != nil {
		t.Fatal(e)
	}
	var head json.RawMessage
	e = s.WithLock(func(tx *Transaction) error {
		head = tx.Current()
		old, _ := tx.Attempt(a.OperationID)
		if old.Status != "committed" || string(head) == "null" {
			t.Fatal("selection not committed")
		}
		if e := tx.CommitSelection(a.OperationID, nil, nil); e != nil {
			t.Fatal("committed recovery repeated effect")
		}
		stale := selectionFixture(t, "stale", json.RawMessage("null"), false)
		if _, e := tx.Prepare(stale); e == nil {
			t.Fatal("expected head conflict accepted")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	tombstone := selectionFixture(t, "unselect-1", head, true)
	e = s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(tombstone); e != nil {
			return e
		}
		prepared, _ := tx.Attempt(tombstone.OperationID)
		return tx.CommitSelection(tombstone.OperationID, decisionFixture(t, s, prepared, false), func() error { return nil })
	})
	if e != nil {
		t.Fatal(e)
	}
	e = s.WithLock(func(tx *Transaction) error {
		var m map[string]any
		_ = json.Unmarshal(tx.Current(), &m)
		if m["tombstone"] != true || m["bundle_digest"] != nil || len(tx.Snapshot().Operations) != 2 {
			t.Fatal("unselect erased history or retained selection")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestSNVAuthorityBinding(t *testing.T) {
	s, _ := NewView(testRoot(t), "00000000-0000-4000-8000-000000000001", "test-view")
	a := selectionFixture(t, "first", json.RawMessage("null"), false)
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(a); e != nil {
			return e
		}
		prepared, _ := tx.Attempt(a.OperationID)
		good := decisionFixture(t, s, prepared, false)
		for _, change := range []func(*ssiagclient.AuthorizationDecision){func(d *ssiagclient.AuthorizationDecision) { d.Target.Resource = "another-view" }, func(d *ssiagclient.AuthorizationDecision) { d.CorrelationID = "00000000-0000-4000-8000-000000000099" }, func(d *ssiagclient.AuthorizationDecision) { d.Capability.Transferable = true }, func(d *ssiagclient.AuthorizationDecision) {
			d.Capability.BindingDigest = "sha256:" + strings.Repeat("0", 64)
		}} {
			var d ssiagclient.AuthorizationDecision
			_ = json.Unmarshal(good, &d)
			change(&d)
			if canonicalAuthorization(testRaw(t, d), s, prepared) == nil {
				t.Fatal("misbound authority accepted")
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}
func TestSNVFilesystemOwnershipAndLock(t *testing.T) {
	root := testRoot(t)
	s, _ := NewEvidence(root)
	if e := s.WithLock(func(tx *Transaction) error {
		if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
			t.Fatal("concurrent lock accepted")
		}
		_, e := tx.Prepare(evidenceFixture(t, "one"))
		return e
	}); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(root, "symphony", "qxctl", "snv", "evidence-v1")
	state := filepath.Join(dir, "state.json")
	linked := filepath.Join(root, "unrelated.json")
	if e := os.Link(state, linked); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("hardlinked state accepted")
	}
	if e := os.Remove(linked); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(state, 0644); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("nonprivate state accepted")
	}
	_ = os.Chmod(state, 0600)
	symlink := filepath.Join(root, "alias")
	if e := os.Symlink(dir, symlink); e != nil {
		t.Fatal(e)
	}
	unsafe, _ := NewEvidence(symlink)
	if e := unsafe.WithLock(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("symlink root accepted")
	}
}

func TestSNVOrphanRecoveryAndArtifactCorrespondence(t *testing.T) {
	s, _ := NewEvidence(testRoot(t))
	a := evidenceFixture(t, "recoverable")
	if e := s.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(a); return e }); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(s.Root, "symphony", "qxctl", "snv", "evidence-v1")
	orphan := filepath.Join(dir, ".state-"+strings.Repeat("a", 32)+".tmp")
	unrelated := filepath.Join(dir, "user-notes.txt")
	if e := os.WriteFile(orphan, []byte("interrupted partial writer bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(unrelated, []byte("keep"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(tx *Transaction) error { return tx.CommitEvidence(a.OperationID, func() error { return nil }) }); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(orphan); !os.IsNotExist(e) {
		t.Fatal("owned interrupted temp was not reconciled")
	}
	if data, e := os.ReadFile(unrelated); e != nil || string(data) != "keep" {
		t.Fatal("unrelated private file removed")
	}
	var plan map[string]any
	_ = json.Unmarshal(a.Plan, &plan)
	plan["source_digest"] = "sha256:" + strings.Repeat("f", 64)
	delete(plan, "digest")
	if _, e := NewAttempt("evidence", a.Input, testSealed(t, plan), json.RawMessage("null"), a.Installation); e == nil {
		t.Fatal("sealed but unbound owner result accepted")
	}
	selection := selectionFixture(t, "one", json.RawMessage("null"), false)
	var transition map[string]any
	_ = json.Unmarshal(selection.Transition, &transition)
	transition["plan_digest"] = "sha256:" + strings.Repeat("f", 64)
	delete(transition, "digest")
	if _, e := NewAttempt("selection", selection.Input, selection.Plan, testSealed(t, transition), selection.Installation); e == nil {
		t.Fatal("sealed transition detached from plan accepted")
	}
}

func TestSNVRetainedAuthorityCorrespondence(t *testing.T) {
	s, _ := NewView(testRoot(t), "00000000-0000-4000-8000-000000000001", "test-view")
	a := selectionFixture(t, "retained", json.RawMessage("null"), false)
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(a); e != nil {
			return e
		}
		prepared, _ := tx.Attempt(a.OperationID)
		return tx.CommitSelection(a.OperationID, decisionFixture(t, s, prepared, false), func() error { return nil })
	}); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(tx *Transaction) error {
		doc := tx.Snapshot()
		retained := doc.Operations[a.OperationID]
		// Expiry does not invalidate history. It only prevents a new effect.
		retained.Authorization = decisionFixture(t, s, retained, true)
		doc.Operations[a.OperationID] = retained
		doc.Digest, _ = seal(doc)
		if e := validateDocument(doc, s); e != nil {
			t.Fatalf("historical authority rejected: %v", e)
		}
		for _, change := range []func(*Attempt){
			func(v *Attempt) { v.Authorization = json.RawMessage("null") },
			func(v *Attempt) { v.CorrelationID = "" },
			func(v *Attempt) {
				var d ssiagclient.AuthorizationDecision
				_ = json.Unmarshal(v.Authorization, &d)
				d.Target.Resource = "another-view"
				v.Authorization = testRaw(t, d)
			},
			func(v *Attempt) { v.PriorAuthorizations = []json.RawMessage{json.RawMessage("null")} },
		} {
			broken := tx.Snapshot()
			attempt := broken.Operations[a.OperationID]
			change(&attempt)
			broken.Operations[a.OperationID] = attempt
			broken.Digest, _ = seal(broken)
			if validateDocument(broken, s) == nil {
				t.Fatal("sealed journal accepted incomplete retained authority")
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}

func TestSNVObservationHasNoPersistentWrites(t *testing.T) {
	parent := testRoot(t)
	root := filepath.Join(parent, "absent-state")
	s, _ := NewEvidence(root)
	read := func(tx *Transaction) error {
		if string(tx.Current()) != "null" || len(tx.Snapshot().Operations) != 0 {
			t.Fatal("absent observation was not empty")
		}
		if _, e := tx.Prepare(evidenceFixture(t, "must-not-save")); e == nil {
			t.Fatal("observational transaction mutated")
		}
		return nil
	}
	if e := s.WithRead(read); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("absent observation created root or lock infrastructure")
	}
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	if e := s.WithRead(read); e != nil {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("existing empty root observation created infrastructure")
	}
	a := evidenceFixture(t, "retained-observation")
	if e := s.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(a); return e }); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(root, "symphony", "qxctl", "snv", "evidence-v1")
	state := filepath.Join(dir, "state.json")
	before, _ := os.ReadFile(state)
	orphan := filepath.Join(dir, ".state-"+strings.Repeat("b", 32)+".tmp")
	if e := os.WriteFile(orphan, []byte("interrupted"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := s.WithRead(func(tx *Transaction) error {
		if _, ok := tx.Attempt(a.OperationID); !ok {
			t.Fatal("observed retained operation missing")
		}
		if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
			t.Fatal("exclusive writer entered while observation held shared lock")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(state)
	if string(before) != string(after) {
		t.Fatal("observation changed journal")
	}
	if data, e := os.ReadFile(orphan); e != nil || string(data) != "interrupted" {
		t.Fatal("observation performed writer recovery")
	}
	if e := os.Remove(filepath.Join(dir, "head.lock")); e != nil {
		t.Fatal(e)
	}
	if e := s.WithRead(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("existing state without stable lock was observed")
	}
}

func TestSNVCapacityPreservesAcknowledgedJournal(t *testing.T) {
	s, _ := NewEvidence(testRoot(t))
	if e := s.WithLock(func(tx *Transaction) error {
		// Acknowledged history is a fixture, published once through the real
		// journal writer. The public next-operation boundary is exercised below.
		next := tx.Snapshot()
		for i := 0; i < 511; i++ {
			a := evidenceFixture(t, fmt.Sprintf("acknowledged-%03d", i))
			next.Operations[a.OperationID] = a
		}
		if e := tx.save(next, nil); e != nil {
			return e
		}
		if _, e := tx.Prepare(evidenceFixture(t, "acknowledged-511")); e != nil {
			return e
		}
		if tx.Snapshot().Operations["acknowledged-511"].Status != "prepared" {
			t.Fatal("operation 512 not durably acknowledged")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(s.Root, "symphony", "qxctl", "snv", "evidence-v1", "state.json")
	before, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(evidenceFixture(t, "operation-513")); e == nil {
			t.Fatal("operation 513 accepted")
		} else {
			requireBoundary(t, e, "snv.capacity_exceeded")
		}
		if len(tx.Snapshot().Operations) != 512 || string(tx.Current()) != "null" {
			t.Fatal("capacity refusal pruned history or selected head")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("capacity refusal changed acknowledged journal bytes")
	}
	if e := s.WithRead(func(tx *Transaction) error {
		for i := 0; i < 512; i++ {
			if a, ok := tx.Attempt(fmt.Sprintf("acknowledged-%03d", i)); !ok || a.Status != "prepared" {
				t.Fatal("acknowledged operation missing after capacity refusal")
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}

func largeSelectionFixture(t *testing.T, id string, prior json.RawMessage) Attempt {
	t.Helper()
	proposal := selectionFixture(t, id, prior, false)
	input, _ := rawObject(proposal.Input)
	bundle := input["bundle"].(map[string]any)
	// Opaque native evidence is retained by this adapter. These structural
	// fixtures exercise accumulated bytes, not owner semantics. Each string
	// stays inside the independently enforced 65,536-byte JSON string bound.
	bundle["artifacts"] = []any{
		map[string]any{"source_utf8": strings.Repeat("a", 60*1024)},
		map[string]any{"source_utf8": strings.Repeat("b", 60*1024)},
		map[string]any{"source_utf8": strings.Repeat("c", 60*1024)},
		map[string]any{"source_utf8": strings.Repeat("d", 60*1024)},
	}
	inputRaw := testRaw(t, input)
	plan, _ := rawObject(proposal.Plan)
	head := plan["head"].(map[string]any)
	head["bundle_digest"], _ = knowledgeengine.SCVDigest(bundle)
	delete(head, "digest")
	plan["head"] = testSealed(t, head)
	plan["input_digest"] = DigestBytes(inputRaw)
	delete(plan, "digest")
	planRaw := testSealed(t, plan)
	transition, _ := rawObject(proposal.Transition)
	transition["head"] = plan["head"]
	transition["plan_digest"] = plan["digest"]
	delete(transition, "digest")
	a, e := NewAttempt("selection", inputRaw, planRaw, testSealed(t, transition), proposal.Installation)
	if e != nil {
		t.Fatal(e)
	}
	return a
}

func TestSNVByteCapacityPreservesAcknowledgedHead(t *testing.T) {
	s, _ := NewView(testRoot(t), "00000000-0000-4000-8000-000000000001", "test-view")
	base := selectionFixture(t, "selected-base", json.RawMessage("null"), false)
	if e := s.WithLock(func(tx *Transaction) error {
		if _, e := tx.Prepare(base); e != nil {
			return e
		}
		prepared, _ := tx.Attempt(base.OperationID)
		return tx.CommitSelection(base.OperationID, decisionFixture(t, s, prepared, false), func() error { return nil })
	}); e != nil {
		t.Fatal(e)
	}
	identity, _ := knowledgeengineDigestIdentity(s.TOPSID, s.ViewID)
	path := filepath.Join(s.Root, "symphony", "qxctl", "snv", "views-v1", s.TOPSID, identity, "state.json")
	var acknowledged json.RawMessage
	refused := false
	if e := s.WithLock(func(tx *Transaction) error {
		acknowledged = tx.Current()
		// Retained near-capacity history is published once. Subsequent public
		// prepares test the refusal boundary without dozens of redundant fsyncs.
		next := tx.Snapshot()
		for i := 0; i < 63; i++ {
			a := largeSelectionFixture(t, fmt.Sprintf("large-evidence-%03d", i), acknowledged)
			a.CorrelationID, _ = stavprotocol.GenerateUUIDv4()
			next.Operations[a.OperationID] = a
		}
		if e := tx.save(next, nil); e != nil {
			return e
		}
		for i := 63; i < 128; i++ {
			a := largeSelectionFixture(t, fmt.Sprintf("large-evidence-%03d", i), acknowledged)
			before, _ := os.ReadFile(path)
			if _, e := tx.Prepare(a); e != nil {
				requireBoundary(t, e, "snv.capacity_exceeded")
				after, _ := os.ReadFile(path)
				if string(after) != string(before) || !same(tx.Current(), acknowledged) || len(after) > 16*1024*1024 {
					t.Fatal("byte refusal changed acknowledged head or journal")
				}
				if _, ok := tx.Attempt(a.OperationID); ok {
					t.Fatal("overflow operation acknowledged")
				}
				refused = true
				break
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if !refused {
		t.Fatal("accumulated native evidence did not reach explicit byte capacity")
	}
	before, _ := os.ReadFile(path)
	oversized := []byte(strings.Repeat(" ", 16*1024*1024+1))
	if e := os.WriteFile(path, oversized, 0600); e != nil {
		t.Fatal(e)
	}
	if e := s.WithRead(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("16MiB+1 existing journal accepted")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(oversized) {
		t.Fatal("oversized observation repaired or truncated user state")
	}
	if e := os.WriteFile(path, before, 0600); e != nil {
		t.Fatal(e)
	}
	if e := s.WithRead(func(tx *Transaction) error {
		if !same(tx.Current(), acknowledged) {
			t.Fatal("restored acknowledged head differs")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}

func TestSNVCompetingWriterProcess(t *testing.T) {
	root := os.Getenv("SYMPHONY_SNV_TEST_COMPETING_ROOT")
	if root == "" {
		t.Skip("separate competing-writer helper")
	}
	s, e := NewEvidence(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(evidenceFixture(t, "competing")); return e }); e == nil {
		t.Fatal("separate competing writer entered locked store")
	}
}

func TestSNVConcurrentWriterAndUnsafeLockPreservation(t *testing.T) {
	s, _ := NewEvidence(testRoot(t))
	if e := s.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(evidenceFixture(t, "acknowledged")); return e }); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(s.Root, "symphony", "qxctl", "snv", "evidence-v1")
	state, lock := filepath.Join(dir, "state.json"), filepath.Join(dir, "head.lock")
	before, _ := os.ReadFile(state)
	if e := s.WithLock(func(tx *Transaction) error {
		binary, e := os.Executable()
		if e != nil {
			return e
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestSNVCompetingWriterProcess$", "-test.v")
		cmd.Env = append(os.Environ(), "SYMPHONY_SNV_TEST_COMPETING_ROOT="+s.Root)
		output, e := cmd.CombinedOutput()
		if e != nil || !strings.Contains(string(output), "PASS") {
			t.Fatalf("competing process: %v %s", e, output)
		}
		if len(tx.Snapshot().Operations) != 1 {
			t.Fatal("competing writer changed acknowledged operations")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(state)
	if string(after) != string(before) {
		t.Fatal("competing writer changed acknowledged journal")
	}
	linked := filepath.Join(s.Root, "linked-lock")
	if e := os.Link(lock, linked); e != nil {
		t.Fatal(e)
	}
	if e := s.WithRead(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("hardlinked lock accepted")
	}
	_ = os.Remove(linked)
	_ = os.Remove(lock)
	if e := os.Symlink(state, lock); e != nil {
		t.Fatal(e)
	}
	if e := s.WithLock(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("symlink lock accepted")
	}
	after, _ = os.ReadFile(state)
	if string(after) != string(before) {
		t.Fatal("unsafe lock refusal changed journal")
	}
	_ = os.Remove(lock)
	if e := os.WriteFile(lock, nil, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(dir, 0755); e != nil {
		t.Fatal(e)
	}
	if e := s.WithRead(func(*Transaction) error { return nil }); e == nil {
		t.Fatal("nonprivate journal directory accepted")
	}
	after, _ = os.ReadFile(state)
	if string(after) != string(before) {
		t.Fatal("unsafe directory refusal changed journal")
	}
}

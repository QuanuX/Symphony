// Package snvstate is the private filesystem transaction adapter for exact
// native SNV plans. Native reducers own record, bundle and head meaning.
package snvstate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const protocol = "symphony.qxctl.snv-store.v1"
const maxStoreBytes = 16 * 1024 * 1024
const maxOperations = 512

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
var topsPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var hashPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type Attempt struct {
	Kind                string                       `json:"kind"`
	OperationID         string                       `json:"operation_id"`
	Input               json.RawMessage              `json:"input"`
	Plan                json.RawMessage              `json:"plan"`
	Transition          json.RawMessage              `json:"transition"`
	Installation        knowledgeengine.Installation `json:"installation"`
	IntentDigest        string                       `json:"intent_digest"`
	Status              string                       `json:"status"`
	CorrelationID       string                       `json:"correlation_id"`
	Authorization       json.RawMessage              `json:"authorization"`
	PriorAuthorizations []json.RawMessage            `json:"prior_authorizations"`
}
type Document struct {
	Protocol    string                     `json:"protocol"`
	Kind        string                     `json:"kind"`
	TOPSID      string                     `json:"tops_id"`
	ViewID      string                     `json:"view_id"`
	Head        json.RawMessage            `json:"head"`
	StateDigest *string                    `json:"state_digest"`
	Operations  map[string]Attempt         `json:"operations"`
	Bundles     map[string]json.RawMessage `json:"bundles"`
	Digest      string                     `json:"digest"`
}
type Store struct{ Root, Kind, TOPSID, ViewID string }
type Transaction struct {
	store    Store
	document Document
	save     func(Document, func() error) error
}

func NewEvidence(root string) (Store, error) { return newStore(root, "evidence", "", "") }
func NewView(root, topsID, viewID string) (Store, error) {
	return newStore(root, "view", topsID, viewID)
}
func newStore(root, kind, topsID, viewID string) (Store, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return Store{}, fmt.Errorf("SNV state root must be an explicit clean absolute descendant")
	}
	if kind != "evidence" && kind != "view" || kind == "evidence" && (topsID != "" || viewID != "") || kind == "view" && (!topsPattern.MatchString(topsID) || !tokenPattern.MatchString(viewID)) {
		return Store{}, fmt.Errorf("invalid SNV store identity")
	}
	return Store{root, kind, topsID, viewID}, nil
}
func normalized(v any) (map[string]any, error) {
	data, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	var obj map[string]any
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	e = d.Decode(&obj)
	return obj, e
}
func seal(v any) (string, error) {
	m, e := normalized(v)
	if e != nil {
		return "", e
	}
	delete(m, "digest")
	return knowledgeengine.SCVDigest(m)
}
func rawObject(raw json.RawMessage) (map[string]any, error) {
	return knowledgeengine.ParseSNVObject(raw)
}
func same(a, b any) bool {
	x, e := valueCanonical(a)
	if e != nil {
		return false
	}
	y, e := valueCanonical(b)
	return e == nil && bytes.Equal(x, y)
}
func valueCanonical(value any) ([]byte, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	var object any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e = d.Decode(&object); e != nil {
		return nil, e
	}
	return knowledgeengine.SCVCanonical(object)
}
func verifySeal(raw json.RawMessage) error {
	m, e := rawObject(raw)
	if e != nil {
		return e
	}
	d, ok := m["digest"].(string)
	if !ok || !hashPattern.MatchString(d) {
		return fmt.Errorf("missing SNV native digest")
	}
	delete(m, "digest")
	got, e := knowledgeengine.SCVDigest(m)
	if e != nil || got != d {
		return fmt.Errorf("SNV native sealed object mismatch")
	}
	return nil
}
func rawField(raw json.RawMessage, key string) (json.RawMessage, error) {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m[key] == nil {
		return nil, fmt.Errorf("missing SNV field %s", key)
	}
	return m[key], nil
}
func stringField(raw json.RawMessage, key string) (string, error) {
	r, e := rawField(raw, key)
	if e != nil {
		return "", e
	}
	var s string
	e = json.Unmarshal(r, &s)
	return s, e
}
func attemptDigest(a Attempt) (string, error) {
	object, e := normalized(map[string]any{"kind": a.Kind, "operation_id": a.OperationID, "input": a.Input, "plan": a.Plan, "transition": a.Transition, "installation": a.Installation})
	if e != nil {
		return "", e
	}
	return knowledgeengine.SCVDigest(object)
}
func NewAttempt(kind string, input, plan, transition json.RawMessage, installation knowledgeengine.Installation) (Attempt, error) {
	id, e := stringField(plan, "operation_id")
	if e != nil || !tokenPattern.MatchString(id) {
		return Attempt{}, fmt.Errorf("invalid stable SNV operation identity")
	}
	a := Attempt{Kind: kind, OperationID: id, Input: input, Plan: plan, Transition: transition, Installation: installation, Status: "prepared", Authorization: json.RawMessage("null"), PriorAuthorizations: []json.RawMessage{}}
	a.IntentDigest, e = attemptDigest(a)
	if e != nil {
		return Attempt{}, e
	}
	if e = validateAttempt(a); e != nil {
		return Attempt{}, e
	}
	return cloneAttempt(a), nil
}
func validateAttempt(a Attempt) error {
	if a.Kind != "evidence" && a.Kind != "selection" || !tokenPattern.MatchString(a.OperationID) || a.Status != "prepared" && a.Status != "committed" || a.Installation.EngineID != "symphony-snv" || a.Installation.ModuleID != "snv-engine" || a.Installation.Version != "0.1.0-dev" {
		return fmt.Errorf("invalid SNV retained intent")
	}
	m, e := rawObject(a.Input)
	if e != nil {
		return e
	}
	if e = verifySeal(a.Plan); e != nil {
		return e
	}
	plan, e := rawObject(a.Plan)
	if e != nil {
		return e
	}
	inputDigest, e := knowledgeengine.SCVDigest(m)
	if e != nil {
		return e
	}
	id, e := stringField(a.Plan, "operation_id")
	if e != nil || id != a.OperationID || m["operation_id"] != id {
		return fmt.Errorf("SNV input/plan operation binding mismatch")
	}
	if a.Kind == "selection" {
		if m["protocol"] != "symphony.snv.state-plan-input.v1" || verifySeal(a.Transition) != nil {
			return fmt.Errorf("invalid native SNV selection artifacts")
		}
		transition, e := rawObject(a.Transition)
		if e != nil {
			return e
		}
		if plan["protocol"] != "symphony.snv.state-plan.v1" || plan["input_digest"] != inputDigest ||
			transition["protocol"] != "symphony.snv.state-transition.v1" || transition["operation_id"] != a.OperationID || transition["effect"] != "proposed_only" ||
			transition["plan_digest"] != plan["digest"] || !same(transition["head"], plan["head"]) ||
			!same(plan["expected_state_digest"], m["expected_state_digest"]) || !same(transition["expected_state_digest"], m["expected_state_digest"]) || plan["change_kind"] != m["change_kind"] {
			return fmt.Errorf("SNV selection input, plan and transition correspondence mismatch")
		}
		headRaw, e := knowledgeengine.SCVCanonical(plan["head"])
		if e != nil || verifySeal(headRaw) != nil {
			return fmt.Errorf("invalid SNV proposed head")
		}
		head, e := rawObject(headRaw)
		if e != nil || !same(head["view"], m["view"]) || head["operation_id"] != a.OperationID {
			return fmt.Errorf("SNV proposed head scope mismatch")
		}
		if a.CorrelationID != "" && stavprotocol.ValidateRequestUUID(a.CorrelationID) != nil {
			return fmt.Errorf("invalid durable SNV correlation")
		}
	} else if m["protocol"] != "symphony.snv.evidence-plan-input.v1" || string(a.Transition) != "null" || a.CorrelationID != "" || string(a.Authorization) != "null" {
		return fmt.Errorf("invalid native SNV retention artifacts")
	} else {
		bundleDigest, e := knowledgeengine.SCVDigest(plan["bundle"])
		if e != nil || plan["protocol"] != "symphony.snv.evidence-plan.v1" || plan["source_digest"] != inputDigest || plan["bundle_digest"] != bundleDigest {
			return fmt.Errorf("SNV retained bundle/input correspondence mismatch")
		}
	}
	expected, e := attemptDigest(a)
	if e != nil || expected != a.IntentDigest {
		return fmt.Errorf("SNV retained intent digest mismatch")
	}
	return nil
}
func cloneAttempt(a Attempt) Attempt {
	data, _ := json.Marshal(a)
	var out Attempt
	_ = json.Unmarshal(data, &out)
	return out
}
func cloneDocument(d Document) Document {
	data, _ := json.Marshal(d)
	var out Document
	_ = json.Unmarshal(data, &out)
	return out
}

func (s Store) WithLock(operation func(*Transaction) error) error {
	if _, e := newStore(s.Root, s.Kind, s.TOPSID, s.ViewID); e != nil {
		return e
	}
	return s.withDirectory(func(read func() ([]byte, error), write func([]byte, func() error) error) error {
		data, e := read()
		if e != nil {
			return e
		}
		doc, e := s.loadDocument(data)
		if e != nil {
			return e
		}
		tx := &Transaction{store: s, document: doc}
		tx.save = func(next Document, guard func() error) error {
			next.Digest, e = seal(next)
			if e != nil {
				return e
			}
			if e = validateDocument(next, s); e != nil {
				return e
			}
			data, e := knowledgeengine.SCVCanonical(next)
			if e != nil {
				return e
			}
			if len(data) > maxStoreBytes {
				return fmt.Errorf("SNV journal capacity exceeded; no history is pruned")
			}
			if e = write(data, guard); e != nil {
				return e
			}
			tx.document = next
			return nil
		}
		return operation(tx)
	})
}

// WithRead captures an existing journal using a shared lock without creating
// directories, lock files, journals or writer recovery artifacts. Missing state
// is an empty snapshot. Semantic plans remain pure until an explicit mutation.
func (s Store) WithRead(operation func(*Transaction) error) error {
	if _, e := newStore(s.Root, s.Kind, s.TOPSID, s.ViewID); e != nil {
		return e
	}
	return s.readDirectory(func(data []byte) error {
		doc, e := s.loadDocument(data)
		if e != nil {
			return e
		}
		tx := &Transaction{store: s, document: doc, save: func(Document, func() error) error {
			return fmt.Errorf("SNV observation cannot mutate retained state")
		}}
		return operation(tx)
	})
}

func (s Store) loadDocument(data []byte) (Document, error) {
	doc := Document{Protocol: protocol, Kind: s.Kind, TOPSID: s.TOPSID, ViewID: s.ViewID, Head: json.RawMessage("null"), Operations: map[string]Attempt{}, Bundles: map[string]json.RawMessage{}}
	if data == nil {
		return doc, nil
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&doc) != nil {
		return Document{}, fmt.Errorf("invalid SNV journal")
	}
	if e := validateDocument(doc, s); e != nil {
		return Document{}, e
	}
	canonical, e := knowledgeengine.SCVCanonical(doc)
	if e != nil || !bytes.Equal(canonical, data) {
		return Document{}, fmt.Errorf("SNV journal is not exact canonical JSON")
	}
	return doc, nil
}
func validateDocument(d Document, s Store) error {
	if d.Protocol != protocol || d.Kind != s.Kind || d.TOPSID != s.TOPSID || d.ViewID != s.ViewID || d.Operations == nil || d.Bundles == nil || len(d.Operations) > maxOperations || len(d.Bundles) > maxOperations {
		return fmt.Errorf("SNV journal identity or capacity mismatch")
	}
	digest, e := seal(d)
	if e != nil || d.Digest != digest {
		return fmt.Errorf("SNV journal digest mismatch")
	}
	for id, a := range d.Operations {
		if id != a.OperationID || validateAttempt(a) != nil || s.Kind == "evidence" && a.Kind != "evidence" || s.Kind == "view" && a.Kind != "selection" {
			return fmt.Errorf("SNV retained operation mismatch")
		}
		if a.Kind == "selection" {
			if stavprotocol.ValidateRequestUUID(a.CorrelationID) != nil || a.Status == "committed" && string(a.Authorization) == "null" {
				return fmt.Errorf("SNV retained selection authority evidence absent")
			}
			if string(a.Authorization) != "null" && canonicalAuthorization(a.Authorization, s, a) != nil {
				return fmt.Errorf("SNV retained selection authority correspondence mismatch")
			}
			for _, prior := range a.PriorAuthorizations {
				if canonicalAuthorization(prior, s, a) != nil {
					return fmt.Errorf("SNV retained prior authority correspondence mismatch")
				}
			}
		}
	}
	for hash, bundle := range d.Bundles {
		m, e := rawObject(bundle)
		if e != nil {
			return e
		}
		actual, e := knowledgeengine.SCVDigest(m)
		if e != nil || hash != actual {
			return fmt.Errorf("SNV retained bundle digest mismatch")
		}
	}
	if s.Kind == "evidence" {
		if string(d.Head) != "null" || d.StateDigest != nil {
			return fmt.Errorf("evidence retention cannot select a view")
		}
	}
	if string(d.Head) != "null" {
		if verifySeal(d.Head) != nil {
			return fmt.Errorf("SNV head digest mismatch")
		}
		hash, e := stringField(d.Head, "digest")
		if e != nil || d.StateDigest == nil || *d.StateDigest != hash {
			return fmt.Errorf("SNV current head binding mismatch")
		}
		id, e := stringField(d.Head, "operation_id")
		a, ok := d.Operations[id]
		if e != nil || !ok || a.Status != "committed" || a.Kind != "selection" {
			return fmt.Errorf("SNV selected head lacks committed attempt")
		}
		proposed, e := rawField(a.Transition, "head")
		if e != nil || !same(proposed, d.Head) {
			return fmt.Errorf("SNV selected head differs from committed transition")
		}
	}
	if string(d.Head) == "null" && d.StateDigest != nil {
		return fmt.Errorf("absent SNV head carries digest")
	}
	return nil
}
func (t *Transaction) Snapshot() Document { return cloneDocument(t.document) }
func (t *Transaction) Current() json.RawMessage {
	return append(json.RawMessage(nil), t.document.Head...)
}
func (t *Transaction) Attempt(id string) (Attempt, bool) {
	a, ok := t.document.Operations[id]
	return cloneAttempt(a), ok
}
func (t *Transaction) Bundle(digest string) (json.RawMessage, bool) {
	b, ok := t.document.Bundles[digest]
	return append(json.RawMessage(nil), b...), ok
}
func (t *Transaction) Prepare(a Attempt) (bool, error) {
	if e := validateAttempt(a); e != nil {
		return false, e
	}
	if t.store.Kind == "evidence" && a.Kind != "evidence" || t.store.Kind == "view" && a.Kind != "selection" {
		return false, fmt.Errorf("SNV store/operation kind mismatch")
	}
	if old, ok := t.Attempt(a.OperationID); ok {
		if old.IntentDigest != a.IntentDigest {
			return false, fmt.Errorf("SNV operation binds another immutable intent")
		}
		return old.Status == "committed", nil
	}
	if len(t.document.Operations) >= maxOperations {
		return false, fmt.Errorf("SNV operation capacity exceeded")
	}
	if a.Kind == "selection" {
		p, e := rawObject(a.Input)
		if e != nil {
			return false, e
		}
		if !same(p["prior_head"], t.Current()) || !same(p["expected_state_digest"], t.document.StateDigest) {
			return false, fmt.Errorf("SNV expected head conflict")
		}
		view, e := normalized(map[string]any{"tops_id": t.store.TOPSID, "view_id": t.store.ViewID})
		if e != nil || !same(p["view"], view) {
			return false, fmt.Errorf("SNV selected view scope mismatch")
		}
		a.CorrelationID, e = stavprotocol.GenerateUUIDv4()
		if e != nil {
			return false, e
		}
	}
	next := t.Snapshot()
	next.Operations[a.OperationID] = a
	if e := t.save(next, nil); e != nil {
		return false, e
	}
	barrier("snv.after_prepare")
	return false, nil
}
func (t *Transaction) CommitEvidence(id string, guard func() error) error {
	a, ok := t.Attempt(id)
	if !ok || a.Kind != "evidence" {
		return fmt.Errorf("SNV evidence attempt absent")
	}
	if a.Status == "committed" {
		return nil
	}
	if guard == nil {
		return fmt.Errorf("SNV retention requires original input replay at the effect boundary")
	}
	plan, e := rawObject(a.Plan)
	if e != nil {
		return e
	}
	hash, ok := plan["bundle_digest"].(string)
	if !ok || !hashPattern.MatchString(hash) {
		return fmt.Errorf("SNV evidence bundle digest missing")
	}
	bundle, e := knowledgeengine.SCVCanonical(plan["bundle"])
	if e != nil {
		return e
	}
	if prior, ok := t.Bundle(hash); ok && !same(prior, json.RawMessage(bundle)) {
		return fmt.Errorf("SNV immutable bundle collision")
	}
	next := t.Snapshot()
	if _, ok := next.Bundles[hash]; !ok && len(next.Bundles) >= maxOperations {
		return fmt.Errorf("SNV retained bundle capacity exceeded")
	}
	next.Bundles[hash] = bundle
	a.Status = "committed"
	next.Operations[id] = a
	return t.save(next, guard)
}
func (t *Transaction) CommitSelection(id string, authorization json.RawMessage, guard func() error) error {
	a, ok := t.Attempt(id)
	if !ok || a.Kind != "selection" {
		return fmt.Errorf("SNV selection attempt absent")
	}
	if a.Status == "committed" {
		return nil
	}
	if e := canonicalAuthorization(authorization, t.store, a); e != nil {
		return e
	}
	input, e := rawObject(a.Input)
	if e != nil {
		return e
	}
	if !same(input["prior_head"], t.Current()) || !same(input["expected_state_digest"], t.document.StateDigest) {
		return fmt.Errorf("SNV expected head conflict")
	}
	next := t.Snapshot()
	if string(a.Authorization) != "null" && !same(a.Authorization, authorization) {
		a.PriorAuthorizations = append(a.PriorAuthorizations, a.Authorization)
	}
	a.Authorization = append(json.RawMessage(nil), authorization...)
	next.Operations[id] = a
	if e = t.save(next, nil); e != nil {
		return e
	}
	barrier("snv.after_authorization")
	head, e := rawField(a.Transition, "head")
	if e != nil {
		return e
	}
	hash, e := stringField(head, "digest")
	if e != nil {
		return e
	}
	a.Status = "committed"
	next = t.Snapshot()
	next.Head = head
	next.StateDigest = &hash
	next.Operations[id] = a
	return t.save(next, func() error {
		if e := authorizationFresh(authorization); e != nil {
			return e
		}
		if guard == nil {
			return fmt.Errorf("SNV selection requires a live effect guard")
		}
		return guard()
	})
}
func Resource(topsID, viewID string) string {
	d, _ := knowledgeengine.SCVDigest(map[string]any{"tops_id": topsID, "view_id": viewID, "owner_engine_id": "symphony-snv"})
	return "symphony.snv.view:" + d[len("sha256:"):]
}
func DigestBytes(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }

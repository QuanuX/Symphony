// Package scvstate is qxctl's protected filesystem transaction adapter. It does
// not derive source generations, locator meaning, or transition semantics;
// those are supplied by the exact receipt-validated C++ source owner.
package scvstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"regexp"
)

const protocol = "symphony.qxctl.scv-source-store.v2"
const legacyProtocol = "symphony.qxctl.scv-source-store.v1"
const maxStoreBytes = 16 * 1024 * 1024
const maxOperations = 128

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var topsPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Intent binds an exact plan, source owner installation and reduced transition.
// The outer random process request ID is intentionally not transaction identity.
type Intent struct {
	OperationID  string                       `json:"operation_id"`
	Plan         json.RawMessage              `json:"plan"`
	Transition   json.RawMessage              `json:"transition"`
	Installation knowledgeengine.Installation `json:"installation"`
	Digest       string                       `json:"digest"`
}
type Attempt struct {
	Intent Intent `json:"intent"`
	Status string `json:"status"`
	// Bound durably before authorization; deliberately outside the native intent.
	// Omission preserves exact legacy documents and untouched historical attempts.
	CorrelationID string `json:"correlation_id,omitempty"`
	// This is the actual SSIAG decision returned after its committed policy
	// audit, never a fabricated STAV source-write receipt.
	Authorization       json.RawMessage   `json:"authorization"`
	PriorAuthorizations []json.RawMessage `json:"prior_authorizations"`
}
type Document struct {
	Protocol        string             `json:"protocol"`
	TOPSID          string             `json:"tops_id"`
	Domain          string             `json:"domain"`
	SourceID        string             `json:"source_id"`
	Source          json.RawMessage    `json:"source"`
	StateDigest     *string            `json:"state_digest"`
	HeadOperationID *string            `json:"head_operation_id"`
	Operations      map[string]Attempt `json:"operations"`
	Digest          string             `json:"digest"`
}
type Store struct{ Root, TOPSID, Domain, SourceID string }
type Transaction struct {
	store            Store
	document         Document
	save             func(Document) error
	publicationGuard func() error
}

func New(root, topsID, domain, sourceID string) (Store, error) {
	if !topsPattern.MatchString(topsID) || !tokenPattern.MatchString(sourceID) {
		return Store{}, fmt.Errorf("invalid TOPS/source identity")
	}
	validDomain := false
	for _, allowed := range knowledgeengine.SCVDomains() {
		if allowed == domain {
			validDomain = true
		}
	}
	if !validDomain {
		return Store{}, fmt.Errorf("invalid source owner domain")
	}
	return Store{Root: root, TOPSID: topsID, Domain: domain, SourceID: sourceID}, nil
}

func normalized(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	return object, nil
}
func seal(value any) (string, error) {
	object, err := normalized(value)
	if err != nil {
		return "", err
	}
	delete(object, "digest")
	return knowledgeengine.SCVDigest(object)
}

func NewIntent(operationID string, plan, transition json.RawMessage, installation knowledgeengine.Installation) (Intent, error) {
	if !tokenPattern.MatchString(operationID) {
		return Intent{}, fmt.Errorf("invalid stable operation identity")
	}
	intent := Intent{OperationID: operationID, Plan: plan, Transition: transition, Installation: installation}
	digest, err := seal(intent)
	intent.Digest = digest
	return intent, err
}

func (s Store) WithLock(operation func(*Transaction) error) error {
	return s.withDirectory(func(read func() ([]byte, error), write func([]byte, func() error) error) error {
		data, err := read()
		if err != nil {
			return err
		}
		document := Document{Protocol: protocol, TOPSID: s.TOPSID, Domain: s.Domain, SourceID: s.SourceID,
			Source: json.RawMessage("null"), Operations: map[string]Attempt{}}
		if data != nil {
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&document); err != nil {
				return fmt.Errorf("invalid source store: %w", err)
			}
			if err := validateDocument(document, s); err != nil {
				return err
			}
			canonical, err := knowledgeengine.SCVCanonical(document)
			if err != nil || !bytes.Equal(canonical, data) {
				return fmt.Errorf("source store encoding is not canonical")
			}
		}
		tx := &Transaction{store: s, document: document}
		tx.save = func(next Document) error {
			var err error
			next.Digest, err = seal(next)
			if err != nil {
				return err
			}
			if err := validateDocument(next, s); err != nil {
				return err
			}
			encoded, err := knowledgeengine.SCVCanonical(next)
			if err != nil {
				return err
			}
			if len(encoded) > maxStoreBytes {
				return fmt.Errorf("source history is full; no implicit history pruning is permitted")
			}
			if err := write(encoded, tx.publicationGuard); err != nil {
				return err
			}
			tx.document = next
			return nil
		}
		return operation(tx)
	})
}

func validateDocument(d Document, s Store) error {
	if (d.Protocol != protocol && d.Protocol != legacyProtocol) || d.TOPSID != s.TOPSID || d.Domain != s.Domain || d.SourceID != s.SourceID || d.Operations == nil || len(d.Operations) > maxOperations {
		return fmt.Errorf("source store identity/bounds mismatch")
	}
	digest, err := seal(d)
	if err != nil || digest != d.Digest {
		return fmt.Errorf("source store digest mismatch")
	}
	if d.StateDigest == nil {
		if !bytes.Equal(d.Source, []byte("null")) || d.HeadOperationID != nil {
			return fmt.Errorf("invalid absent source head")
		}
	} else {
		var source struct {
			SourceID string `json:"source_id"`
			Digest   string `json:"digest"`
		}
		if json.Unmarshal(d.Source, &source) != nil || source.SourceID != d.SourceID || source.Digest != *d.StateDigest || d.HeadOperationID == nil {
			return fmt.Errorf("source head binding mismatch")
		}
		head, ok := d.Operations[*d.HeadOperationID]
		if !ok || head.Status != "committed" {
			return fmt.Errorf("source head lacks committed operation")
		}
		var transition struct {
			State  json.RawMessage `json:"state"`
			Digest string          `json:"state_digest"`
		}
		if json.Unmarshal(head.Intent.Transition, &transition) != nil || !sameJSON(d.Source, transition.State) || transition.Digest != *d.StateDigest {
			return fmt.Errorf("source head differs from committed transition")
		}
	}
	type retainedState struct {
		Generation  int64   `json:"generation"`
		Predecessor *string `json:"predecessor_digest"`
		SourceID    string  `json:"source_id"`
		ProviderID  string  `json:"provider_id"`
		FamilyID    string  `json:"family_id"`
		Digest      string  `json:"digest"`
	}
	committed := map[string]retainedState{}
	for id, attempt := range d.Operations {
		if attempt.CorrelationID != "" && (d.Protocol == legacyProtocol || stavprotocol.ValidateRequestUUID(attempt.CorrelationID) != nil) {
			return fmt.Errorf("invalid source authorization correlation")
		}
		if id != attempt.Intent.OperationID || !tokenPattern.MatchString(id) {
			return fmt.Errorf("source operation identity mismatch")
		}
		expected, err := seal(attempt.Intent)
		if err != nil || expected != attempt.Intent.Digest {
			return fmt.Errorf("source intent digest mismatch")
		}
		if attempt.Status != "prepared" && attempt.Status != "authorized" && attempt.Status != "committed" {
			return fmt.Errorf("invalid source operation status")
		}
		if attempt.Status != "prepared" && (len(attempt.Authorization) == 0 || bytes.Equal(attempt.Authorization, []byte("null"))) {
			return fmt.Errorf("source operation lacks authorization evidence")
		}
		if attempt.PriorAuthorizations == nil || len(attempt.PriorAuthorizations) > 64 {
			return fmt.Errorf("source authorization history is missing or full")
		}
		var transition struct {
			Protocol    string          `json:"protocol"`
			OperationID string          `json:"operation_id"`
			Expected    *string         `json:"expected_state_digest"`
			State       json.RawMessage `json:"state"`
			Digest      string          `json:"state_digest"`
		}
		var plan struct {
			OperationID string          `json:"operation_id"`
			Expected    *string         `json:"expected_state_digest"`
			Source      json.RawMessage `json:"source"`
		}
		if json.Unmarshal(attempt.Intent.Transition, &transition) != nil || json.Unmarshal(attempt.Intent.Plan, &plan) != nil || transition.Protocol != "symphony.scv.source-transition.v1" || transition.OperationID != id || plan.OperationID != id || !sameDigest(transition.Expected, plan.Expected) || !sameJSON(transition.State, plan.Source) {
			return fmt.Errorf("retained source transition does not bind exact intent")
		}
		var source retainedState
		if json.Unmarshal(transition.State, &source) != nil || source.SourceID != d.SourceID || source.Digest != transition.Digest || !sameDigest(source.Predecessor, transition.Expected) || source.Generation < 1 {
			return fmt.Errorf("retained source state binding mismatch")
		}
		if attempt.Status == "committed" {
			if _, duplicate := committed[source.Digest]; duplicate {
				return fmt.Errorf("duplicate committed source revision")
			}
			committed[source.Digest] = source
		}
	}
	// This is a structural history check, never a generation reducer: the C++
	// owner has already selected every generation and predecessor. Verify that
	// persisted links still form the single chain actually committed by CAS.
	predecessors := map[string]bool{}
	for _, source := range committed {
		key := "absent"
		if source.Predecessor != nil {
			key = *source.Predecessor
		}
		if predecessors[key] {
			return fmt.Errorf("committed source history forks")
		}
		predecessors[key] = true
		if source.Predecessor == nil {
			if source.Generation != 1 {
				return fmt.Errorf("source initial generation mismatch")
			}
		} else {
			prior, ok := committed[*source.Predecessor]
			if !ok || prior.Generation+1 != source.Generation || prior.SourceID != source.SourceID || prior.ProviderID != source.ProviderID || prior.FamilyID != source.FamilyID {
				return fmt.Errorf("source predecessor history mismatch")
			}
		}
	}
	if len(committed) > 0 && (d.StateDigest == nil || predecessors[*d.StateDigest]) {
		return fmt.Errorf("selected source head is not terminal committed revision")
	}
	return nil
}

func (t *Transaction) Current() json.RawMessage {
	return append(json.RawMessage(nil), t.document.Source...)
}
func (t *Transaction) Snapshot() Document { return cloneDocument(t.document) }
func (t *Transaction) Attempt(id string) (Attempt, bool) {
	value, ok := t.document.Operations[id]
	return value, ok
}

// Prepare persists the exact intent before any authorization request. It never
// changes the selected source. Reuse with a different plan/installation fails.
func (t *Transaction) Prepare(intent Intent) (bool, error) {
	prior, exists := t.Attempt(intent.OperationID)
	if exists {
		if prior.Intent.Digest != intent.Digest {
			return false, fmt.Errorf("operation_id already binds a different exact intent")
		}
		if prior.Status == "committed" {
			return true, nil
		}
	}
	if !exists && len(t.document.Operations) >= maxOperations {
		return false, fmt.Errorf("source operation history is full (%d entries)", maxOperations)
	}
	var transition struct {
		Expected *string `json:"expected_state_digest"`
	}
	if err := json.Unmarshal(intent.Transition, &transition); err != nil {
		return false, err
	}
	if !sameDigest(transition.Expected, t.document.StateDigest) {
		return false, fmt.Errorf("source compare-and-swap failed")
	}
	if exists && prior.CorrelationID != "" {
		return false, nil
	}
	correlation, err := AuthorizationCorrelation(intent.OperationID, exists)
	if err != nil {
		return false, err
	}
	next := cloneDocument(t.document)
	next.Protocol = protocol
	if !exists {
		prior = Attempt{Intent: intent, Status: "prepared", Authorization: json.RawMessage("null"), PriorAuthorizations: []json.RawMessage{}}
	}
	prior.CorrelationID = correlation
	next.Operations[intent.OperationID] = prior
	return false, t.save(next)
}

// AuthorizationCorrelation preserves an eligible preexisting operation UUID on
// legacy recovery. New attempts always receive a genuinely random UUIDv4; no
// content hash is disguised as a UUID. Callers must persist this before use.
func AuthorizationCorrelation(operationID string, legacy bool) (string, error) {
	if legacy && stavprotocol.ValidateRequestUUID(operationID) == nil {
		return operationID, nil
	}
	return stavprotocol.GenerateUUIDv4()
}

// Commit is called only after the command's authenticated SSIAG client has
// validated the exact decision. First save its real decision, then atomically
// change head and committed operation status in one fsynced document.
func (t *Transaction) Commit(id string, authorization json.RawMessage, stillValid func() error) error {
	attempt, ok := t.Attempt(id)
	if !ok {
		return fmt.Errorf("source intent is absent")
	}
	if attempt.Status == "committed" {
		return nil
	}
	if stavprotocol.ValidateRequestUUID(attempt.CorrelationID) != nil {
		return fmt.Errorf("source authorization correlation must be durably prepared")
	}
	if len(authorization) == 0 || bytes.Equal(authorization, []byte("null")) || stillValid == nil {
		return fmt.Errorf("authorization evidence is required")
	}
	var transition struct {
		Expected *string         `json:"expected_state_digest"`
		State    json.RawMessage `json:"state"`
		Digest   string          `json:"state_digest"`
	}
	if err := json.Unmarshal(attempt.Intent.Transition, &transition); err != nil {
		return err
	}
	if !sameDigest(transition.Expected, t.document.StateDigest) {
		return fmt.Errorf("source compare-and-swap failed")
	}
	if attempt.Status == "authorized" {
		if len(attempt.PriorAuthorizations) >= 64 {
			return fmt.Errorf("source authorization history is full")
		}
		attempt.PriorAuthorizations = append(append([]json.RawMessage{}, attempt.PriorAuthorizations...), append(json.RawMessage(nil), attempt.Authorization...))
	}
	attempt.Status = "authorized"
	attempt.Authorization = authorization
	next := cloneDocument(t.document)
	next.Operations[id] = attempt
	if err := t.save(next); err != nil {
		return err
	}
	attempt.Status = "committed"
	next = cloneDocument(t.document)
	next.Operations[id] = attempt
	next.Source = transition.State
	next.StateDigest = &transition.Digest
	next.HeadOperationID = &id
	t.publicationGuard = stillValid
	defer func() { t.publicationGuard = nil }()
	return t.save(next)
}

func sameDigest(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

func cloneDocument(value Document) Document {
	data, _ := json.Marshal(value)
	var result Document
	_ = json.Unmarshal(data, &result)
	return result
}
func sameJSON(a, b []byte) bool {
	var x, y any
	for _, item := range []struct {
		raw    []byte
		target *any
	}{{a, &x}, {b, &y}} {
		d := json.NewDecoder(bytes.NewReader(item.raw))
		d.UseNumber()
		if d.Decode(item.target) != nil {
			return false
		}
	}
	xb, xe := knowledgeengine.SCVCanonical(x)
	yb, ye := knowledgeengine.SCVCanonical(y)
	return xe == nil && ye == nil && bytes.Equal(xb, yb)
}

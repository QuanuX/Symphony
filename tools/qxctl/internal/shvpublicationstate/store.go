// Package shvpublicationstate is qxctl's protected filesystem transaction adapter. It does
// not derive head generations, manifest meaning, or transition semantics;
// those are supplied by the exact receipt-validated C++ head owner.
// Transaction sequencing is adapted from internal/scvstate/store.go; SHV
// journal identity, lineage validation and authorization checks remain owned here.
package shvpublicationstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"regexp"
)

const protocol = "symphony.qxctl.shv-publication-store.v1"
const maxStoreBytes = 16 * 1024 * 1024
const maxOperations = 128

// Reserve 8 KiB for the native process envelope, including its bounded cwd.
const maxHistoryPayloadBytes = (1 << 20) - 8192

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
var topsPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Intent binds an exact plan, head owner installation and reduced transition.
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
	// This SHV release requires a UUIDv4 correlation for every retained attempt.
	CorrelationID string `json:"correlation_id,omitempty"`
	// This is the actual SSIAG decision returned after its committed policy
	// audit, never a fabricated STAV head-write receipt.
	Authorization       json.RawMessage   `json:"authorization"`
	PriorAuthorizations []json.RawMessage `json:"prior_authorizations"`
}
type Document struct {
	Protocol        string             `json:"protocol"`
	TOPSID          string             `json:"tops_id"`
	CatalogueID     string             `json:"catalogue_id"`
	Head            json.RawMessage    `json:"head"`
	StateDigest     *string            `json:"state_digest"`
	HeadOperationID *string            `json:"head_operation_id"`
	Operations      map[string]Attempt `json:"operations"`
	Digest          string             `json:"digest"`
}
type Store struct{ Root, TOPSID, CatalogueID string }
type Transaction struct {
	store            Store
	document         Document
	save             func(Document) error
	publicationGuard func() error
}

func New(root, topsID, catalogueID string) (Store, error) {
	if !topsPattern.MatchString(topsID) || !tokenPattern.MatchString(catalogueID) || !cleanRoot(root) || root == "/" {
		return Store{}, fmt.Errorf("invalid protected SHV root/TOPS/head identity")
	}
	return Store{Root: root, TOPSID: topsID, CatalogueID: catalogueID}, nil
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
	if err != nil {
		return Intent{}, err
	}
	if err = validateIntentShape(intent); err != nil {
		return Intent{}, err
	}
	intent.Plan = append(json.RawMessage(nil), intent.Plan...)
	intent.Transition = append(json.RawMessage(nil), intent.Transition...)
	return intent, nil
}

func (s Store) WithLock(operation func(*Transaction) error) error {
	if _, err := New(s.Root, s.TOPSID, s.CatalogueID); err != nil {
		return err
	}
	return s.withDirectory(func(read func() ([]byte, error), write func([]byte, func() error) error) error {
		data, err := read()
		if err != nil {
			return err
		}
		document := Document{Protocol: protocol, TOPSID: s.TOPSID, CatalogueID: s.CatalogueID,
			Head: json.RawMessage("null"), Operations: map[string]Attempt{}}
		if data != nil {
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&document); err != nil {
				return fmt.Errorf("invalid head store: %w", err)
			}
			if err := validateDocument(document, s); err != nil {
				return err
			}
			canonical, err := knowledgeengine.SCVCanonical(document)
			if err != nil || !bytes.Equal(canonical, data) {
				return fmt.Errorf("head store encoding is not canonical")
			}
		}
		if data == nil {
			document.Digest, err = seal(document)
			if err != nil {
				return err
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
				return fmt.Errorf("head history is full; no implicit history pruning is permitted")
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

func (t *Transaction) Current() json.RawMessage {
	return append(json.RawMessage(nil), t.document.Head...)
}
func (t *Transaction) Snapshot() Document { return cloneDocument(t.document) }
func (t *Transaction) Attempt(id string) (Attempt, bool) {
	value, ok := cloneDocument(t.document).Operations[id]
	return value, ok
}
func (t *Transaction) History() []json.RawMessage {
	history, _ := documentHistory(t.document)
	return history
}

// Prepare persists the exact intent before any authorization request. It never
// changes the selected head. Reuse with a different plan/installation fails.
func (t *Transaction) Prepare(intent Intent) (bool, error) {
	if err := validateIntentShape(intent); err != nil {
		return false, err
	}
	prior, exists := t.Attempt(intent.OperationID)
	if exists {
		if prior.Intent.Digest != intent.Digest || !sameValue(prior.Intent, intent) {
			return false, fmt.Errorf("operation_id already binds a different exact intent")
		}
		if prior.Status == "committed" {
			return true, nil
		}
	}
	if !exists && len(t.document.Operations) >= maxOperations {
		return false, fmt.Errorf("head operation history is full (%d entries)", maxOperations)
	}
	var transition struct {
		Expected *string `json:"expected_state_digest"`
	}
	if err := json.Unmarshal(intent.Transition, &transition); err != nil {
		return false, err
	}
	if !sameDigest(transition.Expected, t.document.StateDigest) {
		return false, fmt.Errorf("head compare-and-swap failed")
	}
	prospective := t.History()
	var proposed retainedTransition
	if err := json.Unmarshal(intent.Transition, &proposed); err != nil {
		return false, err
	}
	prospective = append(prospective, proposed.Head)
	if err := validateHistoryRaw(prospective); err != nil {
		return false, err
	}

	if exists && prior.CorrelationID != "" {
		return false, nil
	}
	correlation, err := stavprotocol.GenerateUUIDv4()
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
	err = t.save(next)
	if err == nil {
		publicationBarrier("publication.after_prepare")
	}
	return false, err
}

// Commit is called only after the command's authenticated SSIAG client has
// validated the exact decision. First save its real decision, then atomically
// change head and committed operation status in one fsynced document.
func (t *Transaction) Commit(id string, authorization json.RawMessage, stillValid func() error) error {
	attempt, ok := t.Attempt(id)
	if !ok {
		return fmt.Errorf("head intent is absent")
	}
	if attempt.Status == "committed" {
		return nil
	}
	if stavprotocol.ValidateRequestUUID(attempt.CorrelationID) != nil {
		return fmt.Errorf("head authorization correlation must be durably prepared")
	}
	if len(authorization) == 0 || bytes.Equal(authorization, []byte("null")) || stillValid == nil {
		return fmt.Errorf("authorization evidence is required")
	}
	var transition struct {
		Expected *string         `json:"expected_state_digest"`
		Head     json.RawMessage `json:"head"`
	}
	if err := json.Unmarshal(attempt.Intent.Transition, &transition); err != nil {
		return err
	}
	if !sameDigest(transition.Expected, t.document.StateDigest) {
		return fmt.Errorf("head compare-and-swap failed")
	}
	canonical, err := canonicalAuthorization(authorization, t.store, attempt)
	if err != nil {
		return err
	}
	authorization = canonical
	if attempt.Status == "authorized" {
		if len(attempt.PriorAuthorizations) >= 64 {
			return fmt.Errorf("head authorization history is full")
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
	publicationBarrier("publication.after_authorization")
	attempt.Status = "committed"
	next = cloneDocument(t.document)
	next.Operations[id] = attempt
	next.Head = append(json.RawMessage(nil), transition.Head...)
	var head retainedHead
	if err := json.Unmarshal(transition.Head, &head); err != nil {
		return err
	}
	next.StateDigest = &head.Digest
	next.HeadOperationID = &id
	t.publicationGuard = func() error {
		if err := stillValid(); err != nil {
			return err
		}
		return authorizationFresh(authorization)
	}
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

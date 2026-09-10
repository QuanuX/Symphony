// Package scvgraph selects exact, already engine-validated graph artifacts.
// It owns atomic persistence and recovery, never provider or graph semantics.
package scvgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvstate"
	"regexp"
	"time"
)

const protocol = "symphony.qxctl.scv-graph-store.v1"
const maxBytes = 16 * 1024 * 1024

var token = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var tagged = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

type Intent struct {
	OperationID         string                       `json:"operation_id"`
	ExpectedGraphDigest *string                      `json:"expected_graph_digest"`
	ExpectedGeneration  int                          `json:"expected_generation"`
	Graph               json.RawMessage              `json:"graph"`
	GraphDigest         string                       `json:"graph_digest"`
	Installation        knowledgeengine.Installation `json:"installation"`
	Digest              string                       `json:"digest"`
}
type Attempt struct {
	Intent         Intent            `json:"intent"`
	Status         string            `json:"status"`
	Authorizations []json.RawMessage `json:"authorizations"`
}
type Document struct {
	Protocol        string             `json:"protocol"`
	TOPSID          string             `json:"tops_id"`
	Domain          string             `json:"domain"`
	GraphID         string             `json:"graph_id"`
	Generation      int                `json:"generation"`
	HeadOperationID *string            `json:"head_operation_id"`
	GraphDigest     *string            `json:"graph_digest"`
	Operations      map[string]Attempt `json:"operations"`
	Digest          string             `json:"digest"`
}
type Store struct{ Root, TOPSID, Domain, GraphID string }
type Authorization struct {
	Evidence   json.RawMessage
	ValidUntil time.Time
}
type Authorizer func(Intent) (Authorization, error)

func New(root, topsID, domain, graphID string) (Store, error) {
	if _, err := scvstate.New(root, topsID, domain, graphID); err != nil {
		return Store{}, err
	}
	return Store{root, topsID, domain, graphID}, nil
}
func selfDigest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var object map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err = d.Decode(&object); err != nil {
		return "", err
	}
	delete(object, "digest")
	return knowledgeengine.SCVDigest(object)
}
func graphDigest(raw json.RawMessage, domain string) (string, error) {
	var graph map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&graph); err != nil {
		return "", err
	}
	if graph["protocol"] != "symphony.scv.graph.v1" || graph["domain"] != domain {
		return "", fmt.Errorf("graph identity does not match selected domain")
	}
	expected, ok := graph["digest"].(string)
	if !ok || !tagged.MatchString(expected) {
		return "", fmt.Errorf("graph lacks digest")
	}
	actual, err := selfDigest(graph)
	if err != nil || expected != actual {
		return "", fmt.Errorf("graph content digest mismatch")
	}
	return actual, nil
}
func NewIntent(operationID string, expected *string, generation int, graph json.RawMessage, installation knowledgeengine.Installation) (Intent, error) {
	if !token.MatchString(operationID) || generation < 0 || generation >= 128 || expected != nil && !tagged.MatchString(*expected) || (expected == nil) != (generation == 0) {
		return Intent{}, fmt.Errorf("invalid graph operation or expected digest")
	}
	domain := installation.Role
	identity, err := graphDigest(graph, domain)
	if err != nil {
		return Intent{}, err
	}
	result := Intent{OperationID: operationID, ExpectedGraphDigest: expected, ExpectedGeneration: generation, Graph: graph, GraphDigest: identity, Installation: installation}
	result.Digest, err = selfDigest(result)
	return result, err
}
func same(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func copyDocument(d Document) Document {
	result := d
	result.Operations = make(map[string]Attempt, len(d.Operations))
	for id, attempt := range d.Operations {
		attempt.Authorizations = append([]json.RawMessage{}, attempt.Authorizations...)
		result.Operations[id] = attempt
	}
	return result
}
func (s Store) validate(d Document) error {
	if d.Protocol != protocol || d.TOPSID != s.TOPSID || d.Domain != s.Domain || d.GraphID != s.GraphID || d.Operations == nil || len(d.Operations) > 128 || d.Generation < 0 || d.Generation > 128 {
		return fmt.Errorf("graph store identity or bounds mismatch")
	}
	expected, err := selfDigest(d)
	if err != nil || expected != d.Digest {
		return fmt.Errorf("graph store digest mismatch")
	}
	committed := 0
	chain := map[int]Intent{}
	for id, a := range d.Operations {
		if id != a.Intent.OperationID || !token.MatchString(id) {
			return fmt.Errorf("graph intent identity mismatch")
		}
		if a.Intent.ExpectedGeneration < 0 || a.Intent.ExpectedGeneration >= 128 || a.Intent.ExpectedGeneration > d.Generation ||
			(a.Intent.ExpectedGraphDigest == nil) != (a.Intent.ExpectedGeneration == 0) ||
			a.Intent.ExpectedGraphDigest != nil && !tagged.MatchString(*a.Intent.ExpectedGraphDigest) || a.Intent.Installation.Role != s.Domain {
			return fmt.Errorf("invalid graph intent revision or owner")
		}
		digest, err := selfDigest(a.Intent)
		if err != nil || digest != a.Intent.Digest {
			return fmt.Errorf("graph intent digest mismatch")
		}
		gd, err := graphDigest(a.Intent.Graph, s.Domain)
		if err != nil || gd != a.Intent.GraphDigest {
			return fmt.Errorf("graph artifact identity mismatch")
		}
		if a.Status != "prepared" && a.Status != "authorized" && a.Status != "committed" {
			return fmt.Errorf("invalid graph attempt state")
		}
		if len(a.Authorizations) > 128 || a.Status != "prepared" && len(a.Authorizations) == 0 {
			return fmt.Errorf("graph attempt authorization evidence absent")
		}
		if a.Status == "committed" {
			committed++
			if _, exists := chain[a.Intent.ExpectedGeneration]; exists {
				return fmt.Errorf("graph publication history forks")
			}
			chain[a.Intent.ExpectedGeneration] = a.Intent
		}
	}
	if committed != d.Generation {
		return fmt.Errorf("graph selection generation mismatch")
	}
	var previous *string
	var lastOperation *string
	for generation := 0; generation < d.Generation; generation++ {
		entry, exists := chain[generation]
		if !exists || !same(entry.ExpectedGraphDigest, previous) {
			return fmt.Errorf("graph publication history is not contiguous")
		}
		previous = &entry.GraphDigest
		lastOperation = &entry.OperationID
	}
	if !same(previous, d.GraphDigest) || !same(lastOperation, d.HeadOperationID) {
		return fmt.Errorf("graph head is not the terminal publication")
	}
	if d.GraphDigest == nil {
		if d.HeadOperationID != nil || d.Generation != 0 {
			return fmt.Errorf("invalid absent graph head")
		}
	} else {
		if d.HeadOperationID == nil {
			return fmt.Errorf("graph head operation missing")
		}
		a, ok := d.Operations[*d.HeadOperationID]
		if !ok || a.Status != "committed" || a.Intent.GraphDigest != *d.GraphDigest {
			return fmt.Errorf("graph head lacks its committed artifact")
		}
	}
	return nil
}
func (s Store) transaction(operation func(*Document, func(Document, func() error) error) error) error {
	return scvstate.WithProjectionDocument(s.Root, s.TOPSID, s.Domain, s.GraphID, func(read func() ([]byte, error), write func([]byte, func() error) error) error {
		raw, err := read()
		if err != nil {
			return err
		}
		document := Document{Protocol: protocol, TOPSID: s.TOPSID, Domain: s.Domain, GraphID: s.GraphID, Operations: map[string]Attempt{}}
		if raw != nil {
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&document); err != nil {
				return err
			}
			if err := s.validate(document); err != nil {
				return err
			}
			canonical, err := knowledgeengine.SCVCanonical(document)
			if err != nil || !bytes.Equal(raw, canonical) {
				return fmt.Errorf("graph store encoding is not canonical")
			}
		}
		save := func(next Document, guard func() error) error {
			next.Digest, err = selfDigest(next)
			if err != nil {
				return err
			}
			if err := s.validate(next); err != nil {
				return err
			}
			encoded, err := knowledgeengine.SCVCanonical(next)
			if err != nil {
				return err
			}
			if len(encoded) > maxBytes {
				return fmt.Errorf("graph history is full; implicit pruning is prohibited")
			}
			if err := write(encoded, guard); err != nil {
				return err
			}
			document = next
			return nil
		}
		return operation(&document, save)
	})
}
func (s Store) Inspect() (Document, error) {
	var result Document
	err := s.transaction(func(d *Document, _ func(Document, func() error) error) error { result = copyDocument(*d); return nil })
	return result, err
}
func (d Document) SelectedGraph() json.RawMessage {
	if d.HeadOperationID == nil {
		return json.RawMessage("null")
	}
	return append(json.RawMessage{}, d.Operations[*d.HeadOperationID].Intent.Graph...)
}

// Select requires an external authenticated authorizer and a previously
// validated graph. Authorization is requested only after durable intent. A
// retry after a lost response never repeats an already committed selection.
func (s Store) Select(intent Intent, authorize Authorizer) (Document, error) {
	var result Document
	err := s.transaction(func(d *Document, save func(Document, func() error) error) error {
		expected, err := selfDigest(intent)
		if err != nil || expected != intent.Digest {
			return fmt.Errorf("invalid graph intent seal")
		}
		if intent.Installation.Role != s.Domain {
			return fmt.Errorf("graph owner installation differs from store")
		}
		if prior, exists := d.Operations[intent.OperationID]; exists {
			if prior.Intent.Digest != intent.Digest {
				return fmt.Errorf("operation_id binds a different graph intent")
			}
			if prior.Status == "committed" {
				result = copyDocument(*d)
				return nil
			}
		} else {
			if len(d.Operations) >= 128 {
				return fmt.Errorf("graph operation history is full")
			}
			if intent.ExpectedGeneration != d.Generation || !same(intent.ExpectedGraphDigest, d.GraphDigest) {
				return fmt.Errorf("graph compare-and-swap failed")
			}
			next := copyDocument(*d)
			next.Operations[intent.OperationID] = Attempt{Intent: intent, Status: "prepared", Authorizations: []json.RawMessage{}}
			if err := save(next, nil); err != nil {
				return err
			}
		}
		if intent.ExpectedGeneration != d.Generation || !same(intent.ExpectedGraphDigest, d.GraphDigest) {
			return fmt.Errorf("graph compare-and-swap failed")
		}
		if authorize == nil {
			return fmt.Errorf("graph selection requires an authenticated authorization callback")
		}
		authority, err := authorize(intent)
		if err != nil {
			return err
		}
		if !json.Valid(authority.Evidence) || bytes.Equal(authority.Evidence, []byte("null")) || authority.ValidUntil.IsZero() {
			return fmt.Errorf("graph authorization evidence missing")
		}
		next := copyDocument(*d)
		attempt := next.Operations[intent.OperationID]
		attempt.Status = "authorized"
		attempt.Authorizations = append(attempt.Authorizations, authority.Evidence)
		next.Operations[intent.OperationID] = attempt
		if err := save(next, nil); err != nil {
			return err
		}
		guard := func() error {
			if !time.Now().Before(authority.ValidUntil) {
				return fmt.Errorf("graph authority expired before publication; recover with fresh authorization")
			}
			return nil
		}
		next = copyDocument(*d)
		attempt = next.Operations[intent.OperationID]
		attempt.Status = "committed"
		next.Operations[intent.OperationID] = attempt
		next.HeadOperationID = &intent.OperationID
		next.GraphDigest = &intent.GraphDigest
		next.Generation++
		if err := save(next, guard); err != nil {
			return err
		}
		result = copyDocument(*d)
		return nil
	})
	return result, err
}

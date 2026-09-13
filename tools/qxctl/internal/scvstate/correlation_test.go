package scvstate

import (
	"bytes"
	"encoding/json"
	"errors"
	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"testing"
)

func correlationStatePath(t *testing.T, s Store) string {
	t.Helper()
	identity, err := knowledgeengineDigestIdentity(s.Domain, s.SourceID)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(s.Root, "symphony", "qxctl", "scv", "sources-v1", s.TOPSID, identity, "state.json")
}
func correlationRead(t *testing.T, s Store) []byte {
	t.Helper()
	raw, err := os.ReadFile(correlationStatePath(t, s))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func correlationWrite(t *testing.T, s Store, d Document) []byte {
	t.Helper()
	var err error
	d.Digest, err = seal(d)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := knowledgeengine.SCVCanonical(d)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(correlationStatePath(t, s), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return raw
}
func correlationLegacy(t *testing.T, s Store) Document {
	t.Helper()
	var d Document
	if err := json.Unmarshal(correlationRead(t, s), &d); err != nil {
		t.Fatal(err)
	}
	d.Protocol = legacyProtocol
	for id, a := range d.Operations {
		a.CorrelationID = ""
		d.Operations[id] = a
	}
	correlationWrite(t, s, d)
	return d
}
func correlationIntent(t *testing.T, base Intent, id string) Intent {
	t.Helper()
	var plan, transition map[string]any
	if err := json.Unmarshal(base.Plan, &plan); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(base.Transition, &transition); err != nil {
		t.Fatal(err)
	}
	plan["operation_id"], transition["operation_id"] = id, id
	p, _ := json.Marshal(plan)
	tr, _ := json.Marshal(transition)
	v, err := NewIntent(id, p, tr, base.Installation)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSourceCorrelationPreparedBeforeAuthorityAndStableOnRecovery(t *testing.T) {
	s, intent := fixture(t)
	var first string
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(intent); err != nil {
			return err
		}
		a, _ := tx.Attempt(intent.OperationID)
		first = a.CorrelationID
		if stavprotocol.ValidateRequestUUID(first) != nil || first == intent.OperationID || a.Intent.Digest != intent.Digest || tx.Snapshot().Protocol != protocol {
			t.Fatal("correlation did not bind independently of native intent")
		}
		var retained Document
		if err := json.Unmarshal(correlationRead(t, s), &retained); err != nil {
			return err
		}
		if retained.Operations[intent.OperationID].CorrelationID != first || retained.StateDigest != nil {
			t.Fatal("correlation not durable before authorization")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before := correlationRead(t, s)
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(intent); err != nil {
			return err
		}
		a, _ := tx.Attempt(intent.OperationID)
		if a.CorrelationID != first || !bytes.Equal(before, correlationRead(t, s)) {
			t.Fatal("recovery changed correlation or rewrote prepared state")
		}
		return tx.Commit(intent.OperationID, json.RawMessage(`{"fixture":"validated only in storage test"}`), func() error { return errors.New("expired before publication") })
	}); err == nil {
		t.Fatal("expired authorization committed")
	}
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(intent); err != nil {
			return err
		}
		a, _ := tx.Attempt(intent.OperationID)
		if a.CorrelationID != first || a.Status != "authorized" {
			t.Fatal("expiry lost stable correlation/history")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSourceCorrelationLegacyPendingMigrationPreservesIntentAndEvidence(t *testing.T) {
	for _, id := range []string{"opaque-operation", "d03594fd-9d9c-42b8-9471-a32eb2242271", "0199d5e1-9000-7001-8001-a32eb2242271"} {
		t.Run(id, func(t *testing.T) {
			s, original := fixture(t)
			intent := correlationIntent(t, original, id)
			if err := s.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(intent); return e }); err != nil {
				t.Fatal(err)
			}
			d := correlationLegacy(t, s)
			a := d.Operations[id]
			a.Status = "authorized"
			a.Authorization = json.RawMessage(`{"legacy_fixture":"preserve without fabricating validity"}`)
			d.Operations[id] = a
			before := correlationWrite(t, s, d)
			if err := s.WithLock(func(tx *Transaction) error {
				if tx.Snapshot().Protocol != legacyProtocol {
					t.Fatal("read migrated legacy state")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, correlationRead(t, s)) {
				t.Fatal("read changed legacy bytes")
			}
			if err := s.WithLock(func(tx *Transaction) error {
				if _, err := tx.Prepare(intent); err != nil {
					return err
				}
				after, _ := tx.Attempt(id)
				if tx.Snapshot().Protocol != protocol || after.Intent.Digest != a.Intent.Digest || !sameJSON(after.Intent.Plan, a.Intent.Plan) || !bytes.Equal(after.Authorization, a.Authorization) || after.Status != a.Status || stavprotocol.ValidateRequestUUID(after.CorrelationID) != nil {
					t.Fatal("migration rewrote legacy intent/evidence")
				}
				if stavprotocol.ValidateRequestUUID(id) == nil && after.CorrelationID != id {
					t.Fatal("eligible prior operation UUID changed")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSourceCorrelationLegacyCommittedReplayAndStalePendingDoNotWrite(t *testing.T) {
	s, pending := fixture(t)
	committed := correlationIntent(t, pending, "committed-other")
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(pending); err != nil {
			return err
		}
		if _, err := tx.Prepare(committed); err != nil {
			return err
		}
		return tx.Commit(committed.OperationID, json.RawMessage(`{"fixture":"legacy authorization"}`), func() error { return nil })
	}); err != nil {
		t.Fatal(err)
	}
	correlationLegacy(t, s)
	before := correlationRead(t, s)
	if err := s.WithLock(func(tx *Transaction) error {
		done, err := tx.Prepare(committed)
		if err != nil || !done {
			t.Fatalf("legacy committed replay: %v", err)
		}
		if _, err = tx.Prepare(pending); err == nil {
			t.Fatal("stale legacy pending intent migrated")
		}
		if !bytes.Equal(before, correlationRead(t, s)) {
			t.Fatal("replay/stale rejection rewrote legacy state")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSourceCorrelationRejectsInvalidUUIDAndLegacyField(t *testing.T) {
	for _, value := range []string{"opaque", "00000000-0000-0000-0000-000000000000", "d03594fd-9d9c-52b8-9471-a32eb2242271", "legacy-valid"} {
		t.Run(value, func(t *testing.T) {
			s, intent := fixture(t)
			if err := s.WithLock(func(tx *Transaction) error { _, e := tx.Prepare(intent); return e }); err != nil {
				t.Fatal(err)
			}
			var d Document
			if err := json.Unmarshal(correlationRead(t, s), &d); err != nil {
				t.Fatal(err)
			}
			a := d.Operations[intent.OperationID]
			a.CorrelationID = value
			if value == "legacy-valid" {
				d.Protocol = legacyProtocol
				a.CorrelationID = "d03594fd-9d9c-42b8-9471-a32eb2242271"
			}
			d.Operations[intent.OperationID] = a
			correlationWrite(t, s, d)
			if err := s.WithLock(func(*Transaction) error { return nil }); err == nil {
				t.Fatal("invalid correlation accepted under resealed journal")
			}
		})
	}
}

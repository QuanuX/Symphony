package scvstate

import (
	"encoding/json"
	"errors"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) (Store, Intent) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(root, "00000000-0000-4000-8000-000000000001", "schv-aws", "aws-docs")
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	source := map[string]any{"source_id": "aws-docs", "provider_id": "aws", "family_id": "schv", "digest": digest, "generation": 1, "predecessor_digest": nil}
	plan, _ := json.Marshal(map[string]any{"protocol": "symphony.scv.source-plan.v1", "operation_id": "change-1", "expected_state_digest": nil, "source": source})
	transition, _ := json.Marshal(map[string]any{"protocol": "symphony.scv.source-transition.v1", "operation_id": "change-1", "expected_state_digest": nil, "state_digest": digest, "state": source})
	intent, err := NewIntent("change-1", plan, transition, knowledgeengine.Installation{Role: "schv-aws", Version: "0.1.0-dev"})
	if err != nil {
		t.Fatal(err)
	}
	return s, intent
}

func TestSourceIntentAuthorizationCASAndRecovery(t *testing.T) {
	s, intent := fixture(t)
	denied := errors.New("permission denied")
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(intent); err != nil {
			return err
		}
		return denied
	}); !errors.Is(err, denied) {
		t.Fatal(err)
	}
	if err := s.WithLock(func(tx *Transaction) error {
		if string(tx.Current()) != "null" {
			t.Fatal("denied authorization changed source")
		}
		attempt, ok := tx.Attempt(intent.OperationID)
		if !ok || attempt.Status != "prepared" {
			t.Fatal("exact preauthorization intent was lost")
		}
		if err := tx.Commit(intent.OperationID, nil, nil); err == nil {
			t.Fatal("missing authorization accepted")
		}
		return tx.Commit(intent.OperationID, json.RawMessage(`{"test_fixture":"validated authorization decision"}`), func() error { return nil })
	}); err != nil {
		t.Fatal(err)
	}
	// A new process after a lost reply finds one committed operation and head.
	if err := s.WithLock(func(tx *Transaction) error {
		committed, err := tx.Prepare(intent)
		if err != nil || !committed {
			t.Fatalf("lost-reply replay: %v %v", committed, err)
		}
		prior := tx.Snapshot()
		if prior.HeadOperationID == nil || *prior.HeadOperationID != intent.OperationID {
			t.Fatal("head did not bind operation")
		}
		variant, err := NewIntent(intent.OperationID, json.RawMessage(`{"different":true}`), intent.Transition, intent.Installation)
		if err != nil {
			return err
		}
		if _, err := tx.Prepare(variant); err == nil {
			t.Fatal("operation payload collision accepted")
		}
		stale, err := NewIntent("change-2", intent.Plan, intent.Transition, intent.Installation)
		if err != nil {
			return err
		}
		if _, err := tx.Prepare(stale); err == nil {
			t.Fatal("stale source compare-and-swap accepted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSourceInterruptedAuthorizationRetainsOldHead(t *testing.T) {
	s, intent := fixture(t)
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(intent); err != nil {
			return err
		}
		actual := tx.save
		calls := 0
		tx.save = func(document Document) error {
			calls++
			if calls == 2 {
				return errors.New("injected before head replacement")
			}
			return actual(document)
		}
		if err := tx.Commit(intent.OperationID, json.RawMessage(`{"test_fixture":"authorized"}`), func() error { return nil }); err == nil {
			t.Fatal("injected commit interruption ignored")
		}
		attempt, _ := tx.Attempt(intent.OperationID)
		if attempt.Status != "authorized" {
			t.Fatal("failed write changed in-memory committed status")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.WithLock(func(tx *Transaction) error {
		if string(tx.Current()) != "null" {
			t.Fatal("interrupted write selected mixed state")
		}
		attempt, _ := tx.Attempt(intent.OperationID)
		if attempt.Status != "authorized" {
			t.Fatalf("status %s", attempt.Status)
		}
		if err := tx.Commit(intent.OperationID, json.RawMessage(`{"test_fixture":"fresh reauthorization"}`), func() error { return nil }); err != nil {
			return err
		}
		retained, _ := tx.Attempt(intent.OperationID)
		if len(retained.PriorAuthorizations) != 1 {
			t.Fatal("prior authorization evidence lost on recovery")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSourceStoreRejectsSymlinksHardlinksAndTampering(t *testing.T) {
	for _, mode := range []string{"symlink", "hardlink", "tamper"} {
		t.Run(mode, func(t *testing.T) {
			s, intent := fixture(t)
			if err := s.WithLock(func(tx *Transaction) error { _, err := tx.Prepare(intent); return err }); err != nil {
				t.Fatal(err)
			}
			identity, _ := knowledgeengineDigestIdentity(s.Domain, s.SourceID)
			path := filepath.Join(s.Root, "symphony", "qxctl", "scv", "sources-v1", s.TOPSID, identity, "state.json")
			if mode == "symlink" {
				target := path + ".retained"
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "hardlink" {
				if err := os.Link(path, path+".linked"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "tamper" {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				data = []byte(strings.Replace(string(data), "aws-docs", "aws-evil", 1))
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.WithLock(func(*Transaction) error { return nil }); err == nil {
				t.Fatal("unsafe retained state accepted")
			}
		})
	}
}

func TestSourceStoreSerializesAndRejectsEscapingRoot(t *testing.T) {
	s, _ := fixture(t)
	if err := s.WithLock(func(*Transaction) error {
		if err := s.WithLock(func(*Transaction) error { return nil }); err == nil {
			t.Fatal("concurrent lock accepted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "linked-root")
	if err := os.Symlink(s.Root, link); err != nil {
		t.Fatal(err)
	}
	s.Root = link
	if err := s.WithLock(func(*Transaction) error { return nil }); err == nil {
		t.Fatal("symlinked root accepted")
	}
}

func TestSourcePublicationRechecksAuthorizationExpiry(t *testing.T) {
	s, intent := fixture(t)
	if err := s.WithLock(func(tx *Transaction) error {
		if _, err := tx.Prepare(intent); err != nil {
			return err
		}
		guardCalls := 0
		err := tx.Commit(intent.OperationID, json.RawMessage(`{"test_fixture":"expires during durable write"}`), func() error {
			guardCalls++
			attempt, _ := tx.Attempt(intent.OperationID)
			if attempt.Status != "authorized" {
				t.Fatal("publication guard ran before durable authorization")
			}
			return errors.New("expired")
		})
		if err == nil || guardCalls != 1 {
			t.Fatalf("publication expiry guard failed: %v", err)
		}
		if string(tx.Current()) != "null" {
			t.Fatal("expired authorization published source")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.WithLock(func(tx *Transaction) error {
		attempt, _ := tx.Attempt(intent.OperationID)
		if attempt.Status != "authorized" || string(tx.Current()) != "null" {
			t.Fatal("expired publication lost recoverable evidence")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

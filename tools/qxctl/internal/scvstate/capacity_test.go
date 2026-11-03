package scvstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceCapacityPreservesAcknowledgedJournal(t *testing.T) {
	s, first := fixture(t)
	identity, err := knowledgeengineDigestIdentity(s.Domain, s.SourceID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Root, "symphony", "qxctl", "scv", "sources-v1", s.TOPSID, identity, "state.json")
	makeIntent := func(id string) (Intent, error) {
		var plan, transition map[string]any
		if err := json.Unmarshal(first.Plan, &plan); err != nil {
			return Intent{}, err
		}
		if err := json.Unmarshal(first.Transition, &transition); err != nil {
			return Intent{}, err
		}
		plan["operation_id"], transition["operation_id"] = id, id
		p, err := json.Marshal(plan)
		if err != nil {
			return Intent{}, err
		}
		tr, err := json.Marshal(transition)
		if err != nil {
			return Intent{}, err
		}
		return NewIntent(id, p, tr, first.Installation)
	}
	if err := s.WithLock(func(tx *Transaction) error {
		for n := 0; n < maxOperations; n++ {
			next, err := makeIntent(fmt.Sprintf("capacity-%d", n))
			if err != nil {
				return err
			}
			if _, err = tx.Prepare(next); err != nil {
				return err
			}
		}
		before, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		next, err := makeIntent("capacity-overflow")
		if err != nil {
			return err
		}
		if _, err = tx.Prepare(next); err == nil {
			t.Fatal("full source journal accepted another operation")
		}
		after, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(before) != string(after) {
			t.Fatal("capacity refusal rewrote acknowledged journal")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

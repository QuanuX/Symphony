package scvworkflow

import (
	"encoding/json"
	"errors"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compositionJournalFixture(t *testing.T) (Store, CompositionRun) {
	t.Helper()
	store := fixture(t)
	input := json.RawMessage(`{"operation_id":"one","pack_evaluations":[],"pack_evaluation_refs":[],"interpretation_refs":[],"knowledge_refs":[],"composition":{"query_time":"2026-09-11T00:00:00Z","requirements":[],"slots":[],"allowed_guarantee_changes":[],"counterfactuals":[],"bounds":{"max_candidates":1}},"prior_composition_ref":null}`)
	run, err := NewCompositionRun(store, knowledgeengine.Installation{Role: "scv", Version: "0.7.0-dev"}, input)
	if err != nil {
		t.Fatal(err)
	}
	return store, run
}
func TestCompositionWorkflowJournalRejectsOmittedNullAndReboundFields(t *testing.T) {
	store, run := compositionJournalFixture(t)
	run.Stages["explore"] = Checkpoint{InputRef: "sha256:" + strings.Repeat("a", 64)}
	raw, err := Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"omit_complete", "null_complete", "omit_result_ref", "null_result_ref", "omit_installation_field", "null_installation_field", "case_installation_field", "wrong_root", "wrong_tops", "out_of_order"} {
		t.Run(mode, func(t *testing.T) {
			v, e := Decode(raw)
			if e != nil {
				t.Fatal(e)
			}
			checkpoint := v["stages"].(map[string]any)["explore"].(map[string]any)
			inst := v["installation"].(map[string]any)
			switch mode {
			case "omit_complete":
				delete(v, "complete")
			case "null_complete":
				v["complete"] = nil
			case "omit_result_ref":
				delete(checkpoint, "result_ref")
			case "null_result_ref":
				checkpoint["result_ref"] = nil
			case "omit_installation_field":
				delete(inst, "ModuleID")
			case "null_installation_field":
				inst["ModuleID"] = nil
			case "case_installation_field":
				inst["moduleid"] = inst["ModuleID"]
				delete(inst, "ModuleID")
			case "wrong_root":
				v["root"] = store.Root + "/other"
			case "wrong_tops":
				v["tops_id"] = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			case "out_of_order":
				v["stages"].(map[string]any)["reassess"] = checkpoint
			}
			forged, e := Seal(v)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = ReadCompositionRun(forged, store, "one"); e == nil {
				t.Fatal("invalid journal accepted")
			}
		})
	}
}
func TestCompositionWorkflowStorePinsProgressAndRejectsUnsafePaths(t *testing.T) {
	store, run := compositionJournalFixture(t)
	first := "sha256:" + strings.Repeat("a", 64)
	second := "sha256:" + strings.Repeat("b", 64)
	err := store.WithComposition("one", true, func(s *Session, _ *CompositionRun) error {
		saved, e := s.SaveComposition(store, run)
		if e != nil {
			return e
		}
		saved.Stages["explore"] = Checkpoint{InputRef: first}
		saved, e = s.SaveComposition(store, saved)
		if e != nil {
			return e
		}
		changed := saved
		changed.Stages = map[string]Checkpoint{"explore": {InputRef: second}}
		if _, e = s.SaveComposition(store, changed); e == nil {
			t.Fatal("pending stage replaced")
		}
		if e = store.WithComposition("one", true, func(*Session, *CompositionRun) error { t.Fatal("lock not exclusive"); return nil }); e == nil {
			t.Fatal("concurrent mutation admitted")
		}
		saved.Stages["explore"] = Checkpoint{InputRef: first, ResultRef: second}
		saved.Complete = true
		_, e = s.SaveComposition(store, saved)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.WithComposition("one", false, func(_ *Session, r *CompositionRun) error {
		if r == nil || !r.Complete {
			t.Fatal("completed journal missing")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, "symphony", "qxctl", "scv", "workflows-v1", store.TOPSID, "composition-runs-v1", keyForOperation("one"), "run.json")
	target := filepath.Join(store.Root, "original")
	if err = os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err = store.WithComposition("one", false, func(*Session, *CompositionRun) error { return nil }); err == nil {
		t.Fatal("symlink journal accepted")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Link(target, path); err != nil {
		t.Fatal(err)
	}
	if err = store.WithComposition("one", false, func(*Session, *CompositionRun) error { return nil }); err == nil {
		t.Fatal("hardlink journal accepted")
	}
	missing := store
	missing.Root = filepath.Join(store.Root, "missing")
	if err = missing.WithComposition("one", false, func(*Session, *CompositionRun) error { return nil }); err == nil {
		t.Fatal("missing read accepted")
	}
	if _, err = os.Stat(missing.Root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("read created store")
	}
}

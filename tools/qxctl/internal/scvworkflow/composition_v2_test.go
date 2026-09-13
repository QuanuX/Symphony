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

func compositionJournalV2Fixture(t *testing.T) (Store, CompositionRunV2) {
	t.Helper()
	store := fixture(t)
	input := json.RawMessage(`{"protocol":"symphony.qxctl.scv-composition-workflow-request.v2","transport":"bundle","operation_id":"one","pack_evaluations":[],"pack_evaluation_refs":[],"interpretation_refs":[],"knowledge_refs":[],"composition":{"query_time":"2026-09-11T00:00:00Z","requirements":[],"slots":[],"allowed_guarantee_changes":[],"counterfactuals":[],"bounds":{"max_candidates":1}},"prior_composition_ref":null}`)
	run, err := NewCompositionRunV2(store, knowledgeengine.Installation{Role: "scv", Version: "0.10.0-dev"}, input)
	if err != nil {
		t.Fatal(err)
	}
	return store, run
}
func TestCompositionBundleWorkflowJournalRejectsOmittedNullAndReboundFields(t *testing.T) {
	store, run := compositionJournalV2Fixture(t)
	run.Stages["explore"] = Checkpoint{InputRef: "sha256:" + strings.Repeat("a", 64)}
	raw, err := Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"omit_complete", "null_complete", "omit_result_ref", "null_result_ref", "omit_installation_field", "null_installation_field", "case_installation_field", "wrong_root", "wrong_tops", "out_of_order", "transport", "request_protocol", "owner_version", "intent"} {
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
			case "transport":
				v["request"].(map[string]any)["transport"] = "inline"
			case "request_protocol":
				v["request"].(map[string]any)["protocol"] = "symphony.qxctl.scv-composition-workflow-request.v1"
			case "owner_version":
				inst["Version"] = "0.9.0-dev"
			case "intent":
				v["intent_digest"] = "sha256:" + strings.Repeat("b", 64)
			case "out_of_order":
				v["stages"].(map[string]any)["reassess"] = checkpoint
			}
			forged, e := Seal(v)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = ReadCompositionRunV2(forged, store, "one"); e == nil {
				t.Fatal("invalid journal accepted")
			}
		})
	}
}
func TestCompositionBundleWorkflowStorePinsProgressAndRejectsUnsafePaths(t *testing.T) {
	store, run := compositionJournalV2Fixture(t)
	first := "sha256:" + strings.Repeat("a", 64)
	second := "sha256:" + strings.Repeat("b", 64)
	err := store.WithCompositionV2("one", true, func(s *Session, _ *CompositionRunV2) error {
		saved, e := s.SaveCompositionV2(store, run)
		if e != nil {
			return e
		}
		saved.Stages["explore"] = Checkpoint{InputRef: first}
		saved, e = s.SaveCompositionV2(store, saved)
		if e != nil {
			return e
		}
		changed := saved
		changed.Stages = map[string]Checkpoint{"explore": {InputRef: second}}
		if _, e = s.SaveCompositionV2(store, changed); e == nil {
			t.Fatal("pending stage replaced")
		}
		if e = store.WithCompositionV2("one", true, func(*Session, *CompositionRunV2) error { t.Fatal("lock not exclusive"); return nil }); e == nil {
			t.Fatal("concurrent mutation admitted")
		}
		saved.Stages["explore"] = Checkpoint{InputRef: first, ResultRef: second}
		saved.Complete = true
		_, e = s.SaveCompositionV2(store, saved)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.WithCompositionV2("one", false, func(_ *Session, r *CompositionRunV2) error {
		if r == nil || !r.Complete {
			t.Fatal("completed journal missing")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, "symphony", "qxctl", "scv", "workflows-v1", store.TOPSID, "composition-runs-v2", keyForOperation("one"), "run.json")
	target := filepath.Join(store.Root, "original")
	if err = os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err = store.WithCompositionV2("one", false, func(*Session, *CompositionRunV2) error { return nil }); err == nil {
		t.Fatal("symlink journal accepted")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Link(target, path); err != nil {
		t.Fatal(err)
	}
	if err = store.WithCompositionV2("one", false, func(*Session, *CompositionRunV2) error { return nil }); err == nil {
		t.Fatal("hardlink journal accepted")
	}
	missing := store
	missing.Root = filepath.Join(store.Root, "missing")
	if err = missing.WithCompositionV2("one", false, func(*Session, *CompositionRunV2) error { return nil }); err == nil {
		t.Fatal("missing read accepted")
	}
	if _, err = os.Stat(missing.Root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("read created store")
	}
}

func TestCompositionBundleWorkflowRequestBoundsUnicodeAndIsolation(t *testing.T) {
	store, run := compositionJournalV2Fixture(t)
	if _, err := NewCompositionRun(store, run.Installation, run.Request); err == nil {
		t.Fatal("v1 journal reinterpreted v2 request")
	}
	for _, mode := range []string{"missing_transport", "extra", "too_many", "duplicate", "surrogate"} {
		t.Run(mode, func(t *testing.T) {
			v, _ := Decode(run.Request)
			switch mode {
			case "missing_transport":
				delete(v, "transport")
			case "extra":
				v["latest"] = true
			case "too_many":
				for i := 0; i < 17; i++ {
					v["pack_evaluations"] = append(v["pack_evaluations"].([]any), map[string]any{})
				}
			case "duplicate":
				v["knowledge_refs"] = []any{"sha256:" + strings.Repeat("a", 64), "sha256:" + strings.Repeat("a", 64)}
			case "surrogate":
				v["composition"].(map[string]any)["query_time"] = "REPLACE"
			}
			raw, _ := Canonical(v)
			if mode == "surrogate" {
				raw = []byte(strings.Replace(string(raw), "REPLACE", `\ud800`, 1))
			}
			if _, err := NewCompositionRunV2(store, run.Installation, raw); err == nil {
				t.Fatal("invalid request admitted")
			}
		})
	}
	raw, _ := Seal(run)
	bad := []byte(strings.Replace(string(raw), `"Prefix":""`, `"Prefix":"\ud800"`, 1))
	if _, err := ReadCompositionRunV2(bad, store, "one"); err == nil {
		t.Fatal("raw Unicode accepted")
	}
	if err := store.WithCompositionV2("one", true, func(s *Session, _ *CompositionRunV2) error { _, err := s.SaveCompositionV2(store, run); return err }); err != nil {
		t.Fatal(err)
	}
	if err := store.WithComposition("one", false, func(*Session, *CompositionRun) error { t.Fatal("v1 selected v2 journal"); return nil }); err == nil {
		t.Fatal("v1 namespace crossed")
	}
}

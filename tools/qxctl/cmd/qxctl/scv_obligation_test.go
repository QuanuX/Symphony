package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
	"testing"
)

func mockObligationLink(t *testing.T) (*workflowRunner, scvworkflow.Store, map[string]any) {
	t.Helper()
	r, s, _ := mockWorkflow(t)
	r.installation.Version = "0.7.0-dev"
	refs := []any{}
	for _, label := range []string{"before", "after"} {
		raw, err := r.artifact(s, "import", map[string]any{"operation": "composition_explore", "input": map[string]any{"fixture": label}, "result": nil})
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, workflowValue(t, raw)["digest"])
	}
	r.installation.Version = "0.8.0-dev"
	return r, s, map[string]any{"before_ref": refs[0], "after_ref": refs[1], "submissions": []any{map[string]any{"submission_id": "synthetic", "obligation_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000", "provenance": map[string]any{"kind": "observation", "producer": "Synthetic bookkeeping fixture", "reference": "fixture:unverified", "content_digest": nil, "recorded_at": "2026-09-11T00:00:00Z", "description": "No semantic or runtime proof asserted"}}}}
}
func TestSCVObligationLinkRetainsOriginalOwnersAndRecovers(t *testing.T) {
	for _, boundary := range []string{"obligation_result_retained", "obligation_link_retained"} {
		t.Run(boundary, func(t *testing.T) {
			r, s, input := mockObligationLink(t)
			original := r.owner
			calls := []string{}
			r.owner = func(inst knowledgeengine.Installation, op string, in any) (json.RawMessage, error) {
				calls = append(calls, inst.Version+":"+op)
				return original(inst, op, in)
			}
			once := true
			r.afterCheckpoint = func(point string) error {
				if once && point == boundary {
					once = false
					return errors.New("synthetic interruption")
				}
				return nil
			}
			if _, err := r.obligationLink(s, "retain", input); err == nil {
				t.Fatal("expected interruption")
			}
			r.afterCheckpoint = nil
			result, err := r.obligationLink(s, "retain", input)
			if err != nil {
				t.Fatal(err)
			}
			repeat, err := r.obligationLink(s, "retain", input)
			if err != nil || !scvworkflow.Same(result, repeat) {
				t.Fatal("retry changed relationship", err)
			}
			v := workflowValue(t, result)
			r.installation.Version = "0.1.0-dev"
			calls = nil
			shown, err := r.obligationLink(s, "show", map[string]any{"link_ref": v["link_ref"]})
			if err != nil || !scvworkflow.Same(result, shown) {
				t.Fatal("show changed original owners", err)
			}
			expected := []string{"0.7.0-dev:composition_explore", "0.7.0-dev:composition_explore", "0.8.0-dev:composition_followup"}
			if fmt.Sprint(calls) != fmt.Sprint(expected) {
				t.Fatal("owner substitution", calls)
			}
		})
	}
}
func TestSCVObligationLinkRejectsResealedReferenceSubstitution(t *testing.T) {
	r, s, input := mockObligationLink(t)
	raw, err := r.obligationLink(s, "retain", input)
	if err != nil {
		t.Fatal(err)
	}
	v := workflowValue(t, raw)
	for _, change := range []func(map[string]any){func(l map[string]any) { l["after_ref"] = l["before_ref"] }, func(l map[string]any) { l["root"] = "/different-root" }, func(l map[string]any) { l["tops_id"] = "019f0c3a-7b2d-7e11-8c12-0242ac120002" }, func(l map[string]any) {
		l["submissions_digest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	}, func(l map[string]any) { l["extra"] = true }} {
		link := workflowClone(t, v["link"].(map[string]any))
		change(link)
		wrapper, e := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": link})
		if e != nil {
			t.Fatal(e)
		}
		var ref string
		e = s.With("", true, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
			var err error
			ref, err = session.Put("payloads", wrapper)
			return err
		})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = r.obligationLink(s, "show", map[string]any{"link_ref": ref}); e == nil {
			t.Fatal("resealed substituted linkage accepted")
		}
	}
	inspection := r.inspect
	r.inspect = func(role, prefix, version string) (knowledgeengine.Installation, error) {
		inst, e := inspection(role, prefix, version)
		if version == "0.7.0-dev" {
			inst.ReceiptDigest = "changed"
		}
		return inst, e
	}
	if _, e := r.obligationLink(s, "show", map[string]any{"link_ref": v["link_ref"]}); e == nil {
		t.Fatal("original owner drift accepted")
	}
}
func TestSCVObligationLinkRejectsWrongKindsAndUnusedFields(t *testing.T) {
	r, s, input := mockObligationLink(t)
	wrong, err := r.artifact(s, "import", map[string]any{"operation": "knowledge_interpret", "input": map[string]any{"fixture": "wrong-kind"}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	changed := workflowClone(t, input)
	changed["before_ref"] = workflowValue(t, wrong)["digest"]
	if _, err = r.obligationLink(s, "retain", changed); err == nil {
		t.Fatal("wrong kind accepted")
	}
	changed = workflowClone(t, input)
	changed["extra"] = true
	if _, err = r.obligationLink(s, "retain", changed); err == nil {
		t.Fatal("unused input accepted")
	}
	r.installation.Version = "0.7.0-dev"
	if _, err = r.obligationLink(s, "retain", input); err == nil {
		t.Fatal("earlier release admitted new operation")
	}
}

func TestSCVObligationLinkRejectsNormalizedOwnerRecord(t *testing.T) {
	for _, key := range []string{"Role", "Version"} {
		t.Run(key, func(t *testing.T) {
			r, store, input := mockObligationLink(t)
			var forgedRef string
			err := store.With("", true, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
				raw, err := s.Get("records", input["before_ref"].(string))
				if err != nil {
					return err
				}
				object, err := scvworkflow.Decode(raw)
				if err != nil {
					return err
				}
				inst := object["installation"].(map[string]any)
				lower := map[string]string{"Role": "role", "Version": "version"}[key]
				inst[lower] = inst[key]
				delete(inst, key)
				altered, err := scvworkflow.Seal(object)
				if err != nil {
					return err
				}
				forgedRef, err = s.Put("records", altered)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			input["before_ref"] = forgedRef
			if _, err = r.obligationLink(store, "retain", input); err == nil {
				t.Fatal("record with a different typed representation under its seal admitted")
			}
		})
	}
}

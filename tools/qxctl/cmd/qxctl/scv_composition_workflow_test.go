package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These fixtures isolate journal/recovery behavior. The installed test below
// separately invokes real C++ owners and their independent Go consumers.
func mockCompositionWorkflow(t *testing.T) (*workflowRunner, scvworkflow.Store, map[string]any) {
	t.Helper()
	store := workflowStore(t)
	inst := knowledgeengine.Installation{Role: "scv", Version: "0.7.0-dev", Prefix: store.Root, ExecutableDigest: "fixture", ReceiptDigest: "receipt"}
	r := &workflowRunner{installation: inst}
	r.inspect = func(role, prefix, version string) (knowledgeengine.Installation, error) {
		v := inst
		v.Role = role
		v.Prefix = prefix
		v.Version = version
		return v, nil
	}
	r.owner = func(selected knowledgeengine.Installation, op string, input any) (json.RawMessage, error) {
		current, _ := r.inspect(selected.Role, selected.Prefix, selected.Version)
		if current != selected {
			return nil, fmt.Errorf("fixture exact owner changed")
		}
		protocol, ok := knowledgeengine.SCVResultProtocol(op)
		if !ok {
			return nil, fmt.Errorf("unknown fixture operation")
		}
		return scvworkflow.Seal(map[string]any{"protocol": protocol, "domain": selected.Role, "input": input})
	}
	pack, err := r.artifact(store, "import", map[string]any{"operation": "provider_pack_prepare", "input": map[string]any{"fixture": "authored pack"}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	prior, err := r.artifact(store, "import", map[string]any{"operation": "composition_explore", "input": map[string]any{"fixture": "prior exploration"}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"operation_id": "composition-one", "pack_evaluations": []any{map[string]any{"evaluation_id": "selected", "pack_ref": workflowValue(t, pack)["digest"], "evaluation": map[string]any{"captures": []any{}, "bindings": []any{}, "selection_policy": map[string]any{"fixture": "owner policy"}, "fixtures": []any{}}}}, "pack_evaluation_refs": []any{}, "interpretation_refs": []any{}, "knowledge_refs": []any{}, "composition": map[string]any{"query_time": "2026-09-11T00:00:00Z", "requirements": []any{}, "slots": []any{}, "allowed_guarantee_changes": []any{}, "counterfactuals": []any{}, "bounds": map[string]any{"max_candidates": 1}}, "prior_composition_ref": workflowValue(t, prior)["digest"]}
	return r, store, workflowClone(t, input)
}
func compositionRunPath(store scvworkflow.Store, id string) string {
	digest, _ := knowledgeengine.SCVDigest(map[string]any{"operation_id": id})
	return filepath.Join(store.Root, "symphony", "qxctl", "scv", "workflows-v1", store.TOPSID, "composition-runs-v1", strings.TrimPrefix(digest, "sha256:"), "run.json")
}
func TestSCVCompositionWorkflowRecoversEveryDurableBoundary(t *testing.T) {
	for _, boundary := range []string{"prepared", "pack:selected_input", "pack:selected_artifact", "pack:selected_result", "explore_input", "explore_artifact", "explore_result", "reassess_input", "reassess_artifact", "reassess_result", "complete"} {
		t.Run(boundary, func(t *testing.T) {
			r, store, input := mockCompositionWorkflow(t)
			fired := false
			owner := r.owner
			calls := 0
			r.owner = func(i knowledgeengine.Installation, op string, p any) (json.RawMessage, error) {
				calls++
				return owner(i, op, p)
			}
			r.afterCheckpoint = func(stage string) error {
				if stage == boundary && !fired {
					fired = true
					return errors.New("injected interruption")
				}
				return nil
			}
			if _, err := r.executeComposition(store, "run", input); err == nil || !fired {
				t.Fatal("checkpoint did not interrupt", err)
			}
			if boundary == "prepared" && calls != 0 {
				t.Fatal("owner invoked before durable intent")
			}
			before := calls
			status, err := r.executeComposition(store, "status", map[string]any{"operation_id": input["operation_id"]})
			if err != nil {
				t.Fatal(err)
			}
			if calls != before || workflowValue(t, status)["validation"] != "sealed_checkpoints_only" {
				t.Fatal("status executed owner or implied replay")
			}
			r.afterCheckpoint = nil
			recovered, err := r.executeComposition(store, "recover", map[string]any{"operation_id": input["operation_id"]})
			if err != nil {
				t.Fatal(err)
			}
			again, err := r.executeComposition(store, "run", input)
			if err != nil || !scvworkflow.Same(recovered, again) {
				t.Fatal("completed replay changed result", err)
			}
			result := workflowValue(t, recovered)
			if result["status"] != "complete" || result["composition_ref"] == nil || result["reassessment_ref"] == nil || len(result["pack_evaluation_refs"].(map[string]any)) != 1 {
				t.Fatal("incomplete references")
			}
			changed := workflowClone(t, input)
			changed["composition"].(map[string]any)["query_time"] = "2026-09-12T00:00:00Z"
			if _, err = r.executeComposition(store, "run", changed); err == nil {
				t.Fatal("operation identity rebound")
			}
			r.installation.ExecutableDigest = "replacement"
			if _, err = r.executeComposition(store, "recover", map[string]any{"operation_id": input["operation_id"]}); err == nil {
				t.Fatal("changed selected owner accepted")
			}
		})
	}
}
func TestSCVCompositionWorkflowStatusRejectsMatchedCheckpointSubstitution(t *testing.T) {
	r, store, input := mockCompositionWorkflow(t)
	if _, err := r.executeComposition(store, "run", input); err != nil {
		t.Fatal(err)
	}
	path := compositionRunPath(store, input["operation_id"].(string))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	run, err := scvworkflow.ReadCompositionRun(raw, store, input["operation_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	unrelated := json.RawMessage(`{"fixture":"internally consistent but not requested"}`)
	err = store.With("", true, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		payload, e := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": unrelated})
		if e != nil {
			return e
		}
		inputRef, e := s.Put("payloads", payload)
		if e != nil {
			return e
		}
		result, e := r.owner(r.installation, "composition_explore", unrelated)
		if e != nil {
			return e
		}
		record, e := scvworkflow.NewRecord("composition_explore", r.installation, unrelated, result)
		if e != nil {
			return e
		}
		bytes, e := scvworkflow.Canonical(record)
		if e != nil {
			return e
		}
		ref, e := s.Put("records", bytes)
		if e != nil {
			return e
		}
		run.Stages["explore"] = scvworkflow.Checkpoint{InputRef: inputRef, ResultRef: ref}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	forged, err := scvworkflow.Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, forged, 0o600); err != nil {
		t.Fatal(err)
	}
	r.owner = func(knowledgeengine.Installation, string, any) (json.RawMessage, error) {
		t.Fatal("status invoked owner")
		return nil, nil
	}
	if _, err = r.executeComposition(store, "status", map[string]any{"operation_id": input["operation_id"]}); err == nil {
		t.Fatal("matched unrelated checkpoint accepted")
	}
}
func TestSCVCompositionWorkflowPreservesPackOwnerAndCallerSelections(t *testing.T) {
	r, store, input := mockCompositionWorkflow(t)
	leaf := r.installation
	leaf.Role = "schv-aws"
	original := r.installation
	r.installation = leaf
	pack, err := r.artifact(store, "import", map[string]any{"operation": "provider_pack_prepare", "input": map[string]any{"fixture": "leaf pack"}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	r.installation = original
	input["pack_evaluations"].([]any)[0].(map[string]any)["pack_ref"] = workflowValue(t, pack)["digest"]
	called := false
	owner := r.owner
	r.owner = func(inst knowledgeengine.Installation, op string, p any) (json.RawMessage, error) {
		if op == "provider_pack_evaluate" {
			called = true
			if inst != leaf {
				t.Fatal("pack evaluated by substituted parent")
			}
		}
		return owner(inst, op, p)
	}
	result, err := r.executeComposition(store, "run", input)
	if err != nil || !called {
		t.Fatal(err)
	}
	run := workflowValue(t, result)["run"].(map[string]any)
	if !scvcorpusSameForTest(run["request"], input) {
		t.Fatal("caller request changed")
	}
	r.inspect = func(role, prefix, version string) (knowledgeengine.Installation, error) {
		v := original
		v.Role = role
		if role == leaf.Role {
			v.ExecutableDigest = "changed leaf"
		}
		return v, nil
	}
	if _, err = r.executeComposition(store, "recover", map[string]any{"operation_id": input["operation_id"]}); err == nil {
		t.Fatal("original leaf drift not rejected")
	}
	if _, err = r.executeComposition(store, "status", map[string]any{"operation_id": input["operation_id"]}); err != nil {
		t.Fatal("status required current owner", err)
	}
}
func scvcorpusSameForTest(a, b any) bool {
	x, _ := scvworkflow.Canonical(a)
	y, _ := scvworkflow.Canonical(b)
	return scvworkflow.Same(x, y)
}
func TestSCVCompositionWorkflowRejectsBoundsAndUnusedFields(t *testing.T) {
	_, _, input := mockCompositionWorkflow(t)
	for _, mode := range []string{"too_many", "duplicate", "unused_policy", "unexpected_stage_field"} {
		t.Run(mode, func(t *testing.T) {
			v := workflowClone(t, input)
			switch mode {
			case "too_many":
				for i := 0; i < 16; i++ {
					v["knowledge_refs"] = append(v["knowledge_refs"].([]any), fmt.Sprintf("sha256:%064x", i+1))
				}
			case "duplicate":
				v["pack_evaluations"] = append(v["pack_evaluations"].([]any), v["pack_evaluations"].([]any)[0])
			case "unused_policy":
				v["selection_policy"] = map[string]any{}
			case "unexpected_stage_field":
				v["pack_evaluations"].([]any)[0].(map[string]any)["current_owner"] = map[string]any{}
			}
			if _, err := scvworkflow.CompositionPlan(v); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}

func TestInstalledSCVCompositionWorkflowRetainsOriginalOwnersAndRecovers(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_WORKFLOW_PREFIX")
	oldPrefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_PREFIX")
	if prefix == "" || oldPrefix == "" {
		t.Skip("requires exact .7 workflow and preserved .6 package installations")
	}
	store := workflowStore(t)
	old, err := newWorkflowRunner(scvOptions{domain: "schv", prefix: oldPrefix, version: "0.6.0-dev", repository: store.Root}, true)
	if err != nil {
		t.Fatal(err)
	}
	current, err := newWorkflowRunner(scvOptions{domain: "scv", prefix: prefix, version: "0.7.0-dev", repository: store.Root}, true)
	if err != nil {
		t.Fatal(err)
	}
	policy := map[string]any{"policy_id": "workflow-fixture", "max_age_seconds": nil, "partial_capture": "exclude", "allowed_statement_kinds": []any{"documented_fact"}}
	fixture := map[string]any{"fixture_id": "empty-selection", "captures": []any{}, "bindings": []any{}, "selection_policy": policy}
	draft := map[string]any{"protocol": "symphony.scv.provider-pack.v1", "pack_id": "workflow-portable-provider", "pack_version": "fixture-1", "authored_by": "Synthetic fixture author", "provenance": []any{"No provider runtime facts asserted"}, "provider": map[string]any{"provider_id": "fixture-independent", "family_id": "schv", "display_name": "Synthetic unlisted provider", "sources": []any{map[string]any{"source_id": "docs", "provider_id": "fixture-independent", "family_id": "schv", "publisher": "Synthetic fixture", "authority_role": "user_declared", "scope": "Empty mapping test", "locators": []any{map[string]any{"locator_id": "docs", "uri": "https://example.invalid/docs", "role": "reference", "format": "json", "selector": "fixture-1"}}, "continuity_evidence": []any{}}}}, "profiles": []any{}, "structured_profiles": []any{}, "fixtures": []any{map[string]any{"fixture_id": "empty-selection", "label": "No facts from empty selection", "authored_by": "Independent expectation author", "rationale": "Zero captures yields zero claims", "input_digest": nil, "expected_claims": []any{}, "expected_extractions": []any{}}}}
	packed, err := old.artifact(store, "import", map[string]any{"operation": "provider_pack_prepare", "input": map[string]any{"pack": draft, "fixtures": []any{fixture}}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	packRecord := workflowValue(t, packed)
	evaluation := map[string]any{"captures": []any{}, "bindings": []any{}, "selection_policy": policy, "fixtures": []any{fixture}}
	resolution := map[string]any{"kind": "caller_decision", "reference": "fixture:missing-proof", "description": "Caller supplies missing evidence; never execute this reference"}
	requirement := map[string]any{"requirement_id": "capacity", "importance": "required", "operator": "gte", "right": map[string]any{"kind": "literal", "value": map[string]any{"type": "integer", "value": 1, "unit": "units"}}, "resolution": resolution}
	recipe := map[string]any{"recipe_id": "unimplemented", "provider_id": "fixture-independent", "bindings": []any{}, "prerequisites": []any{}, "requires_interfaces": []any{}, "supplies_interfaces": []any{}, "guarantee_changes": []any{}, "implementation": map[string]any{"status": "unimplemented", "reference": nil}, "resolution": resolution}
	composition := map[string]any{"query_time": "2026-09-11T00:00:00Z", "requirements": []any{requirement}, "slots": []any{map[string]any{"slot_id": "compute", "allowed_provider_ids": []any{"fixture-independent"}, "recipes": []any{recipe}}}, "allowed_guarantee_changes": []any{}, "counterfactuals": []any{}, "bounds": map[string]any{"max_candidates": 1}}
	input := workflowClone(t, map[string]any{"operation_id": "installed-composition", "pack_evaluations": []any{map[string]any{"evaluation_id": "original-provider", "pack_ref": packRecord["digest"], "evaluation": evaluation}}, "pack_evaluation_refs": []any{}, "interpretation_refs": []any{}, "knowledge_refs": []any{}, "composition": composition, "prior_composition_ref": nil})
	interrupted := false
	current.afterCheckpoint = func(stage string) error {
		if stage == "pack:original-provider_result" && !interrupted {
			interrupted = true
			return errors.New("interrupted after real owner result")
		}
		return nil
	}
	if _, err = current.executeComposition(store, "run", input); err == nil || !interrupted {
		t.Fatal("real checkpoint did not interrupt", err)
	}
	status, err := current.executeComposition(store, "status", map[string]any{"operation_id": input["operation_id"]})
	if err != nil {
		t.Fatal(err)
	}
	if workflowValue(t, status)["status"] != "evaluating_packs" {
		t.Fatal("status misreported progress")
	}
	current.afterCheckpoint = nil
	done, err := current.executeComposition(store, "recover", map[string]any{"operation_id": input["operation_id"]})
	if err != nil {
		t.Fatal(err)
	}
	value := workflowValue(t, done)
	err = store.With("", false, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		pack, err := current.reference(s, value["pack_evaluation_refs"].(map[string]any)["original-provider"].(string), "provider_pack_evaluation", true)
		if err != nil {
			return err
		}
		if pack.Installation != old.installation {
			t.Fatal("pack original .6 SCHV owner replaced")
		}
		result, err := current.reference(s, value["composition_ref"].(string), "composition", true)
		if err != nil {
			return err
		}
		if result.Installation != current.installation {
			t.Fatal("composition exact .7 owner lost")
		}
		native := workflowValue(t, result.Artifact)
		candidate := native["scenarios"].([]any)[0].(map[string]any)["candidates"].([]any)[0].(map[string]any)
		if candidate["status"] != "unresolved" || candidate["implementation_status"] != "unimplemented" {
			t.Fatal("workflow invented compatibility")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	later := workflowClone(t, input)
	later["operation_id"] = "installed-reassessment"
	later["prior_composition_ref"] = value["composition_ref"]
	later["composition"].(map[string]any)["query_time"] = "2026-09-11T00:00:01Z"
	newer, err := current.executeComposition(store, "run", later)
	if err != nil {
		t.Fatal(err)
	}
	if workflowValue(t, newer)["reassessment_ref"] == nil {
		t.Fatal("requested reassessment not retained")
	}
	priorOwner := current.owner
	current.owner = func(knowledgeengine.Installation, string, any) (json.RawMessage, error) {
		t.Fatal("status invoked native owner")
		return nil, nil
	}
	if _, err = current.executeComposition(store, "status", map[string]any{"operation_id": later["operation_id"]}); err != nil {
		t.Fatal(err)
	}
	current.owner = priorOwner
	again, err := current.executeComposition(store, "run", later)
	if err != nil || !scvworkflow.Same(newer, again) {
		t.Fatal("completed recovery changed result", err)
	}
}

func TestSCVCompositionWorkflowCommandMetadata(t *testing.T) {
	group := newSCVCompositionWorkflowCommand()
	for _, leaf := range []string{"run", "status", "recover"} {
		command, _, err := group.Find([]string{leaf})
		if err != nil {
			t.Fatal(err)
		}
		spec, err := commandregistry.Spec(command)
		if err != nil {
			t.Fatal(err)
		}
		if spec.CommandID != "qxcmd:symphony:scv.composition.workflow."+leaf {
			t.Fatal("unstable command identity")
		}
		if leaf == "status" {
			if command.Flags().Lookup("version") != nil || spec.Mutability != "read_only" || len(spec.BackendOperationIDs) != 0 {
				t.Fatal("status requires native owner or mutation")
			}
		} else {
			if command.Flags().Lookup("version").DefValue != "0.7.0-dev" || spec.RecoveryCommandID == nil || *spec.RecoveryCommandID != "qxcmd:symphony:scv.composition.workflow.recover" {
				t.Fatal("wrong release/recovery mapping")
			}
			if len(spec.BackendOperationIDs) != 6*len(knowledgeengine.SCVDomains()) {
				t.Fatal("missing backend operation inventory")
			}
		}
	}
}

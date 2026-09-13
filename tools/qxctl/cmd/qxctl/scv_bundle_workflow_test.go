package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
)

// These lifecycle tests use previously emitted native semantic fixtures. Only
// the transport owner/version and canonical envelope are reconstructed; every
// retained logical view still crosses the independent production consumer.
func mockBundleWorkflow(t *testing.T) (*workflowRunner, scvworkflow.Store, map[string]any) {
	t.Helper()
	raw, err := os.ReadFile("../../internal/knowledgeengine/testdata/scv-bundle.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err = decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	entries := fixture["operations"].([]any)
	explore, reassess := entries[0].(map[string]any), entries[1].(map[string]any)
	before := reassess["logical_input"].(map[string]any)["before"].(map[string]any)
	after := reassess["logical_input"].(map[string]any)["after"].(map[string]any)
	afterInput := after["input"].(map[string]any)
	store := workflowStore(t)
	installed := knowledgeengine.Installation{Role: "scv", Version: "0.10.0-dev", Prefix: store.Root, ExecutableDigest: "fixture-executable", ReceiptDigest: "fixture-receipt"}
	old := installed
	old.Version = "0.7.0-dev"
	r := &workflowRunner{installation: installed}
	r.inspect = func(role, prefix, version string) (knowledgeengine.Installation, error) {
		v := installed
		v.Role = role
		v.Prefix = prefix
		v.Version = version
		return v, nil
	}
	knowledge := afterInput["additional_knowledge"].([]any)[0].(map[string]any)
	kinput := map[string]any{"captures": knowledge["captures"], "claims": knowledge["claims"], "interpreter_version": knowledge["interpreter_version"], "selection_policy": knowledge["selection_policy"]}
	r.owner = func(owner knowledgeengine.Installation, operation string, input any) (json.RawMessage, error) {
		current, e := r.inspect(owner.Role, owner.Prefix, owner.Version)
		if e != nil || current != owner {
			return nil, fmt.Errorf("original fixture owner changed")
		}
		encoded, e := scvworkflow.Canonical(input)
		if e != nil {
			return nil, e
		}
		if operation == "knowledge_interpret" {
			expected, _ := scvworkflow.Canonical(kinput)
			if !scvworkflow.Same(encoded, expected) {
				return nil, fmt.Errorf("changed fixture knowledge input")
			}
			return scvworkflow.Canonical(knowledge)
		}
		if operation == "composition_explore" {
			expected, _ := scvworkflow.Canonical(before["input"])
			if !scvworkflow.Same(encoded, expected) {
				return nil, fmt.Errorf("changed fixture direct input")
			}
			return scvworkflow.Canonical(before)
		}
		if operation != "composition_bundle_evaluate" {
			return nil, fmt.Errorf("unexpected fixture owner operation")
		}
		invocation := workflowValue(t, encoded)
		bundleRaw, e := scvworkflow.Canonical(invocation["bundle"])
		if e != nil {
			return nil, e
		}
		logical, e := knowledgeengine.SCVBundleDecode(bundleRaw)
		if e != nil {
			return nil, e
		}
		source := explore
		native := any(after)
		expected := any(afterInput)
		if invocation["operation"] == "composition_reassess" {
			source = reassess
			native = reassess["logical_result"]
			expected = reassess["logical_input"]
		}
		expectedRaw, _ := scvworkflow.Canonical(expected)
		if !scvworkflow.Same(logical, expectedRaw) {
			return nil, fmt.Errorf("changed fixture logical input")
		}
		result := workflowClone(t, source["result"].(map[string]any))
		nativeRaw, _ := scvworkflow.Canonical(native)
		outputBundle, e := knowledgeengine.SCVBundleEncode(nativeRaw)
		if e != nil {
			return nil, e
		}
		result["owner"] = invocation["owner"]
		result["input_digest"], e = knowledgeengine.SCVDigest(invocation)
		if e != nil {
			return nil, e
		}
		result["input_root_digest"] = invocation["bundle"].(map[string]any)["root_digest"]
		result["result_bundle"] = json.RawMessage(outputBundle)
		result["native_result_digest"] = native.(map[string]any)["digest"]
		sealed, e := scvworkflow.Seal(result)
		if e != nil {
			return nil, e
		}
		if e = knowledgeengine.ValidateSCVResult(operation, encoded, sealed); e != nil {
			return nil, e
		}
		return sealed, nil
	}
	retain := func(op string, owner knowledgeengine.Installation, input, artifact any) string {
		a, _ := scvworkflow.Canonical(input)
		b, _ := scvworkflow.Canonical(artifact)
		record, e := scvworkflow.NewRecord(op, owner, a, b)
		if e != nil {
			t.Fatal(e)
		}
		bytes, e := scvworkflow.Canonical(record)
		if e != nil {
			t.Fatal(e)
		}
		var ref string
		e = store.With("", true, func(s *scvworkflow.Session, _ *scvworkflow.Run) error { ref, e = s.Put("records", bytes); return e })
		if e != nil {
			t.Fatal(e)
		}
		return ref
	}
	priorRef := retain("composition_explore", old, before["input"], before)
	knowledgeRef := retain("knowledge_interpret", old, kinput, knowledge)
	composition := map[string]any{}
	for _, key := range []string{"query_time", "requirements", "slots", "allowed_guarantee_changes", "counterfactuals", "bounds"} {
		composition[key] = afterInput[key]
	}
	request := map[string]any{"protocol": "symphony.qxctl.scv-composition-workflow-request.v2", "transport": "bundle", "operation_id": "bundle-workflow", "pack_evaluations": []any{}, "pack_evaluation_refs": []any{}, "interpretation_refs": []any{}, "knowledge_refs": []any{knowledgeRef}, "composition": composition, "prior_composition_ref": priorRef}
	return r, store, workflowClone(t, request)
}
func bundleWorkflowRunPath(store scvworkflow.Store, id string) string {
	return strings.Replace(compositionRunPath(store, id), "composition-runs-v1", "composition-runs-v2", 1)
}

func TestSCVBundleWorkflowRecoversEveryCompositionBoundary(t *testing.T) {
	for _, boundary := range []string{"prepared", "explore_input", "explore_artifact", "explore_result", "reassess_input", "reassess_artifact", "reassess_result", "complete"} {
		t.Run(boundary, func(t *testing.T) {
			r, store, input := mockBundleWorkflow(t)
			fired := false
			calls := 0
			owner := r.owner
			r.owner = func(i knowledgeengine.Installation, op string, p any) (json.RawMessage, error) {
				calls++
				return owner(i, op, p)
			}
			r.afterCheckpoint = func(stage string) error {
				if stage == boundary && !fired {
					fired = true
					return errors.New("durable interruption")
				}
				return nil
			}
			if _, err := r.executeBundleComposition(store, "run", input); err == nil || !fired {
				t.Fatal("boundary did not interrupt", err)
			}
			if boundary == "prepared" && calls != 0 {
				t.Fatal("owner before intent")
			}
			saved := calls
			status, err := r.executeBundleComposition(store, "status", map[string]any{"operation_id": input["operation_id"]})
			if err != nil {
				t.Fatal(err)
			}
			if calls != saved || workflowValue(t, status)["validation"] != "sealed_checkpoints_only" {
				t.Fatal("status invoked owner or overstated evidence")
			}
			r.afterCheckpoint = nil
			recovered, err := r.executeBundleComposition(store, "recover", map[string]any{"operation_id": input["operation_id"]})
			if err != nil {
				t.Fatal(err)
			}
			again, err := r.executeBundleComposition(store, "run", input)
			if err != nil || !scvworkflow.Same(recovered, again) {
				t.Fatal("replay changed pinned result", err)
			}
			result := workflowValue(t, recovered)
			if result["status"] != "complete" {
				t.Fatal("incomplete")
			}
			for _, key := range []string{"composition", "reassessment"} {
				descriptor := result[key].(map[string]any)
				refKey := "composition_ref"
				if key == "reassessment" {
					refKey = "reassessment_ref"
				}
				if descriptor["record_ref"] != result[refKey] || descriptor["transport"] == nil {
					t.Fatal("missing exact logical/transport descriptor")
				}
			}
			changed := workflowClone(t, input)
			changed["composition"].(map[string]any)["query_time"] = "2026-09-12T00:00:00Z"
			if _, err = r.executeBundleComposition(store, "run", changed); err == nil {
				t.Fatal("intent rebound")
			}
			r.installation.ReceiptDigest = "replacement"
			if _, err = r.executeBundleComposition(store, "recover", map[string]any{"operation_id": input["operation_id"]}); err == nil {
				t.Fatal("replacement owner accepted")
			}
		})
	}
}

func TestSCVBundleWorkflowRejectsSubstitutionTransportAndOwnerDrift(t *testing.T) {
	r, store, input := mockBundleWorkflow(t)
	if _, err := r.executeBundleComposition(store, "run", input); err != nil {
		t.Fatal(err)
	}
	path := bundleWorkflowRunPath(store, input["operation_id"].(string))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	run, err := scvworkflow.ReadCompositionRunV2(raw, store, input["operation_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	// Copy a fully sealed reassessment input/result pair into explore. Structural
	// journal seals remain valid; the pinned explore inputs must still reject it.
	run.Stages["explore"] = run.Stages["reassess"]
	forged, err := scvworkflow.Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, forged, 0600); err != nil {
		t.Fatal(err)
	}
	r.owner = func(knowledgeengine.Installation, string, any) (json.RawMessage, error) {
		t.Fatal("status invoked owner")
		return nil, nil
	}
	if _, err = r.executeBundleComposition(store, "status", map[string]any{"operation_id": input["operation_id"]}); err == nil {
		t.Fatal("matched unrelated pair accepted")
	}
	for _, mode := range []string{"inline", "absent", "protocol"} {
		v := workflowClone(t, input)
		switch mode {
		case "inline":
			v["transport"] = "inline"
		case "absent":
			delete(v, "transport")
		case "protocol":
			v["protocol"] = "symphony.qxctl.scv-composition-workflow-request.v1"
		}
		if _, err = r.executeBundleComposition(store, "run", v); err == nil {
			t.Fatal("implicit transport admitted")
		}
	}
	r, store, input = mockBundleWorkflow(t)
	if _, err = r.executeBundleComposition(store, "run", input); err != nil {
		t.Fatal(err)
	}
	r.inspect = func(string, string, string) (knowledgeengine.Installation, error) {
		return knowledgeengine.Installation{}, errors.New("original owner unavailable")
	}
	if _, err = r.executeBundleComposition(store, "recover", map[string]any{"operation_id": input["operation_id"]}); err == nil {
		t.Fatal("missing original owner accepted")
	}
	if _, err = r.executeBundleComposition(store, "status", map[string]any{"operation_id": input["operation_id"]}); err != nil {
		t.Fatal("status required owner", err)
	}
}

func TestSCVBundleWorkflowCommandMetadata(t *testing.T) {
	group := newSCVBundleWorkflowCommand()
	for _, leaf := range []string{"run", "status", "recover"} {
		c, _, err := group.Find([]string{leaf})
		if err != nil {
			t.Fatal(err)
		}
		spec, err := commandregistry.Spec(c)
		if err != nil {
			t.Fatal(err)
		}
		if spec.CommandID != "qxcmd:symphony:scv.composition.bundle.workflow."+leaf || spec.OutputProtocols[0] != "symphony.qxctl.scv-composition-workflow-result.v2" {
			t.Fatal("unstable metadata")
		}
		if leaf == "status" {
			if c.Flags().Lookup("version") != nil || len(spec.BackendOperationIDs) != 0 || spec.Mutability != "read_only" {
				t.Fatal("status owner dependency")
			}
		} else if c.Flags().Lookup("version").DefValue != "0.10.0-dev" || len(spec.BackendOperationIDs) != 6*len(knowledgeengine.SCVDomains()) {
			t.Fatal("wrong release/backend inventory")
		}
		if leaf == "run" && spec.InputProtocols[0] != "symphony.qxctl.scv-composition-workflow-request.v2" {
			t.Fatal("wrong request protocol")
		}
	}
}

func TestSCVBundleWorkflowRejectsRawUnicodeAcrossRoutesAndLegacyReferences(t *testing.T) {
	r, store, input := mockBundleWorkflow(t)
	raw, _ := scvworkflow.Canonical(input)
	raw = []byte(strings.Replace(string(raw), "2026-09-10T00:00:01Z", `\ud800`, 1))
	path := filepath.Join(store.Root, "input.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"run", "status", "recover"} {
		err := runSCVBundleWorkflow(action, scvOptions{input: path, topsID: store.TOPSID, version: "0.10.0-dev"}, store.Root)
		if err == nil || !strings.Contains(err.Error(), "surrogate") {
			t.Fatalf("%s lost raw Unicode: %v", action, err)
		}
	}
	// A legacy record's U+FFFD and an unpaired surrogate normalize identically
	// under encoding/json. V2 must reject the latter before using that record.
	inst := r.installation
	inst.Version = "0.6.0-dev"
	result, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.scv.provider-pack.v1", "note": "�"})
	if err != nil {
		t.Fatal(err)
	}
	record, err := scvworkflow.NewRecord("provider_pack_prepare", inst, json.RawMessage(`{"note":"�"}`), result)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := scvworkflow.Canonical(record)
	if err != nil {
		t.Fatal(err)
	}
	var ref string
	if err = store.With("", true, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		var e error
		ref, e = s.Put("records", bytes)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	malformed := []byte(strings.ReplaceAll(string(bytes), "�", `\ud800`))
	path = filepath.Join(store.Root, "symphony", "qxctl", "scv", "workflows-v1", store.TOPSID, "records", strings.TrimPrefix(ref, "sha256:")+".json")
	if err = os.WriteFile(path, malformed, 0600); err != nil {
		t.Fatal(err)
	}
	if err = store.With("", false, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		_, e := r.compositionBundleReference(s, ref, "provider_pack", false)
		return e
	}); err == nil {
		t.Fatal("legacy reference normalized invalid Unicode")
	}
}

func TestSCVBundleWorkflowOutputEnvelopeKeepsDefaultBounds(t *testing.T) {
	r, store, _ := mockBundleWorkflow(t)
	request := map[string]any{"values": []any{}}
	for i := 0; i < 32710; i++ {
		request["values"] = append(request["values"].([]any), nil)
	}
	raw, _ := scvworkflow.Canonical(request)
	if err := knowledgeengine.ValidateJSONObject(raw, scvworkflow.MaxInputBytes); err != nil {
		t.Fatal("fixture child must fit", err)
	}
	run := scvworkflow.CompositionRunV2{Protocol: "symphony.qxctl.scv-composition-workflow-run.v2", OperationID: "envelope", Root: store.Root, TOPSID: store.TOPSID, Installation: r.installation, Request: raw, Stages: map[string]scvworkflow.Checkpoint{}}
	journal, err := scvworkflow.Canonical(run)
	if err != nil {
		t.Fatal(err)
	}
	if err = knowledgeengine.ValidateJSONObject(journal, scvworkflow.MaxBytes); err != nil {
		t.Fatal("fixture journal must still fit", err)
	}
	// The journal contains ordinary metadata and its result adds more. Neither
	// gets the separately scoped two-child immutable-record allowance.
	if _, err := r.bundleCompositionWorkflowResult(nil, run, "sealed_checkpoints_only"); err == nil {
		t.Fatal("workflow result silently widened default value budget")
	}
}
func TestInstalledSCVBundleWorkflowOriginalOwnersAndEveryDurableBoundary(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_BUNDLE_WORKFLOW_PREFIX")
	oldPrefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_PREFIX")
	if prefix == "" || oldPrefix == "" {
		t.Skip("requires exact .10 bundle workflow and preserved .6 package installations")
	}
	store := workflowStore(t)
	old, err := newWorkflowRunner(scvOptions{domain: "schv", prefix: oldPrefix, version: "0.6.0-dev", repository: store.Root}, true)
	if err != nil {
		t.Fatal(err)
	}
	current, err := newWorkflowRunner(scvOptions{domain: "scv", prefix: prefix, version: "0.10.0-dev", repository: store.Root}, true)
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
	input := workflowClone(t, map[string]any{"protocol": "symphony.qxctl.scv-composition-workflow-request.v2", "transport": "bundle", "operation_id": "installed-composition", "pack_evaluations": []any{map[string]any{"evaluation_id": "original-provider", "pack_ref": packRecord["digest"], "evaluation": evaluation}}, "pack_evaluation_refs": []any{}, "interpretation_refs": []any{}, "knowledge_refs": []any{}, "composition": composition, "prior_composition_ref": nil})
	baseline, err := current.executeBundleComposition(store, "run", input)
	if err != nil {
		t.Fatal(err)
	}
	input["prior_composition_ref"] = workflowValue(t, baseline)["composition_ref"]
	input["composition"].(map[string]any)["query_time"] = "2026-09-11T00:00:01Z"
	for n, boundary := range []string{"prepared", "pack:original-provider_input", "pack:original-provider_artifact", "pack:original-provider_result", "explore_input", "explore_artifact", "explore_result", "reassess_input", "reassess_artifact", "reassess_result", "complete"} {
		t.Run(boundary, func(t *testing.T) {
			request := workflowClone(t, input)
			request["operation_id"] = fmt.Sprintf("installed-bundle-%d", n)
			fired := false
			current.afterCheckpoint = func(stage string) error {
				if stage == boundary && !fired {
					fired = true
					return errors.New("real durable interruption")
				}
				return nil
			}
			if _, err = current.executeBundleComposition(store, "run", request); err == nil || !fired {
				t.Fatal("actual boundary did not interrupt", err)
			}
			owner := current.owner
			current.owner = func(knowledgeengine.Installation, string, any) (json.RawMessage, error) {
				t.Fatal("status invoked owner")
				return nil, nil
			}
			status, e := current.executeBundleComposition(store, "status", map[string]any{"operation_id": request["operation_id"]})
			if e != nil || workflowValue(t, status)["validation"] != "sealed_checkpoints_only" {
				t.Fatal("status failed", e)
			}
			current.owner = owner
			current.afterCheckpoint = nil
			done, e := current.executeBundleComposition(store, "recover", map[string]any{"operation_id": request["operation_id"]})
			if e != nil {
				t.Fatal(e)
			}
			value := workflowValue(t, done)
			e = store.With("", false, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
				pack, e := current.compositionBundleReference(s, value["pack_evaluation_refs"].(map[string]any)["original-provider"].(string), "provider_pack_evaluation", false)
				if e != nil {
					return e
				}
				if pack.Installation != old.installation {
					t.Fatal("original pack owner replaced")
				}
				for _, pair := range [][2]string{{"composition_ref", "composition_explore"}, {"reassessment_ref", "composition_reassess"}} {
					view, e := current.resolveLogicalReference(s, value[pair[0]].(string), pair[1], false)
					if e != nil {
						return e
					}
					if view.Record.Installation != current.installation || view.Record.Operation != "composition_bundle_evaluate" {
						t.Fatal("exact bundled owner lost")
					}
				}
				return nil
			})
			if e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestSCVBundleWorkflowRejectsPoisonedImmutableReuse(t *testing.T) {
	for _, kind := range []string{"payloads", "records"} {
		t.Run(kind, func(t *testing.T) {
			store := workflowStore(t)
			var raw []byte
			var err error
			if kind == "payloads" {
				raw, err = scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": map[string]any{"note": "�"}})
			} else {
				result, e := scvworkflow.Seal(map[string]any{"protocol": "symphony.scv.provider-pack.v1", "note": "�"})
				if e != nil {
					t.Fatal(e)
				}
				record, e := scvworkflow.NewRecord("provider_pack_prepare", knowledgeengine.Installation{Role: "scv", Version: "0.6.0-dev"}, json.RawMessage(`{"note":"�"}`), result)
				if e != nil {
					t.Fatal(e)
				}
				raw, err = scvworkflow.Canonical(record)
			}
			if err != nil {
				t.Fatal(err)
			}
			var ref string
			if err = store.With("", true, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
				var e error
				ref, e = s.Put(kind, raw)
				return e
			}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(store.Root, "symphony", "qxctl", "scv", "workflows-v1", store.TOPSID, kind, strings.TrimPrefix(ref, "sha256:")+".json")
			poisoned := []byte(strings.ReplaceAll(string(raw), "�", `\ud800`))
			if err = os.WriteFile(path, poisoned, 0600); err != nil {
				t.Fatal(err)
			}
			if err = store.With("", true, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
				// Document the inherited normalization primitive without changing it.
				if _, e := s.Put(kind, raw); e != nil {
					t.Fatal("fixture no longer demonstrates legacy normalized reuse", e)
				}
				_, e := putBundleWorkflowObject(s, kind, raw)
				return e
			}); err == nil {
				t.Fatal("v2 acknowledged poisoned cached bytes")
			}
		})
	}
}

func TestSCVBundleWorkflowRejectsRawLegacyOwnerResultBeforeCheckpoint(t *testing.T) {
	r, store, input := mockCompositionWorkflow(t)
	input["protocol"] = "symphony.qxctl.scv-composition-workflow-request.v2"
	input["transport"] = "bundle"
	input["prior_composition_ref"] = nil
	r.installation.Version = "0.10.0-dev"
	raw, _ := scvworkflow.Canonical(input)
	run, err := scvworkflow.NewCompositionRunV2(store, r.installation, raw)
	if err != nil {
		t.Fatal(err)
	}
	old := r.installation
	old.Version = "0.6.0-dev"
	valid, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.scv.provider-pack-evaluation.v1", "note": "�"})
	if err != nil {
		t.Fatal(err)
	}
	r.owner = func(knowledgeengine.Installation, string, any) (json.RawMessage, error) {
		return json.RawMessage(strings.ReplaceAll(string(valid), "�", `\ud800`)), nil
	}
	err = store.WithCompositionV2(run.OperationID, true, func(s *scvworkflow.Session, _ *scvworkflow.CompositionRunV2) error {
		run, err = s.SaveCompositionV2(store, run)
		if err != nil {
			return err
		}
		_, e := r.bundleCompositionStage(store, s, &run, "pack:selected", "provider_pack_evaluate", old, json.RawMessage(`{}`))
		if e == nil {
			t.Fatal("raw owner result normalized before retention")
		}
		if run.Stages["pack:selected"].InputRef == "" || run.Stages["pack:selected"].ResultRef != "" {
			t.Fatal("invalid owner result advanced checkpoint")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSCVBundleWorkflowPreservesBundledPriorOriginalOwner(t *testing.T) {
	r, store, input := mockBundleWorkflow(t)
	fixture := logicalReferenceFixture(t)["operations"].([]any)[0].(map[string]any)
	inst := r.installation
	inst.Version = "0.9.0-dev"
	record := logicalTestBundle(t, inst, "composition_explore", fixture["logical_input"], fixture["logical_result"])
	ref := logicalRetain(t, store, record)
	input["prior_composition_ref"] = ref
	calls := 0
	owner := r.owner
	r.owner = func(selected knowledgeengine.Installation, operation string, value any) (json.RawMessage, error) {
		if selected == inst && operation == record.Operation {
			raw, e := scvworkflow.Canonical(value)
			if e != nil {
				return nil, e
			}
			if !scvworkflow.Same(raw, record.Input) {
				return nil, errors.New("changed original bundle invocation")
			}
			calls++
			return record.Artifact, nil
		}
		return owner(selected, operation, value)
	}
	done, err := r.executeBundleComposition(store, "run", input)
	if err != nil {
		t.Fatal(err)
	}
	result := workflowValue(t, done)
	if calls == 0 || result["run"].(map[string]any)["request"].(map[string]any)["prior_composition_ref"] != ref {
		t.Fatal("original bundled prior lost")
	}
	if err = store.With("", false, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		view, e := r.resolveLogicalReference(s, ref, "composition_explore", false)
		if e != nil {
			return e
		}
		if view.Record.Installation != inst || view.Descriptor["transport"] == nil {
			t.Fatal("prior provenance rewritten")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

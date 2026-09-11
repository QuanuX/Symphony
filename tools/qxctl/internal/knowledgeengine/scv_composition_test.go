package knowledgeengine

import (
	"context"
	"os"
	"testing"
)

func compositionTestResolution() map[string]any {
	return map[string]any{"kind": "adapter", "reference": "fixture.adapter.v1", "description": "Verify exact synthetic adapter"}
}
func compositionTestRequirement(n int) map[string]any {
	return map[string]any{"requirement_id": "capacity", "importance": "required", "operator": "gte", "right": map[string]any{"kind": "literal", "value": map[string]any{"type": "integer", "value": n, "unit": "units"}}, "resolution": compositionTestResolution()}
}
func compositionTestRecipe(id string, bind bool) map[string]any {
	bindings, needs, supplies := []any{}, []any{}, []any{}
	if bind {
		bindings = append(bindings, map[string]any{"requirement_id": "capacity", "claim": map[string]any{"claim_id": "limit", "subject": "fixture-service", "scope": map[string]any{"plan": "fixture"}}})
		needs = append(needs, "fixture.store.v1")
	} else {
		supplies = append(supplies, "fixture.store.v1")
	}
	return map[string]any{"recipe_id": id, "provider_id": "user-fixture", "bindings": bindings, "prerequisites": []any{}, "requires_interfaces": needs, "supplies_interfaces": supplies, "guarantee_changes": []any{}, "implementation": map[string]any{"status": "available", "reference": "fixture.adapter.v1"}, "resolution": compositionTestResolution()}
}
func compositionTestInput(t *testing.T) map[string]any {
	return scvTestClone(t, map[string]any{"interpretations": []any{}, "additional_knowledge": []any{}, "provider_packs": []any{}, "query_time": "2026-09-10T12:00:01Z", "requirements": []any{compositionTestRequirement(8)}, "slots": []any{map[string]any{"slot_id": "compute", "allowed_provider_ids": []any{"user-fixture"}, "recipes": []any{compositionTestRecipe("first", true), compositionTestRecipe("second", true)}}, map[string]any{"slot_id": "store", "allowed_provider_ids": []any{"user-fixture"}, "recipes": []any{compositionTestRecipe("local", false), compositionTestRecipe("remote", false)}}}, "allowed_guarantee_changes": []any{}, "counterfactuals": []any{}, "bounds": map[string]any{"max_candidates": 32}})
}
func TestSCVCompositionFiniteSearchAndInputValidation(t *testing.T) {
	input := compositionTestInput(t)
	m, err := compModel(input)
	if err != nil || m.requested != 4 || m.eligible != 4 || len(compProducts(m)) != 4 {
		t.Fatal(m, err)
	}
	for name, change := range map[string]func(map[string]any){"duplicate_requirement": func(i map[string]any) {
		i["requirements"] = append(i["requirements"].([]any), i["requirements"].([]any)[0])
	}, "unknown_binding": func(i map[string]any) {
		i["slots"].([]any)[0].(map[string]any)["recipes"].([]any)[0].(map[string]any)["bindings"].([]any)[0].(map[string]any)["requirement_id"] = "unknown"
	}, "malformed_right": func(i map[string]any) { i["requirements"].([]any)[0].(map[string]any)["right"] = []any{} }, "empty_provider_selection": func(i map[string]any) { i["slots"].([]any)[0].(map[string]any)["allowed_provider_ids"] = []any{} }, "no_available_reference": func(i map[string]any) {
		i["slots"].([]any)[0].(map[string]any)["recipes"].([]any)[0].(map[string]any)["implementation"].(map[string]any)["reference"] = nil
	}} {
		t.Run(name, func(t *testing.T) {
			i := scvTestClone(t, input)
			change(i)
			if _, err := compModel(i); err == nil {
				t.Fatal("malformed recipe model accepted")
			}
		})
	}
}
func TestInstalledSCVCompositionEnumerationAndTamper(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.6.0-dev packages")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range SCVDomains() {
		t.Run(domain, func(t *testing.T) {
			owner := func(op string, input any) map[string]any {
				t.Helper()
				raw, err := SCVCanonical(input)
				if err != nil {
					t.Fatal(err)
				}
				response, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.6.0-dev", cwd, op, raw)
				if err != nil {
					t.Fatal(op, err)
				}
				out, err := scvObject(response.Result)
				if err != nil {
					t.Fatal(err)
				}
				return out
			}
			knowledge := owner("knowledge_interpret", map[string]any{"captures": []any{}, "claims": []any{map[string]any{"claim_id": "limit", "subject": "fixture-service", "predicate": "maximum", "scope": map[string]any{"plan": "fixture"}, "statement_kind": "user_assertion", "value": map[string]any{"type": "integer", "value": 10, "unit": "units"}, "evidence": []any{}, "dependencies": []any{}}}, "interpreter_version": "synthetic-composition-fixture-v1", "selection_policy": map[string]any{"policy_id": "fixture", "max_age_seconds": 60, "partial_capture": "exclude", "allowed_statement_kinds": []any{"user_assertion"}}})
			input := compositionTestInput(t)
			input["additional_knowledge"] = []any{knowledge}
			result := owner("composition_explore", input)
			search := result["search"].(map[string]any)
			if !scvEqual(search["evaluated_candidates"], 4) || search["exhaustive"] != true {
				t.Fatal("not Cartesian enumeration")
			}
			first := result["scenarios"].([]any)[0].(map[string]any)["candidates"].([]any)[0].(map[string]any)
			if first["status"] != "conditional" || len(first["obligations"].([]any)) != 3 {
				t.Fatal("caller assertion or implementation qualification lost")
			}
			for name, change := range map[string]func(map[string]any){"false_searchspace": func(v map[string]any) { v["search"].(map[string]any)["eligible_combinations"] = 5 }, "lost_candidate": func(v map[string]any) {
				s := v["scenarios"].([]any)[0].(map[string]any)
				s["candidates"] = s["candidates"].([]any)[:1]
			}, "dropped_obligation": func(v map[string]any) {
				v["scenarios"].([]any)[0].(map[string]any)["candidates"].([]any)[0].(map[string]any)["obligations"] = []any{}
			}, "changed_comparison": func(v map[string]any) {
				c := v["scenarios"].([]any)[0].(map[string]any)["candidates"].([]any)[0].(map[string]any)
				cs := c["connections"].([]any)
				cs[len(cs)-1].(map[string]any)["checks"].([]any)[0].(map[string]any)["comparison"] = false
			}, "invented_provider": func(v map[string]any) {
				v["scenarios"].([]any)[0].(map[string]any)["candidates"].([]any)[0].(map[string]any)["choices"].([]any)[0].(map[string]any)["provider_id"] = "other"
			}} {
				t.Run(name, func(t *testing.T) {
					v := scvTestClone(t, result)
					change(v)
					corpusBoundaryValidate(t, "composition_explore", input, v, false)
				})
			}
			counter := scvTestClone(t, input)
			counter["counterfactuals"] = []any{map[string]any{"counterfactual_id": "longer", "requirements": []any{compositionTestRequirement(12)}}}
			cr := owner("composition_explore", counter)
			if !scvEqual(cr["evidence_evaluation"], result["evidence_evaluation"]) {
				t.Fatal("counterfactual changed selected evidence")
			}
			cc := cr["scenarios"].([]any)[1].(map[string]any)["candidates"].([]any)[0].(map[string]any)
			if cc["status"] != "conditional" {
				t.Fatal("false caller assertion comparison promoted to provider contradiction")
			}
			limited := scvTestClone(t, input)
			limited["bounds"].(map[string]any)["max_candidates"] = 1
			lr := owner("composition_explore", limited)
			if lr["search"].(map[string]any)["stop_reason"] != "candidate_limit" {
				t.Fatal("partial search not explicit")
			}
			diffInput := map[string]any{"before": result, "after": lr}
			diff := owner("composition_reassess", diffInput)
			if diff["change_axes"].(map[string]any)["bounds"] != true || diff["change_axes"].(map[string]any)["evidence"] != false {
				t.Fatal("search-only axes")
			}

			for _, mode := range []string{"ambiguous", "missing_left"} {
				t.Run(mode+"_dependency", func(t *testing.T) {
					unbound := scvTestClone(t, input)
					firstRecipe := unbound["slots"].([]any)[0].(map[string]any)["recipes"].([]any)[0].(map[string]any)
					if mode == "ambiguous" {
						unbound["slots"].([]any)[1].(map[string]any)["recipes"].([]any)[0].(map[string]any)["bindings"] = firstRecipe["bindings"]
					} else {
						ref := scvTestClone(t, firstRecipe["bindings"].([]any)[0].(map[string]any)["claim"].(map[string]any))
						ref["kind"] = "claim"
						unbound["requirements"].([]any)[0].(map[string]any)["right"] = ref
						firstRecipe["bindings"] = []any{}
					}
					before := owner("composition_explore", unbound)
					claim := scvTestClone(t, knowledge["claims"].([]any)[0].(map[string]any))
					claim["value"].(map[string]any)["value"] = 11
					changedKnowledge := owner("knowledge_interpret", map[string]any{"captures": []any{}, "claims": []any{claim}, "interpreter_version": "synthetic-composition-fixture-v1", "selection_policy": knowledge["selection_policy"]})
					unbound["additional_knowledge"] = []any{changedKnowledge}
					after := owner("composition_explore", unbound)
					candidate := before["scenarios"].([]any)[0].(map[string]any)["candidates"].([]any)[0].(map[string]any)
					if candidate["status"] != "unresolved" || !scvEqual(candidate["claim_ids"], []any{"limit"}) {
						t.Fatal("unbound references disappeared or became credited")
					}
					pair := map[string]any{"before": before, "after": after}
					change := owner("composition_reassess", pair)
					found := false
					for _, raw := range change["candidates"].([]any) {
						row := raw.(map[string]any)
						if row["candidate_id"] == candidate["candidate_id"] {
							found = true
							if row["changed"] != false || row["affected"] != true {
								t.Fatal("unbound evidence change omitted")
							}
							row["affected"] = false
						}
					}
					if !found {
						t.Fatal("missing unbound candidate")
					}
					corpusBoundaryValidate(t, "composition_reassess", pair, change, false)
				})
			}

			t.Run("transitive_dependency", func(t *testing.T) {
				limit := scvTestClone(t, knowledge["claims"].([]any)[0].(map[string]any))
				premise := scvTestClone(t, limit)
				premise["claim_id"], premise["predicate"] = "premise", "premise"
				limit["dependencies"] = []any{map[string]any{"claim_id": "premise", "role": "requires"}}
				build := func() map[string]any {
					return owner("knowledge_interpret", map[string]any{"captures": []any{}, "claims": []any{limit, premise}, "interpreter_version": "composition-dependency-fixture", "selection_policy": knowledge["selection_policy"]})
				}
				selected := scvTestClone(t, input)
				selected["additional_knowledge"] = []any{build()}
				before := owner("composition_explore", selected)
				premise["value"].(map[string]any)["value"] = 11
				selected["additional_knowledge"] = []any{build()}
				after := owner("composition_explore", selected)
				pair := map[string]any{"before": before, "after": after}
				change := owner("composition_reassess", pair)
				for _, raw := range change["candidates"].([]any) {
					row := raw.(map[string]any)
					if row["changed"] != false || row["affected"] != true {
						t.Fatal("transitive dependency omitted")
					}
					row["affected"] = false
				}
				corpusBoundaryValidate(t, "composition_reassess", pair, change, false)
			})
			forged := scvTestClone(t, diff)
			forged["search_changed"] = false
			corpusBoundaryValidate(t, "composition_reassess", diffInput, forged, false)
		})
	}
}

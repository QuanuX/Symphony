package knowledgeengine

import (
	"context"
	"os"
	"strings"
	"testing"
)

func coverageTestDesired(id, provider, family string) map[string]any {
	return map[string]any{"source_id": id, "provider_id": provider, "family_id": family, "publisher": "Synthetic test fixture", "authority_role": "documentation", "scope": "Synthetic bounded service", "locators": []any{map[string]any{"locator_id": "main", "uri": "https://example.invalid/" + id, "role": "primary", "format": "text", "selector": "fixture-v1"}}, "continuity_evidence": []any{}}
}
func coverageTestCandidate(t *testing.T, sources []any) map[string]any {
	return scvTestSeal(t, map[string]any{"protocol": "symphony.scv.provider.v1", "provider_id": "cf", "family_id": "scev", "display_name": "Fixture CF", "sources": sources, "disposition": "candidate", "interpretation_scope": "captured-text-and-explicit-assertions"})
}
func coverageTestMetadata(t *testing.T) (map[string]any, map[string]any) {
	a := corpusBoundaryIndex(t, "a", "2026-09-10T00:00:00Z", "complete")
	b := corpusBoundaryIndex(t, "b", "2026-09-10T00:00:00Z", "failed")
	c := corpusBoundaryIndex(t, "c", "2026-09-10T00:00:00Z", "complete")
	corpus := corpusBoundarySnapshot(t, 1, nil, corpusBoundaryMember("a", a, a), corpusBoundaryMember("b", b, nil), corpusBoundaryMember("c", c, c))
	qi, query := corpusBoundaryQuery(t, corpus, []any{"b", "c"}, "last_complete")
	query = scvTestSeal(t, query)
	provider := coverageTestCandidate(t, []any{coverageTestDesired("b", "cf", "scev"), coverageTestDesired("a", "cf", "scev"), coverageTestDesired("d", "cf", "scev")})
	input := map[string]any{"provider": provider, "corpus_query": qi, "interpretations": []any{}}
	row := func(id string, member any, status string) map[string]any {
		return map[string]any{"source_id": id, "locator_id": "main", "member_id": member, "selection_status": status, "selected_capture_digest": nil, "declaration_match": "not_available", "declaration_difference_fields": []any{}, "interpretations": []any{}}
	}
	result := scvTestSeal(t, map[string]any{"protocol": "symphony.scv.provider-coverage.v1", "domain": "scv", "input": input, "corpus_query_result": query, "sources": []any{row("a", "a", "not_selected"), row("b", "b", "unavailable"), row("d", nil, "not_selected")}, "unlisted_members": []any{map[string]any{"member_id": "c", "family_id": "scev", "provider_id": "cf", "source_id": "c", "locator_id": "main", "reason": "source_not_declared"}}, "unselected_bindings": []any{}, "summary": map[string]any{"declared_sources": 3, "declared_locators": 3, "selected_declared_members": 1, "unselected_declared_locators": 2, "selected_unlisted_members": 1, "replayed_selected_captures": 0, "selected_profile_bindings": 0, "matched_rule_attempts": 0, "unresolved_rule_attempts": 0, "unselected_profile_bindings": 0, "selected_declared_status": map[string]any{"complete": 0, "partial": 0, "failed": 0, "unavailable": 1}, "selected_declared_freshness": map[string]any{"current": 0, "expired": 0, "future": 0, "not_selected": 1}}, "limitations": []any{"Synthetic fixture; selected inventory only"}})
	return input, result
}
func TestSCVProviderCoverageRejectsResealedAccountingAndSelection(t *testing.T) {
	input, result := coverageTestMetadata(t)
	corpusBoundaryValidate(t, "provider_coverage", input, result, true)
	for name, mutate := range map[string]func(map[string]any){
		"false_complete": func(v map[string]any) {
			v["summary"].(map[string]any)["selected_declared_status"].(map[string]any)["complete"] = 1
		},
		"false_body_availability": func(v map[string]any) { v["summary"].(map[string]any)["replayed_selected_captures"] = 1 },
		"lost_declaration":        func(v map[string]any) { v["sources"] = v["sources"].([]any)[:2] },
		"invented_capture": func(v map[string]any) {
			v["sources"].([]any)[1].(map[string]any)["selected_capture_digest"] = strings.Repeat("a", 64)
		},
		"unselected_is_absent":      func(v map[string]any) { v["sources"].([]any)[0].(map[string]any)["member_id"] = nil },
		"metadata_becomes_verified": func(v map[string]any) { v["sources"].([]any)[0].(map[string]any)["declaration_match"] = "matches" },
		"lost_unlisted":             func(v map[string]any) { v["unlisted_members"] = []any{} },
		"wrong_unlisted_reason": func(v map[string]any) {
			v["unlisted_members"].([]any)[0].(map[string]any)["reason"] = "provider_not_declared"
		},
		"reordered_sources": func(v map[string]any) { rows := v["sources"].([]any); rows[0], rows[1] = rows[1], rows[0] },
		"changed_input": func(v map[string]any) {
			v["input"].(map[string]any)["corpus_query"].(map[string]any)["selection"] = "latest_attempt"
		},
		"wrong_query_time": func(v map[string]any) {
			q := v["corpus_query_result"].(map[string]any)
			q["query_time"] = "2026-09-12T00:00:00Z"
			v["corpus_query_result"] = scvTestSeal(t, q)
		},
		"out_of_scope_owner": func(v map[string]any) { v["domain"] = "schv-aws" },
		"missing_limitation": func(v map[string]any) { v["limitations"] = []any{} },
	} {
		t.Run(name, func(t *testing.T) {
			v := scvTestClone(t, result)
			mutate(v)
			corpusBoundaryValidate(t, "provider_coverage", input, v, false)
		})
	}
}

func TestInstalledSCVProviderCoverageSelectedEvidenceAndIndependentPolicies(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COVERAGE_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.5.0-dev packages")
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
				response, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.5.0-dev", cwd, op, raw)
				if err != nil {
					t.Fatal(op, err)
				}
				v, err := scvObject(response.Result)
				if err != nil {
					t.Fatal(err)
				}
				return v
			}
			provider, family := "aws", "schv"
			if strings.HasPrefix(domain, "schv-") {
				provider = strings.TrimPrefix(domain, "schv-")
			} else if domain == "scev" || domain == "scev-cf" {
				provider, family = "cf", "scev"
			}
			desired := coverageTestDesired("docs", provider, family)
			declarationInput := map[string]any{"provider_id": provider, "family_id": family, "display_name": "Synthetic test provider", "sources": []any{desired, coverageTestDesired("unacquired", provider, family)}}
			declaration := owner("provider_onboard", declarationInput)
			capture := func(id, body string) map[string]any {
				d := coverageTestDesired(id, provider, family)
				s := owner("source_plan", map[string]any{"operation_id": "fixture-" + id, "current": nil, "desired": d, "reason": "Synthetic bounded fixture"})["source"]
				return owner("capture_import", map[string]any{"source": s, "locator_id": "main", "resolved_uri": "https://example.invalid/" + id, "redirects": []any{}, "observed_at": "2026-09-10T12:00:00Z", "upstream_revision": nil, "media_type": "text/plain", "body": body, "completeness": "complete", "issues": []any{}})
			}
			selected := capture("docs", "Fixture service. Port: 7844 UDP.")
			previous := capture("docs", "Fixture service. Port: 7845 UDP.")
			unlistedCapture := capture("unlisted", "Fixture service. Port: 9999 UDP.")
			corpus := owner("corpus_build", map[string]any{"corpus_id": "fixture", "previous": nil, "snapshot_time": "2026-09-10T12:00:00Z", "attempts": []any{map[string]any{"member_id": "docs", "capture": owner("capture_index", map[string]any{"capture": selected})}, map[string]any{"member_id": "unlisted", "capture": owner("capture_index", map[string]any{"capture": unlistedCapture})}}})
			qi := map[string]any{"corpus": corpus, "query_time": "2026-09-10T12:00:01Z", "member_ids": []any{}, "selection": "latest_attempt", "max_age_seconds": 60}
			input := map[string]any{"provider": declaration, "corpus_query": qi, "interpretations": []any{}}
			metadata := owner("provider_coverage", input)
			if !scvEqual(metadata["summary"].(map[string]any)["replayed_selected_captures"], 0) {
				t.Fatal("metadata claimed body replay")
			}
			policy := map[string]any{"policy_id": "caller-fixture", "max_age_seconds": 60, "allowed_statement_kinds": []any{"requirement"}, "partial_capture": "exclude"}
			emptyProfile := owner("profile_prepare", map[string]any{"profile": map[string]any{"protocol": "symphony.scv.interpretation-profile.v1", "profile_id": "empty-fixture", "profile_version": "1", "provider_id": provider, "source_id": "docs", "locator_id": "main", "media_types": []any{"text/plain"}, "authored_by": "Synthetic fixture", "rationale": "No authored rules yet", "rules": []any{}}})
			rawWrapper := owner("provider_interpret", map[string]any{"captures": []any{selected}, "profiles": []any{emptyProfile}, "bindings": []any{map[string]any{"profile_digest": emptyProfile["digest"], "capture_digest": selected["digest"]}}, "selection_policy": policy})
			input["interpretations"] = []any{rawWrapper}
			rawCoverage := owner("provider_coverage", input)
			if !scvEqual(rawCoverage["summary"].(map[string]any)["replayed_selected_captures"], 1) || !scvEqual(rawCoverage["summary"].(map[string]any)["selected_profile_bindings"], 1) || rawCoverage["sources"].([]any)[0].(map[string]any)["declaration_match"] != "matches" {
				t.Fatal("zero-rule replayed capture was not independently counted")
			}
			makeWrapper := func(cap map[string]any, policy map[string]any) map[string]any {
				sid := cap["source"].(map[string]any)["source_id"]
				rule := func(id, prefix string) map[string]any {
					return map[string]any{"rule_id": id, "claim_id": id, "subject": "fixture", "predicate": "required-port", "scope": map[string]any{"service": "fixture-v1"}, "statement_kind": "requirement", "dependencies": []any{}, "context": []any{"Fixture service."}, "extractor": map[string]any{"kind": "delimited", "prefix": prefix, "suffix": " UDP.", "type": "integer", "unit": "port"}}
				}
				draft := map[string]any{"protocol": "symphony.scv.interpretation-profile.v1", "profile_id": "fixture", "profile_version": "1", "provider_id": provider, "source_id": sid, "locator_id": "main", "media_types": []any{"text/plain"}, "authored_by": "Synthetic fixture", "rationale": "Exact synthetic token test", "rules": []any{rule("port", "Port: "), rule("unknown", "Missing port: ")}}
				profile := owner("profile_prepare", map[string]any{"profile": draft})
				return owner("provider_interpret", map[string]any{"captures": []any{cap}, "profiles": []any{profile}, "bindings": []any{map[string]any{"profile_digest": profile["digest"], "capture_digest": cap["digest"]}}, "selection_policy": policy})
			}
			first := makeWrapper(selected, policy)
			secondPolicy := scvTestClone(t, policy)
			secondPolicy["max_age_seconds"] = 3600
			input["interpretations"] = []any{makeWrapper(previous, policy), makeWrapper(unlistedCapture, policy), makeWrapper(selected, secondPolicy), first, rawWrapper}
			result := owner("provider_coverage", input)
			summary := result["summary"].(map[string]any)
			for key, n := range map[string]int{"declared_sources": 2, "declared_locators": 2, "selected_declared_members": 1, "unselected_declared_locators": 1, "selected_unlisted_members": 1, "replayed_selected_captures": 1, "selected_profile_bindings": 3, "matched_rule_attempts": 2, "unresolved_rule_attempts": 2, "unselected_profile_bindings": 2} {
				if !scvEqual(summary[key], n) {
					t.Fatalf("%s=%v want %d", key, summary[key], n)
				}
			}
			if !scvEqual(result, owner("provider_coverage", input)) {
				t.Fatal("exact replay changed coverage")
			}
			for name, mutate := range map[string]func(map[string]any){
				"fabricated_match": func(v map[string]any) { v["summary"].(map[string]any)["matched_rule_attempts"] = 3 },
				"missing_rule": func(v map[string]any) {
					for _, raw := range v["sources"].([]any)[0].(map[string]any)["interpretations"].([]any) {
						entry := raw.(map[string]any)
						if len(entry["unresolved_rule_ids"].([]any)) > 0 {
							entry["unresolved_rule_ids"] = []any{}
							break
						}
					}
				},
				"lost_unselected_binding": func(v map[string]any) { v["unselected_bindings"] = []any{} },
				"wrong_profile_provenance": func(v map[string]any) {
					v["sources"].([]any)[0].(map[string]any)["interpretations"].([]any)[0].(map[string]any)["profile_digest"] = selected["digest"]
				},
				"invented_declaration_agreement": func(v map[string]any) { v["sources"].([]any)[1].(map[string]any)["declaration_match"] = "matches" },
			} {
				t.Run(name, func(t *testing.T) {
					v := scvTestClone(t, result)
					mutate(v)
					corpusBoundaryValidate(t, "provider_coverage", input, v, false)
				})
			}
			// Exact owner scope preserves a child corpus and interpretation domain.
			if domain != "scv" {
				raw, _ := SCVCanonical(input)
				response, err := InvokeSCVDomain(context.Background(), "scv", prefix, "0.5.0-dev", cwd, "provider_coverage", raw)
				if err != nil {
					t.Fatal("parent/child coverage", err)
				}
				v, _ := scvObject(response.Result)
				if v["corpus_query_result"].(map[string]any)["domain"] != domain {
					t.Fatal("child corpus provenance rewritten")
				}
			}
			// The same body ID attached to changed metadata must never gain credit.
			forgedInput := scvTestClone(t, input)
			forgedCorpus := forgedInput["corpus_query"].(map[string]any)["corpus"].(map[string]any)
			member := forgedCorpus["members"].([]any)[0].(map[string]any)
			index := member["latest_attempt"].(map[string]any)
			index["requested_uri"] = "https://example.invalid/forged"
			index = scvTestSeal(t, index)
			member["latest_attempt"], member["last_complete"] = index, index
			forgedCorpus = scvTestSeal(t, forgedCorpus)
			forgedInput["corpus_query"].(map[string]any)["corpus"] = forgedCorpus
			raw, _ := SCVCanonical(forgedInput)
			if _, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.5.0-dev", cwd, "provider_coverage", raw); err == nil {
				t.Fatal("native accepted borrowed capture digest")
			}
			// Give the consumer a self-consistently resealed query/outer result as well.
			forgedResult := scvTestClone(t, result)
			forgedResult["input"] = forgedInput
			forgedQuery := forgedResult["corpus_query_result"].(map[string]any)
			forgedQuery["corpus_digest"] = forgedCorpus["digest"]
			forgedQuery["members"].([]any)[0].(map[string]any)["selected"] = index
			forgedResult["corpus_query_result"] = scvTestSeal(t, forgedQuery)
			corpusBoundaryValidate(t, "provider_coverage", forgedInput, forgedResult, false)
			duplicate := scvTestClone(t, input)
			duplicate["interpretations"] = append(duplicate["interpretations"].([]any), first)
			raw, _ = SCVCanonical(duplicate)
			if _, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.5.0-dev", cwd, "provider_coverage", raw); err == nil {
				t.Fatal("duplicate wrapper accepted")
			}
			changed := scvTestClone(t, declarationInput)
			changed["sources"].([]any)[0].(map[string]any)["publisher"] = "Changed caller declaration"
			input["provider"] = owner("provider_onboard", changed)
			different := owner("provider_coverage", input)
			row := different["sources"].([]any)[0].(map[string]any)
			if row["declaration_match"] != "differs" || !scvEqual(row["declaration_difference_fields"], []any{"publisher"}) {
				t.Fatal("source declaration drift hidden")
			}
		})
	}
}

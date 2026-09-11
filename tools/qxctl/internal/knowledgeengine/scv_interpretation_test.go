package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func scvTestSeal(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	delete(value, "digest")
	value["digest"], _ = SCVDigest(value)
	raw, err := SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	result, err := scvObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func scvTestClone(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	raw, err := SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	out, err := scvObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestInstalledSCVProfileConnectionProvenance(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_INTERPRETATION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.3.0-dev packages")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range SCVDomains() {
		t.Run(domain, func(t *testing.T) {
			owner := func(operation string, payload any) map[string]any {
				t.Helper()
				raw, err := SCVCanonical(payload)
				if err != nil {
					t.Fatal(err)
				}
				response, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.3.0-dev", cwd, operation, raw)
				if err != nil {
					t.Fatal(operation, err)
				}
				value, err := scvObject(response.Result)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			descriptor := owner("inspect", map[string]any{})
			if len(descriptor["operations"].([]any)) != 20 {
				t.Fatal("wrong operation set")
			}
			provider, family := "fixture", "schv"
			if strings.HasPrefix(domain, "schv-") {
				provider = strings.TrimPrefix(domain, "schv-")
			} else if domain == "scev" || domain == "scev-cf" {
				provider, family = "cf", "scev"
			}
			desired := map[string]any{"source_id": "profile-doc", "provider_id": provider, "family_id": family, "publisher": "Synthetic fixture", "authority_role": "documentation", "scope": "Selected synthetic service", "locators": []any{map[string]any{"locator_id": "docs", "uri": "https://example.invalid/docs.md", "role": "primary", "format": "markdown", "selector": "fixture-v1"}}, "continuity_evidence": []any{}}
			source := owner("source_plan", map[string]any{"operation_id": "fixture", "current": nil, "desired": desired, "reason": "Local test"})["source"]
			capture := func(body string) map[string]any {
				return owner("capture_import", map[string]any{"source": source, "locator_id": "docs", "resolved_uri": "https://example.invalid/docs.md", "redirects": []any{}, "observed_at": "2026-09-10T12:00:00Z", "upstream_revision": nil, "media_type": "text/markdown", "body": body, "completeness": "complete", "issues": []any{}})
			}
			policy := map[string]any{"policy_id": "explicit-fixture-policy", "max_age_seconds": 60, "allowed_statement_kinds": []any{"documented_fact", "requirement", "recommendation", "user_assertion"}, "partial_capture": "exclude"}
			scope := map[string]any{"service": "fixture-v1"}
			profile := scvTestSeal(t, map[string]any{"protocol": "symphony.scv.interpretation-profile.v1", "profile_id": "fixture-profile", "profile_version": "1", "provider_id": provider, "source_id": "profile-doc", "locator_id": "docs", "media_types": []any{"text/markdown"}, "authored_by": "test fixture", "rationale": "Exact synthetic port extraction, not provider truth", "rules": []any{map[string]any{"rule_id": "port", "claim_id": "service-port", "subject": "edge", "predicate": "required-egress-port", "scope": scope, "statement_kind": "requirement", "dependencies": []any{}, "context": []any{"Fixture service."}, "extractor": map[string]any{"kind": "delimited", "prefix": "Port: ", "suffix": " UDP.", "type": "integer", "unit": "port"}}}})
			interpretInput := func(cap map[string]any) map[string]any {
				return map[string]any{"captures": []any{cap}, "profiles": []any{profile}, "bindings": []any{map[string]any{"profile_digest": profile["digest"], "capture_digest": cap["digest"]}}, "selection_policy": policy}
			}
			oldInput := interpretInput(capture("Fixture service. Port: 7844 UDP."))
			old := owner("provider_interpret", oldInput)
			if !scvEqual(old, owner("provider_interpret", oldInput)) {
				t.Fatal("interpretation replay changed")
			}
			checks := []any{map[string]any{"check_id": "port", "importance": "required", "left": map[string]any{"claim_id": "service-port", "subject": "edge", "scope": scope}, "operator": "eq", "right": map[string]any{"kind": "literal", "value": map[string]any{"type": "integer", "value": 7844, "unit": "port"}}}}
			evalInput := func(interpreted map[string]any) map[string]any {
				return map[string]any{"interpretations": []any{interpreted}, "additional_knowledge": []any{}, "query_time": "2026-09-10T12:00:01Z", "connections": []any{map[string]any{"connection_id": "edge-node", "from_subject": "edge", "to_subject": "node", "checks": checks}}}
			}
			before := owner("connection_evaluate", evalInput(old))
			if before["connections"].([]any)[0].(map[string]any)["status"] != "satisfied" {
				t.Fatal("matching documented check not satisfied")
			}
			next := owner("provider_interpret", interpretInput(capture("Fixture service. Port: 7845 UDP.")))
			after := owner("connection_evaluate", evalInput(next))
			if after["connections"].([]any)[0].(map[string]any)["status"] != "contradicted" {
				t.Fatal("changed requirement not reflected")
			}
			reassessed := owner("connection_reassess", map[string]any{"before": before, "after": after})
			change := reassessed["checks"].([]any)[0].(map[string]any)
			if change["changed"] != true || change["affected"] != true {
				t.Fatal("requirement change missing")
			}
			forgedAxes := scvTestClone(t, reassessed)
			forgedAxes["change_axes"].(map[string]any)["captures"] = false
			forgedAxes = scvTestSeal(t, forgedAxes)
			axisInput, _ := SCVCanonical(map[string]any{"before": before, "after": after})
			axisRaw, _ := SCVCanonical(forgedAxes)
			if err := ValidateSCVResult("connection_reassess", axisInput, axisRaw); err == nil {
				t.Fatal("consumer accepted resealed false source-change attribution")
			}
			// Independently reject resealed result tampering before printing/persisting.
			forged := scvTestClone(t, before)
			forged["connections"].([]any)[0].(map[string]any)["status"] = "contradicted"
			forged = scvTestSeal(t, forged)
			in, _ := SCVCanonical(evalInput(old))
			out, _ := SCVCanonical(forged)
			if err := ValidateSCVResult("connection_evaluate", in, out); err == nil {
				t.Fatal("resealed false aggregate accepted")
			}
			forged = scvTestClone(t, before)
			check := forged["connections"].([]any)[0].(map[string]any)["checks"].([]any)[0].(map[string]any)
			check["comparison"] = false
			forged = scvTestSeal(t, forged)
			out, _ = SCVCanonical(forged)
			if err := ValidateSCVResult("connection_evaluate", in, out); err == nil {
				t.Fatal("resealed false comparison accepted")
			}
			forged = scvTestClone(t, before)
			graphEvaluation := forged["graph_evaluation"].(map[string]any)
			graphEvaluation["findings"].([]any)[0].(map[string]any)["claim"].(map[string]any)["value"].(map[string]any)["value"] = json.Number("9999")
			forged["graph_evaluation"] = scvTestSeal(t, graphEvaluation)
			changedConnection := forged["connections"].([]any)[0].(map[string]any)
			changedConnection["status"] = "contradicted"
			changedCheck := changedConnection["checks"].([]any)[0].(map[string]any)
			changedCheck["status"] = "contradicted"
			changedCheck["comparison"] = false
			changedCheck["reasons"] = []any{"comparison_not_matched"}
			forged = scvTestSeal(t, forged)
			out, _ = SCVCanonical(forged)
			if err := ValidateSCVResult("connection_evaluate", in, out); err == nil {
				t.Fatal("consumer accepted fabricated finding with self-consistent comparison")
			}
			forged = scvTestClone(t, old)
			dropped := forged["knowledge"].(map[string]any)
			droppedClaim := dropped["claims"].([]any)[0].(map[string]any)
			droppedClaim["evidence"] = droppedClaim["evidence"].([]any)[1:]
			forged["knowledge"] = scvTestSeal(t, dropped)
			forged = scvTestSeal(t, forged)
			droppedInput, _ := SCVCanonical(oldInput)
			out, _ = SCVCanonical(forged)
			if err := ValidateSCVResult("provider_interpret", droppedInput, out); err == nil {
				t.Fatal("consumer accepted dropped required context anchor")
			}
			forged = scvTestClone(t, before)
			forged["domain"] = map[string]any{"invalid": true}
			malformedGraph := forged["graph_evaluation"].(map[string]any)
			malformedGraph["domain"] = map[string]any{"invalid": true}
			forged["graph_evaluation"] = scvTestSeal(t, malformedGraph)
			forged = scvTestSeal(t, forged)
			out, _ = SCVCanonical(forged)
			if err := ValidateSCVResult("connection_evaluate", in, out); err == nil {
				t.Fatal("consumer accepted nonscalar owner identity")
			}
			// C++ replays a nested artifact, including generated native graph material.
			forged = scvTestClone(t, old)
			knowledge := forged["knowledge"].(map[string]any)
			knowledge["claims"].([]any)[0].(map[string]any)["value"].(map[string]any)["value"] = json.Number("9999")
			forged["knowledge"] = scvTestSeal(t, knowledge)
			forged = scvTestSeal(t, forged)
			profileInput, _ := SCVCanonical(oldInput)
			forgedRaw, _ := SCVCanonical(forged)
			if err := ValidateSCVResult("provider_interpret", profileInput, forgedRaw); err == nil {
				t.Fatal("consumer accepted resealed fabricated dynamic token")
			}
			raw, _ := SCVCanonical(evalInput(forged))
			if _, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.3.0-dev", cwd, "connection_evaluate", raw); err == nil {
				t.Fatal("owner accepted resealed fabricated extraction")
			}
			future := evalInput(old)
			future["query_time"] = "2026-09-10T12:01:01Z"
			expired := owner("connection_evaluate", future)
			if expired["connections"].([]any)[0].(map[string]any)["status"] != "unresolved" {
				t.Fatal("expired evidence proved compatibility")
			}
			timed := owner("connection_reassess", map[string]any{"before": before, "after": expired})
			if timed["change_axes"].(map[string]any)["query_time"] != true {
				t.Fatal("time-only change not identified")
			}
		})
	}
}
func TestSCVConnectionConsumerRejectsResealedComparison(t *testing.T) {
	scope := map[string]any{"plan": "selected"}
	findings := map[string]map[string]any{"a": {"status": "supported", "claim": map[string]any{"subject": "service", "scope": scope, "statement_kind": "requirement", "value": map[string]any{"type": "decimal", "value": "9007199254740992.1", "unit": "ms"}}}}
	spec := map[string]any{"left": map[string]any{"claim_id": "a", "subject": "service", "scope": scope}, "right": map[string]any{"kind": "literal", "value": map[string]any{"type": "decimal", "value": "9007199254740992.2", "unit": "ms"}}, "operator": "gte"}
	status, comparison, _ := scvCheckOutcome(spec, findings)
	if status != "contradicted" || comparison != false {
		t.Fatal("decimal rounded during comparison")
	}
	spec["right"].(map[string]any)["value"].(map[string]any)["unit"] = "s"
	status, comparison, _ = scvCheckOutcome(spec, findings)
	if status != "unresolved" || comparison != nil {
		t.Fatal("unit mismatch promoted to incompatibility")
	}
	spec["right"].(map[string]any)["value"].(map[string]any)["unit"] = "ms"
	findings["a"]["claim"].(map[string]any)["statement_kind"] = "recommendation"
	status, _, _ = scvCheckOutcome(spec, findings)
	if status != "conditional" {
		t.Fatal("recommendation became mandatory")
	}
	findings["a"]["claim"].(map[string]any)["value"].(map[string]any)["value"] = "1/2"
	status, comparison, _ = scvCheckOutcome(spec, findings)
	if status != "unresolved" || comparison != nil {
		t.Fatal("noncanonical rational accepted as decimal")
	}
}

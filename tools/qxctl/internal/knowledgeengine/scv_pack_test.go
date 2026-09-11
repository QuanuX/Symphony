package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func packFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	var result map[string]any
	d := json.NewDecoder(strings.NewReader(packNativeGolden))
	d.UseNumber()
	if err := d.Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result["input"].(map[string]any), result
}
func TestSCVProviderPackConsumerMatchesNativeFixture(t *testing.T) {
	input, result := packFixture(t)
	if err := validateSCVProviderPack("provider_pack_evaluate", input, result); err != nil {
		t.Fatal(err)
	}
	pack := scvTestClone(t, input["pack"].(map[string]any))
	delete(pack, "digest")
	pack["fixtures"].([]any)[0].(map[string]any)["input_digest"] = nil
	prepare := map[string]any{"pack": pack, "fixtures": input["fixtures"]}
	if err := validateSCVProviderPack("provider_pack_prepare", prepare, input["pack"].(map[string]any)); err != nil {
		t.Fatal(err)
	}
}
func TestSCVProviderPackConsumerRejectsResealedEvidenceAndConformance(t *testing.T) {
	input, original := packFixture(t)
	for name, mutate := range map[string]func(map[string]any){
		"changed value": func(r map[string]any) {
			r["knowledge"].(map[string]any)["claims"].([]any)[0].(map[string]any)["value"].(map[string]any)["value"] = json.Number("99")
		},
		"changed unit": func(r map[string]any) {
			r["knowledge"].(map[string]any)["claims"].([]any)[0].(map[string]any)["value"].(map[string]any)["unit"] = "minutes"
		},
		"changed scope": func(r map[string]any) {
			r["knowledge"].(map[string]any)["claims"].([]any)[0].(map[string]any)["scope"].(map[string]any)["plan"] = "enterprise"
		},
		"promoted recommendation": func(r map[string]any) {
			r["knowledge"].(map[string]any)["claims"].([]any)[0].(map[string]any)["statement_kind"] = "recommendation"
		},
		"lost context evidence": func(r map[string]any) {
			r["knowledge"].(map[string]any)["claims"].([]any)[0].(map[string]any)["evidence"] = []any{}
		},
		"wrong policy": func(r map[string]any) {
			r["knowledge"].(map[string]any)["selection_policy"].(map[string]any)["max_age_seconds"] = json.Number("999")
		},
		"wrong interpreter": func(r map[string]any) { r["knowledge"].(map[string]any)["interpreter_version"] = "another" },
		"lost rule":         func(r map[string]any) { r["extractions"] = []any{} },
		"forged unresolved": func(r map[string]any) { r["extractions"].([]any)[0].(map[string]any)["status"] = "unresolved" },
		"invented qualification": func(r map[string]any) {
			r["extractions"].([]any)[0].(map[string]any)["reasons"] = []any{"partial_capture_qualified"}
		},
		"changed fixture actual":       func(r map[string]any) { r["fixture_results"].([]any)[0].(map[string]any)["actual_claims"] = []any{} },
		"changed authored expectation": func(r map[string]any) { r["fixture_results"].([]any)[0].(map[string]any)["expected_claims"] = []any{} },
		"changed fixture axes": func(r map[string]any) {
			r["fixture_results"].([]any)[0].(map[string]any)["difference_axes"].(map[string]any)["claims"] = true
		},
		"invented pass":      func(r map[string]any) { r["conformance"].(map[string]any)["passed"] = json.Number("2") },
		"lost fixture":       func(r map[string]any) { r["fixture_results"] = []any{} },
		"extra result field": func(r map[string]any) { r["certified"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			r := scvTestClone(t, original)
			mutate(r)
			r["knowledge"] = scvTestSeal(t, r["knowledge"].(map[string]any))
			r = scvTestSeal(t, r)
			if err := validateSCVProviderPack("provider_pack_evaluate", input, r); err == nil {
				t.Fatal("accepted resealed alteration")
			}
		})
	}
}
func TestSCVProviderPackDetachedCasesPreserveNotRunAndFailure(t *testing.T) {
	input, result := packFixture(t)
	input["fixtures"] = []any{}
	result["input"] = input
	c := result["fixture_results"].([]any)[0].(map[string]any)
	c["status"] = "not_run"
	c["actual_claims"] = nil
	c["actual_extractions"] = nil
	c["difference_axes"] = nil
	result["conformance"] = map[string]any{"passed": 0, "failed": 0, "not_run": 1}
	result = scvTestSeal(t, result)
	if err := validateSCVProviderPack("provider_pack_evaluate", input, result); err != nil {
		t.Fatal(err)
	}
	result["fixture_results"].([]any)[0].(map[string]any)["status"] = "passed"
	result = scvTestSeal(t, result)
	if err := validateSCVProviderPack("provider_pack_evaluate", input, result); err == nil {
		t.Fatal("unrun became passing")
	}
}
func TestSCVProviderPackMalformedSelectionsRejectWithoutPanic(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"null capture": func(i map[string]any) { i["captures"] = []any{nil} },
		"bad binding": func(i map[string]any) {
			i["bindings"] = []any{map[string]any{"profile_id": nil, "capture_digest": nil}}
		},
		"missing policy":  func(i map[string]any) { i["selection_policy"] = nil },
		"unknown fixture": func(i map[string]any) { i["fixtures"].([]any)[0].(map[string]any)["fixture_id"] = "missing" },
		"malformed profile": func(i map[string]any) {
			p := i["pack"].(map[string]any)
			p["structured_profiles"].([]any)[0].(map[string]any)["rules"].([]any)[0].(map[string]any)["dependencies"] = nil
			i["pack"] = scvTestSeal(t, p)
		},
	} {
		t.Run(name, func(t *testing.T) {
			i, r := packFixture(t)
			mutate(i)
			r["input"] = i
			r = scvTestSeal(t, r)
			if err := validateSCVProviderPack("provider_pack_evaluate", i, r); err == nil {
				t.Fatal("accepted malformed input")
			}
		})
	}
}
func TestSCVProviderPackIndependentStructuredTokens(t *testing.T) {
	for _, tc := range []struct{ body, pointer, typ, expected, reason string }{
		{`{"n":9.007199254740993e15}`, "/n", "decimal", "9007199254740993", ""},
		{`{"n":-1.2500e-2}`, "/n", "decimal", "-0.0125", ""},
		{`{"a/b~c":[10]}`, "/a~1b~0c/0", "integer", "10", ""},
		{`{"a":[10]}`, "/a/00", "integer", "", "array_index_invalid"},
		{`{"n":10,"$ref":"#/elsewhere"}`, "/n", "integer", "", "reference_unresolved"},
	} {
		nodes, err := scvPackDocument(tc.body)
		if err != nil {
			t.Fatal(err)
		}
		n, reason := scvPackSelect(nodes, tc.pointer)
		if reason != tc.reason {
			t.Fatalf("%s: %s", tc.body, reason)
		}
		if reason != "" {
			continue
		}
		v, ok := scvPackNodeValue(n, map[string]any{"type": tc.typ, "unit": nil})
		if !ok || stringValue(v["value"]) != tc.expected {
			t.Fatalf("exact value mismatch: %v", v)
		}
	}
	for _, body := range []string{`{"n":"\ud800"}`, `{"\udc00":1}`} {
		if _, err := scvPackDocument(body); err == nil {
			t.Fatal("unpaired surrogate accepted")
		}
	}
	nodes, err := scvPackDocument(`{"n":1.` + strings.Repeat("0", 1000) + `}`)
	if err != nil {
		t.Fatal(err)
	}
	n, _ := scvPackSelect(nodes, "/n")
	v, ok := scvPackNodeValue(n, map[string]any{"type": "decimal", "unit": nil})
	if !ok || v["value"] != "1" {
		t.Fatal("exact trailing zeros changed decimal")
	}
	if _, err := scvPackDocument(`{"n":1,"n":2}`); err == nil {
		t.Fatal("duplicate key accepted")
	}
}
func stringValue(v any) string {
	if n, ok := v.(json.Number); ok {
		return n.String()
	}
	s, _ := v.(string)
	return s
}
func TestInstalledSCVProviderPackConsumer(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact .6 installed owner")
	}
	i, expected := packFixture(t)
	payload, _ := SCVCanonical(i)
	response, err := InvokeSCVDomain(context.Background(), "scv", prefix, "0.6.0-dev", t.TempDir(), "provider_pack_evaluate", payload)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := scvObject(response.Result)
	if err != nil {
		t.Fatal(err)
	}
	if !scvEqual(actual, expected) {
		t.Fatal("installed pack result differs from recorded native fixture")
	}
}

// Golden native result from pack_test.cpp, whose 10-unit fixture expectation
// is independently authored in source. This fixed artifact exercises the Go
// consumer across the C++ boundary; it is never regenerated by these tests.
const packNativeGolden = `{"conformance":{"failed":0,"not_run":0,"passed":1},"digest":"sha256:3432f2e3f4574bffb213a2be3a1245619fec625cbd3b17aa5fec4120b0423f50","domain":"scv","extractions":[{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","claim_id":"maximum","profile_id":"limits","reasons":[],"rule_id":"maximum","status":"matched"}],"fixture_results":[{"actual_claims":[{"claim_id":"maximum","dependencies":[],"predicate":"maximum","scope":{"plan":"basic","version":"fixture-1"},"statement_kind":"documented_fact","subject":"service","value":{"type":"integer","unit":"units","value":10}}],"actual_extractions":[{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","claim_id":"maximum","profile_id":"limits","reasons":[],"rule_id":"maximum","status":"matched"}],"difference_axes":{"claims":false,"extractions":false},"expected_claims":[{"claim_id":"maximum","dependencies":[],"predicate":"maximum","scope":{"plan":"basic","version":"fixture-1"},"statement_kind":"documented_fact","subject":"service","value":{"type":"integer","unit":"units","value":10}}],"expected_extractions":[{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","claim_id":"maximum","profile_id":"limits","reasons":[],"rule_id":"maximum","status":"matched"}],"fixture_id":"basic-limit","status":"passed"}],"input":{"bindings":[{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","profile_id":"limits"}],"captures":[{"body":"{\"plan\":\"basic\",\"maximum\":10}","body_digest":"sha256:6354ac97adc55149f2e1003d1271e69d9a121450e49a2d65636b95d50a4f42fb","byte_size":29,"completeness":"complete","digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","issues":[],"locator_id":"main","media_type":"application/json","observed_at":"2026-09-10T00:00:00Z","protocol":"symphony.scv.capture.v1","redirects":[],"resolved_uri":"https://fixture.invalid/docs","source":{"authority_role":"reference","continuity_evidence":[],"digest":"sha256:2af99827c6cddcd70a3dd5bcef23cbd3df7bb49206d0654cdc33961f522c1fe6","family_id":"schv","generation":1,"locators":[{"format":"json","locator_id":"main","role":"primary","selector":"fixture-1","uri":"https://fixture.invalid/docs"}],"predecessor_digest":null,"protocol":"symphony.scv.source.v1","provider_id":"independent","publisher":"Fixture publisher","scope":"authored fixture","source_id":"docs"},"upstream_revision":null}],"fixtures":[{"bindings":[{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","profile_id":"limits"}],"captures":[{"body":"{\"plan\":\"basic\",\"maximum\":10}","body_digest":"sha256:6354ac97adc55149f2e1003d1271e69d9a121450e49a2d65636b95d50a4f42fb","byte_size":29,"completeness":"complete","digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","issues":[],"locator_id":"main","media_type":"application/json","observed_at":"2026-09-10T00:00:00Z","protocol":"symphony.scv.capture.v1","redirects":[],"resolved_uri":"https://fixture.invalid/docs","source":{"authority_role":"reference","continuity_evidence":[],"digest":"sha256:2af99827c6cddcd70a3dd5bcef23cbd3df7bb49206d0654cdc33961f522c1fe6","family_id":"schv","generation":1,"locators":[{"format":"json","locator_id":"main","role":"primary","selector":"fixture-1","uri":"https://fixture.invalid/docs"}],"predecessor_digest":null,"protocol":"symphony.scv.source.v1","provider_id":"independent","publisher":"Fixture publisher","scope":"authored fixture","source_id":"docs"},"upstream_revision":null}],"fixture_id":"basic-limit","selection_policy":{"allowed_statement_kinds":["documented_fact","recommendation"],"max_age_seconds":60,"partial_capture":"exclude","policy_id":"fixture-policy"}}],"pack":{"authored_by":"Pack author","digest":"sha256:38b92ca518e3a7f07c1b7354ca0d302ea9cbae97d8997a70ee879dc1f96784db","fixtures":[{"authored_by":"Fixture author","expected_claims":[{"claim_id":"maximum","dependencies":[],"predicate":"maximum","scope":{"plan":"basic","version":"fixture-1"},"statement_kind":"documented_fact","subject":"service","value":{"type":"integer","unit":"units","value":10}}],"expected_extractions":[{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","claim_id":"maximum","profile_id":"limits","reasons":[],"rule_id":"maximum","status":"matched"}],"fixture_id":"basic-limit","input_digest":"sha256:645ee3f869e36a5310d64a615df8a4553b55bddb2317c1955d45414dd8c3c658","label":"Independent expected maximum","rationale":"Expectation authored as 10 before extraction"}],"pack_id":"independent-fixture","pack_version":"fixture-1","profiles":[],"protocol":"symphony.scv.provider-pack.v1","provenance":["Independent synthetic fixture, not a vendor capability"],"provider":{"display_name":"Independent fixture","family_id":"schv","provider_id":"independent","sources":[{"authority_role":"reference","continuity_evidence":[],"family_id":"schv","locators":[{"format":"json","locator_id":"main","role":"primary","selector":"fixture-1","uri":"https://fixture.invalid/docs"}],"provider_id":"independent","publisher":"Fixture publisher","scope":"authored fixture","source_id":"docs"}]},"structured_profiles":[{"authored_by":"Mapping author","locator_id":"main","media_types":["application/json"],"profile_id":"limits","profile_version":"fixture-1","protocol":"symphony.scv.structured-profile.v1","provider_id":"independent","rationale":"Exact basic-plan maximum; default and exceptions are separate fields","rules":[{"claim_id":"maximum","context":[{"pointer":"/plan","value":{"type":"string","unit":null,"value":"basic"}}],"dependencies":[],"extractor":{"kind":"json_pointer","pointer":"/maximum","type":"integer","unit":"units"},"predicate":"maximum","rule_id":"maximum","scope":{"plan":"basic","version":"fixture-1"},"statement_kind":"documented_fact","subject":"service"}],"source_id":"docs"}]},"selection_policy":{"allowed_statement_kinds":["documented_fact","recommendation"],"max_age_seconds":60,"partial_capture":"exclude","policy_id":"fixture-policy"}},"knowledge":{"captures":[{"body":"{\"plan\":\"basic\",\"maximum\":10}","body_digest":"sha256:6354ac97adc55149f2e1003d1271e69d9a121450e49a2d65636b95d50a4f42fb","byte_size":29,"completeness":"complete","digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","issues":[],"locator_id":"main","media_type":"application/json","observed_at":"2026-09-10T00:00:00Z","protocol":"symphony.scv.capture.v1","redirects":[],"resolved_uri":"https://fixture.invalid/docs","source":{"authority_role":"reference","continuity_evidence":[],"digest":"sha256:2af99827c6cddcd70a3dd5bcef23cbd3df7bb49206d0654cdc33961f522c1fe6","family_id":"schv","generation":1,"locators":[{"format":"json","locator_id":"main","role":"primary","selector":"fixture-1","uri":"https://fixture.invalid/docs"}],"predecessor_digest":null,"protocol":"symphony.scv.source.v1","provider_id":"independent","publisher":"Fixture publisher","scope":"authored fixture","source_id":"docs"},"upstream_revision":null}],"claims":[{"alternative_supports":[],"claim_id":"maximum","dependencies":[],"evidence":[{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","quote":"10"},{"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","quote":"\"basic\""}],"predicate":"maximum","scope":{"plan":"basic","version":"fixture-1"},"scope_dependencies":[],"statement_kind":"documented_fact","subject":"service","valid_from":null,"valid_until":null,"value":{"type":"integer","unit":"units","value":10}}],"digest":"sha256:8000530e8572217a1bfe4cff85a6e4c3c8a5cdac919a17a54f383e4997bb1ab2","domain":"scv","interpreter_version":"provider-pack-v1:sha256:38b92ca518e3a7f07c1b7354ca0d302ea9cbae97d8997a70ee879dc1f96784db","limitations":["caller-proposed claims retain their statement kinds; quote anchoring is not factual verification","coverage is limited to selected captures; absence is not provider-wide unavailability","native extraction describes captured structure; it does not validate provider semantics","unsupported JSON dialect retained without semantic interpretation: sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11"],"native_edges":[],"native_nodes":[{"attributes":{"authority_role":"reference","completeness":"complete","publisher":"Fixture publisher","resolved_uri":"https://fixture.invalid/docs","scope":"authored fixture","source_id":"docs"},"capture_digest":"sha256:89f06c79d44c4d939a2bbd64d6724867743200771e0ad3ed6075500070377c11","kind":"source","node_id":"native:de9df792c2672ba67347ec7c4fab616cf9716709c077149b091e62010f049aea","pointer":""}],"protocol":"symphony.scv.knowledge.v1","selection_policy":{"allowed_statement_kinds":["documented_fact","recommendation"],"max_age_seconds":60,"partial_capture":"exclude","policy_id":"fixture-policy"}},"limitations":["pack mappings and fixture expectations are authored inputs; passing selected fixtures is not certification or provider endorsement","structured extraction evaluates explicit JSON pointers and typed contexts only; references, API behavior and undeclared exceptions remain unevaluated","document numeric lexemes are represented exactly within declared bounds; units, scope and statement kinds are never inferred or converted","fixture selection and production selection are separate; not_run fixtures and unresolved rules provide no positive proof"],"protocol":"symphony.scv.provider-pack-evaluation.v1"}`

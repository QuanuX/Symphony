package knowledgeengine

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func bundleProjectionFixtures(t *testing.T) []any {
	t.Helper()
	raw, err := os.ReadFile("testdata/scv-bundle.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	// The aggregate fixture is not an individual process request. Individual
	// input/result pairs below go through the production bounded raw entrypoint.
	var fixture map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	return fixture["operations"].([]any)
}

func bundleProjectionFailure(t *testing.T, input, result []byte) error {
	t.Helper()
	projection, err := ValidateSCVBundleEvaluation(input, result)
	if err == nil {
		t.Fatal("invalid evaluation returned a successful projection")
	}
	if !reflect.DeepEqual(projection, SCVBundleEvaluation{}) {
		t.Fatal("failed validation exposed a partial logical projection", projection.Operation)
	}
	if ValidateSCVResult("composition_bundle_evaluate", input, result) == nil {
		t.Fatal("public result validation diverges from projection rejection")
	}
	return err
}

func TestSCVBundleProjectionMatchesFourNativeOperations(t *testing.T) {
	for _, raw := range bundleProjectionFixtures(t) {
		item := raw.(map[string]any)
		invocation := item["input"].(map[string]any)
		operation := invocation["operation"].(string)
		t.Run(operation, func(t *testing.T) {
			input, result := bundleTestRaw(t, invocation), bundleTestRaw(t, item["result"])
			projection, err := ValidateSCVBundleEvaluation(input, result)
			if err != nil {
				t.Fatal(err)
			}
			native := item["logical_result"].(map[string]any)
			bundle := item["result"].(map[string]any)["result_bundle"].(map[string]any)
			owner := invocation["owner"].(map[string]any)
			if projection.Operation != operation || projection.Domain != owner["domain"] || projection.Version != owner["version"] ||
				projection.NativeResultDigest != native["digest"] || projection.ResultBundleDigest != bundle["digest"] || projection.ResultRootDigest != bundle["root_digest"] {
				t.Fatal("projection changed exact operation, owner or digest subject")
			}
			if !bytes.Equal(projection.Input, bundleTestRaw(t, item["logical_input"])) || !bytes.Equal(projection.Result, bundleTestRaw(t, native)) {
				t.Fatal("projection differs from the preserved native logical fixture")
			}
			if projection.NativeResultDigest == projection.ResultRootDigest || projection.ResultRootDigest == projection.ResultBundleDigest {
				t.Fatal("distinct native, content-root and transport seals collapsed")
			}
			if err := ValidateSCVResult(operation, projection.Input, projection.Result); err != nil {
				t.Fatal("projection does not pass the independent logical consumer", err)
			}
			if err := ValidateSCVResult("composition_bundle_evaluate", input, result); err != nil {
				t.Fatal("public result validation diverges from projection acceptance", err)
			}
			bundleProjectionFailure(t, input, bundleTestRaw(t, item["inspection"]))
		})
	}
}

func TestSCVBundleProjectionRejectsResealedAttribution(t *testing.T) {
	item := bundleProjectionFixtures(t)[3].(map[string]any)
	cases := map[string]func(map[string]any, map[string]any){
		"protocol": func(_ map[string]any, result map[string]any) {
			result["protocol"] = "symphony.scv.bundle-inspection.v1"
		},
		"native_protocol_as_outer": func(_ map[string]any, result map[string]any) {
			result["protocol"] = "symphony.scv.composition-followup.v1"
		},
		"owner_domain": func(_ map[string]any, result map[string]any) { result["owner"].(map[string]any)["domain"] = "schv" },
		"owner_version": func(_ map[string]any, result map[string]any) {
			result["owner"].(map[string]any)["version"] = "0.10.0-dev"
		},
		"unsupported_exact_version": func(input, result map[string]any) {
			input["owner"].(map[string]any)["version"] = "0.8.0-dev"
			result["owner"].(map[string]any)["version"] = "0.8.0-dev"
		},
		"nested_dispatch": func(input, result map[string]any) {
			input["operation"] = "composition_bundle_evaluate"
			result["operation"] = "composition_bundle_evaluate"
		},
		"logical_operation": func(_ map[string]any, result map[string]any) { result["operation"] = "composition_explore" },
		"input_metrics": func(_ map[string]any, result map[string]any) {
			result["input_metrics"].(map[string]any)["expanded_values"] = 1
		},
		"result_metrics": func(_ map[string]any, result map[string]any) {
			result["result_metrics"].(map[string]any)["materialized_bytes"] = 1
		},
		"native_digest_subject": func(_ map[string]any, result map[string]any) {
			result["native_result_digest"] = result["result_bundle"].(map[string]any)["root_digest"]
		},
		"input_root": func(_ map[string]any, result map[string]any) {
			result["input_root_digest"] = result["result_bundle"].(map[string]any)["root_digest"]
		},
		"validation":    func(_ map[string]any, result map[string]any) { result["validation"] = "owner_replayed" },
		"missing_owner": func(input, _ map[string]any) { delete(input, "owner") },
		"extra_result":  func(_ map[string]any, result map[string]any) { result["cached"] = true },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := scvTestClone(t, item["input"].(map[string]any))
			result := scvTestClone(t, item["result"].(map[string]any))
			mutate(input, result)
			var err error
			result["input_digest"], err = SCVDigest(input)
			if err != nil {
				t.Fatal(err)
			}
			bundleProjectionFailure(t, bundleTestRaw(t, input), bundleTestReseal(t, result))
		})
	}
}

// Change only caller-authored provenance text in an already preserved native
// follow-up fixture. This creates no new provider claims or outcome oracle.
func bundleProjectionTextFixture(t *testing.T) ([]byte, []byte) {
	t.Helper()
	item := bundleProjectionFixtures(t)[3].(map[string]any)
	input := scvTestClone(t, item["logical_input"].(map[string]any))
	native := scvTestClone(t, item["logical_result"].(map[string]any))
	input["submissions"].([]any)[0].(map[string]any)["provenance"].(map[string]any)["description"] = "projection � 😀 \u2028; literal \\u2028"
	native["input"] = input
	native = scvTestSeal(t, native)
	invocation := bundleTestInvocation(t, "composition_followup", input)
	return bundleTestRaw(t, invocation), bundleTestRaw(t, bundleTestResult(t, invocation, native))
}

func TestSCVBundleProjectionPreservesStrictRawInputAndResult(t *testing.T) {
	input, result := bundleProjectionTextFixture(t)
	if _, err := ValidateSCVBundleEvaluation(input, result); err != nil {
		t.Fatal(err)
	}
	zero := regexp.MustCompile(`:0([,}])`)
	for _, side := range []string{"input", "result"} {
		t.Run(side, func(t *testing.T) {
			raw := input
			if side == "result" {
				raw = result
			}
			mutations := map[string]func([]byte) []byte{
				"high_surrogate":      func(raw []byte) []byte { return bytes.ReplaceAll(raw, []byte("�"), []byte(`\ud800`)) },
				"low_surrogate":       func(raw []byte) []byte { return bytes.ReplaceAll(raw, []byte("�"), []byte(`\udc00`)) },
				"wrong_pair":          func(raw []byte) []byte { return bytes.ReplaceAll(raw, []byte("😀"), []byte(`\ud800\u1234`)) },
				"invalid_utf8":        func(raw []byte) []byte { return bytes.ReplaceAll(raw, []byte("�"), []byte{255}) },
				"duplicate_operation": func(raw []byte) []byte { return append([]byte(`{"operation":"composition_followup",`), raw[1:]...) },
				"trailing_json":       func(raw []byte) []byte { return append(bytes.Clone(raw), []byte(` {}`)...) },
				"float":               func(raw []byte) []byte { return zero.ReplaceAll(raw, []byte(`:0.0$1`)) },
				"exponent":            func(raw []byte) []byte { return zero.ReplaceAll(raw, []byte(`:0e0$1`)) },
				"integer_range":       func(raw []byte) []byte { return zero.ReplaceAll(raw, []byte(`:9007199254740992$1`)) },
			}
			for name, mutate := range mutations {
				t.Run(name, func(t *testing.T) {
					changed := mutate(raw)
					if bytes.Equal(raw, changed) {
						t.Fatal("raw mutation did not reach a selected value")
					}
					if side == "input" {
						bundleProjectionFailure(t, changed, result)
					} else {
						bundleProjectionFailure(t, input, changed)
					}
				})
			}
			for name, mutate := range map[string]func([]byte) []byte{
				"paired_surrogate":       func(raw []byte) []byte { return bytes.ReplaceAll(raw, []byte("😀"), []byte(`\ud83d\ude00`)) },
				"integral_negative_zero": func(raw []byte) []byte { return zero.ReplaceAll(raw, []byte(`:-0$1`)) },
				"escaped_key_and_value":  func(raw []byte) []byte { return bytes.ReplaceAll(raw, []byte("scv"), []byte(`\u0073cv`)) },
			} {
				t.Run(name, func(t *testing.T) {
					changed := mutate(raw)
					if bytes.Equal(raw, changed) {
						t.Fatal("valid spelling mutation was not exercised")
					}
					a, b := input, result
					if side == "input" {
						a = changed
					} else {
						b = changed
					}
					projection, err := ValidateSCVBundleEvaluation(a, b)
					if err != nil {
						t.Fatal("equivalent native JSON spelling rejected", err)
					}
					baseline, err := ValidateSCVBundleEvaluation(input, result)
					if err != nil || !reflect.DeepEqual(projection, baseline) {
						t.Fatal("equivalent spelling changed logical projection", err)
					}
				})
			}
		})
	}
}

func TestSCVBundleProjectionRejectsFullyResealedSemanticForgery(t *testing.T) {
	item := bundleProjectionFixtures(t)[3].(map[string]any)
	input := bundleTestMap(t, bundleTestRaw(t, item["input"]))
	native := scvTestClone(t, item["logical_result"].(map[string]any))
	native["entries"].([]any)[0].(map[string]any)["outcome"] = "runtime_verified"
	native = scvTestSeal(t, native)
	result := bundleTestResult(t, input, native)
	// The transport is self-consistent, so only the independent logical consumer
	// can reject this fabricated conclusion after expansion.
	if _, err := SCVBundleDecode(bundleTestRaw(t, result["result_bundle"])); err != nil {
		t.Fatal(err)
	}
	bundleProjectionFailure(t, bundleTestRaw(t, input), bundleTestRaw(t, result))
}

func TestSCVBundleProjectionRejectsNonTextNativeDigests(t *testing.T) {
	item := bundleProjectionFixtures(t)[0].(map[string]any)
	for name, digest := range map[string]any{"null": nil, "integer": 3, "object": map[string]any{}, "array": []any{}} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("untrusted digest type caused panic rather than rejection: %v", recovered)
				}
			}()
			input := bundleTestMap(t, bundleTestRaw(t, item["input"]))
			native := scvTestClone(t, item["logical_result"].(map[string]any))
			// The outer bundle is correctly content-addressed. Its decoded native
			// result and the matching wrapper attribution still need type checks.
			native["digest"] = digest
			result := bundleTestResult(t, input, native)
			if _, err := SCVBundleDecode(bundleTestRaw(t, result["result_bundle"])); err != nil {
				t.Fatal("test did not construct valid transport", err)
			}
			bundleProjectionFailure(t, bundleTestRaw(t, input), bundleTestRaw(t, result))
		})
	}
}

func TestSCVBundleProjectionPreservesExpansionPreflight(t *testing.T) {
	item := bundleProjectionFixtures(t)[0].(map[string]any)
	for _, mode := range []string{"fanout", "materialized"} {
		t.Run(mode, func(t *testing.T) {
			levels := 20
			if mode == "materialized" {
				levels = 50
			}
			input := scvTestClone(t, item["input"].(map[string]any))
			result := scvTestClone(t, item["result"].(map[string]any))
			input["bundle"] = bundleTestMap(t, bundleTestBudgetGraph(t, levels, mode))
			result["input_digest"], _ = SCVDigest(input)
			result["input_root_digest"] = input["bundle"].(map[string]any)["root_digest"]
			err := bundleProjectionFailure(t, bundleTestRaw(t, input), bundleTestReseal(t, result))
			if !strings.Contains(err.Error(), "budget") || strings.Contains(err.Error(), "content digest") {
				t.Fatal("projection failed after materialization instead of at preflight", err)
			}
		})
	}
}

func TestSCVBundleProjectionIsDetachedAndPerCall(t *testing.T) {
	items := bundleProjectionFixtures(t)
	item := items[0].(map[string]any)
	input, result := bundleTestRaw(t, item["input"]), bundleTestRaw(t, item["result"])
	projection, err := ValidateSCVBundleEvaluation(input, result)
	if err != nil {
		t.Fatal(err)
	}
	expectedInput, expectedResult := bytes.Clone(projection.Input), bytes.Clone(projection.Result)
	input[0], result[0] = '!', '!'
	if !bytes.Equal(projection.Input, expectedInput) || !bytes.Equal(projection.Result, expectedResult) {
		t.Fatal("projection aliases caller raw storage")
	}
	projection.Input[0], projection.Result[0] = '!', '!'
	for _, raw := range []any{items[3], items[0]} {
		current := raw.(map[string]any)
		got, err := ValidateSCVBundleEvaluation(bundleTestRaw(t, current["input"]), bundleTestRaw(t, current["result"]))
		if err != nil || !bytes.Equal(got.Input, bundleTestRaw(t, current["logical_input"])) || !bytes.Equal(got.Result, bundleTestRaw(t, current["logical_result"])) {
			t.Fatal("projection reused mutated storage or previous logical result", err)
		}
	}
	bundleProjectionFailure(t, input, result)
}

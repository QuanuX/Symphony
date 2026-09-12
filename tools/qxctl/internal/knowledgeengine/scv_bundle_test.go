package knowledgeengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

func bundleTestMap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	v, e := scvBundleObject(raw)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func bundleTestEncode(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, e := SCVCanonical(v)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := SCVBundleEncode(raw)
	if e != nil {
		t.Fatal(e)
	}
	return bundleTestMap(t, encoded)
}
func bundleTestRaw(t *testing.T, v any) []byte {
	t.Helper()
	raw, e := SCVCanonical(v)
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func bundleTestReseal(t *testing.T, v map[string]any) []byte {
	t.Helper()
	delete(v, "digest")
	id, e := SCVDigest(v)
	if e != nil {
		t.Fatal(e)
	}
	v["digest"] = id
	return bundleTestRaw(t, v)
}
func bundleTestRoot(t *testing.T, b map[string]any) map[string]any {
	t.Helper()
	for _, raw := range b["objects"].([]any) {
		n := raw.(map[string]any)
		if n["digest"] == b["root_digest"] {
			return n
		}
	}
	t.Fatal("missing fixture root")
	return nil
}

func TestSCVBundleCodecPreservesExactJSONAndDigestSubjects(t *testing.T) {
	raw := []byte(`{"a":{"values":[9007199254740991,-9007199254740991,-0,"0.000","\u2028","\\u2028","\ud83d\ude00"]},"b":{"values":[9007199254740991,-9007199254740991,0,"0.000","\u2028","\\u2028","\ud83d\ude00"]},"ref":{"scalar":{"ref":"ordinary data"}},"empty":{},"array":[]}`)
	encoded, err := SCVBundleEncode(raw)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := SCVBundleDecode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	expected := bundleTestMap(t, raw)
	if string(decoded) != string(bundleTestRaw(t, expected)) || strings.Contains(string(decoded), ",-0,") {
		t.Fatal("canonical round trip changed integer/source data")
	}
	b := bundleTestMap(t, encoded)
	root := bundleTestRoot(t, b)
	members := root["members"].(map[string]any)
	if !scvEqual(members["a"], members["b"]) {
		t.Fatal("equivalent normalized content did not deduplicate")
	}
	sealedNative := map[string]any{"protocol": "synthetic", "domain": "scv"}
	native := scvTestSeal(t, sealedNative)
	nb := bundleTestEncode(t, native)
	full, err := SCVDigest(native)
	if err != nil {
		t.Fatal(err)
	}
	if nb["root_digest"] != full || nb["root_digest"] == native["digest"] {
		t.Fatal("native self-seal confused with full transport content hash")
	}
}

func TestSCVBundleMetricsCountSharedOccurrences(t *testing.T) {
	b := bundleTestEncode(t, map[string]any{"a": []any{}, "b": []any{}})
	raw, metrics, err := scvBundleExpand(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"a":[],"b":[]}` {
		t.Fatal(string(raw))
	}
	expected := map[string]any{"object_count": 2, "reference_count": 2, "traversal_steps": 4, "expanded_bytes": 15, "expanded_values": 5, "expanded_depth": 1, "materialized_bytes": 17}
	if !scvEqual(metrics, expected) {
		t.Fatal("incorrect shared occurrence metrics", metrics)
	}
}

func TestSCVBundleCodecRejectsInvalidRawValues(t *testing.T) {
	cases := map[string][]byte{"duplicate": []byte(`{"x":1,"x":2}`), "float": []byte(`{"x":1.0}`), "exponent": []byte(`{"x":1e1}`), "range": []byte(`{"x":9007199254740992}`), "root_array": []byte(`[]`), "surrogate": []byte(`{"x":"\ud800"}`), "low_surrogate": []byte(`{"x":"\udc00"}`), "wrong_pair": []byte(`{"x":"\ud800\u1234"}`), "utf8": []byte{'{', '"', 'x', '"', ':', '"', 255, '"', '}'}, "string_bound": []byte(`{"x":"` + strings.Repeat("x", 65537) + `"}`), "depth": []byte(`{"x":` + strings.Repeat("[", 65) + `null` + strings.Repeat("]", 65) + `}`)}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := SCVBundleEncode(raw); err == nil {
				t.Fatal("invalid logical input admitted")
			}
		})
	}
}

func TestSCVBundleConsumerRejectsMalformedAndResealedClosure(t *testing.T) {
	original := bundleTestEncode(t, map[string]any{"a": map[string]any{"x": 1}, "b": []any{true}})
	cases := map[string]func(map[string]any){
		"extra_bundle":   func(b map[string]any) { b["other"] = true },
		"missing_bundle": func(b map[string]any) { delete(b, "objects") },
		"wrong_root":     func(b map[string]any) { b["root_digest"] = "sha256:" + strings.Repeat("f", 64) },
		"wrong_content_hash": func(b map[string]any) {
			root := bundleTestRoot(t, b)
			root["members"].(map[string]any)["a"] = map[string]any{"scalar": "invented"}
		},
		"unsorted":     func(b map[string]any) { a := b["objects"].([]any); a[0], a[1] = a[1], a[0] },
		"duplicate":    func(b map[string]any) { a := b["objects"].([]any); b["objects"] = append(a, a[0]) },
		"unknown_node": func(b map[string]any) { b["objects"].([]any)[0].(map[string]any)["kind"] = "source" },
		"extra_node":   func(b map[string]any) { b["objects"].([]any)[0].(map[string]any)["metadata"] = nil },
		"missing_ref": func(b map[string]any) {
			bundleTestRoot(t, b)["members"].(map[string]any)["a"] = map[string]any{"ref": "sha256:" + strings.Repeat("f", 64)}
		},
		"scalar_container": func(b map[string]any) {
			bundleTestRoot(t, b)["members"].(map[string]any)["a"] = map[string]any{"scalar": map[string]any{}}
		},
		"both_tags": func(b map[string]any) {
			bundleTestRoot(t, b)["members"].(map[string]any)["a"] = map[string]any{"scalar": 1, "ref": b["root_digest"]}
		},
		"cycle": func(b map[string]any) {
			bundleTestRoot(t, b)["members"].(map[string]any)["a"] = map[string]any{"ref": b["root_digest"]}
		},
		"unreachable": func(b map[string]any) {
			extra := map[string]any{"kind": "array", "items": []any{}, "digest": "sha256:" + strings.Repeat("e", 64)}
			a := append(b["objects"].([]any), extra)
			sort.Slice(a, func(i, j int) bool {
				return a[i].(map[string]any)["digest"].(string) < a[j].(map[string]any)["digest"].(string)
			})
			b["objects"] = a
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b := scvTestClone(t, original)
			mutate(b)
			if _, err := SCVBundleDecode(bundleTestReseal(t, b)); err == nil {
				t.Fatal("resealed invalid closure accepted")
			}
		})
	}
}

// Fake hashes are intentional: a bounded preflight must reject excessive graph
// expansion before allocating/hashing expanded values, not only after hash failure.
func bundleTestBudgetGraph(t *testing.T, levels int, mode string) []byte {
	t.Helper()
	nodes := []any{}
	id := func(i int) string { return fmt.Sprintf("sha256:%064x", i+1) }
	nodes = append(nodes, map[string]any{"digest": id(0), "kind": "array", "items": []any{map[string]any{"scalar": nil}}})
	for i := 1; i <= levels; i++ {
		items := []any{map[string]any{"ref": id(i - 1)}}
		if mode == "fanout" {
			items = append(items, map[string]any{"ref": id(i - 1)})
		}
		if mode == "materialized" {
			items = append(items, map[string]any{"scalar": strings.Repeat("x", 65000)})
		}
		nodes = append(nodes, map[string]any{"digest": id(i), "kind": "array", "items": items})
	}
	nodes = append(nodes, map[string]any{"digest": id(levels + 1), "kind": "object", "members": map[string]any{"x": map[string]any{"ref": id(levels)}}})
	b := map[string]any{"protocol": scvBundleProtocol, "objects": nodes, "root_digest": id(levels + 1)}
	return bundleTestReseal(t, b)
}
func TestSCVBundleExpansionBudgetsPrecedeMaterialization(t *testing.T) {
	for _, c := range []struct {
		name   string
		levels int
		mode   string
		want   string
	}{{"fanout", 20, "fanout", "budget"}, {"depth", 65, "chain", "depth"}, {"materialized", 50, "materialized", "budget"}} {
		t.Run(c.name, func(t *testing.T) {
			_, err := SCVBundleDecode(bundleTestBudgetGraph(t, c.levels, c.mode))
			if err == nil || !strings.Contains(err.Error(), c.want) || strings.Contains(err.Error(), "content digest") {
				t.Fatal("preflight did not reject before materialization", err)
			}
		})
	}
}

func bundleTestInvocation(t *testing.T, op string, input any) map[string]any {
	return map[string]any{"operation": op, "owner": map[string]any{"domain": "scv", "version": "0.9.0-dev"}, "bundle": bundleTestEncode(t, input)}
}
func bundleTestResult(t *testing.T, invocation map[string]any, native any) map[string]any {
	t.Helper()
	_, im, err := scvBundleExpand(invocation["bundle"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	rb := bundleTestEncode(t, native)
	_, rm, err := scvBundleExpand(rb)
	if err != nil {
		t.Fatal(err)
	}
	id, err := SCVDigest(invocation)
	if err != nil {
		t.Fatal(err)
	}
	v := map[string]any{"protocol": "symphony.scv.composition-bundle-evaluation.v1", "domain": "scv", "operation": invocation["operation"], "owner": invocation["owner"], "input_digest": id, "input_root_digest": invocation["bundle"].(map[string]any)["root_digest"], "input_metrics": im, "result_bundle": rb, "native_result_digest": native.(map[string]any)["digest"], "result_metrics": rm, "validation": "owner_evaluated", "limitations": scvBundleEvaluationLimitations}
	return scvTestSeal(t, v)
}
func TestSCVBundleIndependentConsumerRejectsResealedNativeOutcomes(t *testing.T) {
	raw, err := os.ReadFile("testdata/scv-obligation.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture := bundleTestMap(t, raw)
	inv := bundleTestInvocation(t, "composition_followup", fixture["followup_input"])
	result := bundleTestResult(t, inv, fixture["followup_result"])
	if err := validateSCVBundleResult("composition_bundle_evaluate", inv, result); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){"metrics": func(v map[string]any) { v["result_metrics"].(map[string]any)["expanded_bytes"] = 1 }, "owner": func(v map[string]any) { v["owner"] = map[string]any{"domain": "schv", "version": "0.9.0-dev"} }, "validation": func(v map[string]any) { v["validation"] = "certified" }, "native_seal": func(v map[string]any) { v["native_result_digest"] = v["result_bundle"].(map[string]any)["root_digest"] }, "input_root": func(v map[string]any) { v["input_root_digest"] = v["result_bundle"].(map[string]any)["root_digest"] }, "extra": func(v map[string]any) { v["other"] = true }} {
		t.Run(name, func(t *testing.T) {
			v := scvTestClone(t, result)
			mutate(v)
			v = scvTestSeal(t, v)
			if err := validateSCVBundleResult("composition_bundle_evaluate", inv, v); err == nil {
				t.Fatal("resealed wrapper mismatch accepted")
			}
		})
	}
	t.Run("fully_resealed_fabricated_outcome", func(t *testing.T) {
		native := scvTestClone(t, fixture["followup_result"].(map[string]any))
		native["entries"].([]any)[0].(map[string]any)["outcome"] = "runtime_verified"
		native = scvTestSeal(t, native)
		v := bundleTestResult(t, inv, native)
		if err := validateSCVBundleResult("composition_bundle_evaluate", inv, v); err == nil {
			t.Fatal("transport reconstruction bypassed semantic consumer")
		}
	})
}

func TestInstalledSCVBundleCodecAndFourLogicalOperations(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_BUNDLE_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed .9 owner")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("testdata/scv-obligation.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	f := bundleTestMap(t, raw)
	cases := map[string]any{"composition_explore": f["before"].(map[string]any)["input"], "composition_reassess": map[string]any{"before": f["before"], "after": f["after"]}, "composition_obligations": f["obligations_input"], "composition_followup": f["followup_input"]}
	for op, input := range cases {
		t.Run(op, func(t *testing.T) {
			inv := bundleTestInvocation(t, op, input)
			payload := bundleTestRaw(t, inv)
			for _, operation := range []string{"bundle_inspect", "composition_bundle_evaluate"} {
				response, err := InvokeSCVDomain(context.Background(), "scv", prefix, "0.9.0-dev", cwd, operation, payload)
				if err != nil {
					t.Fatal(operation, err)
				}
				result := bundleTestMap(t, response.Result)
				if operation == "composition_bundle_evaluate" {
					expanded, err := SCVBundleDecode(bundleTestRaw(t, result["result_bundle"]))
					if err != nil {
						t.Fatal(err)
					}
					if err = ValidateSCVResult(op, bundleTestRaw(t, input), expanded); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
	t.Run("native_scalar_parity", func(t *testing.T) {
		encoded, err := SCVBundleEncode([]byte(`{"a":-0,"b":9007199254740991,"c":-9007199254740991,"d":"\u2028","e":"\\u2028","ref":{"ref":"ordinary"}}`))
		if err != nil {
			t.Fatal(err)
		}
		inv := map[string]any{"operation": "composition_explore", "owner": map[string]any{"domain": "scv", "version": "0.9.0-dev"}, "bundle": bundleTestMap(t, encoded)}
		if _, err = InvokeSCVDomain(context.Background(), "scv", prefix, "0.9.0-dev", cwd, "bundle_inspect", bundleTestRaw(t, inv)); err != nil {
			t.Fatal(err)
		}
	})
}

// This file is preserved byte-for-byte from the independently authored C++
// bundle_test fixture emitter. Codec equality alone is not a semantic oracle;
// every operation result also passes the independent original Go consumer.
func TestSCVBundleNativeFixtureParity(t *testing.T) {
	path := os.Getenv("SYMPHONY_SCV_BUNDLE_FIXTURE")
	if path == "" {
		path = "testdata/scv-bundle.v1.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	codec := fixture["codec"].(map[string]any)
	encoded, err := SCVBundleEncode([]byte(codec["raw_input"].(string)))
	if err != nil {
		t.Fatal(err)
	}
	if !scvEqual(bundleTestMap(t, encoded), codec["bundle"]) {
		t.Fatal("Go and native codec encodings differ")
	}
	decoded, metrics, err := scvBundleExpand(bundleTestMap(t, encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !scvEqual(bundleTestMap(t, decoded), codec["input"]) || !scvEqual(metrics, codec["metrics"]) {
		t.Fatal("Go/native canonical content or accounting differs")
	}
	for _, raw := range fixture["operations"].([]any) {
		item := raw.(map[string]any)
		input := item["input"].(map[string]any)
		name := input["operation"].(string)
		t.Run(name, func(t *testing.T) {
			packed, err := SCVBundleEncode(bundleTestRaw(t, item["logical_input"]))
			if err != nil {
				t.Fatal(err)
			}
			if !scvEqual(bundleTestMap(t, packed), input["bundle"]) {
				t.Fatal("logical input native encoding differs")
			}
			if err = validateSCVBundleWireResult("bundle_inspect", bundleTestRaw(t, input), bundleTestRaw(t, item["inspection"])); err != nil {
				t.Fatal(err)
			}
			if err = validateSCVBundleWireResult("composition_bundle_evaluate", bundleTestRaw(t, input), bundleTestRaw(t, item["result"])); err != nil {
				t.Fatal(err)
			}
			expanded, err := SCVBundleDecode(bundleTestRaw(t, item["result"].(map[string]any)["result_bundle"]))
			if err != nil {
				t.Fatal(err)
			}
			if !scvEqual(bundleTestMap(t, expanded), item["logical_result"]) {
				t.Fatal("native result did not reconstruct exactly")
			}
		})
	}
}

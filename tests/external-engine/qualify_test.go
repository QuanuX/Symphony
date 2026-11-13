package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func panics(t *testing.T, contains string, fn func()) {
	t.Helper()
	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("expected refusal")
		}
		if !strings.Contains(toText(value), contains) {
			t.Fatalf("wrong refusal: %v", value)
		}
	}()
	fn()
}
func toText(value any) string {
	if err, ok := value.(error); ok {
		return err.Error()
	}
	if s, ok := value.(string); ok {
		return s
	}
	return "unknown"
}

func TestExactOracleIndependentArithmetic(t *testing.T) {
	got := oracle([]int64{-3, -3, 2, 4}, "0.1.0")
	expected := object{"protocol": "quanux.beta.empirical-price-result.v1", "engine_id": engineID, "engine_version": "0.1.0", "sample_count": "4",
		"support": []any{object{"price_nanos": "-3", "weight": object{"numerator": "1", "denominator": "2"}}, object{"price_nanos": "2", "weight": object{"numerator": "1", "denominator": "4"}}, object{"price_nanos": "4", "weight": object{"numerator": "1", "denominator": "4"}}}, "mean_price_nanos": object{"numerator": "0", "denominator": "1"}}
	if !equal(got, expected) {
		t.Fatalf("complete independent oracle mismatch: %s", canonical(got))
	}
}
func TestOracleTotalCannotOverflowInt64(t *testing.T) {
	got := oracle([]int64{math.MaxInt64, math.MaxInt64, -math.MaxInt64}, "0.1.1")
	if !equal(got["mean_price_nanos"], object{"numerator": "9223372036854775807", "denominator": "3"}) {
		t.Fatal(got)
	}
}
func TestOracleCanonicalDigestAndIntegers(t *testing.T) {
	got := canonical(decode([]byte(`{"z":9223372036854775807,"a":"<&>"}`)))
	if string(got) != `{"a":"<&>","z":9223372036854775807}` {
		t.Fatalf("canonical bytes changed: %s", got)
	}
	if _, ok := obj(decode(got))["z"].(json.Number); !ok {
		t.Fatal("integer precision lost")
	}
}
func TestOracleRefusesEmpty(t *testing.T) { panics(t, "observations", func() { oracle(nil, "0.1.0") }) }

func TestJSONRefusesInvalidUTF8(t *testing.T) {
	for _, input := range [][]byte{[]byte{'{', '"', 0xff, '"', ':', '1', '}'}, []byte{'[', '"', 0xc3, '"', ']'}} {
		panics(t, "invalid UTF-8", func() { decode(input) })
	}
	for _, value := range []any{object{string([]byte{0xff}): "key"}, []any{string([]byte{0xc3})}} {
		panics(t, "invalid UTF-8", func() { canonical(value) })
	}
}

func TestJSONUnicodePreservesCanonicalBytes(t *testing.T) {
	for _, input := range []string{`{"v":"\ud800"}`, `{"\udc00":1}`, `{"v":"\ud800\u0041"}`} {
		panics(t, "unpaired JSON surrogate", func() { decode([]byte(input)) })
	}
	input := []byte(`{"v":"\ud83d\ude00","literal":"\\u2028","line":"\u2028\u2029"}`)
	want := "{\"line\":\"\u2028\u2029\",\"literal\":\"\\\\u2028\",\"v\":\"😀\"}"
	if got := string(canonical(decode(input))); got != want {
		t.Fatalf("canonical Unicode differs: %q != %q", got, want)
	}
}

func seedSDK(t *testing.T) *campaign {
	t.Helper()
	c := &campaign{sdk: t.TempDir()}
	for _, identity := range [][2]string{{"knowledge-vector-engine-cpp", "0.2.0-dev"}, {"sbv-engine", "0.33.0-dev"}} {
		component, version := identity[0], identity[1]
		member := component + ".txt"
		must(os.WriteFile(filepath.Join(c.sdk, member), []byte("owned SDK bytes"), 0600))
		data := object{"component_id": component, "version": version, "files": []any{object{"path": member, "size": 15, "digest": "sha256:" + digest(filepath.Join(c.sdk, member))}}}
		sealReceipt(c, component, version, data)
	}
	return c
}
func sealReceipt(c *campaign, component, version string, data object) {
	delete(data, "receipt_digest")
	data["receipt_digest"] = "sha256:" + digestBytes(canonical(data))
	path := filepath.Join(c.sdk, "share/symphony/receipts", component, version, "install-receipt.json")
	must(os.MkdirAll(filepath.Dir(path), 0700))
	must(os.WriteFile(path, canonical(data), 0600))
}
func TestCompleteSDKByteBinding(t *testing.T) {
	c := seedSDK(t)
	before := c.verifySDK()
	if len(before) != 2 {
		t.Fatal(before)
	}
	if !reflect.DeepEqual(before, c.verifySDK()) {
		t.Fatal("unstable receipts")
	}
}
func TestSDKOwnedByteTamper(t *testing.T) {
	c := seedSDK(t)
	must(os.WriteFile(filepath.Join(c.sdk, "sbv-engine.txt"), []byte("changed bytes!!"), 0600))
	panics(t, "SDK owned byte mismatch", func() { c.verifySDK() })
}
func TestSDKReceiptDigestTamper(t *testing.T) {
	c := seedSDK(t)
	path := filepath.Join(c.sdk, "share/symphony/receipts/sbv-engine/0.33.0-dev/install-receipt.json")
	data := obj(readJSON(path))
	data["version"] = "0.32.0-dev"
	must(os.WriteFile(path, canonical(data), 0600))
	panics(t, "receipt digest mismatch", func() { c.verifySDK() })
}
func TestSDKWrongSealedIdentity(t *testing.T) {
	c := seedSDK(t)
	path := filepath.Join(c.sdk, "share/symphony/receipts/sbv-engine/0.33.0-dev/install-receipt.json")
	data := obj(readJSON(path))
	data["version"] = "0.32.0-dev"
	sealReceipt(c, "sbv-engine", "0.33.0-dev", data)
	panics(t, "exact SDK identity", func() { c.verifySDK() })
}
func TestSDKUnsafeReceiptPath(t *testing.T) {
	for _, path := range []string{"../outside", "/absolute/outside"} {
		t.Run(path, func(t *testing.T) {
			c := seedSDK(t)
			receipt := filepath.Join(c.sdk, "share/symphony/receipts/sbv-engine/0.33.0-dev/install-receipt.json")
			data := obj(readJSON(receipt))
			obj(array(data["files"])[0])["path"] = path
			sealReceipt(c, "sbv-engine", "0.33.0-dev", data)
			panics(t, "unsafe SDK receipt path", func() { c.verifySDK() })
		})
	}
}
func TestSDKSymlinkedMemberRefused(t *testing.T) {
	c := seedSDK(t)
	path := filepath.Join(c.sdk, "sbv-engine.txt")
	must(os.Rename(path, path+".original"))
	must(os.Symlink(path+".original", path))
	panics(t, "symlinked SDK receipt member", func() { c.verifySDK() })
}
func TestRemovalPreflightsAllOwnedBytes(t *testing.T) {
	root := t.TempDir()
	a, b, sentinel := filepath.Join(root, "a"), filepath.Join(root, "b"), filepath.Join(root, "sentinel")
	for _, path := range []string{a, b, sentinel} {
		must(os.WriteFile(path, []byte("retained"), 0600))
	}
	inventory := map[string]string{a: digest(a), b: digest(b)}
	must(os.WriteFile(b, []byte("changed"), 0600))
	if removal(inventory) == nil {
		t.Fatal("tampered removal passed")
	}
	if !exists(a) || !exists(b) || string(read(sentinel)) != "retained" {
		t.Fatal("preflight mutated a file")
	}
	must(os.WriteFile(b, []byte("retained"), 0600))
	must(removal(inventory))
	if exists(a) || exists(b) || string(read(sentinel)) != "retained" {
		t.Fatal("wrong removal ownership")
	}
}
func TestRemovalRefusesSymlink(t *testing.T) {
	root := t.TempDir()
	owner := filepath.Join(root, "owner")
	must(os.WriteFile(owner, []byte("bytes"), 0600))
	link := filepath.Join(root, "link")
	must(os.Symlink(owner, link))
	if removal(map[string]string{link: digest(owner)}) == nil || !exists(owner) || !exists(link) {
		t.Fatal("unsafe link removal")
	}
}
func TestEvidenceOutputIsExclusive(t *testing.T) {
	c := &campaign{}
	p := filepath.Join(t.TempDir(), "evidence.json")
	c.save(p, object{"first": true})
	before := digest(p)
	panics(t, "file exists", func() { c.save(p, object{"first": false}) })
	if digest(p) != before {
		t.Fatal("overwrote evidence")
	}
}
func TestExpectedRefusalNeedsExecutedCommand(t *testing.T) {
	work := t.TempDir()
	must(os.Mkdir(filepath.Join(work, "logs"), 0700))
	c := &campaign{work: work, env: os.Environ()}
	panics(t, "command could not execute", func() { c.run("missing", []string{filepath.Join(work, "absent")}, nil, false) })
	if len(c.checks) != 0 {
		t.Fatal("missing command counted as product refusal")
	}
}

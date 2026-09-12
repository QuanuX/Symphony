package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

func TestSCVBundlePackingPreservesCallerDataAndExplicitOwner(t *testing.T) {
	input := json.RawMessage(`{"provider":"caller-defined","requirements":[{"ref":"literal-data","value":9007199254740991}],"policy":{"allow_experimentation":true}}`)
	raw, err := knowledgeengine.SCVCanonical(map[string]any{"operation": "composition_explore", "input": input})
	if err != nil {
		t.Fatal(err)
	}
	packed, err := packSCVBundle(raw, scvOptions{domain: "schv", version: "0.9.0-dev"})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err = json.Unmarshal(packed, &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 || string(result["owner"]) != `{"domain":"schv","version":"0.9.0-dev"}` || string(result["operation"]) != `"composition_explore"` {
		t.Fatalf("caller selection changed: %s", packed)
	}
	expanded, err := knowledgeengine.SCVBundleDecode(result["bundle"])
	if err != nil {
		t.Fatal(err)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err = decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	expected, err := knowledgeengine.SCVCanonical(value)
	if err != nil || string(expanded) != string(expected) {
		t.Fatalf("input changed: %s; %v", expanded, err)
	}
	// Packing describes transport; native inspection/evaluation supplies validation.
	if result["validation"] != nil || result["result"] != nil {
		t.Fatal("packing claimed semantic validation")
	}
}

func bundleCLIFile(t *testing.T, raw []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func bundleCLIRaw(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSCVBundleRoutesRejectRawSurrogateBeforeOwnerInvocation(t *testing.T) {
	// The native content hashes name an actual U+FFFD scalar. Changing only its
	// wire spelling to an invalid surrogate must not be silently normalized.
	packed, err := packSCVBundle([]byte(`{"operation":"composition_explore","input":{"note":"�"}}`), scvOptions{domain: "scv", version: "0.9.0-dev"})
	if err != nil {
		t.Fatal(err)
	}
	invalid := bytes.ReplaceAll(packed, []byte("�"), []byte(`\ud800`))
	for _, action := range []string{"inspect", "evaluate", "expand"} {
		t.Run(action, func(t *testing.T) {
			err := runSCVBundle(action, scvOptions{input: bundleCLIFile(t, invalid), domain: "scv", version: "0.9.0-dev", prefix: filepath.Join(t.TempDir(), "absent-installation")})
			if err == nil || !strings.Contains(err.Error(), "surrogate") {
				t.Fatalf("raw invalid text reached owner lookup: %v", err)
			}
		})
	}
	for _, operation := range []string{"bundle_inspect", "composition_bundle_evaluate"} {
		t.Run("artifact-import-"+operation, func(t *testing.T) {
			raw := bundleCLIRaw(t, map[string]any{"operation": operation, "input": json.RawMessage(invalid), "result": nil})
			err := runSCVArtifact("import", scvOptions{input: bundleCLIFile(t, raw), domain: "scv", version: "0.9.0-dev", topsID: ssiagTestTOPSID}, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), "surrogate") {
				t.Fatalf("invalid new artifact input reached owner lookup: %v", err)
			}
		})
	}
}

func TestSCVBundleRawGuardPreservesValidUnicodeAndLegacyScope(t *testing.T) {
	for _, raw := range []string{`{"note":"�"}`, `{"note":"\ufffd"}`, `{"note":"\ud83d\ude00"}`, `{"note":"\\ud800"}`, `{"note":"\u2028\u2029","number":-0}`} {
		if err := knowledgeengine.ValidateSCVBundleText([]byte(raw)); err != nil {
			t.Fatalf("valid literal/pair/reference-looking data rejected: %s %v", raw, err)
		}
	}
	for _, raw := range []string{`{"note":"\ud800"}`, `{"note":"\udc00"}`, `{"note":"\ud800\u1234"}`, `{"x":1,"x":2}`, `{"x":1.0}`} {
		if err := knowledgeengine.ValidateSCVBundleText([]byte(raw)); err == nil {
			t.Fatalf("invalid raw text accepted: %s", raw)
		}
	}
	// The guard applies to the two new operations only, preserving the existing
	// generic import decoder behavior for historical operations.
	path := bundleCLIFile(t, []byte(`{"operation":"profile_prepare","input":{"note":"\ud800"},"result":null}`))
	if _, err := scvArtifactInput("import", scvOptions{input: path}); err != nil {
		t.Fatalf("new guard changed legacy artifact input behavior: %v", err)
	}
}

func bundleCLICapture(t *testing.T, action func() error) []byte {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout-")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	previous := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = previous }()
	err = action()
	os.Stdout = previous
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestInstalledSCVBundleRoutesPreserveValidReplacementAndSurrogatePair(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_BUNDLE_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed .9 bundle owner")
	}
	raw, err := os.ReadFile("../../internal/knowledgeengine/testdata/scv-bundle.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	logical := fixture["operations"].([]any)[0].(map[string]any)["logical_input"].(map[string]any)
	logical["requirements"].([]any)[0].(map[string]any)["resolution"].(map[string]any)["description"] = "Retain � and 😀 exactly; opaque caller text."
	options := scvOptions{domain: "scv", version: "0.9.0-dev", prefix: prefix, topsID: ssiagTestTOPSID}
	packed, err := packSCVBundle(bundleCLIRaw(t, map[string]any{"operation": "composition_explore", "input": logical}), options)
	if err != nil {
		t.Fatal(err)
	}
	// Literal replacement is valid. A properly paired escaped surrogate is also
	// valid and has exactly the same canonical content identity as the emoji.
	valid := bytes.ReplaceAll(packed, []byte("😀"), []byte(`\ud83d\ude00`))
	options.input = bundleCLIFile(t, valid)
	for _, action := range []string{"inspect", "evaluate", "expand"} {
		t.Run(action, func(t *testing.T) {
			result := bundleCLICapture(t, func() error { return runSCVBundle(action, options) })
			if !json.Valid(result) {
				t.Fatalf("invalid output: %s", result)
			}
			if action == "expand" && (!bytes.Contains(result, []byte("�")) || !bytes.Contains(result, []byte("😀"))) {
				t.Fatal("expanded caller text did not retain exact Unicode values")
			}
		})
	}
	for _, operation := range []string{"bundle_inspect", "composition_bundle_evaluate"} {
		t.Run("artifact-import-"+operation, func(t *testing.T) {
			options.input = bundleCLIFile(t, bundleCLIRaw(t, map[string]any{"operation": operation, "input": json.RawMessage(valid), "result": nil}))
			store := workflowStore(t)
			result := bundleCLICapture(t, func() error { return runSCVArtifact("import", options, store.Root) })
			if !json.Valid(result) || !bytes.Contains(result, []byte("�")) || !bytes.Contains(result, []byte("😀")) {
				t.Fatal("retained bundle input lost exact valid Unicode data")
			}
		})
	}
}

func TestSCVBundlePackingRejectsWrongEnvelopeAndOperation(t *testing.T) {
	for _, raw := range []string{
		`{"operation":"source_validate","input":{}}`,
		`{"operation":"composition_bundle_evaluate","input":{}}`,
		`{"operation":"composition_explore","input":{},"owner":{}}`,
		`{"operation":"composition_explore"}`,
		`{"operation":"composition_explore","input":[]}`,
		`{"operation":"composition_explore","input":null}`,
	} {
		if _, err := packSCVBundle([]byte(raw), scvOptions{domain: "scv", version: "0.9.0-dev"}); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestInstalledSCVBundlePackOutputRoundtripsAtFileBound(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_BUNDLE_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed .9 bundle owner")
	}
	// Deliberately synthetic transport-only data near the native request bound.
	// Its compact invocation fits, while pretty indentation alone would not.
	input := map[string]any{}
	for i := 0; i < 64; i++ {
		input[fmt.Sprintf("k%02d", i)] = strings.Repeat("x", 16330)
	}
	options := scvOptions{domain: "scv", version: "0.9.0-dev", prefix: prefix}
	options.input = bundleCLIFile(t, bundleCLIRaw(t, map[string]any{"operation": "composition_explore", "input": input}))
	packed := bundleCLICapture(t, func() error { return runSCVBundle("pack", options) })
	if len(packed) > 1<<20 {
		t.Fatalf("ready packed output exceeds input bound: %d", len(packed))
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, packed, "", "  "); err != nil {
		t.Fatal(err)
	}
	if pretty.Len() <= 1<<20 {
		t.Fatal("fixture does not exercise formatting overrun")
	}
	options.input = bundleCLIFile(t, packed)
	inspected := bundleCLICapture(t, func() error { return runSCVBundle("inspect", options) })
	var result map[string]any
	if err := json.Unmarshal(inspected, &result); err != nil {
		t.Fatal(err)
	}
	if result["validation"] != "transport_only" {
		t.Fatal("inspection claimed semantic evaluation")
	}
}

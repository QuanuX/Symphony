package knowledgeengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSCVExactReceiptRejectsTamperingAndLegacy(t *testing.T) {
	prefix := t.TempDir()
	spec, err := scvSpec("schv-aws")
	if err != nil {
		t.Fatal(err)
	}
	spec.expectedFiles = func(version string) map[string]struct{} {
		return map[string]struct{}{filepath.ToSlash(filepath.Join("libexec", "symphony", spec.moduleID, version, spec.engineID)): {}}
	}
	receiptPath, receipt := createInstalledV2Fixture(t, spec, prefix, "0.1.0-dev")
	installed, err := InspectSCVDomain("schv-aws", prefix, "0.1.0-dev")
	if err != nil {
		t.Fatal(err)
	}
	if installed.ReceiptDigest != receipt.ReceiptDigest || installed.EngineID != "symphony-schv-aws" {
		t.Fatal("receipt identity mismatch")
	}
	if _, err := InspectSCVDomain("scv", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("wrong domain receipt accepted")
	}
	if _, err := InspectSCVDomain("schv-aws", prefix, "latest"); err == nil {
		t.Fatal("uninstalled version accepted")
	}
	if err := os.WriteFile(installed.ExecutablePath, []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSCVDomain("schv-aws", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("tampered executable accepted")
	}
	if err := os.WriteFile(receiptPath, []byte(`{"protocol":"symphony.knowledge.install-receipt.v1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSCVDomain("schv-aws", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("legacy receipt accepted for new SCV engine")
	}
}

func TestInstalledSCVDomainDescriptors(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_ACCEPTANCE_PREFIX")
	if prefix == "" {
		t.Skip("requires exact staged SCV domain installations")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range SCVDomains() {
		t.Run(domain, func(t *testing.T) {
			response, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.1.0-dev", cwd, "inspect", []byte(`{}`))
			if err != nil {
				t.Fatal(err)
			}
			if response.EngineID != "symphony-"+domain {
				t.Fatal("domain selected wrong process")
			}
		})
	}
}

func TestInstalledSCVRetainsExactOldAndNewOperationSets(t *testing.T) {
	oldPrefix, newPrefix := os.Getenv("SYMPHONY_SCV_ACCEPTANCE_PREFIX"), os.Getenv("SYMPHONY_SCV_CORPUS_PREFIX")
	if oldPrefix == "" || newPrefix == "" {
		t.Skip("requires retained 0.1.0-dev and separate 0.2.0-dev installations")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	installations := []struct {
		prefix, version string
		count           int
	}{{oldPrefix, "0.1.0-dev", 13}, {newPrefix, "0.2.0-dev", 17}}
	if third := os.Getenv("SYMPHONY_SCV_INTERPRETATION_PREFIX"); third != "" {
		installations = append(installations, struct {
			prefix, version string
			count           int
		}{third, "0.3.0-dev", 20})
	}
	for _, domain := range SCVDomains() {
		for _, installed := range installations {
			t.Run(domain+"/"+installed.version, func(t *testing.T) {
				response, err := InvokeSCVDomain(context.Background(), domain, installed.prefix, installed.version, cwd, "inspect", []byte(`{}`))
				if err != nil {
					t.Fatal(err)
				}
				value, err := scvObject(response.Result)
				if err != nil {
					t.Fatal(err)
				}
				if len(value["operations"].([]any)) != installed.count {
					t.Fatal("exact version operation surface changed")
				}
			})
		}
		if _, err := InvokeSCVDomain(context.Background(), domain, oldPrefix, "0.1.0-dev", cwd, "corpus_build", []byte(`{}`)); err == nil {
			t.Fatal("new corpus operation silently substituted for old version")
		}
	}
}

func TestSCVOperationVersionsAreExplicit(t *testing.T) {
	for _, version := range []string{"latest", "0.4.0-dev", "", "0.1.0"} {
		if SCVOperationSupported(version, "inspect") {
			t.Fatalf("unsupported exact version %s accepted", version)
		}
	}
	if SCVOperationSupported("0.1.0-dev", "capture_index") || !SCVOperationSupported("0.2.0-dev", "capture_index") ||
		SCVOperationSupported("0.2.0-dev", "provider_interpret") || !SCVOperationSupported("0.3.0-dev", "provider_interpret") {
		t.Fatal("version operation boundary broadened")
	}
}

func captureResult(t *testing.T) ([]byte, map[string]any) {
	t.Helper()
	payload := map[string]any{"source": map[string]any{"source_id": "docs"}, "locator_id": "docs", "resolved_uri": "https://example.com/",
		"redirects": []string{}, "observed_at": "2026-09-10T00:00:00Z", "upstream_revision": nil, "media_type": "text/plain",
		"body": "source text", "completeness": "complete", "issues": []string{}}
	input, _ := SCVCanonical(payload)
	result := map[string]any{}
	for key, value := range payload {
		result[key] = value
	}
	body := []byte("source text")
	hash := sha256.Sum256(body)
	result["protocol"] = "symphony.scv.capture.v1"
	result["body_digest"] = "sha256:" + hex.EncodeToString(hash[:])
	result["byte_size"] = len(body)
	result["digest"], _ = SCVDigest(result)
	return input, result
}

func TestSCVResultBoundaryRejectsChangedCaptureAndUnknownFields(t *testing.T) {
	input, result := captureResult(t)
	raw, _ := SCVCanonical(result)
	if err := ValidateSCVResult("capture_import", input, raw); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(map[string]any){
		func(v map[string]any) { v["body"] = "changed" }, func(v map[string]any) { v["byte_size"] = 123 },
		func(v map[string]any) { v["source"] = nil }, func(v map[string]any) { v["unknown"] = true },
		func(v map[string]any) { v["protocol"] = "symphony.scv.graph.v1" }, func(v map[string]any) { v["body"] = strings.Repeat("a", 65537) },
	} {
		_, value := captureResult(t)
		mutate(value)
		delete(value, "digest")
		value["digest"], _ = SCVDigest(value)
		encoded, _ := SCVCanonical(value)
		if err := ValidateSCVResult("capture_import", input, encoded); err == nil {
			t.Fatal("re-sealed changed capture accepted")
		}
	}
}

func TestSCVCanonicalPreservesUTF8AndLiteralEscapes(t *testing.T) {
	value := map[string]any{"line": "a\u2028b\u2029c", "literal": `\u2028`, "maximum": json.Number("9007199254740991"), "html": "<a>&"}
	encoded, err := SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte("a\u2028b\u2029c")) || !bytes.Contains(encoded, []byte(`"literal":"\\u2028"`)) {
		t.Fatalf("canonical JSON drift: %s", encoded)
	}
	var decoded map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["literal"] != value["literal"] || decoded["maximum"] != value["maximum"] {
		t.Fatal("canonicalization changed evidence")
	}
}

func TestSCVEmptyQueryRequiresExactAbsenceQualification(t *testing.T) {
	graph := map[string]any{"digest": "sha256:" + strings.Repeat("a", 64), "selection_policy": map[string]any{"policy_id": "fixture"}}
	input := map[string]any{"graph": graph, "query_time": "2026-09-10T00:00:00Z"}
	value := map[string]any{"protocol": "symphony.scv.query-result.v1", "domain": "scv", "graph_digest": graph["digest"], "query_time": input["query_time"],
		"selection_policy": graph["selection_policy"], "findings": []any{}, "conflicts": []any{}, "evidence_origins": []any{}, "limitations": []any{}, "coverage": map[string]any{},
		"query_selection": map[string]any{"query_time": input["query_time"]}, "absence": "no matching claim in the bounded selected corpus; no availability conclusion"}
	value["digest"], _ = SCVDigest(value)
	raw, _ := SCVCanonical(value)
	payload, _ := SCVCanonical(input)
	if err := ValidateSCVResult("graph_query", payload, raw); err != nil {
		t.Fatal(err)
	}
	value["absence"] = "provider does not offer this"
	delete(value, "digest")
	value["digest"], _ = SCVDigest(value)
	raw, _ = SCVCanonical(value)
	if err := ValidateSCVResult("graph_query", payload, raw); err == nil {
		t.Fatal("unbounded absence claim accepted")
	}
	delete(value, "absence")
	delete(value, "digest")
	value["digest"], _ = SCVDigest(value)
	raw, _ = SCVCanonical(value)
	if err := ValidateSCVResult("graph_query", payload, raw); err == nil {
		t.Fatal("missing absence qualifier accepted")
	}
}

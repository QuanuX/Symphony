package knowledgeengine

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func partitionedCatalogue(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "modules", "sbv-engine", "schemas", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	value, err := sbvDecodeOwnedResource(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestSBVPartitionedRegisteredShapes(t *testing.T) {
	catalogue := partitionedCatalogue(t, "admin.schema.json")
	templates := partitionedCatalogue(t, "admin.templates.json")
	requests := catalogue["requests"].(map[string]any)
	results := catalogue["results"].(map[string]any)
	variants := templates["variants"].(map[string]any)
	legacy := templates["templates"].(map[string]any)
	for op, value := range variants {
		t.Run(op, func(t *testing.T) {
			p := value.(map[string]any)["partitioned"].(map[string]any)
			shape := requests[op].(map[string]any)
			if !sbvInputShape(shape, p) || sbvPartitionedRequestPreflight(op, p) != nil {
				t.Fatal("complete partitioned template rejected")
			}
			old := recoveryClone(t, legacy[op].(map[string]any))
			if op == "evaluate" {
				// Preserve the legacy template's intentional replacement marker;
				// admit a populated instance rather than its nonnumeric example.
				old["census"].(map[string]any)["signals"].([]any)[0].(map[string]any)["available_ns"] = "0"
			}
			if !sbvInputShape(shape, old) {
				t.Fatal("legacy template changed")
			}
			for _, mutate := range []func(map[string]any){
				func(m map[string]any) { m["output_path"] = "/private/legacy.json" },
				func(m map[string]any) { delete(m["output"].(map[string]any), "workspace_path") },
				func(m map[string]any) { m["output"].(map[string]any)["automatic_cleanup"] = true },
			} {
				bad := recoveryClone(t, p)
				mutate(bad)
				if sbvInputShape(shape, bad) {
					t.Fatal("invalid output alternative admitted")
				}
			}
		})
	}
	get := func(op string) map[string]any {
		return recoveryClone(t, variants[op].(map[string]any)["partitioned"].(map[string]any))
	}
	accept := func(op string, p map[string]any, want bool) {
		t.Helper()
		if got := sbvInputShape(requests[op].(map[string]any), p); got != want {
			t.Fatalf("%s schema admitted=%v want=%v", op, got, want)
		}
	}
	p := get("evaluate")
	source := partitionedSourceFixture()
	p["model"] = map[string]any{"protocol": "symphony.sbv.model-selection.v1", "id": "external_outcomes", "version": "1", "horizon_ns": "10", "parameters": map[string]any{
		"producer": map[string]any{"id": "caller", "version": "1", "artifact_sha256": "", "reproducibility": "uncaptured"}, "measure": "signed_coefficient", "conditioning": "scenario", "calibration_reference": "", "outcomes_source": source,
	}}
	accept("evaluate", p, true)
	p["model"].(map[string]any)["parameters"].(map[string]any)["outcomes"] = []any{}
	accept("evaluate", p, false)
	p = get("evaluate")
	delete(p, "output")
	p["output_path"] = "/private/legacy.json"
	accept("evaluate", p, false) // No implicit upgrade of the legacy reader.
	p = get("economics")
	for _, selection := range []map[string]any{
		{"kind": "all"}, {"kind": "range", "first": "0", "count": nil},
		{"kind": "range", "first": "0", "count": "0"},
		{"kind": "ids", "ids": []any{}, "order": "census"},
		{"kind": "referenced_ids", "source": source, "order": "supplied"},
	} {
		p["selections"].(map[string]any)["signals"] = selection
		accept("economics", p, true)
	}
	p["selections"] = map[string]any{"kind": "referenced_rows", "source": source}
	accept("economics", p, true)
	p["path"], p["expected_sha256"] = "/private/mixed.json", "hash"
	accept("economics", p, false)
	p = get("economics")
	p["selections"].(map[string]any)["signals"] = map[string]any{"kind": "ids", "ids": []any{"same", "same"}, "order": "supplied"}
	accept("economics", p, false)
	p = get("dataset_execute")
	p["request"].(map[string]any)["output_path"] = "/private/child.json"
	accept("dataset_execute", p, false)
	p = get("dataset_execute")
	p["operation"] = "book"
	accept("dataset_execute", p, false)
	p = get("run")
	delete(p, "output")
	p["output_path"] = "/private/legacy.json"
	accept("run", p, true) // Explicit nullable selection cap also migrated.

	_, receipt := partitionedFixture(t)
	_, recovery := partitionedRecoveryFixture(t)
	for _, value := range []map[string]any{receipt, recovery} {
		if !sbvSchemaShape(results["generate_census"], value, 0) {
			t.Fatal("typed partitioned result schema mismatch")
		}
	}
	delete(recovery, "cause")
	if sbvSchemaShape(results["generate_census"], recovery, 0) {
		t.Fatal("recovery missing cause admitted")
	}
	for op, shape := range requests {
		combined := map[string]any{"input": shape, "output": results[op], "$defs": catalogue["$defs"]}
		if _, err := sbvCanonicalControlResource(combined); err != nil {
			t.Fatalf("%s discoverable schema exceeds control frame: %v", op, err)
		}
	}
}

func TestSBVPartitionedOwnedCatalogueByteScope(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "modules", "sbv-engine", "schemas", "v1", "admin.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) <= maxRequestBytes {
		t.Fatal("fixture must cross old catalogue byte ceiling")
	}
	if _, err = sbvDecodeOwnedResource(raw); err != nil {
		t.Fatal(err)
	}
	if _, err = sqavObject(raw, maxRequestBytes); err == nil {
		t.Fatal("untrusted control admission widened")
	}
	// Each explicit request template remains a small portable control value.
	templates := partitionedCatalogue(t, "admin.templates.json")
	for _, entries := range templates["variants"].(map[string]any) {
		for _, value := range entries.(map[string]any) {
			encoded, _ := json.Marshal(value)
			if _, err = sqavObject(encoded, maxRequestBytes); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestSBVPartitionedOwnedResourceExactBytePin(t *testing.T) {
	prefix := t.TempDir()
	relative := "share/symphony/schemas/sbv-engine/test/admin.schema.json"
	receiptPath := "share/symphony/receipts/sbv-engine/test/install-receipt.json"
	data := bytes.Repeat([]byte("x"), maxRequestBytes+17)
	digest := digestBytes(data)
	inst := Installation{Prefix: prefix, ModuleID: "sbv-engine", Version: "test", ReceiptDigest: "pinned-test-receipt"}
	receipt := receiptV2{ReceiptDigest: inst.ReceiptDigest, Files: []receiptV2File{{Path: relative, Kind: "regular", Size: uint64(len(data)), Digest: digest}}}
	write := func(path string, value []byte) {
		t.Helper()
		full := filepath.Join(prefix, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, value, 0600); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := json.Marshal(receipt)
	write(receiptPath, raw)
	write(relative, data)
	actual, err := sbvOwnedResourceBytes(inst, relative, digest)
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatal("exact owned bytes beyond message limit rejected", err)
	}
	if _, err := sbvOwnedResourceBytes(inst, relative, "sha256:wrong"); err == nil {
		t.Fatal("wrong compiled resource pin admitted")
	}
	write(relative, append(append([]byte{}, data...), 'x'))
	if _, err := sbvOwnedResourceBytes(inst, relative, digest); err == nil {
		t.Fatal("read exceeded exact receipt byte admission")
	}
	write(relative, data)
	for _, size := range []uint64{uint64(math.MaxInt64), uint64(math.MaxInt64) + 1, math.MaxUint64} {
		receipt.Files[0].Size = size
		raw, _ = json.Marshal(receipt)
		write(receiptPath, raw)
		if _, err := sbvOwnedResourceBytes(inst, relative, digest); err == nil || !strings.Contains(err.Error(), "size representation mismatch") {
			t.Fatalf("size %d did not reject before extra-byte read: %v", size, err)
		}
	}
	receipt.Files[0].Size = uint64(len(data))
	receipt.ReceiptDigest = "changed-receipt"
	raw, _ = json.Marshal(receipt)
	write(receiptPath, raw)
	if _, err := sbvOwnedResourceBytes(inst, relative, digest); err == nil {
		t.Fatal("changed receipt admitted")
	}
}

package knowledgeengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Read-only installed admission: never dispatches the selected mutation.
func TestSBVInstalledResourceAdmission(t *testing.T) {
	prefix, input := os.Getenv("SYMPHONY_SBV_RESOURCE_PREFIX"), os.Getenv("SYMPHONY_SBV_RESOURCE_INPUT")
	if prefix == "" {
		t.Skip("selected installed resource evidence not configured")
	}
	if _, err := InspectSBV(prefix, SBVAdministrationInterfaceVersion); err != nil {
		t.Fatalf("installation admission: %v", err)
	}
	t.Log("receipt, entrypoint and owner-interface admission passed")
	if input != "" {
		_, schema, err := SBVResource(prefix, SBVAdministrationInterfaceVersion, "bundle_import", false)
		if err != nil {
			t.Fatalf("owned schema catalogue selection: %v", err)
		}
		raw, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		request, err := sqavObject(raw, maxRequestBytes)
		if err != nil {
			t.Fatalf("request transport: %v", err)
		}
		shape, err := sqavObject(schema, maxRequestBytes)
		if err != nil || !sbvInputShape(shape, request) {
			t.Fatalf("selected input shape: %v", err)
		}
	}
	for _, op := range SBVOperations {
		for _, template := range []bool{false, true} {
			_, raw, err := SBVResource(prefix, SBVAdministrationInterfaceVersion, op, template)
			if err != nil {
				t.Fatalf("resource %s template=%v: %v", op, template, err)
			}
			if _, err := sqavObject(raw, maxRequestBytes); err != nil {
				t.Fatalf("selected resource %s template=%v transport: %v", op, template, err)
			}
		}
		if _, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, op); err != nil {
			t.Fatalf("discoverable combined schema %s: %v", op, err)
		}
	}
	for _, op := range []string{"run", "generate_census", "evaluate", "economics", "compose_economics", "dataset_execute"} {
		_, raw, err := SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, op, "partitioned")
		if err != nil {
			t.Fatalf("partitioned template %s: %v", op, err)
		}
		p, err := sqavObject(raw, maxRequestBytes)
		if err != nil || sbvPartitionedRequestPreflight(op, p) != nil {
			t.Fatalf("partitioned template %s transport/preflight: %v", op, err)
		}
		_, schema, err := SBVResource(prefix, SBVAdministrationInterfaceVersion, op, false)
		if err != nil {
			t.Fatal(err)
		}
		shape, err := sqavObject(schema, maxRequestBytes)
		if err != nil || !sbvInputShape(shape, p) {
			t.Fatalf("partitioned template %s registered shape: %v", op, err)
		}
	}
	if _, _, err := SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, "book", "partitioned"); err == nil {
		t.Fatal("unimplemented partitioned book template admitted")
	}
	t.Logf("all %d schemas/default templates and six partitioned variants admitted within individual control bounds; no operation invoked", len(SBVOperations))
}

func TestSBVOwnedCatalogueExceedsMessageCount(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "modules", "sbv-engine", "schemas", "v1", "admin.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if digestBytes(raw) != sbvAdministrationInterfaceResources["admin.schema.json"] {
		t.Fatal("source catalogue differs from compiled exact resource pin")
	}
	if err := validateJSONObject(raw, maxRequestBytes); err == nil || (!strings.Contains(err.Error(), "value count") && !strings.Contains(err.Error(), "byte bound")) {
		t.Fatalf("regression requires a catalogue larger than one process message: %v", err)
	}
	m, err := sbvDecodeOwnedResource(raw)
	if err != nil {
		t.Fatal(err)
	}
	for op, shape := range m["requests"].(map[string]any) {
		selected, err := SCVCanonical(shape)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqavObject(selected, maxRequestBytes); err != nil {
			t.Fatalf("individual request schema %s: %v", op, err)
		}
	}
	// Enlarging the owned catalogue admission must not change untrusted control
	// parsing, including the exact same bytes passed as a request.
	if _, err := sqavObject(raw, maxRequestBytes); err == nil {
		t.Fatal("message admission was widened")
	}
}

func TestSBVOwnedCatalogueRetainsStructuralChecks(t *testing.T) {
	for name, raw := range map[string][]byte{
		"duplicate":          []byte(`{"a":0,"a":1}`),
		"invalid_utf8":       {'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'},
		"unpaired_surrogate": []byte(`{"x":"\ud800"}`),
		"fraction":           []byte(`{"x":1.5}`),
		"large_integer":      []byte(`{"x":9223372036854775808}`),
		"depth":              []byte(strings.Repeat(`{"x":`, maxJSONDepth+1) + `null` + strings.Repeat(`}`, maxJSONDepth+1)),
		"string":             []byte(`{"x":"` + strings.Repeat("x", maxStringBytes+1) + `"}`),
		"bytes":              []byte(`{"x":"` + strings.Repeat("x", maxRequestBytes) + `"}`),
		"array_root":         []byte(`[]`),
		"second_document":    []byte(`{} {}`),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := sbvDecodeOwnedResource(raw); err == nil {
				t.Fatal("malformed resource admitted")
			}
		})
	}
}

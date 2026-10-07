package knowledgeengine

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Installed artifact fixtures are supplied explicitly at the increment gate.
// This checks the published schema vocabulary against real owner responses and
// retained portable sections; it is not a general JSON Schema implementation.
func TestSBVInstalledOwnerEvidenceSchema(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_OWNER_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("select installed owner-route evidence")
	}
	raw, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "source_retain")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(raw, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs, ok := contract["$defs"].(map[string]any)
	if !ok {
		t.Fatal("missing installed definitions")
	}
	var expand func(any, int) any
	expand = func(v any, depth int) any {
		if depth > 64 {
			t.Fatal("schema cycle")
		}
		switch x := v.(type) {
		case map[string]any:
			if ref, ok := x["$ref"].(string); ok {
				key := strings.TrimPrefix(ref, "#/$defs/")
				target, found := defs[key]
				if !found {
					t.Fatal("unresolved definition", ref)
				}
				return expand(target, depth+1)
			}
			out := map[string]any{}
			for k, item := range x {
				out[k] = expand(item, depth+1)
			}
			return out
		case []any:
			out := make([]any, len(x))
			for i, item := range x {
				out[i] = expand(item, depth+1)
			}
			return out
		default:
			return v
		}
	}
	contracts := map[string]string{"symphony.sbv.retained-source.v1": "retained_source_evidence", "symphony.sbv.source-delivery.v1": "source_delivery_evidence", "symphony.sbv.source-owner-profile.v1": "source_owner_profile", "symphony.sbv.source-binary-receipt.v1": "source_binary_receipt"}
	shapes := map[string]any{}
	for _, key := range contracts {
		shapes[key] = expand(defs[key], 0)
	}
	shapes["dataset_feed"] = expand(defs["dataset_feed"], 0)
	for _, key := range []string{"source_retention_recovery", "source_export_recovery"} {
		shapes[key] = expand(defs[key], 0)
	}
	seen := map[string]int{}
	var inspect func(any, string)
	inspect = func(v any, path string) {
		switch x := v.(type) {
		case map[string]any:
			protocol, _ := x["protocol"].(string)
			key := contracts[protocol]
			if x["status"] == "recovery_required" {
				if protocol == "symphony.sbv.source-retain.v1" {
					key = "source_retention_recovery"
				}
				if protocol == "symphony.sbv.source-export.v1" {
					key = "source_export_recovery"
				}
			}
			if key != "" {
				if !sbvSchemaShape(shapes[key], x, 0) {
					t.Fatalf("%s does not match %s", path, key)
				}
				seen[key]++
			}
			for k, item := range x {
				if k == "dataset_feed" {
					if !sbvSchemaShape(shapes[k], item, 0) {
						t.Fatal(path, "dataset feed schema mismatch")
					}
					seen[k]++
				}
				inspect(item, path+"/"+k)
			}
		case []any:
			for _, item := range x {
				inspect(item, path+"/[]")
			}
		}
	}
	err = filepath.WalkDir(fixture, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if compact := bytes.TrimSpace(b); len(compact) != 0 && compact[0] == '[' && json.Valid(b) {
			return nil // Auxiliary comparison-path lists contain no owner payload.
		}
		v, e := sqavObject(b, 128<<20)
		if e != nil {
			return e
		}
		inspect(v, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("SYMPHONY_SBV_RECOVERY_FIXTURE"); dir != "" {
		for _, name := range []string{"export-confirmed", "export-unconfirmed", "retain-confirmed"} {
			path := filepath.Join(dir, name+"-recovery.json")
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			v, err := sqavObject(b, maxResponseBytes)
			if err != nil {
				t.Fatal(err)
			}
			inspect(v, path)
		}
	}
	for _, key := range contracts {
		if seen[key] == 0 {
			t.Fatal("missing fixture contract", key)
		}
	}
	if seen["dataset_feed"] == 0 {
		t.Fatal("missing dataset feed evidence")
	}
	t.Logf("validated retained owner schema occurrences: %v", seen)
}

func TestSBVNativeRecoveryFixtures(t *testing.T) {
	dir := os.Getenv("SYMPHONY_SBV_RECOVERY_FIXTURE")
	if dir == "" {
		t.Skip("select native emitted recovery fixtures")
	}
	for _, name := range []string{"export-confirmed", "export-unconfirmed", "retain-confirmed"} {
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(dir, name+"-request.json"))
			if err != nil {
				t.Fatal(err)
			}
			p, err := sqavObject(b, maxRequestBytes)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, name+"-recovery.json"))
			if err != nil {
				t.Fatal(err)
			}
			op := "source_export"
			if strings.HasPrefix(name, "retain") {
				op = "source_retain"
			}
			if err := validateSBVResult(op, p, raw); err != nil {
				t.Fatal(err)
			}
			p["output_path"] = "/changed/user-choice"
			if err := validateSBVResult(op, p, raw); err == nil {
				t.Fatal("native recovery detached from request")
			}
		})
	}
}

func TestSBVRetainedSourceInputChoices(t *testing.T) {
	read := func(name string) map[string]any {
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
	schemas := read("admin.schema.json")["requests"].(map[string]any)
	templates := read("admin.templates.json")["templates"].(map[string]any)
	clone := func(v any) map[string]any {
		b, _ := json.Marshal(v)
		x, err := sqavObject(b, maxRequestBytes)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	var hashes func(any)
	hashes = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, value := range x {
				if k == "source_sha256" || k == "expected_sha256" {
					x[k] = strings.Repeat("a", 64)
				} else {
					hashes(value)
				}
			}
		case []any:
			for _, value := range x {
				hashes(value)
			}
		}
	}
	selection := clone(templates["source_export"].(map[string]any)["retained_source"])
	hashes(selection)
	for _, op := range []string{"run", "evaluate", "book", "generate_census", "dataset_load"} {
		t.Run(op, func(t *testing.T) {
			file := clone(templates[op])
			hashes(file)
			if op == "evaluate" {
				file["census"].(map[string]any)["signals"].([]any)[0].(map[string]any)["available_ns"] = "1"
			}
			schema := schemas[op].(map[string]any)
			if !sbvInputShape(schema, file) {
				t.Fatal("legacy file choice rejected")
			}
			retained := clone(file)
			for _, k := range []string{"source_path", "source_sha256", "dataset"} {
				delete(retained, k)
			}
			retained["retained_source"] = clone(selection)
			if !sbvInputShape(schema, retained) {
				t.Fatal("retained-only choice rejected")
			}
			for _, k := range []string{"source_path", "source_sha256", "dataset"} {
				mixed := clone(retained)
				mixed[k] = file[k]
				if sbvInputShape(schema, mixed) {
					t.Fatal("mixed source choice admitted:", k)
				}
				partial := clone(file)
				delete(partial, k)
				if sbvInputShape(schema, partial) {
					t.Fatal("partial file source admitted:", k)
				}
			}
			missing := clone(retained)
			delete(missing, "retained_source")
			if sbvInputShape(schema, missing) {
				t.Fatal("missing source choice admitted")
			}
			changed := clone(retained)
			changed["retained_source"].(map[string]any)["delivery"].(map[string]any)["checkpoint"] = map[string]any{}
			if sbvInputShape(schema, changed) {
				t.Fatal("unsupported checkpoint admitted")
			}
		})
	}
}

func TestSBVOwnerRequestConstraints(t *testing.T) {
	read := func(name string) map[string]any {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "modules", "sbv-engine", "schemas", "v1", name))
		if err != nil {
			t.Fatal(err)
		}
		v, err := sbvDecodeOwnedResource(b)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	schemas := read("admin.schema.json")["requests"].(map[string]any)
	templates := read("admin.templates.json")["templates"].(map[string]any)
	clone := func(op string) map[string]any {
		b, _ := json.Marshal(templates[op])
		v, err := sqavObject(b, maxRequestBytes)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, op := range []string{"source_retain", "source_export"} {
		if !sbvInputShape(schemas[op].(map[string]any), clone(op)) {
			t.Fatalf("valid %s rejected", op)
		}
	}
	checks := []struct {
		name   string
		mutate func(map[string]any)
		want   bool
	}{
		{"declared_count", func(p map[string]any) { p["capture_description"].(map[string]any)["source_record_count"] = "100000" }, true},
		{"negative_count", func(p map[string]any) { p["capture_description"].(map[string]any)["source_record_count"] = "-1" }, false},
		{"missing_acquisition", func(p map[string]any) { p["capture_description"].(map[string]any)["times"] = []any{} }, false},
		{"duplicate_acquisition", func(p map[string]any) {
			d := p["capture_description"].(map[string]any)
			rows := d["times"].([]any)
			d["times"] = append(rows, rows[0])
		}, false},
		{"event_with_acquisition", func(p map[string]any) {
			d := p["capture_description"].(map[string]any)
			rows := d["times"].([]any)
			row := map[string]any{}
			for k, v := range rows[0].(map[string]any) {
				row[k] = v
			}
			row["role"] = "event"
			d["times"] = append(rows, row)
		}, true},
		{"duplicate_event", func(p map[string]any) {
			d := p["capture_description"].(map[string]any)
			rows := d["times"].([]any)
			row := map[string]any{}
			for k, v := range rows[0].(map[string]any) {
				row[k] = v
			}
			row["role"] = "event"
			d["times"] = append(rows, row, row)
		}, false},
		{"partial_without_evidence", func(p map[string]any) { p["capture_description"].(map[string]any)["coverage_evidence_ref"] = "" }, false},
		{"unknown_without_evidence", func(p map[string]any) {
			d := p["capture_description"].(map[string]any)
			d["coverage"] = "unknown"
			d["coverage_evidence_ref"] = ""
		}, true},
		{"path_too_long", func(p map[string]any) { p["output_path"] = "/" + strings.Repeat("a", 4096) }, false},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			p := clone("source_retain")
			check.mutate(p)
			if got := sbvInputShape(schemas["source_retain"].(map[string]any), p); got != check.want {
				t.Fatalf("admitted=%v, want %v", got, check.want)
			}
		})
	}
	p := clone("source_export")
	p["retained_source"].(map[string]any)["delivery"].(map[string]any)["processed_ack"] = "dataset_admitted"
	if sbvInputShape(schemas["source_export"].(map[string]any), p) {
		t.Fatal("export cannot acknowledge dataset admission")
	}
	// Exactly one branch must match, even if both would individually accept.
	if sbvSchemaShape(map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"minLength": int64(1)}}}, "x", 0) {
		t.Fatal("ambiguous oneOf admitted")
	}
}

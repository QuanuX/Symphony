package scvworkflow

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

// These are storage-boundary fixtures, not proof of an installed owner or of
// native semantics. Invocation and receipt verification remain separate checks.
func exactRecordFixture(t *testing.T, operation, version string) []byte {
	t.Helper()
	protocol, ok := knowledgeengine.SCVResultProtocol(operation)
	if !ok {
		t.Fatal("unknown fixture operation")
	}
	native, err := Seal(map[string]any{"protocol": protocol, "fixture": "synthetic record-shape test"})
	if err != nil {
		t.Fatal(err)
	}
	record, err := NewRecord(operation, knowledgeengine.Installation{Role: "scv", Version: version}, json.RawMessage(`{}`), native)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Canonical(record)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestWorkflowRecordExactIdentityPreservesValidHistoricalAndBundleRecords(t *testing.T) {
	for operation, version := range map[string]string{"knowledge_interpret": "0.3.0-dev", "bundle_inspect": "0.9.0-dev", "composition_bundle_evaluate": "0.9.0-dev"} {
		t.Run(operation, func(t *testing.T) {
			raw := exactRecordFixture(t, operation, version)
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, raw, "", "  "); err != nil {
				t.Fatal(err)
			}
			for _, input := range [][]byte{raw, pretty.Bytes()} {
				record, err := ReadRecord(input)
				if err != nil {
					t.Fatal(err)
				}
				again, err := Canonical(record)
				if err != nil || !bytes.Equal(raw, again) {
					t.Fatal("valid record changed its canonical bytes or seal")
				}
			}
		})
	}
}

func TestWorkflowRecordExactIdentityRejectsResealedNormalization(t *testing.T) {
	original := exactRecordFixture(t, "bundle_inspect", "0.9.0-dev")
	mutations := map[string]func(map[string]any){
		"role_alias": func(m map[string]any) {
			i := m["installation"].(map[string]any)
			i["role"] = i["Role"]
			delete(i, "Role")
		},
		"version_alias": func(m map[string]any) {
			i := m["installation"].(map[string]any)
			i["version"] = i["Version"]
			delete(i, "Version")
		},
		"missing_installation_field": func(m map[string]any) { delete(m["installation"].(map[string]any), "Prefix") },
		"null_installation_field":    func(m map[string]any) { m["installation"].(map[string]any)["Prefix"] = nil },
		"duplicate_case_alias":       func(m map[string]any) { m["installation"].(map[string]any)["role"] = "scv" },
		"operation_alias":            func(m map[string]any) { m["Operation"] = m["operation"]; delete(m, "operation") },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			value, err := Decode(original)
			if err != nil {
				t.Fatal(err)
			}
			mutate(value)
			altered, err := Seal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Digest(altered); err != nil {
				t.Fatal("mutation must retain a valid outer seal:", err)
			}
			if _, err := ReadRecord(altered); err == nil {
				t.Fatal("accepted record with a different typed identity")
			}
			store := fixture(t)
			if err := store.With("", true, func(session *Session, _ *Run) error { _, err := session.Put("records", altered); return err }); err == nil {
				t.Fatal("published a record that changes under typed interpretation")
			}
		})
	}
}

func TestWorkflowBundleRecordRejectsInvalidRawUnicode(t *testing.T) {
	for _, operation := range []string{"bundle_inspect", "composition_bundle_evaluate"} {
		for _, location := range []string{"input", "artifact", "installation"} {
			t.Run(operation+"/"+location, func(t *testing.T) {
				protocol, _ := knowledgeengine.SCVResultProtocol(operation)
				input := json.RawMessage(`{}`)
				artifact := map[string]any{"protocol": protocol, "fixture": "synthetic storage Unicode test"}
				installation := knowledgeengine.Installation{Role: "scv", Version: "0.9.0-dev"}
				switch location {
				case "input":
					input = json.RawMessage(`{"fixture":"�"}`)
				case "artifact":
					artifact["fixture"] = "�"
				case "installation":
					installation.Prefix = "�"
				}
				native, err := Seal(artifact)
				if err != nil {
					t.Fatal(err)
				}
				record, err := NewRecord(operation, installation, input, native)
				if err != nil {
					t.Fatal("valid literal replacement character rejected:", err)
				}
				original, err := Canonical(record)
				if err != nil {
					t.Fatal(err)
				}
				altered := bytes.Replace(original, []byte("�"), []byte(`\ud800`), 1)
				if bytes.Equal(original, altered) {
					t.Fatal("fixture lacks selected token")
				}
				// The common Go decoder otherwise replaces the malformed escape
				// with U+FFFD, so both existing seals still appear to verify.
				if _, err := Digest(altered); err != nil {
					t.Fatal("expected normalized original seal:", err)
				}
				if _, err := ReadRecord(altered); err == nil {
					t.Fatal("accepted lone surrogate in raw record")
				}
				store := fixture(t)
				if err := store.With("", true, func(session *Session, _ *Run) error { _, err := session.Put("records", altered); return err }); err == nil {
					t.Fatal("published malformed raw Unicode")
				}
			})
		}
	}
}

func TestWorkflowBundleRecordPreservesUnicodeAndSeparateEnvelopeBudget(t *testing.T) {
	for _, operation := range []string{"bundle_inspect", "composition_bundle_evaluate"} {
		t.Run(operation, func(t *testing.T) {
			protocol, _ := knowledgeengine.SCVResultProtocol(operation)
			native, err := Seal(map[string]any{"protocol": protocol, "values": make([]int, 16743), "pair": "🚀", "literal": `\ud800`})
			if err != nil {
				t.Fatal(err)
			}
			record, err := NewRecord(operation, knowledgeengine.Installation{Role: "scv", Version: "0.9.0-dev"}, budgetDocument(t, 16526), native)
			if err != nil {
				t.Fatal(err)
			}
			original, err := Canonical(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := knowledgeengine.ValidateJSONObject(original, MaxBytes); err == nil {
				t.Fatal("fixture must exceed the single-document value budget")
			}
			escaped := bytes.Replace(original, []byte("🚀"), []byte(`\ud83d\ude80`), 1)
			if bytes.Equal(original, escaped) {
				t.Fatal("missing paired Unicode fixture")
			}
			parsed, err := ReadRecord(escaped)
			if err != nil {
				t.Fatal("valid Unicode or two-child envelope rejected:", err)
			}
			normalized, err := Canonical(parsed)
			if err != nil || !bytes.Equal(original, normalized) {
				t.Fatal("valid escape changed record contents or seals")
			}
		})
	}
}

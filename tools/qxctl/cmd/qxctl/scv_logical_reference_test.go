package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
)

// Logical outputs come from the checked-in actual C++ fixture. Transport
// envelopes below are rebuilt around those outputs solely for adapter tests;
// synthetic Installation values do not assert a receipt or a process run.
func logicalReferenceFixture(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "knowledgeengine", "testdata", "scv-bundle.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err = d.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func logicalRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func logicalTestMetrics(t *testing.T, value any, bundle map[string]any) map[string]any {
	t.Helper()
	unique := map[string]int{}
	var walk func(any, int) (int, int)
	walk = func(v any, depth int) (int, int) {
		count, maximum := 1, depth
		child := func(x any) {
			n, d := walk(x, depth+1)
			count += n
			if d > maximum {
				maximum = d
			}
		}
		switch x := v.(type) {
		case map[string]any:
			digest, err := knowledgeengine.SCVDigest(x)
			if err != nil {
				t.Fatal(err)
			}
			unique[digest] = len(logicalRaw(t, x))
			for _, item := range x {
				count++
				child(item)
			}
		case []any:
			digest, err := knowledgeengine.SCVDigest(x)
			if err != nil {
				t.Fatal(err)
			}
			unique[digest] = len(logicalRaw(t, x))
			for _, item := range x {
				child(item)
			}
		}
		return count, maximum
	}
	values, depth := walk(value, 0)
	references := 0
	for _, raw := range bundle["objects"].([]any) {
		node := raw.(map[string]any)
		check := func(v any) {
			if _, ok := v.(map[string]any)["ref"]; ok {
				references++
			}
		}
		if node["kind"] == "object" {
			for _, c := range node["members"].(map[string]any) {
				check(c)
			}
		} else {
			for _, c := range node["items"].([]any) {
				check(c)
			}
		}
	}
	materialized := 0
	for _, size := range unique {
		materialized += size
	}
	return map[string]any{"object_count": len(unique), "reference_count": references, "traversal_steps": len(unique) + references, "expanded_bytes": len(logicalRaw(t, value)), "expanded_values": values, "expanded_depth": depth, "materialized_bytes": materialized}
}

func logicalTestBundle(t *testing.T, inst knowledgeengine.Installation, operation string, input, native any) scvworkflow.Record {
	t.Helper()
	payload, err := bundleLogicalInput(inst, operation, input)
	if err != nil {
		t.Fatal(err)
	}
	in := workflowValue(t, payload)
	encoded, err := knowledgeengine.SCVBundleEncode(logicalRaw(t, native))
	if err != nil {
		t.Fatal(err)
	}
	out := workflowValue(t, encoded)
	inputDigest, err := knowledgeengine.SCVDigest(in)
	if err != nil {
		t.Fatal(err)
	}
	fixture := logicalReferenceFixture(t)["operations"].([]any)[0].(map[string]any)["result"].(map[string]any)
	result, err := scvworkflow.Seal(map[string]any{
		"protocol": "symphony.scv.composition-bundle-evaluation.v1", "domain": inst.Role, "operation": operation,
		"owner": in["owner"], "input_digest": inputDigest, "input_root_digest": in["bundle"].(map[string]any)["root_digest"],
		"input_metrics": logicalTestMetrics(t, input, in["bundle"].(map[string]any)), "result_bundle": out,
		"native_result_digest": native.(map[string]any)["digest"], "result_metrics": logicalTestMetrics(t, native, out),
		"validation": "owner_evaluated", "limitations": fixture["limitations"],
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = knowledgeengine.ValidateSCVResult("composition_bundle_evaluate", payload, result); err != nil {
		t.Fatal("mechanical fixture wrapper failed independent validation:", err)
	}
	record, err := scvworkflow.NewRecord("composition_bundle_evaluate", inst, payload, result)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func logicalTestDirect(t *testing.T, inst knowledgeengine.Installation, operation string, input, native any) scvworkflow.Record {
	t.Helper()
	record, err := scvworkflow.NewRecord(operation, inst, logicalRaw(t, input), logicalRaw(t, native))
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func logicalRetain(t *testing.T, store scvworkflow.Store, record scvworkflow.Record) string {
	t.Helper()
	raw, err := scvworkflow.Canonical(record)
	if err != nil {
		t.Fatal(err)
	}
	var ref string
	if err = store.With("", true, func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		var e error
		ref, e = s.Put("records", raw)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestSCVLogicalReferenceSeparatesEquivalentProvenance(t *testing.T) {
	s := workflowStore(t)
	r := &workflowRunner{}
	op := logicalReferenceFixture(t)["operations"].([]any)[0].(map[string]any)
	direct := logicalTestDirect(t, knowledgeengine.Installation{Role: "scv", Version: "0.7.0-dev"}, "composition_explore", op["logical_input"], op["logical_result"])
	bundled := logicalTestBundle(t, knowledgeengine.Installation{Role: "scv", Version: "0.9.0-dev"}, "composition_explore", op["logical_input"], op["logical_result"])
	a, b := logicalRetain(t, s, direct), logicalRetain(t, s, bundled)
	err := s.With("", false, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		x, e := r.resolveLogicalReference(session, a, "composition_explore", false)
		if e != nil {
			return e
		}
		y, e := r.resolveLogicalReference(session, b, "composition_explore", false)
		if e != nil {
			return e
		}
		if x.Descriptor["record_ref"] == y.Descriptor["record_ref"] || x.Descriptor["artifact_digest"] == y.Descriptor["artifact_digest"] {
			t.Fatal("collapsed original provenance")
		}
		if x.Descriptor["logical_digest"] != y.Descriptor["logical_digest"] || !scvworkflow.Same(x.Artifact, y.Artifact) || !scvworkflow.Same(x.Input, y.Input) {
			t.Fatal("changed logical meaning")
		}
		if x.Descriptor["transport"] != nil || y.Descriptor["transport"] == nil {
			t.Fatal("lost transport distinction")
		}
		transport := y.Descriptor["transport"].(map[string]any)
		if transport["result_root_digest"] == y.Descriptor["logical_digest"] || transport["result_bundle_digest"] == y.Descriptor["artifact_digest"] {
			t.Fatal("confused full content hash, native seal or owner wrapper")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSCVLogicalReferenceRequiresOriginalOwnerAndLogicalKind(t *testing.T) {
	s := workflowStore(t)
	f := logicalReferenceFixture(t)["operations"].([]any)
	r := &workflowRunner{owner: func(knowledgeengine.Installation, string, any) (json.RawMessage, error) {
		return nil, errors.New("original executable absent")
	}}
	op := f[0].(map[string]any)
	direct := logicalTestDirect(t, knowledgeengine.Installation{Role: "scv", Version: "0.7.0-dev"}, "composition_explore", op["logical_input"], op["logical_result"])
	ref := logicalRetain(t, s, direct)
	inspection, err := scvworkflow.NewRecord("bundle_inspect", knowledgeengine.Installation{Role: "scv", Version: "0.9.0-dev"}, logicalRaw(t, op["input"]), logicalRaw(t, op["inspection"]))
	if err != nil {
		t.Fatal(err)
	}
	wrongRef := logicalRetain(t, s, inspection)
	reassess := f[1].(map[string]any)
	otherRef := logicalRetain(t, s, logicalTestBundle(t, knowledgeengine.Installation{Role: "scv", Version: "0.9.0-dev"}, "composition_reassess", reassess["logical_input"], reassess["logical_result"]))
	if err = s.With("", false, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		if _, e := r.resolveLogicalReference(session, ref, "composition_explore", false); e != nil {
			return e
		}
		if _, e := r.resolveLogicalReference(session, ref, "composition_explore", true); e == nil {
			t.Fatal("replaced unavailable owner")
		}
		for _, id := range []string{wrongRef, otherRef} {
			if _, e := r.resolveLogicalReference(session, id, "composition_explore", false); e == nil {
				t.Fatal("admitted wrong logical operation")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSCVLogicalReferenceRejectsResealedBundleAttribution(t *testing.T) {
	s := workflowStore(t)
	op := logicalReferenceFixture(t)["operations"].([]any)[0].(map[string]any)
	base := logicalTestBundle(t, knowledgeengine.Installation{Role: "scv", Version: "0.9.0-dev"}, "composition_explore", op["logical_input"], op["logical_result"])
	for _, change := range []string{"native_digest", "owner"} {
		t.Run(change, func(t *testing.T) {
			raw, _ := scvworkflow.Canonical(base)
			m := workflowValue(t, raw)
			if change == "native_digest" {
				result := m["artifact"].(map[string]any)
				result["native_result_digest"] = result["result_bundle"].(map[string]any)["root_digest"]
				sealed, e := scvworkflow.Seal(result)
				if e != nil {
					t.Fatal(e)
				}
				m["artifact"] = workflowValue(t, sealed)
				m["artifact_digest"] = m["artifact"].(map[string]any)["digest"]
			} else {
				m["installation"].(map[string]any)["Version"] = "0.10.0-dev"
			}
			altered, e := scvworkflow.Seal(m)
			if e != nil {
				t.Fatal(e)
			}
			record, e := scvworkflow.ReadRecord(altered)
			if e != nil {
				t.Fatal(e)
			}
			ref := logicalRetain(t, s, record)
			e = s.With("", false, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
				_, err := (&workflowRunner{}).resolveLogicalReference(session, ref, "composition_explore", false)
				return err
			})
			if e == nil {
				t.Fatal("accepted resealed attribution mismatch")
			}
		})
	}
}

func TestSCVLogicalReferenceRejectsLegacyRawUnicode(t *testing.T) {
	s := workflowStore(t)
	op := logicalReferenceFixture(t)["operations"].([]any)[0].(map[string]any)
	record := logicalTestDirect(t, knowledgeengine.Installation{Role: "scv", Version: "0.7.0-dev", Prefix: "�"}, "composition_explore", op["logical_input"], op["logical_result"])
	raw, _ := scvworkflow.Canonical(record)
	raw = bytes.Replace(raw, []byte("�"), []byte(`\ud800`), 1)
	var ref string
	if err := s.With("", true, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		var e error
		ref, e = session.Put("records", raw)
		return e
	}); err != nil {
		t.Fatal("legacy storage fixture failed:", err)
	}
	if err := s.With("", false, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		_, e := (&workflowRunner{}).resolveLogicalReference(session, ref, "composition_explore", false)
		return e
	}); err == nil {
		t.Fatal("normalized malformed Unicode in v2 reference path")
	}
}

func TestSCVLogicalBundleInputPreservesSeparateBounds(t *testing.T) {
	inst := knowledgeengine.Installation{Role: "scv", Version: "0.9.0-dev"}
	repeated := make([]any, 24)
	for i := range repeated {
		repeated[i] = map[string]any{"body": strings.Repeat("x", 65000)}
	}
	input := map[string]any{"caller_data": repeated}
	logical := logicalRaw(t, input)
	if len(logical) <= 1<<20 {
		t.Fatal("fixture must exceed legacy request bytes")
	}
	packed, err := bundleLogicalInput(inst, "composition_explore", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(packed) >= 1<<20 {
		t.Fatal("repeated evidence did not compact")
	}
	decoded, err := knowledgeengine.SCVBundleDecode(logicalRaw(t, workflowValue(t, packed)["bundle"]))
	if err != nil || !scvworkflow.Same(decoded, logical) {
		t.Fatal("changed complete caller input", err)
	}
	unique := make([]any, 24)
	for i := range unique {
		unique[i] = strings.Repeat(string(rune('A'+i)), 65000)
	}
	for name, value := range map[string]any{"wire_bytes": map[string]any{"data": unique}, "logical_values": map[string]any{"data": make([]any, 32769)}, "logical_bytes": map[string]any{"data": append(append(repeated, repeated...), repeated...)}} {
		t.Run(name, func(t *testing.T) {
			if _, e := bundleLogicalInput(inst, "composition_explore", value); e == nil {
				t.Fatal("widened explicit bundle boundary")
			}
		})
	}
}

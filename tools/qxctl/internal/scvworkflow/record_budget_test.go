package scvworkflow

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

// These synthetic payloads exercise retention bounds, not knowledge semantics.
// Actual owner/consumer replay remains the integration layer's responsibility.
func budgetDocument(t *testing.T, values int) json.RawMessage {
	t.Helper()
	raw, err := Canonical(map[string]any{"values": make([]int, values)})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func budgetArtifact(t *testing.T, values int) json.RawMessage {
	t.Helper()
	raw, err := Seal(map[string]any{"protocol": "symphony.scv.knowledge.v1", "values": make([]int, values)})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func budgetRecord(t *testing.T, input, result json.RawMessage) Record {
	t.Helper()
	record, err := NewRecord("knowledge_interpret", knowledgeengine.Installation{Role: "scv", Version: "0.3.0-dev"}, input, result)
	if err != nil {
		t.Fatal(err)
	}
	return record
}
func budgetRaw(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := Canonical(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func budgetObject(t *testing.T, value any) map[string]any {
	t.Helper()
	raw := budgetRaw(t, value)
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var object map[string]any
	if err := d.Decode(&object); err != nil {
		t.Fatal(err)
	}
	return object
}

// Construct a correct outer seal independently, so rejection cannot merely be
// due to a stale digest after deliberately breaking a child or envelope bound.
func uncheckedBudgetSeal(t *testing.T, value any) []byte {
	t.Helper()
	object := budgetObject(t, value)
	delete(object, "digest")
	digest, err := knowledgeengine.SCVDigest(object)
	if err != nil {
		t.Fatal(err)
	}
	object["digest"] = digest
	return budgetRaw(t, object)
}
func TestWorkflowRecordBudgetRetainsTwoBoundedNativeDocuments(t *testing.T) {
	input, result := budgetDocument(t, 16526), budgetArtifact(t, 16743)
	// Counts include object keys and containers: 16529 input, 16750 result.
	if err := knowledgeengine.ValidateJSONObject(input, MaxInputBytes); err != nil {
		t.Fatal(err)
	}
	if err := knowledgeengine.ValidateJSONObject(result, MaxBytes); err != nil {
		t.Fatal(err)
	}
	record := budgetRecord(t, input, result)
	raw := budgetRaw(t, record)
	if err := knowledgeengine.ValidateJSONObject(raw, MaxBytes); err == nil {
		t.Fatal("fixture did not exceed the default combined value count")
	}
	parsed, err := ReadRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !Same(raw, budgetRaw(t, parsed)) || !Same(parsed.Input, input) || !Same(parsed.Artifact, result) {
		t.Fatal("record changed native document or seal representation")
	}
	store := fixture(t)
	if err := store.With("", true, func(session *Session, _ *Run) error {
		ref, err := session.Put("records", raw)
		if err != nil {
			return err
		}
		again, err := session.Put("records", raw)
		if err != nil {
			return err
		}
		if ref != again {
			t.Fatal("immutable reuse changed record identity")
		}
		retained, err := session.Get("records", ref)
		if err != nil {
			return err
		}
		if !Same(retained, raw) {
			t.Fatal("retention changed combined record")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func TestWorkflowRecordBudgetAllowsExactChildBoundAndUnsealedEnvelope(t *testing.T) {
	input, result := budgetDocument(t, maxNativeJSONValues-3), budgetArtifact(t, maxNativeJSONValues-7)
	record := budgetRecord(t, input, result)
	original := budgetRaw(t, record)
	draft := budgetObject(t, record)
	delete(draft, "digest")
	sealed, err := Seal(draft)
	if err != nil || !Same(sealed, original) {
		t.Fatal("pre-seal record envelope failed or changed identity", err)
	}
	draft["digest"] = ""
	sealed, err = Seal(draft)
	if err != nil || !Same(sealed, original) {
		t.Fatal("empty pre-seal digest failed or changed identity", err)
	}
}
func TestWorkflowRecordBudgetRejectsOversizedChildrenAndNestedEscape(t *testing.T) {
	record := budgetRecord(t, budgetDocument(t, 17000), budgetArtifact(t, 17000))
	cases := []map[string]any{}
	inputTooLarge := budgetObject(t, record)
	inputTooLarge["input"] = budgetObject(t, budgetDocument(t, maxNativeJSONValues-2))
	cases = append(cases, inputTooLarge)
	resultTooLarge := budgetObject(t, record)
	resultTooLarge["artifact"] = map[string]any{"protocol": "symphony.scv.knowledge.v1", "values": make([]int, maxNativeJSONValues)}
	cases = append(cases, resultTooLarge)
	nested := budgetObject(t, record)
	nested["artifact"] = budgetObject(t, record)
	cases = append(cases, nested)
	for i, value := range cases {
		raw := uncheckedBudgetSeal(t, value)
		if err := knowledgeengine.ValidateJSONObjectWithValueLimit(raw, MaxBytes, maxRecordJSONValues); err != nil {
			t.Fatalf("case%d does not fit envelope budget: %v", i, err)
		}
		if _, err := Decode(raw); err == nil {
			t.Fatalf("case%d child inherited envelope allowance", i)
		}
		if _, err := ReadRecord(raw); err == nil {
			t.Fatalf("case%d oversized native child accepted", i)
		}
		if _, err := Seal(value); err == nil {
			t.Fatalf("case%d oversized child sealed", i)
		}
	}
}
func TestWorkflowRecordBudgetLeavesOtherObjectsAndMetadataBounded(t *testing.T) {
	if _, err := Decode(budgetDocument(t, maxNativeJSONValues-2)); err == nil {
		t.Fatal("generic32769-value document admitted")
	}
	large := budgetRecord(t, budgetDocument(t, 17000), budgetArtifact(t, 17000))
	nonrecord := budgetObject(t, large)
	nonrecord["protocol"] = "symphony.qxctl.scv-workflow-payload.v1"
	if _, err := Decode(uncheckedBudgetSeal(t, nonrecord)); err == nil {
		t.Fatal("nonrecord namespace inherited larger budget")
	}
	unknown := budgetObject(t, large)
	unknown["extra"] = true
	if _, err := Decode(uncheckedBudgetSeal(t, unknown)); err == nil {
		t.Fatal("unknown envelope field admitted")
	}
	missing := budgetObject(t, large)
	delete(missing, "kind")
	if _, err := Decode(uncheckedBudgetSeal(t, missing)); err == nil {
		t.Fatal("missing required envelope field admitted")
	}
	metadata := budgetObject(t, large)
	metadata["installation"] = map[string]any{"padding": make([]int, 257)}
	if _, err := Decode(uncheckedBudgetSeal(t, metadata)); err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatal("metadata failed to enforce its own256-value budget", err)
	}
}
func TestWorkflowRecordBudgetPreservesSharedStructuralChecks(t *testing.T) {
	large := budgetRecord(t, budgetDocument(t, 17000), budgetArtifact(t, 17000))
	raw := budgetRaw(t, large)
	duplicated := bytes.Replace(raw, []byte(`"kind":"knowledge"`), []byte(`"kind":"knowledge","kind":"knowledge"`), 1)
	if bytes.Equal(duplicated, raw) {
		t.Fatal("duplicate fixture replacement failed")
	}
	if _, err := Decode(duplicated); err == nil {
		t.Fatal("duplicate envelope key admitted")
	}
	deep := budgetObject(t, large)
	var nested any = true
	for i := 0; i < 65; i++ {
		nested = map[string]any{"nested": nested}
	}
	deep["input"].(map[string]any)["deep"] = nested
	if _, err := Decode(uncheckedBudgetSeal(t, deep)); err == nil {
		t.Fatal("depth limit bypassed")
	}
	floating := budgetObject(t, large)
	floating["input"].(map[string]any)["bad"] = 1.5
	if _, err := Decode(uncheckedBudgetSeal(t, floating)); err == nil {
		t.Fatal("native numeric restrictions bypassed")
	}
	if _, err := Decode(append(raw, []byte(` {}`)...)); err == nil {
		t.Fatal("trailing document accepted")
	}
}
func TestWorkflowRecordBudgetPreservesByteLimits(t *testing.T) {
	chunks := func(count int) []string {
		v := make([]string, count)
		for i := range v {
			v[i] = strings.Repeat("x", 65536)
		}
		return v
	}
	result, err := Seal(map[string]any{"protocol": "symphony.scv.knowledge.v1", "chunks": chunks(48)})
	if err != nil {
		t.Fatal(err)
	}
	// Preserve the established four-MiB native result allowance, not two MiB.
	record := budgetRecord(t, json.RawMessage(`{}`), result)
	if len(record.Artifact) <= 2<<20 {
		t.Fatal("large native result fixture too small")
	}
	oversizedInput := budgetRaw(t, map[string]any{"chunks": chunks(16)})
	if len(oversizedInput) <= MaxInputBytes {
		t.Fatal("input fixture should exceed one MiB with JSON framing")
	}
	if _, err := NewRecord("knowledge_interpret", record.Installation, oversizedInput, result); err == nil {
		t.Fatal("input byte budget widened")
	}
	raw := uncheckedBudgetSeal(t, map[string]any{"protocol": "symphony.scv.knowledge.v1", "chunks": chunks(64)})
	if len(raw) <= MaxBytes {
		t.Fatal("result fixture should exceed four MiB with framing")
	}
	if _, err := ReadRecord(raw); err == nil {
		t.Fatal("record byte budget widened")
	}
	if _, err := NewRecord("knowledge_interpret", record.Installation, json.RawMessage(`{}`), raw); err == nil {
		t.Fatal("artifact byte budget widened")
	}
	// Individually valid byte-bounded children still cannot exceed the whole
	// record's unchanged four-MiB maximum once combined with metadata.
	input := budgetRaw(t, map[string]any{"chunks": chunks(15)})
	result, err = Seal(map[string]any{"protocol": "symphony.scv.knowledge.v1", "chunks": chunks(49)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewRecord("knowledge_interpret", record.Installation, input, result); err == nil {
		t.Fatal("combined four-MiB record limit widened")
	}
}

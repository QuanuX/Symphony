package knowledgeengine

import (
	"os"
	"testing"
)

func publicationFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	b, e := os.ReadFile("testdata/publication-initial.json")
	if e != nil {
		t.Fatal(e)
	}
	plan, e := shvObject(b)
	if e != nil {
		t.Fatal(e)
	}
	input := map[string]any{"operation_id": plan["operation_id"], "current": nil, "desired": shvMap(plan["head"])["definition"], "reason": plan["reason"]}
	return input, plan
}
func pubTestRaw(t *testing.T, v any) []byte {
	t.Helper()
	b, e := SCVCanonical(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestSHVPublicationIndependentCorrespondence(t *testing.T) {
	in, p := publicationFixture(t)
	if e := ValidateSHVPublicationResult("publication_plan", pubTestRaw(t, in), pubTestRaw(t, p)); e != nil {
		t.Fatal(e)
	}
	for _, field := range []string{"reason", "operation_id", "expected_state_digest", "change_kind"} {
		t.Run(field, func(t *testing.T) {
			input, plan := publicationFixture(t)
			plan[field] = "changed"
			delete(plan, "digest")
			d, e := SCVDigest(plan)
			if e != nil {
				t.Fatal(e)
			}
			plan["digest"] = d
			if ValidateSHVPublicationResult("publication_plan", pubTestRaw(t, input), pubTestRaw(t, plan)) == nil {
				t.Fatal("resealed result drift accepted")
			}
		})
	}
	for _, value := range []any{nil, []any{map[string]any{}}, "invalid"} {
		input, _ := publicationFixture(t)
		shvMap(input["desired"])["members"] = value
		if _, e := pubPlan(input); e == nil {
			t.Fatal("invalid members accepted")
		}
	}
	head := shvMap(p["head"])
	status := map[string]any{"protocol": "symphony.shv.publication-status.v1", "head": head, "history_digests": []any{head["digest"]}}
	d, _ := SCVDigest(status)
	status["digest"] = d
	if e := ValidateSHVPublicationResult("publication_status", pubTestRaw(t, map[string]any{"history": []any{head}}), pubTestRaw(t, status)); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVPublicationResult("publication_status", pubTestRaw(t, map[string]any{"history": []any{head, head}}), pubTestRaw(t, status)) == nil {
		t.Fatal("duplicate history accepted")
	}
	bad := shvMap(head["definition"])
	bad["tops_id"] = "00000000-0000-0000-0000-000000000000"
	if pubDefinition(bad) == nil {
		t.Fatal("invalid TOPS accepted")
	}
}

package main

import (
	"encoding/json"
	"testing"
)

func TestSHVComparisonAdmission(t *testing.T) {
	for _, raw := range []string{`null`, `{"previous":{},"current":{},"extra":true}`, `{"previous":{},"current":{}}`, `{"previous":null,"current":null}`} {
		if _, e := compareSHVRefresh(json.RawMessage(raw)); e == nil {
			t.Fatal("invalid comparison accepted", raw)
		}
	}
	template := comparisonTemplate()
	raw, _ := json.Marshal(template)
	if _, e := compareSHVRefresh(raw); e == nil {
		t.Fatal("unanswered template admitted")
	}
	out, status := invokeCLI(t, "shv", "refresh", "compare", "--json")
	var d map[string]any
	if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
		t.Fatal(status, out)
	}
}
func TestSHVComparisonRows(t *testing.T) {
	previous := json.RawMessage(`{"subjects":[{"id":"b","manufacturer":"Old","assertions":[{"predicate":"cores","qualifier":"documented","source_id":"page","value":24}]},{"id":"a","assertions":[]}]}`)
	current := json.RawMessage(`{"subjects":[{"id":"b","manufacturer":"New","assertions":[{"predicate":"cores","qualifier":"documented","source_id":"page","value":24},{"predicate":"mode","value":"dual"}]},{"id":"c","assertions":[]}]}`)
	subjects, assertions := comparisonCatalogueRows(previous, current)
	if len(subjects) != 3 || len(assertions) != 1 {
		t.Fatal(subjects, assertions)
	}
	for i, id := range []string{"a", "b", "c"} {
		if subjects[i].(map[string]any)["subject_id"] != id {
			t.Fatal("unstable ordering")
		}
	}
	if assertions[0].(map[string]any)["kind"] != "added" {
		t.Fatal(assertions)
	}
	reverseS, reverseA := comparisonCatalogueRows(current, previous)
	if len(reverseS) != 3 || reverseA[0].(map[string]any)["kind"] != "removed" {
		t.Fatal("direction lost")
	}
	sameS, sameA := comparisonCatalogueRows(previous, previous)
	if len(sameS) != 0 || len(sameA) != 0 {
		t.Fatal("equal input differs")
	}
}

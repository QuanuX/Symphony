package knowledgeengine

import (
	"os"
	"strings"
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

// Admission belongs to the selected consuming owner, not to a latest alias.
func TestSHVPublicationCompositionAdmission(t *testing.T) {
	for _, pv := range []string{"0.2.0-dev", "0.3.0-dev", "0.4.0-dev", "0.5.0-dev", "latest"} {
		input, _ := publicationFixture(t)
		inst := shvMap(shvMap(input["desired"])["partition_installation"])
		old := shvText(inst["Version"])
		for _, k := range []string{"ReceiptPath", "ExecutablePath"} {
			inst[k] = strings.ReplaceAll(shvText(inst[k]), old, pv)
		}
		inst["Version"] = pv
		for _, reader := range []string{"0.1.0-dev", "0.2.0-dev", "0.3.0-dev", "0.4.0-dev"} {
			_, err := pubPlanVersion(input, reader)
			want := pv == "0.2.0-dev" || (reader == "0.4.0-dev" && (pv == "0.3.0-dev" || pv == "0.4.0-dev"))
			if (err == nil) != want {
				t.Fatalf("reader %s partition %s: %v", reader, pv, err)
			}
		}
	}
	for _, writer := range []string{"0.1.0-dev", "0.2.0-dev", "0.3.0-dev", "0.4.0-dev", "0.5.0-dev"} {
		module := "shv-graph-duckdb-connector"
		engine := "symphony-shv-graph-duckdb-connector"
		h := "sha256:" + strings.Repeat("1", 64)
		inst := map[string]any{"Role": module, "ModuleID": module, "EngineID": engine, "Version": writer, "Prefix": "/store", "ReceiptPath": "/store/share/symphony/receipts/" + module + "/" + writer + "/install-receipt.json", "ReceiptDigest": h, "ReceiptProtocol": receiptProtocolV2, "ExecutablePath": "/store/libexec/symphony/" + module + "/" + writer + "/" + engine, "ExecutableDigest": h}
		for _, reader := range []string{"0.1.0-dev", "0.2.0-dev", "0.3.0-dev", "0.4.0-dev"} {
			want := writer == "0.1.0-dev" || (reader != "0.1.0-dev" && (writer == "0.2.0-dev" || writer == "0.3.0-dev")) || (reader == "0.4.0-dev" && writer == "0.4.0-dev")
			if (pubStoreInstallation(inst, reader) == nil) != want {
				t.Fatalf("reader %s writer %s", reader, writer)
			}
		}
	}
}

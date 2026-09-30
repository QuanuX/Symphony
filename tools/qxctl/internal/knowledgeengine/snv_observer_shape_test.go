package knowledgeengine

import (
	"path/filepath"
	"testing"
)

func TestSNVObserverActualPortableResultShape(t *testing.T) {
	module := filepath.Join(snvRepository(t), "modules/snv-local-observer")
	input := snvReadObject(t, filepath.Join(module, "tests/fixtures/observe.json"))
	result := snvReadObject(t, filepath.Join(module, "tests/fixtures/portable-unavailable-result.v1.json"))
	schema := snvReadObject(t, filepath.Join(module, "schemas/v1/admin.schema.json"))
	contract := map[string]any{"input": schema["requests"].(map[string]any)["observe"], "result": schema["results"].(map[string]any)["observe"]}
	if err := ValidateSNVResult("local-observer", "observe", snvEncode(t, input), snvEncode(t, result), snvEncode(t, contract)); err != nil {
		t.Fatal("Actual portable unavailable output rejected", err)
	}
	for _, mutate := range []func(map[string]any){
		func(r map[string]any) { r["acquisition_route"] = "sdk_supplied_reader" },
		func(r map[string]any) { r["raw_boot_id"] = "forbidden" },
		func(r map[string]any) { r["node_ref"] = "invented-node" },
		func(r map[string]any) { r["fields"].([]any)[0].(map[string]any)["field"] = "serial" },
		func(r map[string]any) { r["capture"].(map[string]any)["hostname"] = "forbidden" },
		func(r map[string]any) { r["platform"].(map[string]any)["machine_id"] = "forbidden" },
	} {
		changed := snvClone(t, result)
		mutate(changed)
		if err := ValidateSNVResult("local-observer", "observe", snvEncode(t, input), snvEncode(t, changed), snvEncode(t, contract)); err == nil {
			t.Fatal("Unsupported field/invented binding accepted")
		}
	}
}

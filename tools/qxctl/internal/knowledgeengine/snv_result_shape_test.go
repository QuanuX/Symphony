package knowledgeengine

import (
	"path/filepath"
	"testing"
)

func TestSNVActualNativeResultClosedShapes(t *testing.T) {
	root := snvRepository(t)
	schema := snvReadObject(t, filepath.Join(root, "modules/snv-engine/schemas/v1/admin.schema.json"))
	fixtures := snvReadObject(t, filepath.Join(root, "modules/snv-engine/tests/fixtures/native-result-contract-cases.v1.json"))
	for _, item := range fixtures["cases"].([]any) {
		c := item.(map[string]any)
		operation := c["operation"].(string)
		input := c["input"].(map[string]any)
		result := c["result"].(map[string]any)
		label := operation
		if mode, ok := input["mode"].(string); ok {
			label += "/" + mode
		}
		t.Run(label, func(t *testing.T) {
			contract := map[string]any{"input": schema["requests"].(map[string]any)[operation], "result": schema["results"].(map[string]any)[operation], "$defs": schema["$defs"]}
			if err := ValidateSNVResult("snv", operation, snvEncode(t, input), snvEncode(t, result), snvEncode(t, contract)); err != nil {
				t.Fatal("Actual native output rejected", err)
			}
			mutations := []func(map[string]any){func(r map[string]any) { r["unexpected"] = true }}
			if operation == "snv_inspect" && input["mode"] != "history" {
				mutations = append(mutations, func(r map[string]any) { r["history"] = []any{} })
			}
			if input["mode"] == "history" {
				mutations = append(mutations, func(r map[string]any) {
					r["history"].([]any)[0].(map[string]any)["record"].(map[string]any)["unexpected"] = true
				})
			}
			if _, present := result["views"]; present {
				mutations = append(mutations, func(r map[string]any) { r["views"].(map[string]any)["sniv"].(map[string]any)["unexpected"] = true })
			}
			if operation == "snv_evidence_plan" {
				mutations = append(mutations, func(r map[string]any) { r["replay"].(map[string]any)["unexpected"] = true })
			}
			for _, mutate := range mutations {
				changed := snvClone(t, result)
				mutate(changed)
				if _, sealed := changed["digest"]; sealed {
					delete(changed, "digest")
					changed["digest"] = digestBytes(snvEncode(t, changed))
				}
				if err := ValidateSNVResult("snv", operation, snvEncode(t, input), snvEncode(t, changed), snvEncode(t, contract)); err == nil {
					t.Fatal("Unknown native output field accepted even after recomputing its seal")
				}
			}
		})
	}
}

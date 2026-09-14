package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed shv_materialization.schema.json
var jobSchema json.RawMessage

func jobDiscovery(op string) error {
	m := map[string]any{"protocol": "symphony.qxctl.shv-materialization-" + op + ".v1", "origin": "qxctl_embedded"}
	if op == "schema" {
		m["schema"] = jobSchema
	} else {
		endpoint := map[string]any{}
		for _, k := range comparisonEndpointKeys {
			endpoint[k] = nil
		}
		m["template"] = map[string]any{"tasks": []any{map[string]any{"task_id": nil, "endpoint": endpoint}}, "required_references": []any{}}
		m["status"] = "unanswered_template_not_validated_input"
	}
	raw, e := sealSHVActivation(m)
	if e != nil {
		return e
	}
	return printIndentedJSON(raw)
}

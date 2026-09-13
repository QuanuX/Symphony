package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"sort"
)

func SHVSourceSchemaEntries() map[string]string {
	entries := map[string]string{}
	inputs := map[string]string{"inspect": "InspectInput", "source_plan": "SourcePlanInput", "source_reduce": "SourceReduceInput", "source_status": "SourceStatusInput", "capture_import": "CaptureImportInput", "capture_compare": "CaptureCompareInput", "graph_project": "GraphProjectInput", "graph_validate": "GraphValidateInput"}
	outputs := map[string]string{"inspect": "Descriptor", "source_plan": "SourcePlan", "source_reduce": "SourceTransition", "source_status": "SourceStatus", "capture_import": "Capture", "capture_compare": "CaptureComparison", "graph_project": "Graph", "graph_validate": "GraphValidation"}
	for op, def := range inputs {
		entries[SHVSourceInputProtocol(op)] = def
		entries[shvSourceOutputs[op]] = outputs[op]
	}
	entries["symphony.shv.source-revision.v1"] = "Source"
	entries["symphony.shv.source-bundle.v1"] = "Bundle"
	return entries
}

// SHVDiscovery admits only this compiled release's protocol/operation names.
// Templates retain unanswered nulls; they are not prevalidated factual inputs.
func SHVSourceDiscovery(prefix, version string, template bool, selection string) (json.RawMessage, error) {
	inst, raw, e := SHVSourceResource(prefix, version, template)
	if e != nil {
		return nil, e
	}
	doc, e := shvObject(raw)
	if e != nil {
		return nil, e
	}
	if template {
		if _, ok := SHVSourceResultProtocol(selection); !ok {
			return nil, fmt.Errorf("select an exact supported --operation for template")
		}
		value, ok := doc[selection]
		if !ok || shvMap(value) == nil {
			return nil, fmt.Errorf("owned templates lack admitted operation")
		}
		m := shvSealNew(map[string]any{"protocol": "symphony.qxctl.shv-template.v1", "installation": inst, "origin": "receipt_owned", "operation": selection, "input_protocol": SHVSourceInputProtocol(selection), "resource_digest": digestBytes(raw), "template": value, "status": "unanswered_template_not_validated_input"})
		return SCVCanonical(m)
	}
	entries := SHVSourceSchemaEntries()
	defs := shvMap(doc["$defs"])
	keys := []string{}
	for p, def := range entries {
		if shvMap(defs[def]) == nil {
			return nil, fmt.Errorf("owned schema lacks compiled definition %s", def)
		}
		keys = append(keys, p)
	}
	entries["symphony.qxctl.shv-schema.v1"] = "SHVSchema"
	entries["symphony.qxctl.shv-template.v1"] = "SHVTemplate"
	keys = append(keys, "symphony.qxctl.shv-schema.v1", "symphony.qxctl.shv-template.v1")
	cliDoc, err := shvObject(shvDiscoverySchema)
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	rows := []any{}
	var selected any
	for _, p := range keys {
		if selection != "" && selection != p {
			continue
		}
		origin, document, rawBytes := "receipt_owned", doc, []byte(raw)
		if p == "symphony.qxctl.shv-schema.v1" || p == "symphony.qxctl.shv-template.v1" {
			origin, document, rawBytes = "qxctl_embedded", cliDoc, shvDiscoverySchema
		}
		rows = append(rows, map[string]any{"protocol": p, "fragment": "#/$defs/" + entries[p], "origin": origin, "schema_digest": digestBytes(rawBytes)})
		if selection == p {
			selected = document
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("protocol not admitted by exact SHV schema release")
	}
	return SCVCanonical(shvSealNew(map[string]any{"protocol": "symphony.qxctl.shv-schema.v1", "installation": inst, "entries": rows, "schema": selected}))
}

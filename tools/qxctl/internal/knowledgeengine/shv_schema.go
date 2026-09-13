package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"sort"
)

func SHVSchemaEntries(adapter bool) map[string]string {
	if adapter {
		return map[string]string{SHVInputProtocol("inspect", true): "InspectInput", SHVInputProtocol("roundtrip", true): "RoundtripInput", SHVInputProtocol("query", true): "QueryInput", "symphony.graph.adapter-result.v1": "AdapterResult", "symphony.graph.exchange.v1": "Graph", "symphony.knowledge.engine-descriptor.v2": "Descriptor"}
	}
	return map[string]string{SHVInputProtocol("inspect", false): "InspectInput", SHVInputProtocol("coverage_default", false): "CoverageDefaultInput", SHVInputProtocol("coverage_plan", false): "CoveragePlanInput", SHVInputProtocol("catalogue_build", false): "CatalogueBuildInput", SHVInputProtocol("catalogue_query", false): "CatalogueQueryInput", SHVInputProtocol("evaluate", false): "EvaluateInput", SHVInputProtocol("graph_project", false): "GraphProjectInput", SHVInputProtocol("graph_validate", false): "GraphValidateInput", "symphony.shv.coverage-profile.v1": "CoverageProfile", "symphony.shv.coverage-result.v1": "CoverageResult", "symphony.shv.catalogue.v1": "Catalogue", "symphony.shv.query-result.v1": "QueryResult", "symphony.shv.evaluation.v1": "Evaluation", "symphony.graph.exchange.v1": "Graph", "symphony.shv.graph-validation.v1": "GraphValidation", "symphony.knowledge.engine-descriptor.v2": "Descriptor"}
}

// SHVDiscovery admits only this compiled release's protocol/operation names.
// Templates retain unanswered nulls; they are not prevalidated factual inputs.
func SHVDiscovery(prefix, version string, adapter, template bool, selection string) (json.RawMessage, error) {
	inst, raw, e := SHVResource(prefix, version, adapter, template)
	if e != nil {
		return nil, e
	}
	doc, e := shvObject(raw)
	if e != nil {
		return nil, e
	}
	if template {
		if _, ok := SHVResultProtocol(selection, adapter); !ok {
			return nil, fmt.Errorf("select an exact supported --operation for template")
		}
		value, ok := doc[selection]
		if !ok || shvMap(value) == nil {
			return nil, fmt.Errorf("owned templates lack admitted operation")
		}
		m := shvSealNew(map[string]any{"protocol": "symphony.qxctl.shv-template.v1", "installation": inst, "origin": "receipt_owned", "operation": selection, "input_protocol": SHVInputProtocol(selection, adapter), "resource_digest": digestBytes(raw), "template": value, "status": "unanswered_template_not_validated_input"})
		return SCVCanonical(m)
	}
	entries := SHVSchemaEntries(adapter)
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

package knowledgeengine

import (
	"encoding/json"
	"sort"
)

// pdfGraph reconstructs the precise owner projection; generic transport never
// decides identifier semantics or adds documentary lineages.
func pdfGraph(x map[string]any) map[string]any {
	source := shvMap(x["source"])
	sid := "source:" + shvText(source["digest"])
	did := "derivation:" + shvText(x["digest"])
	nodes := []map[string]any{{"id": sid, "labels": []string{"DocumentSource"}, "properties": source}, {"id": did, "labels": []string{"DocumentDerivation"}, "properties": map[string]any{"profile": x["profile"], "decoder": x["decoder"], "text_digest": x["text_digest"], "documentary_lineages": 1, "namespace_equivalence": "not_asserted"}}}
	edges := []map[string]any{{"id": "derives:" + did, "from": did, "to": sid, "label": "derived_from", "properties": map[string]any{}}}
	rows, _ := x["rows"].([]any)
	for i, v := range rows {
		row := shvMap(v)
		id := "row:" + shvText(x["digest"]) + ":" + shvText(row["opn"])
		nodes = append(nodes, map[string]any{"id": id, "labels": []string{"DocumentSubject"}, "properties": map[string]any{"model": row["model"], "source_id": source["id"], "identity_verified": false}})
		citation := map[string]any{"source_id": source["id"], "source_digest": source["digest"], "derivation_digest": x["digest"], "page_index": 12, "table": 8, "row_index": i}
		edges = append(edges, map[string]any{"id": "asserts:" + id, "from": sid, "to": id, "label": "documents_identifier", "properties": map[string]any{"predicate": "oem_identifier", "qualifier": "issuer=AMD;namespace=opn;profile=1", "value": row["opn"], "citation": citation}}, map[string]any{"id": "extracts:" + id, "from": did, "to": id, "label": "extracts_subject", "properties": map[string]any{}})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i]["id"].(string) < nodes[j]["id"].(string) })
	sort.Slice(edges, func(i, j int) bool { return edges[i]["id"].(string) < edges[j]["id"].(string) })
	return pdfSealed(map[string]any{"protocol": "symphony.graph.exchange.v1", "owner": map[string]any{"engine_id": "symphony-shv-pdf", "engine_version": SHVPDFGraphVersion, "artifact_protocol": x["protocol"], "artifact_digest": x["digest"]}, "owner_artifact": x, "nodes": nodes, "edges": edges})
}
func validatePDFGraphResult(op string, p, r map[string]any, version string) error {
	if version != SHVPDFGraphVersion {
		return shvFail()
	}
	request := p
	graph := r
	if op == "graph_validate" {
		if !shvFields(p, "request", "graph") {
			return shvFail()
		}
		request = shvMap(p["request"])
		graph = shvMap(p["graph"])
	}
	x := shvMap(graph["owner_artifact"])
	input, _ := json.Marshal(request)
	artifact, _ := json.Marshal(x)
	if ValidateSHVPDFResult("extract", input, artifact) != nil {
		return shvFail()
	}
	if !scvEqual(pdfGraph(x), graph) {
		return shvFail()
	}
	if op == "graph_validate" {
		expected := pdfSealed(map[string]any{"protocol": "symphony.shv.pdf-graph-validation.v1", "graph_digest": graph["digest"], "derivation_digest": x["digest"], "replayed": true, "documentary_lineages": 1})
		if !scvEqual(expected, r) {
			return shvFail()
		}
	}
	return nil
}

func pdfSealed(v map[string]any) map[string]any {
	raw, _ := json.Marshal(v)
	v["digest"] = digestBytes(raw)
	return v
}

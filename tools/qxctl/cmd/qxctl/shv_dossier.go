package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
	"os"
	"sort"
)

//go:embed shv_dossier.schema.json
var dossierSchema json.RawMessage

func newSHVDossierCommand() *cobra.Command {
	root := structural("dossier", fmt.Errorf("dossier operation required"))
	for _, op := range []string{"run", "graph", "schema", "template"} {
		var prefix, version, input, ap, av string
		c := &cobra.Command{Use: op, Args: usageOnlyArgs, Short: "Retain caller associations and exact source-scoped citations", RunE: func(*cobra.Command, []string) error {
			if op == "schema" || op == "template" {
				out := map[string]any{"protocol": "symphony.qxctl.shv-dossier-" + op + ".v1", "origin": "qxctl_embedded"}
				if op == "schema" {
					out["schema"] = dossierSchema
				} else {
					out["template"] = map[string]any{"component": map[string]any{"id": nil, "label": nil}, "inventory": nil, "associations": []any{}}
					out["status"] = "unanswered_template_not_validated_input"
				}
				raw, e := sealSHVActivation(out)
				if e != nil {
					return e
				}
				return printIndentedJSON(raw)
			}
			raw, e := knowledgeengine.ReadPayload(input)
			if e != nil {
				return e
			}
			result, e := evaluateSHVDossier(raw, prefix, version)
			if e != nil {
				return e
			}
			if op == "run" {
				return printIndentedJSON(result)
			}
			graph, e := dossierGraph(result)
			if e != nil {
				return e
			}
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			inst, e := knowledgeengine.InspectSHVGraphAdapter(ap, av)
			if e != nil {
				return e
			}
			transport, e := knowledgeengine.InvokeSHVGraphAdapter(context.Background(), ap, av, cwd, "roundtrip", jobMarshal(map[string]any{"graph": graph}))
			if e != nil {
				return e
			}
			after, e := knowledgeengine.InspectSHVGraphAdapter(ap, av)
			if e != nil {
				return e
			}
			if after != inst {
				return fmt.Errorf("dossier adapter installation changed")
			}
			sealed, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-dossier-graph.v1", "dossier": result, "transport": transport.Result, "installation": inst, "catalogue_published": false})
			if e != nil {
				return e
			}
			return printIndentedJSON(sealed)
		}}
		c.Flags().Bool("json", false, "emit structured evidence")
		if op == "run" || op == "graph" {
			c.Flags().StringVar(&prefix, "prefix", "", "exact native coverage installation")
			c.Flags().StringVar(&version, "version", "", "exact native coverage version")
			c.Flags().StringVar(&input, "input", "", "explicit dossier request")
			for _, f := range []string{"prefix", "version", "input"} {
				_ = c.MarkFlagRequired(f)
			}
		}
		if op == "graph" {
			c.Flags().StringVar(&ap, "adapter-prefix", "", "exact generic adapter installation")
			c.Flags().StringVar(&av, "adapter-version", "", "exact generic adapter version")
			_ = c.MarkFlagRequired("adapter-prefix")
			_ = c.MarkFlagRequired("adapter-version")
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "discover"
		if op == "run" || op == "graph" {
			interaction = "invoke"
		}
		s := commandSpec("shv.dossier."+op, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		protocol := "symphony.qxctl.shv-dossier-" + op + ".v1"
		if op == "run" {
			protocol = "symphony.qxctl.shv-dossier.v1"
		}
		if interaction == "invoke" {
			s.InputProtocols = []string{"symphony.qxctl.shv-dossier-input.v1"}
			s.BackendOperationIDs = []string{"engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project"}
			for _, f := range []string{"shv-engine", "shv-source-engine"} {
				s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + f, Interaction: "invoke"})
			}
			if op == "graph" {
				s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:shv-graph-adapter.roundtrip")
				s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-graph-adapter", Interaction: "invoke"})
			}
		}
		s.OutputProtocols = []string{protocol}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	return root
}

func dossierAdmission(raw json.RawMessage) (map[string]json.RawMessage, []map[string]json.RawMessage, error) {
	in, e := jobDecode(raw, "component", "inventory", "associations")
	if e != nil {
		return nil, nil, e
	}
	component, e := jobDecode(in["component"], "id", "label")
	if e != nil {
		return nil, nil, e
	}
	if !jobToken.MatchString(refreshString(component["id"])) || len(refreshString(component["label"])) == 0 || len(refreshString(component["label"])) > 256 {
		return nil, nil, fmt.Errorf("invalid component declaration")
	}
	_, roster, _, e := inventoryRequest(in["inventory"])
	if e != nil {
		return nil, nil, e
	}
	members := map[string]bool{}
	for _, r := range roster {
		members[refreshString(r["id"])] = true
	}
	var associations []map[string]json.RawMessage
	if json.Unmarshal(in["associations"], &associations) != nil || associations == nil || len(associations) > 32 {
		return nil, nil, fmt.Errorf("invalid association bounds")
	}
	ids := map[string]bool{}
	for _, a := range associations {
		if _, e = jobDecode(jobMarshal(a), "id", "roster_id", "relation", "citation_digests"); e != nil {
			return nil, nil, e
		}
		id := refreshString(a["id"])
		if !jobToken.MatchString(id) || ids[id] || !jobToken.MatchString(refreshString(a["relation"])) || !members[refreshString(a["roster_id"])] {
			return nil, nil, fmt.Errorf("invalid or unlisted association")
		}
		ids[id] = true
		var citations []string
		if json.Unmarshal(a["citation_digests"], &citations) != nil || citations == nil || len(citations) > 8 {
			return nil, nil, fmt.Errorf("invalid citation bounds")
		}
		seen := map[string]bool{}
		for _, h := range citations {
			if !relocationHash(h) || seen[h] {
				return nil, nil, fmt.Errorf("invalid duplicate citation")
			}
			seen[h] = true
		}
	}
	return in, associations, nil
}

// Exact bundle seals must still match the already replayed observations. This
// reread supplies retained catalogue values, never a replacement source proof.
func dossierBundles(evidence []map[string]json.RawMessage, observations []map[string]json.RawMessage) (map[string]map[string]json.RawMessage, error) {
	expected := map[string]json.RawMessage{}
	for _, o := range observations {
		expected[refreshString(o["evidence_id"])] = o["bundle_digest"]
	}
	out := map[string]map[string]json.RawMessage{}
	for _, ev := range evidence {
		ep, e := comparisonEndpoint(ev["endpoint"])
		if e != nil {
			return nil, e
		}
		raw, e := knowledgeengine.ReadPayload(ep.input)
		if e != nil {
			return nil, e
		}
		var b map[string]json.RawMessage
		if json.Unmarshal(raw, &b) != nil {
			return nil, fmt.Errorf("invalid retained bundle")
		}
		id := refreshString(ev["evidence_id"])
		body := map[string]any{}
		for k, v := range b {
			if k != "digest" {
				body[k] = v
			}
		}
		sealed, e := sealSHVActivation(body)
		if e != nil {
			return nil, e
		}
		if !refreshEqual(sealed, json.RawMessage(raw)) || !refreshEqual(b["digest"], expected[id]) {
			return nil, fmt.Errorf("dossier bundle changed since replay")
		}
		out[id] = b
	}
	return out, nil
}
func evaluateSHVDossier(raw json.RawMessage, prefix, version string) (json.RawMessage, error) {
	in, associations, e := dossierAdmission(raw)
	if e != nil {
		return nil, e
	}
	inventory, e := evaluateSHVInventory(in["inventory"], prefix, version)
	if e != nil {
		return nil, e
	}
	_, roster, evidence, e := inventoryRequest(in["inventory"])
	if e != nil {
		return nil, e
	}
	var inv map[string]json.RawMessage
	_ = json.Unmarshal(inventory, &inv)
	var observations, stages []map[string]json.RawMessage
	_ = json.Unmarshal(inv["evidence"], &observations)
	_ = json.Unmarshal(inv["rows"], &stages)
	bundles, e := dossierBundles(evidence, observations)
	if e != nil {
		return nil, e
	}
	byID := map[string]map[string]json.RawMessage{}
	stageByID := map[string]map[string]json.RawMessage{}
	for _, r := range roster {
		byID[refreshString(r["id"])] = r
	}
	for _, s := range stages {
		stageByID[refreshString(s["id"])] = s
	}
	rows := []any{}
	evaluations := []any{}
	for _, ev := range evidence {
		id := refreshString(ev["evidence_id"])
		evaluations = append(evaluations, map[string]any{"evidence_id": id, "evaluation": bundles[id]["evaluation"]})
	}
	for _, a := range associations {
		id := refreshString(a["roster_id"])
		r := byID[id]
		b, available := bundles[refreshString(r["evidence_id"])]
		var subject any
		var catalogueDigest any
		assertions := map[string]json.RawMessage{}
		status := "subject_unresolved"
		if available {
			var cat map[string]json.RawMessage
			_ = json.Unmarshal(b["catalogue"], &cat)
			catalogueDigest = cat["digest"]
			var subjects []map[string]json.RawMessage
			_ = json.Unmarshal(cat["subjects"], &subjects)
			for _, s := range subjects {
				if refreshEqual(s["id"], r["subject_id"]) {
					subject = s
					if refreshString(stageByID[id]["stage"]) == "interpreted" {
						status = "caller_association"
						var aa []json.RawMessage
						_ = json.Unmarshal(s["assertions"], &aa)
						for _, v := range aa {
							h, e := dossierAssertionDigest(v)
							if e != nil {
								return nil, e
							}
							assertions[h] = v
						}
					}
					break
				}
			}
		}
		var selected []string
		_ = json.Unmarshal(a["citation_digests"], &selected)
		citations := []any{}
		all := true
		for _, h := range selected {
			v, found := assertions[h]
			finding := "missing_assertion"
			var value any
			if found {
				finding = "exact_assertion_present"
				value = v
			} else {
				all = false
			}
			citations = append(citations, map[string]any{"digest": h, "finding": finding, "assertion": value})
		}
		if status == "caller_association" && len(selected) > 0 {
			if all {
				status = "source_cited_association"
			} else {
				status = "citation_unresolved"
			}
		}
		rows = append(rows, map[string]any{"id": a["id"], "roster_id": a["roster_id"], "relation": a["relation"], "status": status, "catalogue_digest": catalogueDigest, "subject": subject, "citations": citations, "association_authority": "caller", "identity_verified": false})
	}
	return sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-dossier.v1", "request": json.RawMessage(raw), "inventory": inventory, "associations": rows, "evaluations": evaluations, "identity_resolution": "not_performed", "catalogue_published": false})
}
func dossierAssertionDigest(raw json.RawMessage) (string, error) {
	sealed, e := sealSHVActivation(map[string]any{"assertion": raw})
	if e != nil {
		return "", e
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(sealed, &m)
	return refreshString(m["digest"]), nil
}
func dossierGraph(raw json.RawMessage) (json.RawMessage, error) {
	var d, req, component map[string]json.RawMessage
	_ = json.Unmarshal(raw, &d)
	_ = json.Unmarshal(d["request"], &req)
	_ = json.Unmarshal(req["component"], &component)
	nodes := []map[string]any{{"id": "component", "labels": []string{"CallerComponent"}, "properties": map[string]any{"declaration": component, "identity_verified": false}}}
	edges := []map[string]any{}
	var rows []map[string]json.RawMessage
	_ = json.Unmarshal(d["associations"], &rows)
	for _, row := range rows {
		id := "association:" + refreshString(row["id"])
		nodes = append(nodes, map[string]any{"id": id, "labels": []string{"CallerAssociation"}, "properties": row})
		edges = append(edges, map[string]any{"id": "edge:" + refreshString(row["id"]), "from": "component", "to": id, "label": "caller_associates", "properties": map[string]any{"relation": row["relation"], "identity_verified": false}})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i]["id"].(string) < nodes[j]["id"].(string) })
	sort.Slice(edges, func(i, j int) bool { return edges[i]["id"].(string) < edges[j]["id"].(string) })
	return sealSHVActivation(map[string]any{"protocol": "symphony.graph.exchange.v1", "owner": map[string]any{"engine_id": "qxctl-shv-dossier", "engine_version": "1", "artifact_protocol": d["protocol"], "artifact_digest": d["digest"]}, "owner_artifact": raw, "nodes": nodes, "edges": edges})
}

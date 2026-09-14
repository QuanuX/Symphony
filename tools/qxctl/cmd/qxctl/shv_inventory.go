package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
	"net/url"
	"os"
)

//go:embed shv_inventory.schema.json
var inventorySchema json.RawMessage

func newSHVInventoryCommand() *cobra.Command {
	root := structural("inventory", fmt.Errorf("inventory operation required"))
	for _, op := range []string{"run", "compare", "schema", "template"} {
		var prefix, version, input string
		c := &cobra.Command{Use: op, Args: usageOnlyArgs, Short: "Account for a caller roster using native evidence replay", RunE: func(*cobra.Command, []string) error {
			if op == "schema" || op == "template" {
				m := map[string]any{"protocol": "symphony.qxctl.shv-inventory-" + op + ".v1", "origin": "qxctl_embedded"}
				if op == "schema" {
					m["schema"] = inventorySchema
				} else {
					m["template"] = map[string]any{"profile": nil, "roster": []any{}, "evidence": []any{}}
					m["status"] = "unanswered_template_not_validated_input"
				}
				r, e := sealSHVActivation(m)
				if e != nil {
					return e
				}
				return printIndentedJSON(r)
			}
			raw, e := knowledgeengine.ReadPayload(input)
			if e != nil {
				return e
			}
			if op == "run" {
				r, e := evaluateSHVInventory(raw, prefix, version)
				if e != nil {
					return e
				}
				return printIndentedJSON(r)
			}
			in, e := jobDecode(raw, "previous", "current")
			if e != nil {
				return e
			}
			before, e := evaluateSHVInventory(in["previous"], prefix, version)
			if e != nil {
				return e
			}
			after, e := evaluateSHVInventory(in["current"], prefix, version)
			if e != nil {
				return e
			}
			changes, e := inventoryChanges(before, after)
			if e != nil {
				return e
			}
			var prev, current map[string]json.RawMessage
			_ = json.Unmarshal(before, &prev)
			_ = json.Unmarshal(after, &current)
			if !refreshEqual(prev["installation"], current["installation"]) {
				return fmt.Errorf("coverage installation changed between inventory observations")
			}
			dimensions := map[string]bool{}
			for _, key := range []string{"request", "evidence", "coverage"} {
				dimensions[key] = !refreshEqual(prev[key], current[key])
			}
			r, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-inventory-comparison.v1", "previous": before, "current": after, "changes": changes, "change_scope": "stage_rows_only", "dimensions_changed": dimensions, "catalogue_published": false})
			if e != nil {
				return e
			}
			return printIndentedJSON(r)
		}}
		c.Flags().Bool("json", false, "emit structured evidence")
		if op == "run" || op == "compare" {
			c.Flags().StringVar(&prefix, "prefix", "", "explicit coverage engine installation")
			c.Flags().StringVar(&version, "version", "", "exact coverage engine version")
			c.Flags().StringVar(&input, "input", "", "caller inventory or pair of inventory requests")
			for _, f := range []string{"prefix", "version", "input"} {
				_ = c.MarkFlagRequired(f)
			}
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "discover"
		if op == "run" || op == "compare" {
			interaction = "invoke"
		}
		s := commandSpec("shv.inventory."+op, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		protocol := "symphony.qxctl.shv-inventory-" + op + ".v1"
		if op == "run" || op == "compare" {
			if op == "run" {
				protocol = "symphony.qxctl.shv-inventory.v1"
			} else {
				protocol = "symphony.qxctl.shv-inventory-comparison.v1"
			}
			s.InputProtocols = []string{"symphony.qxctl.shv-inventory-input.v1"}
			if op == "compare" {
				s.InputProtocols = []string{"symphony.qxctl.shv-inventory-comparison-input.v1"}
			}
			s.BackendOperationIDs = []string{"engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project"}
			for _, f := range []string{"shv-engine", "shv-source-engine"} {
				s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + f, Interaction: "invoke"})
			}
		}
		s.OutputProtocols = []string{protocol}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	return root
}
func inventoryRequest(raw json.RawMessage) (map[string]json.RawMessage, []map[string]json.RawMessage, []map[string]json.RawMessage, error) {
	in, e := jobDecode(raw, "profile", "roster", "evidence")
	if e != nil {
		return nil, nil, nil, e
	}
	var roster, evidence []map[string]json.RawMessage
	if json.Unmarshal(in["roster"], &roster) != nil || roster == nil || len(roster) > 32 || json.Unmarshal(in["evidence"], &evidence) != nil || evidence == nil || len(evidence) > 8 {
		return nil, nil, nil, fmt.Errorf("inventory outside array bounds")
	}
	ids := map[string]bool{}
	for _, ev := range evidence {
		if _, e = jobDecode(jobMarshal(ev), "evidence_id", "endpoint"); e != nil {
			return nil, nil, nil, e
		}
		id := refreshString(ev["evidence_id"])
		if !jobToken.MatchString(id) || ids[id] {
			return nil, nil, nil, fmt.Errorf("invalid/duplicate evidence ID")
		}
		ids[id] = true
		if _, e = comparisonEndpoint(ev["endpoint"]); e != nil {
			return nil, nil, nil, e
		}
	}
	seen := map[string]bool{}
	for _, row := range roster {
		if _, e = jobDecode(jobMarshal(row), "id", "subject_id", "manufacturer", "model", "hardware_class", "locator", "evidence_id"); e != nil {
			return nil, nil, nil, e
		}
		id := refreshString(row["id"])
		if !jobToken.MatchString(id) || seen[id] || !jobToken.MatchString(refreshString(row["subject_id"])) || !jobToken.MatchString(refreshString(row["hardware_class"])) {
			return nil, nil, nil, fmt.Errorf("invalid/duplicate roster identity")
		}
		seen[id] = true
		for _, key := range []string{"manufacturer", "model"} {
			s := refreshString(row[key])
			if s == "" || len(s) > 256 {
				return nil, nil, nil, fmt.Errorf("invalid roster label")
			}
		}
		if string(row["locator"]) != "null" {
			s := refreshString(row["locator"])
			u, e := url.Parse(s)
			if e != nil || !u.IsAbs() || len(s) > 2048 {
				return nil, nil, nil, fmt.Errorf("invalid declared locator")
			}
		}
		if string(row["evidence_id"]) != "null" && !ids[refreshString(row["evidence_id"])] {
			return nil, nil, nil, fmt.Errorf("unlisted evidence ID")
		}
	}
	return in, roster, evidence, nil
}
func evaluateSHVInventory(raw json.RawMessage, prefix, version string) (json.RawMessage, error) {
	in, roster, evidence, e := inventoryRequest(raw)
	if e != nil {
		return nil, e
	}
	inst, e := knowledgeengine.InspectSHV(prefix, version)
	if e != nil {
		return nil, e
	}
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	// Validate caller policy before replaying any evidence.
	if _, e = knowledgeengine.InvokeSHV(context.Background(), prefix, version, cwd, "coverage_plan", jobMarshal(map[string]any{"profile": in["profile"], "subjects": []any{}})); e != nil {
		return nil, e
	}
	bundles := map[string]map[string]json.RawMessage{}
	observations := []any{}
	for _, ev := range evidence {
		ep, e := comparisonEndpoint(ev["endpoint"])
		if e != nil {
			return nil, e
		}
		body, e := knowledgeengine.ReadPayload(ep.input)
		if e != nil {
			return nil, e
		}
		var proof json.RawMessage
		if e = emitSHVRefresh("verify", ep, body, func(r json.RawMessage) error { proof = r; return nil }); e != nil {
			return nil, e
		}
		var bundle map[string]json.RawMessage
		_ = json.Unmarshal(body, &bundle)
		id := refreshString(ev["evidence_id"])
		bundles[id] = bundle
		var src, cat map[string]json.RawMessage
		_ = json.Unmarshal(bundle["source"], &src)
		_ = json.Unmarshal(bundle["catalogue"], &cat)
		pi, e := partitionFromBundle(bundle)
		if e != nil {
			return nil, e
		}
		var p map[string]json.RawMessage
		_ = json.Unmarshal(pi, &p)
		observations = append(observations, map[string]any{"evidence_id": id, "bundle_digest": bundle["digest"], "source_revision_digest": src["digest"], "dependencies": p["dependencies"], "source_installation": bundle["source_installation"], "kernel_installation": bundle["kernel_installation"], "replay": proof})
	}
	rows := []any{}
	summaries := []any{}
	counts := map[string]int{"expected": 0, "discovered": 0, "acquired": 0, "interpreted": 0}
	for _, row := range roster {
		stage := "expected"
		finding := "no_locator_or_evidence"
		var introduced any
		if string(row["locator"]) != "null" {
			stage = "discovered"
			finding = "locator_is_caller_declaration"
		}
		if b, ok := bundles[refreshString(row["evidence_id"])]; ok {
			stage = "acquired"
			finding = "subject_not_interpreted"
			var cat map[string]json.RawMessage
			_ = json.Unmarshal(b["catalogue"], &cat)
			var subjects []map[string]json.RawMessage
			_ = json.Unmarshal(cat["subjects"], &subjects)
			for _, s := range subjects {
				if !refreshEqual(s["id"], row["subject_id"]) {
					continue
				}
				matched := true
				for _, key := range []string{"manufacturer", "model", "hardware_class"} {
					matched = matched && refreshEqual(s[key], row[key])
				}
				if matched {
					stage = "interpreted"
					finding = "none"
					introduced = s["introduced"]
				} else {
					finding = "roster_identity_mismatch"
				}
				break
			}
		}
		counts[stage]++
		rows = append(rows, map[string]any{"id": row["id"], "stage": stage, "finding": finding, "introduced": introduced})
		summaries = append(summaries, map[string]any{"id": row["id"], "manufacturer": row["manufacturer"], "model": row["model"], "hardware_class": row["hardware_class"], "introduced": introduced})
	}
	coverage, e := knowledgeengine.InvokeSHV(context.Background(), prefix, version, cwd, "coverage_plan", jobMarshal(map[string]any{"profile": in["profile"], "subjects": summaries}))
	if e != nil {
		return nil, e
	}
	after, e := knowledgeengine.InspectSHV(prefix, version)
	if e != nil {
		return nil, e
	}
	if after != inst {
		return nil, fmt.Errorf("inventory coverage installation changed")
	}
	return sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-inventory.v1", "request": json.RawMessage(raw), "rows": rows, "stage_counts": counts, "coverage": coverage.Result, "evidence": observations, "installation": inst, "locator_validation": "caller_declaration", "dependency_validation": "listed_evidence_sequentially_replayed", "catalogue_published": false})
}
func inventoryChanges(before, after json.RawMessage) ([]any, error) {
	var a, b map[string]json.RawMessage
	if e := json.Unmarshal(before, &a); e != nil {
		return nil, e
	}
	if e := json.Unmarshal(after, &b); e != nil {
		return nil, e
	}
	var old, now []map[string]json.RawMessage
	_ = json.Unmarshal(a["rows"], &old)
	_ = json.Unmarshal(b["rows"], &now)
	index := map[string]map[string]json.RawMessage{}
	for _, r := range old {
		index[refreshString(r["id"])] = r
	}
	changes := []any{}
	seen := map[string]bool{}
	for _, r := range now {
		id := refreshString(r["id"])
		seen[id] = true
		prior, exists := index[id]
		if !exists || !refreshEqual(prior, r) {
			var previous any
			if exists {
				previous = prior
			}
			changes = append(changes, map[string]any{"id": id, "previous": previous, "current": r})
		}
	}
	for _, r := range old {
		id := refreshString(r["id"])
		if !seen[id] {
			changes = append(changes, map[string]any{"id": id, "previous": r, "current": nil})
		}
	}
	return changes, nil
}

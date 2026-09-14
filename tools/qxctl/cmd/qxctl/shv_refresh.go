package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvstate"
	"github.com/spf13/cobra"
)

const shvRefreshProtocol = "symphony.qxctl.shv-refresh-bundle.v1"

//go:embed shv_refresh.schema.json
var shvRefreshSchema json.RawMessage

type shvRefreshOptions struct{ sourcePrefix, sourceVersion, prefix, version, stateRoot, topsID, sourceID, sourceRoot, input string }

func newSHVRefreshCommand() *cobra.Command {
	root := structural("refresh", fmt.Errorf("refresh requires build, verify, schema or template"))
	for _, op := range []string{"build", "verify", "schema", "template"} {
		o := shvRefreshOptions{}
		c := &cobra.Command{Use: op, Short: "Materialize or replay evidence under an exact protected source revision", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSHVRefresh(op, o) }}
		for _, f := range []struct {
			name  string
			value *string
		}{{"source-prefix", &o.sourcePrefix}, {"source-version", &o.sourceVersion}, {"prefix", &o.prefix}, {"version", &o.version}} {
			c.Flags().StringVar(f.value, f.name, "", "explicit exact installed engine selection")
			_ = c.MarkFlagRequired(f.name)
		}
		if op == "build" || op == "verify" {
			for _, f := range []struct {
				name  string
				value *string
			}{{"state-root", &o.stateRoot}, {"tops-id", &o.topsID}, {"source-id", &o.sourceID}, {"source-root", &o.sourceRoot}, {"input", &o.input}} {
				c.Flags().StringVar(f.value, f.name, "", "explicit source selection or retained input")
				_ = c.MarkFlagRequired(f.name)
			}
		}
		c.Flags().Bool("json", false, "emit complete structured evidence")
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "invoke"
		protocol := shvRefreshProtocol
		if op == "verify" {
			protocol = "symphony.qxctl.shv-refresh-verification.v1"
		}
		if op == "schema" || op == "template" {
			interaction = "discover"
			protocol = "symphony.qxctl.shv-refresh-" + op + ".v1"
		}
		s := commandSpec("shv.refresh."+op, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		s.OutputProtocols = []string{protocol}
		s.ResultValidationProtocols = s.OutputProtocols
		if interaction == "invoke" {
			s.InputProtocols = []string{"symphony.qxctl.shv-refresh-input.v1"}
			if op == "verify" {
				s.InputProtocols = []string{shvRefreshProtocol}
			}
			s.BackendOperationIDs = []string{"engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project"}
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-source-engine", Interaction: "invoke"}, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-engine", Interaction: "invoke"})
		}
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	return root
}

func refreshObject(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, error) {
	if e := knowledgeengine.ValidateSCVBundleText(raw); e != nil {
		return nil, e
	}
	var m map[string]json.RawMessage
	if e := json.Unmarshal(raw, &m); e != nil {
		return nil, e
	}
	if m == nil || len(m) != len(keys) {
		return nil, fmt.Errorf("refresh object has unexpected fields")
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return nil, fmt.Errorf("refresh object missing %s", k)
		}
	}
	return m, nil
}
func refreshEqual(a, b any) bool {
	x, _ := sealSHVActivation(map[string]any{"value": a})
	y, _ := sealSHVActivation(map[string]any{"value": b})
	return bytes.Equal(x, y)
}
func refreshString(raw json.RawMessage) string { var v string; _ = json.Unmarshal(raw, &v); return v }

func runSHVRefresh(op string, o shvRefreshOptions) error {
	sourceInstall, e := knowledgeengine.InspectSHVSource(o.sourcePrefix, o.sourceVersion)
	if e != nil {
		return e
	}
	kernelInstall, e := knowledgeengine.InspectSHV(o.prefix, o.version)
	if e != nil {
		return e
	}
	if op == "schema" || op == "template" {
		result := map[string]any{"protocol": "symphony.qxctl.shv-refresh-" + op + ".v1", "source_installation": sourceInstall, "kernel_installation": kernelInstall}
		if op == "schema" {
			result["schema"] = shvRefreshSchema
			result["origin"] = "qxctl_embedded"
		} else {
			result["template"] = map[string]any{"expected_source_digest": nil, "captures": nil, "mapping": nil, "profile": nil, "subject_ids": nil, "requirements": nil}
			result["status"] = "unanswered_template_not_validated_input"
		}
		raw, e := sealSHVActivation(result)
		if e != nil {
			return e
		}
		return printIndentedJSON(raw)
	}
	if op != "build" && op != "verify" {
		return fmt.Errorf("unknown refresh operation")
	}
	raw, e := knowledgeengine.ReadPayload(o.input)
	if e != nil {
		return e
	}
	if e = knowledgeengine.ValidateSCVBundleText(raw); e != nil {
		return e
	}
	request := raw
	var bundle map[string]json.RawMessage
	if op == "verify" {
		bundle, e = refreshObject(raw, "protocol", "tops_id", "source_id", "source", "source_installation", "kernel_installation", "request", "captures", "catalogue", "coverage", "evaluation", "source_graph", "catalogue_graph", "digest")
		if e != nil {
			return e
		}
		if refreshString(bundle["protocol"]) != shvRefreshProtocol || refreshString(bundle["tops_id"]) != o.topsID || refreshString(bundle["source_id"]) != o.sourceID || !refreshEqual(bundle["source_installation"], sourceInstall) || !refreshEqual(bundle["kernel_installation"], kernelInstall) {
			return fmt.Errorf("refresh bundle selection or installation mismatch")
		}
		request = bundle["request"]
	}
	in, e := refreshObject(request, "expected_source_digest", "captures", "mapping", "profile", "subject_ids", "requirements")
	if e != nil {
		return e
	}
	expected := refreshString(in["expected_source_digest"])
	if expected == "" {
		return fmt.Errorf("refresh requires exact source digest")
	}
	store, e := shvstate.New(o.stateRoot, o.topsID, o.sourceID)
	if e != nil {
		return e
	}
	var output json.RawMessage
	e = store.WithLock(func(tx *shvstate.Transaction) error {
		source := tx.Current()
		if bytes.Equal(source, []byte("null")) || len(source) == 0 {
			return fmt.Errorf("refresh requires committed source")
		}
		var selected map[string]json.RawMessage
		if e := json.Unmarshal(source, &selected); e != nil {
			return e
		}
		currentDigest := refreshString(selected["digest"])
		if op == "build" && expected != currentDigest {
			return fmt.Errorf("refresh source head changed")
		}
		if op == "verify" {
			found := false
			for _, revision := range tx.History() {
				if refreshEqual(revision, bundle["source"]) {
					source = revision
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("refresh source is not in committed history")
			}
			if e := json.Unmarshal(source, &selected); e != nil {
				return e
			}
		}
		if refreshString(selected["digest"]) != expected {
			return fmt.Errorf("refresh request does not select source revision")
		}
		built, e := buildSHVRefresh(o, source, request, in, sourceInstall, kernelInstall)
		if e != nil {
			return e
		}
		si, e := knowledgeengine.InspectSHVSource(o.sourcePrefix, o.sourceVersion)
		if e != nil {
			return e
		}
		ki, e := knowledgeengine.InspectSHV(o.prefix, o.version)
		if e != nil {
			return e
		}
		if !refreshEqual(si, sourceInstall) || !refreshEqual(ki, kernelInstall) {
			return fmt.Errorf("refresh installation changed during materialization")
		}
		output = built
		if op == "verify" {
			if !refreshEqual(json.RawMessage(raw), built) {
				return fmt.Errorf("refresh bundle differs from native replay")
			}
			var b map[string]json.RawMessage
			_ = json.Unmarshal(built, &b)
			output, e = sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-refresh-verification.v1", "bundle_digest": b["digest"], "selected_source_digest": expected, "current_source_digest": currentDigest, "source_is_current": expected == currentDigest, "valid": true})
			return e
		}
		return nil
	})
	if e != nil {
		return e
	}
	return printIndentedJSON(output)
}

func buildSHVRefresh(o shvRefreshOptions, source, request json.RawMessage, in map[string]json.RawMessage, si, ki knowledgeengine.Installation) (json.RawMessage, error) {
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	invoke := func(sourceOwner bool, op string, input any) (json.RawMessage, error) {
		raw, e := json.Marshal(input)
		if e != nil {
			return nil, e
		}
		var r knowledgeengine.Response
		if sourceOwner {
			r, e = knowledgeengine.InvokeSHVSource(context.Background(), o.sourcePrefix, o.sourceVersion, cwd, op, raw)
		} else {
			r, e = knowledgeengine.InvokeSHV(context.Background(), o.prefix, o.version, cwd, op, raw)
		}
		return r.Result, e
	}
	var specs []json.RawMessage
	if e = json.Unmarshal(in["captures"], &specs); e != nil || len(specs) < 1 || len(specs) > 8 {
		return nil, fmt.Errorf("refresh requires one to eight captures")
	}
	captures := []json.RawMessage{}
	manifests := []json.RawMessage{}
	for _, spec := range specs {
		m, e := refreshObject(spec, "locator_id", "resolved_uri", "redirect_chain", "observed_at", "upstream_revision", "manifest", "completeness", "issues")
		if e != nil {
			return nil, e
		}
		input := map[string]any{"source_root": o.sourceRoot, "source": source}
		for k, v := range m {
			input[k] = v
		}
		capture, e := invoke(true, "capture_import", input)
		if e != nil {
			return nil, e
		}
		captures = append(captures, capture)
		manifests = append(manifests, m["manifest"])
	}
	// The owner validates capture uniqueness/budgets before any catalogue work.
	sourceGraph, e := invoke(true, "graph_project", map[string]any{"source_root": o.sourceRoot, "captures": captures})
	if e != nil {
		return nil, e
	}
	var sr struct {
		Definition struct {
			SubjectIDs []string `json:"subject_ids"`
		} `json:"definition"`
	}
	if e = json.Unmarshal(source, &sr); e != nil {
		return nil, e
	}
	scope := map[string]bool{}
	for _, id := range sr.Definition.SubjectIDs {
		scope[id] = true
	}
	var mapping []struct {
		ID string `json:"id"`
	}
	if e = json.Unmarshal(in["mapping"], &mapping); e != nil || mapping == nil {
		return nil, fmt.Errorf("refresh mapping must be an array")
	}
	for _, m := range mapping {
		if !scope[m.ID] {
			return nil, fmt.Errorf("mapped subject is outside selected source scope")
		}
	}
	catalogue, e := invoke(false, "catalogue_build", map[string]any{"source_root": o.sourceRoot, "sources": manifests, "subjects": in["mapping"]})
	if e != nil {
		return nil, e
	}
	var cat struct {
		Subjects []map[string]json.RawMessage `json:"subjects"`
	}
	if e = json.Unmarshal(catalogue, &cat); e != nil {
		return nil, e
	}
	inventory := []map[string]json.RawMessage{}
	for _, s := range cat.Subjects {
		row := map[string]json.RawMessage{}
		for _, k := range []string{"id", "manufacturer", "model", "hardware_class", "introduced"} {
			row[k] = s[k]
		}
		inventory = append(inventory, row)
	}
	coverage, e := invoke(false, "coverage_plan", map[string]any{"profile": in["profile"], "subjects": inventory})
	if e != nil {
		return nil, e
	}
	evaluation, e := invoke(false, "evaluate", map[string]any{"source_root": o.sourceRoot, "catalogue": catalogue, "subject_ids": in["subject_ids"], "requirements": in["requirements"]})
	if e != nil {
		return nil, e
	}
	graph, e := invoke(false, "graph_project", map[string]any{"source_root": o.sourceRoot, "catalogue": catalogue})
	if e != nil {
		return nil, e
	}
	result, e := sealSHVActivation(map[string]any{"protocol": shvRefreshProtocol, "tops_id": o.topsID, "source_id": o.sourceID, "source": source, "source_installation": si, "kernel_installation": ki, "request": request, "captures": captures, "catalogue": catalogue, "coverage": coverage, "evaluation": evaluation, "source_graph": sourceGraph, "catalogue_graph": graph})
	if e != nil {
		return nil, e
	}
	if e = refreshReplayBound(result); e != nil {
		return nil, e
	}
	if e = knowledgeengine.ValidateSCVBundleText(result); e != nil {
		return nil, e
	}
	return result, nil
}

// Admission binds the actual indented CLI artifact, including its final newline.
func refreshReplayBound(raw json.RawMessage) error {
	pretty, e := json.MarshalIndent(raw, "", "  ")
	if e != nil {
		return e
	}
	if len(pretty)+1 > 1<<20 {
		return fmt.Errorf("refresh bundle exceeds replay input bound")
	}
	return nil
}

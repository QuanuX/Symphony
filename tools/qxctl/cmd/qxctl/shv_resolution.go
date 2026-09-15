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
)

//go:embed shv_resolution.schema.json
var resolutionSchema json.RawMessage

func newSHVResolutionCommand() *cobra.Command {
	root := structural("resolution", fmt.Errorf("resolution operation required"))
	for _, op := range []string{"run", "schema", "template"} {
		var prefix, version, input string
		c := &cobra.Command{Use: op, Args: usageOnlyArgs, Short: "Resolve declared missing partitions from explicit exact inventory", RunE: func(*cobra.Command, []string) error {
			if op != "run" {
				m := map[string]any{"protocol": "symphony.qxctl.shv-resolution-" + op + ".v1", "origin": "qxctl_embedded"}
				if op == "schema" {
					m["schema"] = resolutionSchema
				} else {
					m["template"] = map[string]any{"manifest": nil, "candidates": []any{}}
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
			plan, e := prepareSHVResolutionVersion(raw, version)
			if e != nil {
				return e
			}
			inst, e := knowledgeengine.InspectSHVPartition(prefix, version)
			if e != nil {
				return e
			}
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			// Native ownership validates base, all candidates (including unused), then result.
			for _, in := range []json.RawMessage{plan.baseInput, plan.candidateInput} {
				if _, e = knowledgeengine.InvokeSHVPartition(context.Background(), prefix, version, cwd, "manifest_build", in); e != nil {
					return e
				}
			}
			out, e := knowledgeengine.InvokeSHVPartition(context.Background(), prefix, version, cwd, "manifest_build", plan.resultInput)
			if e != nil {
				return e
			}
			after, e := knowledgeengine.InspectSHVPartition(prefix, version)
			if e != nil {
				return e
			}
			if after != inst {
				return fmt.Errorf("resolution installation changed")
			}
			result, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-resolution.v1", "input_digest": plan.inputDigest, "base_manifest_digest": plan.baseDigest, "manifest": out.Result, "resolved_partition_digests": plan.resolved, "unused_candidate_digests": plan.unused, "dependency_validation": "reference_declarations_only", "catalogue_published": false, "installation": inst})
			if e != nil {
				return e
			}
			return printIndentedJSON(result)
		}}
		c.Flags().Bool("json", false, "emit structured evidence")
		if op == "run" {
			c.Flags().StringVar(&prefix, "prefix", "", "exact partition installation")
			c.Flags().StringVar(&version, "version", "", "exact partition version")
			c.Flags().StringVar(&input, "input", "", "manifest and explicit candidate inventory JSON")
			for _, f := range []string{"prefix", "version", "input"} {
				_ = c.MarkFlagRequired(f)
			}
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "discover"
		if op == "run" {
			interaction = "invoke"
		}
		s := commandSpec("shv.partition.resolution."+op, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		protocol := "symphony.qxctl.shv-resolution-" + op + ".v1"
		if op == "run" {
			protocol = "symphony.qxctl.shv-resolution.v1"
			s.InputProtocols = []string{"symphony.qxctl.shv-resolution-input.v1"}
			s.BackendOperationIDs = []string{"engop:symphony:shv-partition.manifest.build"}
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-partition-engine", Interaction: "invoke"})
		}
		s.OutputProtocols = []string{protocol}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	return root
}

type shvResolutionPlan struct {
	baseInput, candidateInput, resultInput json.RawMessage
	baseDigest, inputDigest                string
	resolved, unused                       []string
}

func prepareSHVResolution(raw json.RawMessage) (shvResolutionPlan, error) {
	return prepareSHVResolutionVersion(raw, knowledgeengine.SHVPartitionVersion)
}
func prepareSHVResolutionVersion(raw json.RawMessage, version string) (shvResolutionPlan, error) {
	p := shvResolutionPlan{resolved: []string{}, unused: []string{}}
	in, e := refreshObject(raw, "manifest", "candidates")
	if e != nil {
		return p, e
	}
	m, e := refreshObject(in["manifest"], "protocol", "entries", "required_references", "reference_statuses", "loaded_count", "missing_count", "complete_inventory", "digest")
	if e != nil {
		return p, e
	}
	p.baseInput = jobMarshal(map[string]any{"entries": m["entries"], "required_references": m["required_references"]})
	if e = knowledgeengine.ValidateSHVPartitionResultVersion("manifest_build", p.baseInput, in["manifest"], version); e != nil {
		return p, e
	}
	var candidates []json.RawMessage
	if json.Unmarshal(in["candidates"], &candidates) != nil || string(in["candidates"]) == "null" || len(candidates) > 64 {
		return p, fmt.Errorf("candidate inventory must be an array of at most 64 partitions")
	}
	entries := []any{}
	byID := map[string]json.RawMessage{}
	ordered := []string{}
	for _, candidate := range candidates {
		var c map[string]json.RawMessage
		if json.Unmarshal(candidate, &c) != nil {
			return p, fmt.Errorf("invalid candidate")
		}
		id := refreshString(c["digest"])
		if _, exists := byID[id]; exists {
			return p, fmt.Errorf("duplicate candidate digest")
		}
		byID[id] = candidate
		ordered = append(ordered, id)
		entries = append(entries, map[string]any{"partition_digest": id, "partition": candidate})
	}
	p.candidateInput = jobMarshal(map[string]any{"entries": entries, "required_references": []any{}})
	if _, e = knowledgeengine.ExpectedSHVPartitionVersion("manifest_build", p.candidateInput, version); e != nil {
		return p, e
	}
	var base []map[string]json.RawMessage
	_ = json.Unmarshal(m["entries"], &base)
	used := map[string]bool{}
	for _, entry := range base {
		id := refreshString(entry["partition_digest"])
		if candidate, exists := byID[id]; exists && string(entry["partition"]) == "null" {
			entry["partition"] = candidate
			p.resolved = append(p.resolved, id)
			used[id] = true
		}
	}
	for _, id := range ordered {
		if !used[id] {
			p.unused = append(p.unused, id)
		}
	}
	p.resultInput = jobMarshal(map[string]any{"entries": base, "required_references": m["required_references"]})
	if _, e = knowledgeengine.ExpectedSHVPartitionVersion("manifest_build", p.resultInput, version); e != nil {
		return p, e
	}
	p.baseDigest = refreshString(m["digest"])
	// Digest the complete request, preserving number semantics through RawMessage normalization.
	sealed, e := sealSHVActivation(map[string]any{"manifest": in["manifest"], "candidates": in["candidates"]})
	if e != nil {
		return p, e
	}
	var identity map[string]json.RawMessage
	_ = json.Unmarshal(sealed, &identity)
	p.inputDigest = refreshString(identity["digest"])
	return p, nil
}

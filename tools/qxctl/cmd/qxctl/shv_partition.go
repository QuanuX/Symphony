package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
	"os"
)

func newSHVPartitionCommand() *cobra.Command {
	root := structural("partition", fmt.Errorf("partition operation required"))
	for _, leaf := range []string{"inspect", "build", "manifest", "query", "schema", "template", "from-refresh"} {
		var prefix, version, input, operation string
		c := &cobra.Command{Use: leaf, Args: usageOnlyArgs, Short: "Invoke exact native immutable partition owner", RunE: func(*cobra.Command, []string) error {
			if leaf == "schema" || leaf == "template" {
				inst, raw, e := knowledgeengine.SHVPartitionResource(prefix, version, leaf == "template")
				if e != nil {
					return e
				}
				out := map[string]any{"protocol": "symphony.qxctl.shv-partition-" + leaf + ".v1", "installation": inst}
				if leaf == "schema" {
					out["schema"] = raw
				} else {
					var templates map[string]json.RawMessage
					_ = json.Unmarshal(raw, &templates)
					v, ok := templates[operation]
					if !ok {
						return fmt.Errorf("unknown partition template")
					}
					out["operation"] = operation
					out["template"] = v
					out["status"] = "unanswered_template_not_validated_input"
				}
				sealed, e := sealSHVActivation(out)
				if e != nil {
					return e
				}
				return printIndentedJSON(sealed)
			}
			raw := json.RawMessage(`{}`)
			var e error
			if leaf != "inspect" {
				raw, e = knowledgeengine.ReadPayload(input)
				if e != nil {
					return e
				}
			}
			var bundle map[string]json.RawMessage
			var proof json.RawMessage
			if leaf == "from-refresh" {
				o, e := comparisonEndpoint(raw)
				if e != nil {
					return e
				}
				body, e := knowledgeengine.ReadPayload(o.input)
				if e != nil {
					return e
				}
				if e = emitSHVRefresh("verify", o, body, func(v json.RawMessage) error { proof = v; return nil }); e != nil {
					return e
				}
				if e = json.Unmarshal(body, &bundle); e != nil {
					return e
				}
				raw, e = partitionFromBundle(bundle)
				if e != nil {
					return e
				}
			}
			op := map[string]string{"inspect": "inspect", "build": "partition_build", "manifest": "manifest_build", "query": "manifest_query", "from-refresh": "partition_build"}[leaf]
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			selected, e := knowledgeengine.InspectSHVPartition(prefix, version)
			if e != nil {
				return e
			}
			response, e := knowledgeengine.InvokeSHVPartition(context.Background(), prefix, version, cwd, op, raw)
			if e != nil {
				return e
			}
			if leaf != "from-refresh" {
				return printIndentedJSON(response)
			}
			inst, e := knowledgeengine.InspectSHVPartition(prefix, version)
			if e != nil {
				return e
			}
			if inst != selected {
				return fmt.Errorf("partition installation changed before result")
			}
			sealed, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-partition-from-refresh.v1", "partition": response.Result, "bundle_digest": bundle["digest"], "replay": proof, "installation": inst})
			if e != nil {
				return e
			}
			return printIndentedJSON(sealed)
		}}
		c.Flags().StringVar(&prefix, "prefix", "", "exact partition engine installation")
		c.Flags().StringVar(&version, "version", "", "exact partition engine version")
		_ = c.MarkFlagRequired("prefix")
		_ = c.MarkFlagRequired("version")
		c.Flags().Bool("json", false, "emit structured result")
		if leaf != "inspect" && leaf != "schema" && leaf != "template" {
			c.Flags().StringVar(&input, "input", "", "bounded operation or explicit refresh endpoint JSON")
			_ = c.MarkFlagRequired("input")
		}
		if leaf == "template" {
			c.Flags().StringVar(&operation, "operation", "", "partition_build, manifest_build or manifest_query")
			_ = c.MarkFlagRequired("operation")
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "invoke"
		if leaf == "inspect" {
			interaction = "inspect"
		}
		if leaf == "query" {
			interaction = "query"
		}
		if leaf == "schema" || leaf == "template" {
			interaction = "discover"
		}
		s := commandSpec("shv.partition."+leaf, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-partition-engine", Interaction: interaction})
		op := map[string]string{"inspect": "inspect", "build": "partition_build", "manifest": "manifest_build", "query": "manifest_query", "from-refresh": "partition_build"}[leaf]
		if op != "" {
			s.BackendOperationIDs = []string{"engop:symphony:shv-partition." + map[string]string{"inspect": "inspect", "partition_build": "partition.build", "manifest_build": "manifest.build", "manifest_query": "manifest.query"}[op]}
			s.InputProtocols = []string{knowledgeengine.SHVPartitionInputProtocol(op)}
			protocol, _ := knowledgeengine.SHVPartitionResultProtocol(op)
			s.OutputProtocols = []string{protocol}
		} else {
			s.OutputProtocols = []string{"symphony.qxctl.shv-partition-" + leaf + ".v1"}
		}
		if leaf == "from-refresh" {
			s.InputProtocols = []string{"symphony.qxctl.shv-refresh-endpoint.v1"}
			s.OutputProtocols = []string{"symphony.qxctl.shv-partition-from-refresh.v1"}
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-source-engine", Interaction: "invoke"}, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-engine", Interaction: "invoke"})
			s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project")
		}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	root.AddCommand(newSHVResolutionCommand())
	return root
}
func partitionFromBundle(b map[string]json.RawMessage) (json.RawMessage, error) {
	var source map[string]json.RawMessage
	_ = json.Unmarshal(b["source"], &source)
	var cat map[string]json.RawMessage
	_ = json.Unmarshal(b["catalogue"], &cat)
	engine := func(raw json.RawMessage) map[string]any {
		var i knowledgeengine.Installation
		_ = json.Unmarshal(raw, &i)
		return map[string]any{"engine_id": i.EngineID, "version": i.Version, "executable_digest": i.ExecutableDigest}
	}
	var captures []map[string]json.RawMessage
	_ = json.Unmarshal(b["captures"], &captures)
	refs := []any{}
	for _, c := range captures {
		var m map[string]json.RawMessage
		_ = json.Unmarshal(c["manifest"], &m)
		refs = append(refs, map[string]any{"capture_id": m["id"], "capture_digest": c["digest"], "content_digest": m["digest"], "bytes": m["bytes"]})
	}
	var subjects []map[string]json.RawMessage
	_ = json.Unmarshal(cat["subjects"], &subjects)
	ids := []any{}
	for _, s := range subjects {
		ids = append(ids, s["id"])
	}
	var mapping any
	decoder := json.NewDecoder(bytes.NewReader(cat["mapping"]))
	decoder.UseNumber()
	if e := decoder.Decode(&mapping); e != nil {
		return nil, e
	}
	md, e := knowledgeengine.SCVDigest(map[string]any{"mapping": mapping})
	if e != nil {
		return nil, e
	}
	return json.Marshal(map[string]any{"dependencies": map[string]any{"source_revision_digest": source["digest"], "source_engine": engine(b["source_installation"]), "kernel_engine": engine(b["kernel_installation"]), "captures": refs, "mapping_digest": md, "catalogue_digest": cat["digest"]}, "subject_ids": ids})
}

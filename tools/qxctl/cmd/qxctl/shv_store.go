package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
	"os"
)

func newSHVStoreCommand() *cobra.Command {
	root := structural("store", fmt.Errorf("graph store operation required"))
	for _, op := range []string{"inspect", "prepare", "commit", "status", "query", "export", "inventory", "schema", "template"} {
		var prefix, version, backend, storeRoot, input, selection string
		c := &cobra.Command{Use: op, Short: "Persist or read structural graph evidence using an exact durable adapter", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
			if backend != "duckdb" {
				return fmt.Errorf("unsupported graph store backend")
			}
			if op == "schema" || op == "template" {
				inst, raw, e := knowledgeengine.SHVStoreResource(prefix, version, op == "template")
				if e != nil {
					return e
				}
				out := map[string]any{"protocol": "symphony.qxctl.shv-graph-store-" + op + ".v1", "installation": inst}
				if op == "schema" {
					out["schema"] = raw
				} else {
					var all map[string]json.RawMessage
					if e = json.Unmarshal(raw, &all); e != nil {
						return e
					}
					v, ok := all[selection]
					if !ok {
						return fmt.Errorf("unknown graph store template")
					}
					out["operation"] = selection
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
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			if op != "inspect" {
				raw, e = knowledgeengine.ReadPayload(input)
				if e != nil {
					return e
				}
				cwd = storeRoot
			}
			if op == "prepare" {
				var p map[string]json.RawMessage
				if e = json.Unmarshal(raw, &p); e != nil {
					return e
				}
				if _, exists := p["connector"]; exists {
					return fmt.Errorf("connector identity is selected only through connector flags")
				}
				inst, e := knowledgeengine.InspectSHVStore(prefix, version)
				if e != nil {
					return e
				}
				p["connector"], e = json.Marshal(inst)
				if e != nil {
					return e
				}
				raw, e = json.Marshal(p)
				if e != nil {
					return e
				}
			}
			r, e := knowledgeengine.InvokeSHVStore(context.Background(), prefix, version, cwd, op, raw)
			if e != nil {
				return e
			}
			return printIndentedJSON(r)
		}}
		c.Flags().StringVar(&prefix, "connector-prefix", "", "exact installed graph store connector")
		c.Flags().StringVar(&version, "connector-version", "", "exact connector version")
		c.MarkFlagRequired("connector-prefix")
		c.MarkFlagRequired("connector-version")
		c.Flags().StringVar(&backend, "backend", "duckdb", "explicit storage backend")
		c.Flags().Bool("json", false, "structured output")
		if op != "inspect" && op != "schema" && op != "template" {
			c.Flags().StringVar(&storeRoot, "store-root", "", "existing private caller-owned database directory")
			c.MarkFlagRequired("store-root")
			c.Flags().StringVar(&input, "input", "", "bounded operation JSON; scope declared here only")
			c.MarkFlagRequired("input")
		}
		if op == "template" {
			c.Flags().StringVar(&selection, "operation", "", "native operation template")
			c.MarkFlagRequired("operation")
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "query"
		if op == "prepare" || op == "commit" {
			interaction = "invoke"
		}
		if op == "inspect" || op == "status" {
			interaction = "inspect"
		}
		if op == "schema" || op == "template" {
			interaction = "discover"
		}
		spec := commandSpec("shv.graph.store."+op, featureSHVAdministration, interaction)
		spec.Mutability = "evidence_only"
		spec.TargetScope = "local"
		if op == "inspect" || op == "schema" || op == "template" || op == "inventory" {
			spec.Mutability = "read_only"
			spec.TargetScope = "local"
		}
		spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-graph-duckdb-connector", Interaction: interaction})
		if out, ok := knowledgeengine.SHVStoreResultProtocol(op); ok {
			spec.BackendOperationIDs = []string{"engop:symphony:shv.graph-store." + op}
			spec.InputProtocols = []string{knowledgeengine.SHVStoreInputProtocol(op)}
			spec.OutputProtocols = []string{out}
		} else {
			spec.OutputProtocols = []string{"symphony.qxctl.shv-graph-store-" + op + ".v1"}
		}
		if op == "commit" || op == "status" {
			spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-graph-duckdb-connector", Interaction: "recover"})
		}
		if op == "prepare" {
			spec.InputProtocols = []string{"symphony.qxctl.shv-graph-store-prepare-input.v1"}
		}
		spec.ResultValidationProtocols = spec.OutputProtocols
		commandregistry.Attach(c, spec)
		root.AddCommand(c)
	}
	return root
}

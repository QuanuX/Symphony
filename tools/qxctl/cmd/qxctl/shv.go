package main

import (
	"context"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

const featureSHVAdministration = "ssfv:symphony:qxctl.shv-administration"

func newSHVCommand() *cobra.Command {
	root := structural("shv", fmt.Errorf("SHV subcommand is required"))
	root.AddCommand(newSHVLeaf("inspect", "shv.inspect", "inspect", false), newSHVLeaf("evaluate", "shv.evaluate", "evaluate", false), newSHVDiscovery(false), newSHVDiscovery(true))
	coverage := structural("coverage", fmt.Errorf("coverage requires default or plan"))
	coverage.AddCommand(newSHVLeaf("default", "shv.coverage.default", "coverage_default", false), newSHVLeaf("plan", "shv.coverage.plan", "coverage_plan", false))
	root.AddCommand(coverage)
	catalogue := structural("catalogue", fmt.Errorf("catalogue requires build or query"))
	catalogue.AddCommand(newSHVLeaf("build", "shv.catalogue.build", "catalogue_build", false), newSHVLeaf("query", "shv.catalogue.query", "catalogue_query", false))
	root.AddCommand(catalogue)
	graph := structural("graph", fmt.Errorf("graph requires project, validate or adapter"))
	graph.AddCommand(newSHVLeaf("project", "shv.graph.project", "graph_project", false), newSHVLeaf("validate", "shv.graph.validate", "graph_validate", false))
	adapter := structural("adapter", fmt.Errorf("select an explicit installed graph adapter operation"))
	for _, op := range []string{"inspect", "roundtrip", "query"} {
		adapter.AddCommand(newSHVLeaf(op, "shv.graph.adapter."+op, op, true))
	}
	graph.AddCommand(adapter)
	root.AddCommand(graph)
	root.AddCommand(newSHVSourceCommand())
	return root
}
func newSHVLeaf(leaf, key, op string, adapter bool) *cobra.Command {
	var prefix, version, input string
	c := &cobra.Command{Use: leaf, Short: "Invoke exact installed SHV owner; portable adapter validates structure only", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		payload := []byte(`{}`)
		var err error
		if op != "inspect" {
			payload, err = knowledgeengine.ReadPayload(input)
			if err != nil {
				return err
			}
		} else if input != "" {
			return fmt.Errorf("inspect does not accept --input")
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		var response knowledgeengine.Response
		if adapter {
			response, err = knowledgeengine.InvokeSHVGraphAdapter(context.Background(), prefix, version, cwd, op, payload)
		} else {
			response, err = knowledgeengine.InvokeSHV(context.Background(), prefix, version, cwd, op, payload)
		}
		if err != nil {
			if response.Protocol != "" {
				_ = printIndentedJSON(response)
			}
			return err
		}
		return printIndentedJSON(response)
	}}
	pflag, vflag := "prefix", "version"
	if adapter {
		pflag, vflag = "adapter-prefix", "adapter-version"
	}
	c.Flags().StringVar(&prefix, pflag, "", "exact receipt-owned installation prefix")
	c.Flags().StringVar(&version, vflag, "", "exact supported version; no default or upgrade")
	c.Flags().StringVar(&input, "input", "", "bounded no-follow operation payload JSON")
	c.Flags().Bool("json", false, "emit validated structured JSON")
	_ = c.MarkFlagRequired(pflag)
	_ = c.MarkFlagRequired(vflag)
	if op != "inspect" {
		_ = c.MarkFlagRequired("input")
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	interaction := "invoke"
	if op == "inspect" {
		interaction = "inspect"
	}
	if op == "catalogue_query" || op == "query" {
		interaction = "query"
	}
	spec := commandSpec(key, featureSHVAdministration, interaction)
	spec.Mutability = "read_only"

	domain, feature := "shv", "ssfv:symphony:shv-engine"
	if adapter {
		domain, feature = "shv-graph-adapter", "ssfv:symphony:shv-graph-adapter"
	}
	spec.BackendOperationIDs = []string{"engop:symphony:" + domain + "." + strings.ReplaceAll(op, "_", ".")}
	spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: feature, Interaction: interaction})
	spec.InputProtocols = []string{knowledgeengine.SHVInputProtocol(op, adapter)}
	out, _ := knowledgeengine.SHVResultProtocol(op, adapter)
	spec.OutputProtocols = []string{out}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(c, spec)
	return c
}
func newSHVDiscovery(template bool) *cobra.Command {
	var prefix, version, adapterPrefix, adapterVersion, selection string
	leaf, protocol := "schema", "symphony.qxctl.shv-schema.v1"
	if template {
		leaf, protocol = "template", "symphony.qxctl.shv-template.v1"
	}
	c := &cobra.Command{Use: leaf, Short: "Inspect exact receipt-owned SHV or adapter resources without a checkout", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		adapter := adapterPrefix != "" || adapterVersion != ""
		if adapter {
			if prefix != "" || version != "" {
				return fmt.Errorf("select one exact SHV or adapter installation")
			}
			prefix, version = adapterPrefix, adapterVersion
		}
		out, err := knowledgeengine.SHVDiscovery(prefix, version, adapter, template, selection)
		if err != nil {
			return err
		}
		return printIndentedJSON(out)
	}}
	c.Flags().StringVar(&prefix, "prefix", "", "exact SHV prefix")
	c.Flags().StringVar(&version, "version", "", "exact SHV version")
	c.Flags().StringVar(&adapterPrefix, "adapter-prefix", "", "explicit alternative graph adapter prefix")
	c.Flags().StringVar(&adapterVersion, "adapter-version", "", "exact alternative adapter version")
	c.Flags().Bool("json", false, "emit structured JSON")
	if template {
		c.Flags().StringVar(&selection, "operation", "", "exact operation name; template marks unanswered fields")
		_ = c.MarkFlagRequired("operation")
	} else {
		c.Flags().StringVar(&selection, "protocol", "", "exact protocol to show; omit to list")
	}
	spec := commandSpec("shv."+leaf, featureSHVAdministration, "discover")
	spec.Mutability = "read_only"
	spec.OutputProtocols = []string{protocol}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(c, spec)
	return c
}

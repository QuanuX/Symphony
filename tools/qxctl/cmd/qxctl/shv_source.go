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

func newSHVSourceCommand() *cobra.Command {
	root := structural("source", fmt.Errorf("source subcommand required"))
	for _, op := range []string{"inspect", "plan", "reduce", "status"} {
		native := op
		if op != "inspect" {
			native = "source_" + op
		}
		root.AddCommand(newSHVSourceLeaf(op, "shv.source."+op, native))
	}
	capture := structural("capture", fmt.Errorf("capture requires import or compare"))
	for _, op := range []string{"import", "compare"} {
		capture.AddCommand(newSHVSourceLeaf(op, "shv.source.capture."+op, "capture_"+op))
	}
	root.AddCommand(capture)
	graph := structural("graph", fmt.Errorf("graph requires project or validate"))
	for _, op := range []string{"project", "validate"} {
		graph.AddCommand(newSHVSourceLeaf(op, "shv.source.graph."+op, "graph_"+op))
	}
	root.AddCommand(graph, newSHVSourceDiscovery(false), newSHVSourceDiscovery(true))
	return root
}
func newSHVSourceLeaf(leaf, key, op string) *cobra.Command {
	var prefix, version, input string
	c := &cobra.Command{Use: leaf, Short: "Read-only source lifecycle candidate; no persistence or authority authentication", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
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
		response, err := knowledgeengine.InvokeSHVSource(context.Background(), prefix, version, cwd, op, payload)
		if err != nil {
			if response.Protocol != "" {
				_ = printIndentedJSON(response)
			}
			return err
		}
		return printIndentedJSON(response)
	}}
	pflag, vflag := "prefix", "version"
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
	if op == "inspect" || op == "source_status" {
		interaction = "inspect"
	}
	spec := commandSpec(key, featureSHVAdministration, interaction)
	spec.Mutability = "read_only"

	domain, feature := "shv-source", "ssfv:symphony:shv-source-engine"
	spec.BackendOperationIDs = []string{"engop:symphony:" + domain + "." + strings.ReplaceAll(op, "_", ".")}
	spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: feature, Interaction: interaction})
	spec.InputProtocols = []string{knowledgeengine.SHVSourceInputProtocol(op)}
	out, _ := knowledgeengine.SHVSourceResultProtocol(op)
	spec.OutputProtocols = []string{out}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(c, spec)
	return c
}

func newSHVSourceDiscovery(template bool) *cobra.Command {
	var prefix, version, selection string
	leaf, protocol := "schema", "symphony.qxctl.shv-schema.v1"
	if template {
		leaf, protocol = "template", "symphony.qxctl.shv-template.v1"
	}
	c := &cobra.Command{Use: leaf, Short: "Read exact installed source lifecycle schema or unanswered template", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		out, e := knowledgeengine.SHVSourceDiscovery(prefix, version, template, selection)
		if e != nil {
			return e
		}
		return printIndentedJSON(out)
	}}
	c.Flags().StringVar(&prefix, "prefix", "", "exact receipt-owned prefix")
	c.Flags().StringVar(&version, "version", "", "exact supported version")
	c.Flags().Bool("json", false, "emit structured JSON")
	_ = c.MarkFlagRequired("prefix")
	_ = c.MarkFlagRequired("version")
	if template {
		c.Flags().StringVar(&selection, "operation", "", "exact native operation name")
		_ = c.MarkFlagRequired("operation")
	} else {
		c.Flags().StringVar(&selection, "protocol", "", "protocol to show; omit to list")
	}
	spec := commandSpec("shv.source."+leaf, featureSHVAdministration, "discover")
	spec.Mutability = "read_only"
	spec.OutputProtocols = []string{protocol}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(c, spec)
	return c
}

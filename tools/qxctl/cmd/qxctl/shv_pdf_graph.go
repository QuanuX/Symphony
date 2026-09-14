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

func newSHVPDFGraphCommand() *cobra.Command {
	root := structural("graph", fmt.Errorf("PDF graph operation required"))
	for _, leaf := range []string{"project", "validate", "roundtrip"} {
		var prefix, version, input, ap, av string
		c := &cobra.Command{Use: leaf, Args: usageOnlyArgs, Short: "Replay original PDF provenance through its exact native owner", RunE: func(*cobra.Command, []string) error {
			raw, e := knowledgeengine.ReadPayload(input)
			if e != nil {
				return e
			}
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			inst, e := knowledgeengine.InspectSHVPDF(prefix, version)
			if e != nil {
				return e
			}
			op := "graph_" + leaf
			if leaf == "roundtrip" {
				op = "graph_validate"
			}
			result, e := knowledgeengine.InvokeSHVPDF(context.Background(), prefix, version, cwd, op, raw)
			if e != nil {
				return e
			}
			if leaf != "roundtrip" {
				return printIndentedJSON(result)
			}
			p, e := jobDecode(raw, "request", "graph")
			if e != nil {
				return e
			}
			ai, e := knowledgeengine.InspectSHVGraphAdapter(ap, av)
			if e != nil {
				return e
			}
			transport, e := knowledgeengine.InvokeSHVGraphAdapter(context.Background(), ap, av, cwd, "roundtrip", jobMarshal(map[string]any{"graph": p["graph"]}))
			if e != nil {
				return e
			}
			var returned map[string]json.RawMessage
			if e = json.Unmarshal(transport.Result, &returned); e != nil {
				return e
			}
			replay, e := knowledgeengine.InvokeSHVPDF(context.Background(), prefix, version, cwd, "graph_validate", jobMarshal(map[string]any{"request": p["request"], "graph": returned["graph"]}))
			if e != nil {
				return e
			}
			after, e := knowledgeengine.InspectSHVPDF(prefix, version)
			if e != nil {
				return e
			}
			aa, e := knowledgeengine.InspectSHVGraphAdapter(ap, av)
			if e != nil {
				return e
			}
			if after != inst || aa != ai {
				return fmt.Errorf("PDF graph installation changed")
			}
			out, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-pdf-graph-roundtrip.v1", "installation": inst, "adapter_installation": ai, "transport": transport.Result, "source_validation": replay.Result, "documentary_lineages": 1})
			if e != nil {
				return e
			}
			return printIndentedJSON(out)
		}}
		c.Flags().StringVar(&prefix, "prefix", "", "exact PDF adapter installation")
		c.Flags().StringVar(&version, "version", "", "exact PDF adapter version")
		c.Flags().StringVar(&input, "input", "", "original-source extraction request or request/graph pair")
		c.Flags().Bool("json", false, "structured output/errors")
		for _, f := range []string{"prefix", "version", "input"} {
			_ = c.MarkFlagRequired(f)
		}
		s := commandSpec("shv.pdf.graph."+leaf, featureSHVAdministration, "invoke")
		s.Mutability = "read_only"
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-pdf-adapter", Interaction: "invoke"})
		op := "graph_" + leaf
		if leaf == "roundtrip" {
			op = "graph_validate"
		}
		protocol, _ := knowledgeengine.SHVPDFResultProtocol(op)
		s.InputProtocols = []string{knowledgeengine.SHVPDFInputProtocol(op)}
		s.BackendOperationIDs = []string{"engop:symphony:shv-pdf." + "graph." + leaf}
		if leaf == "roundtrip" {
			c.Flags().StringVar(&ap, "adapter-prefix", "", "exact generic graph installation")
			c.Flags().StringVar(&av, "adapter-version", "", "exact generic graph version")
			_ = c.MarkFlagRequired("adapter-prefix")
			_ = c.MarkFlagRequired("adapter-version")
			protocol = "symphony.qxctl.shv-pdf-graph-roundtrip.v1"
			s.BackendOperationIDs = []string{"engop:symphony:shv-pdf.graph.validate", "engop:symphony:shv-graph-adapter.roundtrip"}
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-graph-adapter", Interaction: "invoke"})
		}
		s.OutputProtocols = []string{protocol}
		s.ResultValidationProtocols = s.OutputProtocols
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	return root
}

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

func newSHVPDFCommand() *cobra.Command {
	root := structural("pdf", fmt.Errorf("PDF operation required"))
	for _, leaf := range []string{"inspect", "extract", "verify", "schema", "template"} {
		var prefix, version, input string
		c := &cobra.Command{Use: leaf, Args: usageOnlyArgs, Short: "Invoke exact installed bounded SHV PDF adapter", RunE: func(*cobra.Command, []string) error {
			if leaf == "schema" || leaf == "template" {
				inst, raw, e := knowledgeengine.SHVPDFResource(prefix, version, leaf == "template")
				if e != nil {
					return e
				}
				out, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-pdf-" + leaf + ".v1", "installation": inst, "resource": raw})
				if e != nil {
					return e
				}
				return printIndentedJSON(out)
			}
			raw := []byte(`{}`)
			var e error
			if leaf != "inspect" {
				raw, e = knowledgeengine.ReadPayload(input)
				if e != nil {
					return e
				}
			}
			var expected json.RawMessage
			if leaf == "verify" {
				in, e := jobDecode(raw, "request", "artifact")
				if e != nil {
					return e
				}
				raw = in["request"]
				expected = in["artifact"]
			}
			op := leaf
			if leaf == "verify" {
				op = "extract"
			}
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			inst, e := knowledgeengine.InspectSHVPDF(prefix, version)
			if e != nil {
				return e
			}
			response, e := knowledgeengine.InvokeSHVPDF(context.Background(), prefix, version, cwd, op, raw)
			if e != nil {
				return e
			}
			after, e := knowledgeengine.InspectSHVPDF(prefix, version)
			if e != nil {
				return e
			}
			if inst != after {
				return fmt.Errorf("PDF adapter changed")
			}
			if leaf == "verify" {
				if !refreshEqual(expected, response.Result) {
					return fmt.Errorf("PDF derivation differs from replay")
				}
				out, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-pdf-verification.v1", "artifact": response.Result, "installation": inst, "replayed": true})
				if e != nil {
					return e
				}
				return printIndentedJSON(out)
			}
			return printIndentedJSON(response)
		}}
		c.Flags().StringVar(&prefix, "prefix", "", "exact adapter installation")
		c.Flags().StringVar(&version, "version", "", "exact adapter version")
		c.Flags().Bool("json", false, "structured results/errors")
		_ = c.MarkFlagRequired("prefix")
		_ = c.MarkFlagRequired("version")
		if leaf == "extract" || leaf == "verify" {
			c.Flags().StringVar(&input, "input", "", "explicit original source and decoder identities")
			_ = c.MarkFlagRequired("input")
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "invoke"
		if leaf == "inspect" {
			interaction = "inspect"
		}
		if leaf == "schema" || leaf == "template" {
			interaction = "discover"
		}
		s := commandSpec("shv.pdf."+leaf, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-pdf-adapter", Interaction: interaction})
		protocol := "symphony.qxctl.shv-pdf-" + leaf + ".v1"
		if leaf == "extract" || leaf == "inspect" {
			protocol, _ = knowledgeengine.SHVPDFResultProtocol(leaf)
			s.InputProtocols = []string{knowledgeengine.SHVPDFInputProtocol(leaf)}
			s.BackendOperationIDs = []string{"engop:symphony:shv-pdf." + leaf}
		}
		if leaf == "verify" {
			protocol = "symphony.qxctl.shv-pdf-verification.v1"
			s.InputProtocols = []string{"symphony.qxctl.shv-pdf-verification-input.v1"}
			s.BackendOperationIDs = []string{"engop:symphony:shv-pdf.extract"}
		}
		s.OutputProtocols = []string{protocol}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	root.AddCommand(newSHVPDFGraphCommand())
	return root
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func newSHVProfileLeaf(family, leaf string) *cobra.Command {
	var prefix, version, input, selection string
	resource := leaf == "schema" || leaf == "template"
	op := family + "_" + leaf
	if leaf == "inspect" {
		op = "inspect"
	}
	c := &cobra.Command{Use: leaf, Short: "Check caller class declarations or bind a portable universe to exact local evidence", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		if resource {
			inst, raw, e := knowledgeengine.SHVProfileResource(prefix, version, leaf == "template")
			if e != nil {
				return e
			}
			m := map[string]any{"protocol": "symphony.qxctl.shv-profile-" + leaf + ".v1", "installation": inst}
			if leaf == "schema" {
				m["schema"] = raw
			} else {
				var templates map[string]json.RawMessage
				if e = json.Unmarshal(raw, &templates); e != nil {
					return e
				}
				v, ok := templates[selection]
				if !ok {
					return fmt.Errorf("unknown profile engine template")
				}
				m["template"] = v
				m["status"] = "unanswered_template_not_validated_input"
			}
			out, e := sealSHVActivation(m)
			if e != nil {
				return e
			}
			return printIndentedJSON(out)
		}
		payload := []byte(`{}`)
		var e error
		if op != "inspect" {
			payload, e = knowledgeengine.ReadPayload(input)
			if e != nil {
				return e
			}
		}
		cwd, e := os.Getwd()
		if e != nil {
			return e
		}
		r, e := knowledgeengine.InvokeSHVProfile(context.Background(), prefix, version, cwd, op, payload)
		if e != nil {
			return e
		}
		return printIndentedJSON(r)
	}}
	c.Flags().StringVar(&prefix, "prefix", "", "exact profile engine installation prefix")
	c.Flags().StringVar(&version, "version", "", "exact profile engine version; no default or upgrade")
	c.MarkFlagRequired("prefix")
	c.MarkFlagRequired("version")
	c.Flags().Bool("json", false, "emit independently validated structured JSON")
	if !resource && op != "inspect" {
		c.Flags().StringVar(&input, "input", "", "bounded no-follow operation payload JSON")
		c.MarkFlagRequired("input")
	}
	if leaf == "template" {
		c.Flags().StringVar(&selection, "operation", "", "profile_compile, mapping_diagnose, universe_build or universe_bind")
		c.MarkFlagRequired("operation")
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	interaction := "invoke"
	if resource {
		interaction = "discover"
	}
	if op == "inspect" {
		interaction = "inspect"
	}
	if op == "mapping_diagnose" {
		interaction = "query"
	}
	s := commandSpec("shv."+family+"."+leaf, featureSHVAdministration, interaction)
	s.Mutability = "read_only"
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-profile-engine", Interaction: interaction})
	out := "symphony.qxctl.shv-profile-" + leaf + ".v1"
	if !resource {
		s.BackendOperationIDs = []string{"engop:symphony:shv-profile." + strings.ReplaceAll(op, "_", ".")}
		s.InputProtocols = []string{knowledgeengine.SHVProfileInputProtocol(op)}
		out, _ = knowledgeengine.SHVProfileResultProtocol(op)
	}
	s.OutputProtocols = []string{out}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}

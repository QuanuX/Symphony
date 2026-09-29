package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

const featureSQVAdministration = "ssfv:symphony:qxctl.sqv-administration"

func newSQVCommand() *cobra.Command {
	root := structural("sqv", errUsageOnly)
	acquisition := structural("acquisition", errUsageOnly)
	acquisition.AddCommand(newSQVLeaf("validate"))
	groups := map[string]*cobra.Command{"acquisition": acquisition}
	for _, op := range knowledgeengine.SQVAdministrationOperations {
		group := groups[op.Domain]
		if group == nil {
			group = structural(op.Domain, errUsageOnly)
			groups[op.Domain] = group
			root.AddCommand(group)
		}
		group.AddCommand(newSQVAdministrationLeaf(op))
	}
	root.AddCommand(acquisition, newSQVLeaf("schema"), newSQVLeaf("template"))
	return root
}
func sqvSafeError(err error) error {
	var pe *knowledgeengine.ProcessError
	if errors.As(err, &pe) && safeSCVEngineCode(pe.Code) != nil {
		return &knowledgeengine.ProcessError{Code: pe.Code, Message: "SQV administration refused"}
	}
	return fmt.Errorf("SQV command refused; check the selected installation and request contract")
}
func newSQVLeaf(leaf string) *cobra.Command {
	var prefix, version, input, adapter, operation string
	resource := leaf != "validate"
	short := "Validate a native SQAV request"
	if resource {
		short = "Inspect an exact installed SQV operation contract"
	}
	c := &cobra.Command{Use: leaf, Short: short, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		if resource {
			var inst knowledgeengine.Installation
			var raw []byte
			var err error
			if operation == "request_validate" {
				if adapter == "" {
					return errUsageOnly
				}
				inst, raw, err = knowledgeengine.SQAVRequestResource(prefix, version, adapter, leaf == "template")
			} else {
				if adapter != "" {
					return errUsageOnly
				}
				inst, raw, err = knowledgeengine.SQVAdministrationResource(prefix, version, operation, leaf == "template")
			}
			if err != nil {
				return sqvSafeError(err)
			}
			out := map[string]any{"protocol": "symphony.qxctl.sqv-" + leaf + ".v1", "adapter": adapter, "operation": operation, "installation": inst, leaf: json.RawMessage(raw)}
			if leaf == "template" {
				out["status"] = "unanswered_template_not_validated_input"
			}
			return printIndentedJSON(out)
		}
		raw, err := knowledgeengine.ReadPayload(input)
		if err != nil {
			return sqvSafeError(err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return sqvSafeError(err)
		}
		r, err := knowledgeengine.InvokeSQAVRequest(context.Background(), prefix, version, cwd, raw)
		if err != nil {
			return sqvSafeError(err)
		}
		return printIndentedJSON(r)
	}}
	c.Flags().StringVar(&prefix, "prefix", "", "exact owner engine installation prefix")
	c.Flags().StringVar(&version, "version", "", "exact engine release; no implicit upgrade")
	_ = c.MarkFlagRequired("prefix")
	_ = c.MarkFlagRequired("version")
	c.Flags().Bool("json", false, "emit structured evidence and machine-readable failures")
	if resource {
		c.Flags().StringVar(&operation, "operation", "", "exact declared native operation, e.g. metadata_inspect")
		_ = c.MarkFlagRequired("operation")
		c.Flags().StringVar(&adapter, "adapter", "", "request variant: fred, databento_historical, databento_reference")
	} else {
		c.Flags().StringVar(&input, "input", "", "bounded no-follow request JSON file")
		_ = c.MarkFlagRequired("input")
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	key, interaction, out := "sqv."+leaf, "discover", "symphony.qxctl.sqv-"+leaf+".v1"
	if !resource {
		key = "sqv.acquisition.validate"
		interaction = "validate"
		out = "symphony.sqav.request-validation.v1"
	}
	s := commandSpec(key, featureSQVAdministration, interaction)
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:sqav-request-engine", Interaction: interaction})
	if resource {
		seen := map[string]bool{}
		for _, op := range knowledgeengine.SQVAdministrationOperations {
			if !seen[op.ModuleID] {
				s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + op.ModuleID, Interaction: "discover"})
				seen[op.ModuleID] = true
			}
		}
	}
	if !resource {
		s.BackendOperationIDs = []string{"engop:symphony:sqav-request.validate"}
		s.InputProtocols = []string{knowledgeengine.SQAVRequestInputProtocol}
	}
	s.OutputProtocols = []string{out}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}

func newSQVAdministrationLeaf(op knowledgeengine.SQVAdministrationOperation) *cobra.Command {
	var prefix, version, input string
	c := &cobra.Command{Use: op.Leaf, Short: "Run the exact native " + op.Operation + " contract", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		raw, err := knowledgeengine.ReadPayload(input)
		if err != nil {
			return sqvSafeError(err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return sqvSafeError(err)
		}
		result, err := knowledgeengine.InvokeSQVAdministration(context.Background(), prefix, version, cwd, op.Operation, raw)
		if err != nil {
			return sqvSafeError(err)
		}
		return printIndentedJSON(result)
	}}
	c.Flags().StringVar(&prefix, "prefix", "", "exact owner engine installation prefix")
	c.Flags().StringVar(&version, "version", "", "exact engine release; no implicit upgrade")
	c.Flags().StringVar(&input, "input", "", "bounded no-follow request JSON file")
	c.Flags().Bool("json", false, "emit structured evidence and machine-readable failures")
	for _, flag := range []string{"prefix", "version", "input"} {
		_ = c.MarkFlagRequired(flag)
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	interaction := "inspect"
	if op.Leaf == "validate" {
		interaction = "validate"
	}
	s := commandSpec("sqv."+op.Domain+"."+op.Leaf, featureSQVAdministration, interaction)
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + op.ModuleID, Interaction: interaction})
	s.BackendOperationIDs = []string{op.BackendID}
	s.InputProtocols = []string{op.InputProtocol}
	s.OutputProtocols = []string{op.OutputProtocol}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}

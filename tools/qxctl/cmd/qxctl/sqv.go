package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
	"os"
)

const featureSQVAdministration = "ssfv:symphony:qxctl.sqv-administration"

func newSQVCommand() *cobra.Command {
	root := structural("sqv", errUsageOnly)
	acquisition := structural("acquisition", errUsageOnly)
	acquisition.AddCommand(newSQVLeaf("validate"))
	root.AddCommand(acquisition, newSQVLeaf("schema"), newSQVLeaf("template"))
	return root
}
func sqvSafeError(err error) error {
	var pe *knowledgeengine.ProcessError
	if errors.As(err, &pe) && safeSCVEngineCode(pe.Code) != nil {
		return &knowledgeengine.ProcessError{Code: pe.Code, Message: "SQAV request validation refused"}
	}
	return fmt.Errorf("SQV command refused; check the selected installation and request contract")
}
func newSQVLeaf(leaf string) *cobra.Command {
	var prefix, version, input, adapter, operation string
	resource := leaf != "validate"
	c := &cobra.Command{Use: leaf, Short: "Validate native SQAV requests or discover their exact installed contract", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		if resource {
			if operation != "request_validate" {
				return errUsageOnly
			}
			inst, raw, err := knowledgeengine.SQAVRequestResource(prefix, version, adapter, leaf == "template")
			if err != nil {
				return sqvSafeError(err)
			}
			out := map[string]any{"protocol": "symphony.qxctl.sqv-" + leaf + ".v1", "adapter": adapter, "operation": operation, "installation": inst, leaf: raw}
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
	c.Flags().StringVar(&prefix, "prefix", "", "exact SQAV request engine installation prefix")
	c.Flags().StringVar(&version, "version", "", "exact engine release; no implicit upgrade")
	_ = c.MarkFlagRequired("prefix")
	_ = c.MarkFlagRequired("version")
	c.Flags().Bool("json", false, "emit structured evidence and machine-readable failures")
	if resource {
		c.Flags().StringVar(&operation, "operation", "", "exact native operation: request_validate")
		_ = c.MarkFlagRequired("operation")
		c.Flags().StringVar(&adapter, "adapter", "", "request variant: fred, databento_historical, databento_reference")
		_ = c.MarkFlagRequired("adapter")
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
	if !resource {
		s.BackendOperationIDs = []string{"engop:symphony:sqav-request.validate"}
		s.InputProtocols = []string{knowledgeengine.SQAVRequestInputProtocol}
	}
	s.OutputProtocols = []string{out}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}

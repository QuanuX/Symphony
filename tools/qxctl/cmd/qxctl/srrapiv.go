package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

const featureSRRAPIVAdministration = "ssfv:symphony:qxctl.srrapiv-administration"
const featureSRRAPIVRuntime = "ssfv:symphony:srrapiv-runtime-cpp"

func newSRRAPIVCommand() *cobra.Command {
	root := structural("rithmic", errUsageOnly)
	groups := map[string]*cobra.Command{}
	for _, op := range knowledgeengine.SRRAPIVOperations {
		group := groups[op.Domain]
		if group == nil {
			group = structural(op.Domain, errUsageOnly)
			groups[op.Domain] = group
			root.AddCommand(group)
		}
		group.AddCommand(newSRRAPIVLeaf(op))
	}
	root.AddCommand(newSRRAPIVResource("schema"), newSRRAPIVResource("template"))
	return root
}

func srrapivSafeError(err error) error {
	var pe *knowledgeengine.ProcessError
	if errors.As(err, &pe) && safeSCVEngineCode(pe.Code) != nil {
		return &knowledgeengine.ProcessError{Code: pe.Code, Message: "SRRAPIV command refused; inspect runtime status and attempts before retrying a mutation"}
	}
	return fmt.Errorf("SRRAPIV command refused; check the selected installation and request contract; inspect runtime status and attempts before retrying a mutation")
}

func srrapivCommonFlags(c *cobra.Command, prefix, version *string) {
	c.Flags().StringVar(prefix, "prefix", "", "exact native installation prefix")
	c.Flags().StringVar(version, "version", "", "exact engine release; no implicit upgrade")
	c.Flags().Bool("json", false, "emit structured evidence and machine-readable failures")
	_ = c.MarkFlagRequired("prefix")
	_ = c.MarkFlagRequired("version")
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
}

func newSRRAPIVLeaf(op knowledgeengine.SRRAPIVOperation) *cobra.Command {
	var prefix, version, input string
	c := &cobra.Command{Use: op.Leaf, Short: "Run the exact native SRRAPIV " + op.Operation + " contract", Args: usageOnlyArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			payload, err := knowledgeengine.ReadSRRAPIVPayload(input)
			if err != nil {
				return srrapivSafeError(err)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return srrapivSafeError(err)
			}
			result, err := knowledgeengine.InvokeSRRAPIV(c.Context(), prefix, version, cwd, op.Operation, payload)
			if err != nil {
				return srrapivSafeError(err)
			}
			if err = printIndentedJSON(result); err != nil {
				return err
			}
			// Preserve the native refusal evidence without a second JSON document.
			var envelope struct {
				Result struct {
					OK *bool `json:"ok"`
				} `json:"result"`
			}
			if json.Unmarshal(result.Result, &envelope) == nil && envelope.Result.OK != nil && !*envelope.Result.OK {
				return &exactEvidenceExitError{code: 2}
			}
			return nil
		}}
	srrapivCommonFlags(c, &prefix, &version)
	c.Flags().StringVar(&input, "input", "", "bounded no-follow request JSON file; credentials are never command arguments")
	_ = c.MarkFlagRequired("input")
	spec := commandSpec("sqv.rithmic."+op.Domain+"."+op.Leaf, featureSRRAPIVAdministration, op.Interaction)
	spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: featureSRRAPIVRuntime, Interaction: op.Interaction})
	spec.BackendOperationIDs = []string{"engop:symphony:srrapiv." + op.Domain + "." + op.Leaf}
	spec.InputProtocols = []string{knowledgeengine.SRRAPIVInputProtocol}
	spec.OutputProtocols = []string{knowledgeengine.SRRAPIVOutputProtocol}
	spec.ResultValidationProtocols = spec.OutputProtocols
	if op.Runtime {
		spec.AuthorityMode = "target_host_permission"
	}
	if op.Mutating {
		spec.Mutability = "permission_backed_mutation"
		recovery := "qxcmd:symphony:sqv.rithmic.runtime.status"
		if op.Operation == "runtime_invoke" {
			recovery = "qxcmd:symphony:sqv.rithmic.runtime.attempts"
		}
		spec.RecoveryCommandID = &recovery
	}
	commandregistry.Attach(c, spec)
	return c
}

func newSRRAPIVResource(leaf string) *cobra.Command {
	var prefix, version, operation string
	protocol := "symphony.qxctl.srrapiv-" + leaf + ".v1"
	c := &cobra.Command{Use: leaf, Short: "Inspect an exact installed SRRAPIV operation contract", Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			inst, resource, err := knowledgeengine.SRRAPIVResource(prefix, version, operation, leaf == "template")
			if err != nil {
				return srrapivSafeError(err)
			}
			out := map[string]any{"protocol": protocol, "operation": operation, "installation": inst, leaf: resource}
			if leaf == "template" {
				out["status"] = "unanswered_template_not_validated_input"
			}
			return printIndentedJSON(out)
		}}
	srrapivCommonFlags(c, &prefix, &version)
	c.Flags().StringVar(&operation, "operation", "", "exact native operation, e.g. runtime_invoke or binding_validate")
	_ = c.MarkFlagRequired("operation")
	spec := commandSpec("sqv.rithmic."+leaf, featureSRRAPIVAdministration, "discover")
	spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: featureSRRAPIVRuntime, Interaction: "discover"})
	spec.OutputProtocols = []string{protocol}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(c, spec)
	return c
}

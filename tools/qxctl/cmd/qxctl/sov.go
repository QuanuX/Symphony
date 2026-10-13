package main

import (
	"encoding/json"
	"fmt"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sovremote"
	"github.com/spf13/cobra"
)

const featureSOVRemotePlanning = "ssfv:symphony:qxctl-sov-remote-planning"

func newSOVCommand() *cobra.Command {
	root := structural("sov", errUsageOnly)
	remote := structural("remote", errUsageOnly)
	for _, op := range []string{"plan", "assess", "schema", "template"} {
		remote.AddCommand(newSOVRemoteLeaf(op))
	}
	root.AddCommand(remote)
	return root
}
func newSOVRemoteLeaf(op string) *cobra.Command {
	var input, operation, format string
	var machine bool
	resource := op == "schema" || op == "template"
	c := &cobra.Command{Use: op, Short: "Read-only SOV remote handoff " + op, Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		if format != "text" && format != "json" {
			return errUsageOnly
		}
		if machine {
			if c.Flags().Changed("format") && format != "json" {
				return errUsageOnly
			}
			format = "json"
		}
		var result any
		var err error
		if resource {
			if op == "schema" {
				result, err = sovremote.Schema(operation)
			} else {
				result, err = sovremote.Template(operation)
			}
			result = map[string]any{"protocol": "symphony.sov.remote-" + op + ".v1", "operation": operation, op: result}
		} else {
			var raw []byte
			raw, err = knowledgeengine.ReadPayload(input)
			if err == nil {
				if op == "plan" {
					result, err = sovremote.MakePlan(raw)
				} else {
					result, err = sovremote.Assess(raw)
				}
			}
		}
		if err != nil {
			return fmt.Errorf("SOV remote %s rejected the control document; no operation was dispatched", op)
		}
		raw, err := json.Marshal(result)
		if err != nil {
			return err
		}
		// Existing generic tree rendering preserves every field and escapes terminal
		// controls. The SBV-specific NDJSON protocol is intentionally not offered.
		return renderCompleteData(c.OutOrStdout(), raw, format, "SOV remote")
	}}
	c.Flags().StringVar(&format, "format", "text", "complete text or JSON output")
	c.Flags().BoolVar(&machine, "json", false, "complete portable JSON output")
	if resource {
		c.Flags().StringVar(&operation, "operation", "", "plan or assess")
		_ = c.MarkFlagRequired("operation")
	} else {
		c.Flags().StringVar(&input, "input", "", "local control JSON; no endpoint is contacted")
		_ = c.MarkFlagRequired("input")
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	interaction := "validate"
	if resource {
		interaction = "discover"
	}
	spec := commandSpec("sov.remote."+op, featureSOVRemotePlanning, interaction)
	if !resource {
		spec.InputProtocols = []string{"symphony.sov.remote-" + op + "-input.v1"}
	}
	spec.OutputProtocols = []string{"symphony.sov.remote-" + op + ".v1"}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(c, spec)
	return c
}

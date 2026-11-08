package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/transportprofile"
	w "github.com/QuanuX/Symphony/tools/qxctl/internal/transportprofile/wire"
	"github.com/spf13/cobra"
)

const featureTransportProfile = "ssfv:symphony:qxctl.sov-transport-profile-administration"

func newSOVTransport() *cobra.Command {
	root := structural("transport", errUsageOnly)
	profile := structural("profile", errUsageOnly)
	for _, operation := range w.Operations {
		profile.AddCommand(newSOVTransportProfileLeaf(operation))
	}
	root.AddCommand(profile)
	return root
}
func newSOVTransportProfileLeaf(operation string) *cobra.Command {
	var input, format string
	var machine bool
	c := &cobra.Command{Use: operation, Short: "Protected transport profile " + operation, Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		if machine {
			if c.Flags().Changed("format") && format != "json" {
				return errUsageOnly
			}
			format = "json"
		}
		if format != "text" && format != "json" {
			return errUsageOnly
		}
		var result any
		completed := true
		switch operation {
		case "schema":
			if e := json.Unmarshal([]byte(w.SchemaJSON), &result); e != nil {
				return e
			}
		case "template":
			result = w.Template()
		default:
			raw, e := knowledgeengine.ReadPayload(input)
			if e != nil {
				return fmt.Errorf("transport profile input unavailable")
			}
			value, e := transportprofile.Run(c.Context(), operation, raw)
			if e != nil {
				return e
			}
			result = value
			completed = value.Completed
		}
		encoded, e := json.Marshal(result)
		if e != nil {
			return e
		}
		if e := renderCompleteData(c.OutOrStdout(), encoded, format, "SOV transport profile"); e != nil {
			return e
		}
		if !completed {
			return &exactEvidenceExitError{code: 5}
		}
		return nil
	}}
	c.Flags().StringVar(&format, "format", "text", "complete text or JSON output")
	c.Flags().BoolVar(&machine, "json", false, "complete JSON output")
	if operation != "schema" && operation != "template" {
		c.Flags().StringVar(&input, "input", "", "closed profile request JSON")
		_ = c.MarkFlagRequired("input")
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	interaction := "inspect"
	if operation == "schema" || operation == "template" {
		interaction = "discover"
	}
	if operation == "plan" {
		interaction = "propose"
	}
	if operation == "apply" {
		interaction = "apply"
	}
	if operation == "recover" {
		interaction = "recover"
	}
	s := commandSpec("sov.transport.profile."+operation, featureTransportProfile, interaction)
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:sov-transport-profile-admin", Interaction: interaction})
	s.BackendOperationIDs = []string{"engop:symphony:sov.transport.profile." + operation}
	s.InputProtocols = []string{w.RequestProtocol}
	s.OutputProtocols = []string{w.ResultProtocol}
	if operation == "schema" {
		s.InputProtocols = []string{}
		s.OutputProtocols = []string{"symphony.sov.transport-profile.schemas.v1"}
	}
	if operation == "template" {
		s.InputProtocols = []string{}
		s.OutputProtocols = []string{w.RequestProtocol}
	}
	s.ResultValidationProtocols = s.OutputProtocols
	if operation == "plan" {
		s.Mutability = "proposal_only"
	}
	if operation == "apply" || operation == "recover" {
		s.Mutability = "permission_backed_mutation"
		s.AuthorityMode = "target_host_permission"
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:sov.transport.profile.recover")
	}
	commandregistry.Attach(c, s)
	return c
}

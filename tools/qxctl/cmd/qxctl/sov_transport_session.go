package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sovremote"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofile"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/transportsession"
	"github.com/spf13/cobra"
)

func newSOVTransportSession() *cobra.Command {
	root := structural("session", errUsageOnly)
	for _, op := range append([]string{"schema", "template"}, transportsession.Operations...) {
		root.AddCommand(newSOVTransportSessionLeaf(op))
	}
	return root
}
func newSOVTransportSessionLeaf(op string) *cobra.Command {
	var input, format string
	var machine bool
	c := &cobra.Command{Use: op, Short: "Explicit user-owned Cloudflare Access session " + op, Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		if machine {
			if c.Flags().Changed("format") && format != "json" {
				return errUsageOnly
			}
			format = "json"
		}
		if format != "json" && format != "text" {
			return errUsageOnly
		}
		var result any
		completed := true
		switch op {
		case "schema":
			result = transportsession.Schema()
		case "template":
			endpoint, err := sovremote.Template("ssh_endpoint_v2")
			if err != nil {
				return err
			}
			result = transportsession.Template(endpoint.(sshprofile.Endpoint))
		default:
			raw, err := knowledgeengine.ReadPayload(input)
			if err != nil {
				return fmt.Errorf("transport session input unavailable")
			}
			value, err := transportsession.Run(c.Context(), op, raw)
			if err != nil {
				return err
			}
			result = value
			completed = value.Completed
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			return err
		}
		if err = renderCompleteData(c.OutOrStdout(), encoded, format, "SOV transport session"); err != nil {
			return err
		}
		if !completed {
			return &exactEvidenceExitError{code: 5}
		}
		return nil
	}}
	if op != "schema" && op != "template" {
		c.Flags().StringVar(&input, "input", "", "closed nonsecret session request JSON")
		_ = c.MarkFlagRequired("input")
	}
	c.Flags().StringVar(&format, "format", "text", "complete text or JSON output")
	c.Flags().BoolVar(&machine, "json", false, "complete JSON output")
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	interaction := "inspect"
	if op == "schema" || op == "template" {
		interaction = "discover"
	}
	if op == "plan" {
		interaction = "propose"
	}
	if op == "authenticate" || op == "renew" || op == "clear" {
		interaction = "apply"
	}
	s := commandSpec("sov.transport.session."+op, featureCloudflare, interaction)
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:sov-cloudflare-adapter", Interaction: interaction})
	s.BackendOperationIDs = []string{"engop:symphony:sov.transport.session." + op}
	s.InputProtocols = []string{transportsession.RequestProtocol}
	s.OutputProtocols = []string{transportsession.ResultProtocol}
	s.ResultValidationProtocols = s.OutputProtocols
	s.AuthorityMode = "target_host_permission"
	if op == "plan" {
		s.Mutability = "proposal_only"
	} else if op == "authenticate" || op == "renew" || op == "clear" {
		s.Mutability = "permission_backed_mutation"
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:sov.transport.session.inspect")
	}
	if op == "schema" || op == "template" {
		s.BackendOperationIDs = []string{}
		s.InputProtocols = []string{}
		s.ResultValidationProtocols = []string{}
		if op == "schema" {
			s.OutputProtocols = []string{"symphony.sov.transport-session.schemas.v1"}
		} else {
			s.OutputProtocols = []string{transportsession.RequestProtocol}
		}
	}
	commandregistry.Attach(c, s)
	return c
}

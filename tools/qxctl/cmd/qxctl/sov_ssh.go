package main

import (
	"encoding/json"
	"fmt"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sovremote"
	"github.com/spf13/cobra"
)

const featureSOVSSH = "ssfv:symphony:qxctl-sov-ssh-carrier"

func newSOVSSH(op string) *cobra.Command {
	var input string
	var transportProfile string
	var machine bool
	c := &cobra.Command{Use: op, Short: "Select an exact optional SSH carrier and expose complete evidence", Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		raw, err := knowledgeengine.ReadPayload(input)
		if err != nil {
			return fmt.Errorf("invalid SSH control input")
		}
		var result any
		var failed bool
		if op == "ssh-endpoint" {
			if transportProfile != "" {
				result, err = sovremote.PlanSSHProfile(raw, transportProfile)
			} else {
				result, err = sovremote.InspectSSHEndpoint(raw)
			}
		} else {
			ctx, stop := sbvSignalContext(c.Context())
			defer stop()
			if sovremote.IsSSHProfile(raw) {
				r, e := sovremote.RunSSHProfile(ctx, raw)
				if r.Protocol == "" {
					return fmt.Errorf("SSH profile invocation rejected before dispatch")
				}
				failed = e != nil || r.TransportOutcome != "response_received" || r.TargetResult == nil || r.TargetResult.Status != "completed"
				result = r
			} else {
				var r sovremote.SSHInvocationResult
				r, err = sovremote.RunSSH(ctx, raw)
				if r.Protocol == "" {
					return fmt.Errorf("SSH invocation rejected before dispatch")
				}
				failed = err != nil || r.TransportOutcome != "response_received" || r.TargetResult == nil || r.TargetResult.Status != "completed"
				result = r
				err = nil
			}
		}
		if err != nil {
			return fmt.Errorf("SSH endpoint selection rejected; no operation dispatched")
		}
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		format := "text"
		if machine {
			format = "json"
		}
		if err = renderCompleteData(c.OutOrStdout(), b, format, "SOV SSH"); err != nil {
			return err
		}
		if failed {
			return &exactEvidenceExitError{code: 5}
		}
		return nil
	}}
	if op == "ssh-endpoint" {
		c.Flags().StringVar(&transportProfile, "transport-profile", "", "resolve a protected profile by name during endpoint planning")
	}
	c.Flags().StringVar(&input, "input", "", "complete caller-selected endpoint or invocation JSON")
	_ = c.MarkFlagRequired("input")
	c.Flags().BoolVar(&machine, "json", false, "complete portable JSON")
	s := commandSpec("sov.remote."+op, featureSOVSSH, "validate")
	s.InputProtocols = []string{"symphony.sov.ssh-endpoint.v1", "symphony.sov.ssh-profile-selection.v1"}
	s.OutputProtocols = []string{"symphony.sov.ssh-endpoint-result.v1", "symphony.sov.ssh-endpoint-result.v2"}
	if op == "ssh-invoke" {
		s.FeatureBindings[0].Interaction = "invoke"
		s.Mutability = "permission_backed_mutation"
		s.AuthorityMode = "target_host_permission"
		s.TargetScope = "target_host"
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:sov.remote.ssh-invoke")
		s.InputProtocols = []string{"symphony.sov.ssh-invoke-input.v1", "symphony.sov.ssh-invoke-input.v2"}
		s.OutputProtocols = []string{"symphony.sov.ssh-invoke.v1", "symphony.sov.ssh-invoke.v2"}
	}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}

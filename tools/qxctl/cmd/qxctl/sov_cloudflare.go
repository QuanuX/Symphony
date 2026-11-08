package main

import (
	"fmt"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sovcloudflare"
	"github.com/spf13/cobra"
)

const featureCloudflare = "ssfv:symphony:qxctl.sov-cloudflare-administration"

func newSOVCloudflare() *cobra.Command {
	root := structural("cloudflare", errUsageOnly)
	cf := structural("cf", errUsageOnly)
	root.AddCommand(cf)
	families := map[string]*cobra.Command{}
	for _, op := range sovcloudflare.Operations {
		parts := strings.Split(op, ".")
		family := families[parts[0]]
		if family == nil {
			family = structural(parts[0], errUsageOnly)
			families[parts[0]] = family
			cf.AddCommand(family)
		}
		family.AddCommand(newSOVCloudflareLeaf(op, parts[1]))
	}
	return root
}
func newSOVCloudflareLeaf(operation, leaf string) *cobra.Command {
	var prefix, version, input, format string
	var machine bool
	c := &cobra.Command{Use: leaf, Short: "Exact native Cloudflare " + operation, Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		if machine {
			if c.Flags().Changed("format") && format != "json" {
				return errUsageOnly
			}
			format = "json"
		}
		if format != "text" && format != "json" {
			return errUsageOnly
		}
		raw, err := knowledgeengine.ReadPayload(input)
		if err != nil {
			return fmt.Errorf("Cloudflare input unavailable")
		}
		result, completed, err := sovcloudflare.Run(c.Context(), prefix, version, operation, raw)
		if err != nil {
			return err
		}
		if err = renderCompleteData(c.OutOrStdout(), result, format, "SOV Cloudflare"); err != nil {
			return err
		}
		if !completed {
			return &exactEvidenceExitError{code: 5}
		}
		return nil
	}}
	c.Flags().StringVar(&prefix, "adapter-prefix", "", "exact native adapter installation prefix")
	c.Flags().StringVar(&version, "adapter-version", "", "exact adapter version; no implicit selection")
	c.Flags().StringVar(&input, "input", "", "closed operation input JSON")
	c.Flags().StringVar(&format, "format", "text", "complete text or JSON output")
	c.Flags().BoolVar(&machine, "json", false, "complete JSON output")
	for _, f := range []string{"adapter-prefix", "adapter-version", "input"} {
		_ = c.MarkFlagRequired(f)
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	interaction := "inspect"
	if strings.HasPrefix(operation, "capability.") {
		interaction = "discover"
	}
	if leaf == "plan" {
		interaction = "propose"
	}
	if leaf == "apply" {
		interaction = "apply"
	}
	if leaf == "reconcile" {
		interaction = "recover"
	}
	s := commandSpec("sov.cloudflare.cf."+operation, featureCloudflare, interaction)
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:sov-cloudflare-adapter", Interaction: interaction})
	s.BackendOperationIDs = []string{"engop:symphony:sov.cloudflare." + operation}
	s.InputProtocols = []string{sovcloudflare.RequestProtocol}
	s.OutputProtocols = []string{sovcloudflare.ResultProtocol}
	s.ResultValidationProtocols = s.OutputProtocols
	if leaf == "plan" {
		s.Mutability = "proposal_only"
	}
	if leaf == "apply" || leaf == "reconcile" {
		s.Mutability = "permission_backed_mutation"
		s.AuthorityMode = "target_host_permission"
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:sov.cloudflare.cf.installation.reconcile")
	}
	commandregistry.Attach(c, s)
	return c
}

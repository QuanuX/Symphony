package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

// The native owner defines interpretation and composition. These leaves expose
// exact installed operations without choosing requirements or executing recipes.
func scvExtensionLeaf(group, leaf, operation, interaction, mutability string) *cobra.Command {
	options := scvOptions{}
	command := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		return runSCV(operation, options)
	}}
	scvFlagsVersion(command, &options, "0.6.0-dev")
	attachSCV(command, "scv."+group+"."+leaf, operation, interaction, mutability, false)
	return command
}

func newSCVProviderPackCommand() *cobra.Command {
	group := structural("pack", fmt.Errorf("provider pack subcommand is required: prepare, evaluate"))
	group.AddCommand(scvExtensionLeaf("provider.pack", "prepare", "provider_pack_prepare", "propose", "proposal_only"))
	group.AddCommand(scvExtensionLeaf("provider.pack", "evaluate", "provider_pack_evaluate", "query", "read_only"))
	return group
}

func newSCVCompositionCommand() *cobra.Command {
	group := structural("composition", fmt.Errorf("composition subcommand is required: explore, reassess, workflow, obligations, bundle"))
	group.AddCommand(scvExtensionLeaf("composition", "explore", "composition_explore", "query", "read_only"))
	group.AddCommand(scvExtensionLeaf("composition", "reassess", "composition_reassess", "validate", "evidence_only"))
	group.AddCommand(newSCVCompositionWorkflowCommand())
	group.AddCommand(newSCVObligationsCommand())
	group.AddCommand(newSCVBundleCommand())
	return group
}

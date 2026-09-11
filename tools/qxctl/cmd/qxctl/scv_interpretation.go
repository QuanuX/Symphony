package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

func scvInterpretationLeaf(group, leaf, operation, interaction string) *cobra.Command {
	options := scvOptions{}
	command := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		return runSCV(operation, options)
	}}
	scvFlagsVersion(command, &options, "0.3.0-dev")
	attachSCV(command, "scv."+group+"."+leaf, operation, interaction, "evidence_only", false)
	return command
}

func newSCVProviderCommand() *cobra.Command {
	group := structural("provider", fmt.Errorf("provider subcommand is required: interpret, coverage, pack"))
	group.AddCommand(scvInterpretationLeaf("provider", "interpret", "provider_interpret", "invoke"))
	group.AddCommand(newSCVProviderCoverageCommand())
	group.AddCommand(newSCVProviderPackCommand())
	return group
}

func newSCVConnectionCommand() *cobra.Command {
	group := structural("connection", fmt.Errorf("connection subcommand is required: evaluate, reassess"))
	group.AddCommand(scvInterpretationLeaf("connection", "evaluate", "connection_evaluate", "validate"))
	group.AddCommand(scvInterpretationLeaf("connection", "reassess", "connection_reassess", "validate"))
	return group
}

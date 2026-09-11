package main

import "github.com/spf13/cobra"

func newSCVProviderCoverageCommand() *cobra.Command {
	options := scvOptions{}
	command := &cobra.Command{Use: "coverage", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCV("provider_coverage", options) }}
	scvFlagsVersion(command, &options, "0.5.0-dev")
	attachSCV(command, "scv.provider.coverage", "provider_coverage", "query", "read_only", false)
	return command
}

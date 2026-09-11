package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

func newSCVProfileCommand() *cobra.Command {
	group := structural("profile", fmt.Errorf("profile subcommand is required: prepare"))
	options := scvOptions{}
	prepare := &cobra.Command{Use: "prepare", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		return runSCV("profile_prepare", options)
	}}
	scvFlagsVersion(prepare, &options, "0.4.0-dev")
	attachSCV(prepare, "scv.profile.prepare", "profile_prepare", "propose", "proposal_only", false)
	group.AddCommand(prepare)
	return group
}

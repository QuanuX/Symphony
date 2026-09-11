package main

import (
	"fmt"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

func newSCVInterfaceCommand() *cobra.Command {
	group := structural("interface", fmt.Errorf("interface subcommand is required: show"))
	options := scvOptions{}
	command := &cobra.Command{Use: "show", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		result, err := knowledgeengine.SCVOwnerInterface(options.domain, options.prefix, options.version)
		if err != nil {
			return err
		}
		return printIndentedJSON(result)
	}}
	command.Flags().StringVar(&options.domain, "domain", "scv", "exact installed SCV domain")
	command.Flags().StringVar(&options.prefix, "prefix", "", "exact receipt-v2 installation prefix")
	command.Flags().StringVar(&options.version, "version", "0.6.0-dev", "exact installed package version")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit validated JSON")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	spec := commandSpec("scv.interface.show", featureSCVAdministration, "discover")
	spec.Mutability = "read_only"
	spec.OutputProtocols = []string{"symphony.qxctl.scv-owner-interface.v1"}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(command, spec)
	group.AddCommand(command)
	return group
}

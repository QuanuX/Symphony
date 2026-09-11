package main

import (
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

func newSCVSchemaCommand() *cobra.Command {
	group := structural("schema", fmt.Errorf("schema subcommand is required: list, show, template"))
	for _, action := range []string{"list", "show", "template"} {
		options := scvOptions{}
		var protocol string
		command := &cobra.Command{Use: action, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
			result, err := knowledgeengine.SCVSchemaDiscovery(options.domain, options.prefix, options.version, action, protocol)
			if err != nil {
				return err
			}
			return printIndentedJSON(result)
		}}
		command.Flags().StringVar(&options.domain, "domain", "scv", "exact installed SCV domain")
		command.Flags().StringVar(&options.prefix, "prefix", "", "exact receipt-v2 installation prefix")
		command.Flags().StringVar(&options.version, "version", "0.4.0-dev", "exact installed package version")
		command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit validated JSON")
		if action != "list" {
			command.Flags().StringVar(&protocol, "protocol", "", "exact protocol ID from the selected packaged schema catalog")
		}
		command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		spec := commandSpec("scv.schema."+action, featureSCVAdministration, "discover")
		spec.Mutability = "read_only"
		spec.OutputProtocols = []string{"symphony.qxctl.scv-schema-" + action + ".v1"}
		spec.ResultValidationProtocols = spec.OutputProtocols
		commandregistry.Attach(command, spec)
		group.AddCommand(command)
	}
	return group
}

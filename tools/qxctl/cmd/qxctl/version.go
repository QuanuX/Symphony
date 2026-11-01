package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "version", Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			if !jsonOutput {
				fmt.Printf("qxctl version %s\n", version.Version)
				return nil
			}
			executable, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolve qxctl executable: %w", err)
			}
			digest, err := commandregistry.DigestExecutable(executable)
			if err != nil {
				return err
			}
			info, err := version.Current(strings.TrimPrefix(digest, "sha256:"))
			if err != nil {
				return err
			}
			return printIndentedJSON(info)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit source stamps, runtime target, and executable digest")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	spec := commandSpec("version", featureQXCTL, "inspect")
	spec.OutputProtocols = []string{version.BuildInfoProtocol}
	commandregistry.Attach(command, spec)
	return command
}

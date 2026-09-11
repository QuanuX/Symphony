package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/spf13/cobra"
)

func printCommandHelp(command *cobra.Command) {
	writeCommandHelp(os.Stdout, command)
}

// Help is a human projection of the same registered Cobra tree used by the
// machine manifest. There is no separately maintained list of leaf commands.
func writeCommandHelp(output io.Writer, command *cobra.Command) {
	command.InitDefaultHelpFlag()
	command.InitDefaultVersionFlag()
	fmt.Fprintln(output, "qxctl - Symphony administrative spine")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Usage:")
	if len(command.Commands()) != 0 {
		fmt.Fprintf(output, "  %s <command>\n", command.CommandPath())
	} else {
		fmt.Fprintf(output, "  %s\n", command.UseLine())
	}
	if command.Long != "" {
		fmt.Fprintf(output, "\n%s\n", command.Long)
	} else if command.Short != "" {
		fmt.Fprintf(output, "\n%s\n", command.Short)
	}
	var leaves []*cobra.Command
	var visit func(*cobra.Command)
	visit = func(current *cobra.Command) {
		if current.Hidden {
			return
		}
		if _, err := commandregistry.Spec(current); err == nil {
			current.InitDefaultHelpFlag()
			leaves = append(leaves, current)
		}
		for _, child := range current.Commands() {
			visit(child)
		}
	}
	for _, child := range command.Commands() {
		visit(child)
	}
	sort.Slice(leaves, func(i, j int) bool { return leaves[i].CommandPath() < leaves[j].CommandPath() })
	if len(leaves) != 0 {
		fmt.Fprintln(output, "\nCommands:")
		for _, leaf := range leaves {
			fmt.Fprintf(output, "  %s", leaf.UseLine())
			if leaf.Short != "" {
				fmt.Fprintf(output, "  %s", leaf.Short)
			}
			fmt.Fprintln(output)
		}
	}
	if flags := command.Flags().FlagUsages(); flags != "" {
		fmt.Fprintf(output, "\nFlags:\n%s", flags)
	}
	if len(leaves) != 0 {
		fmt.Fprintln(output, "\nUse '<command> --help' for exact flags and defaults. Use 'qxctl commands manifest --json' for machine discovery.")
	}
}

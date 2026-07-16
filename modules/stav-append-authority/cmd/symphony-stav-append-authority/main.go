package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/lifecycle"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "symphony-stav-append-authority: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return fmt.Errorf("command is required")
	}
	switch args[0] {
	case "--help", "help":
		printUsage()
		return nil
	case "--version":
		fmt.Printf("symphony-stav-append-authority version %s\n", version.Version)
		return nil
	case "install", "uninstall":
		return runLifecycle(args[0], args[1:])
	default:
		return fmt.Errorf("unknown command %q; operational listener and writer commands are not enabled", args[0])
	}
}

func runLifecycle(command string, args []string) error {
	set := flag.NewFlagSet(command, flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	force := set.Bool("force", false, "replace or remove a differing installed binary")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	scope, err := stavpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}

	var result lifecycle.Result
	switch command {
	case "install":
		result, err = lifecycle.Install(executable, scope, *force)
	case "uninstall":
		result, err = lifecycle.Uninstall(executable, scope, *force)
	}
	if err != nil {
		return err
	}
	if result.Changed {
		fmt.Printf("%s: %s scope=%s binary=%s\n", command, lifecycleVerb(command), result.Scope, result.Binary)
	} else {
		fmt.Printf("%s: no change scope=%s binary=%s\n", command, result.Scope, result.Binary)
	}
	return nil
}

func lifecycleVerb(command string) string {
	if command == "install" {
		return "installed"
	}
	return "uninstalled"
}

func printUsage() {
	fmt.Println("symphony-stav-append-authority - STAV namespace and lifecycle scaffold")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  symphony-stav-append-authority --help")
	fmt.Println("  symphony-stav-append-authority --version")
	fmt.Println("  symphony-stav-append-authority install [--scope user|system] [--force]")
	fmt.Println("  symphony-stav-append-authority uninstall [--scope user|system] [--force]")
	fmt.Println()
	fmt.Println("Operational listener and ledger writer commands are intentionally disabled.")
}

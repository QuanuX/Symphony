package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/contracts"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "--help":
		printHelp()
	case "--version":
		fmt.Println(version.Version)
	case "doctor":
		if err := runDoctor(); err != nil {
			fmt.Fprintf(os.Stderr, "doctor failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("doctor checks passed")
	case "contracts":
		if err := runContracts(); err != nil {
			fmt.Fprintf(os.Stderr, "contracts failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("qxctl - Symphony administrative spine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  qxctl <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  --help     Print concise usage")
	fmt.Println("  --version  Print version")
	fmt.Println("  doctor     Perform local repository/admin-spine checks")
	fmt.Println("  contracts  Verify first runtime-set module contract surfaces")
}

func runContracts() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := contracts.Verify(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("contracts: checks failed")
		return err
	}
	return nil
}

func runDoctor() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	fmt.Printf("found repository root: %s\n", repoRoot)

	modules := []string{
		"modules/node-troll",
		"modules/bus-troll",
		"modules/hotpath-runtime",
	}

	for _, mod := range modules {
		path := filepath.Join(repoRoot, mod)
		if !repository.IsDir(path) {
			return fmt.Errorf("missing required module directory: %s", mod)
		}
		fmt.Printf("verified module exists: %s\n", mod)
	}

	validatorPath := filepath.Join(repoRoot, "tools/symphony-validator")
	if !repository.IsDir(validatorPath) {
		return fmt.Errorf("missing required validator directory: tools/symphony-validator")
	}
	fmt.Printf("verified validator exists: tools/symphony-validator\n")

	return nil
}

package main

import (
	"fmt"
	"os"
	"path/filepath"

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
}

func runDoctor() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := findRepoRoot(cwd)
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
		if !isDir(path) {
			return fmt.Errorf("missing required module directory: %s", mod)
		}
		fmt.Printf("verified module exists: %s\n", mod)
	}

	validatorPath := filepath.Join(repoRoot, "tools/symphony-validator")
	if !isDir(validatorPath) {
		return fmt.Errorf("missing required validator directory: tools/symphony-validator")
	}
	fmt.Printf("verified validator exists: tools/symphony-validator\n")

	return nil
}

func findRepoRoot(start string) (string, error) {
	current := start
	for {
		hasReadme := isFile(filepath.Join(current, "README.md"))
		hasIntent := isFile(filepath.Join(current, "INTENT.md"))

		if hasReadme && hasIntent {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("README.md and INTENT.md not found in any parent directory")
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

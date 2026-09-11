package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/spf13/cobra"
)

func TestCLIHelpCoversEveryPublicRegisteredLeaf(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	writeCommandHelp(&rendered, root)
	lines := strings.Split(rendered.String(), "\n")
	for _, entry := range manifest.Commands {
		if entry.Grammar == nil {
			continue
		}
		grammar := strings.SplitN(*entry.Grammar, " [--", 2)[0]
		found := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == grammar || strings.HasPrefix(line, grammar+" [flags]") || strings.HasPrefix(line, grammar+"  ") {
				found = true
				break
			}
		}
		if found != (entry.Visibility == "public") {
			t.Errorf("help visibility mismatch for %s: found %t, visibility %s", entry.CommandID, found, entry.Visibility)
		}
	}
	// Add an actual registered leaf without editing any help data. This catches
	// a fallback to another manually synchronized command inventory.
	scv, _, err := root.Find([]string{"scv"})
	if err != nil {
		t.Fatal(err)
	}
	future := &cobra.Command{Use: "future-test-leaf", RunE: func(*cobra.Command, []string) error { return nil }}
	registered(future, "scv.future-test-leaf", featureSCVAdministration, "inspect")
	scv.AddCommand(future)
	rendered.Reset()
	writeCommandHelp(&rendered, root)
	if !strings.Contains(rendered.String(), "  qxctl scv future-test-leaf [flags]\n") {
		t.Fatal("new registered leaf did not appear automatically")
	}
}

func TestCLIHelpShowsScopedCommandsAndActualFlagDefaults(t *testing.T) {
	output, status := invokeCLI(t, "scv", "--help")
	if status != 0 || !strings.Contains(output, "qxctl scv provider interpret") || !strings.Contains(output, "qxctl scv corpus acquire") || strings.Contains(output, "qxctl ssiag") {
		t.Fatalf("SCV help is incomplete or not scoped: %d\n%s", status, output)
	}
	output, status = invokeCLI(t, "scv", "provider", "interpret", "--help")
	if status != 0 || !strings.Contains(output, "--input string") || !strings.Contains(output, `default "0.3.0-dev"`) || strings.Contains(output, "qxctl scv corpus acquire") {
		t.Fatalf("leaf help does not show exact flags/defaults: %d\n%s", status, output)
	}
}

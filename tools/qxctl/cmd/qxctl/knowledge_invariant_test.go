package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
)

func TestKnowledgeInvariantCommandsAreStableReadOnlyAgentSurfaces(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"qxcmd:symphony:knowledge.invariant.status": false,
		"qxcmd:symphony:knowledge.invariant.list":   false,
		"qxcmd:symphony:knowledge.invariant.show":   false,
		"qxcmd:symphony:knowledge.invariant.check":  false,
	}
	for _, command := range manifest.Commands {
		if _, exists := want[command.CommandID]; !exists {
			continue
		}
		want[command.CommandID] = true
		if command.Mutability != "read_only" || command.AuthorityMode != "none" ||
			!command.Noninteractive || command.TargetScope != "local" {
			t.Errorf("invariant command posture = %#v", command)
		}
	}
	for commandID, found := range want {
		if !found {
			t.Errorf("stable command ID %s is absent", commandID)
		}
	}
}

func TestKnowledgeInvariantStatusListAndShowReadCanonicalRegistry(t *testing.T) {
	repository, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"knowledge", "invariant", "status", "--repo", repository, "--json"},
		{"knowledge", "invariant", "list", "--repo", repository, "--json"},
		{"knowledge", "invariant", "show", "--repo", repository, "--invariant-id", "invariant:symphony:foundation.audit-closure", "--json"},
	} {
		if err := executeCommand(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if err := executeCommand([]string{"knowledge", "invariant", "show", "--repo", repository}); err == nil {
		t.Fatal("show without invariant identity unexpectedly succeeded")
	}
}

func TestKnowledgeInvariantCheckPreservesValidatedExitCode(t *testing.T) {
	prior := runInvariantValidator
	t.Cleanup(func() { runInvariantValidator = prior })
	runInvariantValidator = func(context.Context, string, string, string) (validation.Result, error) {
		return validation.Result{
			Protocol: validation.ResultProtocol, FormatVersion: 1,
			Evidence: validation.Evidence{
				ValidatorID: "symphony-validator", ValidatorVersion: "0.1.0-dev",
				Outcome: "violation", ExitCode: 26,
			},
		}, nil
	}
	err := runKnowledgeInvariant("check", knowledgeInvariantOptions{prefix: "/exact", version: "0.1.0-dev"})
	var exit *exactEvidenceExitError
	if !errors.As(err, &exit) || exit.code != 26 {
		t.Fatalf("check exit error = %v", err)
	}
	if status := execute([]string{"knowledge", "invariant", "check", "--prefix", "/exact"}); status != 26 {
		t.Fatalf("CLI exit = %d, want 26", status)
	}
}

func TestKnowledgeInvariantCheckRequiresExactInstalledValidator(t *testing.T) {
	if err := executeCommand([]string{"knowledge", "invariant", "check"}); err == nil || err.Error() != "--prefix is required" {
		t.Fatalf("missing-prefix error = %v", err)
	}
}

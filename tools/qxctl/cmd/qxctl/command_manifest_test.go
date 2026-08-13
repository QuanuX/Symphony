package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
)

const commandManifestTestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCommandRegistryCobraParityAndStableIdentity(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.Build(root, commandregistry.Identity{
		ClientVersion: "test", ExecutableDigest: commandManifestTestDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Commands) != 114 {
		t.Fatalf("registered command count = %d, want 114", len(manifest.Commands))
	}
	seen := make(map[string]*string, len(manifest.Commands))
	for _, command := range manifest.Commands {
		if prior, exists := seen[command.CommandID]; exists {
			t.Fatalf("duplicate command ID %s for %v and %v", command.CommandID, prior, command.Grammar)
		}
		seen[command.CommandID] = command.Grammar
		if !command.Noninteractive {
			t.Fatalf("command is not noninteractive = %#v", command)
		}
		if command.CommandID != "qxcmd:symphony:stav.append" && command.Visibility != "public" {
			t.Fatalf("unexpected hidden command posture = %#v", command)
		}
	}
	for _, required := range []string{
		"qxcmd:symphony:commands.manifest",
		"qxcmd:symphony:inventory",
		"qxcmd:symphony:modules",
		"qxcmd:symphony:ssfv.check",
		"qxcmd:symphony:ssfv.administration-check",
		"qxcmd:symphony:knowledge.lifecycle.apply",
	} {
		if _, ok := seen[required]; !ok {
			t.Errorf("required stable command ID %q is absent", required)
		}
	}
	if grammar, ok := seen["qxcmd:symphony:stav.append"]; !ok || grammar == nil || *grammar != "qxctl stav append" {
		t.Fatalf("prohibited STAV append identity/grammar = %v, present=%v", grammar, ok)
	}
}

func TestCommandRegistryBindsVectorCapabilitiesNotBindingSelection(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"qxcmd:symphony:skvi.inspect":              featureSKVI,
		"qxcmd:symphony:skvi.check":                featureSKVIAssurance,
		"qxcmd:symphony:skvi.propose":              featureSKVIProposal,
		"qxcmd:symphony:skvi.project":              featureSKVIProjection,
		"qxcmd:symphony:sclv.inspect":              featureSCLV,
		"qxcmd:symphony:sclv.check":                featureSCLVAssurance,
		"qxcmd:symphony:sclv.propose":              featureSCLVProposal,
		"qxcmd:symphony:sclv.recover":              featureSCLVRecovery,
		"qxcmd:symphony:sclv.project":              featureSCLVProjection,
		"qxcmd:symphony:sacv.inspect":              featureSACV,
		"qxcmd:symphony:sacv.check":                featureSACVConformance,
		"qxcmd:symphony:sacv.diff":                 featureSACVCompatibility,
		"qxcmd:symphony:sacv.propose":              featureSACVProposal,
		"qxcmd:symphony:sacv.project":              featureSACVProjection,
		"qxcmd:symphony:sodv.inspect":              featureSODV,
		"qxcmd:symphony:sodv.check":                featureSODVLedger,
		"qxcmd:symphony:sodv.verify":               featureSODVVerification,
		"qxcmd:symphony:sodv.propose":              featureSODVProposal,
		"qxcmd:symphony:sodv.recover":              featureSODVRecovery,
		"qxcmd:symphony:sodv.project":              featureSODVProjection,
		"qxcmd:symphony:ssfv.inspect":              featureSSFV,
		"qxcmd:symphony:ssfv.check":                featureSSFVSnapshot,
		"qxcmd:symphony:ssfv.diff":                 featureSSFVComparison,
		"qxcmd:symphony:ssfv.propose":              featureSSFVProposal,
		"qxcmd:symphony:ssfv.graph":                featureSSFVProjection,
		"qxcmd:symphony:ssfv.administration-check": featureAdministrationAssurance,
	}
	for _, command := range manifest.Commands {
		featureID, exists := want[command.CommandID]
		if !exists {
			continue
		}
		if len(command.FeatureBindings) != 1 || command.FeatureBindings[0].FeatureID != featureID {
			t.Errorf("%s bindings = %#v, want feature %s", command.CommandID, command.FeatureBindings, featureID)
		}
		delete(want, command.CommandID)
	}
	for commandID := range want {
		t.Errorf("expected vector command %s is absent", commandID)
	}
}

func TestReadExpectedRegistryRejectsTrailingValueAndSymlink(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := commandregistry.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	valid := filepath.Join(directory, "valid.json")
	if err := os.WriteFile(valid, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readExpectedRegistry(valid); err != nil {
		t.Fatalf("valid expected registry rejected: %v", err)
	}
	trailing := filepath.Join(directory, "trailing.json")
	if err := os.WriteFile(trailing, append(encoded, []byte("\n{}")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readExpectedRegistry(trailing); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("trailing JSON value accepted: %v", err)
	}
	link := filepath.Join(directory, "link.json")
	if err := os.Symlink(valid, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readExpectedRegistry(link); err == nil {
		t.Fatal("symlink expected registry input was accepted")
	}
}

func TestObservedRegistryRecordsExistingNonJSONDebt(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.Build(root, commandregistry.Identity{
		ClientVersion: "test", ExecutableDigest: commandManifestTestDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	nonJSON := make([]string, 0)
	for _, command := range manifest.Commands {
		if !command.JSONOutput {
			nonJSON = append(nonJSON, command.CommandID)
		}
	}
	got := strings.Join(nonJSON, ",")
	want := strings.Join([]string{
		"qxcmd:symphony:contracts",
		"qxcmd:symphony:doctor",
		"qxcmd:symphony:module.check",
		"qxcmd:symphony:module.inspect",
		"qxcmd:symphony:modules",
		"qxcmd:symphony:modules.check",
		"qxcmd:symphony:ssiag.doctor",
		"qxcmd:symphony:stav.append",
		"qxcmd:symphony:stav.doctor",
	}, ",")
	if got != want {
		t.Fatalf("non-JSON observed commands = %s, want %s", got, want)
	}
}

func TestCheckedInExpectedRegistryMatchesCommandTree(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	want, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readExpectedRegistry("../../COMMANDS.json")
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := commandregistry.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	gotBytes, err := commandregistry.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Fatalf("COMMANDS.json is stale; regenerate with commands expected --json")
	}
}

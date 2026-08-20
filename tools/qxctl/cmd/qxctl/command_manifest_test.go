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
	if len(manifest.Commands) != 185 {
		t.Fatalf("registered command count = %d, want 185", len(manifest.Commands))
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
		"qxcmd:symphony:knowledge.invariant.status",
		"qxcmd:symphony:knowledge.invariant.list",
		"qxcmd:symphony:knowledge.invariant.show",
		"qxcmd:symphony:knowledge.invariant.check",
		"qxcmd:symphony:ssiag.enrollment.apply",
		"qxcmd:symphony:ssiag.provider.show",
		"qxcmd:symphony:ssiag.provider.verify",
		"qxcmd:symphony:ssiag.provider.installations",
		"qxcmd:symphony:ssiag.provider.binding.status",
		"qxcmd:symphony:ssiag.provider.binding.plan",
		"qxcmd:symphony:ssiag.provider.binding.apply",
		"qxcmd:symphony:ssiag.provider.binding.apply-status",
		"qxcmd:symphony:ssiag.provider.binding.recover",
		"qxcmd:symphony:ssiag.supervisor.recover",
		"qxcmd:symphony:stav.enrollment.plan",
		"qxcmd:symphony:stav.supervisor.status",
		"qxcmd:symphony:validate.warning.sync",
		"qxcmd:symphony:validate.warning.accept",
		"qxcmd:symphony:validate.warning.show",
		"qxcmd:symphony:validate.root-summary",
		"qxcmd:symphony:sav.current",
		"qxcmd:symphony:sav.evaluate",
		"qxcmd:symphony:sev.command-surface",
		"qxcmd:symphony:sev.verify",
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
		"qxcmd:symphony:sav.inspect":               featureSAV,
		"qxcmd:symphony:sav.reference-check":       featureSAV,
		"qxcmd:symphony:sav.current":               featureSAV,
		"qxcmd:symphony:sav.evaluate":              featureSAV,
		"qxcmd:symphony:sav.diff":                  featureSAV,
		"qxcmd:symphony:sav.explain":               featureSAV,
		"qxcmd:symphony:sav.graph":                 featureSAV,
		"qxcmd:symphony:sav.version-validate":      featureSAV,
		"qxcmd:symphony:sav.version-diff":          featureSAV,
		"qxcmd:symphony:sav.capsule-check":         featureSAV,
		"qxcmd:symphony:sav.blueprint-plan":        featureSAV,
		"qxcmd:symphony:sav.compatibility":         featureSAV,
		"qxcmd:symphony:sev.inspect":               featureSEV,
		"qxcmd:symphony:sev.case-open":             featureSEV,
		"qxcmd:symphony:sev.impact":                featureSEV,
		"qxcmd:symphony:sev.plan":                  featureSEV,
		"qxcmd:symphony:sev.verify":                featureSEV,
		"qxcmd:symphony:sev.recalculate":           featureSEV,
		"qxcmd:symphony:sev.status":                featureSEV,
		"qxcmd:symphony:sev.recover":               featureSEV,
		"qxcmd:symphony:sev.close":                 featureSEV,
		"qxcmd:symphony:sev.command-surface":       featureSEV,
		"qxcmd:symphony:sev.novelty-check":         featureSEV,
		"qxcmd:symphony:sev.watch-check":           featureSEV,
		"qxcmd:symphony:sev.trigger-coalesce":      featureSEV,
		"qxcmd:symphony:sev.graph":                 featureSEV,
		"qxcmd:symphony:sev.compatibility":         featureSEV,
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

func TestReviewedBackendFeatureBindingsReachExpectedRegistry(t *testing.T) {
	if len(reviewedBackendFeatureBindings) != 75 {
		t.Fatalf("reviewed backend command count = %d, want 75", len(reviewedBackendFeatureBindings))
	}
	secondaryBindingCount := 0
	for key, bindings := range reviewedBackendFeatureBindings {
		if key == "" || len(bindings) == 0 {
			t.Fatalf("empty reviewed backend binding entry for %q", key)
		}
		secondaryBindingCount += len(bindings)
	}
	if secondaryBindingCount != 78 {
		t.Fatalf("reviewed backend binding count = %d, want 78", secondaryBindingCount)
	}

	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, command := range manifest.Commands {
		key := strings.TrimPrefix(command.CommandID, "qxcmd:symphony:")
		bindings, expected := reviewedBackendFeatureBindings[key]
		if !expected {
			continue
		}
		found++
		if len(command.FeatureBindings) != len(bindings)+1 {
			t.Errorf("%s bindings = %#v, want one wrapper plus %#v", command.CommandID, command.FeatureBindings, bindings)
			continue
		}
		for _, binding := range bindings {
			if !containsFeatureBinding(command.FeatureBindings, binding) {
				t.Errorf("%s is missing reviewed backend binding %#v", command.CommandID, binding)
			}
		}
	}
	if found != len(reviewedBackendFeatureBindings) {
		t.Fatalf("registered reviewed backend command count = %d, want %d", found, len(reviewedBackendFeatureBindings))
	}
}

func containsFeatureBinding(bindings []commandregistry.FeatureBinding, want commandregistry.FeatureBinding) bool {
	for _, binding := range bindings {
		if binding == want {
			return true
		}
	}
	return false
}

func TestValidationLifecycleRegistryContracts(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	records := make(map[string]commandregistry.CommandRecord)
	for _, record := range manifest.Commands {
		records[record.CommandID] = record
	}
	rootSummary := records["qxcmd:symphony:validate.root-summary"]
	if !containsFeatureBinding(rootSummary.FeatureBindings, commandregistry.FeatureBinding{
		FeatureID: backendFeatureRootSummary, Interaction: "inspect",
	}) || len(rootSummary.OutputProtocols) != 1 || rootSummary.OutputProtocols[0] != "symphony.repository.root-summary.v1" ||
		len(rootSummary.ResultValidationProtocols) != 1 || rootSummary.ResultValidationProtocols[0] != "symphony.repository.root-summary.v1" ||
		rootSummary.Mutability != "read_only" {
		t.Fatalf("root-summary command contract drifted: %#v", rootSummary)
	}
	for _, operation := range []string{"status", "list", "show", "sync", "accept", "reopen", "supersede", "mute", "unmute"} {
		record := records["qxcmd:symphony:validate.warning."+operation]
		exactState := operation != "status" && operation != "list"
		if exactState && (len(record.OutputProtocols) != 1 || record.OutputProtocols[0] != "symphony.validation.warning-state.v1" ||
			len(record.ResultValidationProtocols) != 1 || record.ResultValidationProtocols[0] != "symphony.validation.warning-state.v1") {
			t.Errorf("warning lifecycle protocol contract drifted: %#v", record)
		}
		if !exactState && (len(record.OutputProtocols) != 0 || len(record.ResultValidationProtocols) != 0) {
			t.Errorf("warning lifecycle summary command falsely claimed an exact state protocol: %#v", record)
		}
		mutation := operation != "status" && operation != "list" && operation != "show"
		if mutation && (record.Mutability != "permission_backed_mutation" || record.AuthorityMode != "target_host_permission" || record.RecoveryCommandID == nil) {
			t.Errorf("warning lifecycle mutation posture drifted: %#v", record)
		}
		if !mutation && record.Mutability != "read_only" {
			t.Errorf("warning lifecycle read posture drifted: %#v", record)
		}
	}
}

func TestFoundationLifecycleRegistryContracts(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	records := make(map[string]commandregistry.CommandRecord)
	for _, record := range manifest.Commands {
		records[record.CommandID] = record
	}
	for _, component := range []string{"ssiag", "stav"} {
		for _, surface := range []string{"enrollment", "supervisor"} {
			backendFeature := map[string]string{
				"ssiag.enrollment": backendFeatureSSIAGEnrollment,
				"ssiag.supervisor": backendFeatureSSIAGSupervisor,
				"stav.enrollment":  backendFeatureSTAVEnrollment,
				"stav.supervisor":  backendFeatureSTAVSupervisor,
			}[component+"."+surface]
			for _, leaf := range []string{"status", "plan", "apply", "apply-status", "recover"} {
				id := "qxcmd:symphony:" + component + "." + surface + "." + leaf
				record, ok := records[id]
				if !ok {
					t.Errorf("foundational lifecycle command %s is absent", id)
					continue
				}
				operation := leaf
				if leaf == "status" {
					operation = "observe"
				}
				wantOperation := "engop:symphony:" + component + "." + surface + "." + operation
				if len(record.BackendOperationIDs) != 1 || record.BackendOperationIDs[0] != wantOperation ||
					!containsFeatureBinding(record.FeatureBindings, commandregistry.FeatureBinding{FeatureID: backendFeature, Interaction: "lifecycle"}) ||
					len(record.InputProtocols) != 1 || record.InputProtocols[0] != "symphony.foundation.lifecycle-command.v1" ||
					len(record.OutputProtocols) != 1 || record.OutputProtocols[0] != "symphony.foundation.lifecycle-result.v1" ||
					len(record.ResultValidationProtocols) != 1 || record.ResultValidationProtocols[0] != "symphony.foundation.lifecycle-result.v1" ||
					record.TargetScope != "local" || !record.JSONOutput {
					t.Errorf("foundational lifecycle command contract drifted: %#v", record)
				}
				if leaf == "apply" || leaf == "recover" {
					wantRecovery := "qxcmd:symphony:" + component + "." + surface + ".recover"
					if record.Mutability != "permission_backed_mutation" || record.AuthorityMode != "target_host_permission" ||
						record.RecoveryCommandID == nil || *record.RecoveryCommandID != wantRecovery {
						t.Errorf("mutation/recovery posture drifted: %#v", record)
					}
				} else if leaf == "plan" && record.Mutability != "proposal_only" {
					t.Errorf("plan is not proposal-only: %#v", record)
				} else if leaf != "plan" && record.Mutability != "read_only" {
					t.Errorf("read leaf is not read-only: %#v", record)
				}
			}
		}
	}
}

func TestSSIAGProviderTrustRegistryContracts(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	records := make(map[string]commandregistry.CommandRecord)
	for _, record := range manifest.Commands {
		records[record.CommandID] = record
	}
	show := records["qxcmd:symphony:ssiag.provider.show"]
	verify := records["qxcmd:symphony:ssiag.provider.verify"]
	if show.Mutability != "read_only" || show.AuthorityMode != "none" ||
		len(show.BackendOperationIDs) != 1 || show.BackendOperationIDs[0] != "engop:symphony:ssiag.provider.trust.show" ||
		len(show.OutputProtocols) != 1 || show.OutputProtocols[0] != "symphony.ssiag.provider-trust-result.v1" ||
		!containsFeatureBinding(show.FeatureBindings, commandregistry.FeatureBinding{FeatureID: backendFeatureSSIAGProviderTrust, Interaction: "inspect"}) {
		t.Fatalf("provider show contract drifted: %#v", show)
	}
	if verify.Mutability != "evidence_only" || verify.AuthorityMode != "ssiag" ||
		len(verify.BackendOperationIDs) != 2 ||
		verify.BackendOperationIDs[0] != "engop:symphony:ssiag.provider.metadata-handshake" ||
		verify.BackendOperationIDs[1] != "engop:symphony:ssiag.provider.trust.verify" ||
		len(verify.InputProtocols) != 1 || verify.InputProtocols[0] != "symphony.ssiag.provider-trust-verification-request.v1" ||
		len(verify.ResultValidationProtocols) != 1 || verify.ResultValidationProtocols[0] != "symphony.ssiag.provider-trust-result.v1" ||
		!containsFeatureBinding(verify.FeatureBindings, commandregistry.FeatureBinding{FeatureID: backendFeatureSSIAGProviderTrust, Interaction: "validate"}) {
		t.Fatalf("provider verify contract drifted: %#v", verify)
	}
	if !containsFeatureBinding(verify.FeatureBindings, commandregistry.FeatureBinding{
		FeatureID: backendFeatureSSIAGMacOSMetadata, Interaction: "validate",
	}) {
		t.Fatalf("provider verify does not administer the verified macOS metadata handshake: %#v", verify)
	}
	providers := records["qxcmd:symphony:ssiag.providers"]
	if containsFeatureBinding(providers.FeatureBindings, commandregistry.FeatureBinding{
		FeatureID: "ssfv:symphony:ssiag.macos-keychain-metadata", Interaction: "discover",
	}) {
		t.Fatalf("provider list still overclaims an adapter handshake: %#v", providers)
	}
}

func TestSSIAGProviderBindingRegistryContracts(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	records := make(map[string]commandregistry.CommandRecord)
	for _, record := range manifest.Commands {
		records[record.CommandID] = record
	}
	tests := []struct {
		key, backend, interaction, mutability, authority, input, output, recovery string
		feature                                                                   string
	}{
		{"ssiag.provider.installations", "engop:symphony:ssiag.provider.installations.list", "discover", "read_only", "none", "", "symphony.ssiag.provider-installation-inventory.v1", "", backendFeatureSSIAGProviderBinding},
		{"ssiag.provider.binding.status", "engop:symphony:ssiag.provider.binding.observe", "inspect", "read_only", "none", "", "symphony.ssiag.provider-binding-status.v1", "", backendFeatureSSIAGProviderBinding},
		{"ssiag.provider.binding.plan", "engop:symphony:ssiag.provider.binding.plan", "propose", "proposal_only", "ssiag", "symphony.ssiag.provider-binding-plan-request.v1", "symphony.ssiag.provider-binding-plan.v1", "", backendFeatureSSIAGProviderBinding},
		{"ssiag.provider.binding.apply", "engop:symphony:ssiag.provider.binding.apply", "apply", "permission_backed_mutation", "ssiag", "symphony.ssiag.provider-binding-apply-request.v1", "symphony.ssiag.provider-binding-result.v1", "qxcmd:symphony:ssiag.provider.binding.recover", backendFeatureSSIAGProviderBinding},
		{"ssiag.provider.binding.apply-status", "engop:symphony:ssiag.provider.binding.apply-status", "query", "read_only", "ssiag", "", "symphony.ssiag.provider-binding-result.v1", "", backendFeatureSSIAGProviderBinding},
		{"ssiag.provider.binding.recover", "engop:symphony:ssiag.provider.binding.recover", "recover", "permission_backed_mutation", "ssiag", "symphony.ssiag.provider-binding-recovery-request.v1", "symphony.ssiag.provider-binding-result.v1", "qxcmd:symphony:ssiag.provider.binding.recover", backendFeatureSSIAGProviderBinding},
	}
	for _, test := range tests {
		record, present := records["qxcmd:symphony:"+test.key]
		if !present {
			t.Errorf("provider binding command %s is absent", test.key)
			continue
		}
		if len(record.BackendOperationIDs) != 1 || record.BackendOperationIDs[0] != test.backend ||
			record.Mutability != test.mutability || record.AuthorityMode != test.authority ||
			len(record.OutputProtocols) != 1 || record.OutputProtocols[0] != test.output ||
			len(record.ResultValidationProtocols) != 1 || record.ResultValidationProtocols[0] != test.output ||
			!containsFeatureBinding(record.FeatureBindings, commandregistry.FeatureBinding{FeatureID: test.feature, Interaction: test.interaction}) {
			t.Errorf("provider binding command contract drifted: %#v", record)
		}
		if test.input == "" && len(record.InputProtocols) != 0 ||
			test.input != "" && (len(record.InputProtocols) != 1 || record.InputProtocols[0] != test.input) {
			t.Errorf("provider binding input protocol drifted: %#v", record)
		}
		if test.recovery == "" && record.RecoveryCommandID != nil ||
			test.recovery != "" && (record.RecoveryCommandID == nil || *record.RecoveryCommandID != test.recovery) {
			t.Errorf("provider binding recovery identity drifted: %#v", record)
		}
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

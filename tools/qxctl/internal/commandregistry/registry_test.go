package commandregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testSpec(id string) CommandSpec {
	purpose := "test infrastructure"
	return CommandSpec{
		CommandID: id, Status: "experimental", IntroducedIn: "0.1.0-dev",
		ReplacementIDs: []string{}, FeatureBindings: []FeatureBinding{},
		InfrastructurePurpose: &purpose, BackendOperationIDs: []string{},
		Mutability: "read_only", AuthorityMode: "none", TargetScope: "local",
		InputProtocols: []string{}, OutputProtocols: []string{},
		ResultValidationProtocols: []string{}, Noninteractive: true,
	}
}

func TestParityRejectsUnregisteredExecutable(t *testing.T) {
	root := Structural("qxctl", cobra.NoArgs, errTestStructural)
	root.AddCommand(&cobra.Command{Use: "forgotten", RunE: func(*cobra.Command, []string) error { return nil }})
	if err := Validate(root); err == nil || !strings.Contains(err.Error(), "no explicit command-registry role") {
		t.Fatalf("unregistered executable error = %v", err)
	}
}

func TestParityRejectsStructuralHandlerReplacement(t *testing.T) {
	root := Structural("qxctl", cobra.NoArgs, errTestStructural)
	group := Structural("group", cobra.NoArgs, errTestStructural)
	group.AddCommand(Attach(&cobra.Command{Use: "leaf", RunE: func(*cobra.Command, []string) error { return nil }}, testSpec("qxcmd:test:leaf")))
	group.RunE = func(*cobra.Command, []string) error { return nil }
	root.AddCommand(group)
	if err := Validate(root); err == nil || !strings.Contains(err.Error(), "outside the structural factory") {
		t.Fatalf("replaced structural handler error = %v", err)
	}
}

func TestParityRejectsDuplicateCommandID(t *testing.T) {
	root := Structural("qxctl", cobra.NoArgs, errTestStructural)
	for _, name := range []string{"one", "two"} {
		root.AddCommand(Attach(&cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}, testSpec("qxcmd:test:duplicate")))
	}
	if err := Validate(root); err == nil || !strings.Contains(err.Error(), "duplicate command ID") {
		t.Fatalf("duplicate ID error = %v", err)
	}
}

func TestRetiredIdentityIsFailClosedNullGrammarTombstone(t *testing.T) {
	root := Structural("qxctl", cobra.NoArgs, errTestStructural)
	spec := testSpec("qxcmd:test:former")
	deprecatedIn := "0.2.0"
	spec.Status = "retired"
	spec.DeprecatedIn = &deprecatedIn
	spec.Mutability = "prohibited"
	spec.ReplacementIDs = []string{"qxcmd:test:replacement"}
	tombstone := Retired("former", spec)
	root.AddCommand(tombstone)

	manifest, err := BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Commands) != 1 || manifest.Commands[0].Grammar != nil || manifest.Commands[0].Status != "retired" {
		t.Fatalf("retirement projection = %#v", manifest.Commands)
	}
	if err := tombstone.RunE(tombstone, []string{"ignored"}); err == nil || !strings.Contains(err.Error(), "qxcmd:test:replacement") {
		t.Fatalf("retirement diagnostic = %v", err)
	}

	tombstone.RunE = func(*cobra.Command, []string) error { return nil }
	if err := Validate(root); err == nil || !strings.Contains(err.Error(), "outside the tombstone factory") {
		t.Fatalf("replaced tombstone handler error = %v", err)
	}
}

func TestParityRejectsSchemaInvalidIdentifiersAndDuplicateArrays(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CommandSpec)
	}{
		{name: "digit-leading command namespace", mutate: func(spec *CommandSpec) { spec.CommandID = "qxcmd:1test:leaf" }},
		{name: "repeated key separator", mutate: func(spec *CommandSpec) { spec.CommandID = "qxcmd:test:bad..leaf" }},
		{name: "invalid feature identity", mutate: func(spec *CommandSpec) {
			spec.InfrastructurePurpose = nil
			spec.FeatureBindings = []FeatureBinding{{FeatureID: "ssfv:test:bad_thing", Interaction: "inspect"}}
		}},
		{name: "duplicate operation ID", mutate: func(spec *CommandSpec) {
			spec.BackendOperationIDs = []string{"engop:test:leaf.inspect", "engop:test:leaf.inspect"}
		}},
		{name: "duplicate protocol", mutate: func(spec *CommandSpec) {
			spec.OutputProtocols = []string{"fixture.result.v1", "fixture.result.v1"}
		}},
		{name: "feature and infrastructure", mutate: func(spec *CommandSpec) {
			spec.FeatureBindings = []FeatureBinding{{FeatureID: "ssfv:test:leaf", Interaction: "inspect"}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := testSpec("qxcmd:test:leaf")
			test.mutate(&spec)
			root := Structural("qxctl", cobra.NoArgs, errTestStructural)
			root.AddCommand(Attach(&cobra.Command{Use: "leaf", RunE: func(*cobra.Command, []string) error { return nil }}, spec))
			if err := Validate(root); err == nil {
				t.Fatal("schema-invalid CommandSpec was accepted")
			}
		})
	}
}

func TestManifestIsDeterministicAndAliasesShareOneRecord(t *testing.T) {
	root := Structural("qxctl", cobra.NoArgs, errTestStructural)
	leaf := &cobra.Command{Use: "inspect <name>", Aliases: []string{"show", "get"}, RunE: func(*cobra.Command, []string) error { return nil }}
	leaf.Flags().Bool("json", false, "emit JSON")
	root.AddCommand(Attach(leaf, testSpec("qxcmd:test:inspect")))
	first, err := Build(root, Identity{ClientVersion: "test", ExecutableDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(root, Identity{ClientVersion: "test", ExecutableDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	if first.RegistryDigest != second.RegistryDigest || len(first.Commands) != 1 {
		t.Fatalf("manifest is not deterministic or alias created records: %#v %#v", first, second)
	}
	if strings.Join(first.Commands[0].Aliases, ",") != "get,show" || !first.Commands[0].JSONOutput {
		t.Fatalf("alias/JSON projection = %#v", first.Commands[0])
	}

	digest := first.RegistryDigest
	first.RegistryDigest = ""
	canonical, err := canonicalJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(canonical)
	want := "sha256:" + hex.EncodeToString(hash[:])
	if digest != want {
		t.Fatalf("registry digest = %s, want %s over %s", digest, want, canonical)
	}
	first.RegistryDigest = digest
	if err := VerifyDigest(first); err != nil {
		t.Fatalf("verify generated digest: %v", err)
	}
	first.ClientVersion = stringPointer("tampered")
	if err := VerifyDigest(first); err == nil {
		t.Fatal("tampered registry digest unexpectedly verified")
	}
	if strings.Contains(string(canonical), "registry_digest") {
		t.Fatalf("digest preimage includes registry_digest: %s", canonical)
	}
}

func TestObservedReceiptTrustIsExact(t *testing.T) {
	root := Structural("qxctl", cobra.NoArgs, errTestStructural)
	root.AddCommand(Attach(&cobra.Command{Use: "leaf", RunE: func(*cobra.Command, []string) error { return nil }}, testSpec("qxcmd:test:leaf")))
	manifest, err := Build(root, Identity{ClientVersion: "test", ExecutableDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ClientTrust != "unreceipted" || manifest.ReceiptDigest != nil {
		t.Fatalf("unreceipted identity = %#v", manifest)
	}
	receipt := testDigest
	manifest, err = Build(root, Identity{ClientVersion: "test", ExecutableDigest: testDigest, ReceiptDigest: &receipt})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ClientTrust != "receipted" || manifest.ReceiptDigest == nil {
		t.Fatalf("receipted identity = %#v", manifest)
	}
	encoded, err := Marshal(manifest)
	if err != nil || !json.Valid(encoded) {
		t.Fatalf("manifest JSON error = %v; data=%s", err, encoded)
	}
}

var errTestStructural = &testError{}

type testError struct{}

func (*testError) Error() string { return "subcommand required" }

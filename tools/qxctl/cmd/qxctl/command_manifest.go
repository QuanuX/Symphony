package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
	"github.com/spf13/cobra"
)

func newCommandsCommand(root *cobra.Command) *cobra.Command {
	commands := structural("commands", fmt.Errorf("commands subcommand is required: manifest, expected, or verify"))
	var jsonOutput bool
	manifest := &cobra.Command{
		Use:  "manifest",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			if !jsonOutput {
				return fmt.Errorf("--json is required for the machine command registry")
			}
			executable, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolve qxctl executable: %w", err)
			}
			executableDigest, err := commandregistry.DigestExecutable(executable)
			if err != nil {
				return err
			}
			observed, err := commandregistry.Build(root, commandregistry.Identity{
				ClientVersion: commandProtocolVersion(), ExecutableDigest: executableDigest,
			})
			if err != nil {
				return err
			}
			encoded, err := commandregistry.Marshal(observed)
			if err != nil {
				return err
			}
			fmt.Println(string(encoded))
			return nil
		},
	}
	manifest.Flags().BoolVar(&jsonOutput, "json", false, "emit the digest-bound observed command registry")
	manifestSpec := commandSpec("commands.manifest", featureCommandRegistry, "discover")
	manifestSpec.OutputProtocols = []string{commandregistry.Protocol}
	manifestSpec.ResultValidationProtocols = []string{commandregistry.Protocol}
	commandregistry.Attach(manifest, manifestSpec)
	commands.AddCommand(manifest)

	var expectedJSON bool
	expected := &cobra.Command{
		Use:  "expected",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			if !expectedJSON {
				return fmt.Errorf("--json is required for the expected command registry")
			}
			registry, err := commandregistry.BuildExpected(root)
			if err != nil {
				return err
			}
			encoded, err := commandregistry.Marshal(registry)
			if err != nil {
				return err
			}
			fmt.Println(string(encoded))
			return nil
		},
	}
	expected.Flags().BoolVar(&expectedJSON, "json", false, "emit the client-independent expected command registry")
	expectedSpec := commandSpec("commands.expected", featureCommandRegistry, "discover")
	expectedSpec.OutputProtocols = []string{commandregistry.Protocol}
	expectedSpec.ResultValidationProtocols = []string{commandregistry.Protocol}
	commandregistry.Attach(expected, expectedSpec)
	commands.AddCommand(expected)

	var verifyInput string
	var verifyJSON bool
	verify := &cobra.Command{
		Use:  "verify",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			if verifyInput == "" || !verifyJSON {
				return fmt.Errorf("--input and --json are required")
			}
			provided, err := readExpectedRegistry(verifyInput)
			if err != nil {
				return err
			}
			want, err := commandregistry.BuildExpected(root)
			if err != nil {
				return err
			}
			providedBytes, _ := commandregistry.Marshal(provided)
			wantBytes, _ := commandregistry.Marshal(want)
			if !bytes.Equal(providedBytes, wantBytes) {
				return fmt.Errorf("expected command registry does not match the current CommandSpec/Cobra tree")
			}
			return printIndentedJSON(map[string]any{
				"protocol": "symphony.qxctl.command-registry-verification.v1",
				"valid":    true, "registry_digest": want.RegistryDigest,
			})
		},
	}
	verify.Flags().StringVar(&verifyInput, "input", "", "bounded expected command registry JSON")
	verify.Flags().BoolVar(&verifyJSON, "json", false, "emit verification JSON")
	verifySpec := commandSpec("commands.verify", featureCommandRegistry, "validate")
	verifySpec.InputProtocols = []string{commandregistry.Protocol}
	verifySpec.OutputProtocols = []string{"symphony.qxctl.command-registry-verification.v1"}
	verifySpec.ResultValidationProtocols = []string{commandregistry.Protocol}
	commandregistry.Attach(verify, verifySpec)
	commands.AddCommand(verify)
	return commands
}

func commandProtocolVersion() string { return strings.TrimPrefix(version.Version, "qxctl ") }

func readExpectedRegistry(path string) (commandregistry.Manifest, error) {
	var registry commandregistry.Manifest
	data, err := knowledgeengine.ReadPayload(path)
	if err != nil {
		return registry, fmt.Errorf("read expected command registry: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&registry); err != nil {
		return registry, fmt.Errorf("decode expected command registry: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return registry, fmt.Errorf("decode expected command registry: trailing JSON value")
	}
	if registry.RegistryKind != "expected" {
		return registry, fmt.Errorf("registry_kind must be expected")
	}
	if err := commandregistry.VerifyDigest(registry); err != nil {
		return registry, err
	}
	return registry, nil
}

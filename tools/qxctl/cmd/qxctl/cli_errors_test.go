package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
)

func decodeCLIError(t *testing.T, output string, status int, code string) cliErrorEnvelope {
	t.Helper()
	var envelope cliErrorEnvelope
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatalf("CLI output is not one error envelope: %v\n%s", err, output)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("additional output after JSON envelope: %v\n%s", err, output)
	}
	if envelope.Protocol != cliErrorProtocol || envelope.Outcome != "error" || envelope.Error.Code != code || envelope.ExitCode != status || status == 0 {
		t.Fatalf("unexpected error/status: %#v, status %d", envelope, status)
	}
	return envelope
}

func TestSCVJSONFailuresAreSingleSanitizedEnvelopes(t *testing.T) {
	malformed := filepath.Join(t.TempDir(), "secret-input.json")
	if err := os.WriteFile(malformed, []byte(`{"secret-input":`), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, code string
		args       []string
	}{
		{"missing input", "command_failed", []string{"scv", "interpret", "--json"}},
		{"invalid input", "command_failed", []string{"scv", "interpret", "--input", malformed, "--json"}},
		{"missing installation", "command_failed", []string{"scv", "inspect", "--prefix", filepath.Join(t.TempDir(), "secret-installation"), "--json"}},
		{"unsupported domain", "command_failed", []string{"scv", "inspect", "--domain", "secret-domain", "--json"}},
		{"unknown flag before json", "invalid_arguments", []string{"scv", "inspect", "--secret-flag=secret-value", "--json"}},
		{"unknown flag after json", "invalid_arguments", []string{"scv", "inspect", "--json", "--secret-flag=secret-value"}},
		{"missing flag value", "invalid_arguments", []string{"scv", "inspect", "--json", "--prefix"}},
		{"invalid json boolean", "invalid_arguments", []string{"scv", "inspect", "--json=secret-value"}},
		{"extra positional", "invalid_arguments", []string{"scv", "inspect", "secret-positional", "--json"}},
		{"missing leaf", "invalid_arguments", []string{"scv", "provider", "--json"}},
		{"unknown leaf", "invalid_arguments", []string{"scv", "secret-command", "--json"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, status := invokeCLI(t, test.args...)
			decodeCLIError(t, output, status, test.code)
			if strings.Contains(output, "secret-") {
				t.Fatalf("failure leaked caller text: %s", output)
			}
			again, againStatus := invokeCLI(t, test.args...)
			if again != output || againStatus != status {
				t.Fatal("same failure did not produce deterministic bytes and status")
			}
		})
	}
}

func TestSCVJSONIntentRespectsFlagValuesAndTerminators(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"scv", "inspect", "--json"}, true},
		{[]string{"scv", "inspect", "--json=true"}, true},
		{[]string{"scv", "inspect", "--json=1"}, true},
		{[]string{"scv", "inspect", "--json=false"}, false},
		{[]string{"scv", "inspect", "--json=0"}, false},
		{[]string{"scv", "inspect", "--json", "--json=False"}, false},
		{[]string{"scv", "inspect", "--json=false", "--json"}, true},
		{[]string{"scv", "inspect", "--json=invalid"}, true},
		{[]string{"scv", "inspect", "--", "--json"}, false},
		{[]string{"scv", "inspect", "--json", "--", "--json=false"}, true},
		{[]string{"scv", "inspect", "--prefix", "--json"}, false},
		{[]string{"scv", "inspect", "--prefix=--json"}, false},
		{[]string{"scv", "inspect", "--input", "--json", "--json=false"}, false},
		{[]string{"scv", "inspect", "--unknown", "--json"}, true},
		{[]string{"status", "--json"}, false},
	}
	for _, test := range tests {
		if got := scvJSONRequested(root, test.args); got != test.want {
			t.Errorf("JSON intent for %q = %t, want %t", test.args, got, test.want)
		}
	}
	for _, args := range [][]string{
		{"scv", "inspect", "--json=false"},
		{"scv", "inspect", "--json", "--json=false"},
		{"scv", "inspect", "--unknown", "--json=false"},
	} {
		output, status := invokeCLI(t, args...)
		if status != 1 || json.Valid([]byte(output)) || strings.Contains(output, cliErrorProtocol) {
			t.Fatalf("explicit false did not preserve human failure output: %q, %d", output, status)
		}
	}
}

func TestCLIErrorBoundaryPreservesEvidenceAndSanitizesEngineCodes(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := root.Find([]string{"scv", "provider", "interpret"})
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"scv", "provider", "interpret", "--json"}
	for _, code := range []string{"interpretation.invalid", "secret-code", "interpretation.secret", "\nsecret-code"} {
		output := captureStdout(t, func() error {
			finishCommandError(root, command, args, fmt.Errorf("secret-wrapper: %w", &knowledgeengine.ProcessError{Code: code, Message: "secret-message"}))
			return nil
		})
		envelope := decodeCLIError(t, output, 1, "engine_rejected")
		if strings.Contains(output, "secret") {
			t.Fatalf("engine diagnostics leaked into CLI result: %s", output)
		}
		if code == "interpretation.invalid" && (envelope.Error.EngineCode == nil || *envelope.Error.EngineCode != code) {
			t.Fatal("known stable engine code was lost")
		}
		if code != "interpretation.invalid" && envelope.Error.EngineCode != nil {
			t.Fatal("unknown engine code must be null")
		}
	}
	for _, failure := range []struct {
		err  error
		want int
	}{
		{&exactEvidenceExitError{code: 26}, 26},
		{&exactEvidenceExitError{code: 126}, 1},
		{&validationOutcomeError{outcome: "violation"}, 1},
	} {
		const evidence = "{\"existing_validated_evidence\":true}\n"
		output := captureStdout(t, func() error {
			fmt.Print(evidence)
			if status := finishCommandError(root, command, args, failure.err); status != failure.want {
				t.Errorf("specialized status = %d, want %d", status, failure.want)
			}
			return nil
		})
		if output != evidence {
			t.Fatalf("root duplicated or replaced existing evidence: %s", output)
		}
	}
	output := captureStdout(t, func() error {
		status := finishCommandError(root, command, args, &validation.ValidatorExitError{ExitCode: 26, Diagnostics: "secret-diagnostic"})
		if status != 26 {
			t.Fatalf("validator status = %d, want 26", status)
		}
		return nil
	})
	decodeCLIError(t, output, 26, "command_failed")
	if strings.Contains(output, "secret") {
		t.Fatal("validator diagnostics leaked")
	}
}

func TestInstalledSCVJSONFailureAndSuccessCompatibility(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_INTERPRETATION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.3.0-dev package")
	}
	input := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(input, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"scv", "provider", "interpret", "--prefix", prefix, "--input", input, "--version", "0.3.0-dev", "--json"}
	output, status := invokeCLI(t, args...)
	envelope := decodeCLIError(t, output, status, "engine_rejected")
	if envelope.Error.EngineCode == nil || *envelope.Error.EngineCode != "interpretation.invalid" {
		t.Fatalf("real process rejection did not preserve its safe code: %s", output)
	}
	if err := os.WriteFile(input, []byte(`{"source":null}`), 0o600); err != nil {
		t.Fatal(err)
	}
	args = []string{"scv", "source-check", "--prefix", prefix, "--input", input, "--version", "0.3.0-dev", "--json"}
	output, status = invokeCLI(t, args...)
	expected := captureStdout(t, func() error {
		return runSCV("source_status", scvOptions{domain: "scv", prefix: prefix, version: "0.3.0-dev", input: input, jsonOutput: true})
	})
	if status != 0 || output != expected || !json.Valid([]byte(output)) || strings.Contains(output, cliErrorProtocol) {
		t.Fatalf("success payload changed or was wrapped: status %d, output %s, expected %s", status, output, expected)
	}
}

func TestSHVJSONFailuresAreSingleSanitizedEnvelopes(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code string
	}{
		{[]string{"shv", "catalogue", "build", "--json"}, "command_failed"},
		{[]string{"shv", "evaluate", "--json"}, "command_failed"},
		{[]string{"shv", "source", "inspect", "--json"}, "command_failed"},
		{[]string{"shv", "catalogue", "build", "--unknown", "--json"}, "invalid_arguments"},
		{[]string{"shv", "graph", "--json"}, "invalid_arguments"},
	} {
		out, status := invokeCLI(t, tc.args...)
		decodeCLIError(t, out, status, tc.code)
	}
}

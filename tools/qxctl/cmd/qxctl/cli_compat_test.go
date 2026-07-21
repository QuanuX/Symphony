package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLICompatibility(t *testing.T) {
	helpBytes, err := os.ReadFile("testdata/help.golden")
	if err != nil {
		t.Fatal(err)
	}
	help := string(helpBytes)
	tests := []struct {
		name   string
		args   []string
		status int
		output string
	}{
		{name: "no arguments", status: 1, output: help},
		{name: "help", args: []string{"--help"}, output: help},
		{name: "version", args: []string{"--version"}, output: "qxctl version qxctl dev\n"},
		{name: "unknown", args: []string{"unknown"}, status: 1, output: "unknown command: unknown\n" + help},
		{name: "invalid inventory shape", args: []string{"inventory", "extra"}, status: 1, output: help},
		{name: "invalid modules flag", args: []string{"modules", "check", "--json"}, status: 1, output: help},
		{name: "invalid module shape", args: []string{"module", "inspect"}, status: 1, output: help},
		{name: "missing SSIAG subcommand", args: []string{"ssiag"}, status: 1, output: "ssiag failed: SSIAG subcommand is required: status, providers, or doctor\n"},
		{name: "missing STAV subcommand", args: []string{"stav"}, status: 1, output: "stav failed: STAV subcommand is required: status, verify, query, or doctor\n"},
		{name: "unknown STAV subcommand", args: []string{"stav", "unknown"}, status: 1, output: "stav failed: unknown STAV subcommand \"unknown\"\n"},
		{name: "prohibited STAV append", args: []string{"stav", "append"}, status: 1, output: "stav failed: qxctl stav append is prohibited; qxctl never submits arbitrary events or edits ledgers\n"},
		{name: "missing SKVI subcommand", args: []string{"skvi"}, status: 1, output: "skvi failed: SKVI subcommand is required: inspect, check, propose, or project\n"},
		{name: "SKVI prefix required", args: []string{"skvi", "inspect"}, status: 1, output: "skvi inspect failed: --prefix is required\n"},
		{name: "missing SCLV subcommand", args: []string{"sclv"}, status: 1, output: "sclv failed: SCLV subcommand is required: inspect, check, propose, recover, or project\n"},
		{name: "SCLV prefix required", args: []string{"sclv", "inspect"}, status: 1, output: "sclv inspect failed: --prefix is required\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, status := invokeCLI(t, test.args...)
			if status != test.status {
				t.Fatalf("exit status = %d, want %d; output:\n%s", status, test.status, output)
			}
			if output != test.output {
				t.Fatalf("output mismatch\n--- got ---\n%s--- want ---\n%s", output, test.output)
			}
		})
	}
}

func TestSSIAGViperBindingsAreExplicitAndFlagFirst(t *testing.T) {
	valid := ssiagTestTOPSID
	invalid := "INVALID"
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYMPHONY_SSIAG_TOPS_ID", "")
	t.Setenv("TOPS_ID", valid)
	if err := executeCommand([]string{"ssiag", "status"}); err == nil || !strings.Contains(err.Error(), "SYMPHONY_SSIAG_TOPS_ID is required") {
		t.Fatalf("unbound environment variable affected Viper: %v", err)
	}

	t.Setenv("SYMPHONY_SSIAG_TOPS_ID", valid)
	if err := executeCommand([]string{"ssiag", "status", "--tops-id", invalid}); err == nil || !strings.Contains(err.Error(), "canonical lowercase UUID") {
		t.Fatalf("explicit flag did not override environment value: %v", err)
	}

	t.Setenv("SYMPHONY_SSIAG_TOPS_ID", invalid)
	err := executeCommand([]string{"ssiag", "status", "--tops-id", valid})
	if err == nil || strings.Contains(err.Error(), "canonical lowercase UUID") {
		t.Fatalf("explicit flag was not authoritative: %v", err)
	}
}

func TestViperForbiddenCapabilitiesRemainAbsent(t *testing.T) {
	source, err := os.ReadFile("commands.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, "viper.New()") || !strings.Contains(text, `BindEnv("tops-id", "SYMPHONY_SSIAG_TOPS_ID")`) {
		t.Fatal("private Viper instance and explicit SSIAG binding are required")
	}
	for _, forbidden := range []string{
		"AutomaticEnv(", "ReadInConfig(", "AddConfigPath(", "AddRemoteProvider(",
		"WatchConfig(", "WriteConfig(", "SafeWriteConfig(", "SetConfigFile(",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("forbidden Viper capability %q is present", forbidden)
		}
	}
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("QXCTL_TEST_HELPER") != "1" {
		return
	}
	separator := -1
	for index, arg := range os.Args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator < 0 {
		os.Exit(2)
	}
	os.Exit(execute(os.Args[separator+1:]))
}

func invokeCLI(t *testing.T, args ...string) (string, int) {
	t.Helper()
	commandArgs := append([]string{"-test.run=^TestCLIHelperProcess$", "--"}, args...)
	command := exec.Command(os.Args[0], commandArgs...)
	command.Env = append(os.Environ(), "QXCTL_TEST_HELPER=1")
	output, err := command.CombinedOutput()
	if err == nil {
		return string(output), 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	return string(output), exitErr.ExitCode()
}

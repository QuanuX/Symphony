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
		{name: "missing SSIAG subcommand", args: []string{"ssiag"}, status: 1, output: "ssiag failed: SSIAG subcommand is required: status, providers, doctor, grants, policy, enrollment, or supervisor\n"},
		{name: "missing SSIAG policy subcommand", args: []string{"ssiag", "policy"}, status: 1, output: "ssiag failed: SSIAG policy subcommand is required: status, propose, apply, or recover\n"},
		{name: "SSIAG policy proposal identity required", args: []string{"ssiag", "policy", "propose"}, status: 1, output: "ssiag failed: --tops-id is required\n"},
		{name: "SSIAG lifecycle grant subject required", args: []string{"ssiag", "grants", "lifecycle"}, status: 1, output: "ssiag failed: --tops-id and --subject-id are required\n"},
		{name: "missing SSIAG enrollment subcommand", args: []string{"ssiag", "enrollment"}, status: 1, output: "ssiag failed: ssiag enrollment subcommand is required: status, plan, apply, apply-status, or recover\n"},
		{name: "SSIAG enrollment status target required", args: []string{"ssiag", "enrollment", "status"}, status: 1, output: "ssiag enrollment status failed: --prefix and --tops-id are required\n"},
		{name: "missing SSIAG supervisor subcommand", args: []string{"ssiag", "supervisor"}, status: 1, output: "ssiag failed: ssiag supervisor subcommand is required: status, plan, apply, apply-status, or recover\n"},
		{name: "SSIAG supervisor plan target required", args: []string{"ssiag", "supervisor", "plan"}, status: 1, output: "ssiag supervisor plan failed: --prefix and --tops-id are required\n"},
		{name: "missing STAV subcommand", args: []string{"stav"}, status: 1, output: "stav failed: STAV subcommand is required: status, verify, query, doctor, enrollment, or supervisor\n"},
		{name: "unknown STAV subcommand", args: []string{"stav", "unknown"}, status: 1, output: "stav failed: unknown STAV subcommand \"unknown\"\n"},
		{name: "prohibited STAV append", args: []string{"stav", "append"}, status: 1, output: "stav failed: qxctl stav append is prohibited; qxctl never submits arbitrary events or edits ledgers\n"},
		{name: "missing STAV enrollment subcommand", args: []string{"stav", "enrollment"}, status: 1, output: "stav failed: stav enrollment subcommand is required: status, plan, apply, apply-status, or recover\n"},
		{name: "STAV enrollment status target required", args: []string{"stav", "enrollment", "status"}, status: 1, output: "stav enrollment status failed: --prefix and --tops-id are required\n"},
		{name: "missing STAV supervisor subcommand", args: []string{"stav", "supervisor"}, status: 1, output: "stav failed: stav supervisor subcommand is required: status, plan, apply, apply-status, or recover\n"},
		{name: "STAV recover target required", args: []string{"stav", "supervisor", "recover"}, status: 1, output: "stav supervisor recover failed: --prefix and --tops-id are required\n"},
		{name: "missing knowledge subcommand", args: []string{"knowledge"}, status: 1, output: "knowledge failed: knowledge subcommand is required: invariant, engines, reconcile, session, or lifecycle\n"},
		{name: "missing knowledge invariant subcommand", args: []string{"knowledge", "invariant"}, status: 1, output: "knowledge failed: knowledge invariant subcommand is required: status, list, show, or check\n"},
		{name: "knowledge invariant show identity required", args: []string{"knowledge", "invariant", "show"}, status: 1, output: "knowledge invariant show failed: --invariant-id is required\n"},
		{name: "knowledge invariant check prefix required", args: []string{"knowledge", "invariant", "check"}, status: 1, output: "knowledge invariant check failed: --prefix is required\n"},
		{name: "missing knowledge engines subcommand", args: []string{"knowledge", "engines"}, status: 1, output: "knowledge failed: knowledge engines subcommand is required: list, inspect, doctor, bind, or unbind\n"},
		{name: "knowledge bind role required", args: []string{"knowledge", "engines", "bind"}, status: 1, output: help},
		{name: "missing knowledge reconcile subcommand", args: []string{"knowledge", "reconcile"}, status: 1, output: "knowledge failed: knowledge reconcile subcommand is required: compatibility, begin, status, checkpoint, close, or recover\n"},
		{name: "reconcile begin operation id required", args: []string{"knowledge", "reconcile", "begin"}, status: 1, output: "knowledge reconcile begin failed: --operation-id is required\n"},
		{name: "reconcile recover operation id required", args: []string{"knowledge", "reconcile", "recover"}, status: 1, output: "knowledge reconcile recover failed: --operation-id is required\n"},
		{name: "missing knowledge session subcommand", args: []string{"knowledge", "session"}, status: 1, output: "knowledge failed: knowledge session subcommand is required: begin, status, checkpoint, close, recover, transition, or features\n"},
		{name: "session begin TOPS required", args: []string{"knowledge", "session", "begin"}, status: 1, output: "knowledge session begin failed: --tops-id is required\n"},
		{name: "session transition event required", args: []string{"knowledge", "session", "transition"}, status: 1, output: "knowledge session transition failed: --event must be login, refresh, or logout\n"},
		{name: "missing knowledge lifecycle subcommand", args: []string{"knowledge", "lifecycle"}, status: 1, output: "knowledge failed: knowledge lifecycle subcommand is required: profile, ownership, host, observe, report, boot, status, recover, apply, apply-status, or apply-recover\n"},
		{name: "missing lifecycle profile subcommand", args: []string{"knowledge", "lifecycle", "profile"}, status: 1, output: "knowledge failed: knowledge lifecycle profile subcommand is required: list, show, set, or remove\n"},
		{name: "missing lifecycle ownership subcommand", args: []string{"knowledge", "lifecycle", "ownership"}, status: 1, output: "knowledge failed: knowledge lifecycle ownership subcommand is required: status, reconcile, adopt, or release\n"},
		{name: "missing lifecycle host subcommand", args: []string{"knowledge", "lifecycle", "host"}, status: 1, output: "knowledge failed: knowledge lifecycle host subcommand is required: install, update, status, reconcile, enable, disable, uninstall, or run\n"},
		{name: "lifecycle host TOPS required", args: []string{"knowledge", "lifecycle", "host", "status"}, status: 1, output: "knowledge lifecycle host status failed: --tops-id is required\n"},
		{name: "lifecycle ownership root required", args: []string{"knowledge", "lifecycle", "ownership", "status"}, status: 1, output: "knowledge lifecycle ownership status failed: --root is required\n"},
		{name: "lifecycle profile list TOPS required", args: []string{"knowledge", "lifecycle", "profile", "list"}, status: 1, output: "knowledge lifecycle profile list failed: --tops-id is required\n"},
		{name: "lifecycle observe TOPS required", args: []string{"knowledge", "lifecycle", "observe"}, status: 1, output: "knowledge lifecycle observe failed: --tops-id is required\n"},
		{name: "lifecycle report TOPS required", args: []string{"knowledge", "lifecycle", "report"}, status: 1, output: "knowledge lifecycle report failed: --tops-id is required\n"},
		{name: "lifecycle boot mutation identity required", args: []string{"knowledge", "lifecycle", "boot"}, status: 1, output: "knowledge lifecycle boot failed: --operation-id and --expected-journal-digest are required\n"},
		{name: "lifecycle status TOPS required", args: []string{"knowledge", "lifecycle", "status"}, status: 1, output: "knowledge lifecycle status failed: --tops-id is required\n"},
		{name: "lifecycle recover operation required", args: []string{"knowledge", "lifecycle", "recover"}, status: 1, output: "knowledge lifecycle recover failed: --operation-id is required\n"},
		{name: "lifecycle apply mutation identity required", args: []string{"knowledge", "lifecycle", "apply"}, status: 1, output: "knowledge lifecycle apply failed: --operation-id, --source-journal-digest, --expected-apply-journal-digest, and --expected-applied-state-digest are required\n"},
		{name: "lifecycle apply status TOPS required", args: []string{"knowledge", "lifecycle", "apply-status"}, status: 1, output: "knowledge lifecycle apply-status failed: --tops-id is required\n"},
		{name: "lifecycle apply recover operation required", args: []string{"knowledge", "lifecycle", "apply-recover"}, status: 1, output: "knowledge lifecycle apply-recover failed: --operation-id is required\n"},
		{name: "missing validate subcommand", args: []string{"validate"}, status: 1, output: "validate failed: validate subcommand is required: scan, debug, root-summary, profile, baseline, or warning\n"},
		{name: "validate scan identity required", args: []string{"validate", "scan"}, status: 1, output: "validate scan failed: --tops-id and --prefix are required\n"},
		{name: "validate root summary prefix required", args: []string{"validate", "root-summary"}, status: 1, output: "validate root-summary failed: --prefix is required\n"},
		{name: "missing validate profile subcommand", args: []string{"validate", "profile"}, status: 1, output: "validate failed: validate profile subcommand is required: list, show, set, or remove\n"},
		{name: "validate profile TOPS required", args: []string{"validate", "profile", "list"}, status: 1, output: "validate profile list failed: --tops-id is required\n"},
		{name: "missing validate baseline subcommand", args: []string{"validate", "baseline"}, status: 1, output: "validate failed: validate baseline subcommand is required: create, show, or remove\n"},
		{name: "validate baseline TOPS required", args: []string{"validate", "baseline", "show"}, status: 1, output: "validate baseline show failed: --tops-id is required\n"},
		{name: "missing validate warning subcommand", args: []string{"validate", "warning"}, status: 1, output: "validate failed: validate warning subcommand is required: status, list, show, sync, accept, reopen, supersede, mute, or unmute\n"},
		{name: "validate warning TOPS required", args: []string{"validate", "warning", "status"}, status: 1, output: "validate warning status failed: --tops-id is required\n"},
		{name: "missing SKVI subcommand", args: []string{"skvi"}, status: 1, output: "skvi failed: SKVI subcommand is required: inspect, check, propose, or project\n"},
		{name: "SKVI prefix required", args: []string{"skvi", "inspect"}, status: 1, output: "skvi inspect failed: --prefix is required\n"},
		{name: "missing SCLV subcommand", args: []string{"sclv"}, status: 1, output: "sclv failed: SCLV subcommand is required: inspect, check, propose, recover, project, or evidence\n"},
		{name: "SCLV prefix required", args: []string{"sclv", "inspect"}, status: 1, output: "sclv inspect failed: --prefix is required\n"},
		{name: "missing SCLV evidence adapter", args: []string{"sclv", "evidence"}, status: 1, output: "sclv failed: SCLV evidence adapter is required: local-git or airgap\n"},
		{name: "SCLV evidence prefix required", args: []string{"sclv", "evidence", "local-git"}, status: 1, output: "sclv evidence local-git failed: --prefix is required\n"},
		{name: "missing SACV subcommand", args: []string{"sacv"}, status: 1, output: "sacv failed: SACV subcommand is required: inspect, check, diff, propose, or project\n"},
		{name: "SACV prefix required", args: []string{"sacv", "inspect"}, status: 1, output: "sacv inspect failed: --prefix is required\n"},
		{name: "missing SODV subcommand", args: []string{"sodv"}, status: 1, output: "sodv failed: SODV subcommand is required: inspect, check, verify, propose, recover, or project\n"},
		{name: "SODV prefix required", args: []string{"sodv", "inspect"}, status: 1, output: "sodv inspect failed: --prefix is required\n"},
		{name: "missing SSFV subcommand", args: []string{"ssfv"}, status: 1, output: "ssfv failed: SSFV subcommand is required: inspect, check, diff, propose, graph, or administration-check\n"},
		{name: "SSFV prefix required", args: []string{"ssfv", "inspect"}, status: 1, output: "ssfv inspect failed: --prefix is required\n"},
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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/foundation"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/lifecycle"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

const commandTestTOPSID = "01234567-89ab-4def-8123-456789abcdef"

func TestSystemEnrollmentRequiresExplicitAuthorityIdentity(t *testing.T) {
	err := runEnroll([]string{"--scope", "system", "--tops-id", commandTestTOPSID})
	if err == nil || !strings.Contains(err.Error(), "requires explicit --authority-uid and --authority-gid") {
		t.Fatalf("expected explicit system identity error, got %v", err)
	}
}

func TestEnrollmentRequiresAuthorityIdentityPair(t *testing.T) {
	err := runEnroll([]string{"--scope", "system", "--tops-id", commandTestTOPSID, "--authority-uid", "123"})
	if err == nil || !strings.Contains(err.Error(), "must be supplied together") {
		t.Fatalf("expected authority identity pair error, got %v", err)
	}
}

func TestUserEnrollmentRejectsAuthorityIdentityOverride(t *testing.T) {
	err := runEnroll([]string{
		"--scope", "user",
		"--tops-id", commandTestTOPSID,
		"--authority-uid", "123",
		"--authority-gid", "456",
	})
	if err == nil || !strings.Contains(err.Error(), "does not accept an override") {
		t.Fatalf("expected user identity override error, got %v", err)
	}
}

func TestSystemEnrollmentRejectsInvalidAuthorityIdentity(t *testing.T) {
	err := runEnroll([]string{
		"--scope", "system",
		"--tops-id", commandTestTOPSID,
		"--authority-uid", "not-a-number",
		"--authority-gid", "456",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid authority UID") {
		t.Fatalf("expected invalid authority UID error, got %v", err)
	}
}

func TestSupervisorForceAndNoStopCannotBypassTransaction(t *testing.T) {
	if err := runSupervisor([]string{"install", "--scope", "user", "--tops-id", commandTestTOPSID, "--force"}); err == nil || !strings.Contains(err.Error(), "unavailable through the expected-state transaction engine") {
		t.Fatalf("force bypass result: %v", err)
	}
	if err := runSupervisor([]string{"uninstall", "--scope", "user", "--tops-id", commandTestTOPSID, "--no-stop"}); err == nil || !strings.Contains(err.Error(), "requires stop") {
		t.Fatalf("no-stop bypass result: %v", err)
	}
}

func TestFoundationLifecycleRealProcessBoundary(t *testing.T) {
	prefix := t.TempDir()
	installed, err := lifecycle.InstallAt(os.Args[0], stavpaths.ScopeUser, prefix, version.Version)
	if err != nil {
		t.Fatalf("install receipt-backed test executable: %v", err)
	}
	home := t.TempDir()
	environment := []string{
		"SYMPHONY_STAV_FOUNDATION_PROCESS_HELPER=1",
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, "config"),
		"XDG_STATE_HOME=" + filepath.Join(home, "state"),
		"XDG_RUNTIME_DIR=" + filepath.Join(home, "runtime"),
	}

	describe := exec.Command(installed.Binary, "-test.run=^TestFoundationLifecycleProcessHelper$", "--", "foundation-lifecycle", "describe", "--json")
	describe.Env = environment
	descriptorOutput, err := describe.Output()
	if err != nil {
		t.Fatalf("run installed descriptor process: %v", err)
	}
	var descriptor foundation.AdapterDescriptor
	if err := json.Unmarshal(descriptorOutput, &descriptor); err != nil {
		t.Fatalf("descriptor stdout is not exactly one JSON value: %v; output=%q", err, descriptorOutput)
	}
	installedBinary, err := filepath.EvalSymlinks(installed.Binary)
	if err != nil {
		t.Fatalf("resolve installed process: %v", err)
	}
	if descriptor.Protocol != foundation.AdapterProtocol || descriptor.BinaryPath != installedBinary {
		t.Fatalf("descriptor did not bind installed process: %#v", descriptor)
	}

	now := time.Now().UTC().Truncate(time.Second)
	command := foundation.Command{
		Protocol: foundation.CommandProtocol, FormatVersion: 1,
		Operation: "observe", Component: "stav", Surface: "enrollment", Scope: "user", TOPSID: commandTestTOPSID,
		RequestedAt: now.Format(time.RFC3339), DeadlineAt: now.Add(time.Minute).Format(time.RFC3339),
	}
	input, err := json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	invoke := exec.Command(installed.Binary, "-test.run=^TestFoundationLifecycleProcessHelper$", "--", "foundation-lifecycle")
	invoke.Env = environment
	invoke.Stdin = bytes.NewReader(append(input, '\n'))
	resultOutput, err := invoke.Output()
	if err != nil {
		t.Fatalf("run installed stdin/stdout process: %v", err)
	}
	var result foundation.Result
	if err := json.Unmarshal(resultOutput, &result); err != nil {
		t.Fatalf("result stdout is not exactly one JSON value: %v; output=%q", err, resultOutput)
	}
	if result.Protocol != foundation.ResultProtocol || result.Disposition != "observed" || !result.ReadOnly {
		t.Fatalf("unexpected installed process result: %#v", result)
	}

	reject := exec.Command(installed.Binary, "-test.run=^TestFoundationLifecycleProcessHelper$", "--", "foundation-lifecycle")
	reject.Env = environment
	reject.Stdin = bytes.NewBufferString(string(input) + "\n{}\n")
	if output, err := reject.CombinedOutput(); err == nil {
		t.Fatalf("installed process accepted trailing JSON: %q", output)
	}
}

func TestFoundationLifecycleProcessHelper(t *testing.T) {
	if os.Getenv("SYMPHONY_STAV_FOUNDATION_PROCESS_HELPER") != "1" {
		return
	}
	separator := -1
	for index, argument := range os.Args {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator < 0 {
		fmt.Fprintln(os.Stderr, "missing helper argument separator")
		os.Exit(2)
	}
	if err := run(os.Args[separator+1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

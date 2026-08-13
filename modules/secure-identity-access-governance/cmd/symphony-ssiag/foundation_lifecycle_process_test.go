package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationlifecycle"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

func TestFoundationLifecycleRealProcessBoundary(t *testing.T) {
	prefix := t.TempDir()
	installed, err := packageinstall.Install(os.Args[0], prefix, version.Version)
	if err != nil {
		t.Fatalf("install receipt-backed test executable: %v", err)
	}
	home := t.TempDir()
	environment := []string{
		"SYMPHONY_SSIAG_FOUNDATION_PROCESS_HELPER=1",
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
	var descriptor foundationlifecycle.AdapterDescriptor
	if err := json.Unmarshal(descriptorOutput, &descriptor); err != nil {
		t.Fatalf("descriptor stdout is not exactly one JSON value: %v; output=%q", err, descriptorOutput)
	}
	if descriptor.Protocol != foundationlifecycle.AdapterProtocol || descriptor.BinaryPath != installed.Binary {
		t.Fatalf("descriptor did not bind installed process: %#v", descriptor)
	}

	now := time.Now().UTC().Truncate(time.Second)
	command := foundationlifecycle.Command{
		Protocol: foundationlifecycle.CommandProtocol, FormatVersion: 1,
		Operation: "observe", Component: "ssiag", Surface: "enrollment", Scope: "user",
		TOPSID:      "01234567-89ab-4def-8123-456789abcdef",
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
	var result foundationlifecycle.Result
	if err := json.Unmarshal(resultOutput, &result); err != nil {
		t.Fatalf("result stdout is not exactly one JSON value: %v; output=%q", err, resultOutput)
	}
	if result.Protocol != foundationlifecycle.ResultProtocol || result.Disposition != "observed" || !result.ReadOnly {
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
	if os.Getenv("SYMPHONY_SSIAG_FOUNDATION_PROCESS_HELPER") != "1" {
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

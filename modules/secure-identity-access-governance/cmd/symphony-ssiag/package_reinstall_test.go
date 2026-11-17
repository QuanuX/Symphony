package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageretirement"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

func TestInstalledPackageReinstallUsesExclusiveTargetCustody(t *testing.T) {
	for _, mode := range []string{"unchanged", "busy", "retiring", "missing", "other-prefix"} {
		t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", root)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
			t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
			source, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			installed, err := packageinstall.Install(source, filepath.Join(root, "prefix"), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(installed.Receipt)
			if err != nil {
				t.Fatal(err)
			}
			wantFailure := ""
			target := installed.Prefix
			switch mode {
			case "busy":
				use, err := packageinstall.AcquireUseForExecutable(installed.Binary)
				if err != nil {
					t.Fatal(err)
				}
				defer use.Close()
				wantFailure = "package installation is in use"
			case "retiring":
				id, _, err := packageretirement.Inspect(installed.Prefix, packageinstall.ComponentID, version.Version)
				if err != nil {
					t.Fatal(err)
				}
				lease, err := packageretirement.BeginRetirement(installed.Prefix, id)
				if err != nil {
					t.Fatal(err)
				}
				if err := lease.Close(); err != nil {
					t.Fatal(err)
				}
				wantFailure = "retired, retiring"
			case "missing":
				if err := os.RemoveAll(filepath.Join(installed.Prefix, ".symphony-package-retirement")); err != nil {
					t.Fatal(err)
				}
				wantFailure = "package retirement evidence unavailable"
			case "other-prefix":
				target = filepath.Join(root, "second-prefix")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, installed.Binary, "-test.run=^TestFoundationLifecycleProcessHelper$", "--", "package", "install", "--prefix", target, "--version", version.Version)
			command.Env = append(os.Environ(), "SYMPHONY_SSIAG_FOUNDATION_PROCESS_HELPER=1")
			output, runErr := command.CombinedOutput()
			if wantFailure == "" {
				if runErr != nil {
					t.Fatalf("installed owner could not install %s: %v\n%s", mode, runErr, output)
				}
				if mode == "unchanged" && !strings.Contains(string(output), "changed=false") {
					t.Fatalf("reinstall changed its immutable package: %s", output)
				}
			} else if runErr == nil || !strings.Contains(string(output), wantFailure) {
				t.Fatalf("expected %q refusal, got %v\n%s", wantFailure, runErr, output)
			}
			after, err := os.ReadFile(installed.Receipt)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("source receipt changed: %v", err)
			}
			if mode != "missing" {
				_, phase, err := packageretirement.Inspect(installed.Prefix, packageinstall.ComponentID, version.Version)
				wantPhase := "open"
				if mode == "retiring" {
					wantPhase = "retiring"
				}
				if err != nil || phase != wantPhase {
					t.Fatalf("source admission changed: %s %v", phase, err)
				}
			}
		})
	}
}

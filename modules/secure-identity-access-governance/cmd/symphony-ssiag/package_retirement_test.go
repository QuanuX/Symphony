package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageretirement"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

func TestInstalledServeRetirementChild(t *testing.T) {
	expected := os.Getenv("SYMPHONY_SSIAG_RETIREMENT_CHILD")
	if expected == "" {
		return
	}
	err := runServe([]string{"--scope", "user", "--tops-id", "018f0c3a-7b2d-7e11-8c12-0242ac120002"})
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("installed serve reached configuration/provider state before its package guard: %v", err)
	}
}
func TestInstalledServeRefusesClosedOrMissingGuardBeforeState(t *testing.T) {
	for _, mode := range []string{"closed", "missing"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
			t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
			source, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			installed, err := packageinstall.Install(source, filepath.Join(home, "prefix"), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			expected := "package admission is closed"
			if mode == "closed" {
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
			} else {
				if err := os.RemoveAll(filepath.Join(installed.Prefix, ".symphony-package-retirement")); err != nil {
					t.Fatal(err)
				}
				expected = "package retirement evidence unavailable"
			}
			child := exec.Command(installed.Binary, "-test.run=^TestInstalledServeRetirementChild$")
			child.Env = append(os.Environ(), "SYMPHONY_SSIAG_RETIREMENT_CHILD="+expected)
			if output, err := child.CombinedOutput(); err != nil {
				t.Fatalf("guarded serve subprocess: %v\n%s", err, output)
			}
			for _, path := range []string{filepath.Join(home, "config"), filepath.Join(home, "state")} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("refused serve created runtime state: %s %v", path, err)
				}
			}
		})
	}
}

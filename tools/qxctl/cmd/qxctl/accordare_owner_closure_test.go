//go:build darwin || linux

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
)

// Build the actual selected public module, then exercise qxctl's imported
// receipt reader against that module's real installation. A workspace replace
// or a fixture-only receipt must not conceal an old published client.
func TestAccordareOwnerDependencyClosure(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller := filepath.Join(base, "retained-owner")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-mod=readonly", "-trimpath", "-o", controller, "github.com/QuanuX/Symphony/modules/accordare-stav-producer/cmd/symphony-accordare-stav-producer")
	build.Env = append(os.Environ(), "GOWORK=off", "GOENV=off", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build selected native owner: %v\n%s", err, output)
	}
	prefix := filepath.Join(base, "installation")
	install := exec.CommandContext(ctx, controller, "install", "--prefix", prefix, "--version", "0.1.0-dev")
	install.Env = append(os.Environ(), "HOME="+filepath.Join(base, "home"), "XDG_CONFIG_HOME="+filepath.Join(base, "config"), "XDG_STATE_HOME="+filepath.Join(base, "state"), "XDG_RUNTIME_DIR="+filepath.Join(base, "runtime"))
	if output, err := install.CombinedOutput(); err != nil {
		t.Fatalf("install selected native owner: %v\n%s", err, output)
	}
	installed, err := accordareclient.VerifyInstallation(prefix, "0.1.0-dev")
	if err != nil {
		t.Fatalf("qxctl imported reader rejects selected owner's installation: %v", err)
	}
	raw, err := os.ReadFile(installed.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&receipt); err != nil {
		t.Fatal(err)
	}
	capabilities, ok := receipt["provides_capabilities"].([]any)
	if !ok {
		t.Fatal("owner receipt has no capabilities")
	}
	seen := map[string]bool{}
	var altered []any
	for _, capability := range capabilities {
		name, ok := capability.(string)
		if !ok {
			t.Fatal("owner capability is not a string")
		}
		seen[name] = true
		if name != "symphony.package-owner-dispatch.v1" {
			altered = append(altered, name)
		}
	}
	if !seen["symphony.package-retirement.v1"] || !seen["symphony.package-owner-dispatch.v1"] {
		t.Fatal("selected dependency still builds an unqualified legacy owner")
	}
	// Recompute the ordinary receipt digest so refusal proves the selected
	// reader enforces the owner contract, not merely a stale digest string.
	receipt["provides_capabilities"] = altered
	delete(receipt, "receipt_digest")
	canonical, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(canonical)
	receipt["receipt_digest"] = "sha256:" + hex.EncodeToString(hash[:])
	changed, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed.Receipt, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := accordareclient.VerifyInstallation(prefix, "0.1.0-dev"); err == nil {
		t.Fatal("source-bound client accepted the legacy capability contract")
	}
	if err := os.WriteFile(installed.Receipt, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := accordareclient.VerifyInstallation(prefix, "0.1.0-dev"); err != nil {
		t.Fatalf("restored exact owner receipt rejected: %v", err)
	}
}

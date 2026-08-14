package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

func TestInstalledStoppedServiceProviderBindingRecoveryReachesProtectedState(t *testing.T) {
	prefix := t.TempDir()
	installed, err := packageinstall.Install(os.Args[0], prefix, version.Version)
	if err != nil {
		t.Fatalf("install receipt-backed test executable: %v", err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))

	const topsID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"
	layout, err := ssiagpaths.ResolveInstance(ssiagpaths.ScopeUser, topsID)
	if err != nil {
		t.Fatal(err)
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	cfg := config.Default(layout, "Offline Recovery Integration", &uid, &gid)
	cfg.Providers = []config.ProviderConfig{{
		Name: "native", Kind: "macos-keychain", Enabled: true,
		Capabilities: []string{"capability-discovery", "metadata"}, Interactive: true,
	}}
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.ConfigDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ConfigFile, data, 0o600); err != nil {
		t.Fatal(err)
	}

	command := exec.Command(installed.Binary,
		"-test.run=^TestFoundationLifecycleProcessHelper$", "--",
		"provider-binding-recover",
		"--scope", "user",
		"--tops-id", topsID,
		"--provider", "native",
		"--expected-state-digest", "absent",
		"--reason", "exercise installed stopped-service recovery",
	)
	command.Env = append(os.Environ(), "SYMPHONY_SSIAG_FOUNDATION_PROCESS_HELPER=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("installed stopped-service recovery unexpectedly succeeded without durable attempt evidence: %q", output)
	}
	if !strings.Contains(string(output), "SSIAG provider binding recovery evidence is absent") {
		t.Fatalf("installed stopped-service recovery did not reach the protected binding store: %v; output=%q", err, output)
	}
	if _, err := os.Lstat(layout.Socket); !os.IsNotExist(err) {
		t.Fatalf("offline recovery created or retained a service socket: %v", err)
	}
}

func TestOfflineProviderBindingRecoveryRequiresReceiptBoundFoundation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := runProviderBindingRecover([]string{
		"--scope", "user",
		"--tops-id", "018f0c3a-7b2d-7e11-8c12-0242ac120002",
		"--provider", "native",
		"--expected-state-digest", "absent",
		"--reason", "resume an interrupted binding attempt",
	})
	if err == nil || !strings.Contains(err.Error(), "requires receipt-bound SSIAG") {
		t.Fatalf("development binary entered offline provider recovery: %v", err)
	}
}

func TestOfflineProviderBindingRecoveryAuthorityIsScopeExact(t *testing.T) {
	if owner, drop, allowed := offlineRecoveryAuthority(ssiagpaths.ScopeUser, 501, 20, 501, 20); !allowed || drop || owner != 501 {
		t.Fatal("exact user-scope service owner was rejected")
	}
	if _, _, allowed := offlineRecoveryAuthority(ssiagpaths.ScopeUser, 501, 20, 502, 20); allowed {
		t.Fatal("different user was admitted to offline recovery")
	}
	if owner, drop, allowed := offlineRecoveryAuthority(ssiagpaths.ScopeSystem, 0, 0, 0, 0); !allowed || drop || owner != 0 {
		t.Fatal("root-owned system service was rejected")
	}
	if owner, drop, allowed := offlineRecoveryAuthority(ssiagpaths.ScopeSystem, 991, 991, 0, 20); !allowed || !drop || owner != 0 {
		t.Fatal("root could not enter the enrolled non-root system service identity")
	}
	if _, _, allowed := offlineRecoveryAuthority(ssiagpaths.ScopeSystem, 991, 991, 991, 991); allowed {
		t.Fatal("non-root system service was treated as target-host owner")
	}
}

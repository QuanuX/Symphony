package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/supervision"
)

func TestInstallAndUninstallUserBinary(t *testing.T) {
	home := canonicalTempDir(t)
	t.Setenv("HOME", home)
	source := filepath.Join(canonicalTempDir(t), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("version-one"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := installTestFixture(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatal("first install should report a change")
	}
	wantTarget := filepath.Join(home, ".local", "libexec", "symphony", "stav-append-authority", "dev", stavpaths.BinaryName)
	if result.Binary != wantTarget {
		t.Fatalf("binary = %q, want %q", result.Binary, wantTarget)
	}
	manifest := filepath.Join(home, ".local", "share", "symphony", "receipts", "stav-append-authority", "dev", "install-receipt.json")
	if info, err := os.Lstat(manifest); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("receipt-v2 was not published last as a regular file: info=%v error=%v", info, err)
	}
	record, receiptDigest, err := VerifyInstalled(stavpaths.ScopeUser)
	if err != nil || record.Schema != "symphony.knowledge.install-receipt.v2" || receiptDigest != record.ReceiptDigest {
		t.Fatalf("invalid receipt-v2 install evidence: record=%+v digest=%s error=%v", record, receiptDigest, err)
	}

	result, err = installTestFixture(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed {
		t.Fatal("idempotent install should not report a change")
	}

	if err := os.WriteFile(source, []byte("version-two"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := installTestFixture(source, stavpaths.ScopeUser, false); err == nil {
		t.Fatal("differing install should require force")
	}
	if _, err := installTestFixture(source, stavpaths.ScopeUser, true); err == nil {
		t.Fatal("immutable same-version package was replaced")
	}
	if err := os.WriteFile(source, []byte("version-one"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err = Uninstall(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatal("uninstall should report a change")
	}
	if _, err := os.Lstat(wantTarget); !os.IsNotExist(err) {
		t.Fatalf("installed binary still exists: %v", err)
	}

	result, err = Uninstall(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed {
		t.Fatal("idempotent uninstall should not report a change")
	}
}

func TestUninstallRefusesReferencedPackage(t *testing.T) {
	home := canonicalTempDir(t)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	source := filepath.Join(canonicalTempDir(t), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("referenced"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := installTestFixture(source, stavpaths.ScopeUser, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Enroll(stavpaths.ScopeUser, enrollmentTOPSID, uint64(os.Geteuid()), uint64(os.Getegid())); err != nil {
		t.Fatal(err)
	}
	layout, _ := stavpaths.ResolveInstance(stavpaths.ScopeUser, enrollmentTOPSID)
	cfg, err := config.Load(layout.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	install, _ := stavpaths.ResolveInstall(stavpaths.ScopeUser)
	spec, err := supervision.SpecFromConfig(stavpaths.ScopeUser, enrollmentTOPSID, install.Binary, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := supervision.InstallCAS(spec, "absent"); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, true); err == nil {
		t.Fatal("referenced installed package was removed")
	}
}

func TestCustomPrefixReceiptAndCompiledVersionBinding(t *testing.T) {
	t.Setenv("HOME", canonicalTempDir(t))
	source := filepath.Join(canonicalTempDir(t), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("custom-prefix"), 0o755); err != nil {
		t.Fatal(err)
	}
	prefix := filepath.Join(canonicalTempDir(t), "opt", "symphony")
	result, err := installTestFixtureAt(source, stavpaths.ScopeUser, prefix, "dev")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(prefix, "libexec", "symphony", "stav-append-authority", "dev", stavpaths.BinaryName)
	if result.Binary != want {
		t.Fatalf("binary=%q want=%q", result.Binary, want)
	}
	if _, err := installTestFixtureAt(source, stavpaths.ScopeUser, prefix, "other"); err == nil {
		t.Fatal("binary self-labeled a non-compiled version")
	}
	if _, _, _, err := VerifyExecutable(want, stavpaths.ScopeUser); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallRejectsChangedBinaryEvenWithForce(t *testing.T) {
	home := canonicalTempDir(t)
	t.Setenv("HOME", home)
	source := filepath.Join(canonicalTempDir(t), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("expected"), 0755); err != nil {
		t.Fatal(err)
	}
	installed, err := installTestFixture(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed.Binary, []byte("tampered"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, false); err == nil {
		t.Fatal("changed binary should require force")
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, true); err == nil {
		t.Fatal("force bypassed exact receipt-owned binary verification")
	}
	if data, err := os.ReadFile(installed.Binary); err != nil || string(data) != "tampered" {
		t.Fatalf("refused removal changed payload: %q %v", data, err)
	}
}

func TestInstallRejectsSymlinkTarget(t *testing.T) {
	home := canonicalTempDir(t)
	t.Setenv("HOME", home)
	source := filepath.Join(canonicalTempDir(t), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("expected"), 0755); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(home, ".local", "libexec", "symphony", "stav-append-authority", "dev")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, filepath.Join(targetDir, stavpaths.BinaryName)); err != nil {
		t.Fatal(err)
	}
	if _, err := installTestFixture(source, stavpaths.ScopeUser, true); err == nil {
		t.Fatal("symlink target unexpectedly accepted")
	}
}

// Synthetic payloads exercise storage contracts only, never service qualification.
func installTestFixture(source string, scope stavpaths.Scope, force bool) (Result, error) {
	layout, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return Result{}, err
	}
	return installAt(source, layout, force, "")
}
func installTestFixtureAt(source string, scope stavpaths.Scope, prefix, v string) (Result, error) {
	layout, err := stavpaths.ResolveInstallAt(scope, prefix, v)
	if err != nil {
		return Result{}, err
	}
	return installAt(source, layout, false, "")
}
func TestManagedInstallRefusesUnqualifiedSource(t *testing.T) {
	t.Setenv("HOME", canonicalTempDir(t))
	source := filepath.Join(canonicalTempDir(t), "legacy")
	if err := os.WriteFile(source, []byte("legacy"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(source, stavpaths.ScopeUser, true); err == nil {
		t.Fatal("arbitrary copied bytes acquired retirement capability")
	}
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}

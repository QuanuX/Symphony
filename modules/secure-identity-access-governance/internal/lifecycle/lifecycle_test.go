package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func setupUser(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	return home
}

func installTestBinary(t *testing.T, home string) InstallRecord {
	t.Helper()
	source := filepath.Join(home, "source-ssiag")
	if err := os.WriteFile(source, []byte("test executable"), 0755); err != nil {
		t.Fatal(err)
	}
	record, err := Install(source, ssiagpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestInstallIsIdempotentAndUninstallPreservesTOPS(t *testing.T) {
	home := setupUser(t)
	first := installTestBinary(t, home)
	installedDigest, err := fileDigest(first.Binary)
	if err != nil {
		t.Fatal(err)
	}
	if installedDigest != first.BinarySHA256 {
		t.Fatal("installation manifest digest does not bind the installed bytes")
	}
	second := installTestBinary(t, home)
	if first.BinarySHA256 != second.BinarySHA256 {
		t.Fatal("idempotent install changed digest")
	}
	enrollment, err := Enroll(ssiagpaths.ScopeUser, testTOPSID, "Trading desk")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(ssiagpaths.ScopeUser, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(enrollment.ConfigFile); err != nil {
		t.Fatalf("TOPS config should be preserved: %v", err)
	}
	if _, err := os.Stat(first.Binary); !os.IsNotExist(err) {
		t.Fatalf("binary should be removed, got %v", err)
	}
}

func TestMultipleTOPSEnrollmentsAreIsolated(t *testing.T) {
	home := setupUser(t)
	installTestBinary(t, home)
	secondID := "018f0c3a-7b2d-7e11-8c12-0242ac120003"
	first, err := Enroll(ssiagpaths.ScopeUser, testTOPSID, "Desk one")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Enroll(ssiagpaths.ScopeUser, secondID, "Desk two")
	if err != nil {
		t.Fatal(err)
	}
	if first.ConfigFile == second.ConfigFile || first.Socket == second.Socket || first.StateDir == second.StateDir {
		t.Fatal("TOPS enrollment paths collided")
	}
}

func TestDisplayNameCanChangeWithoutMovingState(t *testing.T) {
	home := setupUser(t)
	installTestBinary(t, home)
	first, err := Enroll(ssiagpaths.ScopeUser, testTOPSID, "Old name")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Enroll(ssiagpaths.ScopeUser, testTOPSID, "New name")
	if err != nil {
		t.Fatal(err)
	}
	if first.ConfigFile != second.ConfigFile || second.TOPSName != "New name" {
		t.Fatal("display name change altered security path or was not recorded")
	}
}

func TestUnenrollPreservesUnlessPurged(t *testing.T) {
	home := setupUser(t)
	installTestBinary(t, home)
	record, err := Enroll(ssiagpaths.ScopeUser, testTOPSID, "Desk")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Unenroll(ssiagpaths.ScopeUser, testTOPSID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(record.ConfigFile); err != nil {
		t.Fatalf("config should be preserved: %v", err)
	}
	if _, err := Enroll(ssiagpaths.ScopeUser, testTOPSID, "Desk"); err != nil {
		t.Fatal(err)
	}
	if _, err := Unenroll(ssiagpaths.ScopeUser, testTOPSID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(record.ConfigFile); !os.IsNotExist(err) {
		t.Fatalf("purge should remove config, got %v", err)
	}
}

func TestUninstallRejectsChangedBinary(t *testing.T) {
	home := setupUser(t)
	record := installTestBinary(t, home)
	if err := os.WriteFile(record.Binary, []byte("changed executable"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(ssiagpaths.ScopeUser, false); err == nil {
		t.Fatal("expected digest mismatch error")
	}
}

func TestInstallRejectsSymlinkBinary(t *testing.T) {
	home := setupUser(t)
	source := filepath.Join(home, "source-ssiag")
	if err := os.WriteFile(source, []byte("test executable"), 0755); err != nil {
		t.Fatal(err)
	}
	layout, err := ssiagpaths.ResolveInstall(ssiagpaths.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(layout.Binary), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, layout.Binary); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Install(source, ssiagpaths.ScopeUser, false); err == nil {
		t.Fatal("expected symlink binary to be rejected")
	}
}

func TestInstallRejectsSymlinkedStateAncestor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	external := filepath.Join(home, "external-state")
	if err := os.Mkdir(external, 0700); err != nil {
		t.Fatal(err)
	}
	stateLink := filepath.Join(home, "state-link")
	if err := os.Symlink(external, stateLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("XDG_STATE_HOME", stateLink)
	source := filepath.Join(home, "source-ssiag")
	if err := os.WriteFile(source, []byte("test executable"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(source, ssiagpaths.ScopeUser, false); err == nil {
		t.Fatal("expected symlinked state ancestor to be rejected")
	}
}

func TestEnrollRejectsChangedInstalledBinary(t *testing.T) {
	home := setupUser(t)
	record := installTestBinary(t, home)
	if err := os.WriteFile(record.Binary, []byte("unexpected replacement"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Enroll(ssiagpaths.ScopeUser, testTOPSID, "Desk"); err == nil {
		t.Fatal("expected enrollment to reject changed installed binary")
	}
}

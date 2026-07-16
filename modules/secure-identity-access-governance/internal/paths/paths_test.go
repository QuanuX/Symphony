package paths

import (
	"path/filepath"
	"testing"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestUserLayoutsSeparateInstallAndTOPSState(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, "cfg")
	state := filepath.Join(home, "state")
	runtime := filepath.Join(home, "runtime")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("XDG_RUNTIME_DIR", runtime)

	install, err := ResolveInstall(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := ResolveInstance(ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	if install.InstallManifest != filepath.Join(state, "symphony", "ssiag", "install.json") {
		t.Fatalf("unexpected install manifest %q", install.InstallManifest)
	}
	if instance.ConfigFile != filepath.Join(config, "symphony", testTOPSID, "ssiag", "config.json") {
		t.Fatalf("unexpected config path %q", instance.ConfigFile)
	}
	if instance.Socket != filepath.Join(runtime, "symphony", testTOPSID, "ssiag.sock") {
		t.Fatalf("unexpected socket path %q", instance.Socket)
	}
}

func TestFallbackRuntimeRemainsTOPSIsolated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", "")
	layout, err := ResolveInstance(ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "state", "symphony", testTOPSID, "ssiag", "run", "symphony", testTOPSID, "ssiag.sock")
	if layout.Socket != want {
		t.Fatalf("unexpected fallback socket %q", layout.Socket)
	}
}

func TestTOPSIDValidation(t *testing.T) {
	for _, invalid := range []string{
		"",
		"desk",
		"018F0C3A-7B2D-7E11-8C12-0242AC120002",
		"../018f0c3a-7b2d-7e11-8c12-0242ac120002",
		"00000000-0000-0000-0000-000000000000",
		"018f0c3a-7b2d-0e11-8c12-0242ac120002",
		"018f0c3a-7b2d-9e11-8c12-0242ac120002",
		"018f0c3a-7b2d-7e11-7c12-0242ac120002",
	} {
		if err := ValidateTOPSID(invalid); err == nil {
			t.Fatalf("expected %q to be rejected", invalid)
		}
	}
	if err := ValidateTOPSID(testTOPSID); err != nil {
		t.Fatalf("valid ID rejected: %v", err)
	}
}

func TestParseScopeRejectsUnknown(t *testing.T) {
	if _, err := ParseScope("global"); err == nil {
		t.Fatal("expected unsupported scope error")
	}
}

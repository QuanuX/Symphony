package paths

import (
	"path/filepath"
	"runtime"
	"testing"
)

const testTOPSID = "01234567-89ab-4def-8123-456789abcdef"

func TestResolveUserInstanceUsesStateFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_RUNTIME_DIR", "")

	layout, err := ResolveInstance(ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	wantState := filepath.Join(home, ".local", "state", "symphony", testTOPSID, "stav")
	if layout.StateDir != wantState {
		t.Fatalf("state directory = %q, want %q", layout.StateDir, wantState)
	}
	wantSocket := filepath.Join(wantState, "run", "append.sock")
	if layout.Socket != wantSocket {
		t.Fatalf("socket = %q, want %q", layout.Socket, wantSocket)
	}
	wantConfig := filepath.Join(home, ".config", "symphony", testTOPSID, "stav", "append-authority.json")
	if layout.ConfigFile != wantConfig {
		t.Fatalf("config = %q, want %q", layout.ConfigFile, wantConfig)
	}
	if layout.LedgerFile != filepath.Join(wantState, "ledger-v1.stavlog") || layout.RecoveryDir != filepath.Join(wantState, "recovery") {
		t.Fatalf("unexpected ledger layout: %#v", layout)
	}
}

func TestResolveUserInstanceUsesXDGRuntime(t *testing.T) {
	home := t.TempDir()
	runtimeBase := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_RUNTIME_DIR", runtimeBase)

	layout, err := ResolveInstance(ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(runtimeBase, "symphony", testTOPSID, "stav", "append.sock")
	if layout.Socket != want {
		t.Fatalf("socket = %q, want %q", layout.Socket, want)
	}
}

func TestResolveSystemInstance(t *testing.T) {
	layout, err := ResolveInstance(ScopeSystem, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	if layout.ConfigFile != filepath.Join("/etc/symphony", testTOPSID, "stav", "append-authority.json") {
		t.Fatalf("unexpected config path %q", layout.ConfigFile)
	}
	if layout.StateDir != filepath.Join("/var/lib/symphony", testTOPSID, "stav") {
		t.Fatalf("unexpected state path %q", layout.StateDir)
	}
	runtimeRoot := "/run/symphony"
	if runtime.GOOS == "darwin" {
		runtimeRoot = "/var/run/symphony"
	}
	if layout.Socket != filepath.Join(runtimeRoot, testTOPSID, "stav", "append.sock") {
		t.Fatalf("unexpected socket path %q", layout.Socket)
	}
}

func TestRejectsNonCanonicalTOPSID(t *testing.T) {
	for _, value := range []string{"", "01234567-89AB-cdef-0123-456789abcdef", "0123456789ab-cdef-0123-456789abcdef", "not-a-uuid"} {
		if err := ValidateTOPSID(value); err == nil {
			t.Fatalf("ValidateTOPSID(%q) unexpectedly succeeded", value)
		}
	}
}

func TestParseScope(t *testing.T) {
	if _, err := ParseScope("host"); err == nil {
		t.Fatal("unsupported scope unexpectedly succeeded")
	}
}

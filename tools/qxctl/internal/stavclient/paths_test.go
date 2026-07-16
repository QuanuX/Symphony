package stavclient

import (
	"path/filepath"
	"testing"
)

const testTOPSID = "01234567-89ab-4def-8123-456789abcdef"

func TestSocketForTOPSUserFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("XDG_STATE_HOME", "")

	got, err := SocketForTOPS("user", testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".local", "state", "symphony", testTOPSID, "stav", "run", "append.sock")
	if got != want {
		t.Fatalf("socket = %q, want %q", got, want)
	}
}

func TestSocketForTOPSUserRuntime(t *testing.T) {
	runtimeBase := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeBase)

	got, err := SocketForTOPS("user", testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(runtimeBase, "symphony", testTOPSID, "stav", "append.sock")
	if got != want {
		t.Fatalf("socket = %q, want %q", got, want)
	}
}

func TestSocketForTOPSRejectsInvalidInput(t *testing.T) {
	if _, err := SocketForTOPS("user", "INVALID"); err == nil {
		t.Fatal("invalid TOPS ID unexpectedly accepted")
	}
	if _, err := SocketForTOPS("host", testTOPSID); err == nil {
		t.Fatal("invalid scope unexpectedly accepted")
	}
}

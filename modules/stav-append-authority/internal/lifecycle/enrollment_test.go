package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

const enrollmentTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func TestEnrollPreserveAndExplicitPurge(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	source := filepath.Join(t.TempDir(), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("test-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(source, stavpaths.ScopeUser, false); err != nil {
		t.Fatal(err)
	}
	record, err := Enroll(stavpaths.ScopeUser, enrollmentTOPSID, uint64(os.Geteuid()), uint64(os.Getegid()))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(record.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TOPSID != enrollmentTOPSID || len(cfg.Authentication.Producers) != 0 || len(cfg.Authentication.Readers) != 0 {
		t.Fatalf("unexpected enrollment configuration: %#v", cfg)
	}
	if err := os.WriteFile(filepath.Join(record.StateDir, "preserved"), []byte("evidence"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Unenroll(stavpaths.ScopeUser, enrollmentTOPSID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(record.StateDir, "preserved")); err != nil {
		t.Fatalf("default unenroll removed state: %v", err)
	}
	if _, err := Enroll(stavpaths.ScopeUser, enrollmentTOPSID, uint64(os.Geteuid()), uint64(os.Getegid())); err != nil {
		t.Fatal(err)
	}
	if _, err := Unenroll(stavpaths.ScopeUser, enrollmentTOPSID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(record.StateDir); !os.IsNotExist(err) {
		t.Fatalf("explicit purge left state: %v", err)
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"

	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

const testTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func TestDefaultRoundTripAndLayoutBinding(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", "")
	layout, err := stavpaths.ResolveInstance(stavpaths.ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Default(layout, uint64(os.Geteuid()), uint64(os.Getegid()))
	data, err := Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "append-authority.json")
	formatted := append([]byte("\n  "), data...)
	formatted = append(formatted, '\n')
	if err := os.WriteFile(path, formatted, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateLayout(loaded, layout); err != nil {
		t.Fatal(err)
	}
	if loaded.Authentication.Producers == nil || len(loaded.Authentication.Producers) != 0 || loaded.Authentication.Readers == nil || len(loaded.Authentication.Readers) != 0 {
		t.Fatalf("default grants are not explicit empty arrays: %#v", loaded.Authentication)
	}
}

func TestLoadRejectsUnknownAndDuplicateFields(t *testing.T) {
	for name, data := range map[string]string{
		"unknown":   `{"schema":"symphony.stav.append-authority.config.v1","unknown":true}`,
		"duplicate": `{"schema":"symphony.stav.append-authority.config.v1","schema":"symphony.stav.append-authority.config.v1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("unsafe configuration unexpectedly loaded")
			}
		})
	}
}

func TestLoadRejectsSymlink(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	layout, err := stavpaths.ResolveInstance(stavpaths.ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(Default(layout, uint64(os.Geteuid()), uint64(os.Getegid())))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	target := filepath.Join(root, "target.json")
	link := filepath.Join(root, "config.json")
	if err := os.WriteFile(target, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("symlink configuration unexpectedly loaded")
	}
}

func TestLoadRejectsConfigurationWritableByAnotherIdentity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	layout, err := stavpaths.ResolveInstance(stavpaths.ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(Default(layout, uint64(os.Geteuid()), uint64(os.Getegid())))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0o622); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o622); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("configuration writable by group or other unexpectedly loaded")
	}
}

package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

func TestInstallAndUninstallUserBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	source := filepath.Join(t.TempDir(), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("version-one"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := Install(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatal("first install should report a change")
	}
	wantTarget := filepath.Join(home, ".local", "bin", stavpaths.BinaryName)
	if result.Binary != wantTarget {
		t.Fatalf("binary = %q, want %q", result.Binary, wantTarget)
	}

	result, err = Install(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed {
		t.Fatal("idempotent install should not report a change")
	}

	if err := os.WriteFile(source, []byte("version-two"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(source, stavpaths.ScopeUser, false); err == nil {
		t.Fatal("differing install should require force")
	}
	if _, err := Install(source, stavpaths.ScopeUser, true); err != nil {
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

func TestUninstallRejectsChangedBinaryWithoutForce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	source := filepath.Join(t.TempDir(), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("expected"), 0755); err != nil {
		t.Fatal(err)
	}
	installed, err := Install(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed.Binary, []byte("tampered"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, false); err == nil {
		t.Fatal("changed binary should require force")
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, true); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRejectsSymlinkTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	source := filepath.Join(t.TempDir(), stavpaths.BinaryName)
	if err := os.WriteFile(source, []byte("expected"), 0755); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, filepath.Join(targetDir, stavpaths.BinaryName)); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(source, stavpaths.ScopeUser, true); err == nil {
		t.Fatal("symlink target unexpectedly accepted")
	}
}

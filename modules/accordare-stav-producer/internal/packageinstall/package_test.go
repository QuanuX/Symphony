package packageinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/version"
)

func TestReceiptV2InstallReplayAndUninstall(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	prefix := filepath.Join(t.TempDir(), "prefix")
	installed, err := Install(source, prefix, version.Version)
	if err != nil || !installed.Changed {
		t.Fatalf("install failed: result=%+v err=%v", installed, err)
	}
	if _, err := os.Stat(installed.Binary); err != nil {
		t.Fatal(err)
	}
	replayed, err := Install(source, prefix, version.Version)
	if err != nil || replayed.Changed || replayed.ReceiptDigest != installed.ReceiptDigest {
		t.Fatalf("idempotent install failed: result=%+v err=%v", replayed, err)
	}
	removed, err := Uninstall(prefix, version.Version)
	if err != nil || !removed.Changed {
		t.Fatalf("uninstall failed: result=%+v err=%v", removed, err)
	}
	if _, err := os.Lstat(installed.Binary); !os.IsNotExist(err) {
		t.Fatal("receipt-owned executable remained after uninstall")
	}
}

func TestUninstallRefusesEnrollmentReference(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, "config")
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	prefix := filepath.Join(root, "prefix")
	if _, err := Install(source, prefix, version.Version); err != nil {
		t.Fatal(err)
	}
	enrollment := filepath.Join(configRoot, "symphony", "11111111-1111-4111-8111-111111111111", componentID)
	if err := os.MkdirAll(enrollment, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(enrollment, "config.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(prefix, version.Version); err == nil {
		t.Fatal("uninstall accepted a retained per-TOPS enrollment")
	}
}

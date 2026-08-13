package packageinstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setupPackageTest(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	prefix := filepath.Join(home, "prefix")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return source, prefix
}

func TestReceiptV2InstallInspectAndUninstall(t *testing.T) {
	source, prefix := setupPackageTest(t)
	installed, err := Install(source, prefix, compiledVersion())
	if err != nil {
		t.Fatal(err)
	}
	if !installed.Changed || !validDigest(installed.ReceiptDigest) {
		t.Fatalf("unexpected install result: %+v", installed)
	}
	evidence, current, err := InspectExecutable(installed.Binary)
	if err != nil || !current {
		t.Fatalf("installed entry point was not recognized: current=%t err=%v", current, err)
	}
	if evidence.Version != compiledVersion() || evidence.Binary != installed.Binary || evidence.ReceiptDigest != installed.ReceiptDigest {
		t.Fatalf("unexpected package evidence: %+v", evidence)
	}
	data, err := os.ReadFile(installed.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	files, ok := raw["files"].([]any)
	if !ok || len(files) != 1 || files[0].(map[string]any)["path"] == nil {
		t.Fatalf("receipt file fields are not schema-shaped: %s", data)
	}
	entryPoints, ok := raw["entry_points"].([]any)
	if !ok || len(entryPoints) != 1 || entryPoints[0].(map[string]any)["entry_point_id"] != EntryPointID {
		t.Fatalf("receipt entry point fields are not schema-shaped: %s", data)
	}
	uninstalled, err := Uninstall(prefix, compiledVersion())
	if err != nil {
		t.Fatal(err)
	}
	if !uninstalled.Changed {
		t.Fatal("uninstall did not report a change")
	}
	if _, err := os.Lstat(installed.Binary); !os.IsNotExist(err) {
		t.Fatalf("owned executable survived uninstall: %v", err)
	}
}

func TestInstallRejectsArbitraryVersionLabel(t *testing.T) {
	source, prefix := setupPackageTest(t)
	_, err := Install(source, prefix, compiledVersion()+"-self-labeled")
	if err == nil || !strings.Contains(err.Error(), "compiled binary version") {
		t.Fatalf("arbitrary version label accepted: %v", err)
	}
	if _, statErr := os.Lstat(prefix); !os.IsNotExist(statErr) {
		t.Fatalf("rejected install mutated prefix: %v", statErr)
	}
}

func TestUninstallRefusesSupervisorReference(t *testing.T) {
	source, prefix := setupPackageTest(t)
	installed, err := Install(source, prefix, compiledVersion())
	if err != nil {
		t.Fatal(err)
	}
	var descriptor string
	if runtime.GOOS == "darwin" {
		descriptor = filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "io.github.quanux.symphony.ssiag.018f0c3a-7b2d-4e11-8c12-0242ac120002.plist")
	} else if runtime.GOOS == "linux" {
		descriptor = filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "systemd", "user", "symphony-ssiag@018f0c3a-7b2d-4e11-8c12-0242ac120002.service")
	} else {
		t.Skip("native SSIAG supervisor is supported only on macOS and Linux")
	}
	if err := os.MkdirAll(filepath.Dir(descriptor), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(descriptor, []byte("ExecStart="+installed.Binary+" serve\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(prefix, compiledVersion()); err == nil || !strings.Contains(err.Error(), "supervisor:") {
		t.Fatalf("referenced package was uninstalled: %v", err)
	}
	if _, err := os.Stat(installed.Binary); err != nil {
		t.Fatalf("refused uninstall removed executable: %v", err)
	}
}

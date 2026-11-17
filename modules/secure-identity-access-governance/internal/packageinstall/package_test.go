package packageinstall

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageretirement"
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

func TestRetirementRetainsLivePackageAndRecoversExactAttempt(t *testing.T) {
	source, prefix := setupPackageTest(t)
	installed, err := Install(source, prefix, compiledVersion())
	if err != nil {
		t.Fatal(err)
	}
	use, err := AcquireUseForExecutable(installed.Binary)
	if err != nil || use == nil {
		t.Fatalf("admit installed use: %v", err)
	}
	if _, err := Uninstall(prefix, compiledVersion()); err == nil {
		use.Close()
		t.Fatal("live package removed")
	}
	if _, err := os.Stat(installed.Binary); err != nil {
		t.Fatal(err)
	}
	if retry, err := AcquireUseForExecutable(installed.Binary); err == nil {
		retry.Close()
		t.Fatal("retiring package admitted another use")
	}
	if err := use.Close(); err != nil {
		t.Fatal(err)
	}
	if result, err := Uninstall(prefix, compiledVersion()); err != nil || !result.Changed {
		t.Fatalf("retirement did not recover after terminal use: %+v %v", result, err)
	}
	if result, err := Uninstall(prefix, compiledVersion()); err != nil || result.Changed {
		t.Fatalf("exact tombstone retry: %+v %v", result, err)
	}
	if _, err := Install(source, prefix, compiledVersion()); err == nil {
		t.Fatal("retired installation epoch reopened")
	}
}
func TestMissingGuardRefusesLegacyAdoptionAndRemoval(t *testing.T) {
	source, prefix := setupPackageTest(t)
	installed, err := Install(source, prefix, compiledVersion())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(installed.Prefix, ".symphony-package-retirement")); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(prefix, compiledVersion()); err == nil {
		t.Fatal("missing guard inferred quiescence")
	}
	if _, err := Install(source, prefix, compiledVersion()); err == nil {
		t.Fatal("legacy package silently adopted")
	}
	if _, err := os.Stat(installed.Binary); err != nil {
		t.Fatal(err)
	}
}
func TestInstallRefusesUncontrolledSourceBytes(t *testing.T) {
	_, prefix := setupPackageTest(t)
	source := filepath.Join(filepath.Dir(prefix), "uncontrolled")
	if err := os.WriteFile(source, []byte("uncontrolled older program"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(source, prefix, compiledVersion()); err == nil || !strings.Contains(err.Error(), "executing owner binary") {
		t.Fatalf("uncontrolled source admitted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(prefix, ".symphony-package-retirement")); !os.IsNotExist(err) {
		t.Fatal("uncontrolled source created custody")
	}
}

func TestActivationRequiresExactReceiptAtPublishedPath(t *testing.T) {
	for _, change := range []string{"move_parent", "change_bytes"} {
		t.Run(change, func(t *testing.T) {
			_, prefix := setupPackageTest(t)
			l, err := resolve(prefix, compiledVersion())
			if err != nil {
				t.Fatal(err)
			}
			if err := ensurePrefix(l.prefix); err != nil {
				t.Fatal(err)
			}
			binary := []byte("bounded activation fixture")
			hash := sha256.Sum256(binary)
			receipt := newReceipt(l, uint64(len(binary)), "sha256:"+hex.EncodeToString(hash[:]))
			receipt.ReceiptDigest, err = digestWithoutReceipt(receipt)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.MarshalIndent(receipt, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			raw = append(raw, '\n')
			rawHash := sha256.Sum256(raw)
			lease, err := packageretirement.PrepareInstall(l.prefix, retirementIdentity(l, receipt), true)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.Close()
			if _, err := lease.PublishFile(l.binaryRelative, bytes.NewReader(binary), receipt.Files[0].Digest, uint64(len(binary)), 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := lease.PublishFile(receiptRelative(l), bytes.NewReader(raw), "sha256:"+hex.EncodeToString(rawHash[:]), uint64(len(raw)), 0o644); err != nil {
				t.Fatal(err)
			}
			if change == "move_parent" {
				if err := os.Rename(filepath.Dir(l.receipt), filepath.Dir(l.receipt)+"-moved"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(l.receipt, append(append([]byte{}, raw...), byte(' ')), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := activatePackage(lease, l, receipt, raw); err == nil {
				t.Fatal("activation accepted missing or changed published receipt")
			}
			if lease.State() != "installing" {
				t.Fatalf("failed activation opened admission: %s", lease.State())
			}
			if admitted, err := packageretirement.AcquireUse(l.prefix, retirementIdentity(l, receipt)); err == nil {
				admitted.Close()
				t.Fatal("failed activation admitted service use")
			}
		})
	}
}

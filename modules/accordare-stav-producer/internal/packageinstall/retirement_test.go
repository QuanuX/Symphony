package packageinstall

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/packageretirement"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/version"
)

type activationReader struct {
	reader     io.Reader
	beforeRead func()
}

func (r *activationReader) Read(b []byte) (int, error) {
	if r.beforeRead != nil {
		hook := r.beforeRead
		r.beforeRead = nil
		hook()
	}
	return r.reader.Read(b)
}

func TestActivationRechecksBothFinalFiles(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, changedFile := range []string{"binary-parent", "receipt"} {
			t.Run(map[bool]string{false: "fresh", true: "existing"}[existing]+"-"+changedFile, func(t *testing.T) {
				prefix := canonicalTempDir(t)
				p, err := resolve(prefix, version.Version)
				if err != nil {
					t.Fatal(err)
				}
				payload := []byte("disposable native owner fixture")
				record, err := makeReceipt(p, uint64(len(payload)), bytesDigest(payload))
				if err != nil {
					t.Fatal(err)
				}
				id := retirementIdentity(p, record)
				receiptBytes, err := json.MarshalIndent(record, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				receiptBytes = append(receiptBytes, '\n')
				lease, err := packageretirement.PrepareInstall(prefix, id, true)
				if err != nil {
					t.Fatal(err)
				}
				defer lease.Close()
				if _, err := lease.PublishFile(id.BinaryRelative, bytes.NewReader(payload), id.BinaryDigest, id.BinarySize, 0o755); err != nil {
					t.Fatal(err)
				}
				moveBinary := func() {
					parent := filepath.Dir(p.binary)
					if err := os.Rename(parent, parent+"-retained"); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(parent, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				var input io.Reader = bytes.NewReader(receiptBytes)
				if !existing && changedFile == "binary-parent" {
					input = &activationReader{reader: input, beforeRead: moveBinary}
				}
				if _, err := lease.PublishFile(id.ReceiptRelative, input, bytesDigest(receiptBytes), uint64(len(receiptBytes)), 0o644); err != nil {
					t.Fatal(err)
				}
				if existing {
					if err := activateInstallation(lease, id, receiptBytes); err != nil {
						t.Fatal(err)
					}
					if changedFile == "binary-parent" {
						moveBinary()
					}
				}
				if changedFile == "receipt" {
					if err := os.WriteFile(p.receipt, []byte("changed receipt"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				if err := activateInstallation(lease, id, receiptBytes); err == nil {
					t.Fatal("activation accepted changed final package files")
				}
				if !existing && lease.State() != "installing" {
					t.Fatal("failed publication opened admission")
				}
			})
		}
	}
}

func retirementFixture(t *testing.T) (string, Result) {
	t.Helper()
	root := canonicalTempDir(t)
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	installed, err := Install(source, filepath.Join(root, "prefix"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	return root, installed
}

func TestRetirementClosesAdmissionUntilExactRecovery(t *testing.T) {
	_, installed := retirementFixture(t)
	lease, err := AcquireBinaryUse(installed.Binary)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err := Uninstall(installed.Prefix, version.Version); err == nil {
		t.Fatal("live package was reclaimed")
	}
	if _, phase, err := packageretirement.Inspect(installed.Prefix, componentID, version.Version); err != nil || phase != "retiring" {
		t.Fatalf("busy retirement did not remain closed: %s %v", phase, err)
	}
	if next, err := AcquireBinaryUse(installed.Binary); err == nil {
		next.Close()
		t.Fatal("late use entered retirement")
	}
	if _, err := os.Stat(installed.Receipt); err != nil {
		t.Fatal("busy retirement lost its receipt")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(installed.Prefix, version.Version); err != nil {
		t.Fatal(err)
	}
	if _, phase, err := packageretirement.Inspect(installed.Prefix, componentID, version.Version); err != nil || phase != "retired" {
		t.Fatalf("retirement tombstone absent: %s %v", phase, err)
	}
	source, _ := os.Executable()
	if _, err := Install(source, installed.Prefix, version.Version); err == nil {
		t.Fatal("retired identity was resurrected")
	}
	if replay, err := Uninstall(installed.Prefix, version.Version); err != nil || replay.Changed {
		t.Fatalf("retirement replay: %+v %v", replay, err)
	}
}

func TestMaintenanceFencesStartAndPreservesOpenAdmission(t *testing.T) {
	_, installed := retirementFixture(t)
	use, err := AcquireBinaryUse(installed.Binary)
	if err != nil {
		t.Fatal(err)
	}
	if maintenance, err := acquireBinaryMaintenance(installed.Binary); err == nil {
		maintenance.Close()
		t.Fatal("maintenance bypassed a live consumer")
	}
	use.Close()
	maintenance, err := acquireBinaryMaintenance(installed.Binary)
	if err != nil {
		t.Fatal(err)
	}
	if use, err := AcquireBinaryUse(installed.Binary); err == nil {
		use.Close()
		t.Fatal("start bypassed exclusive maintenance")
	}
	maintenance.Close()
	if use, err := AcquireBinaryUse(installed.Binary); err != nil {
		t.Fatalf("unenrollment-style maintenance retired an open package: %v", err)
	} else {
		use.Close()
	}
}

func TestRetirementRecoversReceiptLastInterruption(t *testing.T) {
	for _, receiptRemoved := range []bool{false, true} {
		t.Run(map[bool]string{false: "binary-removed", true: "receipt-removed"}[receiptRemoved], func(t *testing.T) {
			_, installed := retirementFixture(t)
			id, _, err := packageretirement.Inspect(installed.Prefix, componentID, version.Version)
			if err != nil {
				t.Fatal(err)
			}
			lease, err := packageretirement.BeginRetirement(installed.Prefix, id)
			if err != nil {
				t.Fatal(err)
			}
			if err := lease.RemoveFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
				t.Fatal(err)
			}
			if receiptRemoved {
				data, err := lease.ReadFile(id.ReceiptRelative, 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				if err := lease.RemoveFile(id.ReceiptRelative, bytesDigest(data), uint64(len(data))); err != nil {
					t.Fatal(err)
				}
			}
			lease.Close()
			if removed, err := Uninstall(installed.Prefix, version.Version); err != nil || !removed.Changed {
				t.Fatalf("partial recovery failed: %+v %v", removed, err)
			}
		})
	}
}

func TestOrphanPendingStateBlocksReclamation(t *testing.T) {
	for _, store := range []string{"intents-v1", "outbox-v1"} {
		t.Run(store, func(t *testing.T) {
			root, installed := retirementFixture(t)
			dir := filepath.Join(root, "state", "symphony", "11111111-1111-4111-8111-111111111111", componentID, store)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			pending := filepath.Join(dir, "pending.json")
			if err := os.WriteFile(pending, []byte("unresolved"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Uninstall(installed.Prefix, version.Version); err == nil {
				t.Fatal("orphan audit work allowed package removal")
			}
			if _, phase, err := packageretirement.Inspect(installed.Prefix, componentID, version.Version); err != nil || phase != "open" {
				t.Fatalf("known pending work lost its recovery admission: %s %v", phase, err)
			}
			for _, path := range []string{pending, installed.Binary, installed.Receipt} {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("retained evidence lost: %s %v", path, err)
				}
			}
		})
	}
}

func TestLegacyAndForeignSourcesCannotAcquireCustody(t *testing.T) {
	root, installed := retirementFixture(t)
	foreign := filepath.Join(root, "foreign")
	if err := os.WriteFile(foreign, []byte("legacy bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(foreign, filepath.Join(root, "foreign-prefix"), version.Version); err == nil {
		t.Fatal("arbitrary source was assigned custody")
	}
	guard := filepath.Join(installed.Prefix, ".symphony-package-retirement", componentID, version.Version)
	if err := os.Rename(guard, guard+"-evidence-retained"); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(installed.Prefix, version.Version); err == nil {
		t.Fatal("unguarded legacy package was removed")
	}
	source, _ := os.Executable()
	if _, err := Install(source, installed.Prefix, version.Version); err == nil {
		t.Fatal("existing package was silently adopted")
	}
	if use, err := AcquireBinaryUse(installed.Binary); err == nil {
		use.Close()
		t.Fatal("legacy package was admitted")
	}
}

package lifecycle

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/packageretirement"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/packageuse"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/supervision"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

func managedRetirementFixture(t *testing.T) (string, Result) {
	t.Helper()
	home := canonicalTempDir(t)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	installed, err := Install(source, stavpaths.ScopeUser, false)
	if err != nil {
		t.Fatal(err)
	}
	return source, installed
}

// A real child owns the installed executable's lease; stdin EOF releases it.
func TestSTAVRetirementProcessHolder(t *testing.T) {
	if os.Getenv("SYMPHONY_STAV_RETIREMENT_HOLDER") != "1" {
		return
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	release, err := packageuse.Acquire(self)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	fmt.Println("lease-ready")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestManagedRetirementBlocksUseAndNewReferences(t *testing.T) {
	source, installed := managedRetirementFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, installed.Binary, "-test.run=^TestSTAVRetirementProcessHolder$")
	child.Env = append(os.Environ(), "SYMPHONY_STAV_RETIREMENT_HOLDER=1")
	input, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		_ = input.Close()
		if !waited {
			_ = child.Wait()
		}
	})
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || line != "lease-ready\n" {
		t.Fatalf("child did not acquire use: %q %v", line, err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, true); err == nil || !strings.Contains(err.Error(), "still live") {
		t.Fatalf("active use accepted: %v", err)
	}
	for _, path := range []string{installed.Binary, installed.InstallEvidence} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("busy retirement removed %s: %v", path, err)
		}
	}
	if release, err := packageuse.Acquire(installed.Binary); err == nil {
		release()
		t.Fatal("late use admitted after retirement intent")
	}
	if _, err := EnrollWithExecutable(stavpaths.ScopeUser, enrollmentTOPSID, uint64(os.Geteuid()), uint64(os.Getegid()), installed.Binary); err == nil {
		t.Fatal("late enrollment admitted after retirement")
	}
	spec := supervision.Spec{Scope: stavpaths.ScopeUser, TOPSID: enrollmentTOPSID, Binary: installed.Binary, UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
	if _, err := supervision.InstallCAS(spec, "absent"); err == nil {
		t.Fatal("late supervisor reference admitted")
	}
	if err := supervision.Start(supervision.Record{Binary: installed.Binary, Scope: stavpaths.ScopeUser}); err == nil {
		t.Fatal("late native restart admitted")
	}
	for _, name := range []string{"config", "state", "runtime", "Library"} {
		if _, err := os.Lstat(filepath.Join(os.Getenv("HOME"), name)); !os.IsNotExist(err) {
			t.Fatalf("refused late start created %s: %v", name, err)
		}
	}
	_ = input.Close()
	if err := child.Wait(); err != nil {
		t.Fatal(err)
	}
	waited = true
	if _, err := Uninstall(source, stavpaths.ScopeUser, false); err != nil {
		t.Fatal(err)
	}
	layout, _ := stavpaths.ResolveInstall(stavpaths.ScopeUser)
	_, state, err := packageretirement.Inspect(layout.Prefix, packageuse.Owner, version.Version)
	if err != nil || state != "retired" {
		t.Fatalf("missing durable completion: %s %v", state, err)
	}
	if _, err := Install(source, stavpaths.ScopeUser, true); err == nil {
		t.Fatal("retired installation reopened")
	}
}

func TestManagedRetirementRecoversAfterPayloadUnlink(t *testing.T) {
	source, installed := managedRetirementFixture(t)
	layout, _ := stavpaths.ResolveInstall(stavpaths.ScopeUser)
	identity, _, err := packageretirement.Inspect(layout.Prefix, packageuse.Owner, version.Version)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := packageretirement.BeginRetirement(layout.Prefix, identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.RemoveFile(identity.BinaryRelative, identity.BinaryDigest, identity.BinarySize); err != nil {
		guard.Close()
		t.Fatal(err)
	}
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	} // controller interruption before receipt removal
	if _, err := os.Stat(installed.InstallEvidence); err != nil {
		t.Fatal("receipt was not retained", err)
	}
	sentinel := filepath.Join(layout.Prefix, "unowned-user-file")
	if err := os.WriteFile(sentinel, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, false); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "preserve" {
		t.Fatal("recovery changed unrelated bytes")
	}
	if result, err := Uninstall(source, stavpaths.ScopeUser, true); err != nil || result.Changed {
		t.Fatalf("exact retirement retry: %+v %v", result, err)
	}
}

func TestRetirementRetainsOrphanedNativeDescriptor(t *testing.T) {
	source, installed := managedRetirementFixture(t)
	directory := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	descriptor := filepath.Join(directory, "io.github.quanux.symphony.stav."+enrollmentTOPSID+".plist")
	if err := os.WriteFile(descriptor, []byte(installed.Binary), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, true); err == nil || !strings.Contains(err.Error(), "still referenced") {
		t.Fatalf("orphaned descriptor ignored: %v", err)
	}
	if _, err := os.Stat(installed.Binary); err != nil {
		t.Fatal("referenced bytes removed", err)
	}
}

func TestRetirementRecoveryPreservesUnenrolledData(t *testing.T) {
	source, installed := managedRetirementFixture(t)
	if _, err := EnrollWithExecutable(stavpaths.ScopeUser, enrollmentTOPSID, uint64(os.Geteuid()), uint64(os.Getegid()), installed.Binary); err != nil {
		t.Fatal(err)
	}
	layout, err := stavpaths.ResolveInstance(stavpaths.ScopeUser, enrollmentTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(layout.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, true); err == nil || !strings.Contains(err.Error(), "still referenced") {
		t.Fatalf("enrollment did not retain package: %v", err)
	}
	if _, err := Unenroll(stavpaths.ScopeUser, enrollmentTOPSID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, false); err != nil {
		t.Fatal(err)
	}
	if after, err := os.ReadFile(layout.ConfigFile); err != nil || string(after) != string(before) {
		t.Fatal("retirement changed preserved configuration")
	}
	if info, err := os.Stat(layout.StateDir); err != nil || !info.IsDir() {
		t.Fatal("retirement removed user state", err)
	}
}

func TestManagedAdmissionRefusesMissingOrChangedReceipt(t *testing.T) {
	for _, mode := range []string{"missing", "changed"} {
		t.Run(mode, func(t *testing.T) {
			_, installed := managedRetirementFixture(t)
			if mode == "missing" {
				if err := os.Remove(installed.InstallEvidence); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(installed.InstallEvidence, []byte("{}"), 0644); err != nil {
				t.Fatal(err)
			}
			if release, err := packageuse.Acquire(installed.Binary); err == nil {
				release()
				t.Fatal("invalid receipt admitted")
			}
		})
	}
}

func TestMaintenanceExcludesStartsWithoutReopeningRetirement(t *testing.T) {
	source, installed := managedRetirementFixture(t)
	releaseUse, err := packageuse.Acquire(installed.Binary)
	if err != nil {
		t.Fatal(err)
	}
	if release, err := packageuse.AcquireMaintenance(installed.Binary); err == nil {
		release()
		t.Fatal("maintenance admitted a live service")
	}
	releaseUse()
	releaseMaintenance, err := packageuse.AcquireMaintenance(installed.Binary)
	if err != nil {
		t.Fatal(err)
	}
	if release, err := packageuse.Acquire(installed.Binary); err == nil {
		release()
		t.Fatal("service entered during destructive maintenance")
	}
	releaseMaintenance()
	releaseUse, err = packageuse.Acquire(installed.Binary)
	if err != nil {
		t.Fatal("ordinary maintenance closed admission permanently", err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, false); err == nil {
		t.Fatal("retirement ignored live use")
	}
	releaseUse()
	releaseMaintenance, err = packageuse.AcquireMaintenance(installed.Binary)
	if err != nil {
		t.Fatal(err)
	}
	releaseMaintenance()
	if release, err := packageuse.Acquire(installed.Binary); err == nil {
		release()
		t.Fatal("maintenance reopened pending retirement")
	}
}

func TestSTAVRetirementSelfController(t *testing.T) {
	if os.Getenv("SYMPHONY_STAV_RETIREMENT_SELF") != "1" {
		return
	}
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(source, stavpaths.ScopeUser, true); err == nil || !strings.Contains(err.Error(), "executing controller") {
		t.Fatalf("self unlink allowed: %v", err)
	}
}

func TestRetirementRetainsExecutingController(t *testing.T) {
	_, installed := managedRetirementFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, installed.Binary, "-test.run=^TestSTAVRetirementSelfController$")
	child.Env = append(os.Environ(), "SYMPHONY_STAV_RETIREMENT_SELF=1")
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("self controller: %s %v", out, err)
	}
	if _, err := os.Stat(installed.Binary); err != nil {
		t.Fatal("controller disappeared", err)
	}
	release, err := packageuse.Acquire(installed.Binary)
	if err != nil {
		t.Fatal("refused self retirement changed admission", err)
	}
	release()
}

func TestActivationRefusesReplacedPublicationPaths(t *testing.T) {
	for _, target := range []string{"payload", "receipt"} {
		t.Run(target, func(t *testing.T) {
			root := canonicalTempDir(t)
			payload, receipt := []byte("fixture payload"), []byte("fixture receipt")
			identity := packageretirement.Identity{Owner: packageuse.Owner, Version: version.Version, ReceiptDigest: digestBytes(receipt), BinaryDigest: digestBytes(payload), BinarySize: uint64(len(payload)), BinaryRelative: "libexec/owner", ReceiptRelative: "share/receipt"}
			guard, err := packageretirement.PrepareInstall(root, identity, true)
			if err != nil {
				t.Fatal(err)
			}
			defer guard.Close()
			if _, err := guard.PublishFile(identity.BinaryRelative, bytes.NewReader(payload), identity.BinaryDigest, identity.BinarySize, 0755); err != nil {
				t.Fatal(err)
			}
			if _, err := guard.PublishFile(identity.ReceiptRelative, bytes.NewReader(receipt), digestBytes(receipt), uint64(len(receipt)), 0644); err != nil {
				t.Fatal(err)
			}
			relative := identity.BinaryRelative
			if target == "receipt" {
				relative = identity.ReceiptRelative
			}
			path := filepath.Join(root, relative)
			if err := os.Rename(filepath.Dir(path), filepath.Dir(path)+"-moved"); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("administrator file"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := activateManagedPackage(guard, identity, receipt); err == nil {
				t.Fatal("changed publication opened admission")
			}
			if guard.State() != "installing" {
				t.Fatalf("failed activation changed state: %s", guard.State())
			}
			if actual, err := os.ReadFile(path); err != nil || string(actual) != "administrator file" {
				t.Fatal("activation changed administrator bytes", err)
			}
		})
	}
}

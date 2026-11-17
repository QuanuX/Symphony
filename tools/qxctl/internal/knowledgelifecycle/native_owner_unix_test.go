//go:build darwin || linux

package knowledgelifecycle

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNativeControllerIndependentCustody(t *testing.T) {
	root := resolvedTempDir(t)
	path := filepath.Join(root, "owner")
	data := []byte("exact controller fixture")
	writeTestFile(t, path, data, 0o700)
	binary := receiptV2File{Size: uint64(len(data)), Digest: taggedBytes(data)}
	copyPath, cleanup, err := retainNativeController([]string{path}, binary)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	a, _ := os.Stat(path)
	b, _ := os.Stat(copyPath)
	if os.SameFile(a, b) {
		t.Fatal("retained controller shares carrier inode")
	}
	if err := os.WriteFile(path, []byte("changed"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(copyPath)
	if err != nil || string(got) != string(data) {
		t.Fatal("carrier mutation changed retained controller", err)
	}
	if _, _, err := retainNativeController([]string{path}, binary); err == nil {
		t.Fatal("wrong bytes accepted")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(copyPath, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := retainNativeController([]string{link}, binary); err == nil {
		t.Fatal("symlink accepted")
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(copyPath, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := retainNativeController([]string{link}, binary); err == nil {
		t.Fatal("hardlink accepted")
	}
}

func nativeTestScope(t *testing.T, desired *DesiredState, action PlannedAction) *ExecutionScope {
	t.Helper()
	scope := fixtureExecutionScope(t)
	scope.TOPSID, scope.ProfileID = desired.TOPSID, desired.ProfileID
	desired.DesiredStateDigest = tagged("native desired " + action.Kind)
	scope.DesiredStateDigest = desired.DesiredStateDigest
	scope.Components = []string{action.ComponentID}
	scope.Effects = []ScopeEffect{{ComponentID: action.ComponentID, Kind: action.Kind, TargetStateDigest: action.TargetStateDigest, ExpectedArtifactDigests: action.ExpectedArtifactDigests, InstallRoot: desired.Components[0].InstallRoot}}
	scope.OwnershipEffects = nil
	for _, digest := range action.ExpectedArtifactDigests {
		dispositions := []string{"retained"}
		if action.Kind == "uninstall" {
			dispositions = []string{"release", "retiring"}
		}
		for _, disposition := range dispositions {
			scope.OwnershipEffects = append(scope.OwnershipEffects, ScopeOwnershipEffect{InstallRoot: desired.Components[0].InstallRoot, ComponentID: action.ComponentID, ReceiptDigest: digest, Disposition: disposition})
		}
	}
	resignScope(t, &scope)
	if err := scope.Validate(); err != nil {
		t.Fatal(err)
	}
	return &scope
}

// These are real native-owner executables built from this checkout. Every
// operation stays in disposable roots and no service or host integration is
// started. This test is mandatory on both supported Unix platforms.
func TestNativeOwnersSharedRootDispatch(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	binaries := resolvedTempDir(t)
	components := []string{"secure-identity-access-governance", "stav-append-authority", "accordare-stav-producer"}
	for _, component := range components {
		binary := nativeOwnerBinary(component)
		version := "0.1.0-dev"
		if component == "stav-append-authority" {
			version = "0.2.1"
		}
		cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-mod=readonly", "-trimpath", "-ldflags", "-X github.com/QuanuX/Symphony/modules/"+component+"/internal/version.Version="+version, "-o", filepath.Join(binaries, binary), "./cmd/"+binary)
		cmd.Dir = filepath.Join(repo, "modules", component)
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", component, err, out)
		}
	}
	for _, component := range components {
		t.Run(component, func(t *testing.T) {
			home := resolvedTempDir(t)
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
			t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
			t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
			staged := filepath.Join(resolvedTempDir(t), "stage")
			target := filepath.Join(resolvedTempDir(t), "target")
			version := "0.1.0-dev"
			if component == "stav-append-authority" {
				version = "0.2.1"
			}
			args := nativeOwnerArguments("install", component, staged, version)
			args = args[:len(args)-6] // standalone staging has no inherited authority
			controller := filepath.Join(binaries, nativeOwnerBinary(component))
			if out, err := exec.Command(controller, args...).CombinedOutput(); err != nil {
				t.Fatalf("stage real owner: %v\n%s", err, out)
			}
			candidates, err := scanReceiptRoot(staged)
			if err != nil || len(candidates) != 1 {
				t.Fatalf("staged receipt: %v %d", err, len(candidates))
			}
			var receipt receiptV2
			if err := json.Unmarshal(candidates[0].data, &receipt); err != nil {
				t.Fatal(err)
			}
			state := resolvedTempDir(t)
			executor, err := NewExecutor(state, testTOPSID, "default", []string{staged})
			if err != nil {
				t.Fatal(err)
			}
			if err := executor.SetNativeOwners([]string{component + "=" + controller}); err != nil {
				t.Fatal(err)
			}
			desired := DesiredState{TOPSID: testTOPSID, ProfileID: "default", Components: []DesiredComponent{{ComponentID: component, Presence: "present", InstallRoot: target, SelectedPackage: &PackageIdentity{PackageID: receipt.PackageID, Version: receipt.Version, ReceiptProtocol: receipt.Protocol, ReceiptDigest: receipt.ReceiptDigest}}}}
			action := PlannedAction{ActionID: "native-install", ComponentID: component, Kind: "install", TargetStateDigest: tagged("native install"), ExpectedArtifactDigests: []string{receipt.ReceiptDigest}}
			executor.SetExecutionScope(nativeTestScope(t, &desired, action))
			empty := Observation{TOPSID: testTOPSID, ProfileID: "default"}
			if _, err := executor.ReconcileOwnership(desired, empty); err != nil {
				t.Fatal(err)
			}
			// A native owner called without the actual delegated lock still refuses
			// a shared root, despite the retained claim being present.
			args = nativeOwnerArguments("install", component, target, version)
			if out, err := exec.Command(controller, args[:len(args)-6]...).CombinedOutput(); err == nil {
				t.Fatalf("standalone bypassed shared-root fence: %s", out)
			}
			result := executor.Execute(action, desired, empty)
			if result.Outcome != "committed" {
				t.Fatalf("native dispatch install: %+v", result)
			}
			observed, err := Observe(observationInput(target, time.Now()))
			if err != nil || len(observed.Components) != 1 || observed.Components[0].Packages[0].Integrity != "valid" {
				t.Fatalf("installed observation: %+v %v", observed, err)
			}
			// The parent must keep its exclusive lock after the delegated child
			// closes duplicate descriptors; otherwise another root mutator races.
			owner, _ := NewOwnershipStore(target, state, testTOPSID, "default")
			owner.executionScope = executor.executionScope
			locked, err := lockInstallRoot(target, false)
			if err != nil {
				t.Fatal(err)
			}
			nativeTarget := nativeOwnerTarget{receipt: receipt, receiptPath: candidates[0].relativePath, controller: controller}
			if err := dispatchNativeOwner(locked, owner, "install", nativeTarget); err != nil {
				locked.close()
				t.Fatal(err)
			}
			fd, err := unix.Open(filepath.Join(target, packageMutationLock), unix.O_RDWR|unix.O_NOFOLLOW, 0)
			if err != nil {
				locked.close()
				t.Fatal(err)
			}
			lockErr := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
			unix.Close(fd)
			locked.close()
			if lockErr == nil {
				t.Fatal("delegated child released parent root lock")
			}
			writeTestFile(t, filepath.Join(target, "user-sentinel"), []byte("preserve local data"), 0o600)
			other, err := NewOwnershipStore(target, state, testTOPSID, "another-profile")
			if err != nil {
				t.Fatal(err)
			}
			otherDesired := desired
			otherDesired.ProfileID = "another-profile"
			otherObservation := observed
			otherObservation.ProfileID = "another-profile"
			if _, err := other.Reconcile(otherDesired, otherObservation); err != nil {
				t.Fatal(err)
			}
			desired.Components[0].Presence, desired.Components[0].SelectedPackage = "absent", nil
			action.Kind, action.ActionID, action.TargetStateDigest = "uninstall", "native-uninstall", tagged("native uninstall")
			action.ExpectedBeforeDigest = &observed.Components[0].ObservationDigest
			executor.SetExecutionScope(nativeTestScope(t, &desired, action))
			if _, err := executor.ReconcileOwnership(desired, observed); err != nil {
				t.Fatal(err)
			}
			blocked := executor.Execute(action, desired, observed)
			if blocked.BlockerClass == nil || *blocked.BlockerClass != "dependency_wait" {
				t.Fatalf("foreign retained claim was not honored: %+v", blocked)
			}
			if _, err := os.Stat(filepath.Join(target, receipt.Files[0].Path)); err != nil {
				t.Fatal("foreign claim did not preserve bytes", err)
			}
			otherDesired.Components = append([]DesiredComponent(nil), desired.Components...)
			if _, err := other.Reconcile(otherDesired, otherObservation); err != nil {
				t.Fatal(err)
			}
			// Maintenance must see the real caller's HOME/XDG references. A
			// private staging HOME at dispatch would conceal this native blocker.
			descriptor := nativeTestReference(t, component, filepath.Join(target, receipt.Files[0].Path))
			blocked = executor.Execute(action, desired, observed)
			if blocked.BlockerClass == nil || *blocked.BlockerClass != "critical_state_unknown" || !strings.Contains(blocked.Detail, "referenc") && !strings.Contains(blocked.Detail, "enrolled") {
				t.Fatalf("caller reference was concealed from native owner: %+v", blocked)
			}
			if _, err := os.Stat(filepath.Join(target, receipt.Files[0].Path)); err != nil {
				t.Fatal("referenced package bytes removed", err)
			}
			if _, err := os.Stat(descriptor); err != nil {
				t.Fatal("native owner deleted caller reference", err)
			}
			if err := os.Remove(descriptor); err != nil {
				t.Fatal(err)
			}
			result = executor.Execute(action, desired, observed)
			if result.Outcome != "committed" {
				t.Fatalf("native dispatch uninstall: %+v", result)
			}
			if _, err := os.Stat(filepath.Join(target, receipt.Files[0].Path)); !os.IsNotExist(err) {
				t.Fatal("native owner retained executable", err)
			}
			if data, err := os.ReadFile(filepath.Join(target, "user-sentinel")); err != nil || string(data) != "preserve local data" {
				t.Fatal("native owner changed unrelated data", err)
			}
			snapshot, err := owner.Snapshot()
			if err != nil {
				t.Fatal(err)
			}
			for _, claim := range snapshot.Registry.Claims {
				if owner.ownsClaim(claim) {
					t.Fatal("completed retirement did not release own claim")
				}
			}
			if len(snapshot.Registry.Claims) != 1 || snapshot.Registry.Claims[0].ProfileID != "another-profile" {
				t.Fatal("native retirement altered another profile's claim")
			}
		})
	}
}

func nativeTestReference(t *testing.T, component, binary string) string {
	t.Helper()
	var path string
	if component == "accordare-stav-producer" {
		path = filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "symphony", testTOPSID, component, "config.json")
	} else {
		name := "stav"
		if component == "secure-identity-access-governance" {
			name = "ssiag"
		}
		if runtime.GOOS == "darwin" {
			path = filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "io.github.quanux.symphony."+name+"."+testTOPSID+".plist")
		} else {
			path = filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "systemd", "user", "symphony-"+name+"@"+testTOPSID+".service")
		}
	}
	writeTestFile(t, path, []byte("reference="+binary), 0o600)
	return path
}

func TestNativeOwnerOutputIsBounded(t *testing.T) {
	var output nativeBoundedOutput
	data := strings.Repeat("x", 16384)
	n, err := output.Write([]byte(data))
	if n != len(data) || err != nil || output.Len() != 4096 {
		t.Fatal("owner output bound failed")
	}
}

func TestNativeRecoveryRejectsReplacedRootLock(t *testing.T) {
	root := resolvedTempDir(t)
	locked, err := lockInstallRoot(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer locked.close()
	if err := validateNativeMutationLock(locked); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, packageMutationLock), filepath.Join(root, "retained-lock")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, packageMutationLock), []byte{}, 0o600)
	if err := validateNativeMutationLock(locked); err == nil {
		t.Fatal("replacement lock authorized tombstone recovery")
	}
}

func TestNativeOutcomeRejectsSharedEvidenceInodes(t *testing.T) {
	root := resolvedTempDir(t)
	path := filepath.Join(root, "state.json")
	writeTestFile(t, path, []byte("{}"), 0o600)
	if err := os.Link(path, filepath.Join(root, "shared-state")); err != nil {
		t.Fatal(err)
	}
	fd, err := openOrCreateAbsoluteDirectory(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if _, err := nativeReadAt(fd, "state.json", 8192); err == nil {
		t.Fatal("hardlinked retirement evidence was accepted")
	}
}

func TestNativeSourcesMustSurviveScopedRemoval(t *testing.T) {
	root, source := resolvedTempDir(t), resolvedTempDir(t)
	executor := Executor{executionScope: &ExecutionScope{Effects: []ScopeEffect{{InstallRoot: root}}}}
	if err := executor.validateNativeRetentionSources(source, []string{filepath.Join(source, "owner")}); err != nil {
		t.Fatal(err)
	}
	if err := executor.validateNativeRetentionSources(root, []string{filepath.Join(source, "owner")}); err == nil {
		t.Fatal("target accepted as its own rollback source")
	}
	if err := executor.validateNativeRetentionSources(source, []string{filepath.Join(root, "installed-owner")}); err == nil {
		t.Fatal("installed target accepted as durable controller")
	}
}

package knowledgelifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOwnershipNewRootEnforcesClaimsAndTwoProfileRetirement(t *testing.T) {
	stateA := resolvedTempDir(t)
	stateB := resolvedTempDir(t)
	installRoot := filepath.Join(resolvedTempDir(t), "shared")
	receipt := tagged("shared-receipt")
	desired := ownershipDesired("default", installRoot, receipt, "present")
	observation := ownershipObservation("default", installRoot, "example", nil)

	ownerA, err := NewOwnershipStore(installRoot, stateA, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	first, err := ownerA.Reconcile(desired, observation)
	if err != nil || !first.Changed || first.Snapshot.Registry == nil || first.Snapshot.Registry.EnforcementState != "enforced" ||
		len(first.Snapshot.Registry.Claims) != 1 || first.Snapshot.Registry.Claims[0].Disposition != "retained" {
		t.Fatalf("new root was not immediately protected: result=%+v err=%v", first, err)
	}
	fence, err := os.ReadFile(filepath.Join(installRoot, ownershipFenceRelative))
	if err != nil || !validOwnershipFence(fence) {
		t.Fatalf("new root did not publish the old-client compatibility fence: %v", err)
	}
	protocol, err := receiptProtocol(fence)
	if err != nil || protocol != OwnershipFenceProtocol || protocol == "symphony.knowledge.install-receipt.v1" ||
		protocol == "symphony.knowledge.install-receipt.v2" {
		t.Fatalf("compatibility fence would not reach an older client's unsupported-receipt blocker: protocol=%q err=%v", protocol, err)
	}

	observation = ownershipObservation("default", installRoot, "example", []string{receipt})
	if _, err := ownerA.Reconcile(desired, observation); err != nil {
		t.Fatal(err)
	}
	ownerB, err := NewOwnershipStore(installRoot, stateB, testTOPSID, "other")
	if err != nil {
		t.Fatal(err)
	}
	desiredB := ownershipDesired("other", installRoot, receipt, "present")
	observationB := ownershipObservation("other", installRoot, "example", []string{receipt})
	if _, err := ownerB.Reconcile(desiredB, observationB); err != nil {
		t.Fatal(err)
	}
	absentA := ownershipDesired("default", installRoot, "", "absent")
	retired, err := ownerA.Reconcile(absentA, observation)
	if err != nil {
		t.Fatal(err)
	}
	var retained, retiring int
	for _, claim := range retired.Snapshot.Registry.Claims {
		if claim.Disposition == "retained" {
			retained++
		}
		if claim.Disposition == "retiring" {
			retiring++
		}
	}
	if retained != 1 || retiring != 1 {
		t.Fatalf("two-profile claim accounting drifted: %+v", retired.Snapshot.Registry.Claims)
	}
	locked, err := lockInstallRoot(installRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ownershipReclaimAllowedAt(int(locked.root.Fd()), installRoot, ownerA, "example", receipt); err == nil {
		t.Fatal("another profile's retained claim did not block reclamation")
	}
	locked.close()

	absentB := ownershipDesired("other", installRoot, "", "absent")
	if _, err := ownerB.Reconcile(absentB, observationB); err != nil {
		t.Fatal(err)
	}
	locked, err = lockInstallRoot(installRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := ownershipReclaimAllowedAt(int(locked.root.Fd()), installRoot, ownerA, "example", receipt)
	if err != nil {
		t.Fatalf("unanimous retirement did not permit reclamation: %v", err)
	}
	if err := commitOwnershipReclaimedAt(int(locked.root.Fd()), registry, ownerA, receipt); err != nil {
		t.Fatal(err)
	}
	locked.close()
	final, err := ownerB.Snapshot()
	if err != nil || final.Registry == nil {
		t.Fatalf("read reclaimed ownership registry: snapshot=%+v err=%v", final, err)
	}
	for _, claim := range final.Registry.Claims {
		if claim.ReceiptDigest == receipt {
			t.Fatalf("reclaimed receipt retained a stale cross-domain claim: %+v", claim)
		}
	}
}

func TestOwnershipLegacyAdoptionAndExplicitRelease(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	installRoot := resolvedTempDir(t)
	receipt := tagged("legacy-receipt")
	store, err := NewOwnershipStore(installRoot, stateRoot, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	observed := ownershipObservation("default", installRoot, "example", []string{receipt})
	desired := ownershipDesired("default", installRoot, "", "absent")
	legacy, err := store.Reconcile(desired, observed)
	if err != nil || legacy.Snapshot.Registry == nil || legacy.Snapshot.Registry.EnforcementState != "adoption_required" {
		t.Fatalf("legacy root did not require adoption: result=%+v err=%v", legacy, err)
	}
	adopted, err := store.Adopt(legacy.Snapshot.Registry.OwnershipRegistryDigest)
	if err != nil || adopted.Snapshot.Registry == nil || adopted.Snapshot.Registry.EnforcementState != "enforced" {
		t.Fatalf("legacy root adoption failed: result=%+v err=%v", adopted, err)
	}
	legacyPresent := false
	for _, claim := range adopted.Snapshot.Registry.Claims {
		legacyPresent = legacyPresent || claim.ClaimKind == "legacy"
	}
	if !legacyPresent {
		t.Fatal("adoption silently released an unretained legacy package")
	}
	released, err := store.ReleaseLegacy(receipt, adopted.Snapshot.Registry.OwnershipRegistryDigest)
	if err != nil || !containsString(released.Snapshot.Registry.ReleasedReceiptDigests, receipt) {
		t.Fatalf("legacy release failed: result=%+v err=%v", released, err)
	}
}

func TestOwnershipUnexpectedInventoryReopensAdoption(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	installRoot := filepath.Join(resolvedTempDir(t), "fresh")
	store, err := NewOwnershipStore(installRoot, stateRoot, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	empty := ownershipDesired("default", installRoot, "", "absent")
	if _, err := store.Reconcile(empty, ownershipObservation("default", installRoot, "example", nil)); err != nil {
		t.Fatal(err)
	}
	unexpected := tagged("old-writer-package")
	result, err := store.Reconcile(empty, ownershipObservation("default", installRoot, "unmanaged", []string{unexpected}))
	if err != nil || result.Snapshot.Registry == nil || result.Snapshot.Registry.EnforcementState != "adoption_required" {
		t.Fatalf("unexpected package did not reopen adoption: result=%+v err=%v", result, err)
	}
	found := false
	for _, claim := range result.Snapshot.Registry.Claims {
		found = found || claim.ClaimKind == "legacy" && claim.ReceiptDigest == unexpected
	}
	if !found {
		t.Fatal("unexpected package was not conservatively preserved")
	}
}

func TestOwnershipReconcileHealsDurableRemovalBeforeRegistryCommit(t *testing.T) {
	stateA := resolvedTempDir(t)
	stateB := resolvedTempDir(t)
	installRoot := filepath.Join(resolvedTempDir(t), "shared")
	receipt := tagged("interrupted-reclamation")
	presentA := ownershipDesired("default", installRoot, receipt, "present")
	presentB := ownershipDesired("other", installRoot, receipt, "present")
	observedA := ownershipObservation("default", installRoot, "example", []string{receipt})
	observedB := ownershipObservation("other", installRoot, "example", []string{receipt})
	ownerA, err := NewOwnershipStore(installRoot, stateA, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	ownerB, err := NewOwnershipStore(installRoot, stateB, testTOPSID, "other")
	if err != nil {
		t.Fatal(err)
	}
	// Seed the fresh root before presenting inventory so it is qxctl-owned,
	// then retain and independently release the same exact receipt.
	if _, err := ownerA.Reconcile(presentA, ownershipObservation("default", installRoot, "example", nil)); err != nil {
		t.Fatal(err)
	}
	if _, err := ownerA.Reconcile(presentA, observedA); err != nil {
		t.Fatal(err)
	}
	if _, err := ownerB.Reconcile(presentB, observedB); err != nil {
		t.Fatal(err)
	}
	if _, err := ownerA.Reconcile(ownershipDesired("default", installRoot, "", "absent"), observedA); err != nil {
		t.Fatal(err)
	}
	if _, err := ownerB.Reconcile(ownershipDesired("other", installRoot, "", "absent"), observedB); err != nil {
		t.Fatal(err)
	}

	// Model the crash boundary where files and receipt are already durable as
	// absent but the prior registry generation still contains retiring claims.
	healed, err := ownerA.Reconcile(
		ownershipDesired("default", installRoot, "", "absent"),
		ownershipObservation("default", installRoot, "example", nil))
	if err != nil || healed.Snapshot.Registry == nil {
		t.Fatalf("reconcile did not heal interrupted reclamation: result=%+v err=%v", healed, err)
	}
	for _, claim := range healed.Snapshot.Registry.Claims {
		if claim.ReceiptDigest == receipt && claim.Disposition == "retiring" {
			t.Fatalf("stale retiring claim survived absent package recovery: %+v", claim)
		}
	}
}

func TestOwnershipRegistryRefusesSymlink(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	installRoot := resolvedTempDir(t)
	target := filepath.Join(resolvedTempDir(t), "target.json")
	if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(installRoot, ownershipRegistryFile)); err != nil {
		t.Fatal(err)
	}
	store, err := NewOwnershipStore(installRoot, stateRoot, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(); err == nil {
		t.Fatal("symlinked ownership registry was accepted")
	}
}

func TestOwnershipFenceIsHiddenFromNewObservationAndFailsClosedWhenDamaged(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	installRoot := filepath.Join(resolvedTempDir(t), "fenced")
	store, err := NewOwnershipStore(installRoot, stateRoot, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	empty := ownershipDesired("default", installRoot, "", "absent")
	if _, err := store.Reconcile(empty, ownershipObservation("default", installRoot, "example", nil)); err != nil {
		t.Fatal(err)
	}
	observed, err := Observe(observationInput(installRoot, time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)))
	if err != nil || len(observed.Components) != 0 || len(observed.UnknownPackages) != 0 {
		t.Fatalf("ownership-aware observation did not consume its exact fence: observation=%+v err=%v", observed, err)
	}
	if err := os.Remove(filepath.Join(installRoot, ownershipRegistryFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(); err == nil {
		t.Fatal("orphaned compatibility fence was misreported as an unmanaged root")
	}
	if _, err := store.Reconcile(empty, ownershipObservation("default", installRoot, "example", nil)); err != nil {
		t.Fatalf("reconcile did not repair the orphaned compatibility fence: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installRoot, ownershipFenceRelative), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(); err == nil {
		t.Fatal("damaged ownership compatibility fence was accepted")
	}
	damaged, err := Observe(observationInput(installRoot, time.Date(2026, 8, 12, 12, 0, 1, 0, time.UTC)))
	if err != nil || len(damaged.UnknownPackages) != 1 || !damaged.UnknownPackages[0].Preserved {
		t.Fatalf("damaged fence did not become preserved critical evidence: observation=%+v err=%v", damaged, err)
	}
}

func TestReceiptV2CannotOwnLifecycleControlOrReceiptPaths(t *testing.T) {
	for _, ownedPath := range []string{
		packageMutationLock,
		ownershipRegistryFile,
		ownershipFenceRelative,
		"share/symphony/receipts/another/1/install-receipt.json",
		".symphony-future-root-control",
	} {
		t.Run(ownedPath, func(t *testing.T) {
			receipt := receiptV2{
				Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2,
				ComponentID: "example", ComponentKind: "module", ModuleID: "example",
				PackageID: "example", Version: "1.0.0", InstallScope: "prefix", PrefixMode: "installation_prefix",
				Files: []receiptV2File{{Path: ownedPath, Kind: "regular", Size: 1, Digest: tagged("owned")}},
				EntryPoints: []receiptV2EntryPoint{{
					EntryPointID: "descriptor", Kind: "descriptor", Path: ownedPath, Protocols: []string{"example.v1"},
				}},
				ProvidesCapabilities: []string{}, RequiresCapabilities: []string{},
				CompatibleReceptors: []string{}, PlatformRequirements: []receiptV2Platform{},
			}
			receipt.ReceiptDigest = receiptV2Digest(t, receipt)
			candidate := receiptCandidate{root: resolvedTempDir(t), module: "example", version: "1.0.0"}
			if err := validateReceiptV2(candidate, receipt); err == nil {
				t.Fatalf("receipt-v2 acquired reserved root-control path %q", ownedPath)
			}
		})
	}
}

func ownershipDesired(profileID, installRoot, receipt, presence string) DesiredState {
	component := DesiredComponent{ComponentID: "example", Presence: presence, InstallRoot: installRoot}
	if receipt != "" {
		component.SelectedPackage = &PackageIdentity{
			PackageID: "example-package", Version: "1.0.0",
			ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: receipt,
		}
	}
	return DesiredState{TOPSID: testTOPSID, ProfileID: profileID, Components: []DesiredComponent{component}}
}

func ownershipObservation(profileID, installRoot, componentID string, receipts []string) Observation {
	packages := make([]ObservedPackage, 0, len(receipts))
	for _, receipt := range receipts {
		packages = append(packages, ObservedPackage{
			PackageID: "example-package", Version: "1.0.0", InstallRoot: installRoot,
			ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: receipt,
			Integrity: "valid", EntryPointsValidated: true,
		})
	}
	components := make([]ObservedComponent, 0)
	if len(packages) != 0 {
		components = append(components, ObservedComponent{ComponentID: componentID, Packages: packages})
	}
	return Observation{TOPSID: testTOPSID, ProfileID: profileID, Components: components}
}

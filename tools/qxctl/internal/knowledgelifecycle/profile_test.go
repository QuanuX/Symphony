package knowledgelifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func tagged(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func profileInput(root string) ProfileInput {
	return ProfileInput{
		Protocol: ProfileInputProtocol, FormatVersion: 1, ProfileID: "default", TOPSID: testTOPSID,
		ConfiguredRoots: []string{root}, BootMode: "report",
		Components: []DesiredComponent{{
			ComponentID: "example", ComponentKind: "module", ModuleID: "example",
			Presence: "present", Required: true, InstallScope: "prefix", InstallRoot: root,
			SelectedPackage: &PackageIdentity{
				PackageID: "example", Version: "1.0.0",
				ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: tagged("receipt"),
			},
			Activation: "inactive", Docking: Docking{Disposition: "undocked"},
			Dependencies: []Dependency{}, Compatibility: Compatibility{
				RequiredCapabilities: []string{}, PlatformRequirements: []string{},
			}, Extensions: []Extension{},
		}},
		Extensions: []Extension{},
	}
}

func TestProfileStoreCASIdempotencyAndRemoval(t *testing.T) {
	stateRoot := t.TempDir()
	installRoot := t.TempDir()
	store, err := NewStore(stateRoot, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	input := profileInput(installRoot)
	first, changed, err := store.Set(input, "absent")
	if err != nil || !changed || first.Generation != 1 || first.PreviousProfileDigest != nil ||
		first.DesiredState.PreviousDesiredStateDigest != nil {
		t.Fatalf("first profile set failed: changed=%t profile=%+v err=%v", changed, first, err)
	}
	if first.ProfileDigest == first.DesiredState.DesiredStateDigest {
		t.Fatal("profile and desired-state digests unexpectedly alias")
	}

	retried, changed, err := store.Set(input, "absent")
	if err != nil || changed || retried.ProfileDigest != first.ProfileDigest || retried.Generation != 1 {
		t.Fatalf("semantic retry was not a stable no-op: changed=%t profile=%+v err=%v", changed, retried, err)
	}

	modified := profileInput(installRoot)
	modified.BootMode = "apply-compatible"
	if _, _, err := store.Set(modified, tagged("stale")); err == nil {
		t.Fatal("stale profile compare-and-swap unexpectedly succeeded")
	}
	second, changed, err := store.Set(modified, first.ProfileDigest)
	if err != nil || !changed || second.Generation != 2 || second.PreviousProfileDigest == nil ||
		*second.PreviousProfileDigest != first.ProfileDigest || second.DesiredState.PreviousDesiredStateDigest == nil ||
		*second.DesiredState.PreviousDesiredStateDigest != first.DesiredState.DesiredStateDigest {
		t.Fatalf("linked profile update failed: changed=%t profile=%+v err=%v", changed, second, err)
	}

	snapshot, err := store.Snapshot("default")
	if err != nil || !snapshot.Exists || snapshot.Profile.ProfileDigest != second.ProfileDigest {
		t.Fatalf("profile snapshot mismatch: %+v err=%v", snapshot, err)
	}
	listed, err := store.List()
	if err != nil || len(listed.Profiles) != 1 || !taggedDigest(listed.ListDigest) {
		t.Fatalf("profile list mismatch: %+v err=%v", listed, err)
	}
	if _, err := store.Remove("default", first.ProfileDigest); err == nil {
		t.Fatal("stale profile removal unexpectedly succeeded")
	}
	removed, err := store.Remove("default", second.ProfileDigest)
	if err != nil || !removed {
		t.Fatalf("profile removal failed: removed=%t err=%v", removed, err)
	}
	removed, err = store.Remove("default", second.ProfileDigest)
	if err != nil || removed {
		t.Fatalf("profile removal retry was not idempotent: removed=%t err=%v", removed, err)
	}
}

func TestProfileStoreGuardedMutationAndSharedLease(t *testing.T) {
	stateRoot := t.TempDir()
	installRoot := t.TempDir()
	store, err := NewStore(stateRoot, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	input := profileInput(installRoot)
	first, _, err := store.Set(input, "absent")
	if err != nil {
		t.Fatal(err)
	}

	modified := profileInput(installRoot)
	modified.BootMode = "apply-compatible"
	guarded := false
	if _, _, err := store.SetGuarded(modified, first.ProfileDigest, func(current Profile, exists bool) error {
		guarded = exists && current.ProfileDigest == first.ProfileDigest
		return errors.New("blocked by resource ownership")
	}); err == nil || !guarded {
		t.Fatalf("guarded update was not rejected against the exact current profile: guarded=%t err=%v", guarded, err)
	}
	snapshot, err := store.Snapshot("default")
	if err != nil || snapshot.Profile.ProfileDigest != first.ProfileDigest {
		t.Fatalf("rejected guarded update changed the profile: %+v err=%v", snapshot, err)
	}

	leased := false
	err = store.WithProfileSnapshot("default", func(current Profile) error {
		leased = current.ProfileDigest == first.ProfileDigest
		if _, _, updateErr := store.SetGuarded(modified, first.ProfileDigest, nil); updateErr == nil ||
			!strings.Contains(updateErr.Error(), "store is busy") {
			t.Fatalf("exclusive update entered during shared lease: %v", updateErr)
		}
		return nil
	})
	if err != nil || !leased {
		t.Fatalf("shared profile lease failed: leased=%t err=%v", leased, err)
	}

	removeGuarded := false
	if _, err := store.RemoveGuarded("default", first.ProfileDigest, func(Profile) error {
		removeGuarded = true
		return errors.New("claims remain")
	}); err == nil || !removeGuarded {
		t.Fatalf("guarded removal was not rejected: guarded=%t err=%v", removeGuarded, err)
	}
}

func TestProfileInputRejectsUnboundRootAndDigestDrift(t *testing.T) {
	input := profileInput(t.TempDir())
	input.Components[0].InstallRoot = t.TempDir()
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeProfileInput(encoded); err == nil {
		t.Fatal("component outside configured roots was accepted")
	}

	input = profileInput(t.TempDir())
	input.Extensions = []Extension{{
		ExtensionID: "example", ExtensionVersion: "1", Payload: map[string]any{"enabled": true},
		PayloadDigest: tagged("wrong"),
	}}
	encoded, _ = json.Marshal(input)
	if _, err := DecodeProfileInput(encoded); err == nil {
		t.Fatal("extension payload digest drift was accepted")
	}

	input = profileInput(t.TempDir())
	input.Components[0].Dependencies = nil
	encoded, _ = json.Marshal(input)
	if _, err := DecodeProfileInput(encoded); err == nil {
		t.Fatal("null required dependency collection was accepted")
	}
}

func TestProfileListRejectsFilenameIdentityDrift(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	store, err := NewStore(stateRoot, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	input := profileInput(resolvedTempDir(t))
	if _, _, err := store.Set(input, "absent"); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(stateRoot, "symphony", testTOPSID, "qxctl/knowledge/lifecycle/profiles")
	if err := os.Rename(
		filepath.Join(directory, profileFileName("default")),
		filepath.Join(directory, profileFileName("different")),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(); err == nil {
		t.Fatal("profile filename and embedded identity drift was accepted")
	}
}

func TestObservationV2StableInventoryAndIntegrity(t *testing.T) {
	root := resolvedTempDir(t)
	ownedPath := "lib/example/data.bin"
	ownedData := []byte("content-addressed example\n")
	writeTestFile(t, filepath.Join(root, ownedPath), ownedData, 0o600)
	receipt := receiptV2{
		Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2,
		ComponentID: "example", ComponentKind: "module", ModuleID: "example",
		PackageID: "example", Version: "1.0.0", InstallScope: "prefix", PrefixMode: "installation_prefix",
		Files:                []receiptV2File{{Path: ownedPath, Kind: "regular", Size: uint64(len(ownedData)), Digest: taggedBytes(ownedData)}},
		EntryPoints:          []receiptV2EntryPoint{{EntryPointID: "descriptor", Kind: "descriptor", Path: ownedPath, Protocols: []string{"example.v1"}}},
		ProvidesCapabilities: []string{"example-capability"}, RequiresCapabilities: []string{},
		CompatibleReceptors: []string{}, PlatformRequirements: []receiptV2Platform{},
	}
	receipt.ReceiptDigest = receiptV2Digest(t, receipt)
	receiptPath := filepath.Join(root, "share/symphony/receipts/example/1.0.0/install-receipt.json")
	encoded, _ := json.Marshal(receipt)
	writeTestFile(t, receiptPath, encoded, 0o600)

	first, err := Observe(observationInput(root, time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Components) != 1 || len(first.UnknownPackages) != 0 ||
		first.Components[0].Packages[0].Integrity != "valid" || !first.Components[0].Packages[0].EntryPointsValidated {
		t.Fatalf("valid v2 receipt was not observed exactly: %+v", first)
	}
	second, err := Observe(observationInput(root, time.Date(2026, 8, 4, 16, 1, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if first.ObservationDigest == second.ObservationDigest {
		t.Fatal("timestamp refresh did not change document evidence")
	}
	firstStable, _ := StableInventoryDigest(first)
	secondStable, _ := StableInventoryDigest(second)
	if firstStable != secondStable {
		t.Fatalf("timestamp-only refresh changed stable inventory: %s != %s", firstStable, secondStable)
	}

	writeTestFile(t, filepath.Join(root, ownedPath), []byte("drift"), 0o600)
	drifted, err := Observe(observationInput(root, time.Date(2026, 8, 4, 16, 2, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if drifted.Components[0].Packages[0].Integrity != "invalid" {
		t.Fatal("receipt-owned file drift was not reported as invalid integrity")
	}
}

func TestObservationV1AdapterAndUnsupportedPreservation(t *testing.T) {
	root := resolvedTempDir(t)
	version := "0.1.0-dev"
	base := "share/doc/symphony/skvi-engine/" + version + "/"
	license := "share/licenses/symphony-skvi-engine/" + version + "/"
	paths := []string{
		"libexec/symphony/skvi-engine/" + version + "/symphony-skvi",
		"share/symphony/receipts/skvi-engine/" + version + "/install-receipt.json",
		base + "INTENT.md", base + "MANIFEST.md", base + "INSTALL.md", base + "SKILL.md", base + "SPEC.md",
		license + "LICENSE-AGPL-3.0", license + "nlohmann-json-LICENSE.MIT",
	}
	for _, path := range paths {
		if filepath.Base(path) == "install-receipt.json" {
			continue
		}
		mode := os.FileMode(0o600)
		if filepath.Base(path) == "symphony-skvi" {
			mode = 0o700
		}
		writeTestFile(t, filepath.Join(root, path), []byte("fixture\n"), mode)
	}
	receipt := receiptV1{
		Protocol: "symphony.knowledge.install-receipt.v1", ModuleID: "skvi-engine", Version: version,
		InstallScope: "prefix", PrefixMode: "installation_prefix", State: "installed_undocked",
		Active: false, Files: paths,
	}
	encoded, _ := json.Marshal(receipt)
	writeTestFile(t, filepath.Join(root, paths[1]), encoded, 0o600)

	observed, err := Observe(observationInput(root, time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if len(observed.Components) != 1 || observed.Components[0].ComponentID != "skvi-engine" ||
		observed.Components[0].Packages[0].ReceiptProtocol != "symphony.knowledge.install-receipt.v1" {
		t.Fatalf("v1 adapter did not project the exact known identity: %+v", observed)
	}

	unsupportedPath := filepath.Join(root, "share/symphony/receipts/future/9/install-receipt.json")
	writeTestFile(t, unsupportedPath, []byte(`{"protocol":"symphony.knowledge.install-receipt.v9"}`), 0o600)
	preserved, err := Observe(observationInput(root, time.Date(2026, 8, 4, 16, 1, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if len(preserved.UnknownPackages) != 1 || preserved.UnknownPackages[0].Reason != "unsupported_protocol" ||
		!preserved.UnknownPackages[0].Preserved {
		t.Fatalf("unsupported receipt was not preserved: %+v", preserved.UnknownPackages)
	}
}

func TestObservationTreatsFutureAbsentRootAsEmptyEvidence(t *testing.T) {
	root := filepath.Join(resolvedTempDir(t), "future-prefix")
	observed, err := Observe(observationInput(root, time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if len(observed.Components) != 0 || len(observed.UnknownPackages) != 0 ||
		len(observed.ConfiguredRoots) != 1 || observed.ConfiguredRoots[0] != root {
		t.Fatalf("absent future root was not retained as empty configured evidence: %+v", observed)
	}
}

func TestOverlayDockingPresenceRecomputesEvidenceAndRejectsAmbiguity(t *testing.T) {
	root := resolvedTempDir(t)
	observed, err := Observe(observationInput(root, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	observed.Components = []ObservedComponent{{
		ComponentID: "ssfv-engine", ComponentKind: "vector_engine", ModuleID: "ssfv-engine",
		VectorID: stringPointer("ssfv"), EngineID: stringPointer("symphony-ssfv"),
		Packages: []ObservedPackage{}, Activation: "inactive", Docking: "undocked",
		Capabilities: []string{}, PlatformCompatibility: "compatible",
	}}
	observed.Components[0].ObservationDigest, err = componentObservationDigest(observed.Components[0])
	if err != nil {
		t.Fatal(err)
	}
	observed.ObservationDigest, err = observationDigest(observed)
	if err != nil {
		t.Fatal(err)
	}
	beforeComponent := observed.Components[0].ObservationDigest
	beforeDocument := observed.ObservationDigest
	docked, err := OverlayDockingPresence(observed, []DockingPresence{{
		ComponentID: "ssfv-engine", Disposition: "docked", ReceptorID: "maestro-primary",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if docked.Components[0].Docking != "docked" || docked.Components[0].ReceptorID == nil ||
		*docked.Components[0].ReceptorID != "maestro-primary" ||
		docked.Components[0].ObservationDigest == beforeComponent || docked.ObservationDigest == beforeDocument {
		t.Fatalf("Maestro presence was not incorporated into evidence: %+v", docked.Components[0])
	}
	undocked, err := OverlayDockingPresence(docked, []DockingPresence{{
		ComponentID: "ssfv-engine", Disposition: "undocked", ReceptorID: "maestro-primary",
	}})
	if err != nil || undocked.Components[0].Docking != "undocked" || undocked.Components[0].ReceptorID != nil {
		t.Fatalf("Maestro tombstone did not clear receptor presence: %+v err=%v", undocked.Components[0], err)
	}
	if _, err := OverlayDockingPresence(observed, []DockingPresence{
		{ComponentID: "ssfv-engine", Disposition: "docked", ReceptorID: "maestro-primary"},
		{ComponentID: "ssfv-engine", Disposition: "docked", ReceptorID: "maestro-secondary"},
	}); err == nil {
		t.Fatal("ambiguous multi-receptor presence was accepted")
	}
}

func observationInput(root string, observedAt time.Time) ObservationInput {
	return ObservationInput{
		ProfileID: "default", TOPSID: testTOPSID, ConfiguredRoots: []string{root},
		SelectedReceipts:     map[string]string{},
		QxctlIdentity:        Identity{ComponentID: "qxctl", Version: "qxctl-dev", ExecutableDigest: tagged("qxctl")},
		ProviderAvailability: []ProviderAvailability{{ProviderID: "ssiag", Available: true}},
		ObservedAt:           observedAt,
	}
}

func receiptV2Digest(t *testing.T, receipt receiptV2) string {
	t.Helper()
	receipt.ReceiptDigest = ""
	value, err := objectWithout(mustJSON(receipt), "receipt_digest")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := digestValue(value)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func taggedBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func writeTestFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func resolvedTempDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

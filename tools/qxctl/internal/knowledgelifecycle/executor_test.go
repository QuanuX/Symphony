package knowledgelifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type recordingDockingAdapter struct {
	calls     int
	component DockingComponentEvidence
}

type recordingBindingAdapter struct {
	calls int
}

func (adapter *recordingBindingAdapter) Handles(componentID string) bool {
	return componentID == "skvi-engine" || componentID == "knowledge-session-coordinator"
}

func (adapter *recordingBindingAdapter) ExecuteBinding(
	_ PlannedAction,
	_ DesiredComponent,
	_ bool,
	_ ObservedComponent,
	_ bool,
	_ Observation,
) (string, string, []string, error) {
	adapter.calls++
	return "committed", "test binding committed", []string{tagged("binding-registry")}, nil
}

func (adapter *recordingDockingAdapter) ExecuteDocking(
	_ PlannedAction,
	component DockingComponentEvidence,
) (string, string, []string, error) {
	adapter.calls++
	adapter.component = component
	return "committed", "test Maestro presence committed", []string{tagged("maestro-registry")}, nil
}

func TestRuntimeStoreCASAndSemanticRetry(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	store, err := NewRuntimeStore(stateRoot, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	selected := tagged("runtime-receipt")
	first, changed, err := store.Mutate("example", "select", &selected, "absent")
	if err != nil || !changed || first.Generation != 1 || len(first.Components) != 1 {
		t.Fatalf("runtime selection failed: changed=%t state=%+v err=%v", changed, first, err)
	}
	retried, changed, err := store.Mutate("example", "select", &selected, first.RuntimeStateDigest)
	if err != nil || changed || retried.RuntimeStateDigest != first.RuntimeStateDigest {
		t.Fatalf("runtime semantic retry drifted: changed=%t state=%+v err=%v", changed, retried, err)
	}
	if _, _, err := store.Mutate("example", "activate", nil, tagged("stale")); err == nil {
		t.Fatal("stale runtime compare-and-swap succeeded")
	}
	active, changed, err := store.Mutate("example", "activate", nil, first.RuntimeStateDigest)
	if err != nil || !changed || active.Generation != 2 || active.Components[0].Activation != "active" {
		t.Fatalf("runtime activation failed: changed=%t state=%+v err=%v", changed, active, err)
	}
	if _, _, err := store.Mutate("example", "deselect", nil, active.RuntimeStateDigest); err == nil {
		t.Fatal("active runtime component was deselected")
	}
}

func TestExecutorInstallsAndUninstallsV2WithRollbackProof(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	stagedRoot := resolvedTempDir(t)
	targetRoot := filepath.Join(resolvedTempDir(t), "installed")
	ownedPath := "lib/example/payload.bin"
	ownedData := []byte("immutable staged package\n")
	writeTestFile(t, filepath.Join(stagedRoot, ownedPath), ownedData, 0o600)
	receipt := receiptV2{
		Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2,
		ComponentID: "example", ComponentKind: "module", ModuleID: "example",
		PackageID: "example-package", Version: "2.0.0", InstallScope: "prefix", PrefixMode: "installation_prefix",
		Files:       []receiptV2File{{Path: ownedPath, Kind: "regular", Size: uint64(len(ownedData)), Digest: taggedBytes(ownedData)}},
		EntryPoints: []receiptV2EntryPoint{}, ProvidesCapabilities: []string{}, RequiresCapabilities: []string{},
		CompatibleReceptors: []string{}, PlatformRequirements: []receiptV2Platform{},
	}
	receipt.ReceiptDigest = receiptV2Digest(t, receipt)
	receiptRelative := "share/symphony/receipts/example/2.0.0/install-receipt.json"
	encoded, _ := json.Marshal(receipt)
	writeTestFile(t, filepath.Join(stagedRoot, receiptRelative), encoded, 0o600)

	executor, err := NewExecutor(stateRoot, testTOPSID, "default", []string{stagedRoot})
	if err != nil {
		t.Fatal(err)
	}
	available, err := executor.AvailableArtifactDigests()
	if err != nil || len(available) != 1 || available[0] != receipt.ReceiptDigest {
		t.Fatalf("staged receipt was not available: %v err=%v", available, err)
	}
	selected := PackageIdentity{
		PackageID: receipt.PackageID, Version: receipt.Version,
		ReceiptProtocol: receipt.Protocol, ReceiptDigest: receipt.ReceiptDigest,
	}
	desired := DesiredState{TOPSID: testTOPSID, ProfileID: "default", Components: []DesiredComponent{{
		ComponentID: "example", Presence: "present", InstallRoot: targetRoot, SelectedPackage: &selected,
	}}}
	install := PlannedAction{
		ActionID: "lifecycle-action:install", ComponentID: "example", Kind: "install", Direction: "forward",
		TargetStateDigest: tagged("install-target"), ExpectedArtifactDigests: []string{receipt.ReceiptDigest},
		ExpectedEvidence: []string{}, PrerequisiteActionIDs: []string{}, Disposition: "waiting", Blockers: []Blocker{},
	}
	unclaimed := executor.Execute(install, desired, Observation{})
	if unclaimed.Outcome != "failed" || unclaimed.BlockerClass == nil || *unclaimed.BlockerClass != "dependency_wait" {
		t.Fatalf("package install bypassed the exact retained ownership claim: %+v", unclaimed)
	}
	if _, err := executor.ReconcileOwnership(desired, Observation{TOPSID: testTOPSID, ProfileID: "default"}); err != nil {
		t.Fatal(err)
	}
	result := executor.Execute(install, desired, Observation{})
	if result.Outcome != "committed" || !taggedDigest(result.EvidenceDigest) {
		t.Fatalf("install failed: %+v", result)
	}
	observed, err := Observe(observationInput(targetRoot, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)))
	if err != nil || len(observed.Components) != 1 || observed.Components[0].Packages[0].Integrity != "valid" {
		t.Fatalf("installed package did not verify: %+v err=%v", observed, err)
	}
	if retried := executor.Execute(install, desired, observed); retried.Outcome != "already_applied" {
		t.Fatalf("idempotent install retry failed: %+v", retried)
	}
	runtimeStore, err := NewRuntimeStore(stateRoot, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	runtimeSelected, changed, err := runtimeStore.Mutate("example", "select", &receipt.ReceiptDigest, "absent")
	if err != nil || !changed {
		t.Fatalf("runtime selection failed: changed=%t err=%v", changed, err)
	}
	runtimeActive, changed, err := runtimeStore.Mutate("example", "activate", nil, runtimeSelected.RuntimeStateDigest)
	if err != nil || !changed {
		t.Fatalf("runtime activation failed: changed=%t err=%v", changed, err)
	}
	input := observationInput(targetRoot, time.Date(2026, 8, 10, 12, 1, 0, 0, time.UTC))
	input.RuntimeState = &runtimeActive
	runtimeObserved, err := Observe(input)
	if err != nil || runtimeObserved.Components[0].SelectedPackageDigest == nil ||
		*runtimeObserved.Components[0].SelectedPackageDigest != receipt.ReceiptDigest ||
		runtimeObserved.Components[0].Activation != "active" {
		t.Fatalf("generic runtime state was not projected into observation: %+v err=%v", runtimeObserved, err)
	}

	absent := DesiredState{TOPSID: testTOPSID, ProfileID: "default", Components: []DesiredComponent{{
		ComponentID: "example", Presence: "absent", InstallRoot: targetRoot,
	}}}
	if _, err := executor.ReconcileOwnership(absent, observed); err != nil {
		t.Fatal(err)
	}
	uninstall := PlannedAction{
		ActionID: "lifecycle-action:uninstall", ComponentID: "example", Kind: "uninstall", Direction: "inverse",
		TargetStateDigest: tagged("uninstall-target"), ExpectedArtifactDigests: []string{receipt.ReceiptDigest},
		ExpectedEvidence: []string{}, PrerequisiteActionIDs: []string{}, Disposition: "ready", Blockers: []Blocker{},
	}
	removed := executor.Execute(uninstall, absent, observed)
	if removed.Outcome != "committed" {
		t.Fatalf("uninstall failed: %+v", removed)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, ownedPath)); !os.IsNotExist(err) {
		t.Fatalf("receipt-owned file survived uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, receiptRelative)); !os.IsNotExist(err) {
		t.Fatalf("receipt commit marker survived uninstall: %v", err)
	}
	if retried := executor.Execute(uninstall, absent, Observation{}); retried.Outcome != "already_applied" {
		t.Fatalf("interrupted/uninstalled retry failed: %+v", retried)
	}
}

func TestExecutorRefusesConflictingTargetAndUnprovenRemoval(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	stagedRoot := resolvedTempDir(t)
	targetRoot := resolvedTempDir(t)
	ownedPath := "lib/example/data"
	data := []byte("expected")
	writeTestFile(t, filepath.Join(stagedRoot, ownedPath), data, 0o600)
	receipt := receiptV2{
		Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2,
		ComponentID: "example", ComponentKind: "module", ModuleID: "example", PackageID: "example",
		Version: "1", InstallScope: "prefix", PrefixMode: "installation_prefix",
		Files:       []receiptV2File{{Path: ownedPath, Kind: "regular", Size: uint64(len(data)), Digest: taggedBytes(data)}},
		EntryPoints: []receiptV2EntryPoint{}, ProvidesCapabilities: []string{}, RequiresCapabilities: []string{},
		CompatibleReceptors: []string{}, PlatformRequirements: []receiptV2Platform{},
	}
	receipt.ReceiptDigest = receiptV2Digest(t, receipt)
	receiptRelative := "share/symphony/receipts/example/1/install-receipt.json"
	encoded, _ := json.Marshal(receipt)
	writeTestFile(t, filepath.Join(stagedRoot, receiptRelative), encoded, 0o600)
	writeTestFile(t, filepath.Join(targetRoot, ownedPath), []byte("administrator data"), 0o600)
	executor, _ := NewExecutor(stateRoot, testTOPSID, "default", []string{stagedRoot})
	desired := DesiredState{TOPSID: testTOPSID, ProfileID: "default", Components: []DesiredComponent{{
		ComponentID: "example", Presence: "present", InstallRoot: targetRoot,
		SelectedPackage: &PackageIdentity{PackageID: "example", Version: "1", ReceiptProtocol: receipt.Protocol, ReceiptDigest: receipt.ReceiptDigest},
	}}}
	if _, err := executor.ReconcileOwnership(desired, Observation{TOPSID: testTOPSID, ProfileID: "default"}); err != nil {
		t.Fatal(err)
	}
	action := PlannedAction{
		ActionID: "lifecycle-action:conflict", ComponentID: "example", Kind: "install", Direction: "forward",
		TargetStateDigest: tagged("target"), ExpectedArtifactDigests: []string{receipt.ReceiptDigest},
		ExpectedEvidence: []string{}, PrerequisiteActionIDs: []string{}, Disposition: "waiting", Blockers: []Blocker{},
	}
	result := executor.Execute(action, desired, Observation{})
	if result.Outcome != "failed" || result.BlockerClass == nil || *result.BlockerClass != "integrity_fatal" {
		t.Fatalf("conflicting administrator file was not protected: %+v", result)
	}
}

func TestExecutorBindsRuntimeMutationToPreparedObservation(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	executor, err := NewExecutor(stateRoot, testTOPSID, "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	receiptDigest := tagged("runtime-selected-package")
	desired := DesiredState{Components: []DesiredComponent{{
		ComponentID: "example", Presence: "present",
		SelectedPackage: &PackageIdentity{
			PackageID: "example", Version: "1", ReceiptProtocol: "symphony.knowledge.install-receipt.v2",
			ReceiptDigest: receiptDigest,
		},
	}}}
	preparedDigest := tagged("prepared-component-observation")
	observed := Observation{Components: []ObservedComponent{{
		ComponentID: "example", Activation: "inactive", Docking: "undocked",
		Packages: []ObservedPackage{{
			PackageID: "example", Version: "1", ReceiptProtocol: "symphony.knowledge.install-receipt.v2",
			ReceiptDigest: receiptDigest, Integrity: "valid", EntryPointsValidated: true,
		}},
		ObservationDigest: preparedDigest,
	}}}
	stale := tagged("different-prepared-observation")
	action := PlannedAction{
		ActionID: "lifecycle-action:select", ComponentID: "example", Kind: "select", Direction: "forward",
		ExpectedBeforeDigest: &stale, TargetStateDigest: tagged("select-target"),
		ExpectedArtifactDigests: []string{}, ExpectedEvidence: []string{},
		PrerequisiteActionIDs: []string{}, Disposition: "ready", Blockers: []Blocker{},
	}
	blocked := executor.Execute(action, desired, observed)
	if blocked.Outcome != "failed" || blocked.BlockerClass == nil || *blocked.BlockerClass != "observation_retryable" {
		t.Fatalf("stale prepared runtime mutation was not blocked: %+v", blocked)
	}
	action.ExpectedBeforeDigest = &preparedDigest
	committed := executor.Execute(action, desired, observed)
	if committed.Outcome != "committed" {
		t.Fatalf("matching prepared runtime mutation failed: %+v", committed)
	}
	concurrent := executor.Execute(action, desired, observed)
	if concurrent.Outcome != "failed" || concurrent.BlockerClass == nil || *concurrent.BlockerClass != "observation_retryable" {
		t.Fatalf("runtime mutation raced past stale observed runtime state: %+v", concurrent)
	}
	observed.Components[0].SelectedPackageDigest = &receiptDigest
	observed.Components[0].ObservationDigest = tagged("post-selection-observation")
	action.ExpectedBeforeDigest = &stale
	if retried := executor.Execute(action, desired, observed); retried.Outcome != "already_applied" {
		t.Fatalf("idempotent runtime retry did not self-heal across observation drift: %+v", retried)
	}
}

func TestExecutorRoutesEstablishedSelectionThroughBindingAdapter(t *testing.T) {
	executor, err := NewExecutor(resolvedTempDir(t), testTOPSID, "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingBindingAdapter{}
	executor.SetBindingAdapter(adapter)
	receiptDigest := tagged("established-selected-package")
	desired := DesiredState{Components: []DesiredComponent{{
		ComponentID: "skvi-engine", Presence: "present",
		SelectedPackage: &PackageIdentity{
			PackageID: "skvi-engine", Version: "2.0.0",
			ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: receiptDigest,
		},
	}}}
	before := tagged("established-before")
	observed := Observation{BindingRegistryDigest: stringPointer(tagged("registry-before")), Components: []ObservedComponent{{
		ComponentID: "skvi-engine", Activation: "inactive", Docking: "undocked",
		Packages: []ObservedPackage{{
			PackageID: "skvi-engine", Version: "2.0.0",
			ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: receiptDigest,
			Integrity: "valid", EntryPointsValidated: true,
		}},
		ObservationDigest: before,
	}}}
	action := PlannedAction{
		ActionID: "lifecycle-action:established-select", ComponentID: "skvi-engine",
		Kind: "select", Direction: "forward", ExpectedBeforeDigest: &before,
		TargetStateDigest: tagged("established-target"), ExpectedArtifactDigests: []string{receiptDigest},
		ExpectedEvidence: []string{}, PrerequisiteActionIDs: []string{}, Disposition: "ready", Blockers: []Blocker{},
	}
	result := executor.Execute(action, desired, observed)
	if result.Outcome != "committed" || adapter.calls != 1 ||
		!containsString(result.AfterEvidenceDigests, tagged("binding-registry")) {
		t.Fatalf("established selection did not use binding adapter: result=%+v calls=%d", result, adapter.calls)
	}

	coordinator := action
	coordinator.ActionID = "lifecycle-action:coordinator-deselect"
	coordinator.ComponentID = "knowledge-session-coordinator"
	coordinator.Kind = "deselect"
	blocked := executor.Execute(coordinator, DesiredState{}, Observation{})
	if blocked.Outcome != "failed" || blocked.BlockerClass == nil || *blocked.BlockerClass != "compatibility_blocked" || adapter.calls != 1 {
		t.Fatalf("coordinator deselection was not blocked before adapter mutation: %+v", blocked)
	}
}

func TestExecutorUndocksObservedPackageBeforeReplacementOrRemoval(t *testing.T) {
	stateRoot := resolvedTempDir(t)
	installRoot := resolvedTempDir(t)
	executablePath := "libexec/example/symphony-example"
	executableData := []byte("example process\n")
	writeTestFile(t, filepath.Join(installRoot, executablePath), executableData, 0o700)
	vectorID := "example-vector"
	engineID := "symphony-example"
	receipt := receiptV2{
		Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2,
		ComponentID: "example", ComponentKind: "vector_engine", ModuleID: "example",
		VectorID: &vectorID, EngineID: &engineID, PackageID: "example", Version: "1.0.0",
		InstallScope: "prefix", PrefixMode: "installation_prefix",
		Files: []receiptV2File{{
			Path: executablePath, Kind: "executable", Size: uint64(len(executableData)),
			Digest: taggedBytes(executableData),
		}},
		EntryPoints: []receiptV2EntryPoint{{
			EntryPointID: "engine", Kind: "executable", Path: executablePath,
			Protocols: []string{"symphony.knowledge.engine-process.v1"},
		}},
		ProvidesCapabilities: []string{}, RequiresCapabilities: []string{},
		CompatibleReceptors:  []string{"symphony.maestro.knowledge-engine.v1"},
		PlatformRequirements: []receiptV2Platform{},
	}
	receipt.ReceiptDigest = receiptV2Digest(t, receipt)
	receiptPath := filepath.Join(
		installRoot, "share/symphony/receipts/example/1.0.0/install-receipt.json")
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, receiptPath, encoded, 0o600)

	executor, err := NewExecutor(stateRoot, testTOPSID, "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingDockingAdapter{}
	executor.SetDockingAdapter(adapter)
	selected := receipt.ReceiptDigest
	receptor := "maestro-old"
	before := tagged("docked-old-package")
	observed := Observation{Components: []ObservedComponent{{
		ComponentID: "example", ComponentKind: "vector_engine", ModuleID: "example",
		VectorID: &vectorID, EngineID: &engineID,
		Packages: []ObservedPackage{{
			PackageID: receipt.PackageID, Version: receipt.Version, InstallRoot: installRoot,
			ReceiptProtocol: receipt.Protocol, ReceiptDigest: receipt.ReceiptDigest,
			Integrity: "valid", EntryPointsValidated: true,
		}},
		SelectedPackageDigest: &selected, Activation: "inactive", Docking: "docked",
		ReceptorID: &receptor, Capabilities: []string{}, PlatformCompatibility: "compatible",
		ObservationDigest: before,
	}}}
	desired := DesiredState{Components: []DesiredComponent{{
		ComponentID: "example", Presence: "absent", InstallRoot: installRoot,
	}}}
	action := PlannedAction{
		ActionID: "lifecycle-action:undock-observed", ComponentID: "example", Kind: "undock",
		Direction: "inverse", ExpectedBeforeDigest: &before, TargetStateDigest: tagged("absent-target"),
		ExpectedArtifactDigests: []string{receipt.ReceiptDigest}, ExpectedEvidence: []string{},
		PrerequisiteActionIDs: []string{}, Disposition: "ready", Blockers: []Blocker{},
	}
	result := executor.Execute(action, desired, observed)
	if result.Outcome != "committed" || adapter.calls != 1 ||
		adapter.component.ReceiptDigest != receipt.ReceiptDigest ||
		adapter.component.ExecutableDigest != taggedBytes(executableData) {
		t.Fatalf("inverse undock did not bind the observed installed package: result=%+v adapter=%+v", result, adapter)
	}

	observed.Components[0].Docking = "undocked"
	observed.Components[0].ReceptorID = nil
	observed.Components[0].ObservationDigest = tagged("already-undocked")
	retried := executor.Execute(action, desired, observed)
	if retried.Outcome != "already_applied" || adapter.calls != 1 {
		t.Fatalf("post-crash undock retry did not self-heal from authenticated observation: %+v", retried)
	}
}

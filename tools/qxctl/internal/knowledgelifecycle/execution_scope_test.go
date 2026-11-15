package knowledgelifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureExecutionScope(t *testing.T) ExecutionScope {
	t.Helper()
	data, err := os.ReadFile("../../../../modules/knowledge-session-coordinator/tests/fixtures/lifecycle_execution_scope.json")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := DecodeExecutionScope(data)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}
func resignScope(t *testing.T, scope *ExecutionScope) {
	t.Helper()
	var err error
	scope.ScopeDigest, err = ExecutionScopeDigest(*scope)
	if err != nil {
		t.Fatal(err)
	}
}
func testOwnershipScope(t *testing.T, desired *DesiredState, root string, effects []ScopeOwnershipEffect) *ExecutionScope {
	t.Helper()
	scope := fixtureExecutionScope(t)
	scope.TOPSID = desired.TOPSID
	scope.ProfileID = desired.ProfileID
	desired.DesiredStateDigest = tagged("scoped desired")
	scope.DesiredStateDigest = desired.DesiredStateDigest
	scope.Components = []string{"example"}
	scope.Effects[0].ComponentID = "example"
	scope.Effects[0].InstallRoot = root
	if desired.Components[0].Presence == "absent" {
		scope.Effects[0].Kind = "uninstall"
		scope.Effects[0].ExpectedArtifactDigests = []string{effects[0].ReceiptDigest}
	} else {
		scope.Effects[0].ExpectedArtifactDigests = []string{desired.Components[0].SelectedPackage.ReceiptDigest}
	}
	scope.OwnershipEffects = effects
	resignScope(t, &scope)
	return &scope
}
func TestExecutionScopeCanonicalNativeFixtureAndClosedFields(t *testing.T) {
	scope := fixtureExecutionScope(t)
	if scope.ScopeDigest != "sha256:1c344d49457449df9f9c5e5f687b2874ea4730e1e83933fda513de483d49afe9" {
		t.Fatal("shared native canonical fixture drift")
	}
	encoded, err := scopeJSON(map[string]any{"value": "<>&é\u2028\u2029\\u2028"})
	if err != nil || string(encoded) != "{\"value\":\"<>&é\u2028\u2029\\\\u2028\"}" {
		t.Fatalf("native string encoding drift: %s %v", encoded, err)
	}
	for _, mutation := range []func(map[string]any){
		func(v map[string]any) { delete(v, "canonical") },
		func(v map[string]any) { v["extra"] = false },
		func(v map[string]any) { delete(v["effects"].([]any)[0].(map[string]any), "target_receptor_id") },
		func(v map[string]any) { v["components"] = []string{"module-a", "module-a"} },
		func(v map[string]any) { v["effects"].([]any)[0].(map[string]any)["install_root"] = "/" },
		func(v map[string]any) { v["ownership_effects"] = nil },
	} {
		data, _ := json.Marshal(scope)
		var value map[string]any
		json.Unmarshal(data, &value)
		mutation(value)
		data, _ = json.Marshal(value)
		if _, err := DecodeExecutionScope(data); err == nil {
			t.Fatalf("invalid field set accepted: %s", data)
		}
	}
	data, _ := json.Marshal(scope)
	data = append([]byte(`{"canonical":false,`), data[1:]...)
	if _, err := DecodeExecutionScope(data); err == nil {
		t.Fatal("duplicate field accepted")
	}
}
func TestExecutionScopeExactEffectsCompletionAndFullObservation(t *testing.T) {
	scope := fixtureExecutionScope(t)
	effect := scope.Effects[0]
	desired := DesiredState{TOPSID: scope.TOPSID, ProfileID: scope.ProfileID, DesiredStateDigest: scope.DesiredStateDigest, Components: []DesiredComponent{{ComponentID: effect.ComponentID, InstallRoot: effect.InstallRoot, Presence: "present", SelectedPackage: &PackageIdentity{ReceiptDigest: effect.ExpectedArtifactDigests[0]}}, {ComponentID: "other", InstallRoot: "/opt/other"}}}
	if err := scope.ValidateDesired(scope.ProfileDigest, desired, scope.SourceReportJournalDigest); err != nil {
		t.Fatal(err)
	}
	action := PlannedAction{ActionID: "replanned-id", ComponentID: effect.ComponentID, Kind: effect.Kind, TargetStateDigest: effect.TargetStateDigest, ExpectedArtifactDigests: effect.ExpectedArtifactDigests}
	if !scope.AllowsAction(action, desired) {
		t.Fatal("semantic action after replan was refused")
	}
	action.Kind = "select"
	if scope.AllowsAction(action, desired) {
		t.Fatal("downstream selection was authorized")
	}
	action.Kind = effect.Kind
	action.ComponentID = "other"
	if scope.AllowsAction(action, desired) {
		t.Fatal("unrelated action was authorized")
	}

	before := Observation{Components: []ObservedComponent{{ComponentID: effect.ComponentID}, {ComponentID: "other", Activation: "inactive"}}}
	a, _ := scope.PreservedComponentsDigest(before)
	before.Components[0].Activation = "active"
	b, _ := scope.PreservedComponentsDigest(before)
	if a != b {
		t.Fatal("selected component changed outside-scope digest")
	}
	before.Components[1].Activation = "active"
	c, _ := scope.PreservedComponentsDigest(before)
	if b == c {
		t.Fatal("unrelated drift escaped full-component hash")
	}
}
func TestExecutionScopeReadRejectsSymlinkAndUnsafeFile(t *testing.T) {
	root := resolvedTempDir(t)
	path := filepath.Join(root, "scope.json")
	scope := fixtureExecutionScope(t)
	data, _ := json.Marshal(scope)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadExecutionScope(path); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias.json")
	if err := os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadExecutionScope(alias); err == nil {
		t.Fatal("symlink scope accepted")
	}
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadExecutionScope(path); err == nil {
		t.Fatal("writable-by-others scope accepted")
	}
}
func TestExecutionLeaseSerializesWholeExternalExecution(t *testing.T) {
	store, err := NewStore(resolvedTempDir(t), testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- store.WithExecutionLease(func() error { close(entered); <-release; return nil }) }()
	<-entered
	calls := 0
	err = store.WithExecutionLease(func() error { calls++; return nil })
	if err == nil || !strings.Contains(err.Error(), "busy") || calls != 0 {
		t.Fatalf("duplicate external worker admitted: %v calls=%d", err, calls)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := store.WithExecutionLease(func() error { calls++; return nil }); err != nil || calls != 1 {
		t.Fatalf("released lease unavailable: %v", err)
	}
}
func TestScopedOwnershipPreservesOtherClaimsAndRefusesImplicitAdoption(t *testing.T) {
	root := filepath.Join(resolvedTempDir(t), "install")
	state := resolvedTempDir(t)
	old := tagged("old")
	other := tagged("other")
	desired := ownershipDesired("default", root, old, "present")
	desired.Components = append(desired.Components, DesiredComponent{ComponentID: "other", Presence: "present", InstallRoot: root, SelectedPackage: &PackageIdentity{ReceiptDigest: other}})
	observed := ownershipObservation("default", root, "example", nil)
	owner, err := NewOwnershipStore(root, state, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = owner.Reconcile(desired, observed); err != nil {
		t.Fatal(err)
	}
	observed = ownershipObservation("default", root, "example", []string{old})
	desired.Components[0].Presence = "absent"
	desired.Components[0].SelectedPackage = nil
	desired.Components[1].Presence = "absent"
	desired.Components[1].SelectedPackage = nil
	owner.executionScope = testOwnershipScope(t, &desired, root, []ScopeOwnershipEffect{{InstallRoot: root, ComponentID: "example", ReceiptDigest: old, Disposition: "retiring"}})
	before, err := owner.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	result, err := owner.Reconcile(desired, observed)
	if err != nil {
		t.Fatal(err)
	}
	for _, claim := range before.Registry.Claims {
		if claim.ComponentID == "other" {
			found := false
			for _, after := range result.Snapshot.Registry.Claims {
				found = found || claim == after
			}
			if !found {
				t.Fatal("unrelated claim modified")
			}
		}
	}
	unknown := ownershipObservation("default", root, "unknown", []string{tagged("unknown")})
	unknown.Components = append(unknown.Components, observed.Components...)
	stable := result.Snapshot.Registry.OwnershipRegistryDigest
	if _, err := owner.Reconcile(desired, unknown); err == nil {
		t.Fatal("hidden adoption accepted")
	}
	after, err := owner.Snapshot()
	if err != nil || after.Registry.OwnershipRegistryDigest != stable {
		t.Fatal("failed scoped reconcile wrote registry")
	}
	if err := os.Remove(filepath.Join(root, ownershipFenceRelative)); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Reconcile(desired, observed); err == nil {
		t.Fatal("scope silently repaired ownership fence")
	}
	if _, err := os.Stat(filepath.Join(root, ownershipFenceRelative)); !os.IsNotExist(err) {
		t.Fatal("scope repaired rejected fence")
	}
}
func TestScopedExecutorRejectsUnauthorizedActionBeforeBinding(t *testing.T) {
	executor, err := NewExecutor(resolvedTempDir(t), testTOPSID, "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingBindingAdapter{}
	executor.SetBindingAdapter(adapter)
	scope := fixtureExecutionScope(t)
	desired := DesiredState{TOPSID: testTOPSID, ProfileID: "default", DesiredStateDigest: scope.DesiredStateDigest, Components: []DesiredComponent{{ComponentID: "module-a", InstallRoot: scope.Effects[0].InstallRoot, Presence: "present", SelectedPackage: &PackageIdentity{ReceiptDigest: scope.Effects[0].ExpectedArtifactDigests[0]}}}}
	scope.TOPSID = testTOPSID
	resignScope(t, &scope)
	executor.SetExecutionScope(&scope)
	action := PlannedAction{ActionID: "out-of-scope", ComponentID: "knowledge-session-coordinator", Kind: "select"}
	result := executor.Execute(action, desired, Observation{})
	if result.Outcome != "failed" || adapter.calls != 0 || !strings.Contains(result.Detail, "outside immutable execution scope") {
		t.Fatalf("unauthorized external effect escaped: %+v calls=%d", result, adapter.calls)
	}
}

func TestScopedOwnershipCannotCompleteAlreadyInstalledUnclaimedTarget(t *testing.T) {
	root := filepath.Join(resolvedTempDir(t), "install")
	state := resolvedTempDir(t)
	receipt := tagged("unclaimed-target")
	owner, err := NewOwnershipStore(root, state, testTOPSID, "default")
	if err != nil {
		t.Fatal(err)
	}
	empty := ownershipDesired("default", root, "", "absent")
	observed := ownershipObservation("default", root, "example", nil)
	initial, err := owner.Reconcile(empty, observed)
	if err != nil {
		t.Fatal(err)
	}
	desired := ownershipDesired("default", root, receipt, "present")
	observed = ownershipObservation("default", root, "example", []string{receipt})
	owner.executionScope = testOwnershipScope(t, &desired, root, []ScopeOwnershipEffect{{InstallRoot: root, ComponentID: "example", ReceiptDigest: receipt, Disposition: "retained"}})
	if _, err := owner.Reconcile(desired, observed); err == nil {
		t.Fatal("already-installed target without existing ownership was implicitly adopted")
	}
	after, err := owner.Snapshot()
	if err != nil || after.Registry.OwnershipRegistryDigest != initial.Snapshot.Registry.OwnershipRegistryDigest {
		t.Fatal("refused completion changed ownership")
	}
}
func TestScopedActionProjectionProtectsUnrelatedFields(t *testing.T) {
	receipt := tagged("target")
	other := tagged("other")
	component := ObservedComponent{ComponentID: "example", ComponentKind: "module", ModuleID: "example", Activation: "inactive", Docking: "undocked", Packages: []ObservedPackage{{ReceiptDigest: receipt, InstallRoot: "/opt/test", Integrity: "valid", EntryPointsValidated: true}}, Capabilities: []string{"alpha"}, PlatformCompatibility: "compatible", ObservationDigest: tagged("before")}
	observation := Observation{Components: []ObservedComponent{component}}
	action := PlannedAction{ComponentID: "example", Kind: "activate"}
	before, err := ActionPreservedComponentsDigest(observation, action)
	if err != nil {
		t.Fatal(err)
	}
	observation.Components[0].Activation = "active"
	observation.Components[0].ObservationDigest = tagged("after")
	after, err := ActionPreservedComponentsDigest(observation, action)
	if err != nil || before != after {
		t.Fatal("authorized activation changed preserved fields")
	}
	observation.Components[0].SelectedPackageDigest = &receipt
	changed, _ := ActionPreservedComponentsDigest(observation, action)
	if changed == before {
		t.Fatal("activation concealed a selection")
	}
	action.Kind = "install"
	action.ExpectedArtifactDigests = []string{receipt}
	if _, err := ActionPreservedComponentsDigest(observation, action); err == nil {
		t.Fatal("install concealed active selected state")
	}
	observation.Components[0] = component
	installed, err := ActionPreservedComponentsDigest(observation, action)
	if err != nil {
		t.Fatal(err)
	}
	absent, _ := ActionPreservedComponentsDigest(Observation{Components: []ObservedComponent{}}, action)
	if installed != absent {
		t.Fatal("exact default-state install did not match missing component projection")
	}
	observation.Components[0].Packages = append(observation.Components[0].Packages, ObservedPackage{ReceiptDigest: other, InstallRoot: "/opt/test", Integrity: "valid"})
	a, _ := ActionPreservedComponentsDigest(observation, action)
	observation.Components[0].Packages[1].Integrity = "invalid"
	b, _ := ActionPreservedComponentsDigest(observation, action)
	if a == b {
		t.Fatal("install concealed another receipt changing")
	}
}
func TestScopedEffectCompletionLeavesSelectionPending(t *testing.T) {
	scope := fixtureExecutionScope(t)
	effect := scope.Effects[0]
	receipt := effect.ExpectedArtifactDigests[0]
	desired := DesiredState{Components: []DesiredComponent{{ComponentID: effect.ComponentID, Presence: "present", InstallRoot: effect.InstallRoot, SelectedPackage: &PackageIdentity{ReceiptDigest: receipt}}}}
	observation := Observation{Components: []ObservedComponent{{ComponentID: effect.ComponentID, Activation: "inactive", Docking: "undocked", Packages: []ObservedPackage{{ReceiptDigest: receipt, InstallRoot: effect.InstallRoot, Integrity: "valid", EntryPointsValidated: true}}}}}
	if !scope.EffectsComplete(desired, observation, 0) || scope.EffectsComplete(desired, observation, 1) {
		t.Fatal("exact install scope did not complete independently of selection")
	}
	observation.Components[0].Packages[0].EntryPointsValidated = false
	if scope.EffectsComplete(desired, observation, 0) {
		t.Fatal("unvalidated install was complete")
	}
	observation.Components[0].Packages[0].EntryPointsValidated = true
	observation.Components[0].Packages[0].InstallRoot = "/opt/another"
	if scope.EffectsComplete(desired, observation, 0) {
		t.Fatal("wrong-root receipt completed")
	}
}
func TestInitialScopePlanDigestIgnoresOnlyObservationTimestampBinding(t *testing.T) {
	stable := tagged("stable")
	plan := map[string]any{"protocol": "symphony.knowledge.lifecycle-plan.v1", "observation_digest": tagged("first-time"), "plan_digest": tagged("plan-first"), "actions": []any{map[string]any{"kind": "install"}}, "desired_state_digest": tagged("desired")}
	first, err := InitialScopePlanDigest(plan, stable)
	if err != nil {
		t.Fatal(err)
	}
	plan["observation_digest"] = tagged("later-time")
	plan["plan_digest"] = tagged("plan-later")
	second, err := InitialScopePlanDigest(plan, stable)
	if err != nil || first != second {
		t.Fatal("timestamp-only plan change invalidated reviewed semantics")
	}
	plan["actions"] = []any{map[string]any{"kind": "select"}}
	third, _ := InitialScopePlanDigest(plan, stable)
	if third == first {
		t.Fatal("different action escaped reviewed semantic digest")
	}
}

package foundationlifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestAdapterValidationBindsIndependentReceiptEvidence(t *testing.T) {
	evidence := installEvidence{
		digest: testDigest("receipt"), binary: "/opt/symphony/libexec/symphony/stav-append-authority/v1/symphony-stav-append-authority",
		binaryDigest: testDigest("binary"), version: "v1",
	}
	adapter := validTestAdapter("stav", evidence)
	if err := validateAdapter(adapter, evidence, "stav"); err != nil {
		t.Fatalf("valid adapter rejected: %v", err)
	}
	adapter.InstallEvidenceDigest = testDigest("forged-receipt")
	adapter.DescriptorDigest = testObjectDigest(adapter, "descriptor_digest")
	if err := validateAdapter(adapter, evidence, "stav"); err == nil || !strings.Contains(err.Error(), "capability contract") {
		t.Fatalf("adapter self-asserted forged install evidence was accepted: %v", err)
	}
}

func TestObservationValidationRecomputesStableStateIncludingRecovery(t *testing.T) {
	observation := validTestObservation("stav", "enrollment")
	if err := validateObservation(observation, "stav", "enrollment", "user", testTOPSID); err != nil {
		t.Fatalf("valid observation rejected: %v", err)
	}
	observation.RecoveryRequired = true
	active := testDigest("attempt")
	observation.ActiveAttemptDigest = &active
	observation.ObservationDigest = testObjectDigest(observation, "observation_digest")
	if err := validateObservation(observation, "stav", "enrollment", "user", testTOPSID); err == nil || !strings.Contains(err.Error(), "stable state digest mismatch") {
		t.Fatalf("stable-state recovery tampering was accepted: %v", err)
	}
	observation.StableStateDigest, _ = digestWithoutFields(observation, "observed_at", "stable_state_digest", "observation_digest")
	observation.ObservationDigest = testObjectDigest(observation, "observation_digest")
	if err := validateObservation(observation, "stav", "enrollment", "user", testTOPSID); err != nil {
		t.Fatalf("recovery fields were not accepted when bound into stable state: %v", err)
	}
}

func TestResultValidationPreservesDeferredAuditReconciliation(t *testing.T) {
	evidence := installEvidence{
		digest: testDigest("receipt"), binary: "/opt/symphony/libexec/symphony/stav-append-authority/v1/symphony-stav-append-authority",
		binaryDigest: testDigest("binary"), version: "v1",
	}
	adapter := validTestAdapter("stav", evidence)
	operationID := "fixture-operation"
	request := command{Operation: "apply", Component: "stav", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID, OperationID: &operationID}
	observation := validTestObservation("stav", "enrollment")
	observation.Installation.BinaryPath = &evidence.binary
	observation.Installation.BinaryDigest = &evidence.binaryDigest
	observation.Installation.InstallEvidenceDigest = &evidence.digest
	observation.Installation.ReceiptDigest = &evidence.digest
	observation.StableStateDigest, _ = digestWithoutFields(observation, "observed_at", "stable_state_digest", "observation_digest")
	observation.ObservationDigest = testObjectDigest(observation, "observation_digest")
	result := Result{
		Protocol: ResultProtocol, FormatVersion: 1, Operation: "apply", Component: "stav", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID,
		OperationID: &operationID, Disposition: "applied", DesiredState: stringTestPointer("enrolled"), Observation: observation,
		Changed: true, ReconciliationRequired: true, AttemptDigest: stringTestPointer(testDigest("attempt")),
		AuditState: "audit_deferred", StartedAt: stringTestPointer("2026-08-13T12:00:00Z"), CompletedAt: "2026-08-13T12:00:01Z",
	}
	result.ResultDigest = testObjectDigest(result, "result_digest")
	encoded, _ := json.Marshal(result)
	if _, err := validateResult(encoded, request, adapter); err != nil {
		t.Fatalf("valid audit-deferred result rejected: %v", err)
	}
	result.ReconciliationRequired = false
	result.ResultDigest = testObjectDigest(result, "result_digest")
	encoded, _ = json.Marshal(result)
	if _, err := validateResult(encoded, request, adapter); err == nil {
		t.Fatal("audit-deferred result was mislabeled as fully reconciled")
	}
	result.ReconciliationRequired = true
	result.Disposition = "observed"
	result.ResultDigest = testObjectDigest(result, "result_digest")
	encoded, _ = json.Marshal(result)
	if _, err := validateResult(encoded, request, adapter); err == nil || !strings.Contains(err.Error(), "invalid apply disposition") {
		t.Fatalf("apply accepted an observation disposition: %v", err)
	}
}

func TestControlledEnvironmentExcludesUserStateForSystemScope(t *testing.T) {
	t.Setenv("HOME", "/tmp/forged-home")
	t.Setenv("XDG_STATE_HOME", "/tmp/forged-state")
	joined := strings.Join(controlledEnvironment("system"), "\n")
	if strings.Contains(joined, "HOME=") || strings.Contains(joined, "XDG_") {
		t.Fatalf("system adapter inherited user state environment: %s", joined)
	}
}

func TestOperationShapesFailClosedBeforeInvocation(t *testing.T) {
	operationID := "fixture-operation"
	if err := validateOptions(Options{
		Component: "stav", Prefix: t.TempDir(), Surface: "enrollment", Operation: "observe",
		Scope: "user", TOPSID: testTOPSID, OperationID: &operationID,
	}); err == nil || !strings.Contains(err.Error(), "observe accepts no") {
		t.Fatalf("observe accepted mutation identity: %v", err)
	}
	if err := validateOptions(Options{
		Component: "stav", Prefix: t.TempDir(), Surface: "enrollment", Operation: "apply",
		Scope: "user", TOPSID: testTOPSID, Plan: &Plan{}, OperationID: &operationID,
		ExpectedStateDigest: stringTestPointer("absent"),
	}); err == nil || !strings.Contains(err.Error(), "apply requires exact") {
		t.Fatalf("apply accepted missing attempt evidence: %v", err)
	}
}

func TestUserPlanAllowsOnlyKernelDerivedIdentity(t *testing.T) {
	uid, gid := effectiveIdentity()
	plan := Plan{
		Protocol: PlanProtocol, FormatVersion: 1, Component: "stav", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID,
		OperationID: "fixture-operation", RequestID: "018f0c3a-7b2d-7e11-8c12-0242ac120003", CorrelationID: "018f0c3a-7b2d-7e11-8c12-0242ac120004",
		ExpectedStateDigest: testDigest("state"), DesiredState: "enrolled", AuthorityUID: &uid, AuthorityGID: &gid, AuditMode: "ordinary",
		CreatedAt: "2026-08-13T12:00:00Z", ExpiresAt: "2026-08-13T12:05:00Z",
	}
	plan.PlanDigest = testObjectDigest(plan, "plan_digest")
	if err := validatePlan(plan, "stav", "enrollment", "user", testTOPSID); err != nil {
		t.Fatalf("kernel-derived user identity rejected: %v", err)
	}
	*plan.AuthorityUID++
	plan.PlanDigest = testObjectDigest(plan, "plan_digest")
	if err := validatePlan(plan, "stav", "enrollment", "user", testTOPSID); err == nil || !strings.Contains(err.Error(), "non-kernel") {
		t.Fatalf("non-kernel user identity accepted: %v", err)
	}
}

func TestReadPlanAcceptsOnlyDigestBoundPlanResult(t *testing.T) {
	observation := validTestObservation("stav", "enrollment")
	plan := Plan{
		Protocol: PlanProtocol, FormatVersion: 1, Component: "stav", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID,
		OperationID: "fixture-operation", RequestID: "018f0c3a-7b2d-7e11-8c12-0242ac120003", CorrelationID: "018f0c3a-7b2d-7e11-8c12-0242ac120004",
		ExpectedStateDigest: observation.StableStateDigest, DesiredState: "enrolled", AuditMode: "ordinary",
		CreatedAt: "2026-08-13T12:00:00Z", ExpiresAt: "2026-08-13T12:05:00Z",
	}
	plan.PlanDigest = testObjectDigest(plan, "plan_digest")
	result := Result{
		Protocol: ResultProtocol, FormatVersion: 1, Operation: "plan", Component: "stav", Surface: "enrollment", Scope: "user", TOPSID: testTOPSID,
		OperationID: &plan.OperationID, Disposition: "planned", DesiredState: &plan.DesiredState, Observation: observation, Plan: &plan,
		AuditState: "not_applicable", CompletedAt: "2026-08-13T12:00:00Z", ReadOnly: true,
	}
	result.ResultDigest = testObjectDigest(result, "result_digest")
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadPlan(path)
	if err != nil || loaded.PlanDigest != plan.PlanDigest {
		t.Fatalf("valid result-wrapped plan rejected: %+v err=%v", loaded, err)
	}
	result.Plan.ExpiresAt = result.Plan.CreatedAt
	result.Plan.PlanDigest = testObjectDigest(*result.Plan, "plan_digest")
	result.ResultDigest = testObjectDigest(result, "result_digest")
	encoded, _ = json.Marshal(result)
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPlan(path); err == nil || !strings.Contains(err.Error(), "validity interval") {
		t.Fatalf("zero-validity plan accepted: %v", err)
	}
	result.Plan.ExpiresAt = "2026-08-13T12:05:00Z"
	result.Plan.Component = "foreign"
	result.Plan.PlanDigest = testObjectDigest(*result.Plan, "plan_digest")
	result.Component, result.Observation.Component = "foreign", "foreign"
	result.Observation.StableStateDigest, _ = digestWithoutFields(result.Observation, "observed_at", "stable_state_digest", "observation_digest")
	result.Observation.ObservationDigest = testObjectDigest(result.Observation, "observation_digest")
	result.ResultDigest = testObjectDigest(result, "result_digest")
	encoded, _ = json.Marshal(result)
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPlan(path); err == nil || !strings.Contains(err.Error(), "required contract") {
		t.Fatalf("plan with an unregistered component was accepted: %v", err)
	}
}

func TestInstallationSelectionRequiresExactReceiptV2AndRejectsAmbiguity(t *testing.T) {
	prefix := t.TempDir()
	createFoundationInstallFixture(t, prefix, "stav", "0.1.0")
	selected, err := verifiedInstallation("stav", prefix, "")
	if err != nil || selected.version != "0.1.0" {
		t.Fatalf("one exact receipt-v2 adapter was not selected: %+v err=%v", selected, err)
	}
	createFoundationInstallFixture(t, prefix, "stav", "0.2.0")
	if _, err := verifiedInstallation("stav", prefix, ""); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("multiple installed versions did not require --version: %v", err)
	}
	selected, err = verifiedInstallation("stav", prefix, "0.1.0")
	if err != nil || selected.version != "0.1.0" {
		t.Fatalf("explicit version did not select exact receipt: %+v err=%v", selected, err)
	}
	if err := os.WriteFile(selected.binary, []byte("tampered\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := verifiedInstallation("stav", prefix, "0.1.0"); err == nil {
		t.Fatal("content-tampered adapter was accepted")
	}
}

func validTestAdapter(component string, evidence installEvidence) Adapter {
	operations := make([]string, 0, 10)
	for _, surface := range []string{"enrollment", "supervisor"} {
		for _, operation := range []string{"observe", "plan", "apply", "apply-status", "recover"} {
			operations = append(operations, "engop:symphony:"+component+"."+surface+"."+operation)
		}
	}
	adapter := Adapter{
		Protocol: AdapterProtocol, FormatVersion: 1, Component: component, AdapterVersion: evidence.version,
		BinaryPath: evidence.binary, BinaryDigest: evidence.binaryDigest, InstallEvidenceDigest: evidence.digest,
		Operations: operations, SupportedScopes: []string{"user", "system"}, SupportedManagers: []string{"external"},
		Compatibility: Compatibility{ConfigReadMajors: []uint64{1}, ConfigWriteMajor: 1, RuntimeReadMajors: []uint64{1}, RuntimeWriteMajor: 1, StateReadMajors: []uint64{1}, StateWriteMajor: 1, RollbackReadable: true},
		Limits:        Limits{RequestBytes: 65536, ResponseBytes: 262144, DeadlineMS: 5000, JSONDepth: 32, JSONValues: 4096},
	}
	adapter.DescriptorDigest = testObjectDigest(adapter, "descriptor_digest")
	return adapter
}

func validTestObservation(component, surface string) Observation {
	binary, binaryDigest, receipt := "/opt/symphony/libexec/symphony/adapter", testDigest("binary"), testDigest("receipt")
	observation := Observation{
		Protocol: ObservationProtocol, FormatVersion: 1, Component: component, Surface: surface, Scope: "user", TOPSID: testTOPSID,
		Installation: Installation{State: "installed", BinaryPath: &binary, BinaryDigest: &binaryDigest, InstallEvidenceDigest: &receipt, ReceiptDigest: &receipt},
		Enrollment:   Enrollment{State: "unenrolled", DataPreserved: true},
		Supervisor:   Supervisor{ManagerState: "manager_unavailable", DescriptorState: "absent", Enablement: "not_applicable", ProcessState: "stopped", EndpointState: "absent"},
		ObservedAt:   "2026-08-13T12:00:00Z",
	}
	observation.StableStateDigest, _ = digestWithoutFields(observation, "observed_at", "stable_state_digest", "observation_digest")
	observation.ObservationDigest = testObjectDigest(observation, "observation_digest")
	return observation
}

func testObjectDigest(value any, field string) string {
	encoded, _ := json.Marshal(value)
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	_ = decoder.Decode(&object)
	delete(object, field)
	canonical, _ := json.Marshal(object)
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func testDigest(seed string) string {
	digest := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func stringTestPointer(value string) *string { return &value }

func createFoundationInstallFixture(t *testing.T, prefix, component, version string) {
	t.Helper()
	module := map[string]string{"ssiag": "secure-identity-access-governance", "stav": "stav-append-authority"}[component]
	binaryName := componentBinaries[component]
	relative := filepath.ToSlash(filepath.Join("libexec", "symphony", module, version, binaryName))
	binary := filepath.Join(prefix, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	contents := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(binary, contents, 0o755); err != nil {
		t.Fatal(err)
	}
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	receipt := map[string]any{
		"protocol": "symphony.knowledge.install-receipt.v2", "format_version": 2,
		"component_id": module, "component_kind": "service", "module_id": module,
		"vector_id": nil, "engine_id": nil, "package_id": module, "version": version,
		"install_scope": "prefix", "prefix_mode": "installation_prefix",
		"files": []any{map[string]any{"path": relative, "kind": "executable", "size": len(contents), "digest": testBytesDigest(contents)}},
		"entry_points": []any{map[string]any{
			"entry_point_id": component + ".foundation-lifecycle", "kind": "adapter", "path": relative,
			"protocols": []string{CommandProtocol},
		}},
		"provides_capabilities": []string{AdapterProtocol}, "requires_capabilities": []string{}, "compatible_receptors": []string{},
		"platform_requirements": []any{map[string]any{"os": platform, "architecture": runtime.GOARCH, "kernel_abi": nil, "critical": true}},
	}
	receipt["receipt_digest"] = testObjectDigest(receipt, "receipt_digest")
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(prefix, "share", "symphony", "receipts", module, version, "install-receipt.json")
	if err := os.MkdirAll(filepath.Dir(receiptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

func testBytesDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

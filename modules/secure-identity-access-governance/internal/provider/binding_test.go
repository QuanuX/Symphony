package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

type bindingLauncher struct{ manager *TrustManager }

var testBindingAuditIdentity = ProviderBindingAuditIdentity{
	ActorID: "symphony.host.owner.uid.501", ActorKind: "symphony.identity.host-owner", AuthenticationMethod: "symphony.ssiag.local-peer",
}

func (b *bindingLauncher) Exchange(_ context.Context, declaration ExecutableTrust, request ControlRequest) (ControlResponse, error) {
	return verifiedResponse(b.manager, declaration, request.RequestID, request.CorrelationID), nil
}

func bindingFixture(t *testing.T) (*BindingManager, ExecutableTrust) {
	t.Helper()
	launcher := &bindingLauncher{}
	trust := testManager(t, true, launcher)
	trust.layout.ProviderBindingDir = filepath.Join(filepath.Dir(trust.layout.ProviderTrustDir), "state", "provider-bindings")
	trust.foundation.InstallationDigest = tagged("foundation-installation")
	declaration := writeTrustPackage(t, trust)
	root := providerTestPrefix(declaration.ExecutablePath)
	manager, err := newBindingManager(ssiagpaths.ScopeUser, trust.layout, trust.registry, trust, trust.now, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	launcher.manager = trust
	return manager, declaration
}

func emptyBindingFixture(t *testing.T, foundationReceipt string) (*BindingManager, *TrustManager) {
	t.Helper()
	launcher := &bindingLauncher{}
	trust := testManager(t, true, launcher)
	trust.layout.ProviderBindingDir = filepath.Join(filepath.Dir(trust.layout.ProviderTrustDir), "state", "provider-bindings")
	trust.foundation.InstallationDigest = foundationReceipt
	root := filepath.Join(filepath.Dir(trust.layout.ProviderTrustDir), "prefix")
	manager, err := newBindingManager(ssiagpaths.ScopeUser, trust.layout, trust.registry, trust, trust.now, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	launcher.manager = trust
	return manager, trust
}

func committedBindingReceipt(operationID string) stavprotocol.Receipt {
	return stavprotocol.Receipt{
		CandidateDigest: tagged("candidate-" + operationID),
		Commit: stavprotocol.CommitResult{
			EventDigest: tagged("event-" + operationID), EventID: "b90e1205-1b3b-4e47-9b91-1cd624cd87cd",
			Sequence: 1, State: "committed", Timestamp: "2026-08-14T12:00:00.000000000Z",
		},
		Disposition: "committed", ReasonCode: stavprotocol.ReasonReceiptCommitted, RequestID: operationID,
		Schema: stavprotocol.SchemaReceipt, TOPSID: trustTestTOPSID,
	}
}

func markBindingAudited(manager *BindingManager, attempt ProviderBindingAttempt) (ProviderBindingAttempt, error) {
	receipt := committedBindingReceipt(attempt.OperationID)
	return manager.MarkAudited(attempt.ProviderName, attempt.OperationID, receipt.CandidateDigest, receipt)
}

func TestProviderBindingLifecycleConvergesForwardReverseAndRecovery(t *testing.T) {
	manager, declaration := bindingFixture(t)
	inventory, found, err := manager.Inventory("native")
	if err != nil || !found || len(inventory.Installations) != 1 || inventory.Installations[0].InstallationID != declaration.DeclarationDigest ||
		inventory.Installations[0].CompatibilityState != "exact" || inventory.InventoryDigest != objectDigest(inventory, "inventory_digest") ||
		inventory.OperationalAccessEnabled || inventory.ProviderOperationsEnabled || inventory.SecretChannelEnabled {
		t.Fatalf("unexpected exact provider inventory: %+v", inventory)
	}
	status, found, err := manager.Status("native")
	if err != nil || !found || status.StateDigest != "absent" || status.BindingState != "unbound" || status.RecoveryRequired {
		t.Fatalf("unexpected initial status: %+v err=%v", status, err)
	}
	plan, found, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: "absent", Reason: "activate exact tested metadata provider",
	})
	if err != nil || !found || !plan.Applicable || !plan.Changed || plan.Actions[0].Direction != "forward" || plan.PlanDigest != objectDigest(plan, "plan_digest") {
		t.Fatalf("unexpected forward plan: %+v err=%v", plan, err)
	}
	attempt, found, already, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"}, testBindingAuditIdentity)
	if err != nil || !found || already || attempt.Stage != "prepared" {
		t.Fatalf("unexpected prepared attempt: %+v err=%v", attempt, err)
	}
	attempt, err = manager.VerifyCandidate(context.Background(), attempt)
	if err != nil || attempt.Stage != "candidate_verified" || attempt.CandidateVerifiedAt == nil {
		t.Fatalf("candidate was not independently verified: %+v err=%v", attempt, err)
	}
	previousAuditDigest, newAuditDigest, err := manager.AuditDigests(attempt)
	if err != nil || !validDigest(previousAuditDigest) || previousAuditDigest == "absent" || newAuditDigest != attempt.TargetState.StateDigest {
		t.Fatalf("initial absence was not normalized into a safe STAV digest pair: previous=%q new=%q err=%v", previousAuditDigest, newAuditDigest, err)
	}
	if _, err := manager.MarkAudited("native", attempt.OperationID, tagged("counterfeit-candidate"), stavprotocol.Receipt{
		Disposition: "committed", RequestID: attempt.OperationID, TOPSID: trustTestTOPSID,
	}); err == nil {
		t.Fatal("structurally invalid STAV receipt advanced the provider binding attempt")
	}
	attempt, err = markBindingAudited(manager, attempt)
	if err != nil || attempt.Stage != "audited" || attempt.AuditedAt == nil || attempt.ReceiptDigest == "not_applicable" {
		t.Fatalf("attempt was not audit-bound: %+v err=%v", attempt, err)
	}
	result, err := manager.Commit("native", attempt.OperationID, false)
	if err != nil || !result.Changed || result.BindingState != "bound" || result.InstallationID != declaration.DeclarationDigest || result.RecoveryRequired {
		t.Fatalf("binding did not commit: %+v err=%v", result, err)
	}
	completed, found, err := manager.AttemptStatus("native", attempt.OperationID)
	if err != nil || !found || completed.Operation != "apply-status" || completed.StateDigest != result.StateDigest || completed.RecoveryRequired {
		t.Fatalf("completed operation was not durably queryable: %+v err=%v", completed, err)
	}
	status, _, err = manager.Status("native")
	if err != nil || status.BindingState != "bound" || status.Generation != 1 || status.StateDigest != result.StateDigest || status.AttemptState != "none" {
		t.Fatalf("committed state did not settle: %+v err=%v", status, err)
	}
	trustStatus, found := manager.trust.Show("native")
	if !found || trustStatus.TrustState != "unverified" || trustStatus.AdapterVersion != declaration.AdapterVersion {
		t.Fatalf("provider trust did not consume the managed active binding: %+v", trustStatus)
	}

	// A no-op plan is deterministic and creates no durable attempt.
	noChange, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: status.StateDigest, Reason: "retain exact binding",
	})
	if err != nil || noChange.Changed || noChange.Actions[0].Kind != "retain" {
		t.Fatalf("unexpected retention plan: %+v err=%v", noChange, err)
	}
	noChangeAttempt, _, already, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: noChange.PlanDigest, ExpectedStateDigest: status.StateDigest}, testBindingAuditIdentity)
	if err != nil || !already {
		t.Fatalf("no-op apply was not idempotent: %+v err=%v", noChangeAttempt, err)
	}
	noChangeResult, err := manager.NoChangeResult("native", noChangeAttempt)
	if err != nil || noChangeResult.Changed || noChangeResult.StateDigest != status.StateDigest {
		t.Fatalf("unexpected no-op result: %+v err=%v", noChangeResult, err)
	}

	// Preserve an explicit unbound state, then reverse to the retained predecessor.
	unbind, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: "not_applicable", ExpectedStateDigest: status.StateDigest, Reason: "administratively disable metadata binding",
	})
	if err != nil {
		t.Fatal(err)
	}
	attempt, _, _, err = manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: unbind.PlanDigest, ExpectedStateDigest: status.StateDigest}, testBindingAuditIdentity)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err = manager.VerifyCandidate(context.Background(), attempt)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err = markBindingAudited(manager, attempt)
	if err != nil {
		t.Fatal(err)
	}
	unbound, err := manager.Commit("native", attempt.OperationID, false)
	if err != nil || unbound.BindingState != "unbound" || unbound.PreviousInstallationID != declaration.DeclarationDigest {
		t.Fatalf("explicit unbind did not preserve predecessor: %+v err=%v", unbound, err)
	}
	rollback, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: unbound.StateDigest, Reason: "restore retained exact predecessor",
	})
	if err != nil || rollback.Actions[0].Direction != "reverse" {
		t.Fatalf("rollback was not explicit reverse traversal: %+v err=%v", rollback, err)
	}
	attempt, _, _, err = manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: rollback.PlanDigest, ExpectedStateDigest: unbound.StateDigest}, testBindingAuditIdentity)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a restart after prepare. Recovery uniquely resumes verification,
	// audit, commit, and attempt cleanup from durable evidence.
	pending, _, err := manager.Pending("native", ProviderBindingRecoveryRequest{ExpectedStateDigest: unbound.StateDigest, Reason: "resume interrupted exact rollback"})
	if err != nil || pending.Stage != "prepared" {
		t.Fatalf("prepared recovery evidence missing: %+v err=%v", pending, err)
	}
	pending, err = manager.VerifyCandidate(context.Background(), pending)
	if err != nil {
		t.Fatal(err)
	}
	pending, err = markBindingAudited(manager, pending)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := manager.Commit("native", pending.OperationID, true)
	if err != nil || !recovered.Recovered || recovered.BindingState != "bound" || recovered.InstallationID != declaration.DeclarationDigest {
		t.Fatalf("recovery did not converge: %+v err=%v", recovered, err)
	}
}

func TestProviderBindingRejectsStaleChangedAndUnsafeEvidence(t *testing.T) {
	manager, declaration := bindingFixture(t)
	if _, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: tagged("stale"), Reason: "stale plan",
	}); !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("stale compare-and-swap was accepted: %v", err)
	}
	plan, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: "absent", Reason: "plan before bytes drift",
	})
	if err != nil {
		t.Fatal(err)
	}
	attempt, _, _, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"}, testBindingAuditIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declaration.ExecutablePath, []byte("changed provider fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.VerifyCandidate(context.Background(), attempt); !errors.Is(err, ErrBindingInstallation) {
		t.Fatalf("changed candidate bytes were accepted: %v", err)
	}

	unsafeRoot := filepath.Join(filepath.Dir(manager.layout.ProviderBindingDir), "unsafe-bindings")
	if err := os.Symlink(filepath.Dir(manager.layout.ProviderBindingDir), unsafeRoot); err != nil {
		t.Fatal(err)
	}
	manager.layout.ProviderBindingDir = unsafeRoot
	if _, _, err := manager.Status("native"); err == nil {
		t.Fatal("symlinked binding state root was accepted")
	}
}

func TestProviderBindingReasonMatchesCanonicalUnicodeBound(t *testing.T) {
	if !validBindingReason(strings.Repeat("界", 1024)) {
		t.Fatal("canonical 1024-character UTF-8 reason was rejected")
	}
	for name, value := range map[string]string{
		"too_long": strings.Repeat("界", 1025),
		"newline":  "unsafe\nreason",
		"invalid":  string([]byte{0xff}),
	} {
		if validBindingReason(value) {
			t.Fatalf("%s reason was accepted", name)
		}
	}
}

func TestProviderBindingRevalidatesEveryResumedStage(t *testing.T) {
	for _, stage := range []string{"candidate_verified", "audited"} {
		t.Run(stage, func(t *testing.T) {
			manager, declaration := bindingFixture(t)
			plan, _, err := manager.Plan("native", ProviderBindingPlanRequest{
				InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: "absent", Reason: "exercise resumed-stage verification",
			})
			if err != nil {
				t.Fatal(err)
			}
			attempt, _, _, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"}, testBindingAuditIdentity)
			if err != nil {
				t.Fatal(err)
			}
			attempt, err = manager.VerifyCandidate(context.Background(), attempt)
			if err != nil {
				t.Fatal(err)
			}
			if stage == "audited" {
				attempt, err = markBindingAudited(manager, attempt)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(declaration.ExecutablePath, []byte("changed after durable "+stage), 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err := manager.VerifyCandidate(context.Background(), attempt); !errors.Is(err, ErrBindingInstallation) {
				t.Fatalf("changed bytes after %s were accepted: %v", stage, err)
			}
			if stage == "audited" {
				if _, err := manager.Commit("native", attempt.OperationID, true); !errors.Is(err, ErrBindingInstallation) {
					t.Fatalf("commit accepted changed audited candidate: %v", err)
				}
			}
		})
	}
}

func TestProviderBindingRetryAndPostStateRecoveryRemainConservative(t *testing.T) {
	manager, declaration := bindingFixture(t)
	plan, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: "absent", Reason: "exercise exact retry and post-state recovery",
	})
	if err != nil {
		t.Fatal(err)
	}
	attempt, _, _, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"}, testBindingAuditIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: tagged("stale-retry")}, testBindingAuditIdentity); !errors.Is(err, ErrBindingConflict) {
		t.Fatalf("same-plan retry bypassed expected-state compare-and-swap: %v", err)
	}
	attempt, err = manager.VerifyCandidate(context.Background(), attempt)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err = markBindingAudited(manager, attempt)
	if err != nil {
		t.Fatal(err)
	}
	if err := withBindingStore(manager.layout.ProviderBindingDir, "native", true, func(store *bindingStore) error {
		return store.write("state.json", attempt.TargetState)
	}); err != nil {
		t.Fatal(err)
	}
	status, found, err := manager.AttemptStatus("native", attempt.OperationID)
	if err != nil || !found || !status.RecoveryRequired || status.AttemptState != "audited" || status.ReasonCode != "symphony.ssiag.provider.binding.recovery_required" {
		t.Fatalf("post-state/pre-marker interruption reported success: %+v err=%v", status, err)
	}
}

func TestProviderBindingRecoveryAcrossEveryDurableStage(t *testing.T) {
	for _, crashStage := range []string{"prepared", "candidate_verified", "audited", "state_written", "committed"} {
		t.Run(crashStage, func(t *testing.T) {
			manager, declaration := bindingFixture(t)
			plan, _, err := manager.Plan("native", ProviderBindingPlanRequest{
				InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: "absent", Reason: "exercise durable recovery stage " + crashStage,
			})
			if err != nil {
				t.Fatal(err)
			}
			attempt, _, _, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"}, testBindingAuditIdentity)
			if err != nil {
				t.Fatal(err)
			}
			if crashStage != "prepared" {
				attempt, err = manager.VerifyCandidate(context.Background(), attempt)
				if err != nil {
					t.Fatal(err)
				}
			}
			if crashStage == "audited" || crashStage == "state_written" || crashStage == "committed" {
				attempt, err = markBindingAudited(manager, attempt)
				if err != nil {
					t.Fatal(err)
				}
			}
			if crashStage == "state_written" || crashStage == "committed" {
				if err := withBindingStore(manager.layout.ProviderBindingDir, "native", true, func(store *bindingStore) error {
					if err := store.write("state.json", attempt.TargetState); err != nil {
						return err
					}
					if crashStage == "committed" {
						now := timestamp(manager.now())
						attempt.Stage, attempt.CommittedAt = "committed", &now
						attempt.AttemptDigest = objectDigest(attempt, "attempt_digest")
						return store.write("attempt.json", attempt)
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}

			pending, found, err := manager.Pending("native", ProviderBindingRecoveryRequest{ExpectedStateDigest: func() string {
				if crashStage == "state_written" || crashStage == "committed" {
					return attempt.TargetState.StateDigest
				}
				return "absent"
			}(), Reason: "resume exact durable stage"})
			if err != nil || !found || pending.AuditIdentity != testBindingAuditIdentity {
				t.Fatalf("durable %s evidence lost its original audit identity: %+v err=%v", crashStage, pending, err)
			}
			status, _, err := manager.AttemptStatus("native", pending.OperationID)
			if err != nil || !status.RecoveryRequired {
				t.Fatalf("durable %s interruption did not require recovery: %+v err=%v", crashStage, status, err)
			}
			pending, err = manager.VerifyCandidate(context.Background(), pending)
			if err != nil {
				t.Fatal(err)
			}
			if pending.Stage == "candidate_verified" {
				pending, err = markBindingAudited(manager, pending)
				if err != nil {
					t.Fatal(err)
				}
			}
			result, err := manager.Commit("native", pending.OperationID, true)
			if err != nil || !result.Recovered || result.BindingState != "bound" || result.InstallationID != declaration.DeclarationDigest {
				t.Fatalf("durable %s recovery did not converge: %+v err=%v", crashStage, result, err)
			}
		})
	}
}

func TestProviderBindingInventoryAndStateFailClosedOnBrokenEvidence(t *testing.T) {
	t.Run("missing_package", func(t *testing.T) {
		manager, declaration := bindingFixture(t)
		if err := os.Remove(declaration.ExecutablePath); err != nil {
			t.Fatal(err)
		}
		if _, _, err := manager.Inventory("native"); err == nil {
			t.Fatal("missing provider package was silently omitted")
		}
	})
	t.Run("changed_receipt", func(t *testing.T) {
		manager, declaration := bindingFixture(t)
		receiptPath := filepath.Join(providerTestPrefix(declaration.ExecutablePath), "share", "symphony", "receipts", macOSKeychainPackageID, declaration.AdapterVersion, "install-receipt.json")
		if err := os.WriteFile(receiptPath, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := manager.Inventory("native"); err == nil {
			t.Fatal("changed provider receipt was silently omitted")
		}
	})
	t.Run("symlinked_candidate", func(t *testing.T) {
		manager, declaration := bindingFixture(t)
		receiptRoot := filepath.Join(providerTestPrefix(declaration.ExecutablePath), "share", "symphony", "receipts", macOSKeychainPackageID)
		if err := os.Symlink(declaration.AdapterVersion, filepath.Join(receiptRoot, "linked-version")); err != nil {
			t.Fatal(err)
		}
		if _, _, err := manager.Inventory("native"); err == nil {
			t.Fatal("symlinked provider candidate was silently omitted")
		}
	})
	t.Run("cumulative_multi_root_bound", func(t *testing.T) {
		manager, trust := emptyBindingFixture(t, tagged("foundation-installation"))
		first := writeTrustPackageVersion(t, trust, "0.1.0", []byte("first-root provider fixture"))
		secondRoot := filepath.Join(filepath.Dir(trust.layout.ProviderTrustDir), "z-prefix")
		_ = writeTrustPackageVersionAtPrefix(t, trust, secondRoot, "0.2.0", []byte("legacy-root provider fixture"))
		receiptRoot := filepath.Join(secondRoot, "share", "symphony", "receipts", macOSKeychainPackageID)
		for index := 0; index < 127; index++ {
			if err := os.Mkdir(filepath.Join(receiptRoot, fmt.Sprintf("candidate-%03d", index)), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		manager.roots = []string{providerTestPrefix(first.ExecutablePath)}
		if _, _, err := manager.Inventory("native"); err == nil || !strings.Contains(err.Error(), "cumulative 128-entry bound") {
			t.Fatalf("multi-root provider inventory escaped its cumulative bound: %v", err)
		}
	})
	t.Run("incompatible_protocol", func(t *testing.T) {
		manager, declaration := bindingFixture(t)
		receiptPath := filepath.Join(providerTestPrefix(declaration.ExecutablePath), "share", "symphony", "receipts", macOSKeychainPackageID, declaration.AdapterVersion, "install-receipt.json")
		receipt, _, _, err := readAdapterReceipt(receiptPath, ssiagpaths.ScopeUser)
		if err != nil {
			t.Fatal(err)
		}
		receipt.Protocol = "symphony.knowledge.install-receipt.future"
		receipt.ReceiptDigest = receiptDigest(receipt)
		writeJSONFile(t, receiptPath, receipt)
		inventory, _, err := manager.Inventory("native")
		if err != nil || len(inventory.Installations) != 1 || inventory.Installations[0].CompatibilityState != "incompatible" {
			t.Fatalf("incompatible provider protocol was not explicit: %+v err=%v", inventory, err)
		}
	})
	t.Run("unknown_state_preserved", func(t *testing.T) {
		manager, _ := bindingFixture(t)
		statePath := filepath.Join(manager.layout.ProviderBindingDir, "native", "state.json")
		if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
			t.Fatal(err)
		}
		unknown := map[string]any{
			"protocol": "symphony.ssiag.provider-binding-state.v2", "future_critical_state": true,
		}
		encoded, _ := json.Marshal(unknown)
		if err := os.WriteFile(statePath, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := manager.Status("native"); err == nil {
			t.Fatal("unknown future binding state was accepted")
		}
		after, err := os.ReadFile(statePath)
		if err != nil || string(after) != string(encoded) {
			t.Fatalf("unknown future state was not preserved byte-for-byte: %q err=%v", after, err)
		}
	})
}

func TestProviderBindingRejectsCompetingAttempt(t *testing.T) {
	manager, declaration := bindingFixture(t)
	plan, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: "absent", Reason: "prepare the first serialized attempt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"}, testBindingAuditIdentity); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: "not_applicable", ExpectedStateDigest: "absent", Reason: "compete with an unfinished attempt",
	}); !errors.Is(err, ErrBindingRecoveryRequired) {
		t.Fatalf("competing provider binding attempt was admitted: %v", err)
	}
}

func TestProviderBindingInstallationOrderConvergesFromObservedEvidence(t *testing.T) {
	t.Run("foundation_first", func(t *testing.T) {
		manager, trust := emptyBindingFixture(t, tagged("foundation-installation"))
		inventory, _, err := manager.Inventory("native")
		if err != nil || len(inventory.Installations) != 0 {
			t.Fatalf("foundation-only inventory was not empty: %+v err=%v", inventory, err)
		}
		declaration := writeTrustPackage(t, trust)
		inventory, _, err = manager.Inventory("native")
		if err != nil || len(inventory.Installations) != 1 || inventory.Installations[0].InstallationID != declaration.DeclarationDigest || inventory.Installations[0].CompatibilityState != "exact" {
			t.Fatalf("adapter arrival did not converge foundation-first staging: %+v err=%v", inventory, err)
		}
	})
	t.Run("adapter_first", func(t *testing.T) {
		manager, trust := emptyBindingFixture(t, "not_applicable")
		_ = writeTrustPackage(t, trust)
		inventory, _, err := manager.Inventory("native")
		if err != nil || len(inventory.Installations) != 1 || inventory.Installations[0].CompatibilityState != "incompatible" || inventory.Installations[0].ReasonCode != "symphony.ssiag.provider.binding.foundation_unreceipted" {
			t.Fatalf("adapter-first staging hid its missing foundation receipt: %+v err=%v", inventory, err)
		}
		trust.foundation.InstallationDigest = tagged("foundation-installation")
		inventory, _, err = manager.Inventory("native")
		if err != nil || len(inventory.Installations) != 1 || inventory.Installations[0].CompatibilityState != "exact" {
			t.Fatalf("foundation arrival did not converge adapter-first staging: %+v err=%v", inventory, err)
		}
	})
	t.Run("both_staged", func(t *testing.T) {
		manager, declaration := bindingFixture(t)
		inventory, _, err := manager.Inventory("native")
		if err != nil || len(inventory.Installations) != 1 || inventory.Installations[0].InstallationID != declaration.DeclarationDigest || inventory.Installations[0].CompatibilityState != "exact" {
			t.Fatalf("simultaneously staged exact pair did not converge: %+v err=%v", inventory, err)
		}
	})
}

func TestProviderBindingMultipleExactCandidatesRequireExplicitIdentity(t *testing.T) {
	manager, trust := emptyBindingFixture(t, tagged("foundation-installation"))
	first := writeTrustPackageVersion(t, trust, "0.1.0", []byte("first exact provider fixture"))
	second := writeTrustPackageVersion(t, trust, "0.2.0", []byte("second exact provider fixture"))
	inventory, _, err := manager.Inventory("native")
	if err != nil || len(inventory.Installations) != 2 || inventory.Installations[0].CompatibilityState != "exact" || inventory.Installations[1].CompatibilityState != "exact" {
		t.Fatalf("multiple exact candidates were not reported explicitly: %+v err=%v", inventory, err)
	}
	status, _, err := manager.Status("native")
	if err != nil || status.BindingState != "unbound" || status.InstallationID != "not_applicable" {
		t.Fatalf("inventory ambiguity silently selected a candidate: %+v err=%v", status, err)
	}
	wanted := second.DeclarationDigest
	if first.DeclarationDigest > second.DeclarationDigest {
		wanted = first.DeclarationDigest
	}
	plan, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: wanted, ExpectedStateDigest: "absent", Reason: "select one exact opaque identity explicitly",
	})
	if err != nil || plan.InstallationID != wanted || !plan.Changed {
		t.Fatalf("explicit exact selection did not resolve multiple candidates: %+v err=%v", plan, err)
	}
}

func TestProviderBindingExpiredIssuedPlanFailsBeforeAttempt(t *testing.T) {
	manager, declaration := bindingFixture(t)
	plan, _, err := manager.Plan("native", ProviderBindingPlanRequest{
		InstallationID: declaration.DeclarationDigest, ExpectedStateDigest: "absent", Reason: "issue a bounded plan before advancing time",
	})
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := manager.now()
	manager.now = func() time.Time { return issuedAt.Add(providerBindingPlanLifetime + time.Second) }
	if _, _, _, err := manager.Prepare("native", ProviderBindingApplyRequest{PlanDigest: plan.PlanDigest, ExpectedStateDigest: "absent"}, testBindingAuditIdentity); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired provider binding plan was accepted: %v", err)
	}
	status, _, err := manager.Status("native")
	if err != nil || status.AttemptState != "none" || status.RecoveryRequired {
		t.Fatalf("expired plan created durable attempt evidence: %+v err=%v", status, err)
	}
}

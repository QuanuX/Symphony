package provider

import (
	"context"
	"testing"
)

type fakeReadinessLauncher struct {
	fakeLauncher
	observation AdapterReadinessObservation
	err         error
}

func (f *fakeReadinessLauncher) ObserveReadiness(context.Context, ExecutableTrust) (AdapterReadinessObservation, error) {
	return f.observation, f.err
}

func TestProviderReadinessKeepsThreeLayersAndOperationsDisabled(t *testing.T) {
	launcher := &fakeReadinessLauncher{observation: validReadinessFixture()}
	manager := testManager(t, true, launcher)
	declaration := writeTrustPackage(t, manager)
	launcher.response = verifiedResponse(manager, declaration, trustUUID(1), trustUUID(2))

	result, found := manager.ObserveReadiness(context.Background(), "native")
	if !found || result.ReadinessState != "readiness_proven_operations_disabled" {
		t.Fatalf("readiness observation was not surfaced: %+v", result)
	}
	if result.OperationalAccessEnabled || result.ProviderOperationsEnabled || result.SecretChannelEnabled ||
		result.Observation.OperationalEligibility.Evaluated || result.Observation.AuthorizationDecisionMade {
		t.Fatalf("readiness observation enabled an operation: %+v", result)
	}
	if result.ResultDigest == "" || result.ReadOnly != true || result.CallerClassUsed || result.Canonical {
		t.Fatalf("readiness result metadata is incomplete: %+v", result)
	}
}

func TestProviderReadinessRejectsOperationalOrMalformedObservation(t *testing.T) {
	for name, mutate := range map[string]func(*AdapterReadinessObservation){
		"operational access": func(value *AdapterReadinessObservation) { value.OperationalAccessEnabled = true },
		"eligibility claim":  func(value *AdapterReadinessObservation) { value.OperationalEligibility.Evaluated = true },
		"unsorted reasons":   func(value *AdapterReadinessObservation) { value.ReasonCodes = []string{"z", "a"} },
		"raw evidence":       func(value *AdapterReadinessObservation) { value.SigningIdentifier = "contains spaces" },
	} {
		t.Run(name, func(t *testing.T) {
			observation := validReadinessFixture()
			mutate(&observation)
			if err := validateReadinessObservation(observation); err == nil {
				t.Fatal("unsafe readiness observation was accepted")
			}
		})
	}
}

func validReadinessFixture() AdapterReadinessObservation {
	return AdapterReadinessObservation{
		Protocol: AdapterReadinessProtocol, MetadataOnly: true,
		StructuralValidation:   ReadinessLayer{State: "valid", Evaluated: true, ReasonCode: "symphony.ssiag.provider.readiness.structural_valid"},
		PolicyMatch:            ReadinessLayer{State: "matched", Evaluated: true, ReasonCode: "symphony.ssiag.provider.readiness.policy_matched"},
		OperationalEligibility: ReadinessLayer{State: "disabled", Evaluated: false, ReasonCode: "symphony.ssiag.provider.readiness.phase_10b_operational_gate"},
		AppLikeBundleObserved:  true, ProvisioningProfileFileState: "regular_safe", StaticSignatureState: "valid", DynamicSignatureState: "valid",
		SigningIdentifier: "not_observed", DesignatedRequirementDigest: tagged("designated"), PolicyRequirementDigest: tagged("policy"),
		SecuritySessionObserved: true, SecuritySessionGraphical: true,
		ReasonCodes: []string{
			"symphony.ssiag.provider.readiness.phase_10b_operational_gate",
			"symphony.ssiag.provider.readiness.policy_matched",
			"symphony.ssiag.provider.readiness.structural_valid",
		},
	}
}

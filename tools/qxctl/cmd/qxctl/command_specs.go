package main

import (
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/spf13/cobra"
)

const (
	featureQXCTL                   = "ssfv:symphony:qxctl"
	featureBindings                = "ssfv:symphony:qxctl.engine-bindings"
	featureCommandRegistry         = "ssfv:symphony:qxctl.command-registry"
	featureLifecycle               = "ssfv:symphony:qxctl.lifecycle-convergence"
	featureHost                    = "ssfv:symphony:qxctl.linux-host-receptor"
	featureMaestro                 = "ssfv:symphony:qxctl.maestro-administration"
	featureSessions                = "ssfv:symphony:qxctl.authenticated-sessions"
	featureSSIAG                   = "ssfv:symphony:qxctl.ssiag-administration"
	featureSTAV                    = "ssfv:symphony:qxctl.stav-administration"
	featureValidation              = "ssfv:symphony:qxctl.governed-validation"
	featureSKVI                    = "ssfv:symphony:skvi-engine"
	featureSKVIAssurance           = "ssfv:symphony:skvi-engine.structural-index-assurance"
	featureSKVIProposal            = "ssfv:symphony:skvi-engine.content-addressed-index-change-proposals"
	featureSKVIProjection          = "ssfv:symphony:skvi-engine.disposable-structural-projection"
	featureSCLV                    = "ssfv:symphony:sclv-engine"
	featureSCLVAssurance           = "ssfv:symphony:sclv-engine.append-only-ledger-assurance"
	featureSCLVProposal            = "ssfv:symphony:sclv-engine.evidence-bound-append-proposals"
	featureSCLVRecovery            = "ssfv:symphony:sclv-engine.forward-only-closure-recovery"
	featureSCLVProjection          = "ssfv:symphony:sclv-engine.disposable-provider-neutral-history"
	featureSCLVEvidence            = "ssfv:symphony:sclv-engine.provider-neutral-evidence-normalization"
	featureSACV                    = "ssfv:symphony:sacv-engine"
	featureSACVConformance         = "ssfv:symphony:sacv-engine.api-contract-conformance"
	featureSACVCompatibility       = "ssfv:symphony:sacv-engine.openapi-compatibility-evidence"
	featureSACVProposal            = "ssfv:symphony:sacv-engine.contract-registration-proposal"
	featureSACVProjection          = "ssfv:symphony:sacv-engine.contract-inventory-projection"
	featureSODV                    = "ssfv:symphony:sodv-engine"
	featureSODVLedger              = "ssfv:symphony:sodv-engine.release-ledger-validation"
	featureSODVVerification        = "ssfv:symphony:sodv-engine.observed-publication-verification"
	featureSODVProposal            = "ssfv:symphony:sodv-engine.forward-release-record-proposal"
	featureSODVRecovery            = "ssfv:symphony:sodv-engine.interrupted-publication-reconciliation"
	featureSODVProjection          = "ssfv:symphony:sodv-engine.release-transaction-projection"
	featureSSFV                    = "ssfv:symphony:ssfv-engine"
	featureSSFVSnapshot            = "ssfv:symphony:ssfv-engine.catalog-integrity-snapshot"
	featureSSFVComparison          = "ssfv:symphony:ssfv-engine.semantic-freshness-comparison"
	featureSSFVProposal            = "ssfv:symphony:ssfv-engine.catalog-change-proposal"
	featureSSFVProjection          = "ssfv:symphony:ssfv-engine.semantic-graph-projection"
	featureAdministrationAssurance = "ssfv:symphony:ssfv-engine.administration-assurance"
	backendFeatureCoordinator      = "ssfv:symphony:knowledge-session-coordinator"
	backendFeatureAuthorityEpochs  = "ssfv:symphony:knowledge-session-coordinator.authority-epochs"
	backendFeatureLifecycleApply   = "ssfv:symphony:knowledge-session-coordinator.lifecycle-apply-coordination"
	backendFeatureLifecyclePlan    = "ssfv:symphony:knowledge-session-coordinator.lifecycle-planning"
	backendFeatureReconciliation   = "ssfv:symphony:knowledge-session-coordinator.reconciliation"
	backendFeatureSemanticMaintain = "ssfv:symphony:knowledge-session-coordinator.semantic-maintenance"
	backendFeatureMaestroPresence  = "ssfv:symphony:maestro-presence-authority"
	backendFeatureMaestroInventory = "ssfv:symphony:maestro-presence-authority.complete-inventory"
	backendFeaturePlatform         = "ssfv:symphony:platform"
	backendFeatureSSIAG            = "ssfv:symphony:ssiag-foundation"
	backendFeatureSSIAGPolicy      = "ssfv:symphony:ssiag-foundation.policy-administration"
	backendFeatureSSIAGProviders   = "ssfv:symphony:ssiag-foundation.provider-metadata-registry"
	backendFeatureKeychainMetadata = "ssfv:symphony:ssiag.macos-keychain-metadata"
	backendFeatureSTAV             = "ssfv:symphony:stav-append-authority"
	backendFeatureSTAVQuery        = "ssfv:symphony:stav-append-authority.authorized-query"
	backendFeatureSTAVDurability   = "ssfv:symphony:stav-append-authority.ledger-durability"
	backendFeatureValidator        = "ssfv:symphony:symphony-validator"
)

// reviewedBackendFeatureBindings is the ratified semantic bridge from stable
// qxcmd identities to the backend feature interactions they administer. The
// command's qxctl-owned wrapper binding remains first-class and distinct. This
// table adds no grammar, dispatch, operation identity, availability, or
// authority; the SSFV engine independently evaluates it against the canonical
// administration profile.
var reviewedBackendFeatureBindings = map[string][]commandregistry.FeatureBinding{
	"inventory": {{FeatureID: backendFeaturePlatform, Interaction: "discover"}},
	"knowledge.lifecycle.apply": {
		{FeatureID: backendFeatureLifecycleApply, Interaction: "apply"},
		{FeatureID: backendFeatureMaestroPresence, Interaction: "lifecycle"},
	},
	"knowledge.lifecycle.apply-recover": {{FeatureID: backendFeatureLifecycleApply, Interaction: "recover"}},
	"knowledge.lifecycle.apply-status":  {{FeatureID: backendFeatureLifecycleApply, Interaction: "query"}},
	"knowledge.lifecycle.boot":          {{FeatureID: backendFeatureLifecyclePlan, Interaction: "lifecycle"}},
	"knowledge.lifecycle.observe":       {{FeatureID: backendFeatureLifecyclePlan, Interaction: "inspect"}},
	"knowledge.lifecycle.report":        {{FeatureID: backendFeatureLifecyclePlan, Interaction: "query"}},
	"knowledge.lifecycle.status":        {{FeatureID: backendFeatureLifecyclePlan, Interaction: "query"}},
	"knowledge.reconcile.begin":         {{FeatureID: backendFeatureReconciliation, Interaction: "lifecycle"}},
	"knowledge.reconcile.checkpoint":    {{FeatureID: backendFeatureReconciliation, Interaction: "lifecycle"}},
	"knowledge.reconcile.close":         {{FeatureID: backendFeatureReconciliation, Interaction: "lifecycle"}},
	"knowledge.reconcile.compatibility": {{FeatureID: backendFeatureReconciliation, Interaction: "query"}},
	"knowledge.reconcile.recover":       {{FeatureID: backendFeatureReconciliation, Interaction: "recover"}},
	"knowledge.reconcile.status":        {{FeatureID: backendFeatureReconciliation, Interaction: "query"}},
	"knowledge.session.begin":           {{FeatureID: backendFeatureAuthorityEpochs, Interaction: "lifecycle"}},
	"knowledge.session.checkpoint":      {{FeatureID: backendFeatureAuthorityEpochs, Interaction: "lifecycle"}},
	"knowledge.session.close":           {{FeatureID: backendFeatureAuthorityEpochs, Interaction: "lifecycle"}},
	"knowledge.session.features.begin": {
		{FeatureID: backendFeatureSemanticMaintain, Interaction: "lifecycle"},
	},
	"knowledge.session.features.checkpoint": {
		{FeatureID: backendFeatureSemanticMaintain, Interaction: "lifecycle"},
	},
	"knowledge.session.features.close": {
		{FeatureID: backendFeatureSemanticMaintain, Interaction: "lifecycle"},
	},
	"knowledge.session.features.recover": {{FeatureID: backendFeatureSemanticMaintain, Interaction: "recover"}},
	"knowledge.session.features.status":  {{FeatureID: backendFeatureSemanticMaintain, Interaction: "query"}},
	"knowledge.session.recover":          {{FeatureID: backendFeatureAuthorityEpochs, Interaction: "recover"}},
	"knowledge.session.status":           {{FeatureID: backendFeatureAuthorityEpochs, Interaction: "query"}},
	"knowledge.session.transition":       {{FeatureID: backendFeatureAuthorityEpochs, Interaction: "lifecycle"}},
	"maestro.inspect":                    {{FeatureID: backendFeatureMaestroPresence, Interaction: "inspect"}},
	"maestro.inventory":                  {{FeatureID: backendFeatureMaestroInventory, Interaction: "query"}},
	"maestro.recover":                    {{FeatureID: backendFeatureMaestroPresence, Interaction: "recover"}},
	"maestro.status":                     {{FeatureID: backendFeatureMaestroPresence, Interaction: "query"}},
	"module.check":                       {{FeatureID: backendFeatureCoordinator, Interaction: "validate"}},
	"module.inspect":                     {{FeatureID: backendFeatureCoordinator, Interaction: "inspect"}},
	"modules":                            {{FeatureID: backendFeaturePlatform, Interaction: "discover"}},
	"ssiag.doctor":                       {{FeatureID: backendFeatureSSIAG, Interaction: "validate"}},
	"ssiag.policy.apply":                 {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "apply"}},
	"ssiag.policy.propose":               {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "propose"}},
	"ssiag.policy.recover":               {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "recover"}},
	"ssiag.policy.status":                {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "query"}},
	"ssiag.providers": {
		{FeatureID: backendFeatureSSIAGProviders, Interaction: "discover"},
		{FeatureID: backendFeatureKeychainMetadata, Interaction: "discover"},
	},
	"ssiag.status":  {{FeatureID: backendFeatureSSIAG, Interaction: "query"}},
	"stav.doctor":   {{FeatureID: backendFeatureSTAV, Interaction: "validate"}},
	"stav.query":    {{FeatureID: backendFeatureSTAVQuery, Interaction: "query"}},
	"stav.status":   {{FeatureID: backendFeatureSTAV, Interaction: "query"}},
	"stav.verify":   {{FeatureID: backendFeatureSTAVDurability, Interaction: "validate"}},
	"validate.scan": {{FeatureID: backendFeatureValidator, Interaction: "validate"}},
}

func registered(command *cobra.Command, key, featureID, interaction string) *cobra.Command {
	return commandregistry.Attach(command, commandSpec(key, featureID, interaction))
}

func registeredProposal(command *cobra.Command, key, featureID string) *cobra.Command {
	spec := commandSpec(key, featureID, "propose")
	spec.Mutability = "proposal_only"
	return commandregistry.Attach(command, spec)
}

func registeredEvidence(command *cobra.Command, key, featureID, interaction string) *cobra.Command {
	spec := commandSpec(key, featureID, interaction)
	spec.Mutability = "evidence_only"
	return commandregistry.Attach(command, spec)
}

func registeredMutation(command *cobra.Command, key, featureID, interaction, authority, recoveryID string) *cobra.Command {
	spec := commandSpec(key, featureID, interaction)
	spec.Mutability = "permission_backed_mutation"
	spec.AuthorityMode = authority
	if recoveryID != "" {
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:" + recoveryID)
	} else {
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:" + key)
	}
	return commandregistry.Attach(command, spec)
}

func commandSpec(key, featureID, interaction string) commandregistry.CommandSpec {
	featureBindings := []commandregistry.FeatureBinding{{FeatureID: featureID, Interaction: interaction}}
	featureBindings = append(featureBindings, reviewedBackendFeatureBindings[key]...)
	return commandregistry.CommandSpec{
		CommandID:                 "qxcmd:symphony:" + key,
		Status:                    "experimental",
		IntroducedIn:              "0.1.0-dev",
		ReplacementIDs:            []string{},
		FeatureBindings:           featureBindings,
		BackendOperationIDs:       []string{},
		Mutability:                "read_only",
		AuthorityMode:             "none",
		TargetScope:               "local",
		InputProtocols:            []string{},
		OutputProtocols:           []string{},
		ResultValidationProtocols: []string{},
		Noninteractive:            true,
	}
}

func structural(use string, result error) *cobra.Command {
	return commandregistry.Structural(use, usageOnlyArgs, result)
}

func stringPointer(value string) *string { return &value }

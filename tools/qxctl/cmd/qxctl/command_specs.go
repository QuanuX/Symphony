package main

import (
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/spf13/cobra"
)

const (
	featureQXCTL                       = "ssfv:symphony:qxctl"
	featureBindings                    = "ssfv:symphony:qxctl.engine-bindings"
	featureCommandRegistry             = "ssfv:symphony:qxctl.command-registry"
	featureLifecycle                   = "ssfv:symphony:qxctl.lifecycle-convergence"
	featureHost                        = "ssfv:symphony:qxctl.linux-host-receptor"
	featureInvariantAssurance          = "ssfv:symphony:qxctl.invariant-assurance"
	featureMaestro                     = "ssfv:symphony:qxctl.maestro-administration"
	featureSessions                    = "ssfv:symphony:qxctl.authenticated-sessions"
	featureSSIAG                       = "ssfv:symphony:qxctl.ssiag-administration"
	featureSTAV                        = "ssfv:symphony:qxctl.stav-administration"
	featureValidation                  = "ssfv:symphony:qxctl.governed-validation"
	featureSKVI                        = "ssfv:symphony:skvi-engine"
	featureSKVIAssurance               = "ssfv:symphony:skvi-engine.structural-index-assurance"
	featureSKVIProposal                = "ssfv:symphony:skvi-engine.content-addressed-index-change-proposals"
	featureSKVIProjection              = "ssfv:symphony:skvi-engine.disposable-structural-projection"
	featureSCLV                        = "ssfv:symphony:sclv-engine"
	featureSCLVAssurance               = "ssfv:symphony:sclv-engine.append-only-ledger-assurance"
	featureSCLVProposal                = "ssfv:symphony:sclv-engine.evidence-bound-append-proposals"
	featureSCLVRecovery                = "ssfv:symphony:sclv-engine.forward-only-closure-recovery"
	featureSCLVProjection              = "ssfv:symphony:sclv-engine.disposable-provider-neutral-history"
	featureSCLVEvidence                = "ssfv:symphony:sclv-engine.provider-neutral-evidence-normalization"
	featureSACV                        = "ssfv:symphony:sacv-engine"
	featureSACVConformance             = "ssfv:symphony:sacv-engine.api-contract-conformance"
	featureSACVCompatibility           = "ssfv:symphony:sacv-engine.openapi-compatibility-evidence"
	featureSACVProposal                = "ssfv:symphony:sacv-engine.contract-registration-proposal"
	featureSACVProjection              = "ssfv:symphony:sacv-engine.contract-inventory-projection"
	featureSODV                        = "ssfv:symphony:sodv-engine"
	featureSODVLedger                  = "ssfv:symphony:sodv-engine.release-ledger-validation"
	featureSODVVerification            = "ssfv:symphony:sodv-engine.observed-publication-verification"
	featureSODVProposal                = "ssfv:symphony:sodv-engine.forward-release-record-proposal"
	featureSODVRecovery                = "ssfv:symphony:sodv-engine.interrupted-publication-reconciliation"
	featureSODVProjection              = "ssfv:symphony:sodv-engine.release-transaction-projection"
	featureSAV                         = "ssfv:symphony:sav-engine"
	featureSAVCurrent                  = "ssfv:symphony:sav-engine.current-accord"
	featureSAVNamedVersion             = "ssfv:symphony:sav-engine.named-version"
	featureSAVCapsule                  = "ssfv:symphony:sav-engine.extension-capsule"
	featureSAVBlueprint                = "ssfv:symphony:sav-engine.installation-blueprint"
	featureSEV                         = "ssfv:symphony:sev-engine"
	featureSEVEvolution                = "ssfv:symphony:sev-engine.dynamic-evolution"
	featureSEVSCSEV                    = "ssfv:symphony:sev-engine.scsev"
	featureSEVNoveltyWatch             = "ssfv:symphony:sev-engine.novelty-watch"
	featureSEVLifecycleBinding         = "ssfv:symphony:sev-engine.lifecycle-binding"
	featureSSFV                        = "ssfv:symphony:ssfv-engine"
	featureSSFVSnapshot                = "ssfv:symphony:ssfv-engine.catalog-integrity-snapshot"
	featureSSFVComparison              = "ssfv:symphony:ssfv-engine.semantic-freshness-comparison"
	featureSSFVProposal                = "ssfv:symphony:ssfv-engine.catalog-change-proposal"
	featureSSFVProjection              = "ssfv:symphony:ssfv-engine.semantic-graph-projection"
	featureAdministrationAssurance     = "ssfv:symphony:ssfv-engine.administration-assurance"
	backendFeatureCoordinator          = "ssfv:symphony:knowledge-session-coordinator"
	backendFeatureAuthorityEpochs      = "ssfv:symphony:knowledge-session-coordinator.authority-epochs"
	backendFeatureLifecycleApply       = "ssfv:symphony:knowledge-session-coordinator.lifecycle-apply-coordination"
	backendFeatureLifecyclePlan        = "ssfv:symphony:knowledge-session-coordinator.lifecycle-planning"
	backendFeatureReconciliation       = "ssfv:symphony:knowledge-session-coordinator.reconciliation"
	backendFeatureSemanticMaintain     = "ssfv:symphony:knowledge-session-coordinator.semantic-maintenance"
	backendFeatureNamedVersions        = "ssfv:symphony:knowledge-session-coordinator.named-version-durability"
	backendFeatureMaestroPresence      = "ssfv:symphony:maestro-presence-authority"
	backendFeatureMaestroInventory     = "ssfv:symphony:maestro-presence-authority.complete-inventory"
	backendFeaturePlatform             = "ssfv:symphony:platform"
	backendFeatureSSIAG                = "ssfv:symphony:ssiag-foundation"
	backendFeatureSSIAGEnrollment      = "ssfv:symphony:ssiag-foundation.tops-enrollment"
	backendFeatureSSIAGSupervisor      = "ssfv:symphony:ssiag-foundation.native-supervision"
	backendFeatureSSIAGPolicy          = "ssfv:symphony:ssiag-foundation.policy-administration"
	backendFeatureSSIAGProviders       = "ssfv:symphony:ssiag-foundation.provider-metadata-registry"
	backendFeatureSSIAGProviderTrust   = "ssfv:symphony:ssiag-foundation.provider-trust-assurance"
	backendFeatureSSIAGReadiness       = "ssfv:symphony:ssiag-foundation.provider-readiness-assurance"
	backendFeatureSSIAGProviderBinding = "ssfv:symphony:ssiag-foundation.provider-binding-lifecycle"
	backendFeatureSSIAGMacOSMetadata   = "ssfv:symphony:ssiag.macos-keychain-metadata"
	backendFeatureSSIAGMacOSReadiness  = "ssfv:symphony:ssiag.macos-signed-bundle-readiness"
	backendFeatureSTAV                 = "ssfv:symphony:stav-append-authority"
	backendFeatureSTAVEnrollment       = "ssfv:symphony:stav-append-authority.tops-enrollment"
	backendFeatureSTAVSupervisor       = "ssfv:symphony:stav-append-authority.native-supervision"
	backendFeatureSTAVQuery            = "ssfv:symphony:stav-append-authority.authorized-query"
	backendFeatureSTAVDurability       = "ssfv:symphony:stav-append-authority.ledger-durability"
	backendFeatureAccordareProducer    = "ssfv:symphony:accordare-stav-producer"
	backendFeatureAccordareGrant       = "ssfv:symphony:accordare-stav-producer.grant-administration"
	backendFeatureAccordareIntent      = "ssfv:symphony:accordare-stav-producer.intent-durability"
	backendFeatureAccordareSupervisor  = "ssfv:symphony:accordare-stav-producer.native-supervision"
	backendFeatureValidator            = "ssfv:symphony:symphony-validator"
	backendFeatureRootSummary          = "ssfv:symphony:symphony-validator.root-summary-assurance"
	backendFeatureInvariantAssurance   = "ssfv:symphony:symphony-validator.invariant-ownership-assurance"
)

// reviewedBackendFeatureBindings is the ratified semantic bridge from stable
// qxcmd identities to the backend feature interactions they administer. The
// command's qxctl-owned wrapper binding remains first-class and distinct. This
// table adds no grammar, dispatch, operation identity, availability, or
// authority; the SSFV engine independently evaluates it against the canonical
// administration profile.
var reviewedBackendFeatureBindings = map[string][]commandregistry.FeatureBinding{
	"inventory":                 {{FeatureID: backendFeaturePlatform, Interaction: "discover"}},
	"knowledge.invariant.check": {{FeatureID: backendFeatureInvariantAssurance, Interaction: "validate"}},
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
	"sav.named-version.propose":          {{FeatureID: backendFeatureNamedVersions, Interaction: "propose"}, {FeatureID: backendFeatureAccordareProducer, Interaction: "invoke"}, {FeatureID: backendFeatureAccordareIntent, Interaction: "invoke"}},
	"sav.named-version.seal":             {{FeatureID: backendFeatureNamedVersions, Interaction: "lifecycle"}, {FeatureID: backendFeatureAccordareProducer, Interaction: "invoke"}, {FeatureID: backendFeatureAccordareIntent, Interaction: "invoke"}},
	"sav.named-version.alias":            {{FeatureID: backendFeatureNamedVersions, Interaction: "configure"}, {FeatureID: backendFeatureAccordareProducer, Interaction: "invoke"}, {FeatureID: backendFeatureAccordareIntent, Interaction: "invoke"}},
	"sav.named-version.lookup":           {{FeatureID: backendFeatureNamedVersions, Interaction: "query"}},
	"sav.named-version.status":           {{FeatureID: backendFeatureNamedVersions, Interaction: "query"}},
	"sav.named-version.recover":          {{FeatureID: backendFeatureNamedVersions, Interaction: "recover"}, {FeatureID: backendFeatureAccordareProducer, Interaction: "invoke"}, {FeatureID: backendFeatureAccordareIntent, Interaction: "invoke"}},
	"maestro.inspect":                    {{FeatureID: backendFeatureMaestroPresence, Interaction: "inspect"}},
	"maestro.inventory":                  {{FeatureID: backendFeatureMaestroInventory, Interaction: "query"}},
	"maestro.recover":                    {{FeatureID: backendFeatureMaestroPresence, Interaction: "recover"}},
	"maestro.status":                     {{FeatureID: backendFeatureMaestroPresence, Interaction: "query"}},
	"module.check":                       {{FeatureID: backendFeatureCoordinator, Interaction: "validate"}},
	"module.inspect":                     {{FeatureID: backendFeatureCoordinator, Interaction: "inspect"}},
	"modules":                            {{FeatureID: backendFeaturePlatform, Interaction: "discover"}},
	"ssiag.doctor":                       {{FeatureID: backendFeatureSSIAG, Interaction: "validate"}},
	"ssiag.enrollment.status":            {{FeatureID: backendFeatureSSIAGEnrollment, Interaction: "lifecycle"}},
	"ssiag.enrollment.plan":              {{FeatureID: backendFeatureSSIAGEnrollment, Interaction: "lifecycle"}},
	"ssiag.enrollment.apply":             {{FeatureID: backendFeatureSSIAGEnrollment, Interaction: "lifecycle"}},
	"ssiag.enrollment.apply-status":      {{FeatureID: backendFeatureSSIAGEnrollment, Interaction: "lifecycle"}},
	"ssiag.enrollment.recover":           {{FeatureID: backendFeatureSSIAGEnrollment, Interaction: "lifecycle"}},
	"ssiag.policy.apply":                 {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "apply"}},
	"ssiag.policy.propose":               {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "propose"}},
	"ssiag.policy.recover":               {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "recover"}},
	"ssiag.policy.status":                {{FeatureID: backendFeatureSSIAGPolicy, Interaction: "query"}},
	"ssiag.provider.show":                {{FeatureID: backendFeatureSSIAGProviderTrust, Interaction: "inspect"}},
	"ssiag.provider.verify": {
		{FeatureID: backendFeatureSSIAGProviderTrust, Interaction: "validate"},
		{FeatureID: backendFeatureSSIAGMacOSMetadata, Interaction: "validate"},
	},
	"ssiag.provider.readiness": {
		{FeatureID: backendFeatureSSIAGReadiness, Interaction: "validate"},
		{FeatureID: backendFeatureSSIAGMacOSReadiness, Interaction: "validate"},
	},
	"ssiag.provider.installations":        {{FeatureID: backendFeatureSSIAGProviderBinding, Interaction: "discover"}},
	"ssiag.provider.binding.status":       {{FeatureID: backendFeatureSSIAGProviderBinding, Interaction: "inspect"}},
	"ssiag.provider.binding.plan":         {{FeatureID: backendFeatureSSIAGProviderBinding, Interaction: "propose"}},
	"ssiag.provider.binding.apply":        {{FeatureID: backendFeatureSSIAGProviderBinding, Interaction: "apply"}},
	"ssiag.provider.binding.apply-status": {{FeatureID: backendFeatureSSIAGProviderBinding, Interaction: "query"}},
	"ssiag.provider.binding.recover":      {{FeatureID: backendFeatureSSIAGProviderBinding, Interaction: "recover"}},
	"ssiag.providers":                     {{FeatureID: backendFeatureSSIAGProviders, Interaction: "discover"}},
	"ssiag.status":                        {{FeatureID: backendFeatureSSIAG, Interaction: "query"}},
	"ssiag.supervisor.status":             {{FeatureID: backendFeatureSSIAGSupervisor, Interaction: "lifecycle"}},
	"ssiag.supervisor.plan":               {{FeatureID: backendFeatureSSIAGSupervisor, Interaction: "lifecycle"}},
	"ssiag.supervisor.apply":              {{FeatureID: backendFeatureSSIAGSupervisor, Interaction: "lifecycle"}},
	"ssiag.supervisor.apply-status":       {{FeatureID: backendFeatureSSIAGSupervisor, Interaction: "lifecycle"}},
	"ssiag.supervisor.recover":            {{FeatureID: backendFeatureSSIAGSupervisor, Interaction: "lifecycle"}},
	"stav.doctor":                         {{FeatureID: backendFeatureSTAV, Interaction: "validate"}},
	"stav.accordare.status":               {{FeatureID: backendFeatureAccordareProducer, Interaction: "query"}, {FeatureID: backendFeatureAccordareIntent, Interaction: "query"}},
	"stav.accordare.reconcile":            {{FeatureID: backendFeatureAccordareProducer, Interaction: "recover"}, {FeatureID: backendFeatureAccordareIntent, Interaction: "recover"}},
	"stav.accordare.supervisor-install":   {{FeatureID: backendFeatureAccordareProducer, Interaction: "apply"}, {FeatureID: backendFeatureAccordareSupervisor, Interaction: "apply"}},
	"stav.accordare.supervisor-uninstall": {{FeatureID: backendFeatureAccordareProducer, Interaction: "apply"}, {FeatureID: backendFeatureAccordareSupervisor, Interaction: "apply"}},
	"stav.accordare-grant.install":        {{FeatureID: backendFeatureAccordareGrant, Interaction: "apply"}},
	"stav.accordare-grant.remove":         {{FeatureID: backendFeatureAccordareGrant, Interaction: "apply"}},
	"stav.enrollment.status":              {{FeatureID: backendFeatureSTAVEnrollment, Interaction: "lifecycle"}},
	"stav.enrollment.plan":                {{FeatureID: backendFeatureSTAVEnrollment, Interaction: "lifecycle"}},
	"stav.enrollment.apply":               {{FeatureID: backendFeatureSTAVEnrollment, Interaction: "lifecycle"}},
	"stav.enrollment.apply-status":        {{FeatureID: backendFeatureSTAVEnrollment, Interaction: "lifecycle"}},
	"stav.enrollment.recover":             {{FeatureID: backendFeatureSTAVEnrollment, Interaction: "lifecycle"}},
	"stav.query":                          {{FeatureID: backendFeatureSTAVQuery, Interaction: "query"}},
	"stav.status":                         {{FeatureID: backendFeatureSTAV, Interaction: "query"}},
	"stav.supervisor.status":              {{FeatureID: backendFeatureSTAVSupervisor, Interaction: "lifecycle"}},
	"stav.supervisor.plan":                {{FeatureID: backendFeatureSTAVSupervisor, Interaction: "lifecycle"}},
	"stav.supervisor.apply":               {{FeatureID: backendFeatureSTAVSupervisor, Interaction: "lifecycle"}},
	"stav.supervisor.apply-status":        {{FeatureID: backendFeatureSTAVSupervisor, Interaction: "lifecycle"}},
	"stav.supervisor.recover":             {{FeatureID: backendFeatureSTAVSupervisor, Interaction: "lifecycle"}},
	"stav.verify":                         {{FeatureID: backendFeatureSTAVDurability, Interaction: "validate"}},
	"validate.scan":                       {{FeatureID: backendFeatureValidator, Interaction: "validate"}},
	"validate.root-summary":               {{FeatureID: backendFeatureRootSummary, Interaction: "inspect"}},
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

func registeredValidationWarning(
	command *cobra.Command, key, interaction string, exactState, mutation bool, recoveryID string,
) *cobra.Command {
	spec := commandSpec(key, featureValidation, interaction)
	if exactState {
		spec.OutputProtocols = []string{"symphony.validation.warning-state.v1"}
		spec.ResultValidationProtocols = []string{"symphony.validation.warning-state.v1"}
	}
	if mutation {
		spec.Mutability = "permission_backed_mutation"
		spec.AuthorityMode = "target_host_permission"
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:" + recoveryID)
	}
	return commandregistry.Attach(command, spec)
}

func registeredRootSummary(command *cobra.Command) *cobra.Command {
	spec := commandSpec("validate.root-summary", featureValidation, "inspect")
	spec.OutputProtocols = []string{"symphony.repository.root-summary.v1"}
	spec.ResultValidationProtocols = []string{"symphony.repository.root-summary.v1"}
	return commandregistry.Attach(command, spec)
}

func registeredInvariantQuery(command *cobra.Command, key, interaction string) *cobra.Command {
	spec := commandSpec(key, featureInvariantAssurance, interaction)
	spec.OutputProtocols = []string{"symphony.knowledge.invariant-query-result.v1"}
	spec.ResultValidationProtocols = []string{"symphony.knowledge.invariant-query-result.v1"}
	return commandregistry.Attach(command, spec)
}

func registeredInvariantCheck(command *cobra.Command) *cobra.Command {
	spec := commandSpec("knowledge.invariant.check", featureInvariantAssurance, "validate")
	spec.OutputProtocols = []string{"symphony.validation.result.v1"}
	spec.ResultValidationProtocols = []string{"symphony.validation.result.v1"}
	return commandregistry.Attach(command, spec)
}

func registeredAccordare(
	command *cobra.Command,
	key, featureID, interaction, engineOperationID string,
	inputProtocol, outputProtocol string,
) *cobra.Command {
	spec := commandSpec(key, featureID, interaction)
	spec.BackendOperationIDs = []string{engineOperationID}
	if inputProtocol != "" {
		spec.InputProtocols = []string{inputProtocol}
	}
	if outputProtocol != "" {
		spec.OutputProtocols = []string{outputProtocol}
		spec.ResultValidationProtocols = []string{outputProtocol}
	}
	if interaction == "propose" {
		spec.Mutability = "proposal_only"
	}
	return commandregistry.Attach(command, spec)
}

func registeredNamedVersion(command *cobra.Command, leaf, interaction string) *cobra.Command {
	key := "sav.named-version." + leaf
	spec := commandSpec(key, featureSAVNamedVersion, interaction)
	spec.InputProtocols = []string{"symphony.knowledge.named-version-command.v1"}
	if leaf == "propose" {
		spec.InputProtocols = append([]string{"symphony.sav.named-version-validation-input.v1"}, spec.InputProtocols...)
	}
	spec.OutputProtocols = []string{"symphony.knowledge.named-version-result.v1"}
	spec.ResultValidationProtocols = []string{"symphony.knowledge.named-version-result.v1"}
	coordinatorLeaf := leaf
	if leaf == "propose" {
		coordinatorLeaf = "prepare"
	}
	coordinatorOperation := "engop:symphony:knowledge-session-coordinator.named-version." + coordinatorLeaf
	spec.BackendOperationIDs = []string{coordinatorOperation}
	if leaf == "propose" {
		spec.BackendOperationIDs = append(spec.BackendOperationIDs,
			"engop:symphony:sav.named-version.validate")
	}
	if leaf == "seal" || leaf == "alias" || leaf == "lookup" {
		spec.BackendOperationIDs = append(spec.BackendOperationIDs,
			"engop:symphony:sav.named-version.validate")
	}
	spec.AuthorityMode = "target_host_permission"
	if leaf == "propose" || leaf == "seal" || leaf == "alias" || leaf == "recover" {
		spec.Mutability = "permission_backed_mutation"
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:sav.named-version.recover")
	}
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

func registeredFoundationLifecycle(
	command *cobra.Command,
	component, surface, leaf, wrapperFeature string,
) *cobra.Command {
	key := component + "." + surface + "." + leaf
	interaction := map[string]string{
		"status": "query", "plan": "propose", "apply": "apply",
		"apply-status": "query", "recover": "recover",
	}[leaf]
	spec := commandSpec(key, wrapperFeature, interaction)
	switch leaf {
	case "plan":
		spec.Mutability = "proposal_only"
	case "apply", "recover":
		spec.Mutability = "permission_backed_mutation"
		spec.AuthorityMode = "target_host_permission"
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:" + component + "." + surface + ".recover")
	}
	backendOperation := leaf
	if leaf == "status" {
		backendOperation = "observe"
	}
	spec.BackendOperationIDs = []string{
		"engop:symphony:" + component + "." + surface + "." + backendOperation,
	}
	spec.InputProtocols = []string{"symphony.foundation.lifecycle-command.v1"}
	spec.OutputProtocols = []string{"symphony.foundation.lifecycle-result.v1"}
	spec.ResultValidationProtocols = []string{"symphony.foundation.lifecycle-result.v1"}
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

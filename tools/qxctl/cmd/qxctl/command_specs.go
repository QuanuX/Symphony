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
)

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
	return commandregistry.CommandSpec{
		CommandID:                 "qxcmd:symphony:" + key,
		Status:                    "experimental",
		IntroducedIn:              "0.1.0-dev",
		ReplacementIDs:            []string{},
		FeatureBindings:           []commandregistry.FeatureBinding{{FeatureID: featureID, Interaction: interaction}},
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

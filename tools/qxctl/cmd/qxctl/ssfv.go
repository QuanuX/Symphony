package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
)

var invokeSSFV = knowledgeengine.InvokeSSFV

func runSSFV(operation string, options ssfvOptions) error {
	if operation == "administration-check" && !options.jsonOutput {
		return fmt.Errorf("--json is required for SSFV administration coverage evidence")
	}
	if options.prefix == "" {
		return fmt.Errorf("--prefix is required")
	}
	start := options.repository
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current working directory: %w", err)
		}
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	info, err := os.Lstat(start)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("--repo must identify a no-follow directory")
	}
	invocationRoot := start
	if operation != "administration-check" {
		invocationRoot, err = repository.FindRoot(start)
		if err != nil {
			return fmt.Errorf("could not find Symphony repository root: %w", err)
		}
	}

	var payload []byte
	switch operation {
	case "inspect":
		payload = []byte(`{}`)
	case "check":
		payload, err = ssfvCheckPayload(options)
	case "diff", "propose", "administration-check":
		payload, err = knowledgeengine.ReadPayload(options.input)
	case "graph":
		payload = []byte(`{"format":"json"}`)
	default:
		return fmt.Errorf("unsupported SSFV operation")
	}
	if err != nil {
		return err
	}
	response, err := invokeSSFV(
		context.Background(), options.prefix, options.version, invocationRoot, operation, payload)
	if err != nil {
		return err
	}
	checkValid, err := validateSSFVResult(operation, response.Result)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var output bytes.Buffer
		if err := json.Indent(&output, response.Result, "", "  "); err != nil {
			return fmt.Errorf("format SSFV result: %w", err)
		}
		fmt.Println(output.String())
		if !checkValid {
			return fmt.Errorf("SSFV check reported structural or required-freshness violations")
		}
		return nil
	}
	return printSSFVResult(operation, response.Result)
}

func ssfvCheckPayload(options ssfvOptions) ([]byte, error) {
	modes := map[string]bool{"disabled": true, "report": true, "require": true}
	if !modes[options.freshness] {
		return nil, fmt.Errorf("--freshness must be disabled, report, or require")
	}
	var baseline any
	if options.baseline != "" {
		data, err := knowledgeengine.ReadPayload(options.baseline)
		if err != nil {
			return nil, fmt.Errorf("read SSFV baseline: %w", err)
		}
		baseline = json.RawMessage(data)
	}
	if options.freshness == "disabled" && baseline != nil {
		return nil, fmt.Errorf("--baseline is prohibited when --freshness=disabled")
	}
	if options.freshness != "disabled" && baseline == nil {
		return nil, fmt.Errorf("--baseline is required when --freshness=%s", options.freshness)
	}
	expectedNamespace := any(nil)
	if options.expectedNamespaceDigest != "" {
		expectedNamespace = options.expectedNamespaceDigest
	}
	expectedRegistry := any(nil)
	if options.expectedRegistryDigest != "" {
		expectedRegistry = options.expectedRegistryDigest
	}
	return json.Marshal(map[string]any{
		"expected_namespace_digest": expectedNamespace,
		"expected_registry_digest":  expectedRegistry,
		"freshness":                 options.freshness,
		"baseline":                  baseline,
	})
}

func validateSSFVResult(operation string, result json.RawMessage) (bool, error) {
	switch operation {
	case "inspect":
		var value struct {
			Readiness                  string `json:"readiness"`
			EmptyRegistryValid         *bool  `json:"empty_registry_valid"`
			EngineDecidesWorthiness    *bool  `json:"engine_decides_feature_worthiness"`
			EngineDecidesSemanticTruth *bool  `json:"engine_decides_semantic_truth"`
			CanonicalApplyEnabled      *bool  `json:"canonical_apply_enabled"`
			Descriptor                 struct {
				ModuleID               string  `json:"module_id"`
				EngineID               string  `json:"engine_id"`
				VectorID               string  `json:"vector_id"`
				Language               string  `json:"language"`
				ThermalPath            string  `json:"thermal_path"`
				InstallState           string  `json:"install_state"`
				DefaultReceptor        *string `json:"default_receptor"`
				CanonicalApplyEnabled  *bool   `json:"canonical_apply_enabled"`
				SessionMutationEnabled *bool   `json:"session_mutation_enabled"`
				NetworkListener        *bool   `json:"network_listener"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Readiness != "read_check_diff_propose_graph" ||
			value.Descriptor.ModuleID != "ssfv-engine" ||
			value.Descriptor.EngineID != "symphony-ssfv" ||
			value.Descriptor.VectorID != "ssfv" ||
			value.Descriptor.Language != "C++26" ||
			value.Descriptor.ThermalPath != "freezing" ||
			value.Descriptor.InstallState != "installed_undocked" ||
			value.Descriptor.DefaultReceptor != nil ||
			!explicitTrue(value.EmptyRegistryValid) ||
			!explicitFalse(value.EngineDecidesWorthiness) ||
			!explicitFalse(value.EngineDecidesSemanticTruth) ||
			!explicitFalse(value.CanonicalApplyEnabled) ||
			!explicitFalse(value.Descriptor.CanonicalApplyEnabled) ||
			!explicitFalse(value.Descriptor.SessionMutationEnabled) ||
			!explicitFalse(value.Descriptor.NetworkListener) {
			return false, fmt.Errorf("SSFV inspect result violates the implemented safety contract")
		}
		return true, nil
	case "check":
		var value struct {
			Protocol               string `json:"protocol"`
			StructuralState        string `json:"structural_state"`
			FreshnessMode          string `json:"freshness_mode"`
			SemanticFreshnessState string `json:"semantic_freshness_state"`
			ReadOnly               *bool  `json:"read_only"`
			CanonicalApplyEnabled  *bool  `json:"canonical_apply_enabled"`
			Summary                struct {
				Violation uint64 `json:"violation"`
				State     string `json:"state"`
			} `json:"summary"`
		}
		modes := map[string]bool{"disabled": true, "report": true, "require": true}
		freshnessStates := map[string]bool{"not_evaluated": true, "current": true, "stale": true}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Protocol != "symphony.ssfv.check-result.v2" ||
			(value.StructuralState != "valid" && value.StructuralState != "invalid") ||
			!modes[value.FreshnessMode] || !freshnessStates[value.SemanticFreshnessState] ||
			!explicitTrue(value.ReadOnly) || !explicitFalse(value.CanonicalApplyEnabled) ||
			value.Summary.State == "" {
			return false, fmt.Errorf("SSFV check result violates the implemented safety contract")
		}
		valid := value.StructuralState == "valid" &&
			value.Summary.State == "valid" && value.Summary.Violation == 0
		if value.FreshnessMode == "require" && value.SemanticFreshnessState != "current" {
			valid = false
		}
		return valid, nil
	case "diff":
		var value struct {
			Protocol           string            `json:"protocol"`
			State              string            `json:"state"`
			Added              []string          `json:"added_feature_ids"`
			Changed            []string          `json:"changed_feature_ids"`
			Removed            []string          `json:"removed_feature_ids"`
			Stale              []string          `json:"stale_references"`
			SemanticCandidates []json.RawMessage `json:"semantic_candidates"`
			ReadOnly           *bool             `json:"read_only"`
			Noncanonical       *bool             `json:"noncanonical"`
			ResultDigest       string            `json:"result_digest"`
		}
		states := map[string]bool{
			"identical": true, "additive": true, "semantic_change": true,
			"removal": true, "review_required": true,
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Protocol != "symphony.ssfv.diff-result.v2" ||
			!states[value.State] || value.Added == nil || value.Changed == nil ||
			value.Removed == nil || value.Stale == nil || value.SemanticCandidates == nil ||
			!explicitTrue(value.ReadOnly) || !explicitTrue(value.Noncanonical) ||
			!validTaggedDigest(value.ResultDigest) {
			return false, fmt.Errorf("SSFV diff result violates the implemented safety contract")
		}
		if err := validateSSFVEmbeddedDigest(result, "result_digest", value.ResultDigest); err != nil {
			return false, err
		}
		return true, nil
	case "propose":
		var value struct {
			Protocol              string `json:"protocol"`
			ModuleID              string `json:"module_id"`
			EngineID              string `json:"engine_id"`
			VectorID              string `json:"vector_id"`
			ProposalID            string `json:"proposal_id"`
			ProposalDigest        string `json:"proposal_digest"`
			CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
			Authority             struct {
				CallerDeclaredOperation  *bool `json:"caller_declared_operation"`
				EngineDecidedDomainTruth *bool `json:"engine_decided_domain_truth"`
				Ratified                 *bool `json:"ratified"`
			} `json:"authority"`
			WriteSet []struct {
				TargetPath string `json:"target_path"`
			} `json:"write_set"`
			Operations []struct {
				TargetPath string `json:"target_path"`
			} `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Protocol != "symphony.knowledge.proposal.v1" ||
			value.ModuleID != "ssfv-engine" || value.EngineID != "symphony-ssfv" ||
			value.VectorID != "ssfv" || value.ProposalID == "" ||
			!validTaggedDigest(value.ProposalDigest) ||
			len(value.WriteSet) == 0 || len(value.WriteSet) > 4 ||
			len(value.Operations) != len(value.WriteSet) ||
			!explicitTrue(value.Authority.CallerDeclaredOperation) ||
			!explicitFalse(value.Authority.EngineDecidedDomainTruth) ||
			!explicitFalse(value.Authority.Ratified) ||
			!explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SSFV proposal result violates the implemented safety contract")
		}
		if err := validateSSFVEmbeddedDigest(result, "proposal_digest", value.ProposalDigest); err != nil {
			return false, err
		}
		writes := make(map[string]struct{}, len(value.WriteSet))
		for _, write := range value.WriteSet {
			if !safeSSFVWriteTarget(write.TargetPath) {
				return false, fmt.Errorf("SSFV proposal contains an unsafe write target")
			}
			if _, duplicate := writes[write.TargetPath]; duplicate {
				return false, fmt.Errorf("SSFV proposal contains a duplicate write target")
			}
			writes[write.TargetPath] = struct{}{}
		}
		for _, operation := range value.Operations {
			if _, ok := writes[operation.TargetPath]; !ok {
				return false, fmt.Errorf("SSFV proposal operation is outside its write set")
			}
		}
		return true, nil
	case "graph":
		var value struct {
			Protocol         string            `json:"protocol"`
			ModuleID         string            `json:"module_id"`
			EngineID         string            `json:"engine_id"`
			VectorID         string            `json:"vector_id"`
			NodeCount        *uint64           `json:"node_count"`
			EdgeCount        *uint64           `json:"edge_count"`
			Nodes            []json.RawMessage `json:"nodes"`
			Edges            []json.RawMessage `json:"edges"`
			ProjectionDigest string            `json:"projection_digest"`
			Noncanonical     *bool             `json:"noncanonical"`
			Rebuildable      *bool             `json:"rebuildable"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Protocol != "symphony.ssfv.graph-projection.v1" ||
			value.ModuleID != "ssfv-engine" || value.EngineID != "symphony-ssfv" ||
			value.VectorID != "ssfv" || value.NodeCount == nil || value.EdgeCount == nil ||
			value.Nodes == nil || value.Edges == nil ||
			*value.NodeCount != uint64(len(value.Nodes)) ||
			*value.EdgeCount != uint64(len(value.Edges)) ||
			!validTaggedDigest(value.ProjectionDigest) ||
			!explicitTrue(value.Noncanonical) || !explicitTrue(value.Rebuildable) {
			return false, fmt.Errorf("SSFV graph result violates the implemented safety contract")
		}
		if err := validateSSFVEmbeddedDigest(result, "projection_digest", value.ProjectionDigest); err != nil {
			return false, err
		}
		return true, nil
	case "administration-check":
		return validateSSFVAdministrationResult(result)
	default:
		return false, fmt.Errorf("unsupported SSFV operation")
	}
}

type administrationFinding struct {
	FindingID            string   `json:"finding_id"`
	Severity             string   `json:"severity"`
	FeatureID            *string  `json:"feature_id"`
	Interaction          *string  `json:"interaction"`
	EngineOperationID    *string  `json:"engine_operation_id"`
	CommandID            *string  `json:"command_id"`
	Reason               string   `json:"reason"`
	Missing              []string `json:"missing"`
	ProposalOnly         *bool    `json:"proposal_only"`
	RatificationRequired *bool    `json:"ratification_required"`
}

type administrationSurface struct {
	FeatureID          string                  `json:"feature_id"`
	Interaction        string                  `json:"interaction"`
	DesignState        string                  `json:"design_state"`
	LiveState          string                  `json:"live_state"`
	AuthorizationState string                  `json:"authorization_state"`
	CommandIDs         []string                `json:"command_ids"`
	EngineOperationIDs []string                `json:"engine_operation_ids"`
	Findings           []administrationFinding `json:"findings"`
}

type administrationModuleIntegration struct {
	ModuleID         string                  `json:"module_id"`
	EngineID         *string                 `json:"engine_id"`
	DescriptorDigest *string                 `json:"descriptor_digest"`
	IntegrationState string                  `json:"integration_state"`
	DockingReady     *bool                   `json:"docking_ready"`
	Findings         []administrationFinding `json:"findings"`
}

type administrationRemediation struct {
	RemediationID         string   `json:"remediation_id"`
	FindingIDs            []string `json:"finding_ids"`
	FeatureID             string   `json:"feature_id"`
	Interaction           string   `json:"interaction"`
	BackendOperationIDs   []string `json:"backend_operation_ids"`
	RequiredMutability    string   `json:"required_mutability"`
	RequiredAuthorityMode string   `json:"required_authority_mode"`
	RequiredTargetScope   string   `json:"required_target_scope"`
	RequiredEvidence      []string `json:"required_evidence"`
	ProposedCommandID     *string  `json:"proposed_command_id"`
	ProposedGrammar       *string  `json:"proposed_grammar"`
	ProposalOnly          *bool    `json:"proposal_only"`
	RatificationRequired  *bool    `json:"ratification_required"`
}

type administrationSummary struct {
	FeaturesChecked *uint64 `json:"features_checked"`
	SurfacesChecked *uint64 `json:"surfaces_checked"`
	Satisfied       *uint64 `json:"satisfied"`
	Uncovered       *uint64 `json:"uncovered"`
	Exempt          *uint64 `json:"exempt"`
	Prohibited      *uint64 `json:"prohibited"`
	Stale           *uint64 `json:"stale"`
	Unresolved      *uint64 `json:"unresolved"`
}

type administrationResult struct {
	Protocol                      string                            `json:"protocol"`
	FormatVersion                 uint64                            `json:"format_version"`
	SemanticSnapshotDigest        string                            `json:"semantic_snapshot_digest"`
	ProfileDigest                 string                            `json:"profile_digest"`
	ExpectedCommandRegistryDigest string                            `json:"expected_command_registry_digest"`
	ObservedCommandRegistryDigest *string                           `json:"observed_command_registry_digest"`
	EngineDescriptorDigests       []string                          `json:"engine_descriptor_digests"`
	FeatureFindings               []administrationFinding           `json:"feature_findings"`
	Surfaces                      []administrationSurface           `json:"surfaces"`
	ModuleIntegrations            []administrationModuleIntegration `json:"module_integrations"`
	RemediationConstraints        []administrationRemediation       `json:"remediation_constraints"`
	Summary                       administrationSummary             `json:"summary"`
	ReadOnly                      *bool                             `json:"read_only"`
	Canonical                     *bool                             `json:"canonical"`
	ResultDigest                  string                            `json:"result_digest"`
}

func validateSSFVAdministrationResult(result json.RawMessage) (bool, error) {
	var value administrationResult
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return false, fmt.Errorf("SSFV administration-check result violates the implemented safety contract: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return false, fmt.Errorf("SSFV administration-check result contains trailing JSON")
	}
	if value.Protocol != "symphony.knowledge.administration-coverage-result.v1" ||
		value.FormatVersion != 1 || !validTaggedDigest(value.SemanticSnapshotDigest) ||
		!validTaggedDigest(value.ProfileDigest) ||
		!validTaggedDigest(value.ExpectedCommandRegistryDigest) ||
		(value.ObservedCommandRegistryDigest != nil && !validTaggedDigest(*value.ObservedCommandRegistryDigest)) ||
		value.EngineDescriptorDigests == nil || len(value.EngineDescriptorDigests) > 1024 ||
		value.FeatureFindings == nil || len(value.FeatureFindings) > 8192 ||
		value.Surfaces == nil || len(value.Surfaces) > 8192 ||
		value.ModuleIntegrations == nil || len(value.ModuleIntegrations) > 1024 ||
		value.RemediationConstraints == nil || len(value.RemediationConstraints) > 8192 ||
		!explicitTrue(value.ReadOnly) || !explicitFalse(value.Canonical) ||
		!validTaggedDigest(value.ResultDigest) || !validAdministrationSummary(value.Summary) {
		return false, fmt.Errorf("SSFV administration-check result violates the implemented safety contract")
	}
	if !uniqueValidValues(value.EngineDescriptorDigests, validTaggedDigest) {
		return false, fmt.Errorf("SSFV administration-check descriptor digests are invalid or duplicated")
	}
	for _, finding := range value.FeatureFindings {
		if !validAdministrationFinding(finding) {
			return false, fmt.Errorf("SSFV administration-check feature finding is malformed")
		}
	}
	for _, surface := range value.Surfaces {
		if !validAdministrationSurface(surface) {
			return false, fmt.Errorf("SSFV administration-check surface is malformed")
		}
	}
	for _, integration := range value.ModuleIntegrations {
		if !validAdministrationModule(integration) {
			return false, fmt.Errorf("SSFV administration-check module integration is malformed")
		}
	}
	for _, remediation := range value.RemediationConstraints {
		if !validAdministrationRemediation(remediation) {
			return false, fmt.Errorf("SSFV administration-check remediation constraint is malformed")
		}
	}
	if err := validateSSFVEmbeddedDigest(result, "result_digest", value.ResultDigest); err != nil {
		return false, err
	}
	return true, nil
}

func validAdministrationFinding(value administrationFinding) bool {
	if value.FindingID == "" || len(value.FindingID) > 256 || value.Reason == "" ||
		len(value.Reason) > 4096 || value.Missing == nil || len(value.Missing) > 64 ||
		!ssfvOneOf(value.Severity, "information", "warning", "violation") ||
		!explicitTrue(value.ProposalOnly) || !explicitTrue(value.RatificationRequired) {
		return false
	}
	if value.FeatureID != nil && !strings.HasPrefix(*value.FeatureID, "ssfv:") {
		return false
	}
	if value.Interaction != nil && !validAdministrationInteraction(*value.Interaction) {
		return false
	}
	if value.EngineOperationID != nil && !strings.HasPrefix(*value.EngineOperationID, "engop:") {
		return false
	}
	if value.CommandID != nil && !strings.HasPrefix(*value.CommandID, "qxcmd:") {
		return false
	}
	allowedMissing := func(item string) bool {
		return ssfvOneOf(item, "feature_registration", "administration_expectation", "qxctl_command",
			"engine_operation", "input_protocol", "output_protocol", "result_validator",
			"authorization_binding", "expected_state", "recovery_route", "noninteractive_support",
			"json_output", "implementation_evidence", "test_evidence", "skvi_route")
	}
	return uniqueValidValues(value.Missing, allowedMissing)
}

func validAdministrationSurface(value administrationSurface) bool {
	if !strings.HasPrefix(value.FeatureID, "ssfv:") || !validAdministrationInteraction(value.Interaction) ||
		!ssfvOneOf(value.DesignState, "satisfied", "uncovered", "exempt", "prohibited", "stale", "unresolved") ||
		!ssfvOneOf(value.LiveState, "not_evaluated", "ready", "qxctl_absent", "not_installed", "not_bound", "incompatible", "unreachable", "disabled", "unknown") ||
		!ssfvOneOf(value.AuthorizationState, "not_evaluated", "allowed", "denied", "expired", "indeterminate") ||
		value.CommandIDs == nil || len(value.CommandIDs) > 256 ||
		value.EngineOperationIDs == nil || len(value.EngineOperationIDs) > 256 ||
		value.Findings == nil || len(value.Findings) > 256 ||
		!uniqueValidValues(value.CommandIDs, func(item string) bool { return strings.HasPrefix(item, "qxcmd:") }) ||
		!uniqueValidValues(value.EngineOperationIDs, func(item string) bool { return strings.HasPrefix(item, "engop:") }) {
		return false
	}
	for _, finding := range value.Findings {
		if !validAdministrationFinding(finding) {
			return false
		}
	}
	return true
}

func validAdministrationModule(value administrationModuleIntegration) bool {
	if value.ModuleID == "" || len(value.ModuleID) > 256 || value.DockingReady == nil ||
		value.Findings == nil || len(value.Findings) > 256 ||
		!ssfvOneOf(value.IntegrationState, "unassessed", "descriptor_invalid", "semantic_registration_required",
			"administration_unintegrated", "integration_ready", "blocked_incompatible", "retired") {
		return false
	}
	if value.IntegrationState == "integration_ready" {
		if !*value.DockingReady || value.DescriptorDigest == nil || !validTaggedDigest(*value.DescriptorDigest) {
			return false
		}
	} else if *value.DockingReady {
		return false
	}
	if value.DescriptorDigest != nil && !validTaggedDigest(*value.DescriptorDigest) {
		return false
	}
	for _, finding := range value.Findings {
		if !validAdministrationFinding(finding) {
			return false
		}
	}
	return true
}

func validAdministrationRemediation(value administrationRemediation) bool {
	if value.RemediationID == "" || len(value.RemediationID) > 256 ||
		len(value.FindingIDs) == 0 || len(value.FindingIDs) > 256 ||
		!uniqueValidValues(value.FindingIDs, func(item string) bool { return item != "" && len(item) <= 256 }) ||
		!strings.HasPrefix(value.FeatureID, "ssfv:") || !validAdministrationInteraction(value.Interaction) ||
		value.BackendOperationIDs == nil || len(value.BackendOperationIDs) > 256 ||
		!uniqueValidValues(value.BackendOperationIDs, func(item string) bool { return strings.HasPrefix(item, "engop:") }) ||
		!ssfvOneOf(value.RequiredMutability, "read_only", "evidence_only", "proposal_only", "permission_backed_mutation", "prohibited") ||
		!ssfvOneOf(value.RequiredAuthorityMode, "none", "target_host_permission", "ssiag") ||
		!ssfvOneOf(value.RequiredTargetScope, "local", "target_host") ||
		value.RequiredEvidence == nil || len(value.RequiredEvidence) > 64 ||
		value.ProposedCommandID != nil || value.ProposedGrammar != nil ||
		!explicitTrue(value.ProposalOnly) || !explicitTrue(value.RatificationRequired) {
		return false
	}
	allowedEvidence := func(item string) bool {
		return ssfvOneOf(item, "command_spec", "cobra_leaf", "feature_binding", "backend_binding",
			"input_protocol", "output_protocol", "result_validator", "expected_state",
			"authorization_binding", "recovery_route", "noninteractive_support", "json_output",
			"implementation_test", "help_compatibility", "documentation", "skvi_route")
	}
	return uniqueValidValues(value.RequiredEvidence, allowedEvidence)
}

func validAdministrationSummary(value administrationSummary) bool {
	return value.FeaturesChecked != nil && value.SurfacesChecked != nil && value.Satisfied != nil &&
		value.Uncovered != nil && value.Exempt != nil && value.Prohibited != nil &&
		value.Stale != nil && value.Unresolved != nil
}

func validAdministrationInteraction(value string) bool {
	return ssfvOneOf(value, "discover", "inspect", "query", "validate", "configure", "propose", "invoke", "apply", "lifecycle", "recover")
}

func ssfvOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func uniqueValidValues(values []string, valid func(string) bool) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validateSSFVEmbeddedDigest(result json.RawMessage, field, supplied string) error {
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return fmt.Errorf("decode SSFV %s input: %w", field, err)
	}
	delete(object, field)
	canonical, err := marshalSSFVCanonical(object)
	if err != nil {
		return fmt.Errorf("canonicalize SSFV %s input: %w", field, err)
	}
	digest := sha256.Sum256(canonical)
	expected := "sha256:" + hex.EncodeToString(digest[:])
	if supplied != expected {
		return fmt.Errorf("SSFV %s mismatch", field)
	}
	return nil
}

func marshalSSFVCanonical(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return restoreSSFVJSONLineSeparators(
		bytes.TrimSuffix(output.Bytes(), []byte("\n"))), nil
}

func restoreSSFVJSONLineSeparators(encoded []byte) []byte {
	result := make([]byte, 0, len(encoded))
	for index := 0; index < len(encoded); {
		if encoded[index] != '\\' {
			result = append(result, encoded[index])
			index++
			continue
		}
		runEnd := index
		for runEnd < len(encoded) && encoded[runEnd] == '\\' {
			runEnd++
		}
		runLength := runEnd - index
		if runLength%2 == 1 && runEnd+5 <= len(encoded) &&
			encoded[runEnd] == 'u' {
			escape := string(encoded[runEnd : runEnd+5])
			if escape == "u2028" || escape == "u2029" {
				result = append(result, encoded[index:runEnd-1]...)
				if escape == "u2028" {
					result = append(result, "\u2028"...)
				} else {
					result = append(result, "\u2029"...)
				}
				index = runEnd + 5
				continue
			}
		}
		result = append(result, encoded[index:runEnd]...)
		index = runEnd
	}
	return result
}

func safeSSFVWriteTarget(path string) bool {
	if path == "knowledge/ssfv/NAMESPACES.md" ||
		path == "knowledge/ssfv/REGISTRY.md" ||
		path == "knowledge/skvi/INDEX.md" {
		return true
	}
	if path == "FEATURES.md" {
		return true
	}
	if !strings.HasSuffix(path, "/FEATURES.md") || strings.HasPrefix(path, "/") ||
		strings.Contains(path, "\\") || strings.Contains(path, "//") {
		return false
	}
	for _, component := range strings.Split(path, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
		for _, character := range component {
			if character < 0x20 || character == 0x7f ||
				strings.ContainsRune("*?[]{}", character) {
				return false
			}
		}
	}
	return true
}

func printSSFVResult(operation string, result json.RawMessage) error {
	switch operation {
	case "inspect":
		var value struct {
			Readiness  string `json:"readiness"`
			Descriptor struct {
				EngineID      string `json:"engine_id"`
				EngineVersion string `json:"engine_version"`
				ThermalPath   string `json:"thermal_path"`
				InstallState  string `json:"install_state"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Descriptor.EngineID == "" || value.Readiness == "" {
			return fmt.Errorf("SSFV inspect result is incomplete")
		}
		fmt.Printf("SSFV: engine=%s version=%s readiness=%s thermal=%s state=%s apply=false\n",
			value.Descriptor.EngineID, value.Descriptor.EngineVersion, value.Readiness,
			value.Descriptor.ThermalPath, value.Descriptor.InstallState)
		return nil
	case "check":
		var value struct {
			CoverageState          string `json:"coverage_state"`
			FeatureCount           uint64 `json:"feature_count"`
			FeatureFileCount       uint64 `json:"feature_file_count"`
			StructuralState        string `json:"structural_state"`
			FreshnessMode          string `json:"freshness_mode"`
			SemanticFreshnessState string `json:"semantic_freshness_state"`
			FeatureRegistry        struct {
				Digest string `json:"digest"`
			} `json:"feature_registry"`
			Summary struct {
				Pass      uint64 `json:"pass"`
				Warning   uint64 `json:"warning"`
				Violation uint64 `json:"violation"`
				State     string `json:"state"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Summary.State == "" || value.FeatureRegistry.Digest == "" {
			return fmt.Errorf("SSFV check result is incomplete")
		}
		fmt.Printf("SSFV check: state=%s structural=%s freshness=%s/%s coverage=%s features=%d files=%d pass=%d warning=%d violation=%d registry_digest=%s\n",
			value.Summary.State, value.StructuralState, value.FreshnessMode,
			value.SemanticFreshnessState, value.CoverageState, value.FeatureCount,
			value.FeatureFileCount, value.Summary.Pass, value.Summary.Warning,
			value.Summary.Violation, value.FeatureRegistry.Digest)
		valid, err := validateSSFVResult("check", result)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("SSFV check reported structural or required-freshness violations")
		}
		return nil
	case "diff":
		var value struct {
			State        string `json:"state"`
			ResultDigest string `json:"result_digest"`
			Summary      struct {
				Added          uint64 `json:"added"`
				Changed        uint64 `json:"changed"`
				Removed        uint64 `json:"removed"`
				Stale          uint64 `json:"stale"`
				ReviewRequired uint64 `json:"review_required"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.State == "" || value.ResultDigest == "" {
			return fmt.Errorf("SSFV diff result is incomplete")
		}
		fmt.Printf("SSFV diff: state=%s added=%d changed=%d removed=%d stale=%d review_required=%d digest=%s noncanonical=true\n",
			value.State, value.Summary.Added, value.Summary.Changed, value.Summary.Removed,
			value.Summary.Stale, value.Summary.ReviewRequired, value.ResultDigest)
		return nil
	case "propose":
		var value struct {
			ProposalID     string `json:"proposal_id"`
			ProposalDigest string `json:"proposal_digest"`
			WriteSet       []struct {
				TargetPath string `json:"target_path"`
			} `json:"write_set"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.ProposalID == "" || len(value.WriteSet) == 0 {
			return fmt.Errorf("SSFV proposal result is incomplete")
		}
		fmt.Printf("SSFV proposal: id=%s digest=%s writes=%d ratified=false apply=false\n",
			value.ProposalID, value.ProposalDigest, len(value.WriteSet))
		return nil
	case "graph":
		var value struct {
			NodeCount        uint64 `json:"node_count"`
			EdgeCount        uint64 `json:"edge_count"`
			ProjectionDigest string `json:"projection_digest"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProjectionDigest == "" {
			return fmt.Errorf("SSFV graph result is incomplete")
		}
		fmt.Printf("SSFV graph: nodes=%d edges=%d digest=%s noncanonical=true rebuildable=true\n",
			value.NodeCount, value.EdgeCount, value.ProjectionDigest)
		return nil
	default:
		return fmt.Errorf("unsupported SSFV result")
	}
}

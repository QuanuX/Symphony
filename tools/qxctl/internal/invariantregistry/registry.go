package invariantregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const (
	Protocol       = "symphony.knowledge.invariant-ownership-registry.v1"
	QueryProtocol  = "symphony.knowledge.invariant-query-result.v1"
	RegistryPath   = "knowledge/INVARIANT-OWNERSHIP.json"
	maxRegistry    = 512 * 1024
	maxJSONValues  = 131072
	maxInvariants  = 4096
	maxAdapters    = 64
	maxReferences  = 128
	maxOperations  = 64
	maxCases       = 128
	maxTextBytes   = 4096
	maxTitleBytes  = 256
	maxPathBytes   = 4096
	consumerCheck  = "identity_shape_digest_passed"
	semanticStatus = "not_asserted"
)

var (
	invariantPattern = regexp.MustCompile(`^invariant:symphony:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	adapterPattern   = regexp.MustCompile(`^adapter:symphony:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	operationPattern = regexp.MustCompile(`^engop:[a-z][a-z0-9-]{0,62}:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	tokenPattern     = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,256}$`)
	digestPattern    = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	pathPattern      = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
)

type TestPolicy struct {
	OwnerProducerRegressionRequired   bool `json:"owner_producer_regression_required"`
	ConsumerBoundaryRejectionRequired bool `json:"consumer_boundary_rejection_required"`
	RealProcessRequiredForIPC         bool `json:"real_process_required_for_ipc"`
}

type TestReference struct {
	Path  string   `json:"path"`
	Cases []string `json:"cases"`
}

type Adapter struct {
	AdapterID          string   `json:"adapter_id"`
	Component          string   `json:"component"`
	EntryPointID       string   `json:"entry_point_id"`
	CommandProtocol    string   `json:"command_protocol"`
	FormatVersion      uint64   `json:"format_version"`
	OwnerContract      string   `json:"owner_contract"`
	ImplementationPath string   `json:"implementation_path"`
	VersionPolicy      string   `json:"version_policy"`
	OperationIDs       []string `json:"operation_ids"`
}

type Invariant struct {
	InvariantID                string          `json:"invariant_id"`
	Title                      string          `json:"title"`
	OwnerContract              string          `json:"owner_contract"`
	OwnerComponent             string          `json:"owner_component"`
	Statement                  string          `json:"statement"`
	ProducerImplementations    []string        `json:"producer_implementations"`
	ProducerRegressions        []TestReference `json:"producer_regressions"`
	ConsumerBoundaryRejections []TestReference `json:"consumer_boundary_rejections"`
	AllowedAdapterIDs          []string        `json:"allowed_adapter_ids"`
	IPCBoundary                bool            `json:"ipc_boundary"`
	RealProcessRegressions     []TestReference `json:"real_process_regressions"`
	Status                     string          `json:"status"`
}

type Registry struct {
	Protocol        string      `json:"protocol"`
	FormatVersion   uint64      `json:"format_version"`
	Scope           string      `json:"scope"`
	CatalogScope    string      `json:"catalog_scope"`
	CatalogComplete bool        `json:"catalog_complete"`
	ForwardGate     string      `json:"forward_gate"`
	TestPolicy      TestPolicy  `json:"test_policy"`
	Adapters        []Adapter   `json:"adapters"`
	Invariants      []Invariant `json:"invariants"`
	RegistryDigest  string      `json:"registry_digest"`
}

type InvariantSummary struct {
	InvariantID    string `json:"invariant_id"`
	Title          string `json:"title"`
	OwnerComponent string `json:"owner_component"`
	OwnerContract  string `json:"owner_contract"`
	IPCBoundary    bool   `json:"ipc_boundary"`
	Status         string `json:"status"`
}

type ProjectionEnvelope struct {
	Protocol               string `json:"protocol"`
	FormatVersion          uint64 `json:"format_version"`
	Operation              string `json:"operation"`
	SourcePath             string `json:"source_path"`
	RegistryDigest         string `json:"registry_digest"`
	ConsumerCheck          string `json:"consumer_check"`
	SemanticValidity       string `json:"semantic_validity"`
	CompleteCheckCommandID string `json:"complete_check_command_id"`
	ResultDigest           string `json:"result_digest,omitempty"`
}

type StatusProjection struct {
	ProjectionEnvelope
	Scope           string `json:"scope"`
	CatalogScope    string `json:"catalog_scope"`
	CatalogComplete bool   `json:"catalog_complete"`
	ForwardGate     string `json:"forward_gate"`
	InvariantCount  uint64 `json:"invariant_count"`
	AdapterCount    uint64 `json:"adapter_count"`
}

type ListProjection struct {
	ProjectionEnvelope
	Invariants []InvariantSummary `json:"invariants"`
}

type ShowProjection struct {
	ProjectionEnvelope
	Invariant Invariant `json:"invariant"`
}

func Load(repositoryRoot string) (Registry, error) {
	data, err := knowledgeengine.ReadRepositoryJSONObject(repositoryRoot, RegistryPath, maxRegistry, maxJSONValues)
	if err != nil {
		return Registry{}, err
	}
	var generic map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&generic); err != nil {
		return Registry{}, fmt.Errorf("decode invariant ownership registry: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Registry{}, fmt.Errorf("decode invariant ownership registry: trailing data")
	}
	var registry Registry
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&registry); err != nil {
		return Registry{}, fmt.Errorf("decode invariant ownership registry shape: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Registry{}, fmt.Errorf("decode invariant ownership registry shape: trailing data")
	}
	if err := validate(registry); err != nil {
		return Registry{}, err
	}
	digestValue, ok := generic["registry_digest"].(string)
	if !ok || digestValue != registry.RegistryDigest {
		return Registry{}, fmt.Errorf("invariant ownership registry digest field is invalid")
	}
	delete(generic, "registry_digest")
	canonical, err := canonicalJSON(generic)
	if err != nil {
		return Registry{}, fmt.Errorf("canonicalize invariant ownership registry: %w", err)
	}
	digest := sha256.Sum256(canonical)
	expected := "sha256:" + hex.EncodeToString(digest[:])
	if registry.RegistryDigest != expected {
		return Registry{}, fmt.Errorf("invariant ownership registry digest mismatch: expected %s, observed %s", expected, registry.RegistryDigest)
	}
	return registry, nil
}

func Status(registry Registry) (StatusProjection, error) {
	projection := StatusProjection{
		ProjectionEnvelope: envelope("status", registry.RegistryDigest), Scope: registry.Scope,
		CatalogScope: registry.CatalogScope, CatalogComplete: registry.CatalogComplete,
		ForwardGate: registry.ForwardGate, InvariantCount: uint64(len(registry.Invariants)),
		AdapterCount: uint64(len(registry.Adapters)),
	}
	if err := setResultDigest(&projection, &projection.ResultDigest); err != nil {
		return StatusProjection{}, err
	}
	return projection, nil
}

func List(registry Registry) (ListProjection, error) {
	items := make([]InvariantSummary, 0, len(registry.Invariants))
	for _, invariant := range registry.Invariants {
		items = append(items, InvariantSummary{
			InvariantID: invariant.InvariantID, Title: invariant.Title,
			OwnerComponent: invariant.OwnerComponent, OwnerContract: invariant.OwnerContract,
			IPCBoundary: invariant.IPCBoundary, Status: invariant.Status,
		})
	}
	projection := ListProjection{ProjectionEnvelope: envelope("list", registry.RegistryDigest), Invariants: items}
	if err := setResultDigest(&projection, &projection.ResultDigest); err != nil {
		return ListProjection{}, err
	}
	return projection, nil
}

func Show(registry Registry, invariantID string) (ShowProjection, error) {
	if !invariantPattern.MatchString(invariantID) {
		return ShowProjection{}, fmt.Errorf("--invariant-id must be an exact stable invariant:symphony: identity")
	}
	index := sort.Search(len(registry.Invariants), func(index int) bool {
		return registry.Invariants[index].InvariantID >= invariantID
	})
	if index == len(registry.Invariants) || registry.Invariants[index].InvariantID != invariantID {
		return ShowProjection{}, fmt.Errorf("invariant %q is not registered", invariantID)
	}
	projection := ShowProjection{
		ProjectionEnvelope: envelope("show", registry.RegistryDigest),
		Invariant:          registry.Invariants[index],
	}
	if err := setResultDigest(&projection, &projection.ResultDigest); err != nil {
		return ShowProjection{}, err
	}
	return projection, nil
}

func envelope(operation, digest string) ProjectionEnvelope {
	return ProjectionEnvelope{
		Protocol: QueryProtocol, FormatVersion: 1, Operation: operation,
		SourcePath: RegistryPath, RegistryDigest: digest, ConsumerCheck: consumerCheck,
		SemanticValidity:       semanticStatus,
		CompleteCheckCommandID: "qxcmd:symphony:knowledge.invariant.check",
	}
}

func setResultDigest(value any, target *string) error {
	encoded, err := canonicalJSON(value)
	if err != nil {
		return fmt.Errorf("encode invariant query result: %w", err)
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return fmt.Errorf("decode invariant query result: %w", err)
	}
	delete(object, "result_digest")
	canonical, err := canonicalJSON(object)
	if err != nil {
		return fmt.Errorf("canonicalize invariant query result: %w", err)
	}
	digest := sha256.Sum256(canonical)
	*target = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func validate(registry Registry) error {
	if registry.Protocol != Protocol || registry.FormatVersion != 1 ||
		registry.Scope != "common_lowest_authoritative_layer" ||
		registry.CatalogScope != "registered_incremental" || registry.CatalogComplete ||
		registry.ForwardGate != "enforce_new_or_modified" || !digestPattern.MatchString(registry.RegistryDigest) {
		return fmt.Errorf("invariant ownership registry identity or posture is invalid")
	}
	if !registry.TestPolicy.OwnerProducerRegressionRequired ||
		!registry.TestPolicy.ConsumerBoundaryRejectionRequired ||
		!registry.TestPolicy.RealProcessRequiredForIPC {
		return fmt.Errorf("invariant ownership registry test policy is invalid")
	}
	if len(registry.Adapters) > maxAdapters || len(registry.Invariants) < 1 || len(registry.Invariants) > maxInvariants {
		return fmt.Errorf("invariant ownership registry collection bounds are invalid")
	}
	adapterIDs := make(map[string]struct{}, len(registry.Adapters))
	prior := ""
	for _, adapter := range registry.Adapters {
		if !adapterPattern.MatchString(adapter.AdapterID) || prior != "" && prior >= adapter.AdapterID {
			return fmt.Errorf("invariant ownership adapter identity ordering is invalid")
		}
		prior = adapter.AdapterID
		adapterIDs[adapter.AdapterID] = struct{}{}
		if err := validateAdapter(adapter); err != nil {
			return fmt.Errorf("adapter %s: %w", adapter.AdapterID, err)
		}
	}
	prior = ""
	for _, invariant := range registry.Invariants {
		if !invariantPattern.MatchString(invariant.InvariantID) || prior != "" && prior >= invariant.InvariantID {
			return fmt.Errorf("invariant ownership identity ordering is invalid")
		}
		prior = invariant.InvariantID
		if err := validateInvariant(invariant, adapterIDs); err != nil {
			return fmt.Errorf("invariant %s: %w", invariant.InvariantID, err)
		}
	}
	return nil
}

func validateAdapter(adapter Adapter) error {
	if adapter.FormatVersion != 1 ||
		(adapter.CommandProtocol != "symphony.foundation.lifecycle-command.v1" && adapter.CommandProtocol != "symphony.ssiag.provider.control.v1") ||
		adapter.VersionPolicy != "exact_receipt_v2_entry_point_and_capability_compatible" ||
		len(adapter.OperationIDs) < 1 || len(adapter.OperationIDs) > maxOperations ||
		!safePath(adapter.OwnerContract) || !safePath(adapter.ImplementationPath) {
		return fmt.Errorf("shape is invalid")
	}
	knownPair := adapter.Component == "ssiag" && adapter.EntryPointID == "ssiag.foundation-lifecycle" && adapter.CommandProtocol == "symphony.foundation.lifecycle-command.v1" ||
		adapter.Component == "ssiag" && adapter.EntryPointID == "ssiag.macos-keychain-provider" && adapter.CommandProtocol == "symphony.ssiag.provider.control.v1" ||
		adapter.Component == "stav" && adapter.EntryPointID == "stav.foundation-lifecycle" && adapter.CommandProtocol == "symphony.foundation.lifecycle-command.v1"
	if !knownPair || adapter.AdapterID != "adapter:symphony:"+adapter.EntryPointID+".v1" {
		return fmt.Errorf("component or entry point is invalid")
	}
	if err := validateSortedUnique(adapter.OperationIDs, operationPattern, "operation IDs"); err != nil {
		return err
	}
	if adapter.AdapterID == "adapter:symphony:ssiag.macos-keychain-provider.v1" {
		expected := []string{
			"engop:symphony:ssiag.macos-keychain-provider.readiness.observe",
			"engop:symphony:ssiag.provider.metadata-capabilities",
			"engop:symphony:ssiag.provider.metadata-handshake",
			"engop:symphony:ssiag.provider.metadata-status",
		}
		if len(adapter.OperationIDs) != len(expected) {
			return fmt.Errorf("operation IDs do not match the adapter-owned operation set")
		}
		for index := range expected {
			if adapter.OperationIDs[index] != expected[index] {
				return fmt.Errorf("operation IDs do not match the adapter-owned operation set")
			}
		}
	}
	return nil
}

func validateInvariant(invariant Invariant, adapterIDs map[string]struct{}) error {
	if !boundedText(invariant.Title, maxTitleBytes) || !boundedText(invariant.Statement, maxTextBytes) ||
		!safePath(invariant.OwnerContract) || !tokenPattern.MatchString(invariant.OwnerComponent) ||
		invariant.Status != "active" || len(invariant.ProducerImplementations) < 1 ||
		len(invariant.ProducerImplementations) > maxReferences || len(invariant.ProducerRegressions) < 1 ||
		len(invariant.ProducerRegressions) > maxReferences || len(invariant.ConsumerBoundaryRejections) < 1 ||
		len(invariant.ConsumerBoundaryRejections) > maxReferences || len(invariant.AllowedAdapterIDs) > maxAdapters ||
		len(invariant.RealProcessRegressions) > maxReferences {
		return fmt.Errorf("shape or required evidence is invalid")
	}
	if err := validateSortedPaths(invariant.ProducerImplementations); err != nil {
		return err
	}
	if err := validateReferences(invariant.ProducerRegressions, "producer regressions"); err != nil {
		return err
	}
	if err := validateReferences(invariant.ConsumerBoundaryRejections, "consumer boundary rejections"); err != nil {
		return err
	}
	if err := validateReferences(invariant.RealProcessRegressions, "real-process regressions"); err != nil {
		return err
	}
	if invariant.IPCBoundary && (len(invariant.AllowedAdapterIDs) == 0 || len(invariant.RealProcessRegressions) == 0) ||
		!invariant.IPCBoundary && len(invariant.RealProcessRegressions) != 0 {
		return fmt.Errorf("IPC evidence posture is invalid")
	}
	if err := validateSortedUnique(invariant.AllowedAdapterIDs, adapterPattern, "allowed adapter IDs"); err != nil {
		return err
	}
	for _, adapterID := range invariant.AllowedAdapterIDs {
		if _, exists := adapterIDs[adapterID]; !exists {
			return fmt.Errorf("allowed adapter %s is not registered", adapterID)
		}
	}
	return nil
}

func validateReferences(references []TestReference, label string) error {
	prior := ""
	for _, reference := range references {
		if !safePath(reference.Path) || len(reference.Cases) < 1 || len(reference.Cases) > maxCases ||
			prior != "" && prior >= reference.Path {
			return fmt.Errorf("%s are invalid or not ordered", label)
		}
		prior = reference.Path
		if err := validateSortedUnique(reference.Cases, tokenPattern, label+" cases"); err != nil {
			return err
		}
	}
	return nil
}

func validateSortedPaths(paths []string) error {
	prior := ""
	for _, path := range paths {
		if !safePath(path) || prior != "" && prior >= path {
			return fmt.Errorf("producer implementation paths are invalid or not ordered")
		}
		prior = path
	}
	return nil
}

func validateSortedUnique(values []string, pattern *regexp.Regexp, label string) error {
	prior := ""
	for _, value := range values {
		if !pattern.MatchString(value) || prior != "" && prior >= value {
			return fmt.Errorf("%s are invalid or not ordered", label)
		}
		prior = value
	}
	return nil
}

func safePath(value string) bool {
	if value == "" || len(value) > maxPathBytes || !pathPattern.MatchString(value) || strings.HasPrefix(value, "/") ||
		strings.Contains(value, "//") || strings.Contains(value, "\\") {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

func boundedText(value string, limit int) bool {
	if value == "" || len(value) > limit || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func canonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

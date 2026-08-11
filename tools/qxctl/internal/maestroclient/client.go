package maestroclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	qxversion "github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

const (
	CommandProtocol = "symphony.maestro.knowledge-engine-docking.v1"
	ReceptorKind    = "symphony.maestro.knowledge-engine.v1"
)

var requiredCapabilities = []string{
	"atomic-head-v1", "dual-slot-presence-v1", "exact-receipt-binding-v1",
	"expected-state-cas-v1", "idempotent-operation-v1", "recovery-forward-v1",
	"ssiag-capability-binding-v1",
}

type ClientEvidence struct {
	ClientID              string   `json:"client_id"`
	ClientVersion         string   `json:"client_version"`
	ProcessProtocols      []string `json:"process_protocols"`
	PresenceReadVersions  []uint64 `json:"presence_read_versions"`
	PresenceWriteVersions []uint64 `json:"presence_write_versions"`
	Capabilities          []string `json:"capabilities"`
}

type ComponentEvidence struct {
	ComponentID      string `json:"component_id"`
	ComponentKind    string `json:"component_kind"`
	ModuleID         string `json:"module_id"`
	VectorID         string `json:"vector_id"`
	EngineID         string `json:"engine_id"`
	ReceiptDigest    string `json:"receipt_digest"`
	ExecutableDigest string `json:"executable_digest"`
	ReceptorKind     string `json:"receptor_kind"`
	EvidenceDigest   string `json:"evidence_digest"`
}

type Presence struct {
	Protocol                string  `json:"protocol"`
	FormatVersion           uint64  `json:"format_version"`
	TOPSID                  string  `json:"tops_id"`
	ReceptorID              string  `json:"receptor_id"`
	ReceptorKind            string  `json:"receptor_kind"`
	ComponentID             string  `json:"component_id"`
	ComponentKind           string  `json:"component_kind"`
	ModuleID                string  `json:"module_id"`
	VectorID                string  `json:"vector_id"`
	EngineID                string  `json:"engine_id"`
	Disposition             string  `json:"disposition"`
	ReceiptDigest           string  `json:"receipt_digest"`
	ExecutableDigest        string  `json:"executable_digest"`
	OperationID             string  `json:"operation_id"`
	PreviousPresenceDigest  *string `json:"previous_presence_digest"`
	CapabilityBindingDigest string  `json:"capability_binding_digest"`
	CommittedAt             string  `json:"committed_at"`
	Canonical               bool    `json:"canonical"`
	PresenceDigest          string  `json:"presence_digest"`
}

type Registry struct {
	Protocol               string     `json:"protocol"`
	FormatVersion          uint64     `json:"format_version"`
	TOPSID                 string     `json:"tops_id"`
	ReceptorID             string     `json:"receptor_id"`
	ReceptorKind           string     `json:"receptor_kind"`
	Generation             uint64     `json:"generation"`
	PreviousRegistryDigest *string    `json:"previous_registry_digest"`
	Components             []Presence `json:"components"`
	Extensions             []any      `json:"extensions"`
	Recovery               Recovery   `json:"recovery"`
	UpdatedAt              string     `json:"updated_at"`
	Canonical              bool       `json:"canonical"`
	RegistryDigest         string     `json:"registry_digest"`
}

type Recovery struct {
	State               string  `json:"state"`
	Disposition         string  `json:"disposition"`
	RecoveredFromDigest *string `json:"recovered_from_digest"`
	Detail              string  `json:"detail"`
}

type DescriptorLimits struct {
	Components    uint64 `json:"components"`
	RequestBytes  uint64 `json:"request_bytes"`
	ResponseBytes uint64 `json:"response_bytes"`
}

type Descriptor struct {
	Protocol                string           `json:"protocol"`
	FormatVersion           uint64           `json:"format_version"`
	MaestroID               string           `json:"maestro_id"`
	MaestroVersion          string           `json:"maestro_version"`
	TOPSID                  string           `json:"tops_id"`
	ReceptorID              string           `json:"receptor_id"`
	ReceptorKind            string           `json:"receptor_kind"`
	ProcessProtocols        []string         `json:"process_protocols"`
	DockingProtocols        []string         `json:"docking_protocols"`
	PresenceReadVersions    []uint64         `json:"presence_read_versions"`
	PresenceWriteVersion    uint64           `json:"presence_write_version"`
	SupportedComponentKinds []string         `json:"supported_component_kinds"`
	RequiredCapabilities    []string         `json:"required_capabilities"`
	OptionalCapabilities    []string         `json:"optional_capabilities"`
	Limits                  DescriptorLimits `json:"limits"`
	ThermalPath             string           `json:"thermal_path"`
	ExecutionEnabled        bool             `json:"execution_enabled"`
	NetworkListener         bool             `json:"network_listener"`
	Canonical               bool             `json:"canonical"`
	DescriptorDigest        string           `json:"descriptor_digest"`
}

type Compatibility struct {
	Mode                          string   `json:"mode"`
	ProcessProtocol               *string  `json:"process_protocol"`
	PresenceReadVersion           *uint64  `json:"presence_read_version"`
	PresenceWriteVersion          *uint64  `json:"presence_write_version"`
	MissingCapabilities           []string `json:"missing_capabilities"`
	TwoWayProceduralCompatibility bool     `json:"two_way_procedural_compatibility"`
	Reason                        string   `json:"reason"`
}

type Result struct {
	Protocol         string          `json:"protocol"`
	FormatVersion    uint64          `json:"format_version"`
	Operation        string          `json:"operation"`
	TOPSID           string          `json:"tops_id"`
	ReceptorID       string          `json:"receptor_id"`
	Compatibility    Compatibility   `json:"compatibility"`
	Descriptor       Descriptor      `json:"descriptor"`
	RegistryPresent  bool            `json:"registry_present"`
	Registry         json.RawMessage `json:"registry"`
	RegistryDigest   *string         `json:"registry_digest"`
	PresencePresent  bool            `json:"presence_present"`
	Presence         *Presence       `json:"presence"`
	Outcome          string          `json:"outcome"`
	Changed          bool            `json:"changed"`
	Recovered        bool            `json:"recovered"`
	RepairActions    []string        `json:"repair_actions"`
	ReadOnly         bool            `json:"read_only"`
	ExecutionEnabled bool            `json:"execution_enabled"`
	Canonical        bool            `json:"canonical"`
}

type command struct {
	Protocol               string         `json:"protocol"`
	FormatVersion          uint64         `json:"format_version"`
	Operation              string         `json:"operation"`
	StateRoot              any            `json:"state_root"`
	TOPSID                 string         `json:"tops_id"`
	ReceptorID             string         `json:"receptor_id"`
	OperationID            any            `json:"operation_id"`
	ExpectedRegistryDigest any            `json:"expected_registry_digest"`
	Component              any            `json:"component"`
	AuthorizationDecision  any            `json:"authorization_decision"`
	Client                 ClientEvidence `json:"client"`
}

func Client() ClientEvidence {
	return ClientEvidence{
		ClientID: "qxctl", ClientVersion: strings.ReplaceAll(qxversion.Version, " ", "-"),
		ProcessProtocols:     []string{"symphony.knowledge.engine-process.v1"},
		PresenceReadVersions: []uint64{1}, PresenceWriteVersions: []uint64{1},
		Capabilities: append([]string(nil), requiredCapabilities...),
	}
}

func Resource(topsID, receptorID, operation, componentID, receiptDigest, expected string) string {
	digest := sha256.Sum256([]byte(strings.Join(
		[]string{topsID, receptorID, operation, componentID, receiptDigest, expected}, "\n")))
	return "symphony.maestro.docking:" + hex.EncodeToString(digest[:])
}

func NewComponentEvidence(componentID, moduleID, vectorID, engineID, receiptDigest, executableDigest string) (ComponentEvidence, error) {
	base := map[string]any{
		"component_id": componentID, "component_kind": "vector_engine", "module_id": moduleID,
		"vector_id": vectorID, "engine_id": engineID, "receipt_digest": receiptDigest,
		"executable_digest": executableDigest, "receptor_kind": ReceptorKind,
	}
	encoded, err := json.Marshal(base)
	if err != nil {
		return ComponentEvidence{}, err
	}
	digest := sha256.Sum256(encoded)
	return ComponentEvidence{
		ComponentID: componentID, ComponentKind: "vector_engine", ModuleID: moduleID,
		VectorID: vectorID, EngineID: engineID, ReceiptDigest: receiptDigest,
		ExecutableDigest: executableDigest, ReceptorKind: ReceptorKind,
		EvidenceDigest: "sha256:" + hex.EncodeToString(digest[:]),
	}, nil
}

func (result Result) DecodedRegistry() (*Registry, error) {
	if !result.RegistryPresent {
		if string(result.Registry) != "null" || result.RegistryDigest != nil {
			return nil, fmt.Errorf("absent Maestro registry carries contradictory evidence")
		}
		return nil, nil
	}
	var registry Registry
	if err := decodeExact(result.Registry, &registry); err != nil {
		return nil, fmt.Errorf("decode Maestro presence registry: %w", err)
	}
	if registry.Protocol != "symphony.maestro.docking-presence-registry.v1" ||
		registry.FormatVersion != 1 || registry.TOPSID != result.TOPSID ||
		registry.ReceptorID != result.ReceptorID || registry.ReceptorKind != ReceptorKind ||
		result.RegistryDigest == nil || registry.RegistryDigest != *result.RegistryDigest ||
		registry.Canonical || registry.Components == nil || registry.Extensions == nil {
		return nil, fmt.Errorf("Maestro presence registry identity is invalid")
	}
	return &registry, nil
}

func Inspect(ctx context.Context, prefix, version, repositoryRoot, topsID, receptorID string) (Result, error) {
	return invoke(ctx, prefix, version, repositoryRoot, command{
		Protocol: CommandProtocol, FormatVersion: 1, Operation: "inspect", StateRoot: nil,
		TOPSID: topsID, ReceptorID: receptorID, OperationID: nil,
		ExpectedRegistryDigest: nil, Component: nil, AuthorizationDecision: nil, Client: Client(),
	})
}

func Status(ctx context.Context, prefix, version, repositoryRoot, stateRoot, topsID, receptorID, componentID string,
	decision ssiagclient.AuthorizationDecision) (Result, error) {
	var selected any
	if componentID != "" {
		selected = map[string]string{"component_id": componentID}
	}
	return invoke(ctx, prefix, version, repositoryRoot, command{
		Protocol: CommandProtocol, FormatVersion: 1, Operation: "status", StateRoot: stateRoot,
		TOPSID: topsID, ReceptorID: receptorID, OperationID: nil,
		ExpectedRegistryDigest: nil, Component: selected, AuthorizationDecision: decision, Client: Client(),
	})
}

func Mutate(ctx context.Context, prefix, version, repositoryRoot, stateRoot, topsID, receptorID,
	operation, operationID, expected string, component ComponentEvidence,
	decision ssiagclient.AuthorizationDecision) (Result, error) {
	if operation != "dock" && operation != "undock" {
		return Result{}, fmt.Errorf("unsupported Maestro mutation %q", operation)
	}
	return invoke(ctx, prefix, version, repositoryRoot, command{
		Protocol: CommandProtocol, FormatVersion: 1, Operation: operation, StateRoot: stateRoot,
		TOPSID: topsID, ReceptorID: receptorID, OperationID: operationID,
		ExpectedRegistryDigest: expected, Component: component, AuthorizationDecision: decision, Client: Client(),
	})
}

func Recover(ctx context.Context, prefix, version, repositoryRoot, stateRoot, topsID, receptorID,
	operationID, expected string, decision ssiagclient.AuthorizationDecision) (Result, error) {
	return invoke(ctx, prefix, version, repositoryRoot, command{
		Protocol: CommandProtocol, FormatVersion: 1, Operation: "recover", StateRoot: stateRoot,
		TOPSID: topsID, ReceptorID: receptorID, OperationID: operationID,
		ExpectedRegistryDigest: expected, Component: nil, AuthorizationDecision: decision, Client: Client(),
	})
}

func invoke(ctx context.Context, prefix, version, repositoryRoot string, request command) (Result, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return Result{}, fmt.Errorf("encode Maestro command: %w", err)
	}
	response, err := knowledgeengine.InvokeMaestro(
		ctx, prefix, version, repositoryRoot, request.Operation, payload)
	if err != nil {
		return Result{}, err
	}
	if err := knowledgeengine.ValidateJSONObject(response.Result, 4*1024*1024); err != nil {
		return Result{}, fmt.Errorf("invalid Maestro result: %w", err)
	}
	var result Result
	if err := decodeExact(response.Result, &result); err != nil {
		return Result{}, fmt.Errorf("decode Maestro result: %w", err)
	}
	if result.Protocol != "symphony.maestro.docking-result.v1" || result.FormatVersion != 1 ||
		result.Operation != request.Operation || result.TOPSID != request.TOPSID ||
		result.ReceptorID != request.ReceptorID || result.Canonical || result.ExecutionEnabled ||
		result.Descriptor.Protocol != "symphony.maestro.receptor-descriptor.v1" ||
		result.Descriptor.FormatVersion != 1 || result.Descriptor.MaestroID != "symphony-maestro" ||
		result.Descriptor.MaestroVersion != version || result.Descriptor.TOPSID != request.TOPSID ||
		result.Descriptor.ReceptorID != request.ReceptorID || result.Descriptor.ReceptorKind != ReceptorKind ||
		result.Descriptor.PresenceWriteVersion != 1 || result.Descriptor.ThermalPath != "freezing" ||
		result.Descriptor.ExecutionEnabled || result.Descriptor.NetworkListener || result.Descriptor.Canonical ||
		result.Descriptor.Limits.Components != 4096 || result.Descriptor.Limits.RequestBytes != 1048576 ||
		result.Descriptor.Limits.ResponseBytes != 4194304 ||
		!containsString(result.Descriptor.ProcessProtocols, "symphony.knowledge.engine-process.v1") ||
		!containsString(result.Descriptor.DockingProtocols, CommandProtocol) ||
		!containsVersion(result.Descriptor.PresenceReadVersions, 1) ||
		len(result.Descriptor.SupportedComponentKinds) != 1 ||
		result.Descriptor.SupportedComponentKinds[0] != "vector_engine" ||
		result.Descriptor.RequiredCapabilities == nil || result.Descriptor.OptionalCapabilities == nil ||
		result.Compatibility.MissingCapabilities == nil || result.Compatibility.Reason == "" ||
		!result.Compatibility.TwoWayProceduralCompatibility {
		return Result{}, fmt.Errorf("Maestro result identity or safety boundary is invalid")
	}
	if result.PresencePresent && (result.Presence == nil || result.Presence.Disposition != "docked") ||
		!result.PresencePresent && result.Presence != nil && result.Presence.Disposition == "docked" {
		return Result{}, fmt.Errorf("Maestro result presence evidence is contradictory")
	}
	if _, err := result.DecodedRegistry(); err != nil {
		return Result{}, err
	}
	return result, nil
}

func decodeExact(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsVersion(values []uint64, wanted uint64) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

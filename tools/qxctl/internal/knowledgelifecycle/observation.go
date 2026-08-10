package knowledgelifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const maxReceiptBytes = 256 * 1024

type Identity struct {
	ComponentID      string `json:"component_id"`
	Version          string `json:"version"`
	ExecutableDigest string `json:"executable_digest"`
}

type ProviderAvailability struct {
	ProviderID string `json:"provider_id"`
	Available  bool   `json:"available"`
}

type Platform struct {
	OS                   string                 `json:"os"`
	KernelABI            string                 `json:"kernel_abi"`
	Architecture         string                 `json:"architecture"`
	QxctlIdentity        Identity               `json:"qxctl_identity"`
	CoordinatorIdentity  *Identity              `json:"coordinator_identity"`
	ProviderAvailability []ProviderAvailability `json:"provider_availability"`
	CompatibilityDigest  string                 `json:"compatibility_digest"`
}

type ObservedPackage struct {
	PackageID            string `json:"package_id"`
	Version              string `json:"version"`
	InstallRoot          string `json:"install_root"`
	ReceiptProtocol      string `json:"receipt_protocol"`
	ReceiptDigest        string `json:"receipt_digest"`
	Integrity            string `json:"integrity"`
	EntryPointsValidated bool   `json:"entry_points_validated"`
}

type ObservedComponent struct {
	ComponentID           string            `json:"component_id"`
	ComponentKind         string            `json:"component_kind"`
	ModuleID              string            `json:"module_id"`
	VectorID              *string           `json:"vector_id"`
	EngineID              *string           `json:"engine_id"`
	Packages              []ObservedPackage `json:"packages"`
	SelectedPackageDigest *string           `json:"selected_package_digest"`
	Activation            string            `json:"activation"`
	Docking               string            `json:"docking"`
	ReceptorID            *string           `json:"receptor_id"`
	Capabilities          []string          `json:"capabilities"`
	PlatformCompatibility string            `json:"platform_compatibility"`
	ObservationDigest     string            `json:"observation_digest"`
}

type UnknownPackage struct {
	InstallRoot string `json:"install_root"`
	ReceiptPath string `json:"receipt_path"`
	Reason      string `json:"reason"`
	Preserved   bool   `json:"preserved"`
}

type Observation struct {
	Protocol              string              `json:"protocol"`
	FormatVersion         uint64              `json:"format_version"`
	ProfileID             string              `json:"profile_id"`
	TOPSID                string              `json:"tops_id"`
	ConfiguredRoots       []string            `json:"configured_roots"`
	Platform              Platform            `json:"platform"`
	BindingRegistryDigest *string             `json:"binding_registry_digest"`
	Components            []ObservedComponent `json:"components"`
	UnknownPackages       []UnknownPackage    `json:"unknown_packages"`
	ObservedAt            string              `json:"observed_at"`
	Canonical             bool                `json:"canonical"`
	ObservationDigest     string              `json:"observation_digest"`
}

type ObservationInput struct {
	ProfileID             string
	TOPSID                string
	ConfiguredRoots       []string
	DesiredState          *DesiredState
	BindingRegistryDigest *string
	SelectedReceipts      map[string]string
	RuntimeState          *RuntimeState
	QxctlIdentity         Identity
	CoordinatorIdentity   *Identity
	ProviderAvailability  []ProviderAvailability
	ObservedAt            time.Time
}

type receiptCandidate struct {
	root         string
	relativePath string
	module       string
	version      string
	data         []byte
	readable     bool
}

type componentIdentity struct {
	componentID string
	kind        string
	moduleID    string
	vectorID    *string
	engineID    *string
	role        string
}

type packageEvidence struct {
	identity     componentIdentity
	packageState ObservedPackage
	capabilities []string
	compatible   bool
	receiptPath  string
}

type receiptV1 struct {
	Protocol        string   `json:"protocol"`
	ModuleID        string   `json:"module_id"`
	Version         string   `json:"version"`
	InstallScope    string   `json:"install_scope"`
	PrefixMode      string   `json:"prefix_mode"`
	State           string   `json:"state"`
	Active          bool     `json:"active"`
	DefaultReceptor *string  `json:"default_receptor"`
	Files           []string `json:"files"`
}

type receiptV2File struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   uint64 `json:"size"`
	Digest string `json:"digest"`
}

type receiptV2EntryPoint struct {
	EntryPointID string   `json:"entry_point_id"`
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	Protocols    []string `json:"protocols"`
}

type receiptV2Platform struct {
	OS           string  `json:"os"`
	Architecture string  `json:"architecture"`
	KernelABI    *string `json:"kernel_abi"`
	Critical     bool    `json:"critical"`
}

type receiptV2 struct {
	Protocol             string                `json:"protocol"`
	FormatVersion        uint64                `json:"format_version"`
	ComponentID          string                `json:"component_id"`
	ComponentKind        string                `json:"component_kind"`
	ModuleID             string                `json:"module_id"`
	VectorID             *string               `json:"vector_id"`
	EngineID             *string               `json:"engine_id"`
	PackageID            string                `json:"package_id"`
	Version              string                `json:"version"`
	InstallScope         string                `json:"install_scope"`
	PrefixMode           string                `json:"prefix_mode"`
	Files                []receiptV2File       `json:"files"`
	EntryPoints          []receiptV2EntryPoint `json:"entry_points"`
	ProvidesCapabilities []string              `json:"provides_capabilities"`
	RequiresCapabilities []string              `json:"requires_capabilities"`
	CompatibleReceptors  []string              `json:"compatible_receptors"`
	PlatformRequirements []receiptV2Platform   `json:"platform_requirements"`
	ReceiptDigest        string                `json:"receipt_digest"`
}

var v1Identities = map[string]componentIdentity{
	"knowledge-session-coordinator": {
		componentID: "knowledge-session-coordinator", kind: "coordinator",
		moduleID: "knowledge-session-coordinator", engineID: stringPointer("symphony-knowledge-session"), role: "coordinator",
	},
	"skvi-engine": {componentID: "skvi-engine", kind: "vector_engine", moduleID: "skvi-engine", vectorID: stringPointer("skvi"), engineID: stringPointer("symphony-skvi"), role: "skvi"},
	"sclv-engine": {componentID: "sclv-engine", kind: "vector_engine", moduleID: "sclv-engine", vectorID: stringPointer("sclv"), engineID: stringPointer("symphony-sclv"), role: "sclv"},
	"sacv-engine": {componentID: "sacv-engine", kind: "vector_engine", moduleID: "sacv-engine", vectorID: stringPointer("sacv"), engineID: stringPointer("symphony-sacv"), role: "sacv"},
	"sodv-engine": {componentID: "sodv-engine", kind: "vector_engine", moduleID: "sodv-engine", vectorID: stringPointer("sodv"), engineID: stringPointer("symphony-sodv"), role: "sodv"},
	"ssfv-engine": {componentID: "ssfv-engine", kind: "vector_engine", moduleID: "ssfv-engine", vectorID: stringPointer("ssfv"), engineID: stringPointer("symphony-ssfv"), role: "ssfv"},
}

func Observe(input ObservationInput) (Observation, error) {
	if !safeToken(input.ProfileID, 256) || !validTOPSID(input.TOPSID) ||
		len(input.ConfiguredRoots) == 0 || len(input.ConfiguredRoots) > 64 ||
		!validIdentity(input.QxctlIdentity) ||
		(input.CoordinatorIdentity != nil && !validIdentity(*input.CoordinatorIdentity)) {
		return Observation{}, fmt.Errorf("lifecycle observation input identity is invalid")
	}
	roots := append([]string(nil), input.ConfiguredRoots...)
	sort.Strings(roots)
	for index, root := range roots {
		if !safeAbsolutePath(root) || (index > 0 && root == roots[index-1]) {
			return Observation{}, fmt.Errorf("configured roots are invalid or duplicated")
		}
	}
	if input.DesiredState != nil && (input.DesiredState.ProfileID != input.ProfileID || input.DesiredState.TOPSID != input.TOPSID) {
		return Observation{}, fmt.Errorf("desired state does not match observation identity")
	}
	if input.BindingRegistryDigest != nil && !taggedDigest(*input.BindingRegistryDigest) {
		return Observation{}, fmt.Errorf("binding registry digest is invalid")
	}
	if len(input.SelectedReceipts) > 256 {
		return Observation{}, fmt.Errorf("selected receipt set exceeds its bound")
	}
	runtimeComponents := make(map[string]RuntimeComponent)
	if input.RuntimeState != nil {
		if input.RuntimeState.ProfileID != input.ProfileID || input.RuntimeState.TOPSID != input.TOPSID {
			return Observation{}, fmt.Errorf("lifecycle runtime state does not match observation identity")
		}
		for _, component := range input.RuntimeState.Components {
			runtimeComponents[component.ComponentID] = component
		}
	}
	for role, digest := range input.SelectedReceipts {
		if !safeToken(role, 256) || !taggedDigest(digest) {
			return Observation{}, fmt.Errorf("selected receipt identity is invalid")
		}
	}
	providers := append([]ProviderAvailability(nil), input.ProviderAvailability...)
	sort.Slice(providers, func(i, j int) bool { return canonicalLess(providers[i], providers[j]) })
	seenProviders := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		if !safeToken(provider.ProviderID, 256) {
			return Observation{}, fmt.Errorf("provider availability identity is invalid")
		}
		if _, duplicate := seenProviders[provider.ProviderID]; duplicate {
			return Observation{}, fmt.Errorf("provider availability is duplicated")
		}
		seenProviders[provider.ProviderID] = struct{}{}
	}
	platform := Platform{
		OS: runtime.GOOS, KernelABI: kernelABI(), Architecture: runtime.GOARCH,
		QxctlIdentity: input.QxctlIdentity, CoordinatorIdentity: input.CoordinatorIdentity,
		ProviderAvailability: providers,
	}
	if platform.OS != "linux" && platform.OS != "darwin" {
		return Observation{}, fmt.Errorf("native lifecycle observation is unsupported on %s", platform.OS)
	}
	if platform.OS == "darwin" {
		platform.OS = "macos"
	}
	platformDigestInput := platform
	platformDigestInput.CompatibilityDigest = ""
	platformObject, err := objectWithout(mustJSON(platformDigestInput), "compatibility_digest")
	if err != nil {
		return Observation{}, err
	}
	platform.CompatibilityDigest, err = digestValue(platformObject)
	if err != nil {
		return Observation{}, err
	}

	candidates, err := scanReceiptCandidates(roots)
	if err != nil {
		return Observation{}, err
	}
	if len(candidates) > 4096 {
		return Observation{}, fmt.Errorf("receipt inventory exceeds 4096 entries")
	}
	evidence := make([]packageEvidence, 0, len(candidates))
	unknown := make([]UnknownPackage, 0)
	for _, candidate := range candidates {
		if !candidate.readable {
			unknown = append(unknown, unknownFrom(candidate, "unreadable"))
			continue
		}
		protocol, err := receiptProtocol(candidate.data)
		if err != nil {
			unknown = append(unknown, unknownFrom(candidate, "invalid_receipt"))
			continue
		}
		var item packageEvidence
		switch protocol {
		case "symphony.knowledge.install-receipt.v1":
			item, err = observeV1(candidate)
		case "symphony.knowledge.install-receipt.v2":
			item, err = observeV2(candidate, platform)
		default:
			unknown = append(unknown, unknownFrom(candidate, "unsupported_protocol"))
			continue
		}
		if err != nil {
			unknown = append(unknown, unknownFrom(candidate, "invalid_receipt"))
			continue
		}
		evidence = append(evidence, item)
	}

	grouped := make(map[string][]packageEvidence)
	for _, item := range evidence {
		grouped[item.identity.componentID] = append(grouped[item.identity.componentID], item)
	}
	components := make([]ObservedComponent, 0, len(grouped))
	for componentID, items := range grouped {
		identity := items[0].identity
		conflict := false
		seenReceipts := make(map[string]struct{}, len(items))
		for _, item := range items {
			if !sameComponentIdentity(identity, item.identity) {
				conflict = true
			}
			if _, duplicate := seenReceipts[item.packageState.ReceiptDigest]; duplicate {
				conflict = true
			}
			seenReceipts[item.packageState.ReceiptDigest] = struct{}{}
		}
		if conflict {
			for _, item := range items {
				unknown = append(unknown, UnknownPackage{
					InstallRoot: item.packageState.InstallRoot, ReceiptPath: item.receiptPath,
					Reason: "ambiguous_identity", Preserved: true,
				})
			}
			continue
		}
		component := ObservedComponent{
			ComponentID: componentID, ComponentKind: identity.kind, ModuleID: identity.moduleID,
			VectorID: cloneString(identity.vectorID), EngineID: cloneString(identity.engineID),
			Packages: make([]ObservedPackage, 0, len(items)), Activation: "inactive",
			Docking: "undocked", Capabilities: []string{}, PlatformCompatibility: "compatible",
		}
		for _, item := range items {
			component.Packages = append(component.Packages, item.packageState)
		}
		sort.Slice(component.Packages, func(i, j int) bool { return canonicalLess(component.Packages[i], component.Packages[j]) })
		if selected, ok := input.SelectedReceipts[identity.role]; ok && identity.role != "" {
			for _, item := range items {
				if item.packageState.ReceiptDigest == selected {
					component.SelectedPackageDigest = stringPointer(selected)
					component.Capabilities = append([]string(nil), item.capabilities...)
					component.PlatformCompatibility = compatibilityText(item.compatible)
					break
				}
			}
		}
		if runtimeComponent, ok := runtimeComponents[componentID]; ok {
			if component.SelectedPackageDigest == nil && runtimeComponent.SelectedReceiptDigest != nil {
				for _, item := range items {
					if item.packageState.ReceiptDigest == *runtimeComponent.SelectedReceiptDigest {
						component.SelectedPackageDigest = cloneString(runtimeComponent.SelectedReceiptDigest)
						component.Capabilities = append([]string(nil), item.capabilities...)
						component.PlatformCompatibility = compatibilityText(item.compatible)
						break
					}
				}
			}
			if component.SelectedPackageDigest != nil && runtimeComponent.SelectedReceiptDigest != nil &&
				*component.SelectedPackageDigest == *runtimeComponent.SelectedReceiptDigest {
				component.Activation = runtimeComponent.Activation
			}
		}
		if component.SelectedPackageDigest == nil && input.DesiredState != nil {
			for _, desired := range input.DesiredState.Components {
				if desired.ComponentID != componentID || desired.SelectedPackage == nil {
					continue
				}
				for _, item := range items {
					if item.packageState.ReceiptDigest == desired.SelectedPackage.ReceiptDigest {
						component.PlatformCompatibility = compatibilityText(item.compatible)
						break
					}
				}
			}
		}
		sort.Strings(component.Capabilities)
		component.ObservationDigest, err = componentObservationDigest(component)
		if err != nil {
			return Observation{}, err
		}
		components = append(components, component)
	}
	sort.Slice(components, func(i, j int) bool { return canonicalLess(components[i], components[j]) })
	sort.Slice(unknown, func(i, j int) bool { return canonicalLess(unknown[i], unknown[j]) })
	observedAt := input.ObservedAt.UTC().Truncate(time.Second)
	if observedAt.IsZero() {
		observedAt = Timestamp()
	}
	observation := Observation{
		Protocol: ObservationProtocol, FormatVersion: 1, ProfileID: input.ProfileID,
		TOPSID: input.TOPSID, ConfiguredRoots: roots, Platform: platform,
		BindingRegistryDigest: cloneString(input.BindingRegistryDigest), Components: components,
		UnknownPackages: unknown, ObservedAt: observedAt.Format(time.RFC3339), Canonical: false,
	}
	observation.ObservationDigest, err = observationDigest(observation)
	if err != nil {
		return Observation{}, err
	}
	return observation, nil
}

func StableInventoryDigest(observation Observation) (string, error) {
	observation.ObservationDigest = ""
	observation.ObservedAt = ""
	value := mustJSON(observation)
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return "", err
	}
	delete(object, "observation_digest")
	delete(object, "observed_at")
	normalizeObservationObject(object)
	return digestValue(object)
}

func DigestCurrentExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve qxctl executable: %w", err)
	}
	absolute, err := filepath.Abs(executable)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("qxctl executable must be a no-follow regular file")
	}
	return hashRegularFile(absolute, 128*1024*1024)
}

func observeV1(candidate receiptCandidate) (packageEvidence, error) {
	identity, ok := v1Identities[candidate.module]
	if !ok {
		return packageEvidence{}, fmt.Errorf("no v1 adapter exists for module")
	}
	var receipt receiptV1
	if err := decodeExact(candidate.data, &receipt); err != nil {
		return packageEvidence{}, err
	}
	if receipt.Protocol != "symphony.knowledge.install-receipt.v1" || receipt.ModuleID != candidate.module ||
		receipt.Version != candidate.version || receipt.InstallScope != "prefix" ||
		receipt.PrefixMode != "installation_prefix" || receipt.State != "installed_undocked" ||
		receipt.Active || receipt.DefaultReceptor != nil {
		return packageEvidence{}, fmt.Errorf("v1 receipt identity or lifecycle state is invalid")
	}
	installed, err := knowledgeengine.InspectInstallation(identity.role, candidate.root, candidate.version)
	if err != nil {
		return packageEvidence{}, err
	}
	rawDigest := sha256.Sum256(candidate.data)
	receiptDigest := "sha256:" + hex.EncodeToString(rawDigest[:])
	if installed.ReceiptDigest != receiptDigest || installed.ModuleID != candidate.module {
		return packageEvidence{}, fmt.Errorf("v1 adapter receipt identity mismatch")
	}
	return packageEvidence{
		identity: identity,
		packageState: ObservedPackage{
			PackageID: candidate.module, Version: candidate.version, InstallRoot: candidate.root,
			ReceiptProtocol: receipt.Protocol, ReceiptDigest: receiptDigest,
			Integrity: "valid", EntryPointsValidated: true,
		},
		capabilities: []string{}, compatible: true, receiptPath: candidate.relativePath,
	}, nil
}

func observeV2(candidate receiptCandidate, platform Platform) (packageEvidence, error) {
	var receipt receiptV2
	if err := decodeExact(candidate.data, &receipt); err != nil {
		return packageEvidence{}, err
	}
	if err := validateReceiptV2(candidate, receipt); err != nil {
		return packageEvidence{}, err
	}
	integrity := "valid"
	fileKinds := make(map[string]string, len(receipt.Files))
	for _, file := range receipt.Files {
		fileKinds[file.Path] = file.Kind
		digest, size, err := hashTrustedRelative(candidate.root, file.Path, maxObservedFileBytes(file.Kind))
		if err != nil || size != file.Size || digest != file.Digest {
			integrity = "invalid"
		}
	}
	entryPointsValid := true
	for _, entry := range receipt.EntryPoints {
		kind, exists := fileKinds[entry.Path]
		if !exists || ((entry.Kind == "executable" || entry.Kind == "adapter") && kind != "executable") {
			entryPointsValid = false
		}
	}
	compatible := true
	for _, requirement := range receipt.PlatformRequirements {
		matches := requirement.OS == platform.OS && requirement.Architecture == platform.Architecture &&
			(requirement.KernelABI == nil || *requirement.KernelABI == platform.KernelABI)
		if requirement.Critical && !matches {
			compatible = false
		}
	}
	return packageEvidence{
		identity: componentIdentity{
			componentID: receipt.ComponentID, kind: receipt.ComponentKind, moduleID: receipt.ModuleID,
			vectorID: cloneString(receipt.VectorID), engineID: cloneString(receipt.EngineID),
		},
		packageState: ObservedPackage{
			PackageID: receipt.PackageID, Version: receipt.Version, InstallRoot: candidate.root,
			ReceiptProtocol: receipt.Protocol, ReceiptDigest: receipt.ReceiptDigest,
			Integrity: integrity, EntryPointsValidated: entryPointsValid,
		},
		capabilities: append([]string(nil), receipt.ProvidesCapabilities...), compatible: compatible,
		receiptPath: candidate.relativePath,
	}, nil
}

func validateReceiptV2(candidate receiptCandidate, receipt receiptV2) error {
	if receipt.Protocol != "symphony.knowledge.install-receipt.v2" || receipt.FormatVersion != 2 ||
		receipt.ModuleID != candidate.module || receipt.Version != candidate.version ||
		!safeToken(receipt.ComponentID, 256) || !safeToken(receipt.ModuleID, 256) ||
		!safeToken(receipt.PackageID, 256) || !safeVersion(receipt.Version) ||
		!oneOf(receipt.ComponentKind, "coordinator", "vector_engine", "module", "adapter", "ui", "service") ||
		!oneOf(receipt.InstallScope, "prefix", "user", "system", "tops") ||
		receipt.PrefixMode != "installation_prefix" || !taggedDigest(receipt.ReceiptDigest) ||
		len(receipt.Files) == 0 || len(receipt.Files) > 4096 || len(receipt.EntryPoints) > 128 ||
		receipt.EntryPoints == nil || receipt.ProvidesCapabilities == nil ||
		receipt.RequiresCapabilities == nil || receipt.CompatibleReceptors == nil ||
		receipt.PlatformRequirements == nil {
		return fmt.Errorf("v2 receipt identity or collection bound is invalid")
	}
	for _, optional := range []*string{receipt.VectorID, receipt.EngineID} {
		if optional != nil && !safeToken(*optional, 256) {
			return fmt.Errorf("v2 receipt optional identity is invalid")
		}
	}
	seenFiles := make(map[string]struct{}, len(receipt.Files))
	for _, file := range receipt.Files {
		if !safeRelativePath(file.Path) || !oneOf(file.Kind, "regular", "executable") || !taggedDigest(file.Digest) {
			return fmt.Errorf("v2 receipt file entry is invalid")
		}
		if _, duplicate := seenFiles[file.Path]; duplicate {
			return fmt.Errorf("v2 receipt file path is duplicated")
		}
		seenFiles[file.Path] = struct{}{}
	}
	seenEntries := make(map[string]struct{}, len(receipt.EntryPoints))
	for _, entry := range receipt.EntryPoints {
		if !safeToken(entry.EntryPointID, 256) || !oneOf(entry.Kind, "executable", "descriptor", "adapter") ||
			!safeRelativePath(entry.Path) || len(entry.Protocols) > 64 || validateTokenSet(entry.Protocols, 64) != nil {
			return fmt.Errorf("v2 receipt entry point is invalid")
		}
		if _, duplicate := seenEntries[entry.EntryPointID]; duplicate {
			return fmt.Errorf("v2 receipt entry point identity is duplicated")
		}
		seenEntries[entry.EntryPointID] = struct{}{}
	}
	for _, values := range [][]string{receipt.ProvidesCapabilities, receipt.RequiresCapabilities, receipt.CompatibleReceptors} {
		if err := validateTokenSet(values, 128); err != nil {
			return fmt.Errorf("v2 receipt capability or receptor set is invalid")
		}
	}
	if len(receipt.PlatformRequirements) > 128 {
		return fmt.Errorf("v2 platform requirement bound is exceeded")
	}
	for _, requirement := range receipt.PlatformRequirements {
		if !oneOf(requirement.OS, "linux", "macos") || !safeToken(requirement.Architecture, 256) ||
			(requirement.KernelABI != nil && !safeToken(*requirement.KernelABI, 256)) {
			return fmt.Errorf("v2 platform requirement is invalid")
		}
	}
	receiptCopy := receipt
	receiptCopy.ReceiptDigest = ""
	value, err := objectWithout(mustJSON(receiptCopy), "receipt_digest")
	if err != nil {
		return err
	}
	digest, err := digestValue(value)
	if err != nil || digest != receipt.ReceiptDigest {
		return fmt.Errorf("v2 receipt digest mismatch")
	}
	return nil
}

func receiptProtocol(data []byte) (string, error) {
	if err := knowledgeengine.ValidateJSONObject(data, maxReceiptBytes); err != nil {
		return "", err
	}
	var header struct {
		Protocol string `json:"protocol"`
	}
	if err := json.Unmarshal(data, &header); err != nil || header.Protocol == "" {
		return "", fmt.Errorf("receipt protocol is absent")
	}
	return header.Protocol, nil
}

func componentObservationDigest(component ObservedComponent) (string, error) {
	component.ObservationDigest = ""
	sort.Slice(component.Packages, func(i, j int) bool { return canonicalLess(component.Packages[i], component.Packages[j]) })
	sort.Strings(component.Capabilities)
	value, err := objectWithout(mustJSON(component), "observation_digest")
	if err != nil {
		return "", err
	}
	return digestValue(value)
}

func observationDigest(observation Observation) (string, error) {
	observation.ObservationDigest = ""
	value, err := objectWithout(mustJSON(observation), "observation_digest")
	if err != nil {
		return "", err
	}
	normalizeObservationObject(value)
	return digestValue(value)
}

func normalizeObservationObject(value map[string]any) {
	sortAnyArray(value["configured_roots"])
	if platform, ok := value["platform"].(map[string]any); ok {
		sortAnyArray(platform["provider_availability"])
	}
	if components, ok := value["components"].([]any); ok {
		for _, item := range components {
			if component, ok := item.(map[string]any); ok {
				sortAnyArray(component["packages"])
				sortAnyArray(component["capabilities"])
			}
		}
		sort.Slice(components, func(i, j int) bool { return canonicalLess(components[i], components[j]) })
		value["components"] = components
	}
	sortAnyArray(value["unknown_packages"])
}

func sortAnyArray(value any) {
	if array, ok := value.([]any); ok {
		sort.Slice(array, func(i, j int) bool { return canonicalLess(array[i], array[j]) })
	}
}

func unknownFrom(candidate receiptCandidate, reason string) UnknownPackage {
	return UnknownPackage{
		InstallRoot: candidate.root, ReceiptPath: candidate.relativePath,
		Reason: reason, Preserved: true,
	}
}

func sameComponentIdentity(left, right componentIdentity) bool {
	return left.componentID == right.componentID && left.kind == right.kind && left.moduleID == right.moduleID &&
		equalStringPointer(left.vectorID, right.vectorID) && equalStringPointer(left.engineID, right.engineID)
}

func equalStringPointer(left, right *string) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	return stringPointer(*value)
}

func compatibilityText(value bool) string {
	if value {
		return "compatible"
	}
	return "incompatible"
}

func validIdentity(identity Identity) bool {
	return safeToken(identity.ComponentID, 256) && safeVersion(identity.Version) && taggedDigest(identity.ExecutableDigest)
}

func safeRelativePath(value string) bool {
	if value == "" || len(value) > 4096 || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || strings.Contains(value, "//") {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
		for _, character := range component {
			if character < 0x20 || character == 0x7f {
				return false
			}
		}
	}
	return true
}

func maxObservedFileBytes(kind string) int64 {
	if kind == "executable" {
		return 64 * 1024 * 1024
	}
	return 4 * 1024 * 1024
}

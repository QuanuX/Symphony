package knowledgeengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
)

// ReceiptV2EntryPointSpec identifies one typed executable owned by one exact
// immutable package receipt. It is also used by the existing engine verifier,
// so foundational adapters cannot take a weaker parallel provenance path.
type ReceiptV2EntryPointSpec struct {
	Label                  string
	ComponentID            string
	ComponentKind          string
	ModuleID               string
	PackageID              string
	VectorID               *string
	EngineID               *string
	EntryPointID           string
	EntryPointKind         string
	EntryPointRelativePath string
	RequiredProtocols      []string
	RequiredCapabilities   []string
	RequiredReceptors      []string
}

type ReceiptV2EntryPoint struct {
	Prefix           string
	Version          string
	ReceiptPath      string
	ReceiptDigest    string
	ExecutablePath   string
	ExecutableDigest string
}

// DiscoverReceiptV2Versions returns only safe directory names beneath one
// exact module receipt namespace. Callers must validate each receipt before
// selecting it; lexical order is presentation only and never recency.
func DiscoverReceiptV2Versions(prefix, moduleID string) ([]string, error) {
	canonicalPrefix, err := canonicalDirectory(prefix, "installation prefix")
	if err != nil {
		return nil, err
	}
	if !safeToken(moduleID, 256) {
		return nil, fmt.Errorf("invalid receipt-v2 module identity")
	}
	if err := validateTrustedInstallPrefix(canonicalPrefix); err != nil {
		return nil, fmt.Errorf("installation prefix is untrusted: %w", err)
	}
	directory := filepath.Join(canonicalPrefix, "share", "symphony", "receipts", moduleID)
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("receipt-v2 module namespace is unavailable or unsafe")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read receipt-v2 module namespace: %w", err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil || entry.Type()&os.ModeSymlink != 0 || !entryInfo.IsDir() || !safeVersion(entry.Name()) {
			return nil, fmt.Errorf("receipt-v2 module namespace contains an unsafe version entry")
		}
		versions = append(versions, entry.Name())
	}
	sort.Strings(versions)
	return versions, nil
}

// InspectReceiptV2EntryPoint validates the complete receipt-owned package and
// returns only the exact typed entry point requested by the caller.
func InspectReceiptV2EntryPoint(prefix, version string, spec ReceiptV2EntryPointSpec) (ReceiptV2EntryPoint, error) {
	canonicalPrefix, err := canonicalDirectory(prefix, "installation prefix")
	if err != nil {
		return ReceiptV2EntryPoint{}, err
	}
	if !safeVersion(version) || !safeToken(spec.Label, 256) || !safeToken(spec.ComponentID, 256) ||
		!safeToken(spec.ComponentKind, 256) || !safeToken(spec.ModuleID, 256) || !safeToken(spec.PackageID, 256) ||
		!safeToken(spec.EntryPointID, 256) || !safeRelativePath(spec.EntryPointRelativePath) ||
		(spec.EntryPointKind != "executable" && spec.EntryPointKind != "adapter") ||
		!validUniqueTokens(spec.RequiredProtocols, 64) || len(spec.RequiredProtocols) == 0 ||
		!validOptionalUniqueTokens(spec.RequiredCapabilities, 128) || !validOptionalUniqueTokens(spec.RequiredReceptors, 128) {
		return ReceiptV2EntryPoint{}, fmt.Errorf("invalid receipt-v2 entry-point expectation")
	}
	if err := validateTrustedInstallPrefix(canonicalPrefix); err != nil {
		return ReceiptV2EntryPoint{}, fmt.Errorf("installation prefix is untrusted: %w", err)
	}
	receiptRelative := filepath.ToSlash(filepath.Join(
		"share", "symphony", "receipts", spec.ModuleID, version, "install-receipt.json"))
	receiptBytes, err := readTrustedNoFollowRelative(canonicalPrefix, receiptRelative, maxReceiptBytes)
	if err != nil {
		return ReceiptV2EntryPoint{}, fmt.Errorf("read validated %s receipt: %w", spec.Label, err)
	}
	path, receiptDigest, executableDigest, err := validateReceiptV2EntryPoint(
		canonicalPrefix, version, receiptRelative, receiptBytes, spec)
	if err != nil {
		return ReceiptV2EntryPoint{}, err
	}
	return ReceiptV2EntryPoint{
		Prefix: canonicalPrefix, Version: version,
		ReceiptPath:   filepath.Join(canonicalPrefix, filepath.FromSlash(receiptRelative)),
		ReceiptDigest: receiptDigest, ExecutablePath: path, ExecutableDigest: executableDigest,
	}, nil
}

func validateReceiptV2EntryPoint(prefix, version, receiptRelative string, receiptBytes []byte, spec ReceiptV2EntryPointSpec) (string, string, string, error) {
	if err := validateJSONObject(receiptBytes, maxReceiptBytes); err != nil {
		return "", "", "", fmt.Errorf("invalid %s receipt-v2 JSON: %w", spec.Label, err)
	}
	if err := requireExactFields(receiptBytes, []string{
		"protocol", "format_version", "component_id", "component_kind", "module_id",
		"vector_id", "engine_id", "package_id", "version", "install_scope", "prefix_mode",
		"files", "entry_points", "provides_capabilities", "requires_capabilities",
		"compatible_receptors", "platform_requirements", "receipt_digest",
	}); err != nil {
		return "", "", "", fmt.Errorf("invalid %s receipt-v2 fields: %w", spec.Label, err)
	}
	var installed receiptV2
	if err := decodeExact(receiptBytes, &installed); err != nil {
		return "", "", "", fmt.Errorf("decode %s receipt-v2: %w", spec.Label, err)
	}
	if installed.Protocol != receiptProtocolV2 || installed.FormatVersion != 2 ||
		installed.ComponentID != spec.ComponentID || installed.ModuleID != spec.ModuleID ||
		installed.PackageID != spec.PackageID || installed.Version != version || installed.ComponentKind != spec.ComponentKind ||
		!equalOptionalString(installed.VectorID, spec.VectorID) || !equalOptionalString(installed.EngineID, spec.EngineID) ||
		installed.InstallScope != "prefix" || installed.PrefixMode != "installation_prefix" ||
		len(installed.Files) == 0 || len(installed.Files) > 4096 || installed.EntryPoints == nil ||
		installed.ProvidesCapabilities == nil || installed.RequiresCapabilities == nil ||
		installed.CompatibleReceptors == nil || installed.PlatformRequirements == nil {
		return "", "", "", fmt.Errorf("%s receipt-v2 identity or collection contract mismatch", spec.Label)
	}
	receiptDigest, protocol, err := validatedReceiptIdentity(receiptBytes)
	if err != nil || protocol != receiptProtocolV2 || receiptDigest != installed.ReceiptDigest {
		return "", "", "", fmt.Errorf("validate %s receipt-v2 digest", spec.Label)
	}
	seen := make(map[string]receiptV2File, len(installed.Files))
	for _, file := range installed.Files {
		if !safeRelativePath(file.Path) || (file.Kind != "regular" && file.Kind != "executable") ||
			!taggedSHA256(file.Digest) || file.Path == receiptRelative {
			return "", "", "", fmt.Errorf("%s receipt-v2 contains an invalid file entry", spec.Label)
		}
		if _, duplicate := seen[file.Path]; duplicate {
			return "", "", "", fmt.Errorf("%s receipt-v2 contains a duplicate path", spec.Label)
		}
		data, err := readTrustedNoFollowRelative(prefix, file.Path, maxInstalledFileBytes(file.Path))
		if err != nil {
			return "", "", "", fmt.Errorf("validate receipt-v2-owned file %s: %w", file.Path, err)
		}
		digest := digestBytes(data)
		if uint64(len(data)) != file.Size || digest != file.Digest {
			return "", "", "", fmt.Errorf("receipt-v2-owned file content mismatch: %s", file.Path)
		}
		seen[file.Path] = file
	}
	if !validUniqueTokens(installed.ProvidesCapabilities, 128) || !validUniqueTokens(installed.RequiresCapabilities, 128) ||
		!validUniqueTokens(installed.CompatibleReceptors, 128) ||
		!containsEvery(installed.ProvidesCapabilities, spec.RequiredCapabilities) || !containsEvery(installed.CompatibleReceptors, spec.RequiredReceptors) {
		return "", "", "", fmt.Errorf("%s receipt-v2 capability or receptor set is incompatible", spec.Label)
	}
	platformOS := runtime.GOOS
	if platformOS == "darwin" {
		platformOS = "macos"
	}
	if len(installed.PlatformRequirements) == 0 || len(installed.PlatformRequirements) > 128 {
		return "", "", "", fmt.Errorf("%s receipt-v2 lacks bounded platform requirements", spec.Label)
	}
	seenPlatforms, matchingCriticalPlatform := map[string]struct{}{}, false
	for _, requirement := range installed.PlatformRequirements {
		if requirement.OS != "linux" && requirement.OS != "macos" || !safeToken(requirement.Architecture, 256) ||
			requirement.KernelABI != nil && !safeToken(*requirement.KernelABI, 256) {
			return "", "", "", fmt.Errorf("%s receipt-v2 platform requirement is invalid", spec.Label)
		}
		kernelABI := ""
		if requirement.KernelABI != nil {
			kernelABI = *requirement.KernelABI
		}
		key := requirement.OS + "\n" + requirement.Architecture + "\n" + kernelABI + "\n" + strconv.FormatBool(requirement.Critical)
		if _, duplicate := seenPlatforms[key]; duplicate {
			return "", "", "", fmt.Errorf("%s receipt-v2 platform requirement is duplicated", spec.Label)
		}
		seenPlatforms[key] = struct{}{}
		if requirement.Critical && (requirement.OS != platformOS || requirement.Architecture != runtime.GOARCH || requirement.KernelABI != nil) {
			return "", "", "", fmt.Errorf("%s receipt-v2 is incompatible with this platform", spec.Label)
		}
		matchingCriticalPlatform = matchingCriticalPlatform || requirement.Critical
	}
	if !matchingCriticalPlatform {
		return "", "", "", fmt.Errorf("%s receipt-v2 lacks an exact critical host platform requirement", spec.Label)
	}
	seenEntries, matched := map[string]struct{}{}, false
	for _, entry := range installed.EntryPoints {
		if !safeToken(entry.EntryPointID, 256) || !safeRelativePath(entry.Path) ||
			(entry.Kind != "executable" && entry.Kind != "adapter" && entry.Kind != "descriptor") || !validUniqueTokens(entry.Protocols, 64) {
			return "", "", "", fmt.Errorf("%s receipt-v2 entry point is invalid", spec.Label)
		}
		if _, duplicate := seenEntries[entry.EntryPointID]; duplicate {
			return "", "", "", fmt.Errorf("%s receipt-v2 entry point is duplicated", spec.Label)
		}
		seenEntries[entry.EntryPointID] = struct{}{}
		if _, owned := seen[entry.Path]; !owned {
			return "", "", "", fmt.Errorf("%s receipt-v2 entry point is not receipt-owned", spec.Label)
		}
		if entry.EntryPointID == spec.EntryPointID && entry.Kind == spec.EntryPointKind && entry.Path == spec.EntryPointRelativePath && containsEvery(entry.Protocols, spec.RequiredProtocols) {
			matched = true
		}
	}
	if !matched {
		return "", "", "", fmt.Errorf("%s receipt-v2 lacks the exact typed entry point", spec.Label)
	}
	owned := seen[spec.EntryPointRelativePath]
	if owned.Kind != "executable" {
		return "", "", "", fmt.Errorf("%s typed entry point is not executable receipt content", spec.Label)
	}
	binary := filepath.Join(prefix, filepath.FromSlash(spec.EntryPointRelativePath))
	info, err := os.Lstat(binary)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 || info.Mode().Perm()&0o022 != 0 {
		return "", "", "", fmt.Errorf("%s typed entry point is not a protected no-follow executable regular file", spec.Label)
	}
	return binary, receiptDigest, owned.Digest, nil
}

func digestBytes(value []byte) string {
	digest := sha256Sum(value)
	return "sha256:" + hex.EncodeToString(digest)
}

func sha256Sum(value []byte) []byte {
	hash := sha256.New()
	_, _ = hash.Write(value)
	return hash.Sum(nil)
}

func equalOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func containsEvery(have, required []string) bool {
	for _, value := range required {
		if !containsExact(have, value) {
			return false
		}
	}
	return true
}

func validOptionalUniqueTokens(values []string, maximum int) bool {
	return values == nil || validUniqueTokens(values, maximum)
}

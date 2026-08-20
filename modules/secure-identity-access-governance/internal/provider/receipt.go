package provider

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

type receiptFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   uint64 `json:"size"`
	Digest string `json:"digest"`
}
type receiptEntryPoint struct {
	EntryPointID string   `json:"entry_point_id"`
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	Protocols    []string `json:"protocols"`
}
type receiptPlatform struct {
	OS           string  `json:"os"`
	Architecture string  `json:"architecture"`
	KernelABI    *string `json:"kernel_abi"`
	Critical     bool    `json:"critical"`
}
type installReceiptV2 struct {
	Protocol             string              `json:"protocol"`
	FormatVersion        uint64              `json:"format_version"`
	ComponentID          string              `json:"component_id"`
	ComponentKind        string              `json:"component_kind"`
	ModuleID             string              `json:"module_id"`
	VectorID             *string             `json:"vector_id"`
	EngineID             *string             `json:"engine_id"`
	PackageID            string              `json:"package_id"`
	Version              string              `json:"version"`
	InstallScope         string              `json:"install_scope"`
	PrefixMode           string              `json:"prefix_mode"`
	Files                []receiptFile       `json:"files"`
	EntryPoints          []receiptEntryPoint `json:"entry_points"`
	ProvidesCapabilities []string            `json:"provides_capabilities"`
	RequiresCapabilities []string            `json:"requires_capabilities"`
	CompatibleReceptors  []string            `json:"compatible_receptors"`
	PlatformRequirements []receiptPlatform   `json:"platform_requirements"`
	ReceiptDigest        string              `json:"receipt_digest"`
}

const (
	macOSKeychainPackageID      = "ssiag-macos-keychain-provider"
	macOSKeychainEntryPointID   = "ssiag.macos-keychain-provider"
	macOSKeychainExecutableName = "symphony-ssiag-provider-macos-keychain"
	macOSKeychainBundleName     = "SymphonySSIAGMacOSKeychainProvider.app"
	providerMetadataCapability  = "symphony.ssiag.provider.metadata.v1"
	providerLauncherReceptor    = "symphony.ssiag.provider-launcher.v1"
)

type adapterPackage struct {
	prefix      string
	receiptPath string
	relative    string
	bundleRoot  string
	receipt     installReceiptV2
	receiptUID  uint32
	receiptGID  uint32
}

func validateAdapterReceipt(declaration ExecutableTrust, scope ssiagpaths.Scope) error {
	_, err := inspectAdapterPackage(declaration, scope)
	return err
}

func inspectAdapterPackage(declaration ExecutableTrust, scope ssiagpaths.Scope) (adapterPackage, error) {
	parts := strings.Split(filepath.ToSlash(declaration.ExecutablePath), "/")
	if len(parts) < 6 {
		return adapterPackage{}, fmt.Errorf("provider executable is outside a versioned package")
	}
	marker := -1
	for i := range parts {
		if parts[i] == "libexec" && i+4 < len(parts) && parts[i+1] == "symphony" {
			marker = i
			break
		}
	}
	if marker <= 0 || parts[marker+2] != macOSKeychainPackageID || parts[marker+3] != declaration.AdapterVersion {
		return adapterPackage{}, fmt.Errorf("provider executable path is not an exact versioned package entry point")
	}
	legacy := marker+5 == len(parts) && parts[marker+4] == macOSKeychainExecutableName
	bundle := marker+8 == len(parts) && parts[marker+4] == macOSKeychainBundleName && parts[marker+5] == "Contents" &&
		parts[marker+6] == "MacOS" && parts[marker+7] == macOSKeychainExecutableName
	if !legacy && !bundle {
		return adapterPackage{}, fmt.Errorf("provider executable path is not an exact versioned package entry point")
	}
	prefix := "/" + filepath.Join(parts[1:marker]...)
	packageID := parts[marker+2]
	receiptPath := filepath.Join(prefix, "share", "symphony", "receipts", packageID, declaration.AdapterVersion, "install-receipt.json")
	if err := validateProviderAncestors(declaration.ExecutablePath); err != nil {
		return adapterPackage{}, err
	}
	if err := validateProviderAncestors(receiptPath); err != nil {
		return adapterPackage{}, err
	}
	receipt, receiptUID, receiptGID, err := readAdapterReceipt(receiptPath, scope)
	if err != nil {
		return adapterPackage{}, err
	}
	if receipt.Protocol != "symphony.knowledge.install-receipt.v2" || receipt.FormatVersion != 2 ||
		receipt.ComponentID != macOSKeychainPackageID || receipt.ComponentKind != "adapter" ||
		receipt.ModuleID != macOSKeychainPackageID || receipt.PackageID != packageID || receipt.Version != declaration.AdapterVersion ||
		receipt.InstallScope != "prefix" || receipt.PrefixMode != "installation_prefix" || receipt.VectorID != nil || receipt.EngineID != nil ||
		receipt.ReceiptDigest != declaration.InstallationDigest || receipt.ReceiptDigest != receiptDigest(receipt) {
		return adapterPackage{}, fmt.Errorf("provider receipt identity mismatch")
	}
	if !sameStrings(receipt.ProvidesCapabilities, []string{providerMetadataCapability}) || len(receipt.RequiresCapabilities) != 0 ||
		!sameStrings(receipt.CompatibleReceptors, []string{providerLauncherReceptor}) {
		return adapterPackage{}, fmt.Errorf("provider receipt capability arrays are not deterministic")
	}
	wantOS := runtime.GOOS
	if wantOS == "darwin" {
		wantOS = "macos"
	}
	platform := false
	if len(receipt.PlatformRequirements) != 1 {
		return adapterPackage{}, fmt.Errorf("provider receipt platform requirements are not exact")
	}
	for _, item := range receipt.PlatformRequirements {
		if item.Critical && item.OS == wantOS && item.Architecture == runtime.GOARCH && item.KernelABI == nil {
			platform = true
		}
	}
	if !platform {
		return adapterPackage{}, fmt.Errorf("provider receipt does not admit the current platform")
	}
	relative, err := filepath.Rel(prefix, declaration.ExecutablePath)
	if err != nil || strings.HasPrefix(relative, "..") {
		return adapterPackage{}, fmt.Errorf("provider executable escapes prefix")
	}
	relative = filepath.ToSlash(relative)
	if len(receipt.EntryPoints) != 1 || len(receipt.Files) == 0 || len(receipt.Files) > 4096 {
		return adapterPackage{}, fmt.Errorf("provider receipt ownership is not exact")
	}
	entryMatches := 0
	for _, entry := range receipt.EntryPoints {
		if entry.EntryPointID == macOSKeychainEntryPointID && entry.Kind == "adapter" && entry.Path == relative &&
			len(entry.Protocols) == 1 && contains(entry.Protocols, ProviderControlProtocol) {
			entryMatches++
		}
	}
	if entryMatches != 1 {
		return adapterPackage{}, fmt.Errorf("provider receipt entry point mismatch")
	}
	allowedBundlePaths := map[string]string{}
	bundleRelativeRoot := filepath.ToSlash(filepath.Join("libexec", "symphony", macOSKeychainPackageID, declaration.AdapterVersion, macOSKeychainBundleName))
	for suffix, kind := range map[string]string{
		"Contents/Info.plist":                           "regular",
		"Contents/MacOS/" + macOSKeychainExecutableName: "executable",
		"Contents/Resources/ssiag-signing-policy.json":  "regular",
		"Contents/_CodeSignature/CodeResources":         "regular",
		"Contents/embedded.provisionprofile":            "regular",
	} {
		allowedBundlePaths[bundleRelativeRoot+"/"+suffix] = kind
	}
	if legacy && len(receipt.Files) != 1 {
		return adapterPackage{}, fmt.Errorf("legacy provider receipt ownership is not exact")
	}
	if bundle && len(receipt.Files) < 2 {
		return adapterPackage{}, fmt.Errorf("provider bundle receipt is incomplete")
	}
	fileFound := false
	seen := make(map[string]struct{}, len(receipt.Files))
	previous := ""
	for index, file := range receipt.Files {
		if !validReceiptPath(file.Path) || file.Size == 0 || file.Size > 64<<20 || !validDigest(file.Digest) {
			return adapterPackage{}, fmt.Errorf("provider receipt file evidence is invalid")
		}
		if _, exists := seen[file.Path]; exists || index > 0 && file.Path <= previous {
			return adapterPackage{}, fmt.Errorf("provider receipt file ordering is not deterministic")
		}
		seen[file.Path] = struct{}{}
		previous = file.Path
		if legacy && (file.Path != relative || file.Kind != "executable") {
			return adapterPackage{}, fmt.Errorf("legacy provider receipt contains an unknown file")
		}
		if bundle {
			kind, admitted := allowedBundlePaths[file.Path]
			if !admitted || file.Kind != kind {
				return adapterPackage{}, fmt.Errorf("provider bundle receipt contains an unknown file")
			}
		}
		path := filepath.Join(prefix, filepath.FromSlash(file.Path))
		if !pathWithinPrefix(prefix, path) {
			return adapterPackage{}, fmt.Errorf("provider receipt file escapes prefix")
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || uint64(info.Size()) != file.Size || info.Mode().Perm()&0o022 != 0 {
			return adapterPackage{}, fmt.Errorf("provider package file is unsafe")
		}
		uid, gid, err := fileOwner(info)
		if err != nil || uid != receiptUID || gid != receiptGID || uid != declaration.OwnerUID || gid != declaration.OwnerGID ||
			file.Kind == "executable" && fmt.Sprintf("0%03o", info.Mode().Perm()) != declaration.FileMode {
			return adapterPackage{}, fmt.Errorf("provider package ownership mismatch")
		}
		if scope == ssiagpaths.ScopeUser && uid != uint32(os.Geteuid()) {
			return adapterPackage{}, fmt.Errorf("user provider package owner mismatch")
		}
		if scope == ssiagpaths.ScopeSystem && uid != 0 {
			return adapterPackage{}, fmt.Errorf("system provider package owner mismatch")
		}
		digest, err := digestPathBounded(path, 64<<20)
		if err != nil || digest != file.Digest {
			return adapterPackage{}, fmt.Errorf("provider package digest mismatch")
		}
		if file.Path == relative {
			if file.Kind != "executable" || file.Size > maximumProviderExecutableBytes || file.Digest != declaration.ExecutableDigest {
				return adapterPackage{}, fmt.Errorf("provider receipt executable evidence mismatch")
			}
			fileFound = true
		}
	}
	if !fileFound {
		return adapterPackage{}, fmt.Errorf("provider receipt does not own its entry point")
	}
	bundleRoot := ""
	if bundle {
		bundleRoot = filepath.Join(prefix, filepath.FromSlash(bundleRelativeRoot))
		if _, found := seen[bundleRelativeRoot+"/Contents/Info.plist"]; !found {
			return adapterPackage{}, fmt.Errorf("provider bundle receipt lacks Info.plist")
		}
		if err := rejectUnreceiptedBundleFiles(bundleRoot, prefix, seen); err != nil {
			return adapterPackage{}, err
		}
	}
	return adapterPackage{
		prefix: prefix, receiptPath: receiptPath, relative: relative, bundleRoot: bundleRoot,
		receipt: receipt, receiptUID: receiptUID, receiptGID: receiptGID,
	}, nil
}

func validReceiptPath(value string) bool {
	if value == "" || len(value) > 4096 || filepath.IsAbs(value) || strings.Contains(value, "\\") || strings.Contains(value, "//") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func pathWithinPrefix(prefix, path string) bool {
	relative, err := filepath.Rel(prefix, path)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func rejectUnreceiptedBundleFiles(bundleRoot, prefix string, owned map[string]struct{}) error {
	return filepath.WalkDir(bundleRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("inspect provider bundle: %w", walkErr)
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("provider bundle contains an unsafe entry")
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(prefix, path)
		if err != nil {
			return fmt.Errorf("provider bundle entry escapes prefix")
		}
		if _, exists := owned[filepath.ToSlash(relative)]; !exists {
			return fmt.Errorf("provider bundle contains unreceipted bytes")
		}
		return nil
	})
}

func readAdapterReceipt(path string, scope ssiagpaths.Scope) (installReceiptV2, uint32, uint32, error) {
	file, err := openProviderFile(path)
	if err != nil {
		return installReceiptV2{}, 0, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 1<<20 || info.Mode().Perm()&0o022 != 0 {
		return installReceiptV2{}, 0, 0, fmt.Errorf("provider receipt is unsafe")
	}
	uid, gid, err := fileOwner(info)
	if err != nil || scope == ssiagpaths.ScopeUser && uid != uint32(os.Geteuid()) || scope == ssiagpaths.ScopeSystem && uid != 0 {
		return installReceiptV2{}, 0, 0, fmt.Errorf("provider receipt owner mismatch")
	}
	payload, err := io.ReadAll(io.LimitReader(file, 1<<20+1))
	if err != nil || len(payload) > 1<<20 || validateJSONMembers(payload) != nil || validateReceiptShape(payload) != nil {
		return installReceiptV2{}, 0, 0, fmt.Errorf("provider receipt JSON is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var receipt installReceiptV2
	if err := decoder.Decode(&receipt); err != nil {
		return installReceiptV2{}, 0, 0, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return installReceiptV2{}, 0, 0, fmt.Errorf("multiple receipt values")
	}
	return receipt, uid, gid, nil
}

func validateProviderAncestors(path string) error {
	for current := filepath.Dir(filepath.Clean(path)); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("provider package ancestor is unavailable")
		}
		if info.Mode()&os.ModeSymlink != 0 && permittedProviderSystemAlias(current) {
			// macOS exposes these fixed root-owned compatibility aliases. Their
			// exact destinations are verified before traversal continues.
		} else {
			uid, _, ownerErr := fileOwner(info)
			if ownerErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 ||
				(uid != uint32(os.Geteuid()) && uid != 0) {
				return fmt.Errorf("provider package has an unsafe ancestor")
			}
		}
		if filepath.Dir(current) == current {
			return nil
		}
	}
}

func permittedProviderSystemAlias(path string) bool {
	expected := map[string]string{"/var": "/private/var", "/tmp": "/private/tmp", "/etc": "/private/etc"}
	want, ok := expected[path]
	if !ok {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == want
}

func receiptDigest(receipt installReceiptV2) string {
	receipt.ReceiptDigest = ""
	encoded, _ := json.Marshal(receipt)
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	_ = decoder.Decode(&object)
	delete(object, "receipt_digest")
	canonical, _ := json.Marshal(object)
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func digestPath(path string) (string, error) {
	return digestPathBounded(path, maximumProviderExecutableBytes)
}

func digestPathBounded(path string, maximum int64) (string, error) {
	file, err := openProviderFile(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum {
		return "", fmt.Errorf("provider executable size is unsafe")
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, io.LimitReader(file, maximum+1))
	if copyErr != nil {
		return "", copyErr
	}
	if written != info.Size() || written > maximum {
		return "", fmt.Errorf("provider executable changed or exceeded its size bound")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func validateReceiptShape(payload []byte) error {
	object, err := requireJSONObjectFields(payload, []string{
		"protocol", "format_version", "component_id", "component_kind", "module_id", "vector_id", "engine_id", "package_id",
		"version", "install_scope", "prefix_mode", "files", "entry_points", "provides_capabilities", "requires_capabilities",
		"compatible_receptors", "platform_requirements", "receipt_digest",
	})
	if err != nil {
		return err
	}
	var files []json.RawMessage
	var entries []json.RawMessage
	var platforms []json.RawMessage
	if json.Unmarshal(object["files"], &files) != nil || json.Unmarshal(object["entry_points"], &entries) != nil ||
		json.Unmarshal(object["platform_requirements"], &platforms) != nil {
		return fmt.Errorf("provider receipt arrays are invalid")
	}
	for _, file := range files {
		if _, err := requireJSONObjectFields(file, []string{"path", "kind", "size", "digest"}); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		if _, err := requireJSONObjectFields(entry, []string{"entry_point_id", "kind", "path", "protocols"}); err != nil {
			return err
		}
	}
	for _, platform := range platforms {
		if _, err := requireJSONObjectFields(platform, []string{"os", "architecture", "kernel_abi", "critical"}); err != nil {
			return err
		}
	}
	return nil
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func sortedUnique(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for i, value := range values {
		if !validToken(value) || i > 0 && value == values[i-1] {
			return false
		}
	}
	return true
}

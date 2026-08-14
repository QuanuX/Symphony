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
	providerMetadataCapability  = "symphony.ssiag.provider.metadata.v1"
	providerLauncherReceptor    = "symphony.ssiag.provider-launcher.v1"
)

func validateAdapterReceipt(declaration ExecutableTrust, scope ssiagpaths.Scope) error {
	parts := strings.Split(filepath.ToSlash(declaration.ExecutablePath), "/")
	if len(parts) < 6 {
		return fmt.Errorf("provider executable is outside a versioned package")
	}
	marker := -1
	for i := range parts {
		if parts[i] == "libexec" && i+4 < len(parts) && parts[i+1] == "symphony" {
			marker = i
			break
		}
	}
	if marker <= 0 || parts[marker+2] != macOSKeychainPackageID || parts[marker+3] != declaration.AdapterVersion ||
		parts[marker+4] != macOSKeychainExecutableName || marker+5 != len(parts) {
		return fmt.Errorf("provider executable path is not an exact versioned package entry point")
	}
	prefix := "/" + filepath.Join(parts[1:marker]...)
	packageID := parts[marker+2]
	receiptPath := filepath.Join(prefix, "share", "symphony", "receipts", packageID, declaration.AdapterVersion, "install-receipt.json")
	if err := validateProviderAncestors(declaration.ExecutablePath); err != nil {
		return err
	}
	if err := validateProviderAncestors(receiptPath); err != nil {
		return err
	}
	receipt, receiptUID, receiptGID, err := readAdapterReceipt(receiptPath, scope)
	if err != nil {
		return err
	}
	if receipt.Protocol != "symphony.knowledge.install-receipt.v2" || receipt.FormatVersion != 2 ||
		receipt.ComponentID != macOSKeychainPackageID || receipt.ComponentKind != "adapter" ||
		receipt.ModuleID != macOSKeychainPackageID || receipt.PackageID != packageID || receipt.Version != declaration.AdapterVersion ||
		receipt.InstallScope != "prefix" || receipt.PrefixMode != "installation_prefix" || receipt.VectorID != nil || receipt.EngineID != nil ||
		receipt.ReceiptDigest != declaration.InstallationDigest || receipt.ReceiptDigest != receiptDigest(receipt) {
		return fmt.Errorf("provider receipt identity mismatch")
	}
	if !sameStrings(receipt.ProvidesCapabilities, []string{providerMetadataCapability}) || len(receipt.RequiresCapabilities) != 0 ||
		!sameStrings(receipt.CompatibleReceptors, []string{providerLauncherReceptor}) {
		return fmt.Errorf("provider receipt capability arrays are not deterministic")
	}
	wantOS := runtime.GOOS
	if wantOS == "darwin" {
		wantOS = "macos"
	}
	platform := false
	if len(receipt.PlatformRequirements) != 1 {
		return fmt.Errorf("provider receipt platform requirements are not exact")
	}
	for _, item := range receipt.PlatformRequirements {
		if item.Critical && item.OS == wantOS && item.Architecture == runtime.GOARCH && item.KernelABI == nil {
			platform = true
		}
	}
	if !platform {
		return fmt.Errorf("provider receipt does not admit the current platform")
	}
	relative, err := filepath.Rel(prefix, declaration.ExecutablePath)
	if err != nil || strings.HasPrefix(relative, "..") {
		return fmt.Errorf("provider executable escapes prefix")
	}
	relative = filepath.ToSlash(relative)
	if len(receipt.EntryPoints) != 1 || len(receipt.Files) != 1 {
		return fmt.Errorf("provider receipt ownership is not exact")
	}
	entryMatches := 0
	for _, entry := range receipt.EntryPoints {
		if entry.EntryPointID == macOSKeychainEntryPointID && entry.Kind == "adapter" && entry.Path == relative &&
			len(entry.Protocols) == 1 && contains(entry.Protocols, ProviderControlProtocol) {
			entryMatches++
		}
	}
	if entryMatches != 1 {
		return fmt.Errorf("provider receipt entry point mismatch")
	}
	fileFound := false
	for _, file := range receipt.Files {
		if file.Path != relative {
			continue
		}
		if file.Kind != "executable" || file.Size == 0 || file.Size > maximumProviderExecutableBytes || file.Digest != declaration.ExecutableDigest {
			return fmt.Errorf("provider receipt executable evidence mismatch")
		}
		info, err := os.Lstat(declaration.ExecutablePath)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || uint64(info.Size()) != file.Size || info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("provider executable is unsafe")
		}
		uid, gid, err := fileOwner(info)
		if err != nil || uid != receiptUID || gid != receiptGID || uid != declaration.OwnerUID || gid != declaration.OwnerGID ||
			fmt.Sprintf("0%03o", info.Mode().Perm()) != declaration.FileMode {
			return fmt.Errorf("provider executable ownership mismatch")
		}
		if scope == ssiagpaths.ScopeUser && uid != uint32(os.Geteuid()) {
			return fmt.Errorf("user provider executable owner mismatch")
		}
		if scope == ssiagpaths.ScopeSystem && uid != 0 {
			return fmt.Errorf("system provider executable owner mismatch")
		}
		digest, err := digestPath(declaration.ExecutablePath)
		if err != nil || digest != file.Digest {
			return fmt.Errorf("provider executable digest mismatch")
		}
		fileFound = true
	}
	if !fileFound {
		return fmt.Errorf("provider receipt does not own its entry point")
	}
	return nil
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
	file, err := openProviderFile(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumProviderExecutableBytes {
		return "", fmt.Errorf("provider executable size is unsafe")
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, io.LimitReader(file, maximumProviderExecutableBytes+1))
	if copyErr != nil {
		return "", copyErr
	}
	if written != info.Size() || written > maximumProviderExecutableBytes {
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

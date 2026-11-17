package packageinstall

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageretirement"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

const (
	ReceiptProtocol = "symphony.knowledge.install-receipt.v2"
	ComponentID     = "secure-identity-access-governance"
	EntryPointID    = "ssiag.foundation-lifecycle"
)

type File struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   uint64 `json:"size"`
	Digest string `json:"digest"`
}
type EntryPoint struct {
	EntryPointID string   `json:"entry_point_id"`
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	Protocols    []string `json:"protocols"`
}
type PlatformRequirement struct {
	OS           string  `json:"os"`
	Architecture string  `json:"architecture"`
	KernelABI    *string `json:"kernel_abi"`
	Critical     bool    `json:"critical"`
}
type Receipt struct {
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
	Files                []File                `json:"files"`
	EntryPoints          []EntryPoint          `json:"entry_points"`
	ProvidesCapabilities []string              `json:"provides_capabilities"`
	RequiresCapabilities []string              `json:"requires_capabilities"`
	CompatibleReceptors  []string              `json:"compatible_receptors"`
	PlatformRequirements []PlatformRequirement `json:"platform_requirements"`
	ReceiptDigest        string                `json:"receipt_digest"`
}
type Result struct {
	Prefix, Version, Binary, Receipt, ReceiptDigest string
	Changed                                         bool
}
type Evidence struct{ Prefix, Version, Binary, BinaryDigest, Receipt, ReceiptDigest string }

func Install(source, prefix, version string) (Result, error) {
	return InstallWithDelegation(source, prefix, version, nil)
}

// InstallWithDelegation retains the parent lifecycle root lock and independently validates its exact ownership claim.
func InstallWithDelegation(source, prefix, version string, delegation *packageretirement.Delegation) (Result, error) {
	if version != compiledVersion() {
		return Result{}, fmt.Errorf("SSIAG package version %q does not match compiled binary version %q", version, compiledVersion())
	}
	layout, err := resolve(prefix, version)
	if err != nil {
		return Result{}, err
	}
	self, err := os.Executable()
	if err != nil {
		return Result{}, err
	}
	sourceUse, err := acquireInstallSourceUse(self, layout.binary)
	if err != nil {
		return Result{}, err
	}
	defer sourceUse.Close()
	if err := ensurePrefix(layout.prefix); err != nil {
		return Result{}, err
	}
	if err := validatePrefixTrust(layout.prefix); err != nil {
		return Result{}, err
	}
	size, digest, err := sourceEvidence(source)
	if err != nil {
		return Result{}, err
	}
	selfSize, selfDigest, err := sourceEvidence(self)
	if err != nil || selfSize != size || selfDigest != digest {
		return Result{}, fmt.Errorf("guarded SSIAG installation requires the executing owner binary")
	}
	receipt := newReceipt(layout, size, digest)
	receipt.ReceiptDigest, err = digestWithoutReceipt(receipt)
	if err != nil {
		return Result{}, err
	}
	_, binaryErr := os.Lstat(layout.binary)
	_, receiptErr := os.Lstat(layout.receipt)
	lease, err := packageretirement.PrepareInstall(layout.prefix, retirementIdentity(layout, receipt), os.IsNotExist(binaryErr) && os.IsNotExist(receiptErr), delegation)
	if err != nil {
		return Result{}, err
	}
	defer lease.Close()
	if existing, existingBytes, exists, err := readGuardedReceiptBytes(lease, layout); err != nil {
		return Result{}, err
	} else if exists {
		if err := validateReceipt(existing, layout); err != nil {
			return Result{}, fmt.Errorf("existing SSIAG receipt-v2 package is invalid: %w", err)
		}
		if existing.ReceiptDigest != receipt.ReceiptDigest {
			return Result{}, fmt.Errorf("immutable SSIAG package version already exists with different bytes")
		}
		if err := activatePackage(lease, layout, existing, existingBytes); err != nil {
			return Result{}, err
		}
		return Result{Prefix: layout.prefix, Version: version, Binary: layout.binary, Receipt: layout.receipt, ReceiptDigest: receipt.ReceiptDigest}, nil
	}
	sourceFile, err := os.Open(source)
	if err != nil {
		return Result{}, err
	}
	_, publishErr := lease.PublishFile(layout.binaryRelative, sourceFile, digest, size, 0o755)
	closeErr := sourceFile.Close()
	if publishErr != nil {
		return Result{}, publishErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return Result{}, err
	}
	receiptBytes = append(receiptBytes, '\n')
	rawDigest := sha256.Sum256(receiptBytes)
	if _, err := lease.PublishFile(receiptRelative(layout), bytes.NewReader(receiptBytes), "sha256:"+hex.EncodeToString(rawDigest[:]), uint64(len(receiptBytes)), 0o644); err != nil {
		return Result{}, err
	}
	if err := activatePackage(lease, layout, receipt, receiptBytes); err != nil {
		return Result{}, err
	}
	return Result{Prefix: layout.prefix, Version: version, Binary: layout.binary, Receipt: layout.receipt, ReceiptDigest: receipt.ReceiptDigest, Changed: true}, nil
}

// A same-package reinstall acquires exclusive target custody in PrepareInstall.
// Taking a shared source lease first would conflict with that lock. A different
// installed source still remains pinned throughout the target installation.
func acquireInstallSourceUse(executable, target string) (*packageretirement.Lease, error) {
	if filepath.Clean(executable) == target {
		return nil, nil
	}
	return AcquireRecoveryForExecutable(executable)
}

// Activation is conditional on both exact published objects still being present
// at their pinned package paths, including the receipt bytes validated above.
func activatePackage(lease *packageretirement.Lease, l layout, receipt Receipt, raw []byte) error {
	if err := lease.VerifyFile(l.binaryRelative, receipt.Files[0].Digest, receipt.Files[0].Size); err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if err := lease.VerifyFile(receiptRelative(l), "sha256:"+hex.EncodeToString(digest[:]), uint64(len(raw))); err != nil {
		return err
	}
	return lease.Activate()
}

func Uninstall(prefix, version string) (Result, error) {
	return UninstallWithDelegation(prefix, version, nil)
}

// UninstallWithDelegation preserves all native retirement and reference gates.
func UninstallWithDelegation(prefix, version string, delegation *packageretirement.Delegation) (Result, error) {
	if os.Geteuid() == 0 {
		return Result{}, fmt.Errorf("system-owned SSIAG reclamation is unqualified; retain bytes for target-host owner maintenance")
	}
	if version != compiledVersion() {
		return Result{}, fmt.Errorf("SSIAG package version %q does not match compiled binary version %q", version, compiledVersion())
	}
	layout, err := resolve(prefix, version)
	if err != nil {
		return Result{}, err
	}
	id, _, err := packageretirement.Inspect(layout.prefix, ComponentID, version)
	if err != nil {
		return Result{}, err
	}
	if id.BinaryRelative != layout.binaryRelative || id.ReceiptRelative != receiptRelative(layout) {
		return Result{}, fmt.Errorf("SSIAG retirement path identity differs")
	}
	self, err := os.Executable()
	if err != nil {
		return Result{}, err
	}
	size, digest, err := sourceEvidence(self)
	if err != nil || digest != id.BinaryDigest || size != id.BinarySize {
		return Result{}, fmt.Errorf("SSIAG removal requires the exact retained native owner binary")
	}
	if target, err := os.Stat(layout.binary); err == nil {
		if running, err := os.Stat(self); err == nil && os.SameFile(target, running) {
			return Result{}, fmt.Errorf("SSIAG cannot reclaim its executing controller; use a retained exact owner outside this installation")
		}
	}
	lease, err := packageretirement.BeginRetirement(layout.prefix, id, delegation)
	if err != nil {
		return Result{}, err
	}
	defer lease.Close()
	result := Result{Prefix: layout.prefix, Version: version, Binary: layout.binary, Receipt: layout.receipt, ReceiptDigest: id.ReceiptDigest}
	receipt, receiptBytes, exists, err := readGuardedReceiptBytes(lease, layout)
	if err != nil {
		return Result{}, err
	}
	if !exists {
		if _, err := os.Lstat(layout.binary); !os.IsNotExist(err) {
			return Result{}, fmt.Errorf("SSIAG retirement receipt is absent but payload remains or is unsafe")
		}
		if err := lease.FinishRetirement(); err != nil {
			return Result{}, err
		}
		return result, nil
	}
	if lease.State() == "retired" {
		return Result{}, fmt.Errorf("retired SSIAG installation contains reappeared package files")
	}
	if err := validateReceiptIdentity(receipt, layout); err != nil {
		return Result{}, err
	}
	if retirementIdentity(layout, receipt) != id {
		return Result{}, fmt.Errorf("SSIAG retirement receipt changed")
	}
	if references, err := retainedReferences(layout, receipt); err != nil {
		return Result{}, err
	} else if len(references) != 0 {
		return Result{}, fmt.Errorf("refusing to uninstall referenced SSIAG package: %s", strings.Join(references, ", "))
	}
	receiptSize := uint64(len(receiptBytes))
	rawHash := sha256.Sum256(receiptBytes)
	receiptHash := "sha256:" + hex.EncodeToString(rawHash[:])
	if err := lease.VerifyFile(id.ReceiptRelative, receiptHash, receiptSize); err != nil {
		return Result{}, err
	}
	if err := lease.RemoveFile(layout.binaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		return Result{}, err
	}
	if err := lease.RemoveFile(id.ReceiptRelative, receiptHash, receiptSize); err != nil {
		return Result{}, err
	}
	if err := lease.FinishRetirement(); err != nil {
		return Result{}, err
	}
	result.Changed = true
	return result, nil
}

func readGuardedReceipt(lease *packageretirement.Lease, l layout) (Receipt, bool, error) {
	r, _, exists, err := readGuardedReceiptBytes(lease, l)
	return r, exists, err
}
func readGuardedReceiptBytes(lease *packageretirement.Lease, l layout) (Receipt, []byte, bool, error) {
	data, err := lease.ReadFile(receiptRelative(l), 1<<20)
	if os.IsNotExist(err) {
		return Receipt{}, nil, false, nil
	}
	if err != nil {
		return Receipt{}, nil, false, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var receipt Receipt
	if err := decoder.Decode(&receipt); err != nil {
		return Receipt{}, nil, false, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Receipt{}, nil, false, fmt.Errorf("SSIAG receipt contains trailing data")
	}
	return receipt, data, true, nil
}

func receiptRelative(l layout) string {
	r, _ := filepath.Rel(l.prefix, l.receipt)
	return filepath.ToSlash(r)
}
func retirementIdentity(l layout, r Receipt) packageretirement.Identity {
	return packageretirement.Identity{Owner: ComponentID, Version: l.version, ReceiptDigest: r.ReceiptDigest, BinaryDigest: r.Files[0].Digest, BinaryRelative: l.binaryRelative, ReceiptRelative: receiptRelative(l), BinarySize: r.Files[0].Size}
}

// AcquireUseForExecutable covers only immutable packages installed by this
// participating owner. Other executable layouts retain their legacy contract;
// a matching package layout without its guard always fails closed.
func AcquireUseForExecutable(executable string) (*packageretirement.Lease, error) {
	return acquireExecutable(executable, false)
}

// AcquireRecoveryForExecutable is for the existing native controller, never
// service admission; mutation owners still separately reject retired starts.
func AcquireRecoveryForExecutable(executable string) (*packageretirement.Lease, error) {
	return acquireExecutable(executable, true)
}
func acquireExecutable(executable string, recovery bool) (*packageretirement.Lease, error) {
	executable = filepath.Clean(executable)
	parts := strings.Split(filepath.ToSlash(executable), "/")
	if len(parts) < 6 || parts[len(parts)-5] != "libexec" || parts[len(parts)-4] != "symphony" || parts[len(parts)-3] != ComponentID || parts[len(parts)-1] != "symphony-ssiag" {
		return nil, nil
	}
	prefix := executable
	for range 5 {
		prefix = filepath.Dir(prefix)
	}
	layout, err := resolve(prefix, parts[len(parts)-2])
	if err != nil {
		return nil, err
	}
	if layout.version != compiledVersion() {
		return nil, fmt.Errorf("SSIAG package version differs from executing owner")
	}
	id, _, err := packageretirement.Inspect(layout.prefix, ComponentID, layout.version)
	if err != nil {
		return nil, err
	}
	if id.BinaryRelative != layout.binaryRelative || id.ReceiptRelative != receiptRelative(layout) {
		return nil, fmt.Errorf("SSIAG package retirement path differs")
	}
	var lease *packageretirement.Lease
	if recovery {
		lease, err = packageretirement.AcquireRecoveryUse(layout.prefix, id)
	} else {
		lease, err = packageretirement.AcquireUse(layout.prefix, id)
	}
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*packageretirement.Lease, error) { _ = lease.Close(); return nil, err }
	receipt, exists, err := readGuardedReceipt(lease, layout)
	if err != nil {
		return fail(err)
	}
	if !exists {
		return fail(fmt.Errorf("SSIAG package receipt missing"))
	}
	if err := validateReceiptIdentity(receipt, layout); err != nil {
		return fail(err)
	}
	if retirementIdentity(layout, receipt) != id {
		return fail(fmt.Errorf("SSIAG package receipt differs from admitted generation"))
	}
	if err := lease.VerifyFile(layout.binaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		return fail(err)
	}
	return lease, nil
}

func InspectExecutable(executable string) (Evidence, bool, error) {
	executable = filepath.Clean(executable)
	directory := filepath.Dir(executable)
	prefix := directory
	for range 4 {
		prefix = filepath.Dir(prefix)
	}
	relative, err := filepath.Rel(prefix, executable)
	if err != nil {
		return Evidence{}, false, err
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) != 5 || parts[0] != "libexec" || parts[1] != "symphony" || parts[2] != ComponentID || parts[4] != "symphony-ssiag" {
		return Evidence{}, false, nil
	}
	if parts[3] != compiledVersion() {
		return Evidence{}, false, fmt.Errorf("SSIAG package path version %q does not match compiled binary version %q", parts[3], compiledVersion())
	}
	layout, err := resolve(prefix, parts[3])
	if err != nil || layout.binary != executable {
		return Evidence{}, false, err
	}
	receipt, exists, err := readReceipt(layout.receipt)
	if err != nil || !exists {
		return Evidence{}, false, err
	}
	if err := validateReceipt(receipt, layout); err != nil {
		return Evidence{}, false, err
	}
	return Evidence{Prefix: prefix, Version: parts[3], Binary: executable, BinaryDigest: receipt.Files[0].Digest, Receipt: layout.receipt, ReceiptDigest: receipt.ReceiptDigest}, true, nil
}

type layout struct{ prefix, version, binary, binaryRelative, receipt string }

func resolve(prefix, version string) (layout, error) {
	prefix = filepath.Clean(prefix)
	if !filepath.IsAbs(prefix) || prefix == string(filepath.Separator) {
		return layout{}, fmt.Errorf("SSIAG package prefix must be an absolute non-root path")
	}
	var err error
	prefix, err = canonicalPrefix(prefix)
	if err != nil {
		return layout{}, err
	}
	if version == "" || len(version) > 64 {
		return layout{}, fmt.Errorf("SSIAG package version is invalid")
	}
	for _, char := range version {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || strings.ContainsRune(".+-", char)) {
			return layout{}, fmt.Errorf("SSIAG package version is invalid")
		}
	}
	relative := filepath.ToSlash(filepath.Join("libexec", "symphony", ComponentID, version, "symphony-ssiag"))
	return layout{prefix: prefix, version: version, binary: filepath.Join(prefix, filepath.FromSlash(relative)), binaryRelative: relative, receipt: filepath.Join(prefix, "share", "symphony", "receipts", ComponentID, version, "install-receipt.json")}, nil
}

func canonicalPrefix(prefix string) (string, error) {
	current := prefix
	missing := make([]string, 0)
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				return "", fmt.Errorf("SSIAG package prefix ancestor is not a directory")
			}
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("SSIAG package prefix has no existing ancestor")
		}
		current = parent
	}
	resolved, err := filepath.EvalSymlinks(current)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("resolve SSIAG package prefix ancestor: %w", err)
	}
	for index := len(missing) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, missing[index])
	}
	if resolved == string(filepath.Separator) {
		return "", fmt.Errorf("SSIAG package prefix must not resolve to root")
	}
	return resolved, nil
}

func compiledVersion() string { return version.Version }
func newReceipt(layout layout, size uint64, digest string) Receipt {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	return Receipt{Protocol: ReceiptProtocol, FormatVersion: 2, ComponentID: ComponentID, ComponentKind: "service", ModuleID: ComponentID, PackageID: ComponentID, Version: layout.version, InstallScope: "prefix", PrefixMode: "installation_prefix", Files: []File{{Path: layout.binaryRelative, Kind: "executable", Size: size, Digest: digest}}, EntryPoints: []EntryPoint{{EntryPointID: EntryPointID, Kind: "adapter", Path: layout.binaryRelative, Protocols: []string{"symphony.foundation.lifecycle-command.v1"}}}, ProvidesCapabilities: []string{"symphony.foundation.lifecycle-adapter.v1", packageretirement.Capability, packageretirement.DispatchCapability}, RequiresCapabilities: []string{}, CompatibleReceptors: []string{}, PlatformRequirements: []PlatformRequirement{{OS: osName, Architecture: runtime.GOARCH, Critical: true}}}
}
func validateReceipt(receipt Receipt, layout layout) error {
	if err := validateReceiptIdentity(receipt, layout); err != nil {
		return err
	}
	info, err := os.Lstat(layout.binary)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || uint64(info.Size()) != receipt.Files[0].Size {
		return fmt.Errorf("SSIAG package executable is missing or unsafe")
	}
	if !trustedOwner(info) {
		return fmt.Errorf("SSIAG package executable owner is unsafe")
	}
	receiptInfo, receiptErr := os.Lstat(layout.receipt)
	if receiptErr != nil || ownerUID(receiptInfo) != ownerUID(info) {
		return fmt.Errorf("SSIAG package receipt and executable owners differ")
	}
	digest, err := digestFile(layout.binary)
	if err != nil || digest != receipt.Files[0].Digest {
		return fmt.Errorf("SSIAG package executable digest mismatch")
	}
	return nil
}

func validateReceiptIdentity(receipt Receipt, layout layout) error {
	if err := validatePrefixTrust(layout.prefix); err != nil {
		return err
	}
	if err := safeExistingAncestors(layout.binary); err != nil {
		return err
	}
	if err := safeExistingAncestors(layout.receipt); err != nil {
		return err
	}
	wantOS := runtime.GOOS
	if wantOS == "darwin" {
		wantOS = "macos"
	}
	if receipt.Protocol != ReceiptProtocol || receipt.FormatVersion != 2 || receipt.ComponentID != ComponentID || receipt.ComponentKind != "service" || receipt.ModuleID != ComponentID || receipt.VectorID != nil || receipt.EngineID != nil || receipt.PackageID != ComponentID || receipt.Version != layout.version || receipt.Version != compiledVersion() || receipt.InstallScope != "prefix" || receipt.PrefixMode != "installation_prefix" || len(receipt.Files) != 1 || receipt.Files[0].Path != layout.binaryRelative || receipt.Files[0].Kind != "executable" || receipt.Files[0].Size == 0 || !validDigest(receipt.Files[0].Digest) || len(receipt.EntryPoints) != 1 || receipt.EntryPoints[0].EntryPointID != EntryPointID || receipt.EntryPoints[0].Kind != "adapter" || receipt.EntryPoints[0].Path != layout.binaryRelative || len(receipt.EntryPoints[0].Protocols) != 1 || receipt.EntryPoints[0].Protocols[0] != "symphony.foundation.lifecycle-command.v1" || len(receipt.ProvidesCapabilities) != 3 || receipt.ProvidesCapabilities[0] != "symphony.foundation.lifecycle-adapter.v1" || receipt.ProvidesCapabilities[1] != packageretirement.Capability || receipt.ProvidesCapabilities[2] != packageretirement.DispatchCapability || len(receipt.RequiresCapabilities) != 0 || len(receipt.CompatibleReceptors) != 0 || len(receipt.PlatformRequirements) != 1 || receipt.PlatformRequirements[0].OS != wantOS || receipt.PlatformRequirements[0].Architecture != runtime.GOARCH || receipt.PlatformRequirements[0].KernelABI != nil || !receipt.PlatformRequirements[0].Critical {
		return fmt.Errorf("SSIAG package receipt identity is invalid")
	}
	want, err := digestWithoutReceipt(receipt)
	if err != nil || want != receipt.ReceiptDigest {
		return fmt.Errorf("SSIAG package receipt digest mismatch")
	}
	return nil
}
func readReceipt(path string) (Receipt, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return Receipt{}, false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
		return Receipt{}, false, fmt.Errorf("SSIAG package receipt is unsafe")
	}
	if !trustedOwner(info) {
		return Receipt{}, false, fmt.Errorf("SSIAG package receipt owner is unsafe")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Receipt{}, false, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var receipt Receipt
	if err := decoder.Decode(&receipt); err != nil {
		return Receipt{}, false, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Receipt{}, false, fmt.Errorf("SSIAG package receipt contains trailing JSON")
	}
	return receipt, true, nil
}
func digestWithoutReceipt(receipt Receipt) (string, error) {
	receipt.ReceiptDigest = ""
	raw, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	delete(value, "receipt_digest")
	raw, err = json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
func stageExecutable(source, directory string) (string, uint64, string, error) {
	expectedSize, _, err := sourceEvidence(source)
	if err != nil {
		return "", 0, "", err
	}
	input, err := os.Open(source)
	if err != nil {
		return "", 0, "", err
	}
	defer input.Close()
	temp, err := os.CreateTemp(directory, ".ssiag-package-*")
	if err != nil {
		return "", 0, "", err
	}
	path := temp.Name()
	fail := func(err error) (string, uint64, string, error) {
		_ = temp.Close()
		_ = os.Remove(path)
		return "", 0, "", err
	}
	if err := temp.Chmod(0o755); err != nil {
		return fail(err)
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(input, (256<<20)+1))
	if err != nil {
		return fail(err)
	}
	if written > 256<<20 || uint64(written) != expectedSize {
		return fail(fmt.Errorf("SSIAG package source changed while being staged"))
	}
	if err := temp.Sync(); err != nil {
		return fail(err)
	}
	if err := temp.Close(); err != nil {
		return "", 0, "", err
	}
	return path, uint64(written), "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func sourceEvidence(source string) (uint64, string, error) {
	info, err := os.Lstat(source)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 256<<20 {
		return 0, "", fmt.Errorf("SSIAG package source is unsafe")
	}
	digest, err := digestFile(source)
	if err != nil {
		return 0, "", err
	}
	return uint64(info.Size()), digest, nil
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func trustedOwner(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (stat.Uid == uint32(os.Geteuid()) || stat.Uid == 0)
}

func ownerUID(info os.FileInfo) uint32 {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ^uint32(0)
	}
	return stat.Uid
}

func safeExistingAncestors(path string) error {
	for current := filepath.Dir(filepath.Clean(path)); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("SSIAG package path has an unsafe ancestor")
		}
		if filepath.Dir(current) == current {
			return nil
		}
	}
}

func validatePrefixTrust(prefix string) error {
	info, err := os.Lstat(prefix)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !trustedOwner(info) || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("SSIAG package prefix is unsafe")
	}
	return nil
}

func retainedReferences(layout layout, receipt Receipt) ([]string, error) {
	references := make([]string, 0)
	descriptors, err := descriptorCandidates()
	if err != nil {
		return nil, err
	}
	for _, path := range descriptors {
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
			return nil, fmt.Errorf("SSIAG supervisor descriptor reference is unsafe")
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		if bytes.Contains(content, []byte(layout.binary)) {
			references = append(references, "supervisor:"+filepath.Base(path))
		}
	}
	attemptRoots := []string{filepath.Join(userStateBase(), "symphony", "ssiag", "lifecycle")}
	if os.Geteuid() == 0 {
		attemptRoots = append(attemptRoots, "/var/lib/symphony/ssiag/lifecycle")
	}
	for _, root := range attemptRoots {
		attemptReferences, scanErr := scanAttempts(root, receipt.Files[0].Digest, receipt.ReceiptDigest)
		if scanErr != nil {
			return nil, scanErr
		}
		references = append(references, attemptReferences...)
	}
	for _, root := range socketRoots() {
		entries, readErr := os.ReadDir(root)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			topsID := entry.Name()
			for _, suffix := range []string{filepath.Join("ssiag", "ssiag.sock"), filepath.Join("ssiag", "run", "ssiag.sock")} {
				socket := filepath.Join(root, topsID, suffix)
				connection, dialErr := net.DialTimeout("unix", socket, 50*time.Millisecond)
				if dialErr == nil {
					_ = connection.Close()
					references = append(references, "live-endpoint:"+topsID)
					break
				}
			}
		}
	}
	sort.Strings(references)
	return references, nil
}

func descriptorCandidates() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configBase := os.Getenv("XDG_CONFIG_HOME")
	if configBase == "" {
		configBase = filepath.Join(home, ".config")
	}
	patterns := []string{}
	if runtime.GOOS == "darwin" {
		patterns = append(patterns, filepath.Join(home, "Library", "LaunchAgents", "io.github.quanux.symphony.ssiag.*.plist"))
		if os.Geteuid() == 0 {
			patterns = append(patterns, "/Library/LaunchDaemons/io.github.quanux.symphony.ssiag.*.plist")
		}
	} else if runtime.GOOS == "linux" {
		patterns = append(patterns, filepath.Join(configBase, "systemd", "user", "symphony-ssiag@*.service"))
		if os.Geteuid() == 0 {
			patterns = append(patterns, "/etc/systemd/system/symphony-ssiag@*.service")
		}
	}
	result := make([]string, 0)
	for _, pattern := range patterns {
		matches, globErr := filepath.Glob(pattern)
		if globErr != nil {
			return nil, globErr
		}
		result = append(result, matches...)
	}
	return result, nil
}

func scanAttempts(root, binaryDigest, receiptDigest string) ([]string, error) {
	topsEntries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, topsEntry := range topsEntries {
		if !topsEntry.IsDir() {
			continue
		}
		surfaces, readErr := os.ReadDir(filepath.Join(root, topsEntry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		for _, surface := range surfaces {
			if !surface.IsDir() {
				continue
			}
			path := filepath.Join(root, topsEntry.Name(), surface.Name(), "attempt.json")
			info, statErr := os.Lstat(path)
			if os.IsNotExist(statErr) {
				continue
			}
			if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
				return nil, fmt.Errorf("SSIAG lifecycle attempt reference is unsafe")
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil, readErr
			}
			var attempt struct {
				Phase                 string `json:"phase"`
				AuditState            string `json:"audit_state"`
				BinaryDigest          string `json:"binary_digest"`
				InstallEvidenceDigest string `json:"install_evidence_digest"`
			}
			if err := json.Unmarshal(data, &attempt); err != nil {
				return nil, fmt.Errorf("decode SSIAG lifecycle attempt reference: %w", err)
			}
			if (attempt.Phase != "closed" || attempt.AuditState == "audit_deferred") && (attempt.BinaryDigest == binaryDigest || attempt.InstallEvidenceDigest == receiptDigest) {
				result = append(result, "lifecycle:"+topsEntry.Name()+":"+surface.Name())
			}
		}
	}
	return result, nil
}

func userStateBase() string {
	if value := os.Getenv("XDG_STATE_HOME"); value != "" {
		return value
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state")
}

func socketRoots() []string {
	if runtimeBase := os.Getenv("XDG_RUNTIME_DIR"); runtimeBase != "" {
		return []string{filepath.Join(runtimeBase, "symphony")}
	}
	return []string{filepath.Join(userStateBase(), "symphony")}
}
func writeReceiptLast(path string, receipt Receipt) error {
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(filepath.Dir(path), ".receipt-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
func ensurePrefix(path string) error { return ensureDirectory(path) }
func ensureDirectory(path string) error {
	path = filepath.Clean(path)
	var missing []string
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("unsafe SSIAG package directory")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, current)
		if filepath.Dir(current) == current {
			return fmt.Errorf("no safe SSIAG package ancestor")
		}
	}
	for index := len(missing) - 1; index >= 0; index-- {
		if err := os.Mkdir(missing[index], 0o755); err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}
func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

package lifecycle

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
	"strings"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/packageretirement"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/packageuse"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

type Result struct {
	Scope                 stavpaths.Scope
	Binary                string
	BinaryDigest          string
	InstallEvidence       string
	InstallEvidenceDigest string
	Changed               bool
}

type InstallRecord struct {
	Schema        string
	Scope         stavpaths.Scope
	Binary        string
	BinarySHA256  string
	ReceiptDigest string
}

type legacyInstallRecord struct {
	Schema       string          `json:"schema"`
	Scope        stavpaths.Scope `json:"scope"`
	Binary       string          `json:"binary"`
	BinarySHA256 string          `json:"binary_sha256"`
	InstalledAt  string          `json:"installed_at"`
}

type InstallReceipt struct {
	Protocol             string                `json:"protocol"`
	FormatVersion        int                   `json:"format_version"`
	ComponentID          string                `json:"component_id"`
	ComponentKind        string                `json:"component_kind"`
	ModuleID             string                `json:"module_id"`
	VectorID             *string               `json:"vector_id"`
	EngineID             *string               `json:"engine_id"`
	PackageID            string                `json:"package_id"`
	Version              string                `json:"version"`
	InstallScope         string                `json:"install_scope"`
	PrefixMode           string                `json:"prefix_mode"`
	Files                []ReceiptFile         `json:"files"`
	EntryPoints          []ReceiptEntryPoint   `json:"entry_points"`
	ProvidesCapabilities []string              `json:"provides_capabilities"`
	RequiresCapabilities []string              `json:"requires_capabilities"`
	CompatibleReceptors  []string              `json:"compatible_receptors"`
	PlatformRequirements []PlatformRequirement `json:"platform_requirements"`
	ReceiptDigest        string                `json:"receipt_digest"`
}

type ReceiptFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type ReceiptEntryPoint struct {
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

// Install atomically installs the executable and exact digest-bearing evidence.
// It creates no TOPS configuration, state root, socket, or ledger artifact.
func Install(source string, scope stavpaths.Scope, force bool) (Result, error) {
	layout, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return Result{}, err
	}
	qualifiedDigest, err := requireInvokingExecutable(source)
	if err != nil {
		return Result{}, err
	}
	return installAt(source, layout, force, qualifiedDigest)
}

func InstallAt(source string, scope stavpaths.Scope, prefix, requestedVersion string) (Result, error) {
	layout, err := stavpaths.ResolveInstallAt(scope, prefix, requestedVersion)
	if err != nil {
		return Result{}, err
	}
	qualifiedDigest, err := requireInvokingExecutable(source)
	if err != nil {
		return Result{}, err
	}
	return installAt(source, layout, false, qualifiedDigest)
}

// A fresh guard may only describe this retirement-aware owner build. Older or
// arbitrary copied binaries cannot acquire managed capability through adoption.
func requireInvokingExecutable(source string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	_, want, err := sourceFile(self)
	if err != nil {
		return "", err
	}
	_, got, err := sourceFile(source)
	if err != nil || got != want {
		return "", fmt.Errorf("STAV managed install requires the invoking retirement-aware executable")
	}
	return want, nil
}

func installAt(source string, layout stavpaths.InstallLayout, force bool, qualifiedDigest string) (Result, error) {
	scope := layout.Scope
	source, sourceDigest, err := sourceFile(source)
	if err != nil {
		return Result{}, err
	}
	if qualifiedDigest != "" && sourceDigest != qualifiedDigest {
		return Result{}, fmt.Errorf("STAV bytes differ from the invoking retirement-aware executable")
	}
	if err := ensureDirectory(layout.Prefix); err != nil {
		return Result{}, err
	}
	existingDigest, exists, err := regularFileDigest(layout.Binary)
	if err != nil {
		return Result{}, fmt.Errorf("inspect installed binary: %w", err)
	}
	if exists && existingDigest != sourceDigest {
		return Result{}, fmt.Errorf("immutable STAV package version already exists with different bytes; install a distinct version")
	}
	info, err := os.Stat(source)
	if err != nil {
		return Result{}, err
	}
	relativeBinary, err := filepath.Rel(layout.Prefix, layout.Binary)
	if err != nil {
		return Result{}, err
	}
	platformOS := runtime.GOOS
	if platformOS == "darwin" {
		platformOS = "macos"
	}
	receipt := InstallReceipt{Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2, ComponentID: "stav-append-authority", ComponentKind: "service", ModuleID: "stav-append-authority", PackageID: "stav-append-authority", Version: version.Version, InstallScope: "prefix", PrefixMode: "installation_prefix", Files: []ReceiptFile{{Path: filepath.ToSlash(relativeBinary), Kind: "executable", Size: info.Size(), Digest: "sha256:" + sourceDigest}}, EntryPoints: []ReceiptEntryPoint{{EntryPointID: "stav.append-authority", Kind: "executable", Path: filepath.ToSlash(relativeBinary), Protocols: []string{"symphony.stav.local.v1"}}, {EntryPointID: "stav.foundation-lifecycle", Kind: "adapter", Path: filepath.ToSlash(relativeBinary), Protocols: []string{"symphony.foundation.lifecycle-command.v1"}}}, ProvidesCapabilities: []string{"symphony.foundation.lifecycle-adapter.v1", "symphony.stav.append-authority.v1", "symphony.package-retirement.v1"}, RequiresCapabilities: []string{}, CompatibleReceptors: []string{}, PlatformRequirements: []PlatformRequirement{{OS: platformOS, Architecture: runtime.GOARCH, Critical: true}}}
	receipt.ReceiptDigest, err = canonicalDigestWithout(receipt, "receipt_digest")
	if err != nil {
		return Result{}, err
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return Result{}, err
	}
	receiptBytes := append(data, '\n')
	_, manifestErr := os.Lstat(layout.Manifest)
	if manifestErr != nil && !os.IsNotExist(manifestErr) {
		return Result{}, manifestErr
	}
	identity := retirementIdentity(layout, receipt.ReceiptDigest, "sha256:"+sourceDigest, uint64(info.Size()))
	guard, err := packageretirement.PrepareInstall(layout.Prefix, identity, !exists && os.IsNotExist(manifestErr))
	if err != nil {
		return Result{}, err
	}
	defer guard.Close()
	if guard.State() == "open" {
		record, evidenceDigest, err := verifyInstallLayout(layout)
		if err != nil {
			return Result{}, err
		}
		if err := guard.VerifyFile(identity.BinaryRelative, identity.BinaryDigest, identity.BinarySize); err != nil {
			return Result{}, err
		}
		if err := guard.VerifyFile(identity.ReceiptRelative, digestBytes(receiptBytes), uint64(len(receiptBytes))); err != nil {
			return Result{}, err
		}
		return installResult(layout, record, evidenceDigest, false), nil
	}
	payload, err := os.Open(source)
	if err != nil {
		return Result{}, err
	}
	_, publishErr := guard.PublishFile(identity.BinaryRelative, payload, identity.BinaryDigest, identity.BinarySize, 0755)
	closeErr := payload.Close()
	if publishErr != nil {
		return Result{}, publishErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	if _, err := guard.PublishFile(identity.ReceiptRelative, bytes.NewReader(receiptBytes), digestBytes(receiptBytes), uint64(len(receiptBytes)), 0644); err != nil {
		return Result{}, err
	}
	if err := activateManagedPackage(guard, identity, receiptBytes); err != nil {
		return Result{}, err
	}
	evidenceDigest := receipt.ReceiptDigest
	record := InstallRecord{Schema: receipt.Protocol, Scope: scope, Binary: layout.Binary, BinarySHA256: sourceDigest, ReceiptDigest: receipt.ReceiptDigest}
	return installResult(layout, record, evidenceDigest, true), nil
}

// Activation must re-observe both live paths after publication. A directory or
// file replacement between the two commits leaves installation closed.
func activateManagedPackage(guard *packageretirement.Lease, identity packageretirement.Identity, receiptBytes []byte) error {
	if err := guard.VerifyFile(identity.BinaryRelative, identity.BinaryDigest, identity.BinarySize); err != nil {
		return err
	}
	if err := guard.VerifyFile(identity.ReceiptRelative, digestBytes(receiptBytes), uint64(len(receiptBytes))); err != nil {
		return err
	}
	return guard.Activate()
}

// Uninstall closes managed start admission, proves process quiescence, and removes
// only exact receipt-owned bytes. Force never bypasses ownership or retirement.
func Uninstall(source string, scope stavpaths.Scope, force bool) (Result, error) {
	layout, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return Result{}, err
	}
	return uninstallAt(source, layout, force)
}

func UninstallAt(source string, scope stavpaths.Scope, prefix, requestedVersion string) (Result, error) {
	layout, err := stavpaths.ResolveInstallAt(scope, prefix, requestedVersion)
	if err != nil {
		return Result{}, err
	}
	return uninstallAt(source, layout, false)
}

func retirementIdentity(layout stavpaths.InstallLayout, receiptDigest, binaryDigest string, size uint64) packageretirement.Identity {
	binary, _ := filepath.Rel(layout.Prefix, layout.Binary)
	receipt, _ := filepath.Rel(layout.Prefix, layout.Manifest)
	return packageretirement.Identity{Owner: packageuse.Owner, Version: version.Version, ReceiptDigest: receiptDigest, BinaryDigest: binaryDigest, BinaryRelative: filepath.ToSlash(binary), ReceiptRelative: filepath.ToSlash(receipt), BinarySize: size}
}

func uninstallAt(source string, layout stavpaths.InstallLayout, force bool) (Result, error) {
	if layout.Scope != stavpaths.ScopeUser {
		return Result{}, fmt.Errorf("STAV system or cross-user retirement is not qualified")
	}
	identity, state, err := packageretirement.Inspect(layout.Prefix, packageuse.Owner, version.Version)
	if err != nil {
		return Result{}, fmt.Errorf("STAV removal requires a managed retirement guard; --force cannot bypass it: %w", err)
	}
	want := retirementIdentity(layout, identity.ReceiptDigest, identity.BinaryDigest, identity.BinarySize)
	if identity != want {
		return Result{}, fmt.Errorf("STAV retirement guard does not own the selected installation")
	}
	_, sourceDigest, err := sourceFile(source)
	if err != nil || "sha256:"+sourceDigest != identity.BinaryDigest {
		return Result{}, fmt.Errorf("STAV removal requires the exact invoking executable; --force cannot bypass it")
	}
	controller, err := os.Executable()
	if err != nil {
		return Result{}, err
	}
	controllerInfo, controllerErr := os.Stat(controller)
	targetInfo, targetErr := os.Stat(layout.Binary)
	if filepath.Clean(controller) == layout.Binary || (controllerErr == nil && targetErr == nil && os.SameFile(controllerInfo, targetInfo)) {
		return Result{}, fmt.Errorf("STAV retirement cannot unlink its executing controller; use the retained exact owner from outside the target installation")
	}
	guard, err := packageretirement.BeginRetirement(layout.Prefix, identity)
	if err != nil {
		return Result{}, err
	}
	defer guard.Close()
	if reference, err := FindInstallReference(layout.Scope, layout.Binary); err != nil {
		return Result{}, err
	} else if reference != "" {
		return Result{}, fmt.Errorf("installed STAV package is still referenced by %s; retirement remains pending", reference)
	}
	state = guard.State()
	data, receiptErr := guard.ReadFile(identity.ReceiptRelative, 1024*1024)
	if receiptErr != nil && !os.IsNotExist(receiptErr) {
		return Result{}, receiptErr
	}
	if state == "retired" {
		if _, err := os.Lstat(layout.Binary); !os.IsNotExist(err) || !os.IsNotExist(receiptErr) {
			return Result{}, fmt.Errorf("retired STAV package contains unexpected payload")
		}
		return Result{Scope: layout.Scope, Binary: layout.Binary, InstallEvidence: layout.Manifest}, nil
	}
	if receiptErr == nil {
		receipt, err := decodeInstallReceipt(layout, data)
		if err != nil {
			return Result{}, err
		}
		if retirementIdentity(layout, receipt.ReceiptDigest, receipt.Files[0].Digest, uint64(receipt.Files[0].Size)) != identity {
			return Result{}, fmt.Errorf("STAV receipt differs from retirement identity")
		}
		if err := guard.VerifyFile(identity.ReceiptRelative, digestBytes(data), uint64(len(data))); err != nil {
			return Result{}, err
		}
		if err := guard.RemoveFile(identity.BinaryRelative, identity.BinaryDigest, identity.BinarySize); err != nil {
			return Result{}, err
		}
		if err := guard.RemoveFile(identity.ReceiptRelative, digestBytes(data), uint64(len(data))); err != nil {
			return Result{}, err
		}
	} else if _, err := os.Lstat(layout.Binary); !os.IsNotExist(err) {
		return Result{}, fmt.Errorf("STAV receipt absent but payload still exists; recovery requires review")
	}
	if err := guard.FinishRetirement(); err != nil {
		return Result{}, err
	}
	return Result{Scope: layout.Scope, Binary: layout.Binary, BinaryDigest: identity.BinaryDigest, InstallEvidence: layout.Manifest, Changed: true}, nil
}

func installResult(layout stavpaths.InstallLayout, record InstallRecord, evidenceDigest string, changed bool) Result {
	return Result{Scope: layout.Scope, Binary: layout.Binary, BinaryDigest: "sha256:" + record.BinarySHA256, InstallEvidence: layout.Manifest, InstallEvidenceDigest: evidenceDigest, Changed: changed}
}

// VerifyInstalled binds the exact regular executable to its exact installation evidence.
func VerifyInstalled(scope stavpaths.Scope) (InstallRecord, string, error) {
	layout, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return InstallRecord{}, "", err
	}
	return verifyInstallLayout(layout)
}

func VerifyExecutable(executable string, scope stavpaths.Scope) (stavpaths.InstallLayout, InstallRecord, string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(executable))
	if err != nil {
		return stavpaths.InstallLayout{}, InstallRecord{}, "", err
	}
	suffix := filepath.Join("libexec", "symphony", "stav-append-authority", version.Version, stavpaths.BinaryName)
	if !strings.HasSuffix(resolved, string(filepath.Separator)+suffix) {
		return stavpaths.InstallLayout{}, InstallRecord{}, "", fmt.Errorf("executable is not an immutable STAV package entry point")
	}
	prefix := strings.TrimSuffix(resolved, string(filepath.Separator)+suffix)
	layout, err := stavpaths.ResolveInstallAt(scope, prefix, version.Version)
	if err != nil || layout.Binary != resolved {
		return stavpaths.InstallLayout{}, InstallRecord{}, "", fmt.Errorf("executable package path is invalid")
	}
	record, digest, err := verifyInstallLayout(layout)
	return layout, record, digest, err
}

func verifyInstallLayout(layout stavpaths.InstallLayout) (InstallRecord, string, error) {
	scope := layout.Scope
	data, err := os.ReadFile(layout.Manifest)
	if err != nil {
		return InstallRecord{}, "", fmt.Errorf("read STAV installation evidence: %w", err)
	}
	info, err := os.Lstat(layout.Manifest)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return InstallRecord{}, "", fmt.Errorf("STAV installation evidence is unsafe")
	}
	receipt, err := decodeInstallReceipt(layout, data)
	if err != nil {
		return InstallRecord{}, "", err
	}
	binarySHA := strings.TrimPrefix(receipt.Files[0].Digest, "sha256:")
	digest, exists, err := regularFileDigest(layout.Binary)
	if err != nil || !exists || digest != binarySHA {
		return InstallRecord{}, "", fmt.Errorf("installed STAV binary is missing or differs from its evidence")
	}
	info, err = os.Stat(layout.Binary)
	if err != nil || info.Size() != receipt.Files[0].Size {
		return InstallRecord{}, "", fmt.Errorf("installed STAV binary size differs from its receipt")
	}
	record := InstallRecord{Schema: receipt.Protocol, Scope: scope, Binary: layout.Binary, BinarySHA256: binarySHA, ReceiptDigest: receipt.ReceiptDigest}
	return record, receipt.ReceiptDigest, nil
}

func decodeInstallReceipt(layout stavpaths.InstallLayout, data []byte) (InstallReceipt, error) {
	var receipt InstallReceipt
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return InstallReceipt{}, fmt.Errorf("decode STAV installation evidence: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return InstallReceipt{}, fmt.Errorf("STAV installation evidence has trailing JSON")
	}
	wantReceiptDigest, err := canonicalDigestWithout(receipt, "receipt_digest")
	if err != nil {
		return InstallReceipt{}, fmt.Errorf("canonicalize STAV installation evidence: %w", err)
	}
	if receipt.Protocol != "symphony.knowledge.install-receipt.v2" || receipt.FormatVersion != 2 || receipt.ComponentID != "stav-append-authority" || receipt.ComponentKind != "service" || receipt.ModuleID != "stav-append-authority" || receipt.VectorID != nil || receipt.EngineID != nil || receipt.PackageID != "stav-append-authority" || receipt.Version != version.Version || receipt.InstallScope != "prefix" || receipt.PrefixMode != "installation_prefix" || receipt.ReceiptDigest != wantReceiptDigest || len(receipt.Files) != 1 || len(receipt.EntryPoints) != 2 || len(receipt.RequiresCapabilities) != 0 || len(receipt.CompatibleReceptors) != 0 || len(receipt.PlatformRequirements) != 1 {
		return InstallReceipt{}, fmt.Errorf("STAV installation evidence does not match selected scope")
	}
	relativeBinary, err := filepath.Rel(layout.Prefix, layout.Binary)
	if err != nil || receipt.Files[0].Path != filepath.ToSlash(relativeBinary) || receipt.Files[0].Kind != "executable" || receipt.Files[0].Size < 0 || len(receipt.Files[0].Digest) != len("sha256:")+sha256.Size*2 {
		return InstallReceipt{}, fmt.Errorf("STAV receipt does not own the exact executable")
	}
	service, adapter := receipt.EntryPoints[0], receipt.EntryPoints[1]
	if service.EntryPointID != "stav.append-authority" || service.Kind != "executable" || service.Path != receipt.Files[0].Path || len(service.Protocols) != 1 || service.Protocols[0] != "symphony.stav.local.v1" || adapter.EntryPointID != "stav.foundation-lifecycle" || adapter.Kind != "adapter" || adapter.Path != receipt.Files[0].Path || len(adapter.Protocols) != 1 || adapter.Protocols[0] != CommandProtocolName || !exactStrings(receipt.ProvidesCapabilities, []string{"symphony.foundation.lifecycle-adapter.v1", "symphony.stav.append-authority.v1", "symphony.package-retirement.v1"}) {
		return InstallReceipt{}, fmt.Errorf("STAV receipt entry points or capabilities are invalid")
	}
	wantOS := runtime.GOOS
	if wantOS == "darwin" {
		wantOS = "macos"
	}
	platform := receipt.PlatformRequirements[0]
	if platform.OS != wantOS || platform.Architecture != runtime.GOARCH || platform.KernelABI != nil || !platform.Critical {
		return InstallReceipt{}, fmt.Errorf("STAV receipt platform requirement does not match this adapter")
	}
	binarySHA := receipt.Files[0].Digest[len("sha256:"):]
	if _, err := hex.DecodeString(binarySHA); err != nil {
		return InstallReceipt{}, fmt.Errorf("STAV installation evidence has invalid binary digest")
	}
	return receipt, nil
}

// InspectLegacyInstalled dual-reads historical install-v1 evidence without
// rewriting or granting it current adapter compatibility.
func InspectLegacyInstalled(scope stavpaths.Scope) (InstallRecord, string, error) {
	layout, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return InstallRecord{}, "", err
	}
	data, err := os.ReadFile(layout.LegacyManifest)
	if err != nil {
		return InstallRecord{}, "", err
	}
	info, err := os.Lstat(layout.LegacyManifest)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return InstallRecord{}, "", fmt.Errorf("legacy STAV installation evidence is unsafe")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var legacy legacyInstallRecord
	if err := decoder.Decode(&legacy); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return InstallRecord{}, "", fmt.Errorf("decode legacy STAV installation evidence")
	}
	if legacy.Schema != "symphony.stav.install.v1" || legacy.Scope != scope || legacy.Binary != layout.LegacyBinary || len(legacy.BinarySHA256) != sha256.Size*2 {
		return InstallRecord{}, "", fmt.Errorf("legacy STAV installation evidence does not match selected scope")
	}
	digest, exists, err := regularFileDigest(layout.LegacyBinary)
	if err != nil || !exists || digest != legacy.BinarySHA256 {
		return InstallRecord{}, "", fmt.Errorf("legacy STAV binary differs from its evidence")
	}
	return InstallRecord{Schema: legacy.Schema, Scope: scope, Binary: legacy.Binary, BinarySHA256: legacy.BinarySHA256}, digestBytes(data), nil
}

// FindInstallReference conservatively rejects package removal while any valid
// per-TOPS enrollment record or native supervisor descriptor references it.
func FindInstallReference(scope stavpaths.Scope, packageBinary string) (string, error) {
	base, err := instanceConfigBase(scope)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(base)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect STAV enrollment references: %w", err)
	}
	for _, entry := range entries {
		if stavpaths.ValidateTOPSID(entry.Name()) != nil {
			continue
		}
		if !entry.IsDir() {
			return "", fmt.Errorf("unsafe STAV reference namespace %s", entry.Name())
		}
		layout, layoutErr := stavpaths.ResolveInstance(scope, entry.Name())
		if layoutErr != nil {
			continue
		}
		if _, statErr := os.Lstat(layout.ActiveAttempt); statErr == nil {
			return layout.ActiveAttempt, nil
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}
		for _, retained := range []string{layout.EnrollmentFile} {
			if _, err := os.Lstat(retained); err == nil {
				return retained, nil
			} else if !os.IsNotExist(err) {
				return "", err
			}
		}

	}
	stateBase, err := instanceStateBase(scope)
	if err != nil {
		return "", err
	}
	stateEntries, stateErr := os.ReadDir(stateBase)
	if stateErr != nil && !os.IsNotExist(stateErr) {
		return "", fmt.Errorf("inspect STAV lifecycle references: %w", stateErr)
	}
	for _, entry := range stateEntries {
		if stavpaths.ValidateTOPSID(entry.Name()) != nil {
			continue
		}
		if !entry.IsDir() {
			return "", fmt.Errorf("unsafe STAV reference namespace %s", entry.Name())
		}
		layout, layoutErr := stavpaths.ResolveInstance(scope, entry.Name())
		if layoutErr != nil {
			continue
		}
		if _, statErr := os.Lstat(layout.ActiveAttempt); statErr == nil {
			return layout.ActiveAttempt, nil
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}
	}
	return findSupervisorReference(packageBinary)
}

func findSupervisorReference(binary string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configBase := os.Getenv("XDG_CONFIG_HOME")
	if configBase == "" {
		configBase = filepath.Join(home, ".config")
	}
	for _, directory := range []string{filepath.Join(home, "Library", "LaunchAgents"), filepath.Join(configBase, "systemd", "user")} {
		entries, err := os.ReadDir(directory)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), "io.github.quanux.symphony.stav.") && !strings.HasPrefix(entry.Name(), "symphony-stav@") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() {
				return "", fmt.Errorf("unsafe STAV supervisor reference %s", path)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&#34;", "'", "&#39;").Replace(binary)
			if bytes.Contains(data, []byte(binary)) || bytes.Contains(data, []byte(escaped)) {
				return path, nil
			}
		}
	}
	return "", nil
}

func instanceConfigBase(scope stavpaths.Scope) (string, error) {
	if scope == stavpaths.ScopeSystem {
		return "/etc/symphony", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "symphony"), nil
}

func instanceStateBase(scope stavpaths.Scope) (string, error) {
	if scope == stavpaths.ScopeSystem {
		return "/var/lib/symphony", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "symphony"), nil
}

func canonicalNow() string {
	return time.Now().UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}

const CommandProtocolName = "symphony.foundation.lifecycle-command.v1"

func canonicalDigestWithout(value any, keys ...string) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return "", err
	}
	for _, key := range keys {
		delete(object, key)
	}
	data, err = json.Marshal(object)
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func exactStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sourceFile(path string) (string, string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", "", fmt.Errorf("resolve source executable: %w", err)
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", "", fmt.Errorf("inspect source executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("source executable is not a regular file")
	}
	digest, err := fileDigest(resolved)
	if err != nil {
		return "", "", err
	}
	return resolved, digest, nil
}

func regularFileDigest(path string) (string, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("path is not a regular file: %s", path)
	}
	digest, err := fileDigest(path)
	return digest, true, err
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ensureDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || path == string(filepath.Separator) {
		return fmt.Errorf("unsafe installation directory %q", path)
	}

	var missing []string
	current := path
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("installation path component is not a directory: %s", current)
			}
			break
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect installation directory %s: %w", current, err)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("no existing parent for installation directory %s", path)
		}
		current = parent
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0755); err != nil && !os.IsExist(err) {
			return fmt.Errorf("create installation directory %s: %w", missing[i], err)
		}
		info, err := os.Lstat(missing[i])
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("installation directory is unsafe: %s", missing[i])
		}
	}
	return nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open installation directory for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync installation directory: %w", err)
	}
	return nil
}

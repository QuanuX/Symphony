package packageinstall

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
	"slices"
	"sort"
	"strings"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/enrollment"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/packageretirement"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/version"
)

const (
	receiptProtocol = "symphony.knowledge.install-receipt.v2"
	ComponentID     = "accordare-stav-producer"
	componentID     = ComponentID
	binaryName      = "symphony-accordare-stav-producer"
)

type receiptFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   uint64 `json:"size"`
	Digest string `json:"digest"`
}
type entryPoint struct {
	EntryPointID string   `json:"entry_point_id"`
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	Protocols    []string `json:"protocols"`
}
type platformRequirement struct {
	OS           string  `json:"os"`
	Architecture string  `json:"architecture"`
	KernelABI    *string `json:"kernel_abi"`
	Critical     bool    `json:"critical"`
}
type receipt struct {
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
	Files                []receiptFile         `json:"files"`
	EntryPoints          []entryPoint          `json:"entry_points"`
	ProvidesCapabilities []string              `json:"provides_capabilities"`
	RequiresCapabilities []string              `json:"requires_capabilities"`
	CompatibleReceptors  []string              `json:"compatible_receptors"`
	PlatformRequirements []platformRequirement `json:"platform_requirements"`
	ReceiptDigest        string                `json:"receipt_digest"`
}

type Result struct {
	Prefix, Version, Binary, Receipt, ReceiptDigest string
	Changed                                         bool
}

type layout struct{ prefix, binary, receipt string }

func Inspect(prefix, requestedVersion string) (Result, error) {
	paths, err := resolve(prefix, requestedVersion)
	if err != nil {
		return Result{}, err
	}
	record, present, err := readReceipt(paths)
	if err != nil {
		return Result{}, err
	}
	if !present {
		return Result{}, fmt.Errorf("Accordare producer installation receipt is absent")
	}
	if err := validateReceipt(record, paths); err != nil {
		return Result{}, err
	}
	return result(paths, record.ReceiptDigest, false), nil
}

func Install(source, prefix, requestedVersion string) (Result, error) {
	return InstallWithDelegation(source, prefix, requestedVersion, nil)
}

// InstallWithDelegation retains the parent lifecycle root lock and independently validates its exact ownership claim.
func InstallWithDelegation(source, prefix, requestedVersion string, delegation *packageretirement.Delegation) (Result, error) {
	paths, err := resolve(prefix, requestedVersion)
	if err != nil {
		return Result{}, err
	}
	// Only this participating installer may establish fresh custody. Copying an
	// arbitrary legacy executable would manufacture a lease promise it cannot keep.
	executable, err := os.Executable()
	if err != nil {
		return Result{}, err
	}
	sourceUse, err := acquireInstallSourceUse(executable, paths.binary)
	if err != nil {
		return Result{}, err
	}
	defer sourceUse.Close()
	sourceInfo, sourceErr := os.Stat(source)
	executableInfo, executableErr := os.Stat(executable)
	if sourceErr != nil || executableErr != nil || !os.SameFile(sourceInfo, executableInfo) {
		return Result{}, fmt.Errorf("Accordare installer must copy its own participating executable")
	}
	input, err := openInstallerSource(source)
	if err != nil {
		return Result{}, err
	}
	defer input.Close()
	opened, err := input.Stat()
	if err != nil || !os.SameFile(opened, executableInfo) {
		return Result{}, fmt.Errorf("installer source identity changed")
	}
	size := uint64(opened.Size())
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return Result{}, err
	}
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		return Result{}, err
	}
	record, err := makeReceipt(paths, size, digest)
	if err != nil {
		return Result{}, err
	}
	existing, present, err := readReceipt(paths)
	if err != nil {
		return Result{}, err
	}
	if present {
		if err := validateReceiptIdentity(existing, paths); err != nil {
			return Result{}, err
		}
		if existing.ReceiptDigest != record.ReceiptDigest {
			return Result{}, fmt.Errorf("immutable Accordare producer version already contains different bytes")
		}
	}
	if err := ensureDirectory(paths.prefix); err != nil {
		return Result{}, err
	}
	_, binaryErr := os.Lstat(paths.binary)
	allowCreate := !present && os.IsNotExist(binaryErr)
	lease, err := packageretirement.PrepareInstall(paths.prefix, retirementIdentity(paths, record), allowCreate, delegation)
	if err != nil {
		return Result{}, err
	}
	defer lease.Close()
	if lease.State() == "open" && !present {
		return Result{}, fmt.Errorf("open installation receipt is absent")
	}
	id := retirementIdentity(paths, record)
	binaryChanged, err := lease.PublishFile(id.BinaryRelative, input, digest, size, 0o755)
	if err != nil {
		return Result{}, err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return Result{}, err
	}
	data = append(data, '\n')
	receiptChanged, err := lease.PublishFile(id.ReceiptRelative, bytes.NewReader(data), bytesDigest(data), uint64(len(data)), 0o644)
	if err != nil {
		return Result{}, err
	}
	changed := binaryChanged || receiptChanged || lease.State() == "installing"
	if err := activateInstallation(lease, id, data); err != nil {
		return Result{}, err
	}
	return result(paths, record.ReceiptDigest, changed), nil
}

// A same-package reinstall acquires exclusive target custody in PrepareInstall.
// Taking a shared source lease first would conflict with that lock. A different
// installed source still remains pinned throughout the target installation.
func acquireInstallSourceUse(executable, target string) (*packageretirement.Lease, error) {
	if filepath.Clean(executable) == target {
		return nil, nil
	}
	return acquireControllerUse(executable)
}

// Receipt publication can outlive a replacement of the executable's parent.
// Verify both final files against this pinned root immediately before opening
// admission, including an existing installation or interrupted-install retry.
func activateInstallation(lease *packageretirement.Lease, id packageretirement.Identity, receiptBytes []byte) error {
	if err := lease.VerifyFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		return err
	}
	if err := lease.VerifyFile(id.ReceiptRelative, bytesDigest(receiptBytes), uint64(len(receiptBytes))); err != nil {
		return err
	}
	return lease.Activate()
}

func makeReceipt(paths layout, size uint64, digest string) (receipt, error) {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	relative, _ := filepath.Rel(paths.prefix, paths.binary)
	record := receipt{
		Protocol: receiptProtocol, FormatVersion: 2, ComponentID: componentID, ComponentKind: "service", ModuleID: componentID,
		PackageID: componentID, Version: version.Version, InstallScope: "prefix", PrefixMode: "installation_prefix",
		Files:                []receiptFile{{Path: filepath.ToSlash(relative), Kind: "executable", Size: size, Digest: digest}},
		EntryPoints:          []entryPoint{{EntryPointID: "accordare.stav-producer", Kind: "executable", Path: filepath.ToSlash(relative), Protocols: []string{"symphony.accordare.stav-producer.local.v1", "symphony.accordare.stav-producer.supervisor.v1"}}},
		ProvidesCapabilities: []string{"symphony.accordare.stav-producer.v1", "symphony.accordare.stav-producer.native-supervision.v1", packageretirement.Capability, packageretirement.DispatchCapability}, RequiresCapabilities: []string{"symphony.stav.append-authority.v1", "symphony.ssiag.authorization.v1"},
		CompatibleReceptors: []string{}, PlatformRequirements: []platformRequirement{{OS: osName, Architecture: runtime.GOARCH, Critical: true}},
	}
	var err error
	record.ReceiptDigest, err = receiptDigest(record)
	return record, err
}

func Uninstall(prefix, requestedVersion string) (Result, error) {
	return UninstallWithDelegation(prefix, requestedVersion, nil)
}

// UninstallWithDelegation preserves all native retirement and reference gates.
func UninstallWithDelegation(prefix, requestedVersion string, delegation *packageretirement.Delegation) (Result, error) {
	if os.Geteuid() == 0 {
		return Result{}, fmt.Errorf("Accordare system and cross-user reclamation remains unqualified; native removal requires a user-owned installation")
	}
	paths, err := resolve(prefix, requestedVersion)
	if err != nil {
		return Result{}, err
	}
	record, present, err := readReceipt(paths)
	if err != nil {
		return Result{}, err
	}
	identity, phase, err := packageretirement.Inspect(paths.prefix, componentID, requestedVersion)
	if err != nil {
		return Result{}, fmt.Errorf("Accordare removal requires established native custody: %w", err)
	}
	self, err := os.Executable()
	if err != nil {
		return Result{}, err
	}
	selfSize, selfDigest, err := fileEvidence(self)
	if err != nil || selfSize != identity.BinarySize || selfDigest != identity.BinaryDigest {
		return Result{}, fmt.Errorf("Accordare removal requires the exact retained native owner executable")
	}
	if target, err := os.Stat(paths.binary); err == nil {
		if running, err := os.Stat(self); err == nil && os.SameFile(target, running) {
			return Result{}, fmt.Errorf("Accordare cannot reclaim its executing controller; use the exact retained owner outside this installation")
		}
	}
	if present {
		if err := validateReceiptIdentity(record, paths); err != nil {
			return Result{}, err
		}
		if retirementIdentity(paths, record) != identity {
			return Result{}, fmt.Errorf("Accordare receipt differs from retirement identity")
		}
	} else if phase != "retiring" && phase != "retired" {
		return Result{}, fmt.Errorf("Accordare receipt is absent without closed retirement evidence")
	}
	// Preserve the existing recovery endpoint when work is already known to be
	// pending. Repeat under exclusive ownership before any reclamation below.
	if phase == "open" {
		if err := requireNoOrphanPending(); err != nil {
			return Result{}, err
		}
	}
	lease, err := packageretirement.BeginRetirement(paths.prefix, identity, delegation)
	if err != nil {
		return Result{}, err
	}
	defer lease.Close()
	// Re-read after exclusive use ownership; inspection before admission is not
	// authority to reclaim bytes after another lifecycle operation has run.
	record, present, err = readGuardedReceipt(lease, identity.ReceiptRelative)
	if err != nil {
		return Result{}, err
	}
	if present {
		if err := validateReceiptIdentity(record, paths); err != nil {
			return Result{}, err
		}
		if retirementIdentity(paths, record) != identity {
			return Result{}, fmt.Errorf("Accordare receipt changed during retirement")
		}
	}
	if lease.State() == "retired" {
		if present {
			return Result{}, fmt.Errorf("retired Accordare installation unexpectedly contains a receipt")
		}
		if _, err := os.Lstat(paths.binary); !os.IsNotExist(err) {
			return Result{}, fmt.Errorf("retired Accordare installation unexpectedly contains package bytes")
		}
		return result(paths, identity.ReceiptDigest, false), nil
	}
	references, err := enrollmentReferences()
	if err != nil {
		return Result{}, err
	}
	if len(references) != 0 {
		return Result{}, fmt.Errorf("refusing to uninstall an enrolled Accordare producer package: %s", strings.Join(references, ", "))
	}
	if err := requireNoOrphanPending(); err != nil {
		return Result{}, err
	}
	if !present {
		// Receipt-last cleanup can crash after unlinking the receipt. Never use a
		// tombstone as permission to remove an unreceipted surviving executable.
		if _, err := os.Lstat(paths.binary); !os.IsNotExist(err) {
			return Result{}, fmt.Errorf("receiptless Accordare retirement has surviving or unavailable bytes")
		}
	} else {
		receiptBytes, err := lease.ReadFile(identity.ReceiptRelative, 1<<20)
		if err != nil {
			return Result{}, err
		}
		snapshot, _, err := decodeReceipt(receiptBytes)
		if err != nil {
			return Result{}, err
		}
		if err := validateReceiptIdentity(snapshot, paths); err != nil {
			return Result{}, err
		}
		if retirementIdentity(paths, snapshot) != identity {
			return Result{}, fmt.Errorf("Accordare retirement receipt changed before reclamation")
		}
		receiptSize := uint64(len(receiptBytes))
		receiptHash := bytesDigest(receiptBytes)
		if err := lease.VerifyFile(identity.ReceiptRelative, receiptHash, receiptSize); err != nil {
			return Result{}, err
		}
		if err := lease.RemoveFile(identity.BinaryRelative, identity.BinaryDigest, identity.BinarySize); err != nil {
			return Result{}, err
		}
		if err := lease.RemoveFile(identity.ReceiptRelative, receiptHash, receiptSize); err != nil {
			return Result{}, err
		}
	}
	if err := lease.FinishRetirement(); err != nil {
		return Result{}, err
	}
	return result(paths, identity.ReceiptDigest, true), nil
}

func enrollmentReferences() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configBase := os.Getenv("XDG_CONFIG_HOME")
	if configBase == "" {
		configBase = filepath.Join(home, ".config")
	}
	roots := []string{filepath.Join(configBase, "symphony")}
	if os.Geteuid() == 0 {
		roots = append(roots, "/etc/symphony")
	}
	references := make([]string, 0)
	for _, root := range roots {
		entries, readErr := os.ReadDir(root)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("inspect Accordare enrollment root: %w", readErr)
		}
		for _, entry := range entries {
			if stavprotocol.ValidateTOPSID(entry.Name()) != nil {
				continue
			}
			entryInfo, infoErr := entry.Info()
			if infoErr != nil || entry.Type()&os.ModeSymlink != 0 || !entryInfo.IsDir() {
				return nil, fmt.Errorf("Accordare enrollment root contains an unsafe TOPS entry")
			}
			path := filepath.Join(root, entry.Name(), componentID, "config.json")
			info, statErr := os.Lstat(path)
			if os.IsNotExist(statErr) {
				continue
			}
			if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
				return nil, fmt.Errorf("Accordare enrollment reference is unsafe")
			}
			references = append(references, "enrollment:"+entry.Name())
		}
	}
	sort.Strings(references)
	return references, nil
}

func resolve(prefix, requestedVersion string) (layout, error) {
	if requestedVersion != version.Version || !filepath.IsAbs(prefix) || filepath.Clean(prefix) == string(filepath.Separator) || strings.Contains(requestedVersion, string(filepath.Separator)) {
		return layout{}, fmt.Errorf("invalid Accordare producer installation identity")
	}
	prefix = filepath.Clean(prefix)
	return layout{prefix: prefix, binary: filepath.Join(prefix, "libexec", "symphony", componentID, version.Version, binaryName), receipt: filepath.Join(prefix, "share", "symphony", "receipts", componentID, version.Version, "install-receipt.json")}, nil
}

func readReceipt(paths layout) (receipt, bool, error) {
	info, err := os.Lstat(paths.receipt)
	if os.IsNotExist(err) {
		return receipt{}, false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
		return receipt{}, false, fmt.Errorf("Accordare producer receipt is unsafe")
	}
	data, err := os.ReadFile(paths.receipt)
	if err != nil {
		return receipt{}, false, err
	}
	return decodeReceipt(data)
}

func decodeReceipt(data []byte) (receipt, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record receipt
	if err := decoder.Decode(&record); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return receipt{}, false, fmt.Errorf("Accordare producer receipt is invalid")
	}
	return record, true, nil
}

func readGuardedReceipt(lease *packageretirement.Lease, relative string) (receipt, bool, error) {
	data, err := lease.ReadFile(relative, 1<<20)
	if os.IsNotExist(err) {
		return receipt{}, false, nil
	}
	if err != nil {
		return receipt{}, false, err
	}
	return decodeReceipt(data)
}

func bytesDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateReceipt(record receipt, paths layout) error {
	if err := validateReceiptIdentity(record, paths); err != nil {
		return err
	}
	size, digest, err := fileEvidence(paths.binary)
	if err != nil || size != record.Files[0].Size || digest != record.Files[0].Digest {
		return fmt.Errorf("Accordare producer binary differs from its receipt")
	}
	return nil
}

func validateReceiptIdentity(record receipt, paths layout) error {
	wantDigest, err := receiptDigest(record)
	if err != nil {
		return err
	}
	relative, _ := filepath.Rel(paths.prefix, paths.binary)
	wantOS := runtime.GOOS
	if wantOS == "darwin" {
		wantOS = "macos"
	}
	wantProvides := []string{"symphony.accordare.stav-producer.v1", "symphony.accordare.stav-producer.native-supervision.v1", packageretirement.Capability, packageretirement.DispatchCapability}
	wantRequires := []string{"symphony.stav.append-authority.v1", "symphony.ssiag.authorization.v1"}
	wantProtocols := []string{"symphony.accordare.stav-producer.local.v1", "symphony.accordare.stav-producer.supervisor.v1"}
	if record.Protocol != receiptProtocol || record.FormatVersion != 2 || record.ComponentID != componentID || record.ComponentKind != "service" || record.ModuleID != componentID || record.VectorID != nil || record.EngineID != nil || record.PackageID != componentID || record.Version != version.Version || record.InstallScope != "prefix" || record.PrefixMode != "installation_prefix" || record.ReceiptDigest != wantDigest || len(record.Files) != 1 || record.Files[0].Path != filepath.ToSlash(relative) || record.Files[0].Kind != "executable" || len(record.EntryPoints) != 1 || record.EntryPoints[0].EntryPointID != "accordare.stav-producer" || record.EntryPoints[0].Kind != "executable" || record.EntryPoints[0].Path != record.Files[0].Path || !slices.Equal(record.EntryPoints[0].Protocols, wantProtocols) || !slices.Equal(record.ProvidesCapabilities, wantProvides) || !slices.Equal(record.RequiresCapabilities, wantRequires) || record.CompatibleReceptors == nil || len(record.CompatibleReceptors) != 0 || len(record.PlatformRequirements) != 1 || record.PlatformRequirements[0].OS != wantOS || record.PlatformRequirements[0].Architecture != runtime.GOARCH || record.PlatformRequirements[0].KernelABI != nil || !record.PlatformRequirements[0].Critical {
		return fmt.Errorf("Accordare producer receipt identity is invalid")
	}
	return nil
}

func retirementIdentity(p layout, record receipt) packageretirement.Identity {
	receiptRelative, _ := filepath.Rel(p.prefix, p.receipt)
	return packageretirement.Identity{Owner: componentID, Version: version.Version, ReceiptDigest: record.ReceiptDigest, BinaryRelative: record.Files[0].Path, BinaryDigest: record.Files[0].Digest, BinarySize: record.Files[0].Size, ReceiptRelative: filepath.ToSlash(receiptRelative)}
}

// AcquireBinaryUse is the native owner's launch admission boundary. It accepts
// only the exact versioned executable and fresh, locally established custody.
func AcquireBinaryUse(binary string) (*packageretirement.Lease, error) {
	return acquireBinaryUse(binary, false)
}

func acquireBinaryUse(binary string, recovery bool) (*packageretirement.Lease, error) {
	p, record, err := installedBinary(binary)
	if err != nil {
		return nil, err
	}
	var lease *packageretirement.Lease
	if recovery {
		lease, err = packageretirement.AcquireRecoveryUse(p.prefix, retirementIdentity(p, record))
	} else {
		lease, err = packageretirement.AcquireUse(p.prefix, retirementIdentity(p, record))
	}
	if err != nil {
		return nil, err
	}
	if err := verifyGuardedReceipt(lease, p, record); err != nil {
		lease.Close()
		return nil, err
	}
	if err := lease.VerifyFile(record.Files[0].Path, record.Files[0].Digest, record.Files[0].Size); err != nil {
		lease.Close()
		return nil, err
	}
	return lease, nil
}

func AcquireCurrentUse() (*packageretirement.Lease, error) {
	binary, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return AcquireBinaryUse(binary)
}

// An installed controller remains a consumer during stop/status/recovery.
// External retained controllers do not consume this installation's executable.
func AcquireCurrentControllerUse() (*packageretirement.Lease, error) {
	binary, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return acquireControllerUse(binary)
}

func acquireControllerUse(binary string) (*packageretirement.Lease, error) {
	binary = filepath.Clean(binary)
	parts := strings.Split(filepath.ToSlash(binary), "/")
	if len(parts) < 6 || parts[len(parts)-5] != "libexec" || parts[len(parts)-4] != "symphony" || parts[len(parts)-3] != componentID || parts[len(parts)-1] != binaryName {
		return nil, nil
	}
	return acquireBinaryUse(binary, true)
}

// AcquireCurrentMaintenance serializes unenrollment with the full service
// lifetime and startup admission. Retiring packages remain closed throughout
// maintenance; open packages are not irreversibly retired by unenrollment.
func AcquireCurrentMaintenance() (*packageretirement.Lease, error) {
	binary, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return acquireBinaryMaintenance(binary)
}

func acquireBinaryMaintenance(binary string) (*packageretirement.Lease, error) {
	p, record, err := installedBinary(binary)
	if err != nil {
		return nil, err
	}
	id := retirementIdentity(p, record)
	guardID, phase, err := packageretirement.Inspect(p.prefix, componentID, version.Version)
	if err != nil || guardID != id {
		return nil, fmt.Errorf("Accordare maintenance requires matching package custody: %v", err)
	}
	var lease *packageretirement.Lease
	if phase == "open" {
		lease, err = packageretirement.PrepareInstall(p.prefix, id, false)
	} else if phase == "retiring" {
		lease, err = packageretirement.BeginRetirement(p.prefix, id)
	} else {
		return nil, fmt.Errorf("Accordare package is not available for maintenance")
	}
	if err != nil {
		return nil, err
	}
	if err := verifyGuardedReceipt(lease, p, record); err != nil {
		lease.Close()
		return nil, err
	}
	if err := lease.VerifyFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		lease.Close()
		return nil, err
	}
	return lease, nil
}

func verifyGuardedReceipt(lease *packageretirement.Lease, p layout, expected receipt) error {
	id := retirementIdentity(p, expected)
	actual, present, err := readGuardedReceipt(lease, id.ReceiptRelative)
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("Accordare installation receipt disappeared")
	}
	if err := validateReceiptIdentity(actual, p); err != nil {
		return err
	}
	if retirementIdentity(p, actual) != id {
		return fmt.Errorf("Accordare installation receipt changed")
	}
	return nil
}

func installedBinary(binary string) (layout, receipt, error) {
	if !filepath.IsAbs(binary) || filepath.Clean(binary) != binary {
		return layout{}, receipt{}, fmt.Errorf("Accordare executable must have an exact installed path")
	}
	prefix := binary
	for i := 0; i < 5; i++ {
		prefix = filepath.Dir(prefix)
	}
	p, err := resolve(prefix, version.Version)
	if err != nil || p.binary != binary {
		return layout{}, receipt{}, fmt.Errorf("Accordare executable is not a qualified native installation")
	}
	record, present, err := readReceipt(p)
	if err != nil {
		return layout{}, receipt{}, err
	}
	if !present {
		return layout{}, receipt{}, fmt.Errorf("Accordare installation receipt is absent")
	}
	if err := validateReceiptIdentity(record, p); err != nil {
		return layout{}, receipt{}, err
	}
	return p, record, nil
}

// Unenrollment can predate a crash. Inspect the owner's known state roots as
// well as retained configurations, so orphaned audit work cannot be discarded.
func requireNoOrphanPending() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(home, ".local", "state")
	}
	roots := []string{filepath.Join(base, "symphony")}
	if os.Geteuid() == 0 {
		roots = append(roots, "/var/lib/symphony")
	}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect Accordare state root: %w", err)
		}
		for _, entry := range entries {
			if stavprotocol.ValidateTOPSID(entry.Name()) != nil {
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
				return fmt.Errorf("Accordare state root contains an unsafe TOPS entry")
			}
			stateDir := filepath.Join(root, entry.Name(), componentID)
			if err := enrollment.RequireNoPending(paths.Layout{IntentDir: filepath.Join(stateDir, "intents-v1"), OutboxDir: filepath.Join(stateDir, "outbox-v1")}); err != nil {
				return err
			}
		}
	}
	return nil
}

func receiptDigest(record receipt) (string, error) {
	record.ReceiptDigest = ""
	raw, err := json.Marshal(record)
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

func fileEvidence(path string) (uint64, string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 {
		return 0, "", fmt.Errorf("package source is unavailable or unsafe")
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return 0, "", err
	}
	return uint64(info.Size()), "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func ensureDirectory(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("package directory is unsafe")
	}
	return nil
}

func result(paths layout, digest string, changed bool) Result {
	return Result{Prefix: paths.prefix, Version: version.Version, Binary: paths.binary, Receipt: paths.receipt, ReceiptDigest: digest, Changed: changed}
}

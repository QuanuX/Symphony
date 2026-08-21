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
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/version"
)

const (
	receiptProtocol = "symphony.knowledge.install-receipt.v2"
	componentID     = "accordare-stav-producer"
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
	paths, err := resolve(prefix, requestedVersion)
	if err != nil {
		return Result{}, err
	}
	size, digest, err := fileEvidence(source)
	if err != nil {
		return Result{}, err
	}
	if existing, present, err := readReceipt(paths); err != nil {
		return Result{}, err
	} else if present {
		if err := validateReceipt(existing, paths); err != nil {
			return Result{}, err
		}
		if existing.Files[0].Size != size || existing.Files[0].Digest != digest {
			return Result{}, fmt.Errorf("immutable Accordare producer version already contains different bytes")
		}
		return result(paths, existing.ReceiptDigest, false), nil
	}
	if err := ensureDirectory(filepath.Dir(paths.binary)); err != nil {
		return Result{}, err
	}
	if err := ensureDirectory(filepath.Dir(paths.receipt)); err != nil {
		return Result{}, err
	}
	if info, err := os.Lstat(paths.binary); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return Result{}, fmt.Errorf("unreceipted package path is unsafe")
		}
		existingSize, existingDigest, evidenceErr := fileEvidence(paths.binary)
		if evidenceErr != nil || existingSize != size || existingDigest != digest {
			return Result{}, fmt.Errorf("unreceipted package path contains different bytes")
		}
	} else if !os.IsNotExist(err) {
		return Result{}, err
	} else if err := copyAtomic(source, paths.binary); err != nil {
		return Result{}, err
	}
	installedSize, installedDigest, err := fileEvidence(paths.binary)
	if err != nil || installedSize != size || installedDigest != digest {
		return Result{}, fmt.Errorf("installed Accordare producer bytes differ from the verified package source")
	}
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
		ProvidesCapabilities: []string{"symphony.accordare.stav-producer.v1", "symphony.accordare.stav-producer.native-supervision.v1"}, RequiresCapabilities: []string{"symphony.stav.append-authority.v1", "symphony.ssiag.authorization.v1"},
		CompatibleReceptors: []string{}, PlatformRequirements: []platformRequirement{{OS: osName, Architecture: runtime.GOARCH, Critical: true}},
	}
	record.ReceiptDigest, err = receiptDigest(record)
	if err != nil {
		return Result{}, err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return Result{}, err
	}
	if err := writeAtomic(paths.receipt, append(data, '\n'), 0o600); err != nil {
		return Result{}, err
	}
	return result(paths, record.ReceiptDigest, true), nil
}

func Uninstall(prefix, requestedVersion string) (Result, error) {
	paths, err := resolve(prefix, requestedVersion)
	if err != nil {
		return Result{}, err
	}
	record, present, err := readReceipt(paths)
	if err != nil {
		return Result{}, err
	}
	if !present {
		return result(paths, "", false), nil
	}
	if err := validateReceipt(record, paths); err != nil {
		return Result{}, err
	}
	references, err := enrollmentReferences()
	if err != nil {
		return Result{}, err
	}
	if len(references) != 0 {
		return Result{}, fmt.Errorf("refusing to uninstall an enrolled Accordare producer package: %s", strings.Join(references, ", "))
	}
	if err := os.Remove(paths.binary); err != nil {
		return Result{}, err
	}
	if err := syncDirectory(filepath.Dir(paths.binary)); err != nil {
		return Result{}, err
	}
	if err := os.Remove(paths.receipt); err != nil {
		return Result{}, err
	}
	if err := syncDirectory(filepath.Dir(paths.receipt)); err != nil {
		return Result{}, err
	}
	return result(paths, record.ReceiptDigest, true), nil
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
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record receipt
	if err := decoder.Decode(&record); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return receipt{}, false, fmt.Errorf("Accordare producer receipt is invalid")
	}
	return record, true, nil
}

func validateReceipt(record receipt, paths layout) error {
	wantDigest, err := receiptDigest(record)
	if err != nil {
		return err
	}
	relative, _ := filepath.Rel(paths.prefix, paths.binary)
	wantOS := runtime.GOOS
	if wantOS == "darwin" {
		wantOS = "macos"
	}
	wantProvides := []string{"symphony.accordare.stav-producer.v1", "symphony.accordare.stav-producer.native-supervision.v1"}
	wantRequires := []string{"symphony.stav.append-authority.v1", "symphony.ssiag.authorization.v1"}
	wantProtocols := []string{"symphony.accordare.stav-producer.local.v1", "symphony.accordare.stav-producer.supervisor.v1"}
	if record.Protocol != receiptProtocol || record.FormatVersion != 2 || record.ComponentID != componentID || record.ComponentKind != "service" || record.ModuleID != componentID || record.VectorID != nil || record.EngineID != nil || record.PackageID != componentID || record.Version != version.Version || record.InstallScope != "prefix" || record.PrefixMode != "installation_prefix" || record.ReceiptDigest != wantDigest || len(record.Files) != 1 || record.Files[0].Path != filepath.ToSlash(relative) || record.Files[0].Kind != "executable" || len(record.EntryPoints) != 1 || record.EntryPoints[0].EntryPointID != "accordare.stav-producer" || record.EntryPoints[0].Kind != "executable" || record.EntryPoints[0].Path != record.Files[0].Path || !slices.Equal(record.EntryPoints[0].Protocols, wantProtocols) || !slices.Equal(record.ProvidesCapabilities, wantProvides) || !slices.Equal(record.RequiresCapabilities, wantRequires) || record.CompatibleReceptors == nil || len(record.CompatibleReceptors) != 0 || len(record.PlatformRequirements) != 1 || record.PlatformRequirements[0].OS != wantOS || record.PlatformRequirements[0].Architecture != runtime.GOARCH || record.PlatformRequirements[0].KernelABI != nil || !record.PlatformRequirements[0].Critical {
		return fmt.Errorf("Accordare producer receipt identity is invalid")
	}
	size, digest, err := fileEvidence(paths.binary)
	if err != nil || size != record.Files[0].Size || digest != record.Files[0].Digest {
		return fmt.Errorf("Accordare producer binary differs from its receipt")
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

func copyAtomic(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	temporary, err := os.CreateTemp(filepath.Dir(target), ".accordare-package-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o755); err != nil {
		temporary.Close()
		return err
	}
	if _, err := io.Copy(temporary, input); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, target); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(target))
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

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".accordare-receipt-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
func result(paths layout, digest string, changed bool) Result {
	return Result{Prefix: paths.prefix, Version: version.Version, Binary: paths.binary, Receipt: paths.receipt, ReceiptDigest: digest, Changed: changed}
}

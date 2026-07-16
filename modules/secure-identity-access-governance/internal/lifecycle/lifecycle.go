package lifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

type InstallRecord struct {
	Schema       string           `json:"schema"`
	Scope        ssiagpaths.Scope `json:"scope"`
	Version      string           `json:"version"`
	Binary       string           `json:"binary"`
	BinarySHA256 string           `json:"binary_sha256"`
}

type EnrollmentRecord struct {
	Schema     string           `json:"schema"`
	Scope      ssiagpaths.Scope `json:"scope"`
	TOPSID     string           `json:"tops_id"`
	TOPSName   string           `json:"tops_name"`
	ConfigFile string           `json:"config_file"`
	StateDir   string           `json:"state_dir"`
	Socket     string           `json:"socket"`
}

func Install(source string, scope ssiagpaths.Scope, force bool) (InstallRecord, error) {
	layout, err := ssiagpaths.ResolveInstall(scope)
	if err != nil {
		return InstallRecord{}, err
	}
	source = filepath.Clean(source)
	info, err := os.Stat(source)
	if err != nil {
		return InstallRecord{}, fmt.Errorf("inspect source executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return InstallRecord{}, fmt.Errorf("source executable is not a regular file")
	}
	if err := ensureDirectories(filepath.Dir(layout.Binary), layout.StateDir); err != nil {
		return InstallRecord{}, err
	}
	if err := requireAbsentOrRegular(layout.InstallManifest, "installation manifest"); err != nil {
		return InstallRecord{}, err
	}
	staged, sourceDigest, err := stageExecutable(source, layout.Binary, 0755)
	if err != nil {
		return InstallRecord{}, err
	}
	defer os.Remove(staged)

	existingDigest, exists, err := regularFileDigest(layout.Binary)
	if err != nil {
		return InstallRecord{}, fmt.Errorf("inspect installed binary: %w", err)
	}
	if exists && existingDigest != sourceDigest && !force {
		return InstallRecord{}, fmt.Errorf("installed binary differs; use --force to replace it")
	}
	if !exists || existingDigest != sourceDigest {
		if err := activateExecutable(staged, layout.Binary); err != nil {
			return InstallRecord{}, err
		}
	}

	record := InstallRecord{
		Schema:       "symphony.ssiag.install.v1",
		Scope:        scope,
		Version:      version.Version,
		Binary:       layout.Binary,
		BinarySHA256: sourceDigest,
	}
	if err := writeJSONAtomic(layout.InstallManifest, record); err != nil {
		return InstallRecord{}, err
	}
	return record, nil
}

// Uninstall removes only the host-level binary and its manifest. Per-TOPS
// configuration and state are intentionally outside this operation.
func Uninstall(scope ssiagpaths.Scope, force bool) (InstallRecord, error) {
	layout, err := ssiagpaths.ResolveInstall(scope)
	if err != nil {
		return InstallRecord{}, err
	}
	record, err := readInstallRecord(layout)
	if err != nil {
		return InstallRecord{}, err
	}
	digest, exists, err := regularFileDigest(layout.Binary)
	if err != nil {
		return InstallRecord{}, fmt.Errorf("inspect installed binary: %w", err)
	}
	if exists {
		if digest != record.BinarySHA256 && !force {
			return InstallRecord{}, fmt.Errorf("installed binary digest changed; use --force to remove it")
		}
		if err := os.Remove(layout.Binary); err != nil {
			return InstallRecord{}, fmt.Errorf("remove installed binary: %w", err)
		}
	}
	if err := os.Remove(layout.InstallManifest); err != nil && !os.IsNotExist(err) {
		return InstallRecord{}, fmt.Errorf("remove installation manifest: %w", err)
	}
	_ = os.Remove(layout.StateDir)
	return record, nil
}

func Enroll(scope ssiagpaths.Scope, topsID, topsName string) (EnrollmentRecord, error) {
	installLayout, err := ssiagpaths.ResolveInstall(scope)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	if _, err := verifyInstalled(installLayout); err != nil {
		return EnrollmentRecord{}, fmt.Errorf("SSIAG must be installed before enrollment: %w", err)
	}
	layout, err := ssiagpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	defaultConfig := config.Default(layout, topsName)
	if err := defaultConfig.Validate(); err != nil {
		return EnrollmentRecord{}, err
	}
	if err := ensureDirectories(layout.ConfigDir, layout.StateDir, layout.RuntimeDir); err != nil {
		return EnrollmentRecord{}, err
	}
	if err := requireAbsentOrRegular(layout.ConfigFile, "configuration"); err != nil {
		return EnrollmentRecord{}, err
	}
	if err := requireAbsentOrRegular(layout.EnrollmentManifest, "enrollment manifest"); err != nil {
		return EnrollmentRecord{}, err
	}

	cfg := defaultConfig
	if _, err := os.Lstat(layout.ConfigFile); err == nil {
		cfg, err = config.Load(layout.ConfigFile)
		if err != nil {
			return EnrollmentRecord{}, err
		}
		if cfg.TOPS.ID != topsID || cfg.Mode != string(scope) {
			return EnrollmentRecord{}, fmt.Errorf("existing configuration does not match selected TOPS and scope")
		}
		cfg.TOPS.Name = topsName
	}
	data, err := config.Marshal(cfg)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	if err := writeAtomic(layout.ConfigFile, append(data, '\n'), 0600); err != nil {
		return EnrollmentRecord{}, err
	}
	record := EnrollmentRecord{
		Schema:     "symphony.ssiag.enrollment.v1",
		Scope:      scope,
		TOPSID:     topsID,
		TOPSName:   topsName,
		ConfigFile: layout.ConfigFile,
		StateDir:   layout.StateDir,
		Socket:     layout.Socket,
	}
	if err := writeJSONAtomic(layout.EnrollmentManifest, record); err != nil {
		return EnrollmentRecord{}, err
	}
	return record, nil
}

// Unenroll removes the enrollment marker by default and preserves local data.
// Purge must be explicit and never follows a non-socket at the socket path.
func Unenroll(scope ssiagpaths.Scope, topsID string, purge bool) (EnrollmentRecord, error) {
	layout, err := ssiagpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	record, err := readEnrollmentRecord(layout)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	if err := os.Remove(layout.EnrollmentManifest); err != nil && !os.IsNotExist(err) {
		return EnrollmentRecord{}, fmt.Errorf("remove enrollment manifest: %w", err)
	}
	if purge {
		if err := removeSocketIfPresent(layout.Socket); err != nil {
			return EnrollmentRecord{}, err
		}
		if err := removeTree(layout.ConfigDir, "configuration"); err != nil {
			return EnrollmentRecord{}, err
		}
		if err := removeTree(layout.StateDir, "state"); err != nil {
			return EnrollmentRecord{}, err
		}
		_ = os.Remove(layout.RuntimeDir)
	}
	return record, nil
}

func readInstallRecord(layout ssiagpaths.InstallLayout) (InstallRecord, error) {
	data, err := readRegularFile(layout.InstallManifest, "installation manifest")
	if err != nil {
		return InstallRecord{}, err
	}
	var record InstallRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return InstallRecord{}, fmt.Errorf("decode installation manifest: %w", err)
	}
	if record.Schema != "symphony.ssiag.install.v1" || record.Scope != layout.Scope || record.Binary != layout.Binary || len(record.BinarySHA256) != sha256.Size*2 {
		return InstallRecord{}, fmt.Errorf("installation manifest does not match selected scope")
	}
	if _, err := hex.DecodeString(record.BinarySHA256); err != nil {
		return InstallRecord{}, fmt.Errorf("installation manifest contains invalid binary digest")
	}
	return record, nil
}

func verifyInstalled(layout ssiagpaths.InstallLayout) (InstallRecord, error) {
	record, err := readInstallRecord(layout)
	if err != nil {
		return InstallRecord{}, err
	}
	digest, exists, err := regularFileDigest(layout.Binary)
	if err != nil {
		return InstallRecord{}, err
	}
	if !exists || digest != record.BinarySHA256 {
		return InstallRecord{}, fmt.Errorf("installed SSIAG binary is missing or differs from its manifest")
	}
	return record, nil
}

func readEnrollmentRecord(layout ssiagpaths.InstanceLayout) (EnrollmentRecord, error) {
	data, err := readRegularFile(layout.EnrollmentManifest, "enrollment manifest")
	if err != nil {
		return EnrollmentRecord{}, err
	}
	var record EnrollmentRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return EnrollmentRecord{}, fmt.Errorf("decode enrollment manifest: %w", err)
	}
	if record.Schema != "symphony.ssiag.enrollment.v1" || record.Scope != layout.Scope || record.TOPSID != layout.TOPSID || record.ConfigFile != layout.ConfigFile || record.StateDir != layout.StateDir || record.Socket != layout.Socket {
		return EnrollmentRecord{}, fmt.Errorf("enrollment manifest does not match selected TOPS and scope")
	}
	return record, nil
}

func readRegularFile(path, label string) ([]byte, error) {
	if err := validateDirectoryChain(filepath.Dir(path)); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("refusing non-regular %s: %s", label, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	return data, nil
}

func requireAbsentOrRegular(path, label string) error {
	if err := validateDirectoryChain(filepath.Dir(path)); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular %s: %s", label, path)
	}
	return nil
}

func ensureDirectories(paths ...string) error {
	for _, path := range paths {
		if err := ensureDirectory(path); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing non-absolute directory: %s", path)
	}
	parent := filepath.Dir(path)
	if parent != path {
		if err := ensureDirectory(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 && permittedSystemAlias(path) {
			return nil
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing non-directory or symlink path: %s", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect directory %s: %w", path, err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		return fmt.Errorf("create directory %s: %w", path, err)
	}
	return nil
}

func validateDirectoryChain(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing non-absolute directory: %s", path)
	}
	parent := filepath.Dir(path)
	if parent != path {
		if err := validateDirectoryChain(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect directory %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 && permittedSystemAlias(path) {
		return nil
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing non-directory or symlink path: %s", path)
	}
	return nil
}

func permittedSystemAlias(path string) bool {
	expected := map[string]string{
		"/var": "/private/var",
		"/tmp": "/private/tmp",
		"/etc": "/private/etc",
	}
	want, ok := expected[path]
	if !ok {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == want
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func regularFileDigest(path string) (string, bool, error) {
	if err := validateDirectoryChain(filepath.Dir(path)); err != nil {
		return "", false, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", true, fmt.Errorf("refusing non-regular path: %s", path)
	}
	digest, err := fileDigest(path)
	return digest, true, err
}

// stageExecutable hashes the same bytes it stages, binding the manifest digest
// to the executable that will be activated even if the source path changes.
func stageExecutable(source, destination string, mode os.FileMode) (string, string, error) {
	if err := validateDirectoryChain(filepath.Dir(destination)); err != nil {
		return "", "", err
	}
	input, err := os.Open(source)
	if err != nil {
		return "", "", fmt.Errorf("open source executable: %w", err)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return "", "", fmt.Errorf("inspect open source executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("source executable is not a regular file")
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".symphony-ssiag-*")
	if err != nil {
		return "", "", fmt.Errorf("create temporary binary: %w", err)
	}
	tempPath := temp.Name()
	complete := false
	defer func() {
		if !complete {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(temp, hash), input); err != nil {
		temp.Close()
		return "", "", fmt.Errorf("copy executable: %w", err)
	}
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return "", "", fmt.Errorf("set executable permissions: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", "", fmt.Errorf("sync executable: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", "", fmt.Errorf("close executable: %w", err)
	}
	complete = true
	return tempPath, hex.EncodeToString(hash.Sum(nil)), nil
}

func activateExecutable(staged, destination string) error {
	if err := os.Rename(staged, destination); err != nil {
		return fmt.Errorf("install executable: %w", err)
	}
	return syncDirectory(filepath.Dir(destination))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open installation directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync installation directory: %w", err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(data, '\n'), 0600)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := validateDirectoryChain(filepath.Dir(path)); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".ssiag-data-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return fmt.Errorf("set file permissions: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	return nil
}

func removeSocketIfPresent(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect socket during purge: %w", err)
	}
	if err := validateDirectoryChain(filepath.Dir(path)); err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to purge non-socket path %s", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove socket during purge: %w", err)
	}
	return nil
}

func removeTree(path, label string) error {
	if err := validateDirectoryChain(filepath.Dir(path)); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect %s directory: %w", label, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing non-directory %s path: %s", label, path)
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("purge %s directory: %w", label, err)
	}
	return nil
}

package lifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

type EnrollmentRecord struct {
	Schema      string          `json:"schema"`
	Scope       stavpaths.Scope `json:"scope"`
	TOPSID      string          `json:"tops_id"`
	ConfigFile  string          `json:"config_file"`
	StateDir    string          `json:"state_dir"`
	LedgerFile  string          `json:"ledger_file"`
	RecoveryDir string          `json:"recovery_dir"`
	Socket      string          `json:"socket"`
}

func Enroll(scope stavpaths.Scope, topsID string, authorityUID, authorityGID uint64) (EnrollmentRecord, error) {
	install, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	info, err := os.Lstat(install.Binary)
	if err != nil || !info.Mode().IsRegular() {
		return EnrollmentRecord{}, fmt.Errorf("STAV append authority must be installed before enrollment")
	}
	layout, err := stavpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	for _, directory := range []string{layout.ConfigDir, layout.StateDir, layout.RecoveryDir, layout.RuntimeDir} {
		if err := ensurePrivateDirectory(directory); err != nil {
			return EnrollmentRecord{}, err
		}
	}
	cfg := config.Default(layout, authorityUID, authorityGID)
	if info, err := os.Lstat(layout.ConfigFile); err == nil {
		if !info.Mode().IsRegular() {
			return EnrollmentRecord{}, fmt.Errorf("refusing non-regular STAV configuration")
		}
		cfg, err = config.Load(layout.ConfigFile)
		if err != nil {
			return EnrollmentRecord{}, err
		}
		if err := config.ValidateLayout(cfg, layout); err != nil {
			return EnrollmentRecord{}, err
		}
	} else if !os.IsNotExist(err) {
		return EnrollmentRecord{}, fmt.Errorf("inspect STAV configuration: %w", err)
	}
	data, err := config.Marshal(cfg)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	configMode := os.FileMode(0o600)
	if scope == stavpaths.ScopeSystem {
		// System producers and readers may use distinct operating-system
		// identities. The configuration contains no secrets, remains
		// administrator-writable only, and supplies their endpoint trust data.
		configMode = 0o644
	}
	if err := writeAtomic(layout.ConfigFile, append(data, '\n'), configMode); err != nil {
		return EnrollmentRecord{}, err
	}
	record := EnrollmentRecord{
		Schema:      "symphony.stav.append-authority.enrollment.v1",
		Scope:       scope,
		TOPSID:      topsID,
		ConfigFile:  layout.ConfigFile,
		StateDir:    layout.StateDir,
		LedgerFile:  layout.LedgerFile,
		RecoveryDir: layout.RecoveryDir,
		Socket:      layout.Socket,
	}
	recordData, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return EnrollmentRecord{}, err
	}
	if err := writeAtomic(layout.EnrollmentFile, append(recordData, '\n'), 0o600); err != nil {
		return EnrollmentRecord{}, err
	}
	return record, nil
}

func Unenroll(scope stavpaths.Scope, topsID string, purge bool) (EnrollmentRecord, error) {
	layout, err := stavpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	record, err := readEnrollment(layout)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	if purge {
		if err := rejectActiveSocket(layout.Socket); err != nil {
			return EnrollmentRecord{}, err
		}
	}
	if err := os.Remove(layout.EnrollmentFile); err != nil && !os.IsNotExist(err) {
		return EnrollmentRecord{}, fmt.Errorf("remove STAV enrollment marker: %w", err)
	}
	if err := syncDirectory(layout.ConfigDir); err != nil {
		return EnrollmentRecord{}, err
	}
	if purge {
		for _, directory := range []string{layout.ConfigDir, layout.StateDir} {
			if err := removePrivateTree(directory); err != nil {
				return EnrollmentRecord{}, err
			}
		}
		_ = os.Remove(layout.RuntimeDir)
	}
	return record, nil
}

func readEnrollment(layout stavpaths.InstanceLayout) (EnrollmentRecord, error) {
	info, err := os.Lstat(layout.EnrollmentFile)
	if err != nil || !info.Mode().IsRegular() {
		return EnrollmentRecord{}, fmt.Errorf("STAV enrollment marker is missing or unsafe")
	}
	data, err := os.ReadFile(layout.EnrollmentFile)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	var record EnrollmentRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return EnrollmentRecord{}, fmt.Errorf("decode STAV enrollment marker: %w", err)
	}
	if record.Schema != "symphony.stav.append-authority.enrollment.v1" || record.Scope != layout.Scope || record.TOPSID != layout.TOPSID || record.ConfigFile != layout.ConfigFile || record.StateDir != layout.StateDir || record.LedgerFile != layout.LedgerFile || record.RecoveryDir != layout.RecoveryDir || record.Socket != layout.Socket {
		return EnrollmentRecord{}, fmt.Errorf("STAV enrollment marker does not match the selected TOPS")
	}
	return record, nil
}

func rejectActiveSocket(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing non-socket object at STAV socket path")
	}
	connection, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err == nil {
		_ = connection.Close()
		return fmt.Errorf("refusing to purge an active STAV enrollment")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) && !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("cannot prove STAV socket is stale: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale STAV socket: %w", err)
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing non-absolute STAV directory")
	}
	parent := filepath.Dir(path)
	if parent != path {
		if err := ensurePrivateDirectory(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 && permittedSystemAlias(path) {
			return nil
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing unsafe STAV directory component")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return fmt.Errorf("create STAV directory: %w", err)
	}
	return nil
}

func permittedSystemAlias(path string) bool {
	expected := map[string]string{"/etc": "/private/etc", "/tmp": "/private/tmp", "/var": "/private/var"}
	want, ok := expected[path]
	if !ok {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == want
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := ensurePrivateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular STAV target")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".stav-enrollment-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
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
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func removePrivateTree(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing unsafe STAV purge path")
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("purge STAV directory: %w", err)
	}
	return nil
}

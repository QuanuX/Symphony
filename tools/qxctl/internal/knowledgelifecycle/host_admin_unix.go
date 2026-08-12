//go:build darwin || linux

package knowledgelifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
)

type HostDrift struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type HostIntegrationResult struct {
	Protocol        string           `json:"protocol"`
	Operation       string           `json:"operation"`
	TOPSID          string           `json:"tops_id"`
	ProfileID       string           `json:"profile_id"`
	Present         bool             `json:"present"`
	Integration     *HostIntegration `json:"integration"`
	Drift           []HostDrift      `json:"drift"`
	RepairActions   []string         `json:"repair_actions"`
	Changed         bool             `json:"changed"`
	Recovered       bool             `json:"recovered"`
	ApplyAuthorized bool             `json:"apply_authorized"`
	Canonical       bool             `json:"canonical"`
}

type hostCommandRunner func(...string) (string, error)

type HostAdmin struct {
	store          *Store
	unitRoot       string
	systemctl      string
	run            hostCommandRunner
	requireLinux   bool
	requireRoot    bool
	sourceExecutor string
}

type HostProvisionInput struct {
	Operation       string
	ProfileID       string
	RepositoryRoot  string
	IntegrationRoot string
	RecoveryMode    string
	DesiredEnabled  bool
	ExpectedDigest  string
	Now             time.Time
}

func NewHostAdmin(store *Store) (*HostAdmin, error) {
	if store == nil {
		return nil, fmt.Errorf("lifecycle host store is required")
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve qxctl executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return nil, err
	}
	systemctl := "/usr/bin/systemctl"
	if _, err := os.Lstat(systemctl); errors.Is(err, os.ErrNotExist) {
		systemctl = "/bin/systemctl"
	}
	admin := &HostAdmin{
		store: store, unitRoot: "/etc/systemd/system", systemctl: systemctl,
		requireLinux: true, requireRoot: true, sourceExecutor: executable,
	}
	admin.run = admin.runSystemctl
	return admin, nil
}

func (admin *HostAdmin) Provision(input HostProvisionInput) (HostIntegrationResult, error) {
	if err := admin.preflight(true); err != nil {
		return HostIntegrationResult{}, err
	}
	if input.Operation != "install" && input.Operation != "update" {
		return HostIntegrationResult{}, fmt.Errorf("host provisioning operation is invalid")
	}
	if input.Operation == "install" && input.ExpectedDigest != "absent" {
		return HostIntegrationResult{}, fmt.Errorf("host install requires absent expected state")
	}
	if input.Operation == "update" && !taggedDigest(input.ExpectedDigest) {
		return HostIntegrationResult{}, fmt.Errorf("host update requires an exact existing descriptor digest")
	}
	if input.Now.IsZero() {
		input.Now = time.Now()
	}
	profile, err := admin.store.Snapshot(input.ProfileID)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	if !profile.Exists {
		return HostIntegrationResult{}, fmt.Errorf("lifecycle profile %q is absent", input.ProfileID)
	}
	digest, data, err := digestHostExecutable(admin.sourceExecutor)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	slot := filepath.Join(input.IntegrationRoot, "executors", strings.TrimPrefix(digest, "sha256:"), "qxctl")
	snapshot, err := admin.store.HostSnapshot(input.ProfileID)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	if snapshot.Exists && snapshot.Integration.IntegrationRoot != input.IntegrationRoot {
		return HostIntegrationResult{}, fmt.Errorf("host integration root relocation requires explicit uninstall followed by install")
	}
	fallbacks := make([]HostExecutor, 0, 9)
	if snapshot.Exists {
		fallbacks = append(fallbacks, snapshot.Integration.ActiveExecutor)
		fallbacks = append(fallbacks, snapshot.Integration.FallbackExecutors...)
	}
	if len(fallbacks) > 8 {
		fallbacks = fallbacks[:8]
	}
	desired := HostDesired{
		ProfileID: input.ProfileID, RepositoryRoot: input.RepositoryRoot,
		IntegrationRoot: input.IntegrationRoot, DesiredEnabled: input.DesiredEnabled,
		RecoveryMode: input.RecoveryMode, Executor: HostExecutor{Digest: digest, Path: slot},
		Fallbacks: fallbacks, State: "installed",
	}
	desired, unit, err := admin.store.PrepareHostDesired(desired)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	record, changed, err := admin.store.CommitHost(desired, input.ExpectedDigest, input.Now)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	if err := ensureHostOwner(input.IntegrationRoot, admin.store.topsID, input.ProfileID); err != nil {
		return admin.result(input.Operation, input.ProfileID), err
	}
	executorChanged, err := installHostExecutor(slot, data, digest)
	if err != nil {
		return admin.result(input.Operation, input.ProfileID), err
	}
	result, err := admin.reconcileInstalled(input.Operation, record, unit)
	if err != nil {
		return result, err
	}
	result.Changed = result.Changed || changed || executorChanged
	return result, nil
}

func (admin *HostAdmin) Status(profileID string) (HostIntegrationResult, error) {
	if err := admin.preflight(false); err != nil {
		return HostIntegrationResult{}, err
	}
	snapshot, err := admin.store.HostSnapshot(profileID)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	result := admin.result("status", profileID)
	if !snapshot.Exists {
		return result, nil
	}
	result.Present = true
	result.Integration = &snapshot.Integration
	unit, err := RenderHostSystemdUnit(snapshot.Integration)
	if err != nil {
		return result, err
	}
	result.Drift = admin.inspectDrift(snapshot.Integration, unit)
	return result, nil
}

func (admin *HostAdmin) Reconcile(profileID string, now time.Time) (HostIntegrationResult, error) {
	if err := admin.preflight(true); err != nil {
		return HostIntegrationResult{}, err
	}
	snapshot, err := admin.store.HostSnapshot(profileID)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	if !snapshot.Exists {
		return admin.result("reconcile", profileID), nil
	}
	if snapshot.Integration.State == "retiring" {
		return admin.finishUninstall("reconcile", snapshot.Integration)
	}
	record := snapshot.Integration
	unit, err := RenderHostSystemdUnit(record)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	if !validHostFile(record.ActiveExecutor.Path, record.ActiveExecutor.Digest, 0o111) {
		promoted, ok := firstValidFallback(record.FallbackExecutors)
		if !ok {
			result := admin.result("reconcile", profileID)
			result.Present, result.Integration = true, &record
			result.Drift = []HostDrift{{Code: "active_executor_unavailable", Detail: "no digest-valid fallback executor is available"}}
			return result, fmt.Errorf("host integration has no digest-valid executor candidate")
		}
		fallbacks := append([]HostExecutor{record.ActiveExecutor}, record.FallbackExecutors...)
		if len(fallbacks) > 8 {
			fallbacks = fallbacks[:8]
		}
		desired := HostDesired{
			ProfileID: profileID, RepositoryRoot: record.RepositoryRoot, IntegrationRoot: record.IntegrationRoot,
			DesiredEnabled: record.DesiredEnabled, RecoveryMode: record.RecoveryMode,
			Executor: promoted, Fallbacks: fallbacks, State: "installed",
		}
		desired, unit, err = admin.store.PrepareHostDesired(desired)
		if err != nil {
			return HostIntegrationResult{}, err
		}
		record, _, err = admin.store.CommitHost(desired, record.IntegrationDigest, now)
		if err != nil {
			return HostIntegrationResult{}, err
		}
		result, err := admin.reconcileInstalled("reconcile", record, unit)
		result.Recovered = err == nil
		result.RepairActions = append(result.RepairActions, "promoted_digest_valid_fallback_executor")
		return result, err
	}
	return admin.reconcileInstalled("reconcile", record, unit)
}

func (admin *HostAdmin) SetEnabled(profileID, expected string, enabled bool, now time.Time) (HostIntegrationResult, error) {
	if err := admin.preflight(true); err != nil {
		return HostIntegrationResult{}, err
	}
	snapshot, err := admin.store.HostSnapshot(profileID)
	if err != nil || !snapshot.Exists {
		if err == nil {
			err = fmt.Errorf("lifecycle host integration is absent")
		}
		return HostIntegrationResult{}, err
	}
	record := snapshot.Integration
	desired := HostDesired{
		ProfileID: profileID, RepositoryRoot: record.RepositoryRoot, IntegrationRoot: record.IntegrationRoot,
		DesiredEnabled: enabled, RecoveryMode: record.RecoveryMode, Executor: record.ActiveExecutor,
		Fallbacks: record.FallbackExecutors, State: "installed",
	}
	desired, unit, err := admin.store.PrepareHostDesired(desired)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	record, changed, err := admin.store.CommitHost(desired, expected, now)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	operation := "disable"
	if enabled {
		operation = "enable"
	}
	result, err := admin.reconcileInstalled(operation, record, unit)
	result.Changed = result.Changed || changed
	return result, err
}

func (admin *HostAdmin) Uninstall(profileID, expected string, now time.Time) (HostIntegrationResult, error) {
	if err := admin.preflight(true); err != nil {
		return HostIntegrationResult{}, err
	}
	snapshot, err := admin.store.HostSnapshot(profileID)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	if !snapshot.Exists {
		return admin.result("uninstall", profileID), nil
	}
	record := snapshot.Integration
	desired := HostDesired{
		ProfileID: profileID, RepositoryRoot: record.RepositoryRoot, IntegrationRoot: record.IntegrationRoot,
		DesiredEnabled: false, RecoveryMode: record.RecoveryMode, Executor: record.ActiveExecutor,
		Fallbacks: record.FallbackExecutors, State: "retiring",
	}
	desired, _, err = admin.store.PrepareHostDesired(desired)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	record, _, err = admin.store.CommitHost(desired, expected, now)
	if err != nil {
		return HostIntegrationResult{}, err
	}
	return admin.finishUninstall("uninstall", record)
}

func (admin *HostAdmin) reconcileInstalled(operation string, record HostIntegration, unit []byte) (HostIntegrationResult, error) {
	result := admin.result(operation, record.ProfileID)
	result.Present, result.Integration = true, &record
	unitPath := filepath.Join(admin.unitRoot, record.UnitName)
	if !validHostFile(record.ActiveExecutor.Path, record.ActiveExecutor.Digest, 0o111) {
		result.Drift = []HostDrift{{Code: "active_executor_unavailable", Detail: record.ActiveExecutor.Path}}
		return result, fmt.Errorf("active host executor does not match its content digest")
	}
	if !validHostFile(unitPath, record.UnitDigest, 0) {
		if err := writeHostAtomic(unitPath, unit, 0o644); err != nil {
			return result, err
		}
		result.Changed = true
		result.RepairActions = append(result.RepairActions, "replaced_systemd_unit_from_descriptor")
	}
	if _, err := admin.run("daemon-reload"); err != nil {
		return result, err
	}
	if output, _ := admin.run("is-enabled", record.UnitName); strings.TrimSpace(output) == "masked" {
		if _, err := admin.run("unmask", record.UnitName); err != nil {
			return result, err
		}
		result.Changed = true
		result.RepairActions = append(result.RepairActions, "removed_systemd_unit_mask")
	}
	action := "disable"
	if record.DesiredEnabled {
		action = "enable"
	}
	if _, err := admin.run(action, record.UnitName); err != nil {
		return result, err
	}
	result.Drift = admin.inspectDrift(record, unit)
	if len(result.Drift) != 0 {
		return result, fmt.Errorf("host integration remains divergent after reconciliation")
	}
	return result, nil
}

func (admin *HostAdmin) finishUninstall(operation string, record HostIntegration) (HostIntegrationResult, error) {
	result := admin.result(operation, record.ProfileID)
	result.Present, result.Integration = true, &record
	unitPath := filepath.Join(admin.unitRoot, record.UnitName)
	if info, err := os.Lstat(unitPath); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return result, fmt.Errorf("refusing unsafe lifecycle systemd unit")
		}
		if _, err := admin.run("disable", "--now", record.UnitName); err != nil {
			return result, err
		}
		if err := os.Remove(unitPath); err != nil {
			return result, err
		}
		if err := syncHostDirectory(admin.unitRoot); err != nil {
			return result, err
		}
		result.Changed = true
	}
	if _, err := admin.run("daemon-reload"); err != nil {
		return result, err
	}
	if err := removeOwnedHostRoot(record.IntegrationRoot, record.TOPSID, record.ProfileID); err != nil {
		return result, err
	}
	if err := admin.store.RemoveHost(record.ProfileID, record.IntegrationDigest); err != nil {
		return result, err
	}
	result.Present, result.Integration, result.Changed = false, nil, true
	result.RepairActions = append(result.RepairActions, "completed_retiring_integration_cleanup")
	return result, nil
}

func (admin *HostAdmin) inspectDrift(record HostIntegration, unit []byte) []HostDrift {
	drift := make([]HostDrift, 0)
	if !validHostFile(record.ActiveExecutor.Path, record.ActiveExecutor.Digest, 0o111) {
		drift = append(drift, HostDrift{Code: "active_executor_mismatch", Detail: record.ActiveExecutor.Path})
	}
	unitPath := filepath.Join(admin.unitRoot, record.UnitName)
	if !validHostFile(unitPath, DigestHostUnit(unit), 0) {
		drift = append(drift, HostDrift{Code: "unit_mismatch", Detail: unitPath})
	}
	output, err := admin.run("is-enabled", record.UnitName)
	managerState := strings.TrimSpace(output)
	enabled := err == nil && managerState == "enabled"
	if managerState == "masked" {
		drift = append(drift, HostDrift{Code: "unit_masked", Detail: "systemd unit is masked outside the desired descriptor"})
	}
	if err != nil && !hostOneOf(managerState, "disabled", "static", "masked", "not-found") {
		drift = append(drift, HostDrift{Code: "manager_unavailable", Detail: "systemd enablement could not be observed"})
	}
	if enabled != record.DesiredEnabled {
		drift = append(drift, HostDrift{Code: "enablement_mismatch", Detail: "systemd enablement differs from desired state"})
	}
	sort.Slice(drift, func(i, j int) bool { return drift[i].Code < drift[j].Code })
	return drift
}

func (admin *HostAdmin) preflight(mutation bool) error {
	if admin.requireLinux && runtime.GOOS != "linux" {
		return fmt.Errorf("native lifecycle host integration is Linux-only; use WSL or a remote Linux TOPS node")
	}
	if mutation && admin.requireRoot && os.Geteuid() != 0 {
		return fmt.Errorf("system lifecycle host integration mutation requires administrator privileges")
	}
	return nil
}

func (admin *HostAdmin) result(operation, profileID string) HostIntegrationResult {
	return HostIntegrationResult{
		Protocol: HostResultProtocol, Operation: operation, TOPSID: admin.store.topsID, ProfileID: profileID,
		Drift: make([]HostDrift, 0), RepairActions: make([]string, 0),
		ApplyAuthorized: false, Canonical: false,
	}
}

func (admin *HostAdmin) runSystemctl(arguments ...string) (string, error) {
	info, err := os.Lstat(admin.systemctl)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return "", fmt.Errorf("trusted root-owned systemctl executable is unavailable")
	}
	status, ok := info.Sys().(*syscall.Stat_t)
	if !ok || status.Uid != 0 {
		return "", fmt.Errorf("trusted root-owned systemctl executable is unavailable")
	}
	command := exec.Command(admin.systemctl, arguments...)
	command.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	output, err := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("systemctl %s: %s", strings.Join(arguments, " "), text)
	}
	return text, nil
}

func digestHostExecutable(path string) (string, []byte, error) {
	data, err := readProtectedHostFile(path, 128*1024*1024, 0o111)
	if err != nil {
		return "", nil, fmt.Errorf("qxctl executable is not a protected bounded no-follow regular file: %w", err)
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), data, nil
}

func installHostExecutor(path string, data []byte, digest string) (bool, error) {
	if validHostFile(path, digest, 0o111) {
		return false, nil
	}
	if err := ensureProtectedHostDirectory(filepath.Dir(path)); err != nil {
		return false, err
	}
	if err := writeHostAtomic(path, data, 0o500); err != nil {
		return false, err
	}
	return true, nil
}

func ensureHostOwner(root, topsID, profileID string) error {
	if !safeHostPath(root) || !validTOPSID(topsID) || !safeToken(profileID, 256) {
		return fmt.Errorf("host integration root identity is invalid")
	}
	if err := ensureProtectedHostDirectory(root); err != nil {
		return err
	}
	marker := filepath.Join(root, ".symphony-qxctl-lifecycle-owner")
	wanted := []byte(topsID + "\n" + profileID + "\n")
	if existing, err := readProtectedHostFile(marker, 4096, 0); err == nil {
		if !bytes.Equal(existing, wanted) {
			return fmt.Errorf("host integration root is owned by a different TOPS/profile")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("host integration ownership marker is unsafe: %w", err)
	}
	return writeHostAtomic(marker, wanted, 0o600)
}

func ensureProtectedHostDirectory(path string) error {
	if !safeHostPath(path) {
		return fmt.Errorf("host integration path is unsafe")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		current := string(filepath.Separator)
		for _, part := range strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator)) {
			current = filepath.Join(current, part)
			ancestor, err := os.Lstat(current)
			if err != nil || !ancestor.IsDir() || ancestor.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("host integration path contains an unsafe ancestor")
			}
			status, ok := ancestor.Sys().(*syscall.Stat_t)
			if !ok || (status.Uid != 0 && status.Uid != uint32(os.Geteuid())) ||
				(ancestor.Mode().Perm()&0o022 != 0 && (status.Uid != 0 || ancestor.Mode()&os.ModeSticky == 0)) {
				return fmt.Errorf("host integration path contains an unprotected ancestor")
			}
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("host integration directory is not protected")
	}
	status, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		status.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("host integration directory is not protected")
	}
	return os.Chmod(path, 0o700)
}

func writeHostAtomic(path string, data []byte, mode os.FileMode) error {
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("host integration parent directory is not protected")
	}
	status, ok := info.Sys().(*syscall.Stat_t)
	if !ok || status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("host integration parent directory is not protected")
	}
	temporary, err := os.CreateTemp(parent, ".host-tmp-")
	if err != nil {
		return err
	}
	temp := temporary.Name()
	cleanup := func() { _ = temporary.Close(); _ = os.Remove(temp) }
	if err := temporary.Chmod(mode); err != nil {
		cleanup()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temp)
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return syncHostDirectory(filepath.Dir(path))
}

func syncHostDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func validHostFile(path, digest string, executable os.FileMode) bool {
	data, err := readProtectedHostFile(path, 128*1024*1024, executable)
	if err != nil {
		return false
	}
	hash := sha256.Sum256(data)
	return "sha256:"+hex.EncodeToString(hash[:]) == digest
}

func readProtectedHostFile(path string, maximum int64, executable os.FileMode) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("could not bind protected file descriptor")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maximum ||
		(executable != 0 && info.Mode().Perm()&executable == 0) {
		return nil, fmt.Errorf("file type, size, or executable mode is invalid")
	}
	status, ok := info.Sys().(*syscall.Stat_t)
	if !ok || status.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf("file owner or write mode is invalid")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(data)) > maximum {
		return nil, fmt.Errorf("file read exceeded its bound")
	}
	return data, nil
}

func firstValidFallback(values []HostExecutor) (HostExecutor, bool) {
	for _, value := range values {
		if validHostFile(value.Path, value.Digest, 0o111) {
			return value, true
		}
	}
	return HostExecutor{}, false
}

func removeOwnedHostRoot(root, topsID, profileID string) error {
	marker := filepath.Join(root, ".symphony-qxctl-lifecycle-owner")
	wanted := topsID + "\n" + profileID + "\n"
	data, err := readProtectedHostFile(marker, 4096, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || string(data) != wanted {
		return fmt.Errorf("refusing to remove an unowned host integration root")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != ".symphony-qxctl-lifecycle-owner" && entry.Name() != "executors" {
			return fmt.Errorf("refusing host integration cleanup with unknown object %q", entry.Name())
		}
	}
	if err := os.RemoveAll(filepath.Join(root, "executors")); err != nil {
		return err
	}
	if err := os.Remove(marker); err != nil {
		return err
	}
	return os.Remove(root)
}

func hostOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

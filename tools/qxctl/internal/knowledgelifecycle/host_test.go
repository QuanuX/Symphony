//go:build darwin || linux

package knowledgelifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHostIntegrationDescriptorIsDigestBoundAndCASProtected(t *testing.T) {
	store := hostTestStore(t, t.TempDir())
	desired := hostTestDesired(t, store, "default", []byte("executor-v1"), nil)
	record, changed, err := store.CommitHost(desired, "absent", time.Unix(1_700_000_000, 0))
	if err != nil || !changed || record.Generation != 1 || record.IntegrationDigest == "" {
		t.Fatalf("initial host integration commit failed: %+v changed=%t err=%v", record, changed, err)
	}
	if _, _, err := store.CommitHost(desired, "absent", time.Unix(1_700_000_001, 0)); err == nil {
		t.Fatal("stale absent compare-and-swap was accepted")
	}
	stable, changed, err := store.CommitHost(desired, record.IntegrationDigest, time.Unix(1_700_000_001, 0))
	if err != nil || changed || stable.IntegrationDigest != record.IntegrationDigest || stable.Generation != 1 {
		t.Fatalf("identical desired host integration was not a stable no-op: %+v changed=%t err=%v", stable, changed, err)
	}
	snapshot, err := store.HostSnapshot("default")
	if err != nil || !snapshot.Exists || snapshot.Integration.IntegrationDigest != record.IntegrationDigest {
		t.Fatalf("committed host integration was not readable: %+v err=%v", snapshot, err)
	}
}

func TestHostSystemdUnitIsReportOnlyAndLooselyOrdersSecurityServices(t *testing.T) {
	store := hostTestStore(t, t.TempDir())
	desired := hostTestDesired(t, store, "default", []byte("executor"), nil)
	record, _, err := store.CommitHost(desired, "absent", time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	unit, err := RenderHostSystemdUnit(record)
	if err != nil {
		t.Fatal(err)
	}
	text := string(unit)
	for _, required := range []string{
		"knowledge\" \"lifecycle\" \"host\" \"run", "Type=oneshot", "RemainAfterExit=yes",
		"Wants=symphony-ssiag@" + testTOPSID + ".service symphony-stav@" + testTOPSID + ".service",
		"Restart=on-failure", "NoNewPrivileges=true",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("systemd unit omits %q:\n%s", required, text)
		}
	}
	for _, forbidden := range []string{" knowledge lifecycle apply", "Requires=", "ExecStart=/bin/sh", "ExecStart=/usr/bin/env"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("systemd unit contains forbidden behavior %q:\n%s", forbidden, text)
		}
	}
}

func TestHostReconcilePromotesDigestValidFallbackAndUninstallResumes(t *testing.T) {
	stateRoot := t.TempDir()
	store := hostTestStore(t, stateRoot)
	unitRoot := t.TempDir()
	integrationRoot := filepath.Join(t.TempDir(), "integration")
	if err := ensureHostOwner(integrationRoot, testTOPSID, "default"); err != nil {
		t.Fatal(err)
	}
	active := hostTestExecutor(t, integrationRoot, "active", []byte("active"))
	fallback := hostTestExecutor(t, integrationRoot, "fallback", []byte("fallback"))
	desired := HostDesired{
		ProfileID: "default", RepositoryRoot: t.TempDir(), IntegrationRoot: integrationRoot,
		DesiredEnabled: true, RecoveryMode: "discover", Executor: active,
		Fallbacks: []HostExecutor{fallback}, State: "installed",
	}
	desired, _, err := store.PrepareHostDesired(desired)
	if err != nil {
		t.Fatal(err)
	}
	record, _, err := store.CommitHost(desired, "absent", time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(active.Path); err != nil {
		t.Fatal(err)
	}
	enabled := false
	runner := func(arguments ...string) (string, error) {
		switch arguments[0] {
		case "enable":
			enabled = true
		case "disable":
			enabled = false
		case "is-enabled":
			if enabled {
				return "enabled", nil
			}
			return "disabled", os.ErrNotExist
		}
		return "", nil
	}
	admin := &HostAdmin{store: store, unitRoot: unitRoot, run: runner}
	result, err := admin.Reconcile("default", time.Unix(1_700_000_001, 0))
	if err != nil || !result.Recovered || result.Integration == nil || result.Integration.ActiveExecutor.Digest != fallback.Digest {
		t.Fatalf("fallback promotion failed: %+v err=%v", result, err)
	}
	result, err = admin.Uninstall("default", result.Integration.IntegrationDigest, time.Unix(1_700_000_002, 0))
	if err != nil || result.Present {
		t.Fatalf("host integration uninstall failed: %+v err=%v", result, err)
	}
	status, err := admin.Status("default")
	if err != nil || status.Present {
		t.Fatalf("removed host integration still appears present: %+v err=%v", status, err)
	}
	if _, err := os.Lstat(integrationRoot); !os.IsNotExist(err) {
		t.Fatal("owned integration root survived uninstall")
	}
	_ = record
}

func TestHostExecutableReadsAreNoFollowAndModeProtected(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "qxctl")
	if err := os.WriteFile(executable, []byte("trusted"), 0o500); err != nil {
		t.Fatal(err)
	}
	digest, data, err := digestHostExecutable(executable)
	if err != nil || digest == "" || string(data) != "trusted" {
		t.Fatalf("protected executor was rejected: digest=%q data=%q err=%v", digest, data, err)
	}
	if err := os.Chmod(executable, 0o520); err != nil {
		t.Fatal(err)
	}
	if _, _, err := digestHostExecutable(executable); err == nil {
		t.Fatal("group-writable executor was accepted")
	}
	link := filepath.Join(root, "qxctl-link")
	if err := os.Symlink(executable, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := digestHostExecutable(link); err == nil {
		t.Fatal("symlink executor was accepted")
	}
}

func TestHostProvisionRequiresExactOperationStateAndRepairsItsExecutor(t *testing.T) {
	store := hostTestStore(t, t.TempDir())
	source := filepath.Join(t.TempDir(), "qxctl")
	if err := os.WriteFile(source, []byte("qxctl-v1"), 0o500); err != nil {
		t.Fatal(err)
	}
	enabled := false
	runner := func(arguments ...string) (string, error) {
		switch arguments[0] {
		case "enable":
			enabled = true
		case "disable":
			enabled = false
		case "is-enabled":
			if enabled {
				return "enabled", nil
			}
			return "disabled", os.ErrNotExist
		}
		return "", nil
	}
	admin := &HostAdmin{store: store, unitRoot: t.TempDir(), run: runner, sourceExecutor: source}
	integrationRoot := filepath.Join(t.TempDir(), "integration")
	input := HostProvisionInput{
		Operation: "install", ProfileID: "default", RepositoryRoot: t.TempDir(),
		IntegrationRoot: integrationRoot, RecoveryMode: "discover", DesiredEnabled: true,
		ExpectedDigest: "absent", Now: time.Unix(1_700_000_000, 0),
	}
	installed, err := admin.Provision(input)
	if err != nil || !installed.Present || installed.Integration == nil || !installed.Changed {
		t.Fatalf("host provision failed: %+v err=%v", installed, err)
	}
	if _, err := os.Lstat(installed.Integration.ActiveExecutor.Path); err != nil {
		t.Fatalf("content-addressed executor is absent: %v", err)
	}
	if err := os.Remove(installed.Integration.ActiveExecutor.Path); err != nil {
		t.Fatal(err)
	}
	input.Operation = "update"
	input.ExpectedDigest = installed.Integration.IntegrationDigest
	input.Now = time.Unix(1_700_000_001, 0)
	repaired, err := admin.Provision(input)
	if err != nil || !repaired.Changed || repaired.Integration == nil || repaired.Integration.IntegrationDigest != installed.Integration.IntegrationDigest {
		t.Fatalf("semantic update did not repair the accepted executor: %+v err=%v", repaired, err)
	}
	if !validHostFile(repaired.Integration.ActiveExecutor.Path, repaired.Integration.ActiveExecutor.Digest, 0o111) {
		t.Fatal("repaired executor does not match protected content evidence")
	}
	input.IntegrationRoot = filepath.Join(t.TempDir(), "relocated")
	if _, err := admin.Provision(input); err == nil || !strings.Contains(err.Error(), "uninstall followed by install") {
		t.Fatalf("in-place integration-root relocation was not rejected: %v", err)
	}
	input.Operation = "install"
	if _, err := admin.Provision(input); err == nil || !strings.Contains(err.Error(), "absent expected state") {
		t.Fatalf("install accepted an existing expected digest: %v", err)
	}
}

func TestHostReconcileRemovesSystemdMask(t *testing.T) {
	store := hostTestStore(t, t.TempDir())
	desired := hostTestDesired(t, store, "default", []byte("executor"), nil)
	record, _, err := store.CommitHost(desired, "absent", time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	masked := true
	runner := func(arguments ...string) (string, error) {
		switch arguments[0] {
		case "is-enabled":
			if masked {
				return "masked", os.ErrPermission
			}
			return "enabled", nil
		case "unmask":
			masked = false
		}
		return "", nil
	}
	admin := &HostAdmin{store: store, unitRoot: t.TempDir(), run: runner}
	result, err := admin.Reconcile("default", time.Unix(1_700_000_001, 0))
	if err != nil || masked || !result.Changed || len(result.Drift) != 0 ||
		!containsHostRepair(result.RepairActions, "removed_systemd_unit_mask") {
		t.Fatalf("masked unit was not reconciled: %+v masked=%t err=%v", result, masked, err)
	}
	_ = record
}

func containsHostRepair(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func hostTestStore(t *testing.T, stateRoot string) *Store {
	t.Helper()
	store, err := NewStore(stateRoot, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Set(profileInput(t.TempDir()), "absent"); err != nil {
		t.Fatal(err)
	}
	return store
}

func hostTestDesired(t *testing.T, store *Store, profileID string, executable []byte, fallbacks []HostExecutor) HostDesired {
	t.Helper()
	root := filepath.Join(t.TempDir(), "integration")
	if err := ensureHostOwner(root, testTOPSID, profileID); err != nil {
		t.Fatal(err)
	}
	executor := hostTestExecutor(t, root, "active", executable)
	desired := HostDesired{
		ProfileID: profileID, RepositoryRoot: t.TempDir(), IntegrationRoot: root,
		DesiredEnabled: true, RecoveryMode: "discover", Executor: executor,
		Fallbacks: fallbacks, State: "installed",
	}
	prepared, _, err := store.PrepareHostDesired(desired)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func hostTestExecutor(t *testing.T, root, name string, data []byte) HostExecutor {
	t.Helper()
	_ = name
	digest := sha256.Sum256(data)
	tagged := "sha256:" + hex.EncodeToString(digest[:])
	path := filepath.Join(root, "executors", strings.TrimPrefix(tagged, "sha256:"), "qxctl")
	if _, err := installHostExecutor(path, data, tagged); err != nil {
		t.Fatal(err)
	}
	return HostExecutor{Digest: tagged, Path: path}
}

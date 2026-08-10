package knowledgebinding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTemporalProfilesAndLegacyReadCompatibility(t *testing.T) {
	now := time.Date(2026, time.August, 10, 12, 34, 56, 987654321, time.FixedZone("fixture", -4*60*60))
	registry := nextRegistry(Registry{}, false, now)
	if registry.UpdatedAt != "2026-08-10T16:34:56Z" {
		t.Fatalf("new registry did not use canonical STSC whole-second UTC: %q", registry.UpdatedAt)
	}
	registry.RegistryDigest = "sha256:" + strings.Repeat("0", 64)
	registry.UpdatedAt = "2026-08-10T16:34:56.987654321Z"
	digest, err := calculateDigest(registry)
	if err != nil {
		t.Fatalf("calculate legacy registry digest: %v", err)
	}
	registry.RegistryDigest = digest
	if err := validateRegistry(registry); err != nil {
		t.Fatalf("legacy registry timestamp lost read compatibility: %v", err)
	}
}

func TestBindSnapshotDoctorAndUnbind(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	prefix := createSKVIInstallation(t)
	store, err := NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	registry, changed, err := store.Bind("skvi", prefix, "0.1.0-dev", "absent")
	if err != nil {
		t.Fatalf("first bind failed: %v", err)
	}
	if !changed || registry.Generation != 1 || len(registry.Bindings) != 1 ||
		!taggedDigest(registry.RegistryDigest) {
		t.Fatalf("unexpected first registry: %+v", registry)
	}
	if registry.Bindings[0].State != "bound_undocked" ||
		registry.Bindings[0].DefaultReceptor != nil {
		t.Fatalf("binding acquired an unauthorized lifecycle state: %+v", registry.Bindings[0])
	}

	snapshot, err := store.Snapshot()
	if err != nil || !snapshot.Exists || snapshot.Registry.RegistryDigest != registry.RegistryDigest {
		t.Fatalf("snapshot mismatch: %+v error=%v", snapshot, err)
	}
	same, changed, err := store.Bind("skvi", prefix, "0.1.0-dev", registry.RegistryDigest)
	if err != nil || changed || same.RegistryDigest != registry.RegistryDigest {
		t.Fatalf("idempotent bind changed state: %+v changed=%t error=%v", same, changed, err)
	}
	if _, _, err := store.Bind("skvi", prefix, "0.1.0-dev",
		"sha256:"+strings.Repeat("0", 64)); err == nil {
		t.Fatal("stale expected digest was accepted")
	}

	report, err := store.Doctor()
	if err != nil || !report.Healthy || len(report.Results) != 1 || !report.Results[0].Healthy {
		t.Fatalf("healthy binding failed doctor: %+v error=%v", report, err)
	}

	executable := registry.Bindings[0].ExecutablePath
	if err := os.WriteFile(executable, []byte("changed\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	report, err = store.Doctor()
	if err != nil || report.Healthy || report.Results[0].Code != "binding.content_mismatch" {
		t.Fatalf("changed executable passed doctor: %+v error=%v", report, err)
	}

	updated, changed, err := store.Unbind("skvi", registry.RegistryDigest)
	if err != nil || !changed || updated.Generation != 2 || len(updated.Bindings) != 0 ||
		updated.PreviousRegistryDigest == nil ||
		*updated.PreviousRegistryDigest != registry.RegistryDigest {
		t.Fatalf("unbind failed: %+v changed=%t error=%v", updated, changed, err)
	}
}

func TestRegistryRejectsUnsafeStateAndTampering(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	store, err := NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(stateRoot, "symphony", "qxctl", "knowledge", "engine-bindings")
	lockPath := filepath.Join(directory, "registry.lock")
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "target"), lockPath); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(); err == nil {
		t.Fatal("symlinked registry lock was accepted")
	}

	stateRoot2 := filepath.Join(t.TempDir(), "state")
	prefix := createSKVIInstallation(t)
	store2, _ := NewStore(stateRoot2)
	registry, _, err := store2.Bind("skvi", prefix, "0.1.0-dev", "absent")
	if err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(
		stateRoot2, "symphony", "qxctl", "knowledge", "engine-bindings", registryFileName)
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	object["unexpected"] = true
	data, _ = json.Marshal(object)
	if err := os.WriteFile(registryPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store2.Snapshot(); err == nil {
		t.Fatal("registry with an unknown field was accepted")
	}

	delete(object, "unexpected")
	object["registry_digest"] = registry.RegistryDigest
	object["generation"] = float64(2)
	data, _ = json.Marshal(object)
	if err := os.WriteFile(registryPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store2.Snapshot(); err == nil {
		t.Fatal("tampered registry was accepted")
	}

	duplicate := strings.Replace(
		string(data), `"protocol":"`+Protocol+`"`,
		`"protocol":"`+Protocol+`","protocol":"`+Protocol+`"`, 1)
	if err := os.WriteFile(registryPath, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store2.Snapshot(); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate registry key was accepted or misdiagnosed: %v", err)
	}
}

func TestInvalidRolesExpectedStateAndStateRootsFailClosed(t *testing.T) {
	if _, err := NewStore("relative"); err == nil {
		t.Fatal("relative state root was accepted")
	}
	root := filepath.VolumeName(filepath.Clean(os.TempDir())) + string(os.PathSeparator)
	if _, err := NewStore(root); err == nil {
		t.Fatal("filesystem root was accepted as application state")
	}
	store, _ := NewStore(filepath.Join(t.TempDir(), "state"))
	if _, _, err := store.Bind("unknown", t.TempDir(), "0.1.0-dev", "absent"); err == nil {
		t.Fatal("unknown role was accepted")
	}
	if _, _, err := store.Unbind("skvi", "current"); err == nil {
		t.Fatal("non-exact expected state was accepted")
	}
	if _, _, err := store.Unbind("skvi", "absent"); err == nil {
		t.Fatal("unbind accepted an absent registry")
	}
}

func createSKVIInstallation(t *testing.T) string {
	t.Helper()
	prefix := t.TempDir()
	version := "0.1.0-dev"
	base := "share/doc/symphony/skvi-engine/" + version + "/"
	license := "share/licenses/symphony-skvi-engine/" + version + "/"
	files := []string{
		"libexec/symphony/skvi-engine/" + version + "/symphony-skvi",
		"share/symphony/receipts/skvi-engine/" + version + "/install-receipt.json",
		base + "INTENT.md",
		base + "MANIFEST.md",
		base + "INSTALL.md",
		base + "SKILL.md",
		base + "SPEC.md",
		license + "LICENSE-AGPL-3.0",
		license + "nlohmann-json-LICENSE.MIT",
	}
	for _, relative := range files {
		path := filepath.Join(prefix, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(relative, "install-receipt.json") {
			continue
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(relative, "libexec/") {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte(relative+"\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	receipt := map[string]any{
		"protocol":         "symphony.knowledge.install-receipt.v1",
		"module_id":        "skvi-engine",
		"version":          version,
		"install_scope":    "prefix",
		"prefix_mode":      "installation_prefix",
		"state":            "installed_undocked",
		"active":           false,
		"default_receptor": nil,
		"files":            files,
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(
		prefix, "share/symphony/receipts/skvi-engine", version, "install-receipt.json")
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return prefix
}

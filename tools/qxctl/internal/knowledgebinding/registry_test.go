package knowledgebinding

import (
	"encoding/json"
	"fmt"
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
	for _, protocol := range []string{ProtocolV1, ProtocolV2} {
		registry = fixtureRegistry(t, protocol, nil)
		registry.UpdatedAt = "2026-08-10T16:34:56.987654321Z"
		digest, err := calculateDigest(registry)
		if err != nil {
			t.Fatalf("calculate %s registry digest: %v", protocol, err)
		}
		registry.RegistryDigest = digest
		if _, err := registryCompatibility(registry); err != nil {
			t.Fatalf("%s fractional timestamp lost read compatibility: %v", protocol, err)
		}
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
	if !changed || registry.Protocol != ProtocolV2 || registry.FormatVersion != 2 ||
		registry.PreviousRegistryProtocol != "absent" || registry.Generation != 1 || len(registry.Bindings) != 1 ||
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
		*updated.PreviousRegistryDigest != registry.RegistryDigest ||
		updated.PreviousRegistryProtocol != ProtocolV2 {
		t.Fatalf("unbind failed: %+v changed=%t error=%v", updated, changed, err)
	}
}

func TestV1DualReadAndExplicitMigration(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	store, err := NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	v1 := fixtureRegistry(t, ProtocolV1, []Binding{fixtureBinding(t, "skvi")})
	writeRegistryFixture(t, store, v1)

	administrative, err := store.AdministrativeSnapshot()
	if err != nil || administrative.Compatibility != CompatibilityV1 ||
		administrative.Registry.Protocol != ProtocolV1 {
		t.Fatalf("canonical v1 registry lost dual-read compatibility: %+v error=%v", administrative, err)
	}
	if _, err := store.Snapshot(); err != nil {
		t.Fatalf("canonical v1 registry is not operationally readable: %v", err)
	}
	if _, _, err := store.Bind("skvi", t.TempDir(), "0.1.0-dev", v1.RegistryDigest); err == nil ||
		!strings.Contains(err.Error(), "knowledge engines migrate") {
		t.Fatalf("v1 mutation did not require explicit migration: %v", err)
	}

	migrated, changed, err := store.Migrate(v1.RegistryDigest)
	if err != nil || !changed || migrated.Protocol != ProtocolV2 || migrated.FormatVersion != 2 ||
		migrated.Generation != v1.Generation+1 || migrated.PreviousRegistryDigest == nil ||
		*migrated.PreviousRegistryDigest != v1.RegistryDigest ||
		migrated.PreviousRegistryProtocol != ProtocolV1 ||
		len(migrated.Bindings) != 1 || migrated.Bindings[0] != v1.Bindings[0] {
		t.Fatalf("explicit v1 migration lost exact lineage or binding state: %+v changed=%t error=%v", migrated, changed, err)
	}
	current, err := store.Snapshot()
	if err != nil || current.Compatibility != CompatibilityCurrent {
		t.Fatalf("migrated registry is not current: %+v error=%v", current, err)
	}
	same, changed, err := store.Migrate(migrated.RegistryDigest)
	if err != nil || changed || same.RegistryDigest != migrated.RegistryDigest {
		t.Fatalf("retry against current v2 was not a stable no-op: %+v changed=%t error=%v", same, changed, err)
	}
}

func TestV1DigestPreimageRemainsStable(t *testing.T) {
	registry := fixtureRegistry(t, ProtocolV1, nil)
	const expected = "sha256:8219cf439739f40eee7136c18cf759ad48fbd3fb7362e4dc6323fce91dd558cb"
	if registry.RegistryDigest != expected {
		t.Fatalf("v1 digest preimage changed: got %s want %s", registry.RegistryDigest, expected)
	}
}

func TestLegacyWidenedV1IsMigrationInputNotV1Semantics(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	store, err := NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	legacy := fixtureRegistry(t, ProtocolV1, []Binding{fixtureBinding(t, "sav")})
	if err := validateRegistry(legacy); err == nil || !strings.Contains(err.Error(), "nonconforming v1") {
		t.Fatalf("legacy widened representation was accepted as canonical v1: %v", err)
	}
	writeRegistryFixture(t, store, legacy)

	administrative, err := store.AdministrativeSnapshot()
	if err != nil || administrative.Compatibility != CompatibilityLegacyV1MigrationRequired {
		t.Fatalf("legacy widened v1 was not recognized as migration input: %+v error=%v", administrative, err)
	}
	if _, err := store.Snapshot(); err == nil || !strings.Contains(err.Error(), "explicit migration") {
		t.Fatalf("legacy widened v1 was accepted for operational use: %v", err)
	}
	report, err := store.Doctor()
	if err != nil || report.Healthy || len(report.Results) == 0 ||
		report.Results[0].Code != "binding.registry_incompatible" {
		t.Fatalf("doctor did not expose legacy migration requirement: %+v error=%v", report, err)
	}

	migrated, changed, err := store.Migrate(legacy.RegistryDigest)
	if err != nil || !changed || migrated.Protocol != ProtocolV2 ||
		migrated.PreviousRegistryProtocol != ProtocolV1 || migrated.Bindings[0].Role != "sav" {
		t.Fatalf("legacy widened v1 did not migrate without reinterpretation: %+v changed=%t error=%v", migrated, changed, err)
	}
	if _, err := store.Snapshot(); err != nil {
		t.Fatalf("migrated legacy registry is not operationally readable: %v", err)
	}
}

func TestV2PreservesUnknownBoundedRoleAndFailsOperationally(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	store, err := NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	registry := fixtureRegistry(t, ProtocolV2, []Binding{fixtureGenericBinding(t, "future-vector")})
	writeRegistryFixture(t, store, registry)

	administrative, err := store.AdministrativeSnapshot()
	if err != nil || administrative.Compatibility != CompatibilityUnsupportedRolePreserved ||
		administrative.Registry.Bindings[0].Role != "future-vector" {
		t.Fatalf("bounded future role was not preserved for administration: %+v error=%v", administrative, err)
	}
	if _, err := store.Snapshot(); err == nil || !strings.Contains(err.Error(), "unsupported by this qxctl version") {
		t.Fatalf("unknown future role was accepted for operational use: %v", err)
	}
	if _, _, err := store.Unbind("skvi", registry.RegistryDigest); err == nil ||
		!strings.Contains(err.Error(), "unsupported by this qxctl version") {
		t.Fatalf("partially understood v2 registry was mutated: %v", err)
	}
	report, err := store.Doctor()
	if err != nil || report.Healthy || len(report.Results) != 2 ||
		report.Results[0].Code != "binding.registry_incompatible" ||
		report.Results[1].Code != "binding.role_unsupported" {
		t.Fatalf("doctor did not report preserved unsupported role: %+v error=%v", report, err)
	}
}

func TestVersionedFieldsAndExtensibleRoleBoundFailClosed(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	store, err := NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	v1 := fixtureRegistry(t, ProtocolV1, nil)
	encoded, err := encodeRegistry(v1)
	if err != nil {
		t.Fatal(err)
	}
	withV2Field := strings.Replace(
		string(encoded), `"protocol": "`+ProtocolV1+`",`,
		`"protocol": "`+ProtocolV1+`",`+"\n  \"format_version\": 0,", 1)
	if err := store.withStateLock(true, func(directory *os.File) error {
		return writeRegistry(directory, []byte(withV2Field))
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdministrativeSnapshot(); err == nil || !strings.Contains(err.Error(), "v2-only fields") {
		t.Fatalf("v1 accepted a v2-only field: %v", err)
	}

	base := fixtureGenericBinding(t, "future-000")
	bindings := make([]Binding, maxV2Bindings+1)
	for index := range bindings {
		bindings[index] = base
		bindings[index].Role = fmt.Sprintf("future-%03d", index)
		bindings[index].ModuleID = fmt.Sprintf("future-%03d-engine", index)
		bindings[index].EngineID = fmt.Sprintf("symphony-future-%03d", index)
	}
	overBound := fixtureRegistry(t, ProtocolV2, bindings)
	if err := validateRegistry(overBound); err == nil || !strings.Contains(err.Error(), "exceeds the role bound") {
		t.Fatalf("v2 accepted more than %d bindings: %v", maxV2Bindings, err)
	}
}

func TestFutureSupportedRoleCannotWidenLegacyV1Adapter(t *testing.T) {
	const role = "future-added"
	identity := roleIdentity{moduleID: "future-added-engine", engineID: "symphony-future-added"}
	supportedRoles[role] = identity
	defer delete(supportedRoles, role)

	binding := fixtureGenericBinding(t, role)
	registry := fixtureRegistry(t, ProtocolV1, []Binding{binding})
	if _, err := registryCompatibility(registry); err == nil || !strings.Contains(err.Error(), "unsupported role") {
		t.Fatalf("future supported role silently widened the frozen v1 migration adapter: %v", err)
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

func fixtureRegistry(t *testing.T, protocol string, bindings []Binding) Registry {
	t.Helper()
	registry := Registry{
		Protocol: protocol, Scope: Scope, ProfileID: ProfileID, Generation: 1,
		UpdatedAt: "2026-08-10T16:34:56Z", Bindings: bindings, Canonical: false,
	}
	if protocol == ProtocolV2 {
		registry.FormatVersion = 2
		registry.PreviousRegistryProtocol = "absent"
	}
	digest, err := calculateDigest(registry)
	if err != nil {
		t.Fatal(err)
	}
	registry.RegistryDigest = digest
	return registry
}

func fixtureBinding(t *testing.T, role string) Binding {
	t.Helper()
	identity, ok := supportedRoles[role]
	if !ok {
		t.Fatalf("fixture requested unsupported role %q", role)
	}
	prefix := t.TempDir()
	return Binding{
		Role: role, ModuleID: identity.moduleID, EngineID: identity.engineID,
		Version: "0.1.0-dev", Prefix: prefix,
		ReceiptPath:      filepath.Join(prefix, "share", "receipt.json"),
		ReceiptDigest:    "sha256:" + strings.Repeat("1", 64),
		ExecutablePath:   filepath.Join(prefix, "libexec", identity.engineID),
		ExecutableDigest: "sha256:" + strings.Repeat("2", 64),
		State:            "bound_undocked", DefaultReceptor: nil,
	}
}

func fixtureGenericBinding(t *testing.T, role string) Binding {
	t.Helper()
	prefix := t.TempDir()
	return Binding{
		Role: role, ModuleID: role + "-engine", EngineID: "symphony-" + role,
		Version: "1.0.0", Prefix: prefix,
		ReceiptPath:      filepath.Join(prefix, "share", "receipt.json"),
		ReceiptDigest:    "sha256:" + strings.Repeat("3", 64),
		ExecutablePath:   filepath.Join(prefix, "libexec", "engine"),
		ExecutableDigest: "sha256:" + strings.Repeat("4", 64),
		State:            "bound_undocked", DefaultReceptor: nil,
	}
}

func writeRegistryFixture(t *testing.T, store *Store, registry Registry) {
	t.Helper()
	encoded, err := encodeRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.withStateLock(true, func(directory *os.File) error {
		return writeRegistry(directory, encoded)
	}); err != nil {
		t.Fatal(err)
	}
}

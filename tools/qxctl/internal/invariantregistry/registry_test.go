package invariantregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestLoadAndDeterministicQueryProjections(t *testing.T) {
	repository := repositoryFixture(t)
	registry, err := Load(repository)
	if err != nil {
		t.Fatal(err)
	}
	if (registry.Protocol != Protocol && registry.Protocol != ProtocolV2 && registry.Protocol != ProtocolV3) || len(registry.Invariants) == 0 || len(registry.Adapters) == 0 {
		t.Fatalf("registry identity or counts = %#v", registry)
	}

	status, err := Status(registry)
	if err != nil {
		t.Fatal(err)
	}
	if status.Protocol != QueryProtocol || status.FormatVersion != 1 || status.Operation != "status" ||
		status.SemanticValidity != "not_asserted" || status.ConsumerCheck != "identity_shape_digest_passed" ||
		status.CompleteCheckCommandID != "qxcmd:symphony:knowledge.invariant.check" ||
		status.ResultDigest != queryDigest(t, status) {
		t.Fatalf("status projection = %#v", status)
	}
	list, err := List(registry)
	if err != nil {
		t.Fatal(err)
	}
	if list.Operation != "list" || len(list.Invariants) != len(registry.Invariants) ||
		list.ResultDigest != queryDigest(t, list) {
		t.Fatalf("list projection = %#v", list)
	}
	show, err := Show(registry, registry.Invariants[0].InvariantID)
	if err != nil {
		t.Fatal(err)
	}
	if show.Operation != "show" || show.Invariant.InvariantID != registry.Invariants[0].InvariantID ||
		show.ResultDigest != queryDigest(t, show) {
		t.Fatalf("show projection = %#v", show)
	}
	if _, err := Show(registry, "invariant:symphony:missing.item"); err == nil {
		t.Fatal("unregistered invariant unexpectedly projected")
	}
}

func TestLoadRejectsTrailingValuesAndGarbage(t *testing.T) {
	data := canonicalRegistryBytes(t)
	for name, suffix := range map[string]string{
		"valid value": "\n{}\n",
		"garbage":     "\nnot-json",
	} {
		t.Run(name, func(t *testing.T) {
			repository := writeRepository(t, append(append([]byte(nil), data...), []byte(suffix)...))
			if _, err := Load(repository); err == nil || !strings.Contains(err.Error(), "invalid repository JSON") {
				t.Fatalf("trailing input error = %v", err)
			}
		})
	}
}

func TestGenericEngineAdapterRequiresV2AndPreservesV1(t *testing.T) {
	adapter := Adapter{AdapterID: "adapter:symphony:symphony-schv-aws.v1", Component: "schv-aws-engine", EntryPointID: "symphony-schv-aws",
		CommandProtocol: "symphony.knowledge.engine-process.v1", FormatVersion: 2, OwnerContract: "modules/schv-aws-engine/SPEC.md",
		ImplementationPath: "tools/qxctl/internal/knowledgeengine/scv.go", VersionPolicy: "exact_receipt_v2_entry_point_and_capability_compatible",
		OperationIDs: []string{"engop:symphony:schv-aws.inspect"}}
	if err := validateAdapterVersion(adapter, 2); err != nil {
		t.Fatal(err)
	}
	if err := validateAdapterVersion(adapter, 1); err == nil {
		t.Fatal("v2 generic adapter widened legacy v1")
	}
	adapter.FormatVersion = 1
	if err := validateAdapterVersion(adapter, 2); err == nil {
		t.Fatal("unrecognized legacy pair accepted in v2")
	}
	adapter.FormatVersion = 2
	adapter.CommandProtocol = "invented.protocol.v1"
	if err := validateAdapterVersion(adapter, 2); err == nil {
		t.Fatal("uncontracted process protocol accepted")
	}
	for _, legacy := range []Adapter{
		{AdapterID: "adapter:symphony:ssiag.foundation-lifecycle.v1", Component: "ssiag", EntryPointID: "ssiag.foundation-lifecycle", CommandProtocol: "symphony.foundation.lifecycle-command.v1", FormatVersion: 1, OwnerContract: "knowledge/ssiag/SPEC.md", ImplementationPath: "tools/qxctl/cmd/qxctl/foundation_lifecycle.go", VersionPolicy: "exact_receipt_v2_entry_point_and_capability_compatible", OperationIDs: []string{"engop:symphony:ssiag.enrollment.apply"}},
	} {
		if err := validateAdapterVersion(legacy, 1); err != nil {
			t.Fatal(err)
		}
		if err := validateAdapterVersion(legacy, 2); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadRejectsControlTextAndDigestTampering(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"hostile title": func(object map[string]any) {
			object["invariants"].([]any)[0].(map[string]any)["title"] = "hostile\u001b[31m"
		},
		"hostile statement": func(object map[string]any) {
			object["invariants"].([]any)[0].(map[string]any)["statement"] = "line one\nline two"
		},
	} {
		t.Run(name, func(t *testing.T) {
			object := canonicalRegistryObject(t)
			mutate(object)
			repository := writeRepository(t, encodeRegistry(t, object))
			if _, err := Load(repository); err == nil || !strings.Contains(err.Error(), "shape or required evidence") {
				t.Fatalf("hostile text error = %v", err)
			}
		})
	}
	data := canonicalRegistryBytes(t)
	data = []byte(strings.Replace(string(data), "sha256:", "sha256:0", 1))
	if _, err := Load(writeRepository(t, data)); err == nil {
		t.Fatal("digest tampering unexpectedly accepted")
	}
}

func TestLoadRejectsInventedMacOSProviderOperationIdentity(t *testing.T) {
	object := canonicalRegistryObject(t)
	adapters := object["adapters"].([]any)
	for _, candidate := range adapters {
		adapter := candidate.(map[string]any)
		if adapter["adapter_id"] == "adapter:symphony:ssiag.macos-keychain-provider.v1" {
			adapter["operation_ids"] = []any{"engop:symphony:ssiag.provider.metadata-invented"}
		}
	}
	repository := writeRepository(t, encodeRegistry(t, object))
	if _, err := Load(repository); err == nil || !strings.Contains(err.Error(), "adapter-owned operation set") {
		t.Fatalf("invented adapter operation error = %v", err)
	}
}

func TestLoadRejectsSymlinkedRegistry(t *testing.T) {
	repository := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repository, "knowledge"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(target, canonicalRegistryBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(repository, RegistryPath)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Load(repository); err == nil || !strings.Contains(err.Error(), "no-follow") {
		t.Fatalf("symlink error = %v", err)
	}
}

func repositoryFixture(t *testing.T) string {
	t.Helper()
	return writeRepository(t, canonicalRegistryBytes(t))
}

func writeRepository(t *testing.T, data []byte) string {
	t.Helper()
	repository := t.TempDir()
	path := filepath.Join(repository, RegistryPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return repository
}

func canonicalRegistryBytes(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", RegistryPath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func canonicalRegistryObject(t *testing.T) map[string]any {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(canonicalRegistryBytes(t), &object); err != nil {
		t.Fatal(err)
	}
	return object
}

func encodeRegistry(t *testing.T, object map[string]any) []byte {
	t.Helper()
	delete(object, "registry_digest")
	canonical, err := canonicalJSON(object)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	object["registry_digest"] = "sha256:" + hex.EncodeToString(digest[:])
	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func queryDigest(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	delete(object, "result_digest")
	canonical, err := canonicalJSON(object)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func TestV3ExplicitAdapterIdentityAndFrozenOlderContracts(t *testing.T) {
	adapter := Adapter{AdapterID: "adapter:symphony:symphony-index-worker.v1", Component: "storage-adapter",
		EntryPointID: "symphony-index-worker", CommandProtocol: "symphony.knowledge.engine-process.v1",
		FormatVersion: 3, OwnerContract: "modules/storage-adapter/SPEC.md", ImplementationPath: "modules/storage-adapter",
		VersionPolicy: "exact_receipt_v2_entry_point_and_capability_compatible", OperationIDs: []string{"engop:symphony:scv.graph-index.query"}}
	if err := validateAdapterVersion(adapter, 3); err != nil {
		t.Fatal(err)
	}
	for _, version := range []uint64{1, 2} {
		if err := validateAdapterVersion(adapter, version); err == nil {
			t.Fatal("explicit adapter widened old registry", version)
		}
	}
	for name, mutate := range map[string]func(*Adapter){
		"long-component": func(a *Adapter) {
			a.Component = strings.Repeat("a", 257)
			a.OwnerContract = "modules/" + a.Component + "/SPEC.md"
			a.ImplementationPath = "modules/" + a.Component
		},
		"uppercase-component": func(a *Adapter) {
			a.Component = "Storage-adapter"
			a.OwnerContract = "modules/" + a.Component + "/SPEC.md"
			a.ImplementationPath = "modules/" + a.Component
		},
		"owner":                func(a *Adapter) { a.OwnerContract = "modules/other/SPEC.md" },
		"path":                 func(a *Adapter) { a.ImplementationPath = "modules/storage-adapter/src" },
		"entrypoint":           func(a *Adapter) { a.EntryPointID = "symphony-other" },
		"protocol":             func(a *Adapter) { a.CommandProtocol = "other.v1" },
		"frozen-v2":            func(a *Adapter) { a.FormatVersion = 2 },
		"duplicate-operations": func(a *Adapter) { a.OperationIDs = []string{"engop:symphony:a.query", "engop:symphony:a.query"} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := adapter
			mutate(&changed)
			if validateAdapterVersion(changed, 3) == nil {
				t.Fatal("malformed explicit adapter accepted")
			}
		})
	}
}

func TestV3ResealedRegistryRejectsCrossAdapterOperationOwnership(t *testing.T) {
	object := canonicalRegistryObject(t)
	object["protocol"], object["format_version"] = ProtocolV3, 3
	adapters := object["adapters"].([]any)
	existing := adapters[0].(map[string]any)["operation_ids"].([]any)[0]
	adapters = append(adapters, map[string]any{"adapter_id": "adapter:symphony:zz-index-worker.v1", "component": "storage-adapter",
		"entry_point_id": "zz-index-worker", "command_protocol": "symphony.knowledge.engine-process.v1", "format_version": 3,
		"owner_contract": "modules/storage-adapter/SPEC.md", "implementation_path": "modules/storage-adapter",
		"version_policy": "exact_receipt_v2_entry_point_and_capability_compatible", "operation_ids": []any{existing}})
	// Use a valid, sorted entrypoint so duplicate operation ownership is the rejection.
	added := adapters[len(adapters)-1].(map[string]any)
	added["adapter_id"] = "adapter:symphony:symphony-zz-index-worker.v1"
	added["entry_point_id"] = "symphony-zz-index-worker"
	sort.Slice(adapters, func(i, j int) bool {
		return adapters[i].(map[string]any)["adapter_id"].(string) < adapters[j].(map[string]any)["adapter_id"].(string)
	})
	object["adapters"] = adapters
	_, err := Load(writeRepository(t, encodeRegistry(t, object)))
	if err == nil || !strings.Contains(err.Error(), "multiple adapters") {
		t.Fatalf("duplicate operation ownership: %v", err)
	}
}

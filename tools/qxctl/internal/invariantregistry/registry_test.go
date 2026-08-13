package invariantregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndDeterministicQueryProjections(t *testing.T) {
	repository := repositoryFixture(t)
	registry, err := Load(repository)
	if err != nil {
		t.Fatal(err)
	}
	if registry.Protocol != Protocol || len(registry.Invariants) == 0 || len(registry.Adapters) == 0 {
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

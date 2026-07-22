package knowledgeengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateJSONObjectRejectsAmbiguousAndUnboundedSyntax(t *testing.T) {
	for name, data := range map[string]string{
		"duplicate":      `{"a":1,"a":2}`,
		"float":          `{"a":1.5}`,
		"trailing":       `{} {}`,
		"array root":     `[]`,
		"unsafe integer": `{"a":9007199254740992}`,
		"invalid utf8":   "{\"a\":\"\xff\"}",
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateJSONObject([]byte(data), maxRequestBytes); err == nil {
				t.Fatalf("accepted invalid JSON: %s", data)
			}
		})
	}
	if err := validateJSONObject([]byte(`{"items":[{"name":"first"},{"name":"second"}]}`), maxRequestBytes); err != nil {
		t.Fatalf("rejected valid bounded JSON: %v", err)
	}
}

func TestResolveInstalledRequiresExactReceiptAndNoFollowFiles(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	if _, err := resolveInstalled(prefix, ".."); err == nil {
		t.Fatal("traversal-like engine version was accepted")
	}
	receiptPath, document := createInstalledFixture(t, prefix, "fixture\n")
	binary, err := resolveInstalled(prefix, version)
	if err != nil {
		t.Fatalf("valid installation rejected: %v", err)
	}
	canonicalPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(canonicalPrefix, "libexec/symphony/skvi-engine", version, "symphony-skvi"); binary != want {
		t.Fatalf("binary = %q, want %q", binary, want)
	}

	document["unexpected"] = true
	data, _ := json.Marshal(document)
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInstalled(prefix, version); err == nil {
		t.Fatal("receipt with unknown field was accepted")
	}
	delete(document, "unexpected")
	data, _ = json.Marshal(document)
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(prefix, "share/doc/symphony/skvi-engine", version, "INTENT.md")
	if err := os.Remove(docPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(prefix, "share/doc/symphony/skvi-engine", version, "SPEC.md"), docPath); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInstalled(prefix, version); err == nil {
		t.Fatalf("symlinked receipt-owned file was not rejected: %v", err)
	}
}

func TestResolveInstalledSCLVRequiresExactElevenFileReceipt(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	files := expectedSCLVFiles(version)
	for relative := range files {
		path := filepath.Join(prefix, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(relative, "libexec/") {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte("fixture\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	receiptPath := filepath.Join(prefix, "share/symphony/receipts/sclv-engine", version, "install-receipt.json")
	listed := make([]string, 0, len(files))
	for relative := range files {
		listed = append(listed, relative)
	}
	document := map[string]any{
		"protocol": receiptProtocol, "module_id": sclvModuleID, "version": version,
		"install_scope": "prefix", "prefix_mode": "installation_prefix",
		"state": "installed_undocked", "active": false, "default_receptor": nil,
		"files": listed,
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	binary, err := resolveInstalledFor(sclvSpec, prefix, version)
	if err != nil {
		t.Fatalf("valid SCLV installation rejected: %v", err)
	}
	canonicalPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(canonicalPrefix, "libexec/symphony/sclv-engine", version, "symphony-sclv"); binary != want {
		t.Fatalf("binary = %q, want %q", binary, want)
	}

	delete(files, "libexec/symphony/sclv-engine/"+version+"/symphony-sclv-evidence-airgap")
	document["files"] = func() []string {
		values := make([]string, 0, len(files))
		for relative := range files {
			values = append(values, relative)
		}
		return values
	}()
	data, _ = json.Marshal(document)
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInstalledFor(sclvSpec, prefix, version); err == nil {
		t.Fatal("SCLV receipt missing an adapter was accepted")
	}
}

func TestResolveInstalledSACVRequiresExactNineFileReceipt(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	files := expectedSACVFiles(version)
	for relative := range files {
		path := filepath.Join(prefix, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(relative, "libexec/") {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte("fixture\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	receiptPath := filepath.Join(prefix, "share/symphony/receipts/sacv-engine", version, "install-receipt.json")
	listed := make([]string, 0, len(files))
	for relative := range files {
		listed = append(listed, relative)
	}
	document := map[string]any{
		"protocol": receiptProtocol, "module_id": sacvModuleID, "version": version,
		"install_scope": "prefix", "prefix_mode": "installation_prefix",
		"state": "installed_undocked", "active": false, "default_receptor": nil,
		"files": listed,
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	binary, err := resolveInstalledFor(sacvSpec, prefix, version)
	if err != nil {
		t.Fatalf("valid SACV installation rejected: %v", err)
	}
	canonicalPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(canonicalPrefix, "libexec/symphony/sacv-engine", version, "symphony-sacv"); binary != want {
		t.Fatalf("binary = %q, want %q", binary, want)
	}
}

func TestResolveInstalledSODVRequiresExactNineFileReceipt(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	files := expectedSODVFiles(version)
	for relative := range files {
		path := filepath.Join(prefix, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(relative, "libexec/") {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte("fixture\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	receiptPath := filepath.Join(prefix, "share/symphony/receipts/sodv-engine", version, "install-receipt.json")
	listed := make([]string, 0, len(files))
	for relative := range files {
		listed = append(listed, relative)
	}
	document := map[string]any{
		"protocol": receiptProtocol, "module_id": sodvModuleID, "version": version,
		"install_scope": "prefix", "prefix_mode": "installation_prefix",
		"state": "installed_undocked", "active": false, "default_receptor": nil,
		"files": listed,
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	binary, err := resolveInstalledFor(sodvSpec, prefix, version)
	if err != nil {
		t.Fatalf("valid SODV installation rejected: %v", err)
	}
	canonicalPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(canonicalPrefix, "libexec/symphony/sodv-engine", version, "symphony-sodv"); binary != want {
		t.Fatalf("binary = %q, want %q", binary, want)
	}

	delete(files, "share/doc/symphony/sodv-engine/"+version+"/SPEC.md")
	listed = listed[:0]
	for relative := range files {
		listed = append(listed, relative)
	}
	document["files"] = listed
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInstalledFor(sodvSpec, prefix, version); err == nil {
		t.Fatal("SODV receipt missing a required documentation file was accepted")
	}
}

func TestInvokeEnforcesCallerDeadlineAroundChildProcess(t *testing.T) {
	prefix := t.TempDir()
	createInstalledFixture(t, prefix, "#!/bin/sh\n/bin/sleep 10\n")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, err := Invoke(ctx, prefix, "0.1.0-dev", t.TempDir(), "inspect", []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "hard process deadline") {
		t.Fatalf("blocked child did not fail through the hard deadline: %v", err)
	}
}

func createInstalledFixture(t *testing.T, prefix, binaryContents string) (string, map[string]any) {
	t.Helper()
	version := "0.1.0-dev"
	for relative := range expectedFiles(version) {
		path := filepath.Join(prefix, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		contents := "fixture\n"
		if strings.HasSuffix(relative, "/symphony-skvi") {
			mode = 0o755
			contents = binaryContents
		}
		if err := os.WriteFile(path, []byte(contents), mode); err != nil {
			t.Fatal(err)
		}
	}
	receiptPath := filepath.Join(prefix, "share/symphony/receipts/skvi-engine", version, "install-receipt.json")
	files := make([]string, 0, len(expectedFiles(version)))
	for relative := range expectedFiles(version) {
		files = append(files, relative)
	}
	document := map[string]any{
		"protocol":         receiptProtocol,
		"module_id":        moduleID,
		"version":          version,
		"install_scope":    "prefix",
		"prefix_mode":      "installation_prefix",
		"state":            "installed_undocked",
		"active":           false,
		"default_receptor": nil,
		"files":            files,
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return receiptPath, document
}

func TestValidateResponseBindsIdentityAndDigest(t *testing.T) {
	object := map[string]any{
		"protocol":       processProtocol,
		"request_id":     "request-1",
		"correlation_id": "request-1",
		"operation":      "inspect",
		"engine_id":      engineID,
		"engine_version": "0.1.0-dev",
		"outcome":        "ok",
		"result":         map[string]any{"detail": "<trusted>&bounded", "ready": true},
		"error":          nil,
	}
	canonical, err := marshalCanonical(object)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	object["response_digest"] = "sha256:" + hex.EncodeToString(digest[:])
	data, err := marshalCanonical(object)
	if err != nil {
		t.Fatal(err)
	}
	response, err := validateResponse(data, "request-1", "inspect", "0.1.0-dev")
	if err != nil {
		t.Fatalf("valid response rejected: %v", err)
	}
	if response.Outcome != "ok" {
		t.Fatalf("outcome = %q", response.Outcome)
	}

	object["engine_version"] = "0.2.0"
	tampered, _ := marshalCanonical(object)
	if _, err := validateResponse(tampered, "request-1", "inspect", "0.1.0-dev"); err == nil {
		t.Fatal("tampered response was accepted")
	}
}

func TestReadPayloadRejectsSymlinkAndDuplicateKeys(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "payload.json")
	if err := os.WriteFile(target, []byte(`{"repository":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPayload(target); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if err := os.WriteFile(target, []byte(`{"a":1,"a":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPayload(target); err == nil {
		t.Fatal("duplicate-key payload accepted")
	}
	link := filepath.Join(directory, "payload-link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPayload(link); err == nil {
		t.Fatal("symlink payload accepted")
	}
	if err := os.WriteFile(target, []byte(`{"repository":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	parentLink := filepath.Join(t.TempDir(), "linked-parent")
	if err := os.Symlink(directory, parentLink); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPayload(filepath.Join(parentLink, "payload.json")); err != nil {
		t.Fatalf("canonical parent alias was not resolved safely: %v", err)
	}
}

func TestSafeRelativePathAndTokens(t *testing.T) {
	for _, path := range []string{"../file", "/absolute", "a//b", "a\\b", "a/./b"} {
		if safeRelativePath(path) {
			t.Fatalf("unsafe path accepted: %q", path)
		}
	}
	if !safeRelativePath("share/symphony/file.json") {
		t.Fatal("safe relative path rejected")
	}
	if !safeToken("operation-1", 64) || safeToken("not safe", 64) {
		t.Fatal("token validation mismatch")
	}
	if !safeVersion("0.1.0-dev") || safeVersion("..") || safeVersion("version_with_underscore") {
		t.Fatal("version validation mismatch")
	}
}

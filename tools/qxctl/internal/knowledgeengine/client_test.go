package knowledgeengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

func TestInspectValidatorInstallationRequiresExactNineFileReceipt(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	files := expectedValidatorFiles(version)
	listed := make([]string, 0, len(files))
	for relative := range files {
		listed = append(listed, relative)
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
	receiptPath := filepath.Join(prefix, "share/symphony/receipts/symphony-validator", version, "install-receipt.json")
	document := map[string]any{
		"protocol": receiptProtocol, "module_id": validatorModuleID, "version": version,
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
	installation, err := InspectValidatorInstallation(prefix, version)
	if err != nil {
		t.Fatalf("valid validator installation rejected: %v", err)
	}
	if installation.Role != "validator" || installation.ModuleID != validatorModuleID ||
		installation.EngineID != validatorEngineID || !strings.HasPrefix(installation.ReceiptDigest, "sha256:") {
		t.Fatalf("unexpected validator installation: %+v", installation)
	}
	document["files"] = listed[:len(listed)-1]
	data, _ = json.Marshal(document)
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectValidatorInstallation(prefix, version); err == nil {
		t.Fatal("validator receipt with a missing file was accepted")
	}
}

func TestResolveInstalledSSFVRequiresExactNineFileReceipt(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	files := expectedSSFVFiles(version)
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
	receiptPath := filepath.Join(
		prefix, "share/symphony/receipts/ssfv-engine", version, "install-receipt.json")
	listed := make([]string, 0, len(files))
	for relative := range files {
		listed = append(listed, relative)
	}
	document := map[string]any{
		"protocol": receiptProtocol, "module_id": ssfvModuleID, "version": version,
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
	binary, err := resolveInstalledFor(ssfvSpec, prefix, version)
	if err != nil {
		t.Fatalf("valid SSFV installation rejected: %v", err)
	}
	canonicalPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(
		canonicalPrefix, "libexec/symphony/ssfv-engine", version, "symphony-ssfv"); binary != want {
		t.Fatalf("binary = %q, want %q", binary, want)
	}

	document["active"] = true
	data, _ = json.Marshal(document)
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInstalledFor(ssfvSpec, prefix, version); err == nil {
		t.Fatal("active SSFV receipt was accepted")
	}
	document["active"] = false
	data, _ = json.Marshal(document)
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	untrustedDirectory := filepath.Join(prefix, "share", "doc", "symphony", "ssfv-engine")
	if err := os.Chmod(untrustedDirectory, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInstalledFor(ssfvSpec, prefix, version); err == nil {
		t.Fatal("group/world-writable SSFV installed path component was accepted")
	}
	if err := os.Chmod(untrustedDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(prefix, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInstalledFor(ssfvSpec, prefix, version); err == nil {
		t.Fatal("group/world-writable SSFV installation prefix was accepted")
	}
}

func TestInspectInstallationSupportsExactCoordinatorReceipt(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	files := expectedSessionFiles(version)
	listed := make([]string, 0, len(files))
	for relative := range files {
		listed = append(listed, relative)
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
	document := map[string]any{
		"protocol": receiptProtocol, "module_id": sessionModuleID, "version": version,
		"install_scope": "prefix", "prefix_mode": "installation_prefix",
		"state": "installed_undocked", "active": false, "default_receptor": nil,
		"files": listed,
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(
		prefix, "share/symphony/receipts/knowledge-session-coordinator",
		version, "install-receipt.json")
	if err := os.WriteFile(receiptPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	installed, err := InspectInstallation("coordinator", prefix, version)
	if err != nil {
		t.Fatalf("valid coordinator installation rejected: %v", err)
	}
	if installed.ModuleID != sessionModuleID || installed.EngineID != sessionEngineID ||
		!taggedDigest(installed.ReceiptDigest) || !taggedDigest(installed.ExecutableDigest) {
		t.Fatalf("unexpected coordinator installation evidence: %+v", installed)
	}
}

func TestInspectInstallationSupportsImmutableReceiptV2AndDetectsTampering(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	receiptPath, document := createInstalledV2Fixture(t, skviSpec, prefix, version)
	installed, err := InspectInstallation("skvi", prefix, version)
	if err != nil {
		t.Fatalf("valid receipt-v2 installation rejected: %v", err)
	}
	if installed.ReceiptProtocol != receiptProtocolV2 || installed.ReceiptDigest != document.ReceiptDigest {
		t.Fatalf("receipt-v2 identity was not preserved: %+v", installed)
	}

	binaryPath := filepath.Join(prefix, filepath.FromSlash(
		"libexec/symphony/skvi-engine/"+version+"/symphony-skvi"))
	if err := os.WriteFile(binaryPath, []byte("tampered\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectInstallation("skvi", prefix, version); err == nil {
		t.Fatal("receipt-v2 content tampering was accepted")
	}

	createInstalledV2Fixture(t, skviSpec, prefix, version)
	document.ReceiptDigest = taggedDigestForTest("forged-receipt")
	encoded, _ := json.Marshal(document)
	if err := os.WriteFile(receiptPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectInstallation("skvi", prefix, version); err == nil {
		t.Fatal("forged intrinsic receipt-v2 digest was accepted")
	}
}

func TestResolveSCLVEvidenceAdapterRequiresExactTypedReceiptEntryPoint(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	receiptPath, document := createInstalledV2Fixture(t, sclvSpec, prefix, version)
	localGitRelative := filepath.ToSlash(filepath.Join(
		"libexec", "symphony", sclvModuleID, version, sclvLocalGitID))
	airgapRelative := filepath.ToSlash(filepath.Join(
		"libexec", "symphony", sclvModuleID, version, sclvAirgapID))
	document.EntryPoints = append(document.EntryPoints,
		receiptV2EntryPoint{
			EntryPointID: sclvLocalGitID, Kind: "adapter", Path: localGitRelative,
			Protocols: []string{providerEvidenceProtocol},
		},
		receiptV2EntryPoint{
			EntryPointID: sclvAirgapID, Kind: "adapter", Path: airgapRelative,
			Protocols: []string{providerEvidenceProtocol},
		},
	)
	writeReceiptV2Fixture(t, receiptPath, &document)
	resolved, err := resolveSCLVEvidenceAdapter(prefix, version, sclvLocalGitID)
	if err != nil {
		t.Fatalf("valid receipt-owned SCLV evidence adapter rejected: %v", err)
	}
	canonicalPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(canonicalPrefix, filepath.FromSlash(localGitRelative)); resolved != want {
		t.Fatalf("adapter = %q, want %q", resolved, want)
	}
	if err := os.WriteFile(resolved, []byte("tampered adapter\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveSCLVEvidenceAdapter(prefix, version, sclvLocalGitID); err == nil {
		t.Fatal("content-tampered SCLV evidence adapter was accepted")
	}
	if err := os.WriteFile(resolved, []byte(localGitRelative+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	document.EntryPoints[1].Protocols = []string{processProtocol}
	writeReceiptV2Fixture(t, receiptPath, &document)
	if _, err := resolveSCLVEvidenceAdapter(prefix, version, sclvLocalGitID); err == nil {
		t.Fatal("SCLV evidence adapter with the wrong typed protocol was accepted")
	}

	legacyPrefix := t.TempDir()
	legacyFiles := expectedSCLVFiles(version)
	listed := make([]string, 0, len(legacyFiles))
	for relative := range legacyFiles {
		listed = append(listed, relative)
		path := filepath.Join(legacyPrefix, filepath.FromSlash(relative))
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
	legacyReceipt := map[string]any{
		"protocol": receiptProtocol, "module_id": sclvModuleID, "version": version,
		"install_scope": "prefix", "prefix_mode": "installation_prefix",
		"state": "installed_undocked", "active": false, "default_receptor": nil,
		"files": listed,
	}
	encoded, err := json.Marshal(legacyReceipt)
	if err != nil {
		t.Fatal(err)
	}
	legacyReceiptPath := filepath.Join(
		legacyPrefix, "share/symphony/receipts/sclv-engine", version, "install-receipt.json")
	if err := os.WriteFile(legacyReceiptPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveSCLVEvidenceAdapter(legacyPrefix, version, sclvLocalGitID); err == nil {
		t.Fatal("legacy receipt without a typed adapter entry point was accepted")
	}
}

func TestReceiptV2SupportsForwardFileGrowthAndRejectsSemanticDrift(t *testing.T) {
	prefix := t.TempDir()
	version := "0.1.0-dev"
	receiptPath, document := createInstalledV2Fixture(t, skviSpec, prefix, version)
	extraPath := "share/doc/symphony/skvi-engine/" + version + "/FORWARD-COMPATIBILITY.md"
	extra := []byte("future package-owned evidence\n")
	absoluteExtra := filepath.Join(prefix, filepath.FromSlash(extraPath))
	if err := os.WriteFile(absoluteExtra, extra, 0o644); err != nil {
		t.Fatal(err)
	}
	extraDigest := sha256.Sum256(extra)
	document.Files = append(document.Files, receiptV2File{
		Path: extraPath, Kind: "regular", Size: uint64(len(extra)),
		Digest: "sha256:" + hex.EncodeToString(extraDigest[:]),
	})
	writeReceiptV2Fixture(t, receiptPath, &document)
	if _, err := InspectInstallation("skvi", prefix, version); err != nil {
		t.Fatalf("forward-compatible receipt-owned file was rejected: %v", err)
	}

	document.ComponentKind = "module"
	writeReceiptV2Fixture(t, receiptPath, &document)
	if _, err := InspectInstallation("skvi", prefix, version); err == nil {
		t.Fatal("receipt-v2 component-kind drift was accepted")
	}
	document.ComponentKind = "vector_engine"
	document.VectorID = nil
	writeReceiptV2Fixture(t, receiptPath, &document)
	if _, err := InspectInstallation("skvi", prefix, version); err == nil {
		t.Fatal("receipt-v2 vector identity drift was accepted")
	}
	vectorID := "skvi"
	document.VectorID = &vectorID
	originalPlatform := append([]receiptV2Platform(nil), document.PlatformRequirements...)
	document.PlatformRequirements = []receiptV2Platform{}
	writeReceiptV2Fixture(t, receiptPath, &document)
	if _, err := InspectInstallation("skvi", prefix, version); err == nil {
		t.Fatal("receipt-v2 without a critical host platform was accepted")
	}
	document.PlatformRequirements = originalPlatform
	document.CompatibleReceptors = append(document.CompatibleReceptors, document.CompatibleReceptors[0])
	writeReceiptV2Fixture(t, receiptPath, &document)
	if _, err := InspectInstallation("skvi", prefix, version); err == nil {
		t.Fatal("receipt-v2 duplicate receptor was accepted")
	}
	document.CompatibleReceptors = document.CompatibleReceptors[:1]
	document.EntryPoints = append(document.EntryPoints, receiptV2EntryPoint{
		EntryPointID: "unowned-descriptor", Kind: "descriptor",
		Path: "share/doc/unowned.json", Protocols: []string{},
	})
	writeReceiptV2Fixture(t, receiptPath, &document)
	if _, err := InspectInstallation("skvi", prefix, version); err == nil {
		t.Fatal("receipt-v2 unowned entry point was accepted")
	}
}

func TestInspectValidatorInstallationSupportsReceiptV2Protocol(t *testing.T) {
	prefix := t.TempDir()
	_, document := createInstalledV2Fixture(t, validatorSpec, prefix, "0.1.0-dev")
	installed, err := InspectValidatorInstallation(prefix, "0.1.0-dev")
	if err != nil {
		t.Fatalf("valid validator receipt-v2 installation rejected: %v", err)
	}
	if installed.ReceiptProtocol != receiptProtocolV2 || installed.ReceiptDigest != document.ReceiptDigest ||
		installed.ModuleID != validatorModuleID || installed.EngineID != validatorEngineID {
		t.Fatalf("unexpected validator receipt-v2 identity: %+v", installed)
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

func createInstalledV2Fixture(t *testing.T, spec engineSpec, prefix, version string) (string, receiptV2) {
	t.Helper()
	receiptRelative := filepath.ToSlash(filepath.Join(
		"share", "symphony", "receipts", spec.moduleID, version, "install-receipt.json"))
	files := make([]receiptV2File, 0)
	for relative := range spec.expectedFiles(version) {
		if relative == receiptRelative {
			continue
		}
		path := filepath.Join(prefix, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		kind := "regular"
		if strings.HasPrefix(relative, "libexec/") {
			mode = 0o755
			kind = "executable"
		}
		contents := []byte(relative + "\n")
		if err := os.WriteFile(path, contents, mode); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(contents)
		files = append(files, receiptV2File{
			Path: relative, Kind: kind, Size: uint64(len(contents)),
			Digest: "sha256:" + hex.EncodeToString(digest[:]),
		})
	}
	binaryRelative := filepath.ToSlash(filepath.Join(
		"libexec", "symphony", spec.moduleID, version, spec.engineID))
	engine := spec.engineID
	vector := spec.vectorID
	var vectorID *string
	if vector != "" {
		vectorID = &vector
	}
	platformOS := runtime.GOOS
	if platformOS == "darwin" {
		platformOS = "macos"
	}
	document := receiptV2{
		Protocol: receiptProtocolV2, FormatVersion: 2,
		ComponentID: spec.moduleID, ComponentKind: spec.componentKind, ModuleID: spec.moduleID,
		VectorID: vectorID, EngineID: &engine, PackageID: spec.moduleID, Version: version,
		InstallScope: "prefix", PrefixMode: "installation_prefix", Files: files,
		EntryPoints: []receiptV2EntryPoint{{
			EntryPointID: spec.engineID, Kind: "executable", Path: binaryRelative,
			Protocols: []string{spec.processProtocol},
		}},
		ProvidesCapabilities: []string{}, RequiresCapabilities: []string{},
		CompatibleReceptors: append([]string{}, spec.requiredReceptors...),
		PlatformRequirements: []receiptV2Platform{{
			OS: platformOS, Architecture: runtime.GOARCH, KernelABI: nil, Critical: true,
		}},
	}
	receiptPath := filepath.Join(prefix, filepath.FromSlash(receiptRelative))
	writeReceiptV2Fixture(t, receiptPath, &document)
	return receiptPath, document
}

func writeReceiptV2Fixture(t *testing.T, receiptPath string, document *receiptV2) {
	t.Helper()
	document.ReceiptDigest = ""
	encoded, _ := json.Marshal(document)
	var object map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		t.Fatal(err)
	}
	delete(object, "receipt_digest")
	canonical, _ := json.Marshal(object)
	digest := sha256.Sum256(canonical)
	document.ReceiptDigest = "sha256:" + hex.EncodeToString(digest[:])
	encoded, _ = json.Marshal(document)
	if err := os.MkdirAll(filepath.Dir(receiptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

func taggedDigestForTest(seed string) string {
	digest := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(digest[:])
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

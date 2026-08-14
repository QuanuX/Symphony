package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const trustTestTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

type fakeLauncher struct {
	response ControlResponse
	err      error
	request  ControlRequest
	path     string
}

type blockingLauncher struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingLauncher) Exchange(_ context.Context, _ ExecutableTrust, _ ControlRequest) (ControlResponse, error) {
	b.once.Do(func() { close(b.entered) })
	<-b.release
	return ControlResponse{}, context.Canceled
}

func (f *fakeLauncher) Exchange(_ context.Context, declaration ExecutableTrust, request ControlRequest) (ControlResponse, error) {
	f.path, f.request = declaration.ExecutablePath, request
	return f.response, f.err
}

func TestProviderTrustUnboundAndDisabledRemainTruthful(t *testing.T) {
	manager := testManager(t, true, &fakeLauncher{})
	result, found := manager.Show("native")
	if !found || result.Operation != "show" || result.TrustState != "unbound" || result.AdapterIdentifier != "not_applicable" || !result.ReadOnly || result.Canonical || result.ResultDigest != objectDigest(result, "result_digest") {
		t.Fatalf("unexpected unbound result: %+v", result)
	}
	manager = testManager(t, false, &fakeLauncher{})
	result, found = manager.Verify(context.Background(), "native", trustUUID(3), trustUUID(4))
	if !found || result.Operation != "verify" || result.TrustState != "disabled" || result.MutualTrust.FoundationVerifiedAdapter || result.MutualTrust.AdapterVerifiedFoundation {
		t.Fatalf("disabled provider claimed trust: %+v", result)
	}
}

func TestProviderTrustVerificationBindsReceiptProcessAndMutualHandshake(t *testing.T) {
	launcher := &fakeLauncher{}
	manager := testManager(t, true, launcher)
	declaration := writeTrustPackage(t, manager)
	requestID, correlationID := trustUUID(5), trustUUID(6)
	launcher.response = verifiedResponse(manager, declaration, requestID, correlationID)
	result, found := manager.Verify(context.Background(), "native", requestID, correlationID)
	if !found || result.TrustState != "verified" || !result.MutualTrust.FoundationVerifiedAdapter || !result.MutualTrust.AdapterVerifiedFoundation ||
		result.OperationalAccessEnabled || result.ProviderOperationsEnabled || result.SecretChannelEnabled || launcher.path != declaration.ExecutablePath {
		t.Fatalf("unexpected verified result: %+v", result)
	}
	if launcher.request.RequestDigest != objectDigest(launcher.request, "request_digest") || launcher.request.FoundationExecutableDigest != manager.foundation.ExecutableDigest ||
		launcher.request.OperationalAccessRequested || launcher.request.ProviderOperationRequested || launcher.request.SecretChannelRequested {
		t.Fatalf("unsafe or unbound control request: %+v", launcher.request)
	}
}

func TestProviderTrustRejectsInvokerOrResponseDrift(t *testing.T) {
	launcher := &fakeLauncher{}
	manager := testManager(t, true, launcher)
	declaration := writeTrustPackage(t, manager)
	requestID, correlationID := trustUUID(7), trustUUID(8)
	launcher.response = verifiedResponse(manager, declaration, requestID, correlationID)
	launcher.response.Handshake.FoundationTrust.ExecutableDigest = tagged("different-foundation")
	launcher.response.Handshake.HandshakeDigest = objectDigest(*launcher.response.Handshake, "handshake_digest")
	launcher.response.ResponseDigest = objectDigest(launcher.response, "response_digest")
	result, _ := manager.Verify(context.Background(), "native", requestID, correlationID)
	if result.TrustState != "mismatch" || result.MutualTrust.FoundationVerifiedAdapter || result.MutualTrust.AdapterVerifiedFoundation {
		t.Fatalf("drift did not fail closed: %+v", result)
	}
}

func TestProviderTrustRejectsNestedResponseStatusDrift(t *testing.T) {
	launcher := &fakeLauncher{}
	manager := testManager(t, true, launcher)
	declaration := writeTrustPackage(t, manager)
	requestID, correlationID := trustUUID(7), trustUUID(8)
	launcher.response = verifiedResponse(manager, declaration, requestID, correlationID)
	launcher.response.Status = "disabled"
	launcher.response.ResponseDigest = objectDigest(launcher.response, "response_digest")
	result, _ := manager.Verify(context.Background(), "native", requestID, correlationID)
	if result.TrustState != "mismatch" || result.MutualTrust.FoundationVerifiedAdapter || result.MutualTrust.AdapterVerifiedFoundation {
		t.Fatalf("outer and nested response status drift did not fail closed: %+v", result)
	}
}

func TestProviderTrustRejectsCompatibleButNonExactReceipt(t *testing.T) {
	manager := testManager(t, true, &fakeLauncher{})
	declaration := writeTrustPackage(t, manager)
	prefix := providerTestPrefix(declaration.ExecutablePath)
	receiptPath := filepath.Join(prefix, "share", "symphony", "receipts", macOSKeychainPackageID, declaration.AdapterVersion, "install-receipt.json")
	receipt, _, _, err := readAdapterReceipt(receiptPath, ssiagpaths.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	receipt.CompatibleReceptors = []string{}
	receipt.ReceiptDigest = receiptDigest(receipt)
	writeJSONFile(t, receiptPath, receipt)
	declaration.InstallationDigest = receipt.ReceiptDigest
	declaration.DeclarationDigest = objectDigest(declaration, "declaration_digest")
	writeJSONFile(t, filepath.Join(manager.layout.ProviderTrustDir, "native.json"), declaration)
	result, _ := manager.Show("native")
	if result.TrustState != "mismatch" {
		t.Fatalf("non-exact receipt was accepted: %+v", result)
	}
}

func TestProviderTrustRejectsWritablePackageAncestor(t *testing.T) {
	manager := testManager(t, true, &fakeLauncher{})
	declaration := writeTrustPackage(t, manager)
	prefix := providerTestPrefix(declaration.ExecutablePath)
	if err := os.Chmod(prefix, 0o777); err != nil {
		t.Fatal(err)
	}
	result, _ := manager.Show("native")
	if result.TrustState != "mismatch" {
		t.Fatalf("writable package ancestor was accepted: %+v", result)
	}
}

func TestProviderVerificationAdmissionIsBoundedPerProvider(t *testing.T) {
	launcher := &blockingLauncher{entered: make(chan struct{}), release: make(chan struct{})}
	manager := testManager(t, true, launcher)
	writeTrustPackage(t, manager)
	done := make(chan struct{})
	go func() {
		defer close(done)
		manager.Verify(context.Background(), "native", trustUUID(20), trustUUID(21))
	}()
	<-launcher.entered
	result, found := manager.Verify(context.Background(), "native", trustUUID(22), trustUUID(23))
	if !found || result.TrustState != "unavailable" || len(result.Checks) != 1 || result.Checks[0].ReasonCode != "symphony.ssiag.provider.busy" {
		t.Fatalf("concurrent provider verification was not shed deterministically: %+v", result)
	}
	close(launcher.release)
	<-done
}

func TestProviderVerificationAdmissionHasGlobalCeiling(t *testing.T) {
	manager := testManager(t, true, &fakeLauncher{})
	releases := make([]func(), 0, maximumConcurrentProviderVerifications)
	for index := 0; index < maximumConcurrentProviderVerifications; index++ {
		release, admitted := manager.acquireVerification(fmt.Sprintf("provider-%d", index))
		if !admitted {
			t.Fatalf("global admission closed before its declared ceiling at %d", index)
		}
		releases = append(releases, release)
	}
	if _, admitted := manager.acquireVerification("overflow"); admitted {
		t.Fatal("global admission accepted an unbounded provider process")
	}
	for _, release := range releases {
		release()
	}
}

func TestProviderTrustRequiresExplicitFalseMembers(t *testing.T) {
	manager := testManager(t, true, &fakeLauncher{})
	declaration := writeTrustPackage(t, manager)
	path := filepath.Join(manager.layout.ProviderTrustDir, "native.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatal(err)
	}
	delete(object, "secret_channel_enabled")
	delete(object, "declaration_digest")
	canonical, _ := json.Marshal(object)
	var normalized map[string]any
	_ = json.Unmarshal(canonical, &normalized)
	digest := sha256.Sum256(canonical)
	normalized["declaration_digest"] = "sha256:" + hex.EncodeToString(digest[:])
	writeJSONFile(t, path, normalized)
	result, _ := manager.Show("native")
	if result.TrustState != "mismatch" {
		t.Fatalf("omitted required false member was accepted: %+v declaration=%+v", result, declaration)
	}
}

func TestProviderExecutableSizeBoundFailsBeforeHashing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized-adapter")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o500)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maximumProviderExecutableBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := digestPath(path); err == nil {
		t.Fatal("oversized provider executable was hashed")
	}
}

func TestProcessLauncherRejectsExtraOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	path := filepath.Join(t.TempDir(), "adapter")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '{}\\n{}\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := (processLauncher{}).Exchange(ctx, ExecutableTrust{
		ExecutablePath:   path,
		ExecutableDigest: taggedBytes([]byte("#!/bin/sh\nprintf '{}\\n{}\\n'\n")),
	}, ControlRequest{})
	if err == nil {
		t.Fatal("extra provider output was accepted")
	}
}

func testManager(t *testing.T, enabled bool, launcher Launcher) *TrustManager {
	t.Helper()
	root := t.TempDir()
	layout := ssiagpaths.InstanceLayout{Scope: ssiagpaths.ScopeUser, TOPSID: trustTestTOPSID, ProviderTrustDir: filepath.Join(root, "trust")}
	if err := os.MkdirAll(layout.ProviderTrustDir, 0o700); err != nil {
		t.Fatal(err)
	}
	registry, err := New([]config.ProviderConfig{{Name: "native", Kind: "macos-keychain", Enabled: enabled, Capabilities: []string{"capability-discovery", "metadata"}}})
	if err != nil {
		t.Fatal(err)
	}
	foundation := foundationEvidence{ExecutablePath: filepath.Join(root, "foundation"), InstallationDigest: "not_applicable", ExecutableDigest: tagged("foundation"), SigningIdentity: "not_applicable", OwnerUID: uint32(os.Geteuid()), OwnerGID: uint32(os.Getegid())}
	now := func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) }
	manager, err := newTrustManager(ssiagpaths.ScopeUser, layout, registry, launcher, now, foundation)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func writeTrustPackage(t *testing.T, manager *TrustManager) ExecutableTrust {
	t.Helper()
	prefix := filepath.Join(filepath.Dir(manager.layout.ProviderTrustDir), "prefix")
	version, packageID := "0.1.0", "ssiag-macos-keychain-provider"
	executable := filepath.Join(prefix, "libexec", "symphony", packageID, version, macOSKeychainExecutableName)
	if err := os.MkdirAll(filepath.Dir(executable), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("provider fixture")
	if err := os.WriteFile(executable, payload, 0o700); err != nil {
		t.Fatal(err)
	}
	executableDigest := taggedBytes(payload)
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	relative, _ := filepath.Rel(prefix, executable)
	relative = filepath.ToSlash(relative)
	receipt := installReceiptV2{
		Protocol: "symphony.knowledge.install-receipt.v2", FormatVersion: 2, ComponentID: packageID, ComponentKind: "adapter", ModuleID: packageID,
		PackageID: packageID, Version: version, InstallScope: "prefix", PrefixMode: "installation_prefix",
		Files:                []receiptFile{{Path: relative, Kind: "executable", Size: uint64(len(payload)), Digest: executableDigest}},
		EntryPoints:          []receiptEntryPoint{{EntryPointID: "ssiag.macos-keychain-provider", Kind: "adapter", Path: relative, Protocols: []string{ProviderControlProtocol}}},
		ProvidesCapabilities: []string{providerMetadataCapability}, RequiresCapabilities: []string{}, CompatibleReceptors: []string{providerLauncherReceptor},
		PlatformRequirements: []receiptPlatform{{OS: osName, Architecture: runtime.GOARCH, Critical: true}},
	}
	receipt.ReceiptDigest = receiptDigest(receipt)
	receiptPath := filepath.Join(prefix, "share", "symphony", "receipts", packageID, version, "install-receipt.json")
	if err := os.MkdirAll(filepath.Dir(receiptPath), 0o700); err != nil {
		t.Fatal(err)
	}
	writeJSONFile(t, receiptPath, receipt)
	info, err := os.Stat(executable)
	if err != nil {
		t.Fatal(err)
	}
	uid, gid, err := fileOwner(info)
	if err != nil {
		t.Fatal(err)
	}
	declaration := ExecutableTrust{
		Protocol: "symphony.ssiag.provider-executable-trust.v1", TOPSID: trustTestTOPSID, Scope: "user", ProviderName: "native", ProviderKind: "macos-keychain",
		AdapterIdentifier: MacOSKeychainAdapterID, AdapterVersion: version, ProviderProtocol: ProviderProtocol, ExecutablePath: executable,
		InstallationDigest: receipt.ReceiptDigest, ExecutableDigest: executableDigest, OwnerUID: uid, OwnerGID: gid, FileMode: "0700",
		AdapterSigningIdentity: "not_applicable", FoundationExecutablePath: manager.foundation.ExecutablePath,
		FoundationInstallationDigest: manager.foundation.InstallationDigest, FoundationExecutableDigest: manager.foundation.ExecutableDigest,
		FoundationOwnerUID: manager.foundation.OwnerUID, FoundationOwnerGID: manager.foundation.OwnerGID, FoundationSigningIdentity: manager.foundation.SigningIdentity,
	}
	declaration.DeclarationDigest = objectDigest(declaration, "declaration_digest")
	writeJSONFile(t, filepath.Join(manager.layout.ProviderTrustDir, "native.json"), declaration)
	return declaration
}

func providerTestPrefix(executable string) string {
	current := executable
	for range 5 {
		current = filepath.Dir(current)
	}
	return current
}

func verifiedResponse(manager *TrustManager, declaration ExecutableTrust, requestID, correlationID string) ControlResponse {
	deadline := timestamp(manager.now().Add(defaultProviderTimeout))
	handshake := Handshake{
		Protocol: "symphony.ssiag.provider-handshake.v1", ProviderProtocol: ProviderProtocol, ProviderName: "native", ProviderKind: "macos-keychain",
		AdapterIdentifier: declaration.AdapterIdentifier, AdapterVersion: declaration.AdapterVersion, Platform: func() string {
			if runtime.GOOS == "darwin" {
				return "macos"
			}
			return runtime.GOOS
		}(), Architecture: runtime.GOARCH,
		Transport: "stdio-one-shot-json", ControlRequestProtocol: ControlRequestProtocol, ControlResponseProtocol: ControlResponseProtocol,
		OneShotChannelProtocol: "symphony.ssiag.provider-one-shot-channel.v1", Status: "declared", ReasonCode: "symphony.ssiag.provider.metadata_available",
		Capabilities: []string{"capability-discovery", "metadata"}, SafeOperations: []string{"capabilities", "handshake", "status"},
		Limits:          HandshakeLimits{maximumControlBytes, maximumControlBytes, 5000, 30000, 128, 128, 1, 1},
		FoundationTrust: FoundationTrust{true, manager.foundation.ExecutablePath, manager.foundation.InstallationDigest, manager.foundation.ExecutableDigest, manager.foundation.SigningIdentity, "symphony.ssiag.provider.foundation_verified"},
	}
	handshake.HandshakeDigest = objectDigest(handshake, "handshake_digest")
	response := ControlResponse{Protocol: ControlResponseProtocol, RequestID: requestID, CorrelationID: correlationID, TOPSID: trustTestTOPSID,
		ProviderName: "native", AdapterIdentifier: declaration.AdapterIdentifier, Operation: "handshake", DeadlineAt: deadline, Outcome: "succeeded", Status: "declared",
		ReasonCode: "symphony.ssiag.provider.metadata_available", Handshake: &handshake, Capabilities: handshake.Capabilities, CompletedAt: timestamp(manager.now())}
	response.ResponseDigest = objectDigest(response, "response_digest")
	return response
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
func tagged(seed string) string { return taggedBytes([]byte(seed)) }
func taggedBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func trustUUID(last byte) string {
	return "018f0c3a-7b2d-7e11-8c12-0242ac1200" + string([]byte{'0' + last})
}

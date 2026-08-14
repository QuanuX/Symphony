package provider

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestProviderControlRequestContainsOnlySafeMetadata(t *testing.T) {
	request := ControlRequest{
		Protocol: ControlRequestProtocol, RequestID: trustUUID(1), CorrelationID: trustUUID(2), TOPSID: trustTestTOPSID,
		ProviderName: "native", AdapterIdentifier: MacOSKeychainAdapterID, Operation: "handshake",
		RequestedAt: "2026-08-13T12:00:00Z", DeadlineAt: "2026-08-13T12:00:05Z", TimeoutMilliseconds: 5000,
		FoundationExecutablePath: "/opt/symphony/foundation", FoundationInstallationDigest: "not_applicable",
		FoundationExecutableDigest: tagged("foundation"), FoundationSigningIdentity: "not_applicable",
	}
	request.RequestDigest = objectDigest(request, "request_digest")
	payload, err := json.Marshal(request)
	if err != nil || len(payload) > maximumControlBytes || validateJSONMembers(payload) != nil {
		t.Fatal("safe control request is not bounded strict JSON")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"credential", "proof", "assertion", "token", "secret_value", "provider_payload", "environment", "argument"} {
		if _, present := object[forbidden]; present {
			t.Fatalf("secret-bearing field %q entered control request", forbidden)
		}
	}
	if request.OperationalAccessRequested || request.ProviderOperationRequested || request.SecretChannelRequested {
		t.Fatal("disabled operational boundary was enabled")
	}
}

func TestProviderControlChannelRejectsSecretAndExtraOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	for name, output := range map[string]string{
		"secret-shaped field": `{"secret_value":"forbidden"}` + "\n",
		"extra output":        "{}\n{}\n",
	} {
		t.Run(name, func(t *testing.T) {
			path, digest := writeAdapterScript(t, "#!/bin/sh\nread ignored\nprintf '%s' '"+output+"'\n")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := (processLauncher{}).Exchange(ctx, ExecutableTrust{ExecutablePath: path, ExecutableDigest: digest}, ControlRequest{}); err == nil {
				t.Fatal("unsafe provider output was accepted")
			}
		})
	}
}

func TestProviderControlResponseRequiresExplicitFalseMembers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	encoded, err := json.Marshal(ControlResponse{Capabilities: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	delete(object, "secret_channel_enabled")
	encoded, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	path, digest := writeAdapterScript(t, "#!/bin/sh\nread ignored\nprintf '%s\\n' '"+string(encoded)+"'\n")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := (processLauncher{}).Exchange(ctx, ExecutableTrust{ExecutablePath: path, ExecutableDigest: digest}, ControlRequest{}); err == nil {
		t.Fatal("provider response omitted a required false member")
	}
}

func TestProviderMetadataRealProcessBoundary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	responsePayload, err := json.Marshal(ControlResponse{Capabilities: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	path, digest := writeAdapterScript(t, "#!/bin/sh\nread ignored\nprintf '%s\\n' '"+string(responsePayload)+"'\n")
	process := exec.Command(path, "serve")
	process.Stdin = strings.NewReader("{}\n")
	if output, err := process.CombinedOutput(); err != nil || string(output) != string(responsePayload)+"\n" {
		t.Fatalf("real process fixture failed before launcher coverage: output=%q err=%v", output, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := (processLauncher{}).Exchange(ctx, ExecutableTrust{ExecutablePath: path, ExecutableDigest: digest}, ControlRequest{}); err != nil {
		t.Fatalf("bounded one-request/one-response real process failed: %v", err)
	}
}

func TestProviderLauncherRejectsUntrustedExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	path, _ := writeAdapterScript(t, "#!/bin/sh\nexit 0\n")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := (processLauncher{}).Exchange(ctx, ExecutableTrust{ExecutablePath: path, ExecutableDigest: tagged("different")}, ControlRequest{}); err == nil {
		t.Fatal("changed executable bytes reached process execution")
	}
}

func TestProviderLauncherRejectsDescriptorIdentityAndProtocolDrift(t *testing.T) {
	launcher := &fakeLauncher{}
	manager := testManager(t, true, launcher)
	declaration := writeTrustPackage(t, manager)
	requestID, correlationID := trustUUID(7), trustUUID(8)
	launcher.response = verifiedResponse(manager, declaration, requestID, correlationID)
	launcher.response.Handshake.ProviderProtocol = "symphony.ssiag.provider.v2"
	launcher.response.Handshake.HandshakeDigest = objectDigest(*launcher.response.Handshake, "handshake_digest")
	launcher.response.ResponseDigest = objectDigest(launcher.response, "response_digest")
	result, _ := manager.Verify(context.Background(), "native", requestID, correlationID)
	if result.TrustState != "mismatch" || result.MutualTrust.FoundationVerifiedAdapter || result.MutualTrust.AdapterVerifiedFoundation {
		t.Fatalf("protocol drift did not fail closed: %+v", result)
	}
}

func writeAdapterScript(t *testing.T, contents string) (string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "adapter")
	payload := []byte(contents)
	if err := os.WriteFile(path, payload, 0o700); err != nil {
		t.Fatal(err)
	}
	return path, taggedBytes(payload)
}

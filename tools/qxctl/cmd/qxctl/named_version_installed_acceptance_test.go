package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
)

func TestInstalledNamedVersionAcceptance(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_NAMED_VERSION_ACCEPTANCE_PREFIX")
	repository := os.Getenv("SYMPHONY_NAMED_VERSION_ACCEPTANCE_REPOSITORY")
	if prefix == "" || repository == "" {
		t.Skip("installed Named Version acceptance requires an exact staged prefix and repository")
	}
	if !filepath.IsAbs(prefix) || !filepath.IsAbs(repository) {
		t.Fatal("acceptance prefix and repository must be absolute")
	}

	stateRoot := filepath.Join(t.TempDir(), "state")
	store, err := knowledgebinding.NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	registry, changed, err := store.Bind("coordinator", prefix, "0.1.0-dev", "absent")
	if err != nil || !changed {
		t.Fatalf("bind installed coordinator: changed=%t err=%v", changed, err)
	}
	registry, changed, err = store.Bind("sav", prefix, "0.1.0-dev", registry.RegistryDigest)
	if err != nil || !changed {
		t.Fatalf("bind installed SAV: changed=%t err=%v", changed, err)
	}
	if snapshot, err := store.Snapshot(); err != nil || !snapshot.Exists || len(snapshot.Registry.Bindings) != 2 {
		t.Fatalf("exact installed binding snapshot is incomplete: %+v err=%v", snapshot, err)
	}

	serveNamedVersionAcceptanceSSIAG(t)
	inputPath := filepath.Join(t.TempDir(), "named-version.json")
	artifact := map[string]any{
		"protocol": "symphony.sav.named-version.v1", "named_version_id": "savver:symphony:acceptance",
		"alias": "Acceptance", "predecessor_digest": nil,
		"component_requirements": []any{}, "contract_requirements": []any{},
		"accord_reference_ids": []any{}, "required_traits": []any{}, "extension_points": []any{},
		"platform_bounds": []any{"linux:amd64", "macos:arm64"}, "thermal_restriction": "freezing_only",
		"sealed_at": "2026-08-21T18:00:00Z", "composition_authority_reference": "ssiag:decision:acceptance",
		"sodv_publication_reference": nil, "named_version_digest": nil,
	}
	artifact["named_version_digest"], err = maintenanceObjectDigest(artifact, "named_version_digest")
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(map[string]any{"named_version": artifact})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("write acceptance input: %v", err)
	}

	base := namedVersionOptions{
		topsID: ssiagTestTOPSID, scope: "user", stateRoot: stateRoot,
		repository: repository, ttl: 5 * time.Minute,
	}
	propose := base
	propose.input, propose.operationID, propose.expectedRegistryDigest = inputPath, "acceptance-prepare", "absent"
	prepared, _, err := executeNamedVersion("named_version_prepare", propose)
	if err != nil || prepared.ProposalDigest == nil || prepared.Changed == nil || !*prepared.Changed {
		t.Fatalf("prepare installed Named Version: %+v err=%v", prepared, err)
	}

	seal := base
	seal.operationID, seal.expectedRegistryDigest = "acceptance-seal", "absent"
	seal.preparedOperationID, seal.proposalDigest = propose.operationID, *prepared.ProposalDigest
	sealed, _, err := executeNamedVersion("named_version_seal", seal)
	if err != nil || sealed.RegistryDigest == nil || sealed.Changed == nil || !*sealed.Changed || len(sealed.Artifact) == 0 {
		t.Fatalf("seal installed Named Version: %+v err=%v", sealed, err)
	}
	sealedDigest := *sealed.RegistryDigest

	replayed, _, err := executeNamedVersion("named_version_seal", seal)
	if err != nil || replayed.RegistryDigest == nil || replayed.Changed == nil || *replayed.RegistryDigest != sealedDigest || *replayed.Changed {
		t.Fatalf("idempotent installed seal replay drifted: %+v err=%v", replayed, err)
	}

	alias := base
	alias.operationID, alias.expectedRegistryDigest = "acceptance-alias", sealedDigest
	alias.alias, alias.digest = "stable", artifact["named_version_digest"].(string)
	aliased, _, err := executeNamedVersion("named_version_alias", alias)
	if err != nil || aliased.RegistryDigest == nil || aliased.Changed == nil || *aliased.RegistryDigest == sealedDigest || !*aliased.Changed {
		t.Fatalf("alias installed Named Version: %+v err=%v", aliased, err)
	}

	lookup := base
	lookup.alias = "stable"
	selected, _, err := executeNamedVersion("named_version_lookup", lookup)
	if err != nil || selected.SelectedAlias == nil || selected.ReadOnly == nil || *selected.SelectedAlias != "stable" || !*selected.ReadOnly {
		t.Fatalf("lookup installed Named Version: %+v err=%v", selected, err)
	}
	status, _, err := executeNamedVersion("named_version_status", base)
	if err != nil || status.ReadOnly == nil || status.STAVAppendEnabled == nil || status.VersionCount != 1 || status.AliasCount != 1 || !*status.ReadOnly || *status.STAVAppendEnabled {
		t.Fatalf("status installed Named Version: %+v err=%v", status, err)
	}
}

func serveNamedVersionAcceptanceSSIAG(t *testing.T) {
	t.Helper()
	configHome := filepath.Join(t.TempDir(), "config")
	runtimeHome, err := os.MkdirTemp("/tmp", "qxctl-named-version-")
	if err != nil {
		t.Fatalf("create short SSIAG acceptance runtime root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runtimeHome) })
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_RUNTIME_DIR", runtimeHome)
	t.Setenv("SYMPHONY_SSIAG_SOCKET", "")
	socket := filepath.Join(runtimeHome, "symphony", ssiagTestTOPSID, "ssiag", "ssiag.sock")
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen on isolated SSIAG acceptance socket: %v", err)
	}
	configPath := filepath.Join(configHome, "symphony", ssiagTestTOPSID, "ssiag", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema": "symphony.ssiag.config.v1", "mode": "user",
		"tops":   map[string]any{"id": ssiagTestTOPSID, "name": "Acceptance"},
		"listen": map[string]any{"network": "unix", "address": socket},
		"authentication": map[string]any{
			"mechanism": "unix_peer_credentials",
			"service":   map[string]any{"id": "symphony.ssiag.service", "kind": "symphony.identity.service", "uid": os.Geteuid(), "gid": os.Getegid()},
			"subjects":  []any{},
		},
		"authorization": map[string]any{}, "providers": []any{},
	}
	encoded, _ := json.Marshal(config)
	if err := os.WriteFile(configPath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/v1/status" {
			_ = json.NewEncoder(writer).Encode(ssiagclient.Status{
				Schema: "symphony.ssiag.status.v1", Name: "secure-identity-access-governance", Version: "acceptance",
				Ready: true, Mode: "user", TOPSID: ssiagTestTOPSID, TOPSName: "Acceptance", Transport: "unix",
			})
			return
		}
		if request.URL.Path != "/v1/authorization/decisions" {
			http.Error(writer, "not found", http.StatusNotFound)
			return
		}
		var authorization ssiagclient.AuthorizationRequest
		if err := json.NewDecoder(request.Body).Decode(&authorization); err != nil {
			http.Error(writer, "invalid request", http.StatusBadRequest)
			return
		}
		decision := namedVersionAcceptanceDecision(authorization)
		_ = json.NewEncoder(writer).Encode(decision)
	})
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = listener.Close()
	})
}

func namedVersionAcceptanceDecision(request ssiagclient.AuthorizationRequest) ssiagclient.AuthorizationDecision {
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(2 * time.Minute)
	if request.RequestedExpiresAt.Before(expires) {
		expires = request.RequestedExpiresAt
	}
	subject := ssiagclient.DecisionSubject{ID: "owner.acceptance", Kind: "owner", Authority: "unix_peer_credentials"}
	target := ssiagclient.DecisionTarget{Operation: request.Operation, Resource: request.Resource, Audience: request.Audience, Scope: request.Scope}
	basis := "host_owner"
	capability := &ssiagclient.Capability{
		Protocol: "symphony.ssiag.capability.v1", Subject: subject, TOPSID: ssiagTestTOPSID,
		Target: target, AuthorityBasis: basis, GrantID: "named-version-acceptance",
		RequestID: request.RequestID, CorrelationID: request.CorrelationID, IssuedAt: now, ExpiresAt: expires,
		PolicyDigest: "sha256:" + strings.Repeat("a", 64), ConfigDigest: "sha256:" + strings.Repeat("b", 64),
		Transferable: false, CanonicalApply: false,
	}
	capability.BindingDigest = sessionCapabilityBinding(*capability)
	capability.CapabilityID = "ssiag-capability:" + strings.TrimPrefix(capability.BindingDigest, "sha256:")
	return ssiagclient.AuthorizationDecision{
		Schema: "symphony.ssiag.authorization-decision.v1", DecisionID: "ssiag-decision:acceptance",
		RequestID: request.RequestID, CorrelationID: request.CorrelationID, TOPSID: ssiagTestTOPSID,
		Subject: subject, Target: target, Effect: "allow", ReasonCode: "symphony.ssiag.policy.exact-grant",
		AuthorityBasis: &basis, Capability: capability, PolicyDigest: capability.PolicyDigest,
		ConfigDigest: capability.ConfigDigest, DecidedAt: now, ExpiresAt: &expires,
		CallerClassUsed: false, CanonicalApply: false,
	}
}

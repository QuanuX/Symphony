package config

import (
	"os"
	"path/filepath"
	"testing"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func validConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		Schema:    "symphony.ssiag.config.v1",
		Mode:      "development",
		TOPS:      TOPSConfig{ID: testTOPSID, Name: "Local research TOPS"},
		Listen:    ListenConfig{Network: "unix", Address: filepath.Join(t.TempDir(), "ssiag.sock")},
		Providers: []ProviderConfig{},
	}
}

func TestDefaultSeparatesTOPSIDAndName(t *testing.T) {
	layout := ssiagpaths.InstanceLayout{Scope: ssiagpaths.ScopeUser, TOPSID: testTOPSID, Socket: filepath.Join(t.TempDir(), "ssiag.sock")}
	cfg := Default(layout, "Trading desk", nil, nil)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.TOPS.ID == cfg.TOPS.Name {
		t.Fatal("TOPS ID and display name must remain distinct")
	}
	if cfg.Authentication == nil || cfg.Authentication.Mechanism != "unix_peer_credentials" || cfg.Authentication.Subjects == nil {
		t.Fatalf("default authentication must be explicit: %+v", cfg.Authentication)
	}
}

func TestDuplicateProviderRejected(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers = []ProviderConfig{{Name: "native", Kind: "native-keyring"}, {Name: "native", Kind: "oidc"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate provider error")
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"schema":"symphony.ssiag.config.v1","mode":"user","tops":{"id":"018f0c3a-7b2d-7e11-8c12-0242ac120002","name":"Desk"},"listen":{"network":"unix","address":"/tmp/ssiag.sock"},"providers":[],"secret":"forbidden"}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestRejectsDisplayNameAsIdentity(t *testing.T) {
	cfg := validConfig(t)
	cfg.TOPS.ID = "trading-desk"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-opaque identity to be rejected")
	}
}

func TestAuthenticationMappingIsExplicitAndUnambiguous(t *testing.T) {
	uid, gid := uint32(501), uint32(20)
	cfg := validConfig(t)
	cfg.Authentication = &AuthenticationConfig{
		Mechanism: "unix_peer_credentials",
		Subjects: []SubjectConfig{{
			ID: "operator.primary", Kind: "operator", UID: &uid, GID: &gid,
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	cfg.Authentication.Subjects = append(cfg.Authentication.Subjects, SubjectConfig{
		ID: "service.duplicate", Kind: "service", UID: &uid, GID: &gid,
	})
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected ambiguous operating-system identity to be rejected")
	}
}

func TestAuthenticationMappingRequiresUIDAndGIDPresence(t *testing.T) {
	uid := uint32(0)
	cfg := validConfig(t)
	cfg.Authentication = &AuthenticationConfig{
		Mechanism: "unix_peer_credentials",
		Subjects: []SubjectConfig{{
			ID: "service.root", Kind: "service", UID: &uid,
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing gid to be rejected even though gid zero is valid")
	}
}

func TestCanonicalServiceIdentityUsesPresenceSafeZeroValues(t *testing.T) {
	uid, gid := uint32(0), uint32(0)
	cfg := validConfig(t)
	cfg.Authentication = &AuthenticationConfig{
		Mechanism: "unix_peer_credentials",
		Service: &SubjectConfig{
			ID: ServiceSubjectID, Kind: ServiceSubjectKind, UID: &uid, GID: &gid,
		},
		Subjects: []SubjectConfig{},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("zero-valued service identity must remain explicitly valid: %v", err)
	}
	cfg.Authentication.Service.GID = nil
	if err := cfg.Validate(); err == nil {
		t.Fatal("missing service GID must not be confused with explicit zero")
	}
}

func TestCanonicalServiceIdentityCannotBeRenamedByConfiguration(t *testing.T) {
	uid, gid := uint32(501), uint32(20)
	cfg := validConfig(t)
	cfg.Authentication = &AuthenticationConfig{
		Mechanism: "unix_peer_credentials",
		Service: &SubjectConfig{
			ID: "caller-selected-service", Kind: ServiceSubjectKind, UID: &uid, GID: &gid,
		},
		Subjects: []SubjectConfig{},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected noncanonical service identity to be rejected")
	}
}

func TestLegacyMetadataConfigHasNoSubjectAuthority(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"schema":"symphony.ssiag.config.v1","mode":"user","tops":{"id":"018f0c3a-7b2d-7e11-8c12-0242ac120002","name":"Desk"},"listen":{"network":"unix","address":"/tmp/ssiag.sock"},"providers":[]}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Authentication != nil {
		t.Fatalf("legacy configuration unexpectedly gained subject authority: %+v", cfg.Authentication)
	}
}

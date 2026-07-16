package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const maxConfigBytes = 1 << 20

type ListenConfig struct {
	Network string `json:"network"`
	Address string `json:"address"`
}

type ProviderConfig struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Enabled      bool     `json:"enabled"`
	Capabilities []string `json:"capabilities,omitempty"`
	Exportable   bool     `json:"exportable,omitempty"`
	Interactive  bool     `json:"interactive,omitempty"`
}

type TOPSConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SubjectConfig struct {
	ID   string  `json:"id"`
	Kind string  `json:"kind"`
	UID  *uint32 `json:"uid"`
	GID  *uint32 `json:"gid"`
}

type AuthenticationConfig struct {
	Mechanism string          `json:"mechanism"`
	Subjects  []SubjectConfig `json:"subjects"`
}

type Config struct {
	Schema         string                `json:"schema"`
	Mode           string                `json:"mode"`
	TOPS           TOPSConfig            `json:"tops"`
	Listen         ListenConfig          `json:"listen"`
	Authentication *AuthenticationConfig `json:"authentication,omitempty"`
	Providers      []ProviderConfig      `json:"providers"`
}

func Default(layout ssiagpaths.InstanceLayout, topsName string) Config {
	return Config{
		Schema: "symphony.ssiag.config.v1",
		Mode:   string(layout.Scope),
		TOPS:   TOPSConfig{ID: layout.TOPSID, Name: topsName},
		Listen: ListenConfig{
			Network: "unix",
			Address: layout.Socket,
		},
		Authentication: &AuthenticationConfig{
			Mechanism: "unix_peer_credentials",
			Subjects:  []SubjectConfig{},
		},
		Providers: []ProviderConfig{},
	}
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return Config{}, fmt.Errorf("stat config: %w", err)
	}
	if info.Size() > maxConfigBytes {
		return Config{}, fmt.Errorf("config exceeds %d bytes", maxConfigBytes)
	}

	decoder := json.NewDecoder(io.LimitReader(file, maxConfigBytes+1))
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Marshal(cfg Config) ([]byte, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func (cfg Config) Validate() error {
	if cfg.Schema != "symphony.ssiag.config.v1" {
		return fmt.Errorf("unsupported config schema %q", cfg.Schema)
	}
	if cfg.Mode != "user" && cfg.Mode != "system" && cfg.Mode != "development" {
		return fmt.Errorf("unsupported mode %q", cfg.Mode)
	}
	if err := ssiagpaths.ValidateTOPSID(cfg.TOPS.ID); err != nil {
		return fmt.Errorf("invalid TOPS identity: %w", err)
	}
	if !validDisplayName(cfg.TOPS.Name) {
		return fmt.Errorf("invalid TOPS display name")
	}
	if cfg.Listen.Network != "unix" {
		return fmt.Errorf("unsupported listen network %q: scaffold requires unix", cfg.Listen.Network)
	}
	if !filepath.IsAbs(cfg.Listen.Address) {
		return fmt.Errorf("listen address must be absolute")
	}
	if err := validateAuthentication(cfg.Authentication); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(cfg.Providers))
	for i, provider := range cfg.Providers {
		if !validName(provider.Name) {
			return fmt.Errorf("provider %d has invalid name %q", i, provider.Name)
		}
		if !validName(provider.Kind) {
			return fmt.Errorf("provider %q has invalid kind %q", provider.Name, provider.Kind)
		}
		if _, exists := seen[provider.Name]; exists {
			return fmt.Errorf("duplicate provider name %q", provider.Name)
		}
		seen[provider.Name] = struct{}{}
		for _, capability := range provider.Capabilities {
			if !validName(capability) {
				return fmt.Errorf("provider %q has invalid capability %q", provider.Name, capability)
			}
		}
	}
	return nil
}

func validateAuthentication(authentication *AuthenticationConfig) error {
	// A missing block is accepted only for read compatibility with metadata-only
	// v1 enrollments. The server still authenticates every accepted connection,
	// but no peer from such a configuration can resolve to a mutation subject.
	if authentication == nil {
		return nil
	}
	if authentication.Mechanism != "unix_peer_credentials" {
		return fmt.Errorf("unsupported authentication mechanism %q", authentication.Mechanism)
	}
	if authentication.Subjects == nil {
		return fmt.Errorf("authentication subjects must be an explicit array")
	}
	seenSubjects := make(map[string]struct{}, len(authentication.Subjects))
	type osIdentity struct {
		uid uint32
		gid uint32
	}
	seenIdentities := make(map[osIdentity]string, len(authentication.Subjects))
	for i, subject := range authentication.Subjects {
		if !validName(subject.ID) {
			return fmt.Errorf("authentication subject %d has invalid ID %q", i, subject.ID)
		}
		if !validName(subject.Kind) {
			return fmt.Errorf("authentication subject %q has invalid kind %q", subject.ID, subject.Kind)
		}
		if subject.UID == nil || subject.GID == nil {
			return fmt.Errorf("authentication subject %q must explicitly declare uid and gid", subject.ID)
		}
		if _, exists := seenSubjects[subject.ID]; exists {
			return fmt.Errorf("duplicate authentication subject ID %q", subject.ID)
		}
		identity := osIdentity{uid: *subject.UID, gid: *subject.GID}
		if existing, exists := seenIdentities[identity]; exists {
			return fmt.Errorf("operating-system identity uid=%d gid=%d maps ambiguously to %q and %q", identity.uid, identity.gid, existing, subject.ID)
		}
		seenSubjects[subject.ID] = struct{}{}
		seenIdentities[identity] = subject.ID
	}
	return nil
}

func validDisplayName(value string) bool {
	if value == "" || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func validName(value string) bool {
	if value == "" || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("config contains multiple JSON values")
		}
		return fmt.Errorf("decode trailing config data: %w", err)
	}
	return nil
}

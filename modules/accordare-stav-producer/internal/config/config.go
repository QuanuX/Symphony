package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	accordareprotocol "github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/protocol"
)

const (
	Schema                 = "symphony.accordare.stav-producer.config.v1"
	VocabularyDigest       = "sha256:dd422e3b673c85103cf59974d1727ca037c92d980bffe83256a55ad538c10ad3"
	MaxConfigBytes   int64 = 1 << 20
)

type Identity struct {
	GID     uint64                     `json:"gid"`
	Subject stavprotocol.SafeReference `json:"subject"`
	UID     uint64                     `json:"uid"`
}

type Listen struct {
	Address string `json:"address"`
	Network string `json:"network"`
}

type Config struct {
	Identity         Identity   `json:"identity"`
	Listen           Listen     `json:"listen"`
	Mode             string     `json:"mode"`
	Schema           string     `json:"schema"`
	STAVConfig       string     `json:"stav_config"`
	Submitters       []Identity `json:"submitters"`
	TOPSID           string     `json:"tops_id"`
	VocabularyDigest string     `json:"vocabulary_digest"`
}

func (cfg Config) Validate() error {
	if cfg.Schema != Schema || (cfg.Mode != "user" && cfg.Mode != "system") ||
		cfg.Listen.Network != "unix" || !filepath.IsAbs(cfg.Listen.Address) ||
		!filepath.IsAbs(cfg.STAVConfig) || cfg.VocabularyDigest != VocabularyDigest {
		return fmt.Errorf("Accordare producer configuration identity is invalid")
	}
	if err := stavprotocol.ValidateTOPSID(cfg.TOPSID); err != nil {
		return err
	}
	if err := validateIdentity(cfg.Identity); err != nil {
		return err
	}
	if cfg.Identity.Subject.Kind != "symphony.identity.service" {
		return fmt.Errorf("Accordare producer identity must be a service")
	}
	if cfg.Submitters == nil || len(cfg.Submitters) == 0 || len(cfg.Submitters) > 64 {
		return fmt.Errorf("Accordare producer requires one to 64 submitters")
	}
	seen := map[[2]uint64]struct{}{}
	for _, submitter := range cfg.Submitters {
		if err := validateIdentity(submitter); err != nil {
			return err
		}
		key := [2]uint64{submitter.UID, submitter.GID}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("Accordare producer submitter credentials are ambiguous")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateIdentity(identity Identity) error {
	if identity.UID > uint64(^uint32(0)) || identity.GID > uint64(^uint32(0)) {
		return fmt.Errorf("Accordare producer identity exceeds platform UID/GID range")
	}
	if _, err := accordareprotocol.NormalizeSSIAGSubject(identity.Subject); err != nil {
		return err
	}
	return nil
}

// Load reads strict, bounded, no-symlink configuration.
func Load(path string) (Config, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 || info.Size() <= 0 || info.Size() > MaxConfigBytes {
		return Config{}, fmt.Errorf("Accordare producer configuration is missing or unsafe")
	}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxConfigBytes+1))
	if err != nil {
		return Config{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return Config{}, fmt.Errorf("decode Accordare producer configuration")
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

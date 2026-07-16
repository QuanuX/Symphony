package config

import (
	"fmt"
	"io"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

const maxConfigBytes = 1 << 20

func Default(layout stavpaths.InstanceLayout, authorityUID, authorityGID uint64) stavprotocol.AppendAuthorityConfig {
	return stavprotocol.AppendAuthorityConfig{
		Authentication: stavprotocol.AppendAuthorityAuthentication{
			Authority: stavprotocol.AuthorityGrant{
				GID:     authorityGID,
				Subject: stavprotocol.SafeReference{ID: "stav-append-authority", Kind: "symphony.identity.service"},
				UID:     authorityUID,
			},
			Mechanism: "kernel-peer-credentials",
			Producers: []stavprotocol.ProducerGrant{},
			Readers:   []stavprotocol.ReaderGrant{},
		},
		Ledger: stavprotocol.AppendAuthorityLedger{
			Durability: "fsync-before-receipt",
			MaxBytes:   1_073_741_824,
			Path:       layout.LedgerFile,
			Recovery:   "preserve-incomplete-tail",
			Retention:  "preserve_all",
			Rotation:   "disabled",
		},
		Listen: stavprotocol.AppendAuthorityListen{Address: layout.Socket, Network: "unix"},
		Mode:   string(layout.Scope),
		Schema: stavprotocol.SchemaAppendAuthorityConfig,
		TOPSID: layout.TOPSID,
	}
}

// Load accepts human-formatted strict JSON, canonicalizes it, and then applies
// the exact typed protocol contract.
func Load(path string) (stavprotocol.AppendAuthorityConfig, error) {
	file, err := openNoFollow(path)
	if err != nil {
		return stavprotocol.AppendAuthorityConfig{}, fmt.Errorf("open STAV config: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return stavprotocol.AppendAuthorityConfig{}, fmt.Errorf("stat STAV config: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o022 != 0 || info.Size() <= 0 || info.Size() > maxConfigBytes {
		return stavprotocol.AppendAuthorityConfig{}, fmt.Errorf("STAV config is not a bounded regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil {
		return stavprotocol.AppendAuthorityConfig{}, fmt.Errorf("read STAV config: %w", err)
	}
	canonical, err := stavprotocol.Canonicalize(raw)
	if err != nil {
		return stavprotocol.AppendAuthorityConfig{}, fmt.Errorf("canonicalize STAV config: %w", err)
	}
	cfg, err := stavprotocol.DecodeAppendAuthorityConfig(canonical)
	if err != nil {
		return stavprotocol.AppendAuthorityConfig{}, fmt.Errorf("decode STAV config: %w", err)
	}
	return cfg, nil
}

func Marshal(cfg stavprotocol.AppendAuthorityConfig) ([]byte, error) {
	return stavprotocol.EncodeAppendAuthorityConfig(cfg)
}

func ValidateLayout(cfg stavprotocol.AppendAuthorityConfig, layout stavpaths.InstanceLayout) error {
	if cfg.TOPSID != layout.TOPSID || cfg.Mode != string(layout.Scope) || cfg.Listen.Address != layout.Socket || cfg.Ledger.Path != layout.LedgerFile {
		return fmt.Errorf("STAV configuration does not match the selected TOPS layout")
	}
	return nil
}

package config

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceOwnershipAdmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		uid, gid uint32
		valid    bool
	}{
		{"zero", 0, 0, true},
		{"ordinary", 501, 20, true},
		{"maximum usable", math.MaxUint32 - 1, math.MaxUint32 - 1, true},
		{"reserved UID", math.MaxUint32, 20, false},
		{"reserved GID", 501, math.MaxUint32, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig(t)
			cfg.Authentication = &AuthenticationConfig{
				Mechanism: "unix_peer_credentials", Subjects: []SubjectConfig{},
				Service: &SubjectConfig{ID: ServiceSubjectID, Kind: ServiceSubjectKind, UID: &tc.uid, GID: &tc.gid},
			}
			if err := cfg.Validate(); (err == nil) != tc.valid {
				t.Fatalf("config admission: valid=%t error=%v", tc.valid, err)
			}
			if _, err := Marshal(cfg); (err == nil) != tc.valid {
				t.Fatalf("marshal admission: valid=%t error=%v", tc.valid, err)
			}
			// Bypass the local writer to exercise admission of operator-authored config.
			raw, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); (err == nil) != tc.valid {
				t.Fatalf("load admission: valid=%t error=%v", tc.valid, err)
			}
		})
	}
}

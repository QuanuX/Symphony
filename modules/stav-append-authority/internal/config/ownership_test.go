package config

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

func TestAuthorityOwnershipAdmission(t *testing.T) {
	layout := stavpaths.InstanceLayout{Scope: stavpaths.ScopeUser, TOPSID: testTOPSID, Socket: "/tmp/stav.sock", LedgerFile: "/tmp/stav.ledger"}
	for _, tc := range []struct {
		name     string
		uid, gid uint64
		valid    bool
	}{
		{"zero", 0, 0, true},
		{"ordinary", 501, 20, true},
		{"maximum usable", math.MaxUint32 - 1, math.MaxUint32 - 1, true},
		{"reserved UID", math.MaxUint32, 20, false},
		{"reserved GID", 501, math.MaxUint32, false},
		{"UID overflow", math.MaxUint32 + 1, 20, false},
		{"GID overflow", 501, math.MaxUint32 + 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default(layout, tc.uid, tc.gid)
			if err := ValidateLayout(cfg, layout); (err == nil) != tc.valid {
				t.Fatalf("layout admission: valid=%t error=%v", tc.valid, err)
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

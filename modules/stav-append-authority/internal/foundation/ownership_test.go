package foundation

import (
	"bytes"
	"encoding/json"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMachineEnrollmentOwnershipAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, uid, gid string
		valid          bool
	}{
		{"zero", "0", "0", true},
		{"ordinary", "501", "20", true},
		{"maximum usable", "4294967294", "4294967294", true},
		{"reserved UID", "4294967295", "20", false},
		{"reserved GID", "501", "4294967295", false},
		{"UID overflow", "4294967296", "20", false},
		{"GID overflow", "501", "4294967296", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := commandAt("plan", "enrollment", time.Now().UTC())
			command.Scope = "system"
			uid, gid := uint32(501), uint32(20)
			command.Intent = &Intent{DesiredState: "enrolled", AuthorityUID: &uid, AuthorityGID: &gid, AuditMode: "audit_deferred", TTLSeconds: 60}
			raw, err := json.Marshal(command)
			if err != nil {
				t.Fatal(err)
			}
			input := strings.Replace(string(raw), `"authority_uid":501`, `"authority_uid":`+tc.uid, 1)
			input = strings.Replace(input, `"authority_gid":20`, `"authority_gid":`+tc.gid, 1)
			decoded, err := DecodeCommand(bytes.NewBufferString(input))
			if err == nil {
				// Execute admits the decoded command before observing or mutating.
				err = validateCommand(decoded)
			}
			if (err == nil) != tc.valid {
				t.Fatalf("machine admission: valid=%t error=%v", tc.valid, err)
			}
		})
	}
}

func TestStoredPlanOwnershipAdmissionDoesNotRenewOrExpireRecovery(t *testing.T) {
	for _, tc := range []struct {
		name     string
		uid, gid uint32
		valid    bool
	}{
		{"zero", 0, 0, true},
		{"maximum usable", math.MaxUint32 - 1, math.MaxUint32 - 1, true},
		{"reserved UID", math.MaxUint32, 20, false},
		{"reserved GID", 501, math.MaxUint32, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := Plan{Protocol: PlanProtocol, FormatVersion: 1, Component: "stav", Surface: "enrollment", Scope: "system", TOPSID: foundationTOPSID,
				OperationID: "ownership-recovery", RequestID: "123e4567-e89b-42d3-a456-426614174001", CorrelationID: "123e4567-e89b-42d3-a456-426614174002",
				ExpectedStateDigest: "sha256:" + strings.Repeat("a", 64), DesiredState: "enrolled", AuthorityUID: &tc.uid, AuthorityGID: &tc.gid,
				AuditMode: "audit_deferred", CreatedAt: "2026-01-01T00:00:00Z", ExpiresAt: "2026-01-01T00:01:00Z"}
			plan.PlanDigest = digestWithout(plan, "plan_digest")
			path := filepath.Join(t.TempDir(), "plan.json")
			if err := writeProtectedJSON(path, plan); err != nil {
				t.Fatal(err)
			}
			if _, err := readPlan(path); (err == nil) != tc.valid {
				t.Fatalf("stored plan admission: valid=%t error=%v", tc.valid, err)
			}
			command := commandAt("apply", "enrollment", time.Now().UTC())
			command.Scope, command.Plan = "system", &plan
			if err := validateCommand(command); (err == nil) != tc.valid {
				t.Fatalf("apply ownership admission: valid=%t error=%v", tc.valid, err)
			}
		})
	}
}

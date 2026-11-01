package foundationlifecycle

import (
	"bytes"
	"encoding/json"
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
			uid, gid, name := uint32(501), uint32(20), "Desk"
			op, request, correlation, expected := "ownership-plan", "0a4d72cf-0dc5-4b4a-8bfa-6e6cf27a67c1", "23b79938-b558-4e52-9a16-2fe36b32c6a1", "absent"
			now := time.Now().UTC().Truncate(time.Second)
			command := Command{Protocol: CommandProtocol, FormatVersion: 1, Component: "ssiag", Operation: "plan", Surface: "enrollment", Scope: "system", TOPSID: testTOPSID,
				OperationID: &op, RequestID: &request, CorrelationID: &correlation, ExpectedStateDigest: &expected,
				RequestedAt: now.Format(time.RFC3339), DeadlineAt: now.Add(30 * time.Second).Format(time.RFC3339),
				Intent: &Intent{DesiredState: "enrolled", TOPSName: &name, ServiceUID: &uid, ServiceGID: &gid, AuditMode: "audit_deferred", TTLSeconds: 60}}
			raw, err := json.Marshal(command)
			if err != nil {
				t.Fatal(err)
			}
			input := strings.Replace(string(raw), `"service_uid":501`, `"service_uid":`+tc.uid, 1)
			input = strings.Replace(input, `"service_gid":20`, `"service_gid":`+tc.gid, 1)
			decoded, err := DecodeCommand(bytes.NewBufferString(input))
			if (err == nil) != tc.valid {
				t.Fatalf("machine admission: valid=%t error=%v", tc.valid, err)
			}
			// Plan.validate is also the stored recovery-plan admission path.
			if tc.uid == "4294967296" || tc.gid == "4294967296" {
				return // Overflow is rejected by the typed JSON decoder before a plan exists.
			}
			if err != nil {
				var unvalidated Command
				if err := json.Unmarshal([]byte(input), &unvalidated); err != nil {
					t.Fatal(err)
				}
				decoded = unvalidated
			}
			plan := Plan{Protocol: PlanProtocol, FormatVersion: 1, Component: "ssiag", Surface: "enrollment", Scope: "system", TOPSID: testTOPSID,
				OperationID: op, RequestID: request, CorrelationID: correlation, ExpectedStateDigest: "absent", DesiredState: "enrolled", TOPSName: &name,
				ServiceUID: decoded.Intent.ServiceUID, ServiceGID: decoded.Intent.ServiceGID, AuditMode: "audit_deferred", CreatedAt: "2026-01-01T00:00:00Z", ExpiresAt: "2026-01-01T00:01:00Z"}
			plan.PlanDigest, err = digestWithout(plan, "plan_digest")
			if err != nil {
				t.Fatal(err)
			}
			if err := plan.validate(); (err == nil) != tc.valid {
				t.Fatalf("plan ownership admission: valid=%t error=%v", tc.valid, err)
			}
		})
	}
}

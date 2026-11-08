package contract

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func fixtureConfiguration() Configuration {
	d := "sha256:" + strings.Repeat("a", 64)
	p := PackageRef{PackageID: "11111111-1111-4111-8111-111111111111", Version: "0.1.0-dev", ManifestDigest: d, ReceiptDigest: d, ExecutableDigest: d}
	return Configuration{Name: "example", TransportKind: "cloudflare-access", Platform: "darwin-amd64", ExecutionProfile: "cloudflare-access-user-session.v1", Adapter: p, SSH: p, Helper: p, Applications: []string{"example.invalid"}, Endpoints: []EndpointConstraint{{Host: "ssh.example.invalid", Port: "22", User: "example", HostKeyDigest: d}}, AuthenticationMode: "user_session", SessionRef: "session:example", Limits: Limits{ControlBytes: "8388608", CapturedBytes: "1048576", CleanupSeconds: "5", CallerTimeout: true}, Compatibility: []string{"symphony.sov.ssh-endpoint.v2", "symphony.sov.ssh-invoke-input.v2", "symphony.sov.ssh-carrier-result.v2"}}
}
func TestDigestIndependentGoldenAndExactJSON(t *testing.T) {
	// Golden computed independently from the documented literal byte string;
	// neither a generated schema nor another call to Digest is the oracle.
	v := map[string]any{"generation": "18446744073709551615", "b": []any{"x", nil}, "a": "<", "digest": "excluded"}
	if got := Digest("symphony.sov.transport-profile-test.v1", v); got != "sha256:1f3474b87588c7328eb3813fcf60deca43fa08856ba9744adbbe6fc3f4702177" {
		t.Fatal(got)
	}
	scope := Scope{HostID: "11111111-1111-4111-8111-111111111111", TOPSID: "22222222-2222-4222-8222-222222222222"}
	r := Request{Protocol: RequestProtocol, RequestID: "33333333-3333-4333-8333-333333333333", Operation: "list", Scope: scope}
	raw, _ := json.Marshal(r)
	var decoded Request
	if Decode(raw, &decoded) != nil || ValidateRequest(decoded) != nil {
		t.Fatal("valid request rejected")
	}
	for _, bad := range []string{
		strings.Replace(string(raw), `"operation":"list"`, `"operation":"list","operation":"list"`, 1),
		strings.Replace(string(raw), `"operation":`, `"Operation":`, 1),
		strings.Replace(string(raw), `,"plan":null`, "", 1),
		strings.TrimSuffix(string(raw), "}") + `,"exec":"/bin/sh"}`,
		string(raw) + ` {}`,
		strings.Replace(string(raw), `"list"`, `"\ud800"`, 1),
	} {
		if Decode([]byte(bad), &decoded) == nil {
			t.Fatalf("invalid document accepted: %q", bad)
		}
	}
	if CheckJSON([]byte(`{"counter":18446744073709551615}`)) == nil || CheckJSON([]byte(`{"counter":1e0}`)) == nil {
		t.Fatal("unsafe numeric representation accepted")
	}
}
func TestGenerationCASAndABAPreserveIdentity(t *testing.T) {
	scope := Scope{HostID: "11111111-1111-4111-8111-111111111111", TOPSID: "22222222-2222-4222-8222-222222222222"}
	id, operation := "33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444"
	admin := "sha256:" + strings.Repeat("b", 64)
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	config := fixtureConfiguration()
	principals := []Principal{{UID: "501", GID: "20"}}
	create, e := MakePlan(scope, id, operation, admin, nil, Change{Action: "create", Configuration: &config, Permitted: &principals}, now)
	if e != nil {
		t.Fatal(e)
	}
	first := create.Desired
	if first.Profile.State != "disabled" || first.Profile.Generation != "1" || ValidatePlan(create, nil, now, false) != nil {
		t.Fatal("incorrect create plan")
	}
	enable, e := MakePlan(scope, id, operation, admin, &first, Change{Action: "enable"}, now)
	if e != nil {
		t.Fatal(e)
	}
	second := enable.Desired
	disabled, e := Desired(scope, id, &second, Change{Action: "disable"})
	if e != nil {
		t.Fatal(e)
	}
	if disabled.Profile.Generation != "3" || disabled.Policy.Generation != "3" || disabled.Digest == first.Digest || ValidatePlan(enable, &disabled, now, false) == nil {
		t.Fatal("ABA or stale plan accepted")
	}
	removed, e := Desired(scope, id, &disabled, Change{Action: "remove"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Desired(scope, id, &removed, Change{Action: "enable"}); e == nil {
		t.Fatal("tombstone resurrected")
	}
	full := first
	full.Profile.Generation = "18446744073709551615"
	full.Policy.ProfileGeneration = full.Profile.Generation
	SealSnapshot(&full)
	if _, e = Desired(scope, id, &full, Change{Action: "enable"}); e == nil {
		t.Fatal("generation wrapped")
	}
	if ValidatePlan(create, nil, now.Add(6*time.Minute), false) == nil {
		t.Fatal("expired plan accepted")
	}
	for _, p := range []Principal{{UID: "0", GID: "20"}, {UID: "501", GID: "0"}, {UID: "4294967295", GID: "20"}, {UID: "501", GID: "4294967295"}, {UID: "0501", GID: "20"}} {
		if ValidPrincipals([]Principal{p}) {
			t.Fatal("invalid principal accepted", p)
		}
	}
}

package transportaudit

import (
	"strings"
	"testing"
)

func TestIndependentProfileAuditGoldenAndClosedFields(t *testing.T) {
	const document = `{"action":"create","desired_digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","host_id":"33333333-3333-4333-8333-333333333333","observed_digest":"","operation_id":"22222222-2222-4222-8222-222222222222","phase":"intent","plan_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","prior_digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","profile_id":"55555555-5555-4555-8555-555555555555","protocol":"symphony.ssiag.transport-profile-audit.v1","request_id":"11111111-1111-4111-8111-111111111111","resolution":"pending","tops_id":"44444444-4444-4444-8444-444444444444"}`
	var request Request
	if Decode([]byte(document), &request) != nil || request.Validate() != nil {
		t.Fatal("golden request invalid")
	}
	if Hash(request) != "sha256:82147bbf2339179b73950fb680f12a356c3d8d79072ba9e557208a9e577ed6e1" {
		t.Fatal("independent audit digest differs")
	}
	if _, e := Candidate(request, 501); e == nil {
		t.Fatal("caller-supplied root identity accepted")
	}
	for _, raw := range []string{strings.TrimSuffix(document, "}") + `,"actor":"root"}`, strings.Replace(document, `"action":"create"`, `"action":"create","action":"create"`, 1), strings.Replace(document, `"action":`, `"Action":`, 1)} {
		var invalid Request
		if Decode([]byte(raw), &invalid) == nil {
			t.Fatal("unknown, duplicate or aliased field accepted")
		}
	}
	request.ObservedDigest = request.DesiredDigest
	if request.Validate() == nil {
		t.Fatal("outcome passed as intent")
	}
}

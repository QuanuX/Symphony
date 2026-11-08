package connectionaudit

import (
	"encoding/json"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"strings"
	"testing"
)

func TestConnectionAuditClosedShapeAndIdentity(t *testing.T) {
	r := Request{Protocol, "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333", "sha256:" + strings.Repeat("a", 64), "sha256:" + strings.Repeat("b", 64)}
	for _, uid := range []uint32{0, ^uint32(0)} {
		if _, e := Candidate(r, uid); e == nil {
			t.Fatal("invalid principal")
		}
	}
	c, e := Candidate(r, 501)
	if e != nil {
		t.Fatal(e)
	}
	other, _ := Candidate(r, 502)
	a, _ := stav.CandidateDigest(c)
	b, _ := stav.CandidateDigest(other)
	if a == b {
		t.Fatal("principal not bound")
	}
	raw, _ := json.Marshal(r)
	for _, bad := range []string{strings.TrimSuffix(string(raw), "}") + `,"uid":0}`, strings.Replace(string(raw), `"protocol":`, `"Protocol":`, 1), string(raw) + " {}"} {
		var v Request
		if Decode([]byte(bad), &v) == nil {
			t.Fatal("untrusted actor or malformed JSON admitted")
		}
	}
}

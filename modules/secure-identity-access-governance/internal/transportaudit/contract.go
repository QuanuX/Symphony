// Package transportaudit implements the closed root transport-profile audit
// bridge. The helper and SSIAG retain independent copies and shared vectors;
// neither independently versioned module imports an unpublished owner.
package transportaudit

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"regexp"
)

const Protocol = "symphony.ssiag.transport-profile-audit.v1"
const Endpoint = "/v1/transport-profile/audit"

type Request struct {
	Protocol       string `json:"protocol"`
	RequestID      string `json:"request_id"`
	OperationID    string `json:"operation_id"`
	HostID         string `json:"host_id"`
	TOPSID         string `json:"tops_id"`
	ProfileID      string `json:"profile_id"`
	Action         string `json:"action"`
	PlanDigest     string `json:"plan_digest"`
	PriorDigest    string `json:"prior_digest"`
	DesiredDigest  string `json:"desired_digest"`
	Phase          string `json:"phase"`
	Resolution     string `json:"resolution"`
	ObservedDigest string `json:"observed_digest"`
}
type Response struct {
	Protocol      string       `json:"protocol"`
	RequestDigest string       `json:"request_digest"`
	Receipt       stav.Receipt `json:"receipt"`
}

var digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func (r Request) Validate() error {
	if r.Protocol != Protocol || stav.ValidateRequestUUID(r.RequestID) != nil || stav.ValidateRequestUUID(r.OperationID) != nil || stav.ValidateTOPSID(r.HostID) != nil || stav.ValidateTOPSID(r.TOPSID) != nil || stav.ValidateTOPSID(r.ProfileID) != nil || !digest.MatchString(r.PlanDigest) || !digest.MatchString(r.PriorDigest) || !digest.MatchString(r.DesiredDigest) {
		return fmt.Errorf("transport_audit.invalid")
	}
	switch r.Action {
	case "create", "replace", "enable", "disable", "remove", "policy":
	default:
		return fmt.Errorf("transport_audit.invalid")
	}
	if r.Phase == "intent" && r.Resolution == "pending" && r.ObservedDigest == "" {
		return nil
	}
	if r.Phase == "outcome" && ((r.Resolution == "published" && r.ObservedDigest == r.DesiredDigest) || (r.Resolution == "aborted" && r.ObservedDigest == r.PriorDigest)) {
		return nil
	}
	return fmt.Errorf("transport_audit.invalid")
}
func Hash(r Request) string {
	raw, _ := json.Marshal(r)
	var object map[string]any
	_ = json.Unmarshal(raw, &object)
	body, _ := json.Marshal(object)
	return fmt.Sprintf("sha256:%x", sha256.Sum256(append([]byte(Protocol+"\x00"), body...)))
}
func Candidate(r Request, uid uint32) (stav.Candidate, error) {
	if r.Validate() != nil || uid != 0 {
		return stav.Candidate{}, fmt.Errorf("transport_audit.root_required")
	}
	event, op, outcome, reason := "symphony.sov.transport.profile.intent", "symphony.sov.transport.profile.prepare", "allowed", "symphony.sov.transport.profile.intent.allowed"
	if r.Phase == "outcome" {
		event, op, outcome, reason = "symphony.sov.transport.profile.outcome", "symphony.sov.transport.profile.observe", "succeeded", "symphony.sov.transport.profile."+r.Resolution
	}
	c := stav.Candidate{
		Schema:        stav.SchemaCandidate,
		Actor:         stav.CandidateActor{Principal: stav.SafeReference{ID: "symphony.host.owner.uid.0", Kind: "symphony.identity.host-owner"}, Authentication: stav.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}},
		Configuration: stav.Configuration{PreviousDigest: r.PlanDigest, NewDigest: Hash(r), State: "digests"},
		Correlation:   stav.Correlation{RequestID: r.RequestID, CorrelationID: r.OperationID},
		Operation:     stav.Operation{EventClass: event, OperationID: op, Target: stav.SafeReference{ID: "sov:transport-profile:" + r.ProfileID, Kind: "symphony.sov.transport-profile"}},
		Redaction:     stav.Redaction{Classification: "administrative_metadata"},
		Result:        stav.Result{IntentID: op, Outcome: outcome, ReasonCode: reason},
		Topology:      stav.Topology{TOPSID: r.TOPSID, TROG: stav.TROG{State: "not_applicable", ReasonCode: "symphony.stav.trog.not-applicable"}},
	}
	return c, c.Validate()
}
func (r Response) Validate(request Request) error {
	c, e := Candidate(request, 0)
	if e != nil {
		return e
	}
	want, e := stav.CandidateDigest(c)
	if e != nil || r.Protocol != Protocol || r.RequestDigest != Hash(request) || r.Receipt.Validate() != nil || r.Receipt.Disposition != "committed" || r.Receipt.TOPSID != request.TOPSID || r.Receipt.RequestID != request.RequestID || r.Receipt.CandidateDigest != want {
		return fmt.Errorf("transport_audit.receipt_mismatch")
	}
	return nil
}
func Decode(raw []byte, target any) error {
	if len(raw) > 65536 {
		return fmt.Errorf("transport_audit.invalid")
	}
	canonical, e := stav.Canonicalize(raw)
	if e != nil {
		return fmt.Errorf("transport_audit.invalid")
	}
	d := json.NewDecoder(bytes.NewReader(canonical))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil {
		return fmt.Errorf("transport_audit.invalid")
	}
	encoded, e := stav.MarshalCanonical(target)
	if e != nil || !bytes.Equal(canonical, encoded) {
		return fmt.Errorf("transport_audit.invalid")
	}
	return nil
}

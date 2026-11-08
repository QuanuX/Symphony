package connectionaudit

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const Protocol = "symphony.ssiag.transport-connection-audit.v1"
const Endpoint = "/v1/transport-connection/audit"

type Request struct {
	Protocol         string `json:"protocol"`
	RequestID        string `json:"request_id"`
	TOPSID           string `json:"tops_id"`
	ProfileID        string `json:"profile_id"`
	EndpointDigest   string `json:"endpoint_digest"`
	InvocationDigest string `json:"invocation_digest"`
}
type Response struct {
	Protocol      string       `json:"protocol"`
	RequestDigest string       `json:"request_digest"`
	Receipt       stav.Receipt `json:"receipt"`
}

func hashOK(s string) bool {
	if len(s) != 71 || s[:7] != "sha256:" {
		return false
	}
	for _, c := range s[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func (r Request) Validate() error {
	if r.Protocol != Protocol || stav.ValidateRequestUUID(r.RequestID) != nil || stav.ValidateTOPSID(r.TOPSID) != nil || stav.ValidateTOPSID(r.ProfileID) != nil || !hashOK(r.EndpointDigest) || !hashOK(r.InvocationDigest) {
		return fmt.Errorf("connection_audit.invalid")
	}
	return nil
}
func Hash(r Request) string {
	b, _ := stav.MarshalCanonical(r)
	return fmt.Sprintf("sha256:%x", sha256.Sum256(append([]byte(Protocol+"\x00"), b...)))
}
func Candidate(r Request, uid uint32) (stav.Candidate, error) {
	if r.Validate() != nil || uid == 0 || uid == ^uint32(0) {
		return stav.Candidate{}, fmt.Errorf("connection_audit.user_required")
	}
	c := stav.Candidate{Schema: stav.SchemaCandidate, Actor: stav.CandidateActor{Principal: stav.SafeReference{ID: fmt.Sprintf("symphony.host.user.uid.%d", uid), Kind: "symphony.identity.host-user"}, Authentication: stav.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}}, Configuration: stav.Configuration{PreviousDigest: r.EndpointDigest, NewDigest: Hash(r), State: "digests"}, Correlation: stav.Correlation{RequestID: r.RequestID, CorrelationID: r.RequestID}, Operation: stav.Operation{EventClass: "symphony.sov.transport.connection.intent", OperationID: "symphony.sov.transport.connection.prepare", Target: stav.SafeReference{ID: "sov:transport-profile:" + r.ProfileID, Kind: "symphony.sov.transport-profile"}}, Redaction: stav.Redaction{Classification: "administrative_metadata"}, Result: stav.Result{IntentID: "symphony.sov.transport.connection.prepare", Outcome: "allowed", ReasonCode: "symphony.sov.transport.connection.intent.allowed"}, Topology: stav.Topology{TOPSID: r.TOPSID, TROG: stav.TROG{State: "not_applicable", ReasonCode: "symphony.stav.trog.not-applicable"}}}
	return c, c.Validate()
}
func (r Response) ValidateFor(request Request, uid uint32) error {
	c, e := Candidate(request, uid)
	if e != nil {
		return e
	}
	d, e := stav.CandidateDigest(c)
	if e != nil || r.Protocol != Protocol || r.RequestDigest != Hash(request) || r.Receipt.Validate() != nil || r.Receipt.Disposition != "committed" || r.Receipt.TOPSID != request.TOPSID || r.Receipt.RequestID != request.RequestID || r.Receipt.CandidateDigest != d {
		return fmt.Errorf("connection_audit.receipt_mismatch")
	}
	return nil
}
func Decode(raw []byte, v any) error {
	if len(raw) > 65536 {
		return fmt.Errorf("connection_audit.invalid")
	}
	b, e := stav.Canonicalize(raw)
	if e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(v); e != nil {
		return e
	}
	typed, e := stav.MarshalCanonical(v)
	if e != nil || !bytes.Equal(b, typed) {
		return fmt.Errorf("connection_audit.invalid")
	}
	return nil
}

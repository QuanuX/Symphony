// Package foundationaudit is the closed SSIAG bridge contract for native
// foundational owners. This private wire implementation is mirrored in the
// STAV and SSIAG modules so neither independently published owner gains a new
// unpublished module dependency. Contract vectors must stay identical.
package foundationaudit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const Schema = "symphony.ssiag.foundation-audit.v1"
const Endpoint = "/v1/foundation-lifecycle/audit"

type Request struct {
	Schema              string `json:"schema"`
	Component           string `json:"component"`
	Surface             string `json:"surface"`
	Scope               string `json:"scope"`
	TOPSID              string `json:"tops_id"`
	OperationID         string `json:"operation_id"`
	RequestID           string `json:"request_id"`
	CorrelationID       string `json:"correlation_id"`
	PlanDigest          string `json:"plan_digest"`
	PriorStateDigest    string `json:"prior_state_digest"`
	DesiredState        string `json:"desired_state"`
	AuditMode           string `json:"audit_mode"`
	Phase               string `json:"phase"`
	ObservedStateDigest string `json:"observed_state_digest,omitempty"`
	ObservedAt          string `json:"observed_at,omitempty"`
}

type Response struct {
	Schema        string       `json:"schema"`
	Phase         string       `json:"phase"`
	RequestDigest string       `json:"request_digest"`
	Receipt       stav.Receipt `json:"receipt"`
}

type Submit func(context.Context, Request) (Response, error)

var digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var operation = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

func (r Request) Validate() error {
	if r.Schema != Schema || (r.Component != "ssiag" && r.Component != "stav") || (r.Scope != "user" && r.Scope != "system") || !operation.MatchString(r.OperationID) || !digest.MatchString(r.PlanDigest) || !digest.MatchString(r.PriorStateDigest) {
		return fmt.Errorf("invalid foundational audit binding")
	}
	if err := stav.ValidateTOPSID(r.TOPSID); err != nil {
		return err
	}
	if err := stav.ValidateRequestUUID(r.RequestID); err != nil {
		return err
	}
	if err := stav.ValidateRequestUUID(r.CorrelationID); err != nil {
		return err
	}
	valid := false
	switch r.Surface {
	case "enrollment":
		valid = r.DesiredState == "enrolled" || r.DesiredState == "unenrolled_preserved"
	case "supervisor":
		valid = r.DesiredState == "native_running" || r.DesiredState == "native_installed_stopped" || r.DesiredState == "absent_stopped"
	}
	if !valid || (r.AuditMode != "ordinary" && r.AuditMode != "audit_deferred") {
		return fmt.Errorf("unsupported foundational audit intent")
	}
	switch r.Phase {
	case "intent":
		if r.AuditMode != "ordinary" || r.ObservedAt != "" || r.ObservedStateDigest != "" {
			return fmt.Errorf("intent cannot assert execution evidence")
		}
	case "outcome":
		when, err := time.Parse(time.RFC3339, r.ObservedAt)
		if err != nil || when.UTC().Format(time.RFC3339) != r.ObservedAt || !digest.MatchString(r.ObservedStateDigest) {
			return fmt.Errorf("outcome requires original observed state and time")
		}
	default:
		return fmt.Errorf("unsupported foundational audit phase")
	}
	return nil
}

func Hash(v any) string {
	data, err := stav.MarshalCanonical(v)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func Candidate(r Request, uid uint32) (stav.Candidate, error) {
	if err := r.Validate(); err != nil {
		return stav.Candidate{}, err
	}
	event, op, intent, outcome, reason := "symphony.ssiag.foundation.intent", "symphony.ssiag.foundation.prepare", "symphony.ssiag.foundation.prepare", "allowed", "symphony.ssiag.foundation.intent.allowed"
	if r.Phase == "outcome" {
		event, op, intent, outcome, reason = "symphony.ssiag.foundation.outcome", "symphony.ssiag.foundation.observe", "symphony.ssiag.foundation.observe", "succeeded", "symphony.ssiag.foundation.outcome.observed"
	}
	c := stav.Candidate{
		Schema:        stav.SchemaCandidate,
		Actor:         stav.CandidateActor{Principal: stav.SafeReference{ID: fmt.Sprintf("symphony.host.owner.uid.%d", uid), Kind: "symphony.identity.host-owner"}, Authentication: stav.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}},
		Configuration: stav.Configuration{PreviousDigest: r.PlanDigest, NewDigest: Hash(r), State: "digests"},
		Correlation:   stav.Correlation{RequestID: r.RequestID, CorrelationID: r.CorrelationID},
		Operation:     stav.Operation{EventClass: event, OperationID: op, Target: stav.SafeReference{ID: "symphony.foundation." + r.Component + "." + r.Surface + ":" + r.TOPSID, Kind: "symphony.foundation.lifecycle"}},
		Redaction:     stav.Redaction{Classification: "administrative_metadata"},
		Result:        stav.Result{IntentID: intent, Outcome: outcome, ReasonCode: reason},
		Topology:      stav.Topology{TOPSID: r.TOPSID, TROG: stav.TROG{State: "not_applicable", ReasonCode: "symphony.stav.trog.not-applicable"}},
	}
	return c, c.Validate()
}

func (r Response) Validate(request Request, uid uint32) error {
	c, err := Candidate(request, uid)
	if err != nil {
		return err
	}
	want, err := stav.CandidateDigest(c)
	if err != nil {
		return err
	}
	if r.Schema != Schema || r.Phase != request.Phase || r.RequestDigest != Hash(request) || r.Receipt.Validate() != nil || r.Receipt.Disposition != "committed" || r.Receipt.TOPSID != request.TOPSID || r.Receipt.RequestID != request.RequestID || r.Receipt.CandidateDigest != want {
		return fmt.Errorf("foundational audit receipt is not committed to the exact request and kernel actor")
	}
	return nil
}

func Decode(data []byte, target any) error {
	if len(data) > 65536 {
		return fmt.Errorf("foundational audit value is too large")
	}
	canonical, err := stav.Canonicalize(data)
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(canonical))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing foundational audit data")
	}
	return nil
}

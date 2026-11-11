// Package sessionwire carries only nonsecret controls for the native Access owner.
package transportsession

import (
	"encoding/json"
	"fmt"
	profile "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofile"
	w "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofilewire"
	"regexp"
	"strconv"
	"time"
)

const RequestProtocol = "symphony.sov.transport-session-request.v1"
const NativeProtocol = "symphony.sov.transport-session-native-request.v1"
const ResultProtocol = "symphony.sov.transport-session-native-result.v1"

var Operations = []string{"inspect", "plan", "authenticate", "renew", "clear"}

type Identity struct {
	Audience string `json:"audience"`
	Issuer   string `json:"issuer"`
	Subject  string `json:"subject"`
}
type Intent struct {
	Action   string    `json:"action"`
	Identity *Identity `json:"identity"`
}
type Request struct {
	Protocol   string           `json:"protocol"`
	RequestID  string           `json:"request_id"`
	Operation  string           `json:"operation"`
	Endpoint   profile.Endpoint `json:"endpoint"`
	Intent     *Intent          `json:"intent"`
	PlanDigest *string          `json:"plan_digest"`
	Deadline   string           `json:"deadline_unix_ms"`
}
type Binding struct {
	Application   string `json:"application"`
	SessionRef    string `json:"session_ref"`
	Directory     string `json:"session_directory"`
	ProfileDigest string `json:"profile_digest"`
	EndpointRef   string `json:"endpoint_ref"`
}
type NativeRequest struct {
	Protocol   string  `json:"protocol"`
	RequestID  string  `json:"request_id"`
	Operation  string  `json:"operation"`
	Binding    Binding `json:"binding"`
	Intent     *Intent `json:"intent"`
	PlanDigest *string `json:"plan_digest"`
	Deadline   string  `json:"deadline_unix_ms"`
}
type Result struct {
	Protocol              string  `json:"protocol"`
	RequestID             string  `json:"request_id"`
	Operation             string  `json:"operation"`
	Binding               Binding `json:"binding"`
	HelperDigest          string  `json:"helper_digest"`
	Completed             bool    `json:"completed"`
	Code                  string  `json:"code"`
	State                 string  `json:"state"`
	PlanDigest            *string `json:"plan_digest"`
	Expires               *string `json:"expires_unix_ms"`
	ProviderAuthorization string  `json:"provider_authorization"`
}

func Decode(raw []byte) (Request, error) {
	var r Request
	if len(raw) > 65536 || w.Decode(raw, &r) != nil || r.Protocol != RequestProtocol || !w.ValidRequestID(r.RequestID) || profile.Validate(r.Endpoint) != nil {
		return r, w.ErrBinding
	}
	valid := false
	for _, op := range Operations {
		valid = valid || r.Operation == op
	}
	if !valid {
		return r, w.ErrBinding
	}
	deadline, err := strconv.ParseInt(r.Deadline, 10, 64)
	if err != nil || strconv.FormatInt(deadline, 10) != r.Deadline || deadline <= time.Now().UnixMilli() || deadline > time.Now().Add(5*time.Minute).UnixMilli() {
		return r, w.ErrBinding
	}
	if r.Operation == "inspect" {
		if r.Intent != nil || r.PlanDigest != nil {
			return r, w.ErrBinding
		}
	} else if r.Operation == "plan" {
		if r.PlanDigest != nil || r.Intent == nil {
			return r, w.ErrBinding
		}
		switch r.Intent.Action {
		case "authenticate", "renew":
			if r.Intent.Identity == nil || !validIdentity(*r.Intent.Identity) {
				return r, w.ErrBinding
			}
		case "clear":
			if r.Intent.Identity != nil {
				return r, w.ErrBinding
			}
		default:
			return r, w.ErrBinding
		}
	} else if r.Intent != nil || r.PlanDigest == nil || !w.ValidDigest(*r.PlanDigest) {
		return r, w.ErrBinding
	}
	return r, nil
}
func Native(r Request) NativeRequest {
	return NativeRequest{NativeProtocol, r.RequestID, r.Operation, Binding{r.Endpoint.Application, r.Endpoint.Snapshot.Profile.Configuration.SessionRef, r.Endpoint.SessionDirectory, r.Endpoint.Snapshot.Digest, profile.Ref(r.Endpoint)}, r.Intent, r.PlanDigest, r.Deadline}
}
func ClearOnly(r Request) bool {
	return r.Operation == "inspect" || r.Operation == "clear" || (r.Operation == "plan" && r.Intent != nil && r.Intent.Action == "clear")
}
func DecodeResult(raw []byte, r Request) (Result, error) {
	var v Result
	if len(raw) > 65536 || w.Decode(raw, &v) != nil || v.Protocol != ResultProtocol || v.RequestID != r.RequestID || v.Operation != r.Operation || v.Binding != Native(r).Binding || v.HelperDigest != r.Endpoint.Snapshot.Profile.Configuration.Helper.ExecutableDigest || v.ProviderAuthorization != "not_proven" {
		return v, w.ErrBinding
	}
	switch v.Code {
	case "session.observed", "session.planned", "session.cleared", "session.authenticated", "session.authentication_failed", "session.outcome_unknown", "session.refused":
	default:
		return v, w.ErrBinding
	}
	switch v.State {
	case "locally_usable", "interaction_required", "unknown":
	default:
		return v, w.ErrBinding
	}
	if v.PlanDigest != nil && !w.ValidDigest(*v.PlanDigest) {
		return v, w.ErrBinding
	}
	if r.Operation != "plan" {
		a, _ := json.Marshal(r.PlanDigest)
		b, _ := json.Marshal(v.PlanDigest)
		if string(a) != string(b) {
			return v, w.ErrBinding
		}
	}
	if v.Expires != nil && !w.Decimal(*v.Expires, 9007199254740991, false) {
		return v, w.ErrBinding
	}
	if v.Completed && (v.Code == "session.authentication_failed" || v.Code == "session.outcome_unknown" || v.Code == "session.refused") {
		return v, w.ErrBinding
	}
	if !v.Completed && v.Code != "session.authentication_failed" && v.Code != "session.outcome_unknown" && v.Code != "session.refused" {
		return v, w.ErrBinding
	}
	if (v.State == "locally_usable") != (v.Expires != nil) {
		return v, w.ErrBinding
	}
	if v.Completed {
		expected := map[string]string{"inspect": "session.observed", "plan": "session.planned", "authenticate": "session.authenticated", "renew": "session.authenticated", "clear": "session.cleared"}[r.Operation]
		if v.Code != expected || (r.Operation == "plan" && v.PlanDigest == nil) || (r.Operation == "clear" && v.State != "interaction_required") {
			return v, w.ErrBinding
		}
	}
	if v.Code == "session.authentication_failed" && r.Operation != "authenticate" && r.Operation != "renew" {
		return v, w.ErrBinding
	}
	return v, nil
}
func Failure(r Request, dispatched bool) Result {
	code, state := "session.refused", "interaction_required"
	if dispatched && (r.Operation == "authenticate" || r.Operation == "renew" || r.Operation == "clear") {
		code, state = "session.outcome_unknown", "unknown"
	}
	return Result{ResultProtocol, r.RequestID, r.Operation, Native(r).Binding, r.Endpoint.Snapshot.Profile.Configuration.Helper.ExecutableDigest, false, code, state, r.PlanDigest, nil, "not_proven"}
}
func Error() error {
	return fmt.Errorf("transport session refused; inspect the same private session before retrying")
}

func validIdentity(i Identity) bool {
	return regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(i.Audience) && regexp.MustCompile(`^https://[a-z0-9-]+\.cloudflareaccess\.com$`).MatchString(i.Issuer) && regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:@-]{0,255}$`).MatchString(i.Subject)
}

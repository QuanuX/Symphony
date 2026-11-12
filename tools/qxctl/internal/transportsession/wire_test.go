package transportsession

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	profile "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofile"
	v1 "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fixture() Request {
	d := strings.Repeat("0", 64)
	key := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(append([]byte{0, 0, 0, 11, 's', 's', 'h', '-', 'e', 'd', '2', '5', '5', '1', '9', 0, 0, 0, 32}, make([]byte, 32)...))
	e := profile.Example(v1.Endpoint{Protocol: v1.EndpointProtocol, SSH: v1.Executable{Path: "/usr/bin/ssh", SHA256: d}, SSHExecution: "protected_system_path", Host: "ssh.example.invalid", Port: "22", User: "example", HostPublicKey: key, ClientPublicKey: key, AgentSocket: "/private/agent.sock", TargetLauncher: v1.Executable{Path: "/remote/launcher", SHA256: d}})
	r := Template(e)
	r.Deadline = strconv.FormatInt(time.Now().Add(time.Minute).UnixMilli(), 10)
	return r
}
func TestSessionClosedControls(t *testing.T) {
	r := fixture()
	raw, _ := json.Marshal(r)
	if _, e := Decode(raw); e != nil {
		t.Fatal(e)
	}
	for name, edit := range map[string]func(*Request){"deadline": func(r *Request) { r.Deadline = "01" }, "endpoint": func(r *Request) { r.Endpoint.Protocol = v1.EndpointProtocol }, "root-login": func(r *Request) { r.Operation = "sudo" }, "plan-binding": func(r *Request) { r.Operation = "renew" }, "inspect-intent": func(r *Request) { r.Intent = &Intent{Action: "clear"} }, "identity": func(r *Request) {
		r.Operation = "plan"
		r.Intent = &Intent{Action: "authenticate", Identity: &Identity{}}
	}} {
		t.Run(name, func(t *testing.T) {
			r := fixture()
			edit(&r)
			b, _ := json.Marshal(r)
			if _, e := Decode(b); e == nil {
				t.Fatal("accepted malformed request")
			}
		})
	}
	for _, raw := range [][]byte{append([]byte(`{"protocol":"duplicate",`), raw[1:]...), bytes.Replace(raw, []byte(`"intent":null`), []byte(`"intent":null,"token":"SECRET"`), 1)} {
		if _, e := Decode(raw); e == nil {
			t.Fatal("accepted unknown/duplicate field")
		}
	}
	r.Operation = "plan"
	r.Intent = &Intent{Action: "clear"}
	if !ClearOnly(r) {
		t.Fatal("cleanup re-admits")
	}
	r.Intent = &Intent{Action: "authenticate", Identity: &Identity{strings.Repeat("a", 64), "https://team.cloudflareaccess.com", "user"}}
	raw, _ = json.Marshal(r)
	if _, e := Decode(raw); e != nil || ClearOnly(r) {
		t.Fatal("auth admission")
	}
}
func TestSessionStrictResultAndUncertainDispatch(t *testing.T) {
	r := fixture()
	v := Failure(r, false)
	raw, _ := json.Marshal(v)
	if _, e := DecodeResult(raw, r); e != nil {
		t.Fatal(e)
	}
	v.Completed = true
	v.Code = "session.observed"
	raw, _ = json.Marshal(v)
	if _, e := DecodeResult(raw, r); e != nil {
		t.Fatal(e)
	}
	for _, edit := range []func(*Result){func(v *Result) { v.Binding.Application = "other" }, func(v *Result) { v.Code = "SECRET" }, func(v *Result) { v.ProviderAuthorization = "allowed" }, func(v *Result) { v.State = "locally_usable" }, func(v *Result) { v.Code = "session.authenticated" }} {
		bad := v
		edit(&bad)
		raw, _ = json.Marshal(bad)
		if _, e := DecodeResult(raw, r); e == nil {
			t.Fatal("accepted altered native evidence")
		}
	}
	for _, op := range []string{"authenticate", "renew", "clear"} {
		r.Operation = op
		d := "sha256:" + strings.Repeat("a", 64)
		r.PlanDigest = &d
		v = Failure(r, true)
		if v.Code != "session.outcome_unknown" {
			t.Fatal(op)
		}
		raw, _ = json.Marshal(v)
		if _, e := DecodeResult(raw, r); e != nil {
			t.Fatal(e)
		}
	}
}

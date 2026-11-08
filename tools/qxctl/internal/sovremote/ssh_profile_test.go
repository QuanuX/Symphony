package sovremote

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	p "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofile"
	v1 "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
)

func TestSSHProfileWireBindingAndSchemas(t *testing.T) {
	value, err := Template("ssh_invoke_v2")
	if err != nil {
		t.Fatal(err)
	}
	in := value.(p.Input)
	raw := encoded(t, in)
	if _, err = p.DecodeInput(raw); err != nil {
		t.Fatal(err)
	}
	if _, err = DecodeSSHInvocation(raw); err == nil {
		t.Fatal("v1 acquired v2 behavior")
	}
	inv, err := DecodeInvocation(in.Invocation)
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := p.TargetCommand(in)
	want, _ := InvocationDigest(inv)
	if err != nil || got != want {
		t.Fatal("independent invocation digests differ", err, got, want)
	}
	for _, op := range []string{"ssh_profile_selection", "ssh_endpoint_v2", "ssh_invoke_v2"} {
		v, e := Template(op)
		if e != nil {
			t.Fatal(e)
		}
		schema, e := Schema(op)
		if e != nil || !strings.Contains(string(encoded(t, schema)), "additionalProperties") {
			t.Fatal("closed schema missing", op, e)
		}
		if !json.Valid(encoded(t, v)) {
			t.Fatal(op)
		}
	}
	for name, edit := range map[string]func(*p.Input){
		"application": func(x *p.Input) { x.Endpoint.Application = "other.invalid" },
		"client":      func(x *p.Input) { x.Endpoint.Direct.ClientPublicKey += " comment" },
		"launcher":    func(x *p.Input) { x.Endpoint.Direct.TargetLauncher.Path = "/other/launcher" },
		"session":     func(x *p.Input) { x.Endpoint.SessionDirectory = "/other/session" },
		"profile":     func(x *p.Input) { x.Endpoint.Snapshot.Profile.Generation = "2" },
		"protocol":    func(x *p.Input) { x.Protocol = v1.InputProtocol },
	} {
		t.Run(name, func(t *testing.T) {
			x := in
			edit(&x)
			if _, e := p.DecodeInput(encoded(t, x)); e == nil {
				t.Fatal("binding drift admitted")
			}
		})
	}
	result, err := RunSSHProfile(context.Background(), raw)
	if err == nil || result.TargetResult != nil || result.AutomaticRetry {
		t.Fatal("uninstalled profile dispatched", result, err)
	}
	if os.Geteuid() != 0 && result.TransportOutcome != "not_dispatched" {
		t.Fatal(result)
	}
}

func TestSSHProfileReplyIndependentNativeValidation(t *testing.T) {
	value, _ := Template("ssh_invoke_v2")
	in := value.(p.Input)
	inv, _ := DecodeInvocation(in.Invocation)
	digest, _ := InvocationDigest(inv)
	base := ProfileInvocationResult{Protocol: "symphony.sov.ssh-invoke.v2", RequestID: inv.RequestID, InvocationSHA256: digest, PlanSHA256: inv.ExpectedPlanSHA256, Endpoint: profileEndpoint(in.Endpoint), TransportOutcome: "response_lost", ReasonCode: "transport_profile.response_uncertain", OriginVerification: "not_verified"}
	code := "native_operation_failed"
	target := TargetResult{Protocol: "symphony.sov.target-result.v1", RequestID: inv.RequestID, InvocationSHA256: digest, ConfigurationSHA256: inv.Endpoint.ConfigurationSHA256, Target: inv.Selection.Target, AuthorityMode: "owner_filesystem", Operation: inv.Operation, DispatchStarted: true, Status: "native_error", ErrorCode: &code, Result: json.RawMessage("null")}
	reply := p.Reply{Protocol: p.ReplyProtocol, EndpointSHA256: base.Endpoint.SHA256, InvocationSHA256: digest, HostKeySHA256: base.Endpoint.HostKeySHA256, ClientKeySHA256: base.Endpoint.ClientKeySHA256, ReasonCode: "transport_profile.response_received", TransportOutcome: "response_received", TargetResult: encoded(t, target)}
	for _, mode := range []string{"valid", "endpoint", "invocation", "host", "client", "target", "malformed", "lost", "refused", "session", "secret-reason"} {
		t.Run(mode, func(t *testing.T) {
			r := reply
			switch mode {
			case "endpoint":
				r.EndpointSHA256 = strings.Repeat("f", 64)
			case "invocation":
				r.InvocationSHA256 = strings.Repeat("f", 64)
			case "host":
				r.HostKeySHA256 = strings.Repeat("f", 64)
			case "client":
				r.ClientKeySHA256 = strings.Repeat("f", 64)
			case "target":
				bad := target
				bad.RequestID = "00000000-0000-4000-8000-000000000099"
				r.TargetResult = encoded(t, bad)
			case "malformed":
				r.TargetResult = json.RawMessage(`{"untrusted":"response"}`)
			case "lost":
				r.TransportOutcome = "response_lost"
			case "refused", "session", "secret-reason":
				r.TransportOutcome = "not_dispatched"
				r.TargetResult = json.RawMessage("null")
				r.ReasonCode = "transport_profile.admission_refused"
				if mode == "session" {
					r.ReasonCode = "transport_profile.interaction_required"
				}
				if mode == "secret-reason" {
					r.ReasonCode = "untrusted provider error"
				}
			}
			got := profileReply(base, inv, encoded(t, r))
			switch mode {
			case "valid":
				if got.TransportOutcome != "response_received" || got.TargetResult == nil || got.TargetResult.Status != "native_error" || got.OriginVerification != "selected_adapter_authenticated_ssh_host" {
					t.Fatal(got)
				}
			case "refused", "session":
				if got.TransportOutcome != "not_dispatched" || got.TargetResult != nil || got.ReasonCode != r.ReasonCode {
					t.Fatal(got)
				}
			default:
				if got.TransportOutcome != "response_lost" || got.TargetResult != nil || got.OriginVerification != "not_verified" {
					t.Fatal(got)
				}
			}
			if got.AutomaticRetry {
				t.Fatal("automatic replay")
			}
		})
	}
}

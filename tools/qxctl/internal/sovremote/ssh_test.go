package sovremote

import (
	"context"
	"encoding/json"
	ssh "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func sshFixtureMain() {
	raw, _ := io.ReadAll(os.Stdin)
	in, err := DecodeSSHInvocation(raw)
	if err != nil {
		os.Exit(2)
	}
	mode := os.Getenv("SOV_SSH_FIXTURE")
	if mode == "lost" {
		os.Exit(0)
	}
	if mode == "wait" {
		time.Sleep(10 * time.Second)
		os.Exit(1)
	}
	d, _ := InvocationDigest(in.Invocation)
	code := "native_operation_failed"
	target := TargetResult{Protocol: "symphony.sov.target-result.v1", RequestID: in.Invocation.RequestID, InvocationSHA256: d, ConfigurationSHA256: in.Invocation.Endpoint.ConfigurationSHA256, Target: in.Invocation.Selection.Target, AuthorityMode: "owner_filesystem", Operation: in.Invocation.Operation, DispatchStarted: true, Status: "native_error", ErrorCode: &code, Result: json.RawMessage("null")}
	b, _ := json.Marshal(target)
	h, _ := ssh.PublicKeyDigest(in.Endpoint.HostPublicKey)
	k, _ := ssh.PublicKeyDigest(in.Endpoint.ClientPublicKey)
	r := ssh.Reply{Protocol: ssh.ReplyProtocol, EndpointSHA256: ssh.EndpointDigest(in.Endpoint), InvocationSHA256: d, HostKeySHA256: h, ClientKeySHA256: k, TargetResult: b}
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
		target.RequestID = "00000000-0000-4000-8000-000000000099"
		r.TargetResult, _ = json.Marshal(target)
	}
	_ = json.NewEncoder(os.Stdout).Encode(r)
	os.Exit(0)
}
func TestSSHProcessReplyBindingAndLoss(t *testing.T) {
	self, _ := os.Executable()
	b, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "lost", "endpoint", "invocation", "host", "client", "target", "wait"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("SOV_SSH_FIXTURE", mode)
			v, _ := Template("ssh_invoke")
			in := v.(SSHInvocation)
			in.Invocation.Adapter = Executable{Path: self, SHA256: Digest(b)}
			in.Invocation.Selection.Transport.AdapterSHA256 = in.Invocation.Adapter.SHA256
			replan(t, &in.Invocation)
			ctx := context.Background()
			if mode == "wait" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 250*time.Millisecond)
				defer cancel()
			}
			r, err := RunSSH(ctx, encoded(t, in))
			if err != nil {
				t.Fatal(err)
			}
			if r.AutomaticRetry {
				t.Fatal("automatic retry")
			}
			if mode == "valid" {
				if r.TransportOutcome != "response_received" || r.OriginVerification != "selected_adapter_authenticated_ssh_host" || r.TargetResult == nil || r.TargetResult.Status != "native_error" {
					t.Fatal(r)
				}
			} else {
				if r.TransportOutcome != "response_lost" || r.OriginVerification != "not_verified" || r.TargetResult != nil {
					t.Fatal(r)
				}
			}
		})
	}
}

func TestSSHBindingAndCanonicalParity(t *testing.T) {
	v, err := Template("ssh_invoke")
	if err != nil {
		t.Fatal(err)
	}
	in := v.(SSHInvocation)
	if _, err = DecodeSSHInvocation(encoded(t, in)); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(in.Invocation)
	_, got, err := ssh.TargetCommand(ssh.Input{Protocol: in.Protocol, Endpoint: in.Endpoint, Invocation: raw})
	want, _ := InvocationDigest(in.Invocation)
	if err != nil || got != want {
		t.Fatal("carrier/native invocation digest differs", err, got, want)
	}
	for name, edit := range map[string]func(*SSHInvocation){"host": func(v *SSHInvocation) { v.Endpoint.Host = "changed.invalid" }, "key": func(v *SSHInvocation) { v.Endpoint.ClientPublicKey += " comment" }, "adapter-id": func(v *SSHInvocation) { v.Invocation.Selection.Transport.AdapterID = "other"; replan(t, &v.Invocation) }, "adapter-version": func(v *SSHInvocation) {
		v.Invocation.Selection.Transport.AdapterVersion = "2.0.0"
		replan(t, &v.Invocation)
	}, "family": func(v *SSHInvocation) { v.Invocation.Selection.Transport.Family = "ipc"; replan(t, &v.Invocation) }, "extra-resources": func(v *SSHInvocation) {
		v.Invocation.Selection.Resources.MemoryBytes = ptr(strings.Repeat("9", 100))
		replan(t, &v.Invocation)
	}} {
		t.Run(name, func(t *testing.T) {
			x := in
			edit(&x)
			if _, e := DecodeSSHInvocation(encoded(t, x)); e == nil {
				t.Fatal("unbound selection admitted")
			}
		})
	}
	result, err := RunSSH(context.Background(), encoded(t, in))
	if err == nil || result.TransportOutcome != "not_dispatched" || result.OriginVerification != "not_verified" || result.AutomaticRetry {
		t.Fatal("missing adapter claimed dispatch", result, err)
	}
}
func TestSSHEndpointAllFieldsAndSchema(t *testing.T) {
	v, _ := Template("ssh_endpoint")
	raw := encoded(t, v)
	out, err := InspectSSHEndpoint(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.EndpointRef != ssh.EndpointRef(out.Endpoint) || out.SHA256 != ssh.EndpointDigest(out.Endpoint) {
		t.Fatal("endpoint binding")
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	delete(m, "connect_timeout_seconds")
	if _, e := InspectSSHEndpoint(encoded(t, m)); e == nil {
		t.Fatal("missing explicit null")
	}
	for _, op := range []string{"ssh_endpoint", "ssh_invoke"} {
		s, e := Schema(op)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(string(encoded(t, s)), "additionalProperties") {
			t.Fatal("no closed schema")
		}
	}
}

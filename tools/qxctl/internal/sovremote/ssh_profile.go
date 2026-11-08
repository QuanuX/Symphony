package sovremote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofile"
	ssh "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	"io"
	"os"
	"os/exec"
	"time"
)

type ProfileEndpointResult struct {
	Protocol        string              `json:"protocol"`
	Endpoint        sshprofile.Endpoint `json:"endpoint"`
	SHA256          string              `json:"sha256"`
	EndpointRef     string              `json:"endpoint_ref"`
	HostKeySHA256   string              `json:"host_key_sha256"`
	ClientKeySHA256 string              `json:"client_key_sha256"`
}
type ProfileInvocationResult struct {
	Protocol           string                `json:"protocol"`
	RequestID          string                `json:"request_id"`
	InvocationSHA256   string                `json:"invocation_sha256"`
	PlanSHA256         string                `json:"plan_sha256"`
	Endpoint           ProfileEndpointResult `json:"endpoint"`
	ReasonCode         string                `json:"reason_code"`
	TransportOutcome   string                `json:"transport_outcome"`
	OriginVerification string                `json:"origin_verification"`
	AutomaticRetry     bool                  `json:"automatic_retry"`
	TargetResult       *TargetResult         `json:"target_result"`
}

func profileEndpoint(e sshprofile.Endpoint) ProfileEndpointResult {
	h, _ := ssh.PublicKeyDigest(e.Direct.HostPublicKey)
	k, _ := ssh.PublicKeyDigest(e.Direct.ClientPublicKey)
	return ProfileEndpointResult{"symphony.sov.ssh-endpoint-result.v2", e, sshprofile.Digest(e), sshprofile.Ref(e), h, k}
}
func PlanSSHProfile(raw []byte, name string) (ProfileEndpointResult, error) {
	e, err := sshprofile.Plan(raw, name)
	if err != nil {
		return ProfileEndpointResult{}, err
	}
	return profileEndpoint(e), nil
}
func IsSSHProfile(raw []byte) bool {
	var p struct {
		Protocol string `json:"protocol"`
	}
	return json.Unmarshal(raw, &p) == nil && p.Protocol == sshprofile.InputProtocol
}
func RunSSHProfile(ctx context.Context, raw []byte) (ProfileInvocationResult, error) {
	var out ProfileInvocationResult
	in, err := sshprofile.DecodeInput(raw)
	if err != nil || os.Geteuid() == 0 {
		return out, fmt.Errorf("profile invocation refused")
	}
	inv, err := DecodeInvocation(in.Invocation)
	if err != nil {
		return out, err
	}
	digest, _ := InvocationDigest(inv)
	out = ProfileInvocationResult{Protocol: "symphony.sov.ssh-invoke.v2", RequestID: inv.RequestID, InvocationSHA256: digest, PlanSHA256: inv.ExpectedPlanSHA256, Endpoint: profileEndpoint(in.Endpoint), TransportOutcome: "not_dispatched", ReasonCode: "transport_profile.admission_refused", OriginVerification: "not_verified"}
	s, err := sshprofile.Open(in.Endpoint.Scope)
	if err != nil {
		return out, err
	}
	defer s.Close()
	err = s.With(in.Endpoint, func(carrier, _, _ string) error {
		if inv.Adapter.Path != carrier || "sha256:"+inv.Adapter.SHA256 != in.Endpoint.Snapshot.Profile.Configuration.Adapter.ExecutableDigest {
			return fmt.Errorf("carrier binding mismatch")
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	ctx, cancel := withDeadline(ctx, inv.Selection.Resources.DeadlineUnixMS)
	defer cancel()
	cmd := exec.CommandContext(ctx, inv.Adapter.Path)
	processCancellation(cmd)
	// The carrier must finish its own SSH group cleanup before the controller
	// forces carrier termination. The admitted profile fixes this bound at 5s.
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	cmd.Stdin = bytes.NewReader(raw)
	cmd.Stderr = io.Discard
	var b boundedBuffer
	cmd.Stdout = &b
	if err = cmd.Start(); err != nil {
		return out, err
	}
	defer cleanupProcess(cmd)
	out.TransportOutcome = "response_lost"
	out.ReasonCode = "transport_profile.response_uncertain"
	if cmd.Wait() != nil || b.overflow {
		return out, nil
	}
	return profileReply(out, inv, b.Bytes()), nil
}

func profileReply(out ProfileInvocationResult, inv Invocation, raw []byte) ProfileInvocationResult {
	var reply sshprofile.Reply
	if sshprofile.Decode(raw, &reply) != nil || reply.Protocol != sshprofile.ReplyProtocol || reply.EndpointSHA256 != out.Endpoint.SHA256 || reply.InvocationSHA256 != out.InvocationSHA256 || reply.HostKeySHA256 != out.Endpoint.HostKeySHA256 || reply.ClientKeySHA256 != out.Endpoint.ClientKeySHA256 {
		return out
	}
	if reply.TransportOutcome == "not_dispatched" && bytes.Equal(bytes.TrimSpace(reply.TargetResult), []byte("null")) {
		if reply.ReasonCode != "transport_profile.admission_refused" && reply.ReasonCode != "transport_profile.interaction_required" {
			return out
		}
		out.ReasonCode = reply.ReasonCode
		out.TransportOutcome = "not_dispatched"
		return out
	}
	if reply.TransportOutcome != "response_received" || reply.ReasonCode != "transport_profile.response_received" {
		return out
	}
	var target TargetResult
	if decode(reply.TargetResult, &target) != nil || validateTargetResult(inv, out.InvocationSHA256, target) != nil {
		return out
	}
	out.TransportOutcome = "response_received"
	out.ReasonCode = reply.ReasonCode
	out.OriginVerification = "selected_adapter_authenticated_ssh_host"
	out.TargetResult = &target
	return out
}

func sshProfileTemplate(operation string) (any, error) {
	v, err := sshTemplate("ssh_endpoint")
	if err != nil {
		return nil, err
	}
	direct := v.(ssh.Endpoint)
	endpoint := sshprofile.Example(direct)
	if operation == "ssh_profile_selection" {
		return sshprofile.Selection{Protocol: sshprofile.SelectionProtocol, Direct: direct, Scope: endpoint.Scope, Application: endpoint.Application, SessionDirectory: endpoint.SessionDirectory, JournalDirectory: endpoint.JournalDirectory}, nil
	}
	if operation == "ssh_endpoint_v2" {
		return endpoint, nil
	}
	v, err = Template("invoke")
	if err != nil {
		return nil, err
	}
	in := v.(Invocation)
	in.Adapter.Path = "/example/protected-carrier"
	in.Selection.Transport.Family = "scoped_cli"
	in.Selection.Transport.AdapterID = ssh.AdapterID
	in.Selection.Transport.AdapterVersion = sshprofile.AdapterVersion
	in.Selection.Transport.EndpointRef = sshprofile.Ref(endpoint)
	plan, err := build(in.Selection)
	if err != nil {
		return nil, err
	}
	in.ExpectedPlanSHA256 = plan.PlanSHA256
	raw, _ := json.Marshal(in)
	return sshprofile.Input{Protocol: sshprofile.InputProtocol, Endpoint: endpoint, Invocation: raw}, nil
}

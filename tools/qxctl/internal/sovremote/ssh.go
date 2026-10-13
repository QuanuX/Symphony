package sovremote

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	ssh "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
)

type SSHInvocation struct {
	Protocol   string       `json:"protocol"`
	Endpoint   ssh.Endpoint `json:"endpoint"`
	Invocation Invocation   `json:"invocation"`
}
type SSHEndpointResult struct {
	Protocol        string       `json:"protocol"`
	Endpoint        ssh.Endpoint `json:"endpoint"`
	SHA256          string       `json:"sha256"`
	EndpointRef     string       `json:"endpoint_ref"`
	HostKeySHA256   string       `json:"host_key_sha256"`
	ClientKeySHA256 string       `json:"client_key_sha256"`
}
type SSHInvocationResult struct {
	Protocol           string            `json:"protocol"`
	RequestID          string            `json:"request_id"`
	InvocationSHA256   string            `json:"invocation_sha256"`
	PlanSHA256         string            `json:"plan_sha256"`
	Endpoint           SSHEndpointResult `json:"endpoint"`
	TransportOutcome   string            `json:"transport_outcome"`
	OriginVerification string            `json:"origin_verification"`
	AutomaticRetry     bool              `json:"automatic_retry"`
	TargetResult       *TargetResult     `json:"target_result"`
}

func inspectSSHEndpoint(e ssh.Endpoint) SSHEndpointResult {
	h, _ := ssh.PublicKeyDigest(e.HostPublicKey)
	k, _ := ssh.PublicKeyDigest(e.ClientPublicKey)
	return SSHEndpointResult{Protocol: "symphony.sov.ssh-endpoint-result.v1", Endpoint: e, SHA256: ssh.EndpointDigest(e), EndpointRef: ssh.EndpointRef(e), HostKeySHA256: h, ClientKeySHA256: k}
}
func InspectSSHEndpoint(raw []byte) (SSHEndpointResult, error) {
	var e ssh.Endpoint
	if err := decode(raw, &e); err != nil {
		return SSHEndpointResult{}, err
	}
	if err := ssh.Validate(e); err != nil {
		return SSHEndpointResult{}, err
	}
	return inspectSSHEndpoint(e), nil
}
func DecodeSSHInvocation(raw []byte) (SSHInvocation, error) {
	var in SSHInvocation
	if err := decode(raw, &in); err != nil {
		return in, err
	}
	if in.Protocol != ssh.InputProtocol {
		return in, fmt.Errorf("wrong SSH invocation protocol")
	}
	if err := ssh.Validate(in.Endpoint); err != nil {
		return in, err
	}
	b, _ := json.Marshal(in.Invocation)
	if _, err := DecodeInvocation(b); err != nil {
		return in, err
	}
	_, _, err := ssh.TargetCommand(ssh.Input{Protocol: in.Protocol, Endpoint: in.Endpoint, Invocation: b})
	return in, err
}
func RunSSH(ctx context.Context, raw []byte) (SSHInvocationResult, error) {
	in, err := DecodeSSHInvocation(raw)
	if err != nil {
		return SSHInvocationResult{}, err
	}
	digest, _ := InvocationDigest(in.Invocation)
	out := SSHInvocationResult{Protocol: "symphony.sov.ssh-invoke.v1", RequestID: in.Invocation.RequestID, InvocationSHA256: digest, PlanSHA256: in.Invocation.ExpectedPlanSHA256, Endpoint: inspectSSHEndpoint(in.Endpoint), TransportOutcome: "not_dispatched", OriginVerification: "not_verified"}
	executable, cleanup, err := StageExecutable(in.Invocation.Adapter)
	if err != nil {
		return out, err
	}
	defer cleanup()
	ctx, cancel := withDeadline(ctx, in.Invocation.Selection.Resources.DeadlineUnixMS)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable)
	processCancellation(cmd)
	b, _ := json.Marshal(in)
	cmd.Stdin = bytes.NewReader(b)
	cmd.Stderr = io.Discard
	var response boundedBuffer
	cmd.Stdout = &response
	if err = cmd.Start(); err != nil {
		return out, fmt.Errorf("SSH adapter could not start")
	}
	defer cleanupProcess(cmd)
	out.TransportOutcome = "response_lost"
	if err = cmd.Wait(); err != nil || response.overflow {
		return out, nil
	}
	var reply ssh.Reply
	if decode(response.Bytes(), &reply) != nil || reply.Protocol != ssh.ReplyProtocol || reply.EndpointSHA256 != out.Endpoint.SHA256 || reply.InvocationSHA256 != digest || reply.HostKeySHA256 != out.Endpoint.HostKeySHA256 || reply.ClientKeySHA256 != out.Endpoint.ClientKeySHA256 {
		return out, nil
	}
	var target TargetResult
	if decode(reply.TargetResult, &target) != nil || validateTargetResult(in.Invocation, digest, target) != nil {
		return out, nil
	}
	out.TransportOutcome = "response_received"
	out.OriginVerification = "selected_adapter_authenticated_ssh_host"
	out.TargetResult = &target
	return out, nil
}
func sshTemplate(operation string) (any, error) {
	// Public placeholder key only; no private credential or discovered host.
	key := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(append([]byte{0, 0, 0, 11, 's', 's', 'h', '-', 'e', 'd', '2', '5', '5', '1', '9', 0, 0, 0, 32}, make([]byte, 32)...))
	d := strings.Repeat("0", 64)
	e := ssh.Endpoint{Protocol: ssh.EndpointProtocol, SSHExecution: "protected_system_path", SSH: ssh.Executable{Path: "/usr/bin/ssh", SHA256: d}, Host: "example.invalid", Port: "22", User: "example", HostPublicKey: key, ClientPublicKey: key, AgentSocket: "/example/agent.sock", TargetLauncher: ssh.Executable{Path: "/example/sov-stdio-adapter", SHA256: d}}
	if operation == "ssh_endpoint" {
		return e, nil
	}
	v, err := Template("invoke")
	if err != nil {
		return nil, err
	}
	in := v.(Invocation)
	in.Adapter.Path = "/example/sov-ssh-adapter"
	in.Selection.Transport.Family = "scoped_cli"
	in.Selection.Transport.AdapterID = ssh.AdapterID
	in.Selection.Transport.AdapterVersion = ssh.Version
	in.Selection.Transport.EndpointRef = ssh.EndpointRef(e)
	p, err := build(in.Selection)
	if err != nil {
		return nil, err
	}
	in.ExpectedPlanSHA256 = p.PlanSHA256
	return SSHInvocation{Protocol: ssh.InputProtocol, Endpoint: e, Invocation: in}, nil
}

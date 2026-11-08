// Package profile owns the explicit v2 carrier contract. The v1 contract is
// embedded as target/SSH data only and is never reinterpreted by a v1 reader.
package sshprofile

import (
	"encoding/json"
	"fmt"
	c "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofilewire"
	v1 "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	"strings"
)

const EndpointProtocol = "symphony.sov.ssh-endpoint.v2"
const InputProtocol = "symphony.sov.ssh-invoke-input.v2"
const ReplyProtocol = "symphony.sov.ssh-carrier-result.v2"
const AdapterVersion = v1.Version

type Endpoint struct {
	Protocol         string      `json:"protocol"`
	Direct           v1.Endpoint `json:"direct"`
	Scope            c.Scope     `json:"scope"`
	Application      string      `json:"application"`
	SessionDirectory string      `json:"session_directory"`
	JournalDirectory string      `json:"journal_directory"`
	Snapshot         c.Snapshot  `json:"snapshot"`
}
type Selection struct {
	Protocol         string      `json:"protocol"`
	Direct           v1.Endpoint `json:"direct"`
	Scope            c.Scope     `json:"scope"`
	Application      string      `json:"application"`
	SessionDirectory string      `json:"session_directory"`
	JournalDirectory string      `json:"journal_directory"`
}

const SelectionProtocol = "symphony.sov.ssh-profile-selection.v1"

type Input struct {
	Protocol   string          `json:"protocol"`
	Endpoint   Endpoint        `json:"endpoint"`
	Invocation json.RawMessage `json:"invocation"`
}
type Reply struct {
	Protocol         string          `json:"protocol"`
	EndpointSHA256   string          `json:"endpoint_sha256"`
	InvocationSHA256 string          `json:"invocation_sha256"`
	HostKeySHA256    string          `json:"host_key_sha256"`
	ClientKeySHA256  string          `json:"client_key_sha256"`
	ReasonCode       string          `json:"reason_code"`
	TransportOutcome string          `json:"transport_outcome"`
	TargetResult     json.RawMessage `json:"target_result"`
}

func Decode(raw []byte, into any) error { return c.Decode(raw, into) }
func Digest(e Endpoint) string {
	b, _ := json.Marshal(e)
	b, _ = v1.Canonical(b)
	return v1.Digest(append([]byte(EndpointProtocol+"\x00"), b...))
}
func Ref(e Endpoint) string { return "sov:ssh:v2:sha256:" + Digest(e) }
func Validate(e Endpoint) error {
	if e.Protocol != EndpointProtocol || v1.Validate(e.Direct) != nil || e.Direct.SSHExecution != "protected_system_path" || !c.ValidScope(e.Scope) || c.ValidateSnapshot(e.Snapshot) != nil || e.Snapshot.Profile.Scope != e.Scope || e.Snapshot.Profile.State != "enabled" || e.SessionDirectory == e.JournalDirectory {
		return c.ErrBinding
	}
	config := e.Snapshot.Profile.Configuration
	if !safeAbsolute(e.SessionDirectory) || !safeAbsolute(e.JournalDirectory) {
		return c.ErrBinding
	}
	found := false
	for _, a := range config.Applications {
		if a == e.Application {
			found = true
		}
	}
	if !found {
		return c.ErrBinding
	}
	host, err := v1.PublicKeyDigest(e.Direct.HostPublicKey)
	if err != nil {
		return err
	}
	found = false
	for _, target := range config.Endpoints {
		if target.Host == e.Direct.Host && target.Port == e.Direct.Port && target.User == e.Direct.User && target.HostKeyDigest == "sha256:"+host {
			found = true
		}
	}
	if !found {
		return c.ErrBinding
	}
	if config.SSH.ExecutableDigest != "sha256:"+e.Direct.SSH.SHA256 {
		return c.ErrBinding
	}
	return nil
}
func DecodeInput(raw []byte) (Input, error) {
	var in Input
	if Decode(raw, &in) != nil || in.Protocol != InputProtocol || Validate(in.Endpoint) != nil {
		return in, c.ErrBinding
	}
	_, _, err := TargetCommand(in)
	return in, err
}

// The remote invocation and target launcher remain v1; only the complete opaque
// endpoint reference and exact carrier generation differ in the selection.
func TargetCommand(in Input) (string, string, error) {
	if in.Protocol != InputProtocol || Validate(in.Endpoint) != nil || v1.CheckJSON(in.Invocation) != nil {
		return "", "", c.ErrBinding
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(in.Invocation, &object) != nil {
		return "", "", c.ErrBinding
	}
	var selection map[string]json.RawMessage
	if json.Unmarshal(object["selection"], &selection) != nil {
		return "", "", c.ErrBinding
	}
	var transport map[string]any
	if json.Unmarshal(selection["transport"], &transport) != nil {
		return "", "", c.ErrBinding
	}
	if transport["adapter_version"] != AdapterVersion || transport["endpoint_ref"] != Ref(in.Endpoint) {
		return "", "", c.ErrBinding
	}
	// Validate all existing target bindings with the unchanged v1 implementation.
	transport["adapter_version"] = v1.Version
	transport["endpoint_ref"] = v1.EndpointRef(in.Endpoint.Direct)
	selection["transport"], _ = json.Marshal(transport)
	object["selection"], _ = json.Marshal(selection)
	normalized, _ := json.Marshal(object)
	command, _, err := v1.TargetCommand(v1.Input{Protocol: v1.InputProtocol, Endpoint: in.Endpoint.Direct, Invocation: normalized})
	b, _ := v1.Canonical(in.Invocation)
	return command, v1.Digest(append([]byte("symphony.sov.invocation.v1\x00"), b...)), err
}

// ProxyCommand is a fixed Symphony wrapper protocol, not a vendor CLI passthrough.
// I5 must qualify a wrapper implementing it; no implicit login is permitted.
func ProxyCommand(path string, e Endpoint) (string, error) {
	if Validate(e) != nil || !safeAbsolute(path) {
		return "", c.ErrBinding
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	args := []string{path, "--application", e.Application, "--session-directory", e.SessionDirectory, "--session-ref", e.Snapshot.Profile.Configuration.SessionRef}
	for i, a := range args {
		if strings.ContainsAny(a, "%\r\n\x00") {
			return "", fmt.Errorf("proxy expansion refused")
		}
		args[i] = quote(a)
	}
	// OpenSSH supplies the shell exec prefix itself.
	return strings.Join(args, " "), nil
}

// Example contains synthetic, uninstalled references for discovery only.
func Example(direct v1.Endpoint) Endpoint {
	request := c.Template()
	config := *request.Change.Configuration
	h, _ := v1.PublicKeyDigest(direct.HostPublicKey)
	config.Endpoints = []c.EndpointConstraint{{Host: direct.Host, Port: direct.Port, User: direct.User, HostKeyDigest: "sha256:" + h}}
	config.SSH.ExecutableDigest = "sha256:" + direct.SSH.SHA256
	snapshot := c.Snapshot{Protocol: c.SnapshotProtocol, Profile: c.Profile{Protocol: c.ProfileProtocol, Scope: request.Scope, ProfileID: *request.ProfileID, Generation: "1", State: "enabled", Configuration: config}, Policy: c.Policy{Protocol: c.PolicyProtocol, Scope: request.Scope, ProfileID: *request.ProfileID, ProfileGeneration: "1", Generation: "1", Permitted: *request.Change.Permitted}}
	c.SealSnapshot(&snapshot)
	return Endpoint{EndpointProtocol, direct, request.Scope, config.Applications[0], "/example/private-session", "/example/private-dispatch", snapshot}
}

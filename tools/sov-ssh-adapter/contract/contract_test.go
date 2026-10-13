package contract

import (
	"encoding/base64"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func example(t *testing.T) Input {
	t.Helper()
	d := strings.Repeat("a", 64)
	key := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(append([]byte{0, 0, 0, 11, 's', 's', 'h', '-', 'e', 'd', '2', '5', '5', '1', '9', 0, 0, 0, 32}, make([]byte, 32)...))
	e := Endpoint{Protocol: EndpointProtocol, SSHExecution: "protected_system_path", SSH: Executable{"/usr/bin/ssh", d}, Host: "127.0.0.1", Port: "22", User: "user", HostPublicKey: key, ClientPublicKey: key, AgentSocket: "/private/agent.sock", TargetLauncher: Executable{"/remote/launcher", d}}
	inv := map[string]any{"protocol": "symphony.sov.remote-invoke-input.v1", "selection": map[string]any{"transport": map[string]any{"arrangement": "controller", "family": "scoped_cli", "adapter_id": AdapterID, "adapter_version": Version, "endpoint_ref": EndpointRef(e)}, "runtime": map[string]any{"qxctl_sha256": d}}, "endpoint": map[string]any{"target_qxctl": Executable{"/remote/qxctl", d}, "configuration_path": "/remote/config.json", "configuration_sha256": d}}
	b, _ := json.Marshal(inv)
	return Input{InputProtocol, e, b}
}
func TestControlBindings(t *testing.T) {
	in := example(t)
	raw, _ := json.Marshal(in)
	if _, e := DecodeInput(raw); e != nil {
		t.Fatal(e)
	}
	if _, _, e := TargetCommand(in); e != nil {
		t.Fatal(e)
	}
	for name, edit := range map[string]func(*Endpoint){"host-option": func(e *Endpoint) { e.Host = "-oProxyCommand=bad" }, "host-newline": func(e *Endpoint) { e.Host = "good\nbad" }, "port": func(e *Endpoint) { e.Port = "65536" }, "alias-port": func(e *Endpoint) { e.Port = "022" }, "agent-expansion": func(e *Endpoint) { e.AgentSocket = "/private/${BAD}" }, "key-options": func(e *Endpoint) { e.HostPublicKey = "command=bad " + e.HostPublicKey }, "user-option": func(e *Endpoint) { e.User = "-o" }, "digest": func(e *Endpoint) { e.SSH.SHA256 = "bad" }, "deadline": func(e *Endpoint) { s := "2147483648"; e.ConnectTimeoutSeconds = &s }} {
		t.Run(name, func(t *testing.T) {
			e := in.Endpoint
			edit(&e)
			if Validate(e) == nil {
				t.Fatal("invalid endpoint admitted")
			}
		})
	}
	changed := in
	changed.Endpoint.Host = "different.invalid"
	if _, _, e := TargetCommand(changed); e == nil {
		t.Fatal("unbound endpoint change")
	}
	for _, raw := range [][]byte{[]byte(`{"a":1,"a":2}`), []byte(`{} {}`), []byte(`{"a":1,"\u0061":2}`), {0xff}, []byte(strings.Repeat("[", 66) + strings.Repeat("]", 66))} {
		if CheckJSON(raw) == nil {
			t.Fatal("bad JSON admitted")
		}
	}
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	endpoint := fields["endpoint"].(map[string]any)
	delete(endpoint, "connect_timeout_seconds")
	missing, _ := json.Marshal(fields)
	if _, e := DecodeInput(missing); e == nil {
		t.Fatal("missing null member admitted")
	}
}
func TestRemoteCommandLiteralPaths(t *testing.T) {
	in := example(t)
	// printf records every selected argument without executing a substituted path.
	in.Endpoint.TargetLauncher.Path = "/usr/bin/printf"
	var inv map[string]any
	_ = json.Unmarshal(in.Invocation, &inv)
	inv["selection"].(map[string]any)["transport"].(map[string]any)["endpoint_ref"] = EndpointRef(in.Endpoint)
	weird := "/remote/a '$(printf INJECTED)`printf INJECTED` ; b"
	inv["endpoint"].(map[string]any)["target_qxctl"].(map[string]any)["path"] = weird
	in.Invocation, _ = json.Marshal(inv)
	cmd, _, err := TargetCommand(in)
	if err != nil {
		t.Fatal(err)
	}
	// Replace only the known literal executable with an argv recorder.
	cmd = strings.Replace(cmd, "exec '/usr/bin/printf'", "exec /usr/bin/printf '%s\\n'", 1)
	b, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), weird+"\n") {
		t.Fatal("path not preserved literally", string(b))
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sovremote"
	"os"
	"path/filepath"
	"testing"
)

func TestSOVSSHEndpointCompleteSurface(t *testing.T) {
	v, _ := sovremote.Template("ssh_endpoint")
	raw, _ := json.Marshal(v)
	p := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	out, err := sovCommand(t, "sov", "remote", "ssh-endpoint", "--input", p, "--json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := sovremote.InspectSSHEndpoint(raw)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(want)
	var compact bytes.Buffer
	_ = json.Compact(&compact, out)
	if !bytes.Equal(b, compact.Bytes()) {
		t.Fatal("lost endpoint field")
	}
	text, err := sovCommand(t, "sov", "remote", "ssh-endpoint", "--input", p)
	if err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	_ = renderCompleteData(&rendered, b, "text", "SOV SSH")
	if !bytes.Equal(text, rendered.Bytes()) {
		t.Fatal("terminal projection drift")
	}
	for _, op := range []string{"ssh_endpoint", "ssh_invoke"} {
		for _, command := range []string{"schema", "template"} {
			b, e := sovCommand(t, "sov", "remote", command, "--operation", op, "--json")
			if e != nil || !json.Valid(b) {
				t.Fatal(command, op, e)
			}
		}
	}
}

func TestSOVSSHProfileDiscoveryAndCompleteFailureSurface(t *testing.T) {
	for _, op := range []string{"ssh_profile_selection", "ssh_endpoint_v2", "ssh_invoke_v2"} {
		for _, command := range []string{"schema", "template"} {
			machine, err := sovCommand(t, "sov", "remote", command, "--operation", op, "--json")
			if err != nil || !json.Valid(machine) {
				t.Fatal(command, op, err)
			}
		}
	}
	if newSOVSSH("ssh-invoke").Flags().Lookup("transport-profile") != nil {
		t.Fatal("invocation override admitted")
	}
	if newSOVSSH("ssh-endpoint").Flags().Lookup("transport-profile") == nil {
		t.Fatal("planning selection missing")
	}
	if os.Geteuid() == 0 {
		t.Skip("non-root presentation of missing installation")
	}
	value, err := sovremote.Template("ssh_invoke_v2")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(value)
	path := filepath.Join(t.TempDir(), "invocation.json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	machine, err := sovCommand(t, "sov", "remote", "ssh-invoke", "--input", path, "--json")
	if err == nil || !json.Valid(machine) {
		t.Fatal("missing protected store must return complete refusal", err, string(machine))
	}
	text, err := sovCommand(t, "sov", "remote", "ssh-invoke", "--input", path)
	if err == nil {
		t.Fatal("missing failure exit")
	}
	var rendered bytes.Buffer
	if err = renderCompleteData(&rendered, machine, "text", "SOV SSH"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(text, rendered.Bytes()) {
		t.Fatal("profile machine/text projection drift")
	}
}

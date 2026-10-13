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

package main

import (
	"bytes"
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"os"
	"testing"
)

func TestSessionDiscoveryAndAuthority(t *testing.T) {
	for _, op := range []string{"schema", "template"} {
		raw, e := sovCommand(t, "sov", "transport", "session", op, "--json")
		if e != nil || !json.Valid(raw) {
			t.Fatal(op, e)
		}
		text, e := sovCommand(t, "sov", "transport", "session", op)
		if e != nil {
			t.Fatal(e)
		}
		var compact, want bytes.Buffer
		_ = json.Compact(&compact, raw)
		_ = renderCompleteData(&want, compact.Bytes(), "text", "SOV transport session")
		if !bytes.Equal(text, want.Bytes()) {
			t.Fatal("text lost evidence")
		}
		if op == "schema" {
			expected, e := os.ReadFile("../../../../knowledge/sov/cloudflare/access-sessions.json")
			if e != nil {
				t.Fatal(e)
			}
			var a, b any
			_ = json.Unmarshal(expected, &a)
			_ = json.Unmarshal(raw, &b)
			aa, _ := json.Marshal(a)
			bb, _ := json.Marshal(b)
			if !bytes.Equal(aa, bb) {
				t.Fatal("installed schema projection drift")
			}
		}
	}
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	m, e := commandregistry.BuildExpected(root)
	if e != nil {
		t.Fatal(e)
	}
	found := 0
	for _, r := range m.Commands {
		for _, op := range []string{"inspect", "plan", "authenticate", "renew", "clear", "schema", "template"} {
			if r.CommandID == "qxcmd:symphony:sov.transport.session."+op {
				found++
				if (op == "authenticate" || op == "renew" || op == "clear") && (r.AuthorityMode != "target_host_permission" || r.Mutability != "permission_backed_mutation") {
					t.Fatal("wrong session authority")
				}
			}
		}
	}
	if found != 7 {
		t.Fatal(found)
	}
	if _, e := sovCommand(t, "sov", "transport", "session", "authenticate", "--command", "sh"); e == nil {
		t.Fatal("raw forwarding")
	}
}

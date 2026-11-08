package main

import (
	"bytes"
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	w "github.com/QuanuX/Symphony/tools/qxctl/internal/transportprofile/wire"
	"testing"
)

func TestTransportProfileCompleteDiscoveryAndAuthority(t *testing.T) {
	for _, op := range []string{"schema", "template"} {
		raw, e := sovCommand(t, "sov", "transport", "profile", op, "--json")
		if e != nil || !json.Valid(raw) {
			t.Fatal(op, e)
		}
		text, e := sovCommand(t, "sov", "transport", "profile", op)
		if e != nil {
			t.Fatal(e)
		}
		var compact bytes.Buffer
		_ = json.Compact(&compact, raw)
		var want bytes.Buffer
		if e = renderCompleteData(&want, compact.Bytes(), "text", "SOV transport profile"); e != nil || !bytes.Equal(text, want.Bytes()) {
			t.Fatal("incomplete text projection", op, e)
		}
	}
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	manifest, e := commandregistry.BuildExpected(root)
	if e != nil {
		t.Fatal(e)
	}
	for _, op := range w.Operations {
		found := false
		for _, command := range manifest.Commands {
			if command.CommandID == "qxcmd:symphony:sov.transport.profile."+op {
				found = true
				if (op == "apply" || op == "recover") && command.Mutability != "permission_backed_mutation" {
					t.Fatal("mutation concealed")
				}
			}
		}
		if !found {
			t.Fatal("missing profile command", op)
		}
	}
	if _, e = sovCommand(t, "sov", "transport", "profile", "apply", "--command", "sh"); e == nil {
		t.Fatal("arbitrary forwarding exposed")
	}
}

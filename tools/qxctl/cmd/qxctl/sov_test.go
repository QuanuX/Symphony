package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sovremote"
)

func sovCommand(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	r, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	r.SetOut(&out)
	r.SetErr(&out)
	r.SetArgs(args)
	e = r.Execute()
	return out.Bytes(), e
}
func TestSOVRemoteCLICompleteResults(t *testing.T) {
	for _, op := range []string{"plan", "assess"} {
		t.Run(op, func(t *testing.T) {
			b, e := sovCommand(t, "sov", "remote", "template", "--operation", op, "--json")
			if e != nil {
				t.Fatal(e)
			}
			var wrapper struct {
				Template json.RawMessage `json:"template"`
			}
			if e = json.Unmarshal(b, &wrapper); e != nil {
				t.Fatal(e)
			}
			p := filepath.Join(t.TempDir(), "input.json")
			if e = os.WriteFile(p, wrapper.Template, 0600); e != nil {
				t.Fatal(e)
			}
			b, e = sovCommand(t, "sov", "remote", op, "--input", p, "--json")
			if e != nil {
				t.Fatal(e)
			}
			var expected any
			if op == "plan" {
				expected, e = sovremote.MakePlan(wrapper.Template)
			} else {
				expected, e = sovremote.Assess(wrapper.Template)
			}
			if e != nil {
				t.Fatal(e)
			}
			want, _ := json.Marshal(expected)
			var compact bytes.Buffer
			if e = json.Compact(&compact, b); e != nil || !bytes.Equal(compact.Bytes(), want) {
				t.Fatal("CLI omitted or changed fields")
			}
			text, e := sovCommand(t, "sov", "remote", op, "--input", p)
			if e != nil {
				t.Fatal(e)
			}
			var full bytes.Buffer
			if e = renderCompleteData(&full, want, "text", "SOV remote"); e != nil || !bytes.Equal(text, full.Bytes()) {
				t.Fatal("incomplete terminal result", e)
			}
			schema, e := sovCommand(t, "sov", "remote", "schema", "--operation", op, "--json")
			if e != nil || !bytes.Contains(schema, []byte(`"additionalProperties": false`)) {
				t.Fatal("missing closed schema", e)
			}
		})
	}
}
func TestSOVRemoteCLIRefusesUnshippedActionsAndBadInputs(t *testing.T) {
	for _, args := range [][]string{{"sov", "remote", "execute"}, {"sov", "remote", "template", "--operation", "execute"}, {"sov", "remote", "plan"}, {"sov", "remote", "plan", "--input", "/missing", "--format", "ndjson"}} {
		if _, e := sovCommand(t, args...); e == nil {
			t.Fatal("invalid grammar admitted")
		}
	}
	p := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(p, []byte(`{"secret":"not-for-diagnostics"}`), 0600)
	b, e := sovCommand(t, "sov", "remote", "plan", "--input", p, "--json")
	if e == nil || bytes.Contains(b, []byte("not-for-diagnostics")) {
		t.Fatal("bad input admitted or echoed")
	}
}
func TestSOVRemoteReadOnlyRegistry(t *testing.T) {
	r, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	for _, op := range []string{"plan", "assess", "schema", "template"} {
		c, _, e := r.Find([]string{"sov", "remote", op})
		if e != nil {
			t.Fatal(e)
		}
		s, e := commandregistry.Spec(c)
		if e != nil {
			t.Fatal(e)
		}
		if s.CommandID != "qxcmd:symphony:sov.remote."+op || s.Mutability != "read_only" || s.AuthorityMode != "none" || s.TargetScope != "local" || len(s.BackendOperationIDs) != 0 {
			t.Fatal(s)
		}
	}
}

func TestSOVRemoteProcessEntryAndMachineErrors(t *testing.T) {
	out, status := invokeCLI(t, "sov", "remote", "template", "--operation", "plan", "--json")
	if status != 0 || !json.Valid([]byte(out)) || !strings.Contains(out, "symphony.sov.remote-template.v1") {
		t.Fatalf("entry point unavailable: %d", status)
	}
	for _, flag := range []string{"--json", "--format=json"} {
		out, status = invokeCLI(t, "sov", "remote", "plan", "--input", "/missing-private-input", flag)
		decodeCLIError(t, out, status, "command_failed")
		if strings.Contains(out, "missing-private-input") {
			t.Fatal("path echoed into error")
		}
	}
}
func TestSOVRemoteHumanRenderingEscapesUserMetadata(t *testing.T) {
	v, _ := sovremote.Template("plan")
	s := v.(sovremote.Selection)
	s.Extensions["display"] = "\x1b[2J\u202euser"
	b, _ := json.Marshal(s)
	p := filepath.Join(t.TempDir(), "input.json")
	_ = os.WriteFile(p, b, 0600)
	out, e := sovCommand(t, "sov", "remote", "plan", "--input", p)
	if e != nil {
		t.Fatal(e)
	}
	if strings.ContainsAny(string(out), "\x1b\u202e") || !bytes.Contains(out, []byte(`\u202e`)) {
		t.Fatal("unsafe or incomplete human rendering")
	}
}

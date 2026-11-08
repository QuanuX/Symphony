//go:build darwin || linux

package main

import (
	"bytes"
	"encoding/json"
	c "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/contract"
	"os"
	"testing"
)

func TestClosedCommandAndPrivilegeBoundary(t *testing.T) {
	for _, args := range [][]string{{}, {"administer", "--root", "/tmp"}, {"sh"}, {"query", "--input", "secret"}} {
		var out, err bytes.Buffer
		if run(args, &bytes.Buffer{}, &out, &err) == 0 || out.Len() != 0 {
			t.Fatal("open command surface", args)
		}
	}
	if os.Geteuid() != 0 {
		t.Setenv("SUDO_UID", "0")
		t.Setenv("SUDO_USER", "root")
		var out, err bytes.Buffer
		if run([]string{"administer"}, bytes.NewBufferString("{}"), &out, &err) != 3 {
			t.Fatal("environment granted privilege")
		}
	}
	for _, op := range []string{"schema", "template"} {
		var out, err bytes.Buffer
		if run([]string{op}, &bytes.Buffer{}, &out, &err) != 0 || !json.Valid(out.Bytes()) {
			t.Fatal(op, out.String(), err.String())
		}
		if op == "template" {
			var r c.Request
			if c.Decode(bytes.TrimSpace(out.Bytes()), &r) != nil || c.ValidateRequest(r) != nil {
				t.Fatal("invalid typed template")
			}
		}
	}
}

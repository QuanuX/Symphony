//go:build ignore

// Generate independent qxctl consumers without unpublished module imports.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	check := flag.Bool("check", false, "verify projections")
	flag.Parse()
	root := "../.."
	outputs := map[string][]byte{}
	for _, name := range []string{"endpoint-v2.json", "endpoint-v2.sha256"} {
		raw, e := os.ReadFile(filepath.Join("testdata", name))
		must(e)
		outputs[filepath.Join(root, "tools/qxctl/internal/sshprofile/testdata", name)] = raw
	}
	for _, pair := range [][2]string{{"profile", "sshprofile"}, {"profilewire", "sshprofilewire"}} {
		entries, e := os.ReadDir(pair[0])
		must(e)
		for _, f := range entries {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
				continue
			}
			raw, e := os.ReadFile(filepath.Join(pair[0], f.Name()))
			must(e)
			s := string(raw)
			s = strings.Replace(s, "package "+pair[0], "package "+pair[1], 1)
			s = strings.ReplaceAll(s, "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profilewire", "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofilewire")
			b, e := format.Source([]byte(s))
			must(e)
			outputs[filepath.Join(root, "tools/qxctl/internal", pair[1], f.Name())] = b
		}
	}
	for _, name := range []string{"types.go", "codec.go", "validate.go", "presentation.go"} {
		raw, e := os.ReadFile(filepath.Join(root, "tools/sov-transport-profile-admin/contract", name))
		must(e)
		s := strings.Replace(string(raw), "package contract", "package profilewire", 1)
		b, e := format.Source([]byte(s))
		must(e)
		outputs[filepath.Join("profilewire", name)] = b
		outputs[filepath.Join(root, "tools/qxctl/internal/sshprofilewire", name)] = []byte(strings.Replace(string(b), "package profilewire", "package sshprofilewire", 1))
	}
	raw, e := os.ReadFile("connectionaudit/contract.go")
	must(e)
	outputs[filepath.Join(root, "modules/secure-identity-access-governance/internal/connectionaudit/contract.go")] = raw
	for p, b := range outputs {
		if *check {
			actual, e := os.ReadFile(p)
			if e != nil || string(actual) != string(b) {
				must(fmt.Errorf("stale projection: %s", p))
			}
		} else {
			must(os.MkdirAll(filepath.Dir(p), 0755))
			must(os.WriteFile(p, b, 0644))
		}
	}
}
func must(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

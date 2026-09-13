package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSHVAllNativeRoutesHaveExactSelectionAndRegistry(t *testing.T) {
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	if e = commandregistry.Validate(root); e != nil {
		t.Fatal(e)
	}
	routes := []string{"shv inspect", "shv coverage default", "shv coverage plan", "shv catalogue build", "shv catalogue query", "shv evaluate", "shv graph project", "shv graph validate", "shv graph adapter inspect", "shv graph adapter roundtrip", "shv graph adapter query"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			c, tail, e := root.Find(strings.Fields(route))
			if e != nil || len(tail) != 0 || c.RunE == nil {
				t.Fatal("missing native route", e, tail)
			}
			prefix, version := "prefix", "version"
			if strings.Contains(route, " adapter ") {
				prefix, version = "adapter-prefix", "adapter-version"
			}
			for _, name := range []string{prefix, version, "json"} {
				if c.Flags().Lookup(name) == nil {
					t.Fatal("missing explicit flag", name)
				}
			}
			if c.Flags().Lookup(version).DefValue != "" {
				t.Fatal("silent version default")
			}
			if c.Flags().Lookup(prefix).DefValue != "" {
				t.Fatal("silent installation default")
			}
		})
	}
	for _, route := range []string{"shv schema", "shv template"} {
		c, tail, e := root.Find(strings.Fields(route))
		if e != nil || len(tail) != 0 || c.RunE == nil {
			t.Fatal("missing installed discovery route")
		}
		for _, name := range []string{"prefix", "version", "adapter-prefix", "adapter-version"} {
			if c.Flags().Lookup(name) == nil {
				t.Fatal("discovery cannot choose exact owner", name)
			}
		}
	}
}

func TestSHVProductionDispatch(t *testing.T) {
	output, status := invokeCLI(t, "shv", "--help")
	if status != 0 || strings.Contains(output, "unknown command") {
		t.Fatalf("production execute dispatcher omitted SHV: status %d %s", status, output)
	}
}

// This test executes the built production binary in a directory with no source
// checkout, then traverses all native routes and both discovery ownership paths.
func TestSHVInstalledCLIProcess(t *testing.T) {
	binary, prefix, adapter := os.Getenv("SHV_QXCTL_TEST_BINARY"), os.Getenv("SHV_TEST_PREFIX"), os.Getenv("SHV_ADAPTER_TEST_PREFIX")
	if binary == "" || prefix == "" || adapter == "" {
		t.Skip("requires explicit production qxctl and exact installed SHV/adapter prefixes")
	}
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	sequence := 0
	run := func(route []string, payload any, isAdapter bool) map[string]any {
		t.Helper()
		args := append([]string{}, route...)
		if isAdapter {
			args = append(args, "--adapter-prefix", adapter, "--adapter-version", "0.1.0-dev")
		} else {
			args = append(args, "--prefix", prefix, "--version", "0.1.0-dev")
		}
		if payload != nil {
			raw, e := json.Marshal(payload)
			if e != nil {
				t.Fatal(e)
			}
			sequence++
			path := filepath.Join(root, fmt.Sprintf("input-%d.json", sequence))
			if e = os.WriteFile(path, raw, 0600); e != nil {
				t.Fatal(e)
			}
			args = append(args, "--input", path)
		}
		args = append(args, "--json")
		c := exec.Command(binary, args...)
		c.Dir = root
		c.Env = []string{}
		out, e := c.CombinedOutput()
		if e != nil {
			t.Fatalf("%v: %v\n%s", route, e, out)
		}
		var m map[string]any
		d := json.NewDecoder(bytes.NewReader(out))
		d.UseNumber()
		if e = d.Decode(&m); e != nil {
			t.Fatalf("%v emitted non-JSON: %s", route, out)
		}
		if r, ok := m["result"].(map[string]any); ok {
			return r
		}
		return m
	}
	run([]string{"shv", "inspect"}, nil, false)
	profile := run([]string{"shv", "coverage", "default"}, map[string]any{"as_of": "2026-09-13"}, false)
	run([]string{"shv", "coverage", "plan"}, map[string]any{"profile": profile, "subjects": []any{}}, false)
	html := `<div id="heading"><h1>Custom CPU</h1></div><article id="fields"><dt>Cores</dt><dd>16</dd><dt>End</dt><dd>done</dd></article>`
	if e = os.WriteFile(filepath.Join(root, "source.html"), []byte(html), 0600); e != nil {
		t.Fatal(e)
	}
	source := map[string]any{"id": "source", "path": "source.html", "bytes": len(html), "digest": fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(html))), "format": "html"}
	subject := map[string]any{"id": "cpu", "manufacturer": "Custom", "model": "Custom CPU", "hardware_class": "cpu", "source_id": "source", "heading_section": "div#heading", "field_section": "article#fields", "fields": []any{map[string]any{"predicate": "cores", "label": "Cores", "next_label": "End", "value_type": "integer", "qualifier": "declared"}}}
	catalogue := run([]string{"shv", "catalogue", "build"}, map[string]any{"source_root": root, "sources": []any{source}, "subjects": []any{subject}}, false)
	run([]string{"shv", "catalogue", "query"}, map[string]any{"source_root": root, "catalogue": catalogue, "subject_ids": []any{"cpu", "missing"}}, false)
	evaluation := run([]string{"shv", "evaluate"}, map[string]any{"source_root": root, "catalogue": catalogue, "subject_ids": []any{}, "requirements": []any{map[string]any{"id": "minimum", "predicate": "cores", "operator": "gte", "value": 32, "qualifier": "declared"}}}, false)
	if evaluation["findings"].([]any)[0].(map[string]any)["status"] != "contradicted" {
		t.Fatal("production evaluation concealed mismatch")
	}
	graph := run([]string{"shv", "graph", "project"}, map[string]any{"source_root": root, "catalogue": catalogue}, false)
	run([]string{"shv", "graph", "validate"}, map[string]any{"source_root": root, "graph": graph}, false)
	run([]string{"shv", "graph", "adapter", "inspect"}, nil, true)
	run([]string{"shv", "graph", "adapter", "roundtrip"}, map[string]any{"graph": graph}, true)
	run([]string{"shv", "graph", "adapter", "query"}, map[string]any{"graph": graph, "kind": "nodes", "ids": []any{"subject:cpu", "missing", "source:source"}}, true)
	for _, a := range []bool{false, true} {
		run([]string{"shv", "schema"}, nil, a)
		run([]string{"shv", "schema", "--protocol", "symphony.qxctl.shv-template.v1"}, nil, a)
		template := run([]string{"shv", "template", "--operation", "inspect"}, nil, a)
		if template["status"] != "unanswered_template_not_validated_input" {
			t.Fatal("template overstated validation")
		}
	}
}

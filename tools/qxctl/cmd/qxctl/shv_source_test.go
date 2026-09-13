package main

import (
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

func TestSHVLifecycleRoutes(t *testing.T) {
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	if e = commandregistry.Validate(root); e != nil {
		t.Fatal(e)
	}
	for _, route := range []string{"shv source inspect", "shv source plan", "shv source reduce", "shv source status", "shv source capture import", "shv source capture compare", "shv source graph project", "shv source graph validate", "shv source schema", "shv source template"} {
		t.Run(route, func(t *testing.T) {
			c, tail, e := root.Find(strings.Fields(route))
			if e != nil || len(tail) != 0 || c.RunE == nil {
				t.Fatal("missing route", e)
			}
			for _, f := range []string{"prefix", "version", "json"} {
				if c.Flags().Lookup(f) == nil {
					t.Fatal("missing flag", f)
				}
			}
			for _, f := range []string{"prefix", "version"} {
				if c.Flags().Lookup(f).DefValue != "" {
					t.Fatal("implicit selection")
				}
			}
		})
	}
	out, status := invokeCLI(t, "shv", "source", "--help")
	if status != 0 || strings.Contains(out, "unknown command") {
		t.Fatal(status, out)
	}
}
func TestSHVLifecycleInstalledProductionCLI(t *testing.T) {
	binary, prefix := os.Getenv("SHV_SOURCE_QXCTL_TEST_BINARY"), os.Getenv("SHV_SOURCE_TEST_PREFIX")
	if binary == "" || prefix == "" {
		t.Skip("explicit built qxctl and installed source prefix required")
	}
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	for _, leaf := range []string{"inspect", "schema", "template"} {
		args := []string{"shv", "source", leaf, "--prefix", prefix, "--version", "0.1.0-dev", "--json"}
		if leaf == "template" {
			args = append(args, "--operation", "source_reduce")
		}
		c := exec.Command(binary, args...)
		c.Dir = dir
		c.Env = []string{}
		raw, e := c.CombinedOutput()
		if e != nil {
			t.Fatal(leaf, e, string(raw))
		}
		if !strings.Contains(string(raw), "symphony.") {
			t.Fatal("missing structured output")
		}
	}
	input := filepath.Join(dir, "plan.json")
	body := `{"operation_id":"cli-onboard","current":null,"desired":{"source_id":"family","publisher":"vendor","authority_role":"caller scope","subject_ids":["a","b"],"locators":[{"id":"primary","uri":"https://example.test/family","format":"opaque"}]},"reason":"explicit caller candidate"}`
	if e = os.WriteFile(input, []byte(body), 0600); e != nil {
		t.Fatal(e)
	}
	c := exec.Command(binary, "shv", "source", "plan", "--prefix", prefix, "--version", "0.1.0-dev", "--input", input, "--json")
	c.Dir = dir
	c.Env = []string{}
	raw, e := c.CombinedOutput()
	if e != nil || !strings.Contains(string(raw), "onboard") {
		t.Fatal(e, string(raw))
	}
}

// Exercise every native leaf through the actual main dispatcher, without a checkout cwd.
func TestSHVLifecycleInstalledCLISequence(t *testing.T) {
	binary, prefix := os.Getenv("SHV_SOURCE_QXCTL_TEST_BINARY"), os.Getenv("SHV_SOURCE_TEST_PREFIX")
	if binary == "" || prefix == "" {
		t.Skip("explicit production binary and installed prefix required")
	}
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	seq := 0
	run := func(route []string, p any) map[string]any {
		t.Helper()
		args := append([]string{"shv", "source"}, route...)
		args = append(args, "--prefix", prefix, "--version", "0.1.0-dev", "--json")
		if p != nil {
			raw, e := json.Marshal(p)
			if e != nil {
				t.Fatal(e)
			}
			seq++
			path := filepath.Join(root, fmt.Sprintf("request-%d.json", seq))
			if e = os.WriteFile(path, raw, 0600); e != nil {
				t.Fatal(e)
			}
			args = append(args, "--input", path)
		}
		c := exec.Command(binary, args...)
		c.Dir = root
		c.Env = []string{}
		out, e := c.CombinedOutput()
		if e != nil {
			t.Fatal(route, e, string(out))
		}
		var envelope struct {
			Result map[string]any `json:"result"`
		}
		if e = json.Unmarshal(out, &envelope); e != nil || envelope.Result == nil {
			t.Fatal("invalid response", e, string(out))
		}
		return envelope.Result
	}
	run([]string{"inspect"}, nil)
	definition := map[string]any{"source_id": "manual-family", "publisher": "caller", "authority_role": "declared", "subject_ids": []string{"a", "b"}, "locators": []any{map[string]any{"id": "one", "uri": "https://example.test/doc", "format": "opaque"}}}
	plan := run([]string{"plan"}, map[string]any{"current": nil, "desired": definition, "operation_id": "first", "reason": "candidate only"})
	run([]string{"reduce"}, map[string]any{"current": nil, "plan": plan})
	run([]string{"status"}, map[string]any{"history": []any{plan["source"]}})
	body := []byte("retained opaque bytes")
	if e = os.WriteFile(filepath.Join(root, "body"), body, 0600); e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256(body)
	capture := run([]string{"capture", "import"}, map[string]any{"source_root": root, "source": plan["source"], "locator_id": "one", "resolved_uri": "https://example.test/doc", "redirect_chain": []any{}, "observed_at": "2026-09-13T00:00:00Z", "upstream_revision": nil, "manifest": map[string]any{"id": "capture", "path": "body", "bytes": len(body), "digest": fmt.Sprintf("sha256:%x", hash), "format": "opaque"}, "completeness": "complete", "issues": []any{}})
	run([]string{"capture", "compare"}, map[string]any{"source_root": root, "previous": capture, "current": capture})
	g := run([]string{"graph", "project"}, map[string]any{"source_root": root, "captures": []any{capture}})
	run([]string{"graph", "validate"}, map[string]any{"source_root": root, "graph": g})
}

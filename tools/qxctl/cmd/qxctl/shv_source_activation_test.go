package main

import (
	"bytes"
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSHVActivationCanonicalEnvelope(t *testing.T) {
	value := map[string]any{"installation": struct {
		Z string
		A string
	}{"last", "first"}, "attempt": struct {
		Transition  json.RawMessage
		OperationID string
	}{json.RawMessage(`{"z":3,"a":1}`), "op"}}
	raw, e := sealSHVActivation(value)
	if e != nil {
		t.Fatal(e)
	}
	var normalized map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e = d.Decode(&normalized); e != nil {
		t.Fatal(e)
	}
	got := normalized["digest"]
	delete(normalized, "digest")
	want, e := knowledgeengine.SCVDigest(normalized)
	if e != nil || got != want {
		t.Fatal("struct declaration order leaked into seal", got, want, e)
	}
}

func TestSHVActivationMachineErrors(t *testing.T) {
	out, status := invokeCLI(t, "shv", "source", "activation", "apply", "--json")
	var errorResult map[string]any
	if status == 0 || json.Unmarshal([]byte(out), &errorResult) != nil || errorResult["protocol"] != cliErrorProtocol || errorResult["outcome"] != "error" {
		t.Fatal(status, out)
	}
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	if scvJSONRequested(root, []string{"shv", "source", "activation", "apply", "--input", "--json"}) {
		t.Fatal("flag value treated as output intent")
	}
	if scvJSONRequested(root, []string{"shv", "source", "activation", "apply", "--json=false"}) {
		t.Fatal("false ignored")
	}
}

func TestSHVActivationRoutesAndAuthority(t *testing.T) {
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	manifest, e := commandregistry.Build(root, commandregistry.Identity{ClientVersion: "test", ExecutableDigest: commandManifestTestDigest})
	if e != nil {
		t.Fatal(e)
	}
	found := 0
	for _, c := range manifest.Commands {
		if !strings.HasPrefix(c.CommandID, "qxcmd:symphony:shv.source.activation.") {
			continue
		}
		found++
		leaf := strings.TrimPrefix(c.CommandID, "qxcmd:symphony:shv.source.activation.")
		if leaf == "apply" || leaf == "recover" {
			if c.Mutability != "permission_backed_mutation" || c.AuthorityMode != "target_host_permission" || c.RecoveryCommandID == nil || *c.RecoveryCommandID != "qxcmd:symphony:shv.source.activation.recover" {
				t.Fatal(c)
			}
		}
		route, _, e := root.Find([]string{"shv", "source", "activation", leaf})
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range []string{"prefix", "version"} {
			if route.Flags().Lookup(f) == nil || route.Flags().Lookup(f).DefValue != "" {
				t.Fatal("implicit selection", f)
			}
		}
	}
	if found != 6 {
		t.Fatal("missing activation routes", found)
	}
	for _, kind := range []string{"onboard", "relocation", "authority_change"} {
		a, e := shvActivationAction(kind)
		if e != nil || !strings.HasPrefix(a, "symphony.shv.source.") {
			t.Fatal(kind, a, e)
		}
	}
	if _, e := shvActivationAction("relocate"); e == nil {
		t.Fatal("SCV vocabulary accepted")
	}
	o := shvActivationOptions{topsID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", sourceID: "intel"}
	resource := shvActivationResource(o)
	d, _ := knowledgeengine.SCVDigest(map[string]any{"tops_id": o.topsID, "owner_engine_id": "symphony-shv-source", "source_id": "intel"})
	if resource != "symphony.shv.source:"+strings.TrimPrefix(d, "sha256:") {
		t.Fatal("resource ownership mismatch")
	}
	o.sourceID = "amd"
	if shvActivationResource(o) == resource {
		t.Fatal("source collision")
	}
}

func TestSHVActivationInstalledReadOnlyAndRejections(t *testing.T) {
	binary, prefix := os.Getenv("SHV_ACTIVATION_QXCTL"), os.Getenv("SHV_SOURCE_TEST_PREFIX")
	if binary == "" || prefix == "" {
		t.Skip("explicit production CLI and source installation required")
	}
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	input := filepath.Join(dir, "input.json")
	run := func(leaf string, body string, good bool, extra ...string) map[string]any {
		t.Helper()
		args := []string{"shv", "source", "activation", leaf, "--prefix", prefix, "--version", "0.1.0-dev", "--json"}
		if leaf != "schema" && leaf != "template" {
			args = append(args, "--tops-id", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "--source-id", "intel", "--state-root", dir)
		}
		if body != "" {
			if e := os.WriteFile(input, []byte(body), 0600); e != nil {
				t.Fatal(e)
			}
			args = append(args, "--input", input)
		}
		args = append(args, extra...)
		c := exec.Command(binary, args...)
		c.Dir = dir
		c.Env = []string{}
		raw, e := c.CombinedOutput()
		if (e == nil) != good {
			t.Fatal(leaf, e, string(raw))
		}
		if !good {
			var failure map[string]any
			if json.Unmarshal(raw, &failure) != nil || failure["protocol"] != cliErrorProtocol {
				t.Fatal("activation failure is not structured", string(raw))
			}
			return nil
		}
		var result map[string]any
		if e := json.Unmarshal(raw, &result); e != nil {
			t.Fatal(e, string(raw))
		}
		if got, ok := result["digest"]; ok {
			delete(result, "digest")
			want, e := knowledgeengine.SCVDigest(result)
			if e != nil || got != want {
				t.Fatal("noncanonical CLI result seal", leaf, e)
			}
			result["digest"] = got
		}
		return result
	}
	status := run("status", "", true)
	if status["state_digest"] != nil || len(status["history"].([]any)) != 0 {
		t.Fatal(status)
	}
	schema := run("schema", "", true)
	if schema["origin"] != "qxctl_embedded" {
		t.Fatal(schema)
	}
	run("template", "", true, "--operation", "propose")
	proposal := `{"operation_id":"onboard","desired":{"source_id":"intel","publisher":"Intel","authority_role":"specification","subject_ids":["cpu"],"locators":[{"id":"primary","uri":"https://example.test/cpu","format":"html"}]},"reason":"explicit caller source"}`
	plan := run("propose", proposal, true)
	if plan["protocol"] != "symphony.shv.source-plan.v1" {
		t.Fatal(plan)
	}
	// Malformed surrogate must reject before any committed-replay comparison.
	bad := strings.Replace(proposal, "explicit caller source", `\ud800`, 1)
	run("propose", bad, false)
	run("apply", bad, false)
	run("propose", strings.Replace(proposal, `"reason":`, `"current":null,"reason":`, 1), false)
	run("status", "", false, "--operation-id", "missing")
	run("recover", "", false, "--operation-id", "missing")
	again := run("status", "", true)
	if again["state_digest"] != nil {
		t.Fatal("read-only paths changed head")
	}
}

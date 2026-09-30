package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

func TestSNVCanonicalSurface(t *testing.T) {
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	manifest, e := commandregistry.BuildExpected(root)
	if e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	for _, command := range manifest.Commands {
		if strings.HasPrefix(command.CommandID, "qxcmd:symphony:snv.") {
			if len(command.Aliases) != 0 {
				t.Fatal("unexpected SNV alias")
			}
			seen[command.CommandID] = true
			if command.CommandID == "qxcmd:symphony:snv.state.apply" || command.CommandID == "qxcmd:symphony:snv.state.recover" {
				if command.AuthorityMode != "target_host_permission" || command.Mutability != "permission_backed_mutation" {
					t.Fatal("selection authority missing")
				}
			}
		}
	}
	if len(seen) != 16 {
		t.Fatalf("SNV leaf count %d, want16", len(seen))
	}
	for _, leaf := range snvLeaves {
		if !seen["qxcmd:symphony:snv."+leaf.path] {
			t.Fatal(leaf.path)
		}
	}
	for _, args := range [][]string{{"snv", "names", "validate", "--private-marker=x", "--json"}, {"snv", "private-marker", "--json"}, {"snv", "state", "recover", "--json"}, {"snv", "schema", "--json", "--prefix", "private-marker", "--version", "latest", "--owner", "snv", "--operation", "snv_inspect"}} {
		out, status := invokeCLI(t, args...)
		if status == 0 || strings.Contains(out, "private-marker") || !strings.HasPrefix(out, "{") {
			t.Fatalf("unsafe SNV refusal %d %s", status, out)
		}
	}
	for _, path := range [][]string{{"snv", "identity", "validate"}, {"snv", "state", "apply"}, {"snv", "evidence", "prepare"}, {"snv", "names", "resolve"}, {"snv", "schema"}} {
		args := append(path, "--help")
		out, status := invokeCLI(t, args...)
		if status != 0 || !strings.Contains(out, "--prefix") || !strings.Contains(out, "--version") {
			t.Fatal("SNV help incomplete")
		}
	}
	if !scvJSONRequested(root, []string{"snv", "state", "recover", "--json"}) || scvJSONRequested(root, []string{"snv", "state", "recover", "--operation-id", "--json"}) {
		t.Fatal("SNV JSON flag/value detection")
	}
}

func TestSNVPublicCapsuleCanonicalRoundTrip(t *testing.T) {
	inst := knowledgeengine.Installation{Role: "snv", ModuleID: "snv-engine", EngineID: "symphony-snv", Version: "0.1.0-dev", Prefix: "/test-installed"}
	input, plan := json.RawMessage(`{"z":1,"a":"<x>\u2028"}`), json.RawMessage(`{"z":3,"a":4}`)
	capsule, e := snvSeal(map[string]any{"protocol": snvProposalProtocol, "input": input, "plan": plan, "installation": inst})
	if e != nil {
		t.Fatal(e)
	}
	object, e := snvObject(capsule)
	if e != nil {
		t.Fatal(e)
	}
	// Independent Python UTF-8 sorted-object SHA256 fixture: this fixes the
	// representation, including struct fields, raw object ordering and U+2028.
	if object["digest"] != "sha256:b569ff374316f39e0f27caf4b7c94c5af91722b6a7c28ebd17b7a0fd3b54fc1d" {
		t.Fatal("public capsule is not canonical sorted UTF-8 JSON")
	}
	gotInput, gotPlan, e := snvProposal(capsule, inst)
	if e != nil || !snvSame(gotInput, input) || !snvSame(gotPlan, plan) {
		t.Fatalf("produced public proposal failed consumer verification: %v", e)
	}
	changed := inst
	changed.ReceiptDigest = "sha256:" + strings.Repeat("1", 64)
	if _, _, e := snvProposal(capsule, changed); e == nil {
		t.Fatal("capsule admitted another installation")
	}
}

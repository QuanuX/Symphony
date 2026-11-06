package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

func TestSRRAPIVCanonicalSurface(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]*knowledgeengine.SRRAPIVOperation{
		"qxcmd:symphony:sqv.rithmic.schema":   nil,
		"qxcmd:symphony:sqv.rithmic.template": nil,
	}
	for _, op := range knowledgeengine.SRRAPIVOperations {
		copy := op
		want["qxcmd:symphony:sqv.rithmic."+op.Domain+"."+op.Leaf] = &copy
	}
	for _, command := range manifest.Commands {
		if !strings.HasPrefix(command.CommandID, "qxcmd:symphony:sqv.rithmic.") {
			continue
		}
		op, found := want[command.CommandID]
		if !found || len(command.Aliases) != 0 {
			t.Fatalf("undeclared or aliased command %s", command.CommandID)
		}
		delete(want, command.CommandID)
		if len(command.ResultValidationProtocols) != 1 || len(command.FeatureBindings) != 2 {
			t.Fatalf("incomplete evidence metadata: %s", command.CommandID)
		}
		if op == nil {
			continue
		}
		if len(command.BackendOperationIDs) != 1 || command.BackendOperationIDs[0] != "engop:symphony:srrapiv."+op.Domain+"."+op.Leaf {
			t.Fatalf("native route mismatch %s", command.CommandID)
		}
		if op.Runtime && command.AuthorityMode != "target_host_permission" {
			t.Fatalf("missing host authority %s", command.CommandID)
		}
		if !op.Runtime && (command.AuthorityMode != "none" || command.Mutability != "read_only") {
			t.Fatalf("pure operation gained authority %s", command.CommandID)
		}
		if op.Mutating && (command.Mutability != "permission_backed_mutation" || command.RecoveryCommandID == nil) {
			t.Fatalf("mutation lacks recovery %s", command.CommandID)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing command routes: %#v", want)
	}
}

func TestSRRAPIVInstalledCLI(t *testing.T) {
	prefix := os.Getenv("SRRAPIV_TEST_PREFIX")
	if prefix == "" {
		t.Skip("select the exact native SRRAPIV installation")
	}
	input := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(input, []byte(`{"protocol":"symphony.srrapiv.request.v1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	output, status := invokeCLI(t, "sqv", "rithmic", "api", "describe", "--prefix", prefix, "--version", knowledgeengine.SRRAPIVVersion, "--input", input, "--json")
	var response knowledgeengine.Response
	if status != 0 || json.Unmarshal([]byte(output), &response) != nil || response.EngineID != knowledgeengine.SRRAPIVEngineID || response.Operation != "api_describe" {
		t.Fatalf("installed discovery failed: status=%d output=%s", status, output)
	}
	// Domain rejection is a complete validated native response with nonzero
	// exit status, not a successful exit or a second appended error document.
	if err := os.WriteFile(input, []byte(`{"protocol":"symphony.srrapiv.request.v1","catalog":{"schema":"private-marker"},"binding":{},"compatibility_evidence":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	output, status = invokeCLI(t, "sqv", "rithmic", "binding", "validate", "--prefix", prefix, "--version", knowledgeengine.SRRAPIVVersion, "--input", input, "--json")
	if status != 2 || json.Unmarshal([]byte(output), &response) != nil || response.Operation != "binding_validate" || response.Outcome != "ok" || strings.Contains(output, "private-marker") {
		t.Fatalf("domain refusal lost evidence: status=%d output=%s", status, output)
	}
}

func TestSRRAPIVSafeRefusalsAndHelp(t *testing.T) {
	for _, args := range [][]string{
		{"sqv", "rithmic", "runtime", "invoke", "--private-marker=x", "--json"},
		{"sqv", "rithmic", "private-marker", "--json"},
		{"sqv", "rithmic", "schema", "--prefix", "private-marker", "--version", "latest", "--operation", "api_describe", "--json"},
		{"sqv", "rithmic", "runtime", "invoke", "--prefix", "private-marker", "--version", "0.1.0-dev", "--input", "private-marker", "--json"},
	} {
		output, status := invokeCLI(t, args...)
		if status == 0 || strings.Contains(output, "private-marker") || !strings.HasPrefix(output, "{") {
			t.Fatalf("unsafe refusal %d %s", status, output)
		}
	}
	for _, op := range knowledgeengine.SRRAPIVOperations {
		output, status := invokeCLI(t, "sqv", "rithmic", op.Domain, op.Leaf, "--help")
		if status != 0 || !strings.Contains(output, "--input") || !strings.Contains(output, "--version") || !strings.Contains(output, "--prefix") {
			t.Fatalf("incomplete help %s", op.Operation)
		}
		if strings.Contains(output, "--password") || strings.Contains(output, "--credential") {
			t.Fatalf("credential argument surface %s", op.Operation)
		}
	}
}

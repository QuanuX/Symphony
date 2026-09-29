package main

import (
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"strings"
	"testing"
)

func TestSQVCanonicalSurface(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	m, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"qxcmd:symphony:sqv.acquisition.validate": false, "qxcmd:symphony:sqv.schema": false, "qxcmd:symphony:sqv.template": false}
	for _, op := range knowledgeengine.SQVAdministrationOperations {
		want["qxcmd:symphony:sqv."+op.Domain+"."+op.Leaf] = false
	}
	for _, c := range m.Commands {
		if strings.HasPrefix(c.CommandID, "qxcmd:symphony:sqv.") {
			if _, ok := want[c.CommandID]; !ok || len(c.Aliases) != 0 {
				t.Fatalf("extra or aliased SQV operation: %s", c.CommandID)
			}
			want[c.CommandID] = true
			if c.Mutability != "read_only" || c.AuthorityMode != "none" {
				t.Fatal("pure validation gained effects")
			}
		}
	}
	for id, found := range want {
		if !found {
			t.Fatal(id)
		}
	}
	for _, args := range [][]string{
		{"sqv", "acquisition", "validate", "--private-marker=x", "--json"},
		{"sqv", "private-marker", "--json"},
		{"sqv", "schema", "--json", "--adapter", "private-marker"},
		{"sqv", "template", "--json", "--prefix", "private-marker", "--version", "latest", "--operation", "request_validate", "--adapter", "fred"},
	} {
		output, status := invokeCLI(t, args...)
		if status == 0 || strings.Contains(output, "private-marker") || !strings.HasPrefix(output, "{") {
			t.Fatalf("unsafe refusal: %d %s", status, output)
		}
	}
	output, status := invokeCLI(t, "sqv", "acquisition", "validate", "--help")
	if status != 0 || !strings.Contains(output, "--input") || !strings.Contains(output, "--version") {
		t.Fatal("incomplete SQV help")
	}
}

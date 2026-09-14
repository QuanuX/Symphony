package main

import (
	"encoding/json"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"sort"
	"strings"
	"testing"
)

func TestCommandRegistryUniqueSemanticSurfaces(t *testing.T) {
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	m, e := commandregistry.Build(root, commandregistry.Identity{ClientVersion: "test", ExecutableDigest: commandManifestTestDigest})
	if e != nil {
		t.Fatal(e)
	}
	groups := map[string][]string{}
	ordered := func(v []string) []string { x := append([]string{}, v...); sort.Strings(x); return x }
	for _, c := range m.Commands {
		if len(c.BackendOperationIDs) == 0 {
			continue
		}
		key, _ := json.Marshal([]any{ordered(c.BackendOperationIDs), ordered(c.InputProtocols), ordered(c.OutputProtocols), c.Mutability, c.TargetScope})
		groups[string(key)] = append(groups[string(key)], c.CommandID)
	}
	// Explicit lifecycle distinctions, documented in COMMANDS.md. Sharing a
	// native operation does not make recovery or resume a competing authority.
	reviewed := map[string]bool{
		"qxcmd:symphony:scv.graph-index.transfer,qxcmd:symphony:scv.graph-index.transfer-recover": false,
		"qxcmd:symphony:shv.materialization.resume,qxcmd:symphony:shv.materialization.run":        false,
	}
	for _, ids := range groups {
		if len(ids) < 2 {
			continue
		}
		sort.Strings(ids)
		key := strings.Join(ids, ",")
		if _, ok := reviewed[key]; !ok {
			t.Errorf("unreviewed competing semantic surfaces: %s", key)
		} else {
			reviewed[key] = true
		}
	}
	for ids, seen := range reviewed {
		if !seen {
			t.Errorf("stale reviewed lifecycle distinction: %s", ids)
		}
	}
}

package main

import (
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"strings"
	"testing"
)

func TestSCVGraphIndexSchemaDiscoveryProfiles(t *testing.T) {
	for _, v := range []struct{ version, path string }{{"0.1.0-dev", "v1"}, {"0.2.0-dev", "v2"}} {
		raw, err := os.ReadFile("../../../../modules/scv-graph-duckdb-connector/schemas/" + v.path + "/graph-index.schema.json")
		if err != nil {
			t.Fatal(err)
		}
		inst := knowledgeengine.Installation{Version: v.version}
		list, err := graphIndexSchemaResult(inst, raw, "")
		if err != nil {
			t.Fatal(err)
		}
		m := graphIndexCLIMap(t, list)
		if m["schema"] != nil {
			t.Fatal("list unexpectedly selects schema")
		}
		for protocol, definition := range graphIndexSchemaEntries(v.version) {
			t.Run(v.version+"/"+protocol, func(t *testing.T) {
				result, err := graphIndexSchemaResult(inst, raw, protocol)
				if err != nil {
					t.Fatal(err)
				}
				m := graphIndexCLIMap(t, result)
				rows := m["entries"].([]any)
				if len(rows) != 1 {
					t.Fatal("selection differs")
				}
				e := rows[0].(map[string]any)
				doc := m["schema"].(map[string]any)
				if doc["$defs"].(map[string]any)[definition] == nil || e["fragment"] != "#/$defs/"+definition {
					t.Fatal("unresolved definition")
				}
				if strings.Contains(protocol, "scv-index-transfer") && e["origin"] != "qxctl_embedded" {
					t.Fatal("CLI contract misattributed to native receipt")
				}
				if m["digest"] != graphIndexCLISeal(t, m)["digest"] {
					t.Fatal("noncanonical seal")
				}
			})
		}
		if _, err = graphIndexSchemaResult(inst, raw, "unknown"); err == nil {
			t.Fatal("unknown protocol accepted")
		}
		if v.version == "0.1.0-dev" {
			if _, err = graphIndexSchemaResult(inst, raw, "symphony.qxctl.scv-graph-index-inventory-input.v1"); err == nil {
				t.Fatal("legacy discovery upgraded silently")
			}
		}
	}
}

package main

import (
	"encoding/json"
	"testing"
)

func TestSHVPartitionCommandErrors(t *testing.T) {
	for _, op := range []string{"inspect", "build", "manifest", "query", "schema", "template", "from-refresh"} {
		t.Run(op, func(t *testing.T) {
			out, status := invokeCLI(t, "shv", "partition", op, "--json")
			var d map[string]any
			if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
				t.Fatal(status, out)
			}
		})
	}
}

package main

import "testing"

func TestSHVPDFStructuredFailures(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code string
	}{
		{[]string{"shv", "pdf", "extract", "--json"}, "command_failed"},
		{[]string{"shv", "pdf", "inspect", "--prefix", "/missing", "--version", "0.1.0-dev", "--json"}, "command_failed"},
		{[]string{"shv", "pdf", "extract", "--unknown", "--json"}, "invalid_arguments"},
		{[]string{"shv", "pdf", "--json"}, "invalid_arguments"},
		{[]string{"shv", "pdf", "graph", "project", "--json"}, "command_failed"},
		{[]string{"shv", "pdf", "graph", "roundtrip", "--unknown", "--json"}, "invalid_arguments"},
	} {
		out, status := invokeCLI(t, tc.args...)
		decodeCLIError(t, out, status, tc.code)
	}
}
